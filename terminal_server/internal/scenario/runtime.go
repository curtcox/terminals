package scenario

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

// ErrNoMatchingScenario indicates no scenario handled the trigger.
var ErrNoMatchingScenario = errors.New("no matching scenario")

// Runtime coordinates trigger matching and activation.
type Runtime struct {
	Engine *Engine
	Env    *Environment
}

// NewRuntime creates a runtime with engine and environment.
func NewRuntime(engine *Engine, env *Environment) *Runtime {
	return &Runtime{
		Engine: engine,
		Env:    env,
	}
}

// HandleTrigger matches and activates a scenario for the selected devices.
func (r *Runtime) HandleTrigger(ctx context.Context, trigger Trigger) (string, error) {
	match, ok := r.Engine.MatchActivation(ActivationRequest{
		Trigger:     trigger,
		RequestedAt: time.Now().UTC(),
	})
	if !ok {
		return "", ErrNoMatchingScenario
	}

	deviceIDs := targetDevices(ctx, r.Env, trigger)
	if err := r.Engine.ActivateMatched(ctx, r.Env, match, deviceIDs); err != nil {
		return "", err
	}
	return match.Registration.name(), nil
}

// HandleVoiceText parses spoken text and routes to HandleTrigger.
func (r *Runtime) HandleVoiceText(ctx context.Context, sourceID, spoken string, now time.Time) (string, error) {
	return r.HandleTrigger(ctx, ParseVoiceTrigger(sourceID, spoken, now))
}

// StopTrigger matches and stops a scenario for the selected devices.
func (r *Runtime) StopTrigger(ctx context.Context, trigger Trigger) (string, error) {
	match, ok := r.Engine.MatchActivation(ActivationRequest{
		Trigger:     trigger,
		RequestedAt: time.Now().UTC(),
	})
	if !ok {
		return "", ErrNoMatchingScenario
	}

	deviceIDs := targetDevices(ctx, r.Env, trigger)
	name := match.Registration.name()
	if err := r.Engine.Stop(ctx, r.Env, name, deviceIDs); err != nil {
		return "", err
	}
	return name, nil
}

// StopVoiceText parses spoken text and routes to StopTrigger.
func (r *Runtime) StopVoiceText(ctx context.Context, sourceID, spoken string, now time.Time) (string, error) {
	return r.StopTrigger(ctx, ParseVoiceTrigger(sourceID, spoken, now))
}

// StartScenario requests scenario activation by scenario name and target
// devices. This helper is intended for administrative controls where no
// natural voice/manual trigger text exists.
func (r *Runtime) StartScenario(ctx context.Context, scenarioName string, deviceIDs []string) (string, error) {
	deviceIDs = normalizeDeviceIDs(deviceIDs)
	args := map[string]string{}
	if len(deviceIDs) > 0 {
		args["device_ids"] = strings.Join(deviceIDs, ",")
	}
	sourceID := ""
	if len(deviceIDs) > 0 {
		sourceID = deviceIDs[0]
	}
	return r.HandleTrigger(ctx, Trigger{
		Kind:      TriggerManual,
		SourceID:  sourceID,
		Intent:    strings.TrimSpace(scenarioName),
		Arguments: args,
	})
}

// StopScenario requests scenario stop by scenario name and target devices.
// This helper is intended for administrative controls where no natural
// stop trigger text exists.
func (r *Runtime) StopScenario(ctx context.Context, scenarioName string, deviceIDs []string) (string, error) {
	deviceIDs = normalizeDeviceIDs(deviceIDs)
	args := map[string]string{}
	if len(deviceIDs) > 0 {
		args["device_ids"] = strings.Join(deviceIDs, ",")
	}
	sourceID := ""
	if len(deviceIDs) > 0 {
		sourceID = deviceIDs[0]
	}
	return r.StopTrigger(ctx, Trigger{
		Kind:      TriggerManual,
		SourceID:  sourceID,
		Intent:    strings.TrimSpace(scenarioName),
		Arguments: args,
	})
}

