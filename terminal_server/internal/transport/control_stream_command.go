package transport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func manualPassthroughTrigger(cmd *CommandRequest) (scenario.Trigger, bool) {
	if cmd == nil {
		return scenario.Trigger{}, false
	}
	intent := strings.TrimSpace(cmd.Intent)
	switch intent {
	case ManualIntentBluetoothScan:
		args := copyStringMap(cmd.Arguments)
		if strings.TrimSpace(args["action"]) == "" {
			args["action"] = "scan"
		}
		return scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    "bluetooth_passthrough",
			Arguments: args,
			IntentV2: &scenario.IntentRecord{
				Action: "bluetooth_passthrough",
				Slots:  copyStringMap(args),
				Source: scenario.SourceManual,
			},
		}, true
	case ManualIntentBluetoothConnect:
		args := copyStringMap(cmd.Arguments)
		if strings.TrimSpace(args["action"]) == "" {
			args["action"] = "connect"
		}
		if strings.TrimSpace(args["target_id"]) == "" {
			if target := strings.TrimSpace(args["target"]); target != "" {
				args["target_id"] = target
			}
		}
		return scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    "bluetooth_passthrough",
			Arguments: args,
			IntentV2: &scenario.IntentRecord{
				Action: "bluetooth_passthrough",
				Slots:  copyStringMap(args),
				Source: scenario.SourceManual,
			},
		}, true
	case ManualIntentUSBEnumerate:
		args := copyStringMap(cmd.Arguments)
		if strings.TrimSpace(args["action"]) == "" {
			args["action"] = "enumerate"
		}
		return scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    "usb_passthrough",
			Arguments: args,
			IntentV2: &scenario.IntentRecord{
				Action: "usb_passthrough",
				Slots:  copyStringMap(args),
				Source: scenario.SourceManual,
			},
		}, true
	case ManualIntentUSBClaim:
		args := copyStringMap(cmd.Arguments)
		if strings.TrimSpace(args["action"]) == "" {
			args["action"] = "claim"
		}
		return scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    "usb_passthrough",
			Arguments: args,
			IntentV2: &scenario.IntentRecord{
				Action: "usb_passthrough",
				Slots:  copyStringMap(args),
				Source: scenario.SourceManual,
			},
		}, true
	default:
		return scenario.Trigger{}, false
	}
}

// handleVoiceAudio accumulates inbound mic audio per device and, on IsFinal,
// runs STT on the assembled buffer, drives the voice command pipeline through
// Runtime.HandleVoiceText, then synthesizes the resulting response via TTS and
// returns it as a PlayAudio server message targeted at the source device.
func (h *StreamHandler) handleVoiceAudio(ctx context.Context, va *VoiceAudioRequest) ([]ServerMessage, error) {
	return h.voicePipeline.HandleAudio(ctx, va)
}

func (h *StreamHandler) deviceAllowsVoiceAudio(deviceID string) bool {
	if h == nil || h.control == nil || h.control.devices == nil {
		return true
	}
	current, ok := h.control.devices.Get(strings.TrimSpace(deviceID))
	if !ok {
		return true
	}
	if current.Generation == 0 {
		return true
	}
	return truthyCapability(current.Capabilities["microphone.present"]) ||
		truthyCapability(current.Capabilities["microphone.endpoint_count"])
}

// latestBroadcastForDevice returns the most recent broadcast message emitted
// after beforeCount that targets deviceID (or the most recent message overall
// if none explicitly target the device). Returns "" if no new events exist.
func (h *StreamHandler) latestBroadcastForDevice(deviceID string, beforeCount int) string {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Broadcast == nil {
		return ""
	}
	eventReader, ok := h.runtime.Env.Broadcast.(interface {
		Events() []ui.BroadcastEvent
	})
	if !ok {
		return ""
	}
	events := eventReader.Events()
	if beforeCount < 0 {
		beforeCount = 0
	}
	if beforeCount > len(events) {
		beforeCount = len(events)
	}
	newEvents := events[beforeCount:]
	if len(newEvents) == 0 {
		return ""
	}
	deviceID = strings.TrimSpace(deviceID)
	fallback := ""
	for _, event := range newEvents {
		fallback = event.Message
		for _, target := range event.DeviceIDs {
			if strings.TrimSpace(target) == deviceID {
				return event.Message
			}
		}
	}
	return fallback
}

