package transport

import (
	"context"
	"sync"
)

// CommandDispatcher orchestrates the per-command flow that follows
// HandleMessage's Command branch: dedupe lookup, pre-command snapshots,
// invocation of the scenario-routing command body on StreamHandler,
// audit-buffer recording, post-command response assembly (route updates,
// PA transitions, overlay clears, broadcast notifications, UI host
// messages), and the final RememberSetUI bookkeeping.
//
// The dispatcher is a thin orchestrator; the actual command body
// (handleCommand and its scenario-routing helpers) stays on StreamHandler
// and is invoked through a function-typed callback wired in the
// constructor. Validation sentinels live at package scope and are
// returned from the run-command callback unchanged.
//
// The dispatcher owns the recent-command audit buffer behind its own
// mutex. The dedupe seen/seenOrder maps remain on StreamHandler because
// they are transport response cache state, not audit history.
type CommandDispatcher struct {
	h          *StreamHandler
	runCommand func(context.Context, *CommandRequest) (ServerMessage, error)

	mu          sync.Mutex
	recent      []CommandEvent
	recentLimit int
}

// NewCommandDispatcher returns a dispatcher bound to h. The runCommand
// callback is the scenario-routing command body StreamHandler exposes
// (StreamHandler.handleCommand) — keeping it as a callback lets the
// dispatcher own the orchestration shape without owning the scenario
// engine plumbing.
func NewCommandDispatcher(
	h *StreamHandler,
	runCommand func(context.Context, *CommandRequest) (ServerMessage, error),
	recentLimit int,
) *CommandDispatcher {
	return &CommandDispatcher{
		h:           h,
		runCommand:  runCommand,
		recent:      []CommandEvent{},
		recentLimit: recentLimit,
	}
}