// ProcessSensorReading dispatches telemetry snapshots to the active scenario
// for the source device when that scenario declares SensorConsumer support.
func (r *Runtime) ProcessSensorReading(ctx context.Context, reading SensorReading) error {
	if r == nil || r.Engine == nil || r.Env == nil {
		return nil
	}
	activeScenario, ok := r.Engine.ActiveScenario(strings.TrimSpace(reading.DeviceID))
	if !ok {
		return nil
	}
	consumer, ok := activeScenario.(SensorConsumer)
	if !ok {
		return nil
	}
	return consumer.HandleSensor(ctx, r.Env, reading)
}

// DispatchBluetoothCommand sends a Bluetooth passthrough command via the
// configured bridge when available.
func (r *Runtime) DispatchBluetoothCommand(ctx context.Context, cmd BluetoothCommand) error {
	if r == nil || r.Env == nil || r.Env.Passthrough == nil {
		return nil
	}
	cmd.DeviceID = strings.TrimSpace(cmd.DeviceID)
	cmd.Action = strings.TrimSpace(cmd.Action)
	cmd.TargetID = strings.TrimSpace(cmd.TargetID)
	if cmd.Parameters == nil {
		cmd.Parameters = map[string]string{}
	} else {
		cmd.Parameters = copyStringMap(cmd.Parameters)
	}
	return r.Env.Passthrough.DispatchBluetoothCommand(ctx, cmd)
}

// DispatchUSBCommand sends a USB passthrough command via the configured
// bridge when available.
func (r *Runtime) DispatchUSBCommand(ctx context.Context, cmd USBCommand) error {
	if r == nil || r.Env == nil || r.Env.Passthrough == nil {
		return nil
	}
	cmd.DeviceID = strings.TrimSpace(cmd.DeviceID)
	cmd.Action = strings.TrimSpace(cmd.Action)
	cmd.VendorID = strings.TrimSpace(cmd.VendorID)
	cmd.ProductID = strings.TrimSpace(cmd.ProductID)
	if cmd.Parameters == nil {
		cmd.Parameters = map[string]string{}
	} else {
		cmd.Parameters = copyStringMap(cmd.Parameters)
	}
	return r.Env.Passthrough.DispatchUSBCommand(ctx, cmd)
}

// ProcessBluetoothEvent dispatches a Bluetooth passthrough event to the active
// scenario for the source device when that scenario supports the hook.
func (r *Runtime) ProcessBluetoothEvent(ctx context.Context, event BluetoothEvent) error {
	if r == nil || r.Engine == nil || r.Env == nil {
		return nil
	}
	deviceID := strings.TrimSpace(event.DeviceID)
	activeScenario, ok := r.Engine.ActiveScenario(deviceID)
	if !ok {
		return nil
	}
	consumer, ok := activeScenario.(BluetoothEventConsumer)
	if !ok {
		return nil
	}
	event.DeviceID = deviceID
	event.Event = strings.TrimSpace(event.Event)
	if event.Data == nil {
		event.Data = map[string]string{}
	} else {
		event.Data = copyStringMap(event.Data)
	}
	return consumer.HandleBluetoothEvent(ctx, r.Env, event)
}

// ProcessUSBEvent dispatches a USB passthrough event to the active scenario
// for the source device when that scenario supports the hook.
func (r *Runtime) ProcessUSBEvent(ctx context.Context, event USBEvent) error {
	if r == nil || r.Engine == nil || r.Env == nil {
		return nil
	}
	deviceID := strings.TrimSpace(event.DeviceID)
	activeScenario, ok := r.Engine.ActiveScenario(deviceID)
	if !ok {
		return nil
	}
	consumer, ok := activeScenario.(USBEventConsumer)
	if !ok {
		return nil
	}
	event.DeviceID = deviceID
	event.Event = strings.TrimSpace(event.Event)
	if event.Data == nil {
		event.Data = map[string]string{}
	} else {
		event.Data = copyStringMap(event.Data)
	}
	return consumer.HandleUSBEvent(ctx, r.Env, event)
}