func (h *StreamHandler) handleCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	kind := cmd.Kind
	if kind == "" {
		kind = CommandKindManual
	}

	if kind == CommandKindSystem {
		return h.handleSystemCommand(ctx, cmd)
	}
	if strings.TrimSpace(cmd.DeviceID) == "" {
		return ServerMessage{}, ErrMissingCommandDeviceID
	}

	action := cmd.Action
	if action == "" {
		action = CommandActionStart
	}
	if action != CommandActionStart && action != CommandActionStop {
		return ServerMessage{}, ErrInvalidCommandAction
	}
	manualIntent := strings.TrimSpace(cmd.Intent)
	if h.runtime == nil {
		if kind != CommandKindManual ||
			(manualIntent != SystemIntentTerminalRefresh && manualIntent != ManualIntentPlaybackMetadata) {
			return ServerMessage{}, errors.New("scenario runtime not configured")
		}
	}

	switch kind {
	case CommandKindVoice:
		if strings.TrimSpace(cmd.Text) == "" {
			return ServerMessage{}, ErrMissingCommandText
		}
		if action == CommandActionStop {
			name, err := h.runtime.StopVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
			if err != nil {
				return ServerMessage{}, err
			}
			return ServerMessage{
				ScenarioStop: name,
				Notification: "Scenario stopped: " + name,
			}, nil
		}
		name, err := h.runtime.HandleVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
		if err != nil {
			return ServerMessage{}, err
		}
		return ServerMessage{
			ScenarioStart: name,
			Notification:  "Scenario started: " + name,
		}, nil
	case CommandKindManual:
		if manualIntent == "" {
			return ServerMessage{}, ErrMissingCommandIntent
		}
		if manualIntent == SystemIntentTerminalRefresh {
			if action == CommandActionStop {
				return ServerMessage{}, ErrInvalidCommandAction
			}
			return ServerMessage{
				Notification: "Terminal refresh requested",
				Data: map[string]string{
					"device_id": cmd.DeviceID,
				},
			}, nil
		}
		if manualIntent == ManualIntentPlaybackMetadata {
			if action == CommandActionStop {
				return ServerMessage{}, ErrInvalidCommandAction
			}
			artifactID := strings.TrimSpace(cmd.Arguments["artifact_id"])
			if artifactID == "" {
				return ServerMessage{}, fmt.Errorf("playback_metadata requires artifact_id")
			}
			targetDeviceID := strings.TrimSpace(cmd.Arguments["target_device_id"])
			if targetDeviceID == "" {
				targetDeviceID = strings.TrimSpace(cmd.DeviceID)
			}
			if targetDeviceID == "" {
				return ServerMessage{}, ErrMissingCommandDeviceID
			}
			metadata, ok := h.playbackMetadataForTarget(artifactID, targetDeviceID)
			if !ok {
				return ServerMessage{}, fmt.Errorf("playback artifact not found: %s", artifactID)
			}
			return ServerMessage{
				Notification: "Playback metadata ready",
				Data:         metadata,
			}, nil
		}
		if passthroughTrigger, ok := manualPassthroughTrigger(cmd); ok {
			if action == CommandActionStop {
				return ServerMessage{}, ErrInvalidCommandAction
			}
			name, err := h.runtime.HandleTrigger(ctx, passthroughTrigger)
			if err != nil {
				return ServerMessage{}, err
			}
			return ServerMessage{
				ScenarioStart: name,
				Notification:  "Scenario started: " + name,
			}, nil
		}
		trigger := scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    cmd.Intent,
			Arguments: copyStringMap(cmd.Arguments),
			IntentV2: &scenario.IntentRecord{
				Action: strings.TrimSpace(cmd.Intent),
				Slots:  copyStringMap(cmd.Arguments),
				Source: scenario.SourceManual,
			},
		}
		if action == CommandActionStop {
			name, err := h.runtime.StopTrigger(ctx, trigger)
			if err != nil {
				return ServerMessage{}, err
			}
			return ServerMessage{
				ScenarioStop: name,
				Notification: "Scenario stopped: " + name,
			}, nil
		}
		name, err := h.runtime.HandleTrigger(ctx, trigger)
		if err != nil {
			return ServerMessage{}, err
		}
		return ServerMessage{
			ScenarioStart: name,
			Notification:  "Scenario started: " + name,
		}, nil
	default:
		return ServerMessage{}, ErrInvalidCommandKind
	}
}

