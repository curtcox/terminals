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
	switch strings.TrimSpace(cmd.Intent) {
	case ManualIntentBluetoothScan:
		return manualPassthroughTriggerFor(cmd, "bluetooth_passthrough", "scan", nil), true
	case ManualIntentBluetoothConnect:
		return manualPassthroughTriggerFor(cmd, "bluetooth_passthrough", "connect", fillBluetoothConnectTarget), true
	case ManualIntentUSBEnumerate:
		return manualPassthroughTriggerFor(cmd, "usb_passthrough", "enumerate", nil), true
	case ManualIntentUSBClaim:
		return manualPassthroughTriggerFor(cmd, "usb_passthrough", "claim", nil), true
	default:
		return scenario.Trigger{}, false
	}
}

func manualPassthroughTriggerFor(
	cmd *CommandRequest,
	intentName, defaultAction string,
	patch func(map[string]string),
) scenario.Trigger {
	args := copyStringMap(cmd.Arguments)
	if strings.TrimSpace(args["action"]) == "" {
		args["action"] = defaultAction
	}
	if patch != nil {
		patch(args)
	}
	return scenario.Trigger{
		Kind:      scenario.TriggerManual,
		SourceID:  cmd.DeviceID,
		Intent:    intentName,
		Arguments: args,
		IntentV2: &scenario.IntentRecord{
			Action: intentName,
			Slots:  copyStringMap(args),
			Source: scenario.SourceManual,
		},
	}
}

func fillBluetoothConnectTarget(args map[string]string) {
	if strings.TrimSpace(args["target_id"]) != "" {
		return
	}
	if target := strings.TrimSpace(args["target"]); target != "" {
		args["target_id"] = target
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
		return h.handleVoiceCommand(ctx, cmd, action)
	case CommandKindManual:
		return h.handleManualCommand(ctx, cmd, action, manualIntent)
	default:
		return ServerMessage{}, ErrInvalidCommandKind
	}
}

func (h *StreamHandler) handleVoiceCommand(ctx context.Context, cmd *CommandRequest, action string) (ServerMessage, error) {
	if strings.TrimSpace(cmd.Text) == "" {
		return ServerMessage{}, ErrMissingCommandText
	}
	if action == CommandActionStop {
		name, err := h.runtime.StopVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
		if err != nil {
			return ServerMessage{}, err
		}
		return scenarioStoppedMessage(name), nil
	}
	name, err := h.runtime.HandleVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
	if err != nil {
		return ServerMessage{}, err
	}
	return scenarioStartedMessage(name), nil
}

func (h *StreamHandler) handleManualCommand(ctx context.Context, cmd *CommandRequest, action, manualIntent string) (ServerMessage, error) {
	if manualIntent == "" {
		return ServerMessage{}, ErrMissingCommandIntent
	}
	if manualIntent == SystemIntentTerminalRefresh {
		return terminalRefreshCommandMessage(cmd.DeviceID, action)
	}
	if manualIntent == ManualIntentPlaybackMetadata {
		return h.playbackMetadataCommandMessage(cmd, action)
	}
	if passthroughTrigger, ok := manualPassthroughTrigger(cmd); ok {
		return h.handlePassthroughTrigger(ctx, passthroughTrigger, action)
	}
	return h.handleManualScenarioTrigger(ctx, cmd, action)
}

func terminalRefreshCommandMessage(deviceID, action string) (ServerMessage, error) {
	if action == CommandActionStop {
		return ServerMessage{}, ErrInvalidCommandAction
	}
	return ServerMessage{
		Notification: "Terminal refresh requested",
		Data: map[string]string{
			"device_id": deviceID,
		},
	}, nil
}