// StatusData returns runtime-focused counters for control-plane system queries.
func (r *Runtime) StatusData() map[string]string {
	activeScenarios := 0
	registeredScenarios := 0
	if r != nil && r.Engine != nil {
		activeScenarios = len(r.Engine.ActiveSnapshot())
		registeredScenarios = len(r.Engine.RegistrySnapshot())
	}

	activeRoutes := 0
	pendingTimers := 0
	if r != nil && r.Env != nil && r.Env.IO != nil {
		activeRoutes = r.Env.IO.RouteCount()
	}
	if r != nil && r.Env != nil && r.Env.Scheduler != nil {
		pendingTimers = len(r.Env.Scheduler.Due(math.MaxInt64))
	}

	return map[string]string{
		"active_scenarios":     strconv.Itoa(activeScenarios),
		"active_routes":        strconv.Itoa(activeRoutes),
		"registered_scenarios": strconv.Itoa(registeredScenarios),
		"pending_timers":       strconv.Itoa(pendingTimers),
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ProcessDueTimers emits notifications for due timer keys and removes them.
// It returns the number of processed keys.
func (r *Runtime) ProcessDueTimers(ctx context.Context, now time.Time) (int, error) {
	if r == nil || r.Env == nil || r.Env.Scheduler == nil {
		return 0, nil
	}

	due := r.Env.Scheduler.Due(now.UnixMilli())
	processed := 0
	for _, key := range due {
		if strings.HasPrefix(key, "timer:") {
			targetDevice := ""
			parts := strings.Split(key, ":")
			if len(parts) >= 2 {
				targetDevice = parts[1]
			}
			if r.Env.Broadcast != nil {
				deviceIDs := []string{}
				if targetDevice != "" {
					deviceIDs = []string{targetDevice}
				}
				if err := r.Env.Broadcast.Notify(ctx, deviceIDs, "Timer complete"); err != nil {
					return processed, err
				}
			}
		}
		if err := r.Env.Scheduler.Remove(ctx, key); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func targetDevices(ctx context.Context, env *Environment, trigger Trigger) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	if explicitMany, ok := trigger.Arguments["device_ids"]; ok && strings.TrimSpace(explicitMany) != "" {
		parts := strings.Split(explicitMany, ",")
		out := make([]string, 0, len(parts))
		seen := map[string]struct{}{}
		for _, part := range parts {
			deviceID := strings.TrimSpace(part)
			if deviceID == "" {
				continue
			}
			if _, exists := seen[deviceID]; exists {
				continue
			}
			seen[deviceID] = struct{}{}
			out = append(out, deviceID)
		}
		if len(out) > 0 {
			return out
		}
	}
	if explicit, ok := trigger.Arguments["device_id"]; ok && explicit != "" {
		return []string{explicit}
	}
	if env.Placement != nil {
		zone := strings.TrimSpace(trigger.Arguments["zone"])
		role := strings.TrimSpace(trigger.Arguments["role"])
		scopeDeviceID := strings.TrimSpace(trigger.Arguments["scope_device_id"])
		if scopeDeviceID == "" {
			scopeDeviceID = strings.TrimSpace(trigger.SourceID)
		}
		nearestCap := strings.TrimSpace(trigger.Arguments["nearest_capability"])
		if nearestCap != "" || strings.EqualFold(strings.TrimSpace(trigger.Arguments["nearest"]), "true") {
			ref, err := env.Placement.NearestWith(ctx, DeviceRef{DeviceID: scopeDeviceID}, nearestCap)
			if err == nil && strings.TrimSpace(ref.DeviceID) != "" {
				return []string{ref.DeviceID}
			}
		}
		if zone != "" || role != "" || scopeDeviceID != "" {
			query := PlacementQuery{
				Scope: TargetScope{
					Zone:      zone,
					Role:      role,
					DeviceID:  strings.TrimSpace(trigger.Arguments["placement_device_id"]),
					Source:    DeviceRef{DeviceID: scopeDeviceID},
					Broadcast: true,
				},
			}
			refs, err := env.Placement.Find(ctx, query)
			if err == nil && len(refs) > 0 {
				out := make([]string, 0, len(refs))
				for _, ref := range refs {
					if id := strings.TrimSpace(ref.DeviceID); id != "" {
						out = append(out, id)
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
	}
	return env.Devices.ListDeviceIDs()
}

func normalizeDeviceIDs(deviceIDs []string) []string {
	out := make([]string, 0, len(deviceIDs))
	seen := make(map[string]struct{}, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		out = append(out, deviceID)
	}
	return out
}