func (h *StreamHandler) handleSystemCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	parsed, err := ParseSystemIntent(cmd.Intent)
	if err != nil {
		return ServerMessage{}, err
	}
	switch parsed.Name {
	case SystemIntentHelp:
		return ServerMessage{
			Notification: "System query: system_help",
			Data: map[string]string{
				"system_intents":  SystemHelpIntentsString(),
				"command_kinds":   "voice,manual,system",
				"command_actions": "start,stop",
			},
		}, nil
	case SystemIntentServerStatus:
		return ServerMessage{
			Notification: "System query: server_status",
			Data:         h.control.StatusData(),
		}, nil
	case SystemIntentRuntimeStatus:
		data := map[string]string{}
		if h.runtime != nil {
			for k, v := range h.runtime.StatusData() {
				data[k] = v
			}
		}
		for k, v := range h.mediaStreamStatusData() {
			data[k] = v
		}
		for k, v := range h.sensorStatusData() {
			data[k] = v
		}
		for k, v := range h.recordingStatusData() {
			data[k] = v
		}
		return ServerMessage{
			Notification: "System query: runtime_status",
			Data:         data,
		}, nil
	case SystemIntentScenarioRegistry:
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Engine != nil {
			for _, item := range h.runtime.Engine.RegistrySnapshot() {
				data[item.Name] = fmt.Sprintf("priority=%d", item.Priority)
			}
		}
		return ServerMessage{
			Notification: "System query: scenario_registry",
			Data:         data,
		}, nil
	case SystemIntentRunDueTimers:
		processed := 0
		if h.runtime != nil {
			count, err := h.runtime.ProcessDueTimers(ctx, h.control.now().UTC())
			if err != nil {
				return ServerMessage{}, err
			}
			processed = count
		}
		return ServerMessage{
			Notification: "System query: run_due_timers",
			Data: map[string]string{
				"processed": toString(int64(processed)),
			},
		}, nil
	case SystemIntentReconcileLiveness:
		timeout := 2 * time.Minute
		timeoutSeconds := "120"
		if parsed.Arg != "" {
			seconds, convErr := strconv.Atoi(parsed.Arg)
			if convErr != nil || seconds < 0 {
				return ServerMessage{}, fmt.Errorf("invalid reconcile_liveness seconds: %s", parsed.Arg)
			}
			timeout = time.Duration(seconds) * time.Second
			timeoutSeconds = parsed.Arg
		}
		updated := h.control.ReconcileLiveness(timeout)
		return ServerMessage{
			Notification: "System query: reconcile_liveness",
			Data: map[string]string{
				"updated":         toString(int64(updated)),
				"timeout_seconds": timeoutSeconds,
			},
		}, nil
	case SystemIntentTransportMetrics:
		data := map[string]string{}
		if h.metrics != nil {
			for k, v := range h.metrics.Snapshot() {
				data[k] = v
			}
		}
		return ServerMessage{
			Notification: "System query: transport_metrics",
			Data:         data,
		}, nil
	case SystemIntentListDevices:
		data := map[string]string{}
		devices := h.control.devices.List()
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].DeviceID < devices[j].DeviceID
		})
		for _, d := range devices {
			data[d.DeviceID] = fmt.Sprintf("%s|%s|%s", d.DeviceName, d.Platform, d.State)
		}
		return ServerMessage{
			Notification: "System query: list_devices",
			Data:         data,
		}, nil
	case SystemIntentActiveScenarios:
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Engine != nil {
			for deviceID, scenarioName := range h.runtime.Engine.ActiveSnapshot() {
				data[deviceID] = scenarioName
			}
		}
		return ServerMessage{
			Notification: "System query: active_scenarios",
			Data:         data,
		}, nil
	case SystemIntentPendingTimers:
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Env != nil && h.runtime.Env.Scheduler != nil {
			if structured, ok := h.runtime.Env.Scheduler.(interface {
				DueRecords(int64) []storage.ScheduleRecord
			}); ok {
				for _, record := range structured.DueRecords(math.MaxInt64) {
					data[record.Key] = pendingTimerRecordValue(record)
				}
			} else {
				for _, key := range h.runtime.Env.Scheduler.Due(math.MaxInt64) {
					data[key] = "scheduled"
				}
			}
		}
		return ServerMessage{
			Notification: "System query: pending_timers",
			Data:         data,
		}, nil
	case SystemIntentRecentCommands:
		data := map[string]string{}
		events := h.commandDispatcher.Recent()
		for i, ev := range events {
			key := fmt.Sprintf("%03d", i)
			data[key] = strings.Join([]string{
				ev.RequestID,
				ev.DeviceID,
				ev.Kind,
				ev.Action,
				ev.Intent,
				ev.Outcome,
				strconv.FormatInt(ev.WhenUnix, 10),
			}, "|")
		}
		return ServerMessage{
			Notification: "System query: recent_commands",
			Data:         data,
		}, nil
	case SystemIntentRecordingEvents:
		data := map[string]string{}
		var events []recording.Event
		if h.mediaControl != nil {
			events = h.mediaControl.RecentRecordingEvents(50)
		}
		for i, event := range events {
			key := fmt.Sprintf("%03d", i)
			data[key] = strings.Join([]string{
				strconv.FormatInt(event.AtUnixMS, 10),
				event.Action,
				event.StreamID,
				event.Kind,
				event.SourceID,
				event.TargetID,
			}, "|")
		}
		return ServerMessage{
			Notification: "System query: recording_events",
			Data:         data,
		}, nil
	case SystemIntentListPlaybackFiles:
		data := map[string]string{}
		for i, artifact := range h.listPlaybackArtifacts() {
			key := fmt.Sprintf("%03d", i)
			data[key] = strings.Join([]string{
				artifact.ArtifactID,
				artifact.StreamID,
				artifact.Kind,
				artifact.SourceDeviceID,
				artifact.TargetDeviceID,
				strconv.FormatInt(artifact.SizeBytes, 10),
				strconv.FormatInt(artifact.UpdatedUnixMS, 10),
				artifact.AudioPath,
			}, "|")
		}
		return ServerMessage{
			Notification: "System query: list_playback_artifacts",
			Data:         data,
		}, nil
	case SystemIntentTerminalRefresh:
		targetDeviceID := strings.TrimSpace(parsed.Arg)
		if targetDeviceID == "" {
			targetDeviceID = strings.TrimSpace(cmd.DeviceID)
		}
		if targetDeviceID == "" {
			return ServerMessage{}, ErrMissingCommandDeviceID
		}
		return ServerMessage{
			Notification: "System query: terminal_refresh",
			Data: map[string]string{
				"device_id": targetDeviceID,
			},
		}, nil
	default:
		if parsed.Name == SystemIntentDeviceStatus && parsed.Arg != "" {
			deviceState, ok := h.control.devices.Get(parsed.Arg)
			if !ok {
				return ServerMessage{}, fmt.Errorf("device not found: %s", parsed.Arg)
			}
			data := map[string]string{
				"device_id":   deviceState.DeviceID,
				"device_name": deviceState.DeviceName,
				"device_type": deviceState.DeviceType,
				"platform":    deviceState.Platform,
				"state":       string(deviceState.State),
			}
			for k, v := range deviceState.Capabilities {
				data["cap."+k] = v
			}
			if snapshot, ok := h.sensorDataForDevice(parsed.Arg); ok {
				data["sensor.unix_ms"] = strconv.FormatInt(snapshot.UnixMS, 10)
				keys := make([]string, 0, len(snapshot.Values))
				for key := range snapshot.Values {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					data["sensor."+key] = strconv.FormatFloat(snapshot.Values[key], 'f', -1, 64)
				}
			}
			return ServerMessage{
				Notification: "System query: device_status",
				Data:         data,
			}, nil
		}
		return ServerMessage{}, fmt.Errorf("unknown system intent: %s", cmd.Intent)
	}
}