// Dispatch handles a single Command message end-to-end and returns the
// outgoing ServerMessage slice plus any terminal error. It mirrors the
// previous inline body of HandleMessage's Command branch exactly: dedupe
// short-circuit, pre-command snapshots, runCommand invocation, audit
// append on error, audit append on success, multi-window resume capture,
// dedupe seen-map record, command response assembly, and UI session
// remembering.
func (d *CommandDispatcher) Dispatch(ctx context.Context, cmd *CommandRequest) ([]ServerMessage, error) {
	h := d.h
	h.metrics.commandReceived.Add(1)
	priorActiveScenario := h.activeScenarioName(cmd.DeviceID)
	if cmd.RequestID != "" {
		h.mu.Lock()
		if prior, ok := h.seen[cmd.RequestID]; ok {
			if h.metrics != nil {
				h.metrics.dedupeHits.Add(1)
			}
			d.appendEvent(CommandEvent{
				RequestID: cmd.RequestID,
				DeviceID:  cmd.DeviceID,
				Kind:      cmd.Kind,
				Action:    defaultAction(cmd.Action),
				Intent:    cmd.Intent,
				Outcome:   "deduped",
				WhenUnix:  h.control.now().UTC().UnixMilli(),
			})
			h.mu.Unlock()
			return []ServerMessage{prior}, nil
		}
		h.mu.Unlock()
	}
	beforeRoutes := h.routeSnapshotForDevice(cmd.DeviceID)
	beforeBroadcastEvents := h.broadcastEventCount()
	beforeUIEvents := h.uiHostEventCount()
	commandResult, err := d.runCommand(ctx, cmd)
	if err != nil {
		h.metrics.commandErrors.Add(1)
		d.appendEvent(CommandEvent{
			RequestID: cmd.RequestID,
			DeviceID:  cmd.DeviceID,
			Kind:      cmd.Kind,
			Action:    defaultAction(cmd.Action),
			Intent:    cmd.Intent,
			Outcome:   "error:" + err.Error(),
			WhenUnix:  h.control.now().UTC().UnixMilli(),
		})
		return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
	}
	d.appendEvent(CommandEvent{
		RequestID: cmd.RequestID,
		DeviceID:  cmd.DeviceID,
		Kind:      cmd.Kind,
		Action:    defaultAction(cmd.Action),
		Intent:    cmd.Intent,
		Outcome:   commandOutcome(commandResult),
		WhenUnix:  h.control.now().UTC().UnixMilli(),
	})
	if commandResult.ScenarioStart == "multi_window" && defaultAction(cmd.Action) == CommandActionStart {
		h.captureMultiWindowResume(cmd.DeviceID, priorActiveScenario)
	}
	if cmd.RequestID != "" {
		commandResult.CommandAck = cmd.RequestID
		h.mu.Lock()
		h.seen[cmd.RequestID] = commandResult
		h.seenOrder = append(h.seenOrder, cmd.RequestID)
		if len(h.seenOrder) > h.seenLimit {
			evict := h.seenOrder[0]
			h.seenOrder = h.seenOrder[1:]
			delete(h.seen, evict)
		}
		h.mu.Unlock()
	}
	postResponses := h.commandResponses(ctx, cmd, commandResult)
	afterRoutes := h.routeSnapshotForDevice(cmd.DeviceID)
	routeUpdates := h.routeUpdatesForCommand(cmd, commandResult, beforeRoutes, afterRoutes)
	if len(routeUpdates) > 0 {
		postResponses = append(postResponses, routeUpdates...)
	}
	paTransitions := h.paTransitionsForCommand(cmd, commandResult, beforeRoutes, afterRoutes)
	if len(paTransitions) > 0 {
		postResponses = append(postResponses, paTransitions...)
	}
	overlayClears := h.paOverlayClearsForCommand(cmd, commandResult, beforeRoutes)
	if len(overlayClears) > 0 {
		postResponses = append(postResponses, overlayClears...)
	}
	broadcastNotifications := d.BroadcastNotificationsForCommand(cmd, commandResult, beforeBroadcastEvents)
	if len(broadcastNotifications) > 0 {
		postResponses = append(postResponses, broadcastNotifications...)
	}
	uiMessages := h.uiHostMessagesSince(beforeUIEvents, cmd.DeviceID, true)
	if len(uiMessages) > 0 {
		postResponses = append(postResponses, uiMessages...)
	}
	h.uiSession.RememberSetUI(cmd.DeviceID, postResponses)
	return postResponses, nil
}

// BroadcastNotificationsForCommand fans out broadcast events that were
// emitted by the runtime while the command body executed. It only
// produces output when the command resulted in a scenario start or stop
// — other command outcomes do not trigger broadcast fan-out.
func (d *CommandDispatcher) BroadcastNotificationsForCommand(
	cmd *CommandRequest,
	commandResult ServerMessage,
	beforeCount int,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	if commandResult.ScenarioStart == "" && commandResult.ScenarioStop == "" {
		return nil
	}
	return d.h.broadcastNotificationsSince(beforeCount, cmd.DeviceID, false)
}

// Recent returns a copy of the recent-command audit buffer.
func (d *CommandDispatcher) Recent() []CommandEvent {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]CommandEvent(nil), d.recent...)
}

// SetRecentLimit updates the audit buffer limit and trims existing
// events if needed. Limits below 1 keep the buffer empty.
func (d *CommandDispatcher) SetRecentLimit(limit int) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.recentLimit = limit
	d.trimRecentLocked()
}

// appendEvent appends ev to the recent-command audit buffer and trims it
// to recentLimit.
func (d *CommandDispatcher) appendEvent(ev CommandEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.recent = append(d.recent, ev)
	d.trimRecentLocked()
}

func (d *CommandDispatcher) trimRecentLocked() {
	if d.recentLimit < 1 {
		d.recent = nil
		return
	}
	if len(d.recent) > d.recentLimit {
		d.recent = d.recent[len(d.recent)-d.recentLimit:]
	}
}