func (h *StreamHandler) playbackMetadataCommandMessage(cmd *CommandRequest, action string) (ServerMessage, error) {
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

func (h *StreamHandler) handlePassthroughTrigger(ctx context.Context, trigger scenario.Trigger, action string) (ServerMessage, error) {
	if action == CommandActionStop {
		return ServerMessage{}, ErrInvalidCommandAction
	}
	name, err := h.runtime.HandleTrigger(ctx, trigger)
	if err != nil {
		return ServerMessage{}, err
	}
	return scenarioStartedMessage(name), nil
}

func (h *StreamHandler) handleManualScenarioTrigger(ctx context.Context, cmd *CommandRequest, action string) (ServerMessage, error) {
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
		return scenarioStoppedMessage(name), nil
	}
	name, err := h.runtime.HandleTrigger(ctx, trigger)
	if err != nil {
		return ServerMessage{}, err
	}
	return scenarioStartedMessage(name), nil
}

func scenarioStartedMessage(name string) ServerMessage {
	return ServerMessage{
		ScenarioStart: name,
		Notification:  "Scenario started: " + name,
	}
}

func scenarioStoppedMessage(name string) ServerMessage {
	return ServerMessage{
		ScenarioStop: name,
		Notification: "Scenario stopped: " + name,
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
	if msg, ok, err := h.systemStatusCommand(ctx, parsed); ok || err != nil {
		return msg, err
	}
	if msg, ok := h.systemInventoryCommand(parsed); ok {
		return msg, nil
	}
	if parsed.Name == SystemIntentTerminalRefresh {
		return h.systemTerminalRefreshMessage(cmd.DeviceID, parsed.Arg)
	}
	if parsed.Name == SystemIntentDeviceStatus && parsed.Arg != "" {
		data, err := h.deviceStatusData(parsed.Arg)
		if err != nil {
			return ServerMessage{}, err
		}
		return ServerMessage{
			Notification: "System query: device_status",
			Data:         data,
		}, nil
	}
	return ServerMessage{}, fmt.Errorf("unknown system intent: %s", cmd.Intent)
}

func (h *StreamHandler) systemStatusCommand(ctx context.Context, parsed ParsedSystemIntent) (ServerMessage, bool, error) {
	switch parsed.Name {
	case SystemIntentHelp:
		return ServerMessage{
			Notification: "System query: system_help",
			Data: map[string]string{
				"system_intents":  SystemHelpIntentsString(),
				"command_kinds":   "voice,manual,system",
				"command_actions": "start,stop",
			},
		}, true, nil
	case SystemIntentServerStatus:
		return ServerMessage{
			Notification: "System query: server_status",
			Data:         h.control.StatusData(),
		}, true, nil
	case SystemIntentRuntimeStatus:
		return ServerMessage{
			Notification: "System query: runtime_status",
			Data:         h.runtimeStatusData(),
		}, true, nil
	case SystemIntentScenarioRegistry:
		return ServerMessage{
			Notification: "System query: scenario_registry",
			Data:         h.scenarioRegistryData(),
		}, true, nil
	case SystemIntentRunDueTimers:
		processed := 0
		if h.runtime != nil {
			count, err := h.runtime.ProcessDueTimers(ctx, h.control.now().UTC())
			if err != nil {
				return ServerMessage{}, true, err
			}
			processed = count
		}
		return ServerMessage{
			Notification: "System query: run_due_timers",
			Data: map[string]string{
				"processed": toString(int64(processed)),
			},
		}, true, nil
	case SystemIntentReconcileLiveness:
		timeout := 2 * time.Minute
		timeoutSeconds := "120"
		if parsed.Arg != "" {
			seconds, convErr := strconv.Atoi(parsed.Arg)
			if convErr != nil || seconds < 0 {
				return ServerMessage{}, true, fmt.Errorf("invalid reconcile_liveness seconds: %s", parsed.Arg)
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
		}, true, nil
	case SystemIntentTransportMetrics:
		return ServerMessage{
			Notification: "System query: transport_metrics",
			Data:         h.transportMetricsData(),
		}, true, nil
	default:
		return ServerMessage{}, false, nil
	}
}

func (h *StreamHandler) systemInventoryCommand(parsed ParsedSystemIntent) (ServerMessage, bool) {
	switch parsed.Name {
	case SystemIntentListDevices:
		return ServerMessage{
			Notification: "System query: list_devices",
			Data:         h.listDevicesData(),
		}, true
	case SystemIntentActiveScenarios:
		return ServerMessage{
			Notification: "System query: active_scenarios",
			Data:         h.activeScenariosData(),
		}, true
	case SystemIntentPendingTimers:
		return ServerMessage{
			Notification: "System query: pending_timers",
			Data:         h.pendingTimersData(),
		}, true
	case SystemIntentRecentCommands:
		return ServerMessage{
			Notification: "System query: recent_commands",
			Data:         h.recentCommandsData(),
		}, true
	case SystemIntentRecordingEvents:
		return ServerMessage{
			Notification: "System query: recording_events",
			Data:         h.recordingEventsData(),
		}, true
	case SystemIntentListPlaybackFiles:
		return ServerMessage{
			Notification: "System query: list_playback_artifacts",
			Data:         h.listPlaybackArtifactsData(),
		}, true
	default:
		return ServerMessage{}, false
	}
}

func (h *StreamHandler) systemTerminalRefreshMessage(commandDeviceID, arg string) (ServerMessage, error) {
	targetDeviceID := strings.TrimSpace(arg)
	if targetDeviceID == "" {
		targetDeviceID = strings.TrimSpace(commandDeviceID)
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
}

func (h *StreamHandler) runtimeStatusData() map[string]string {
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
	return data
}

func (h *StreamHandler) scenarioRegistryData() map[string]string {
	data := map[string]string{}
	if h.runtime == nil || h.runtime.Engine == nil {
		return data
	}
	for _, item := range h.runtime.Engine.RegistrySnapshot() {
		data[item.Name] = fmt.Sprintf("priority=%d", item.Priority)
	}
	return data
}

func (h *StreamHandler) transportMetricsData() map[string]string {
	data := map[string]string{}
	if h.metrics == nil {
		return data
	}
	for k, v := range h.metrics.Snapshot() {
		data[k] = v
	}
	return data
}

func (h *StreamHandler) listDevicesData() map[string]string {
	data := map[string]string{}
	devices := h.control.devices.List()
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DeviceID < devices[j].DeviceID
	})
	for _, d := range devices {
		data[d.DeviceID] = fmt.Sprintf("%s|%s|%s", d.DeviceName, d.Platform, d.State)
	}
	return data
}

func (h *StreamHandler) activeScenariosData() map[string]string {
	data := map[string]string{}
	if h.runtime == nil || h.runtime.Engine == nil {
		return data
	}
	for deviceID, scenarioName := range h.runtime.Engine.ActiveSnapshot() {
		data[deviceID] = scenarioName
	}
	return data
}

func (h *StreamHandler) pendingTimersData() map[string]string {
	data := map[string]string{}
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Scheduler == nil {
		return data
	}
	if structured, ok := h.runtime.Env.Scheduler.(interface {
		DueRecords(int64) []storage.ScheduleRecord
	}); ok {
		for _, record := range structured.DueRecords(math.MaxInt64) {
			data[record.Key] = pendingTimerRecordValue(record)
		}
		return data
	}
	for _, key := range h.runtime.Env.Scheduler.Due(math.MaxInt64) {
		data[key] = "scheduled"
	}
	return data
}

func (h *StreamHandler) recentCommandsData() map[string]string {
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
	return data
}

func (h *StreamHandler) recordingEventsData() map[string]string {
	data := map[string]string{}
	if h.mediaControl == nil {
		return data
	}
	for i, event := range h.mediaControl.RecentRecordingEvents(50) {
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
	return data
}

func (h *StreamHandler) listPlaybackArtifactsData() map[string]string {
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
	return data
}

func (h *StreamHandler) deviceStatusData(deviceID string) (map[string]string, error) {
	deviceState, ok := h.control.devices.Get(deviceID)
	if !ok {
		return nil, fmt.Errorf("device not found: %s", deviceID)
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
	h.addDeviceSensorData(data, deviceID)
	return data, nil
}

func (h *StreamHandler) addDeviceSensorData(data map[string]string, deviceID string) {
	snapshot, ok := h.sensorDataForDevice(deviceID)
	if !ok {
		return
	}
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