func pendingTimerRecordValue(record storage.ScheduleRecord) string {
	if record.Kind == "" && record.Subject == "" && record.DeviceID == "" && len(record.Payload) == 0 {
		return "scheduled"
	}
	parts := []string{}
	if record.Kind != "" {
		parts = append(parts, "kind="+record.Kind)
	}
	if record.DeviceID != "" {
		parts = append(parts, "device="+record.DeviceID)
	}
	if record.Subject != "" {
		parts = append(parts, "subject="+record.Subject)
	}
	if duration := strings.TrimSpace(record.Payload["duration_seconds"]); duration != "" {
		parts = append(parts, "duration_seconds="+duration)
	}
	if len(parts) == 0 {
		return "scheduled"
	}
	return strings.Join(parts, "|")
}

func (h *StreamHandler) listPlaybackArtifacts() []recording.Artifact {
	if h.mediaControl == nil {
		return nil
	}
	return h.mediaControl.ListPlaybackArtifacts()
}

func (h *StreamHandler) playbackMetadataForTarget(artifactID, targetDeviceID string) (map[string]string, bool) {
	if h.mediaControl == nil {
		return nil, false
	}
	metadata, ok := h.mediaControl.PlaybackMetadataForTarget(artifactID, targetDeviceID)
	if !ok {
		return nil, false
	}
	return map[string]string{
		"artifact_id":      metadata.Artifact.ArtifactID,
		"stream_id":        metadata.Artifact.StreamID,
		"kind":             metadata.Artifact.Kind,
		"source_device_id": metadata.Artifact.SourceDeviceID,
		"target_device_id": metadata.TargetDeviceID,
		"audio_path":       metadata.Artifact.AudioPath,
		"format":           "pcm16",
		"size_bytes":       strconv.FormatInt(metadata.Artifact.SizeBytes, 10),
		"updated_unix_ms":  strconv.FormatInt(metadata.Artifact.UpdatedUnixMS, 10),
	}, true
}

// NoteProtocolError increments protocol error counters from session-level validation.
