package scenario

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScheduleMonitorScenario schedules a check and confirms activation.
type ScheduleMonitorScenario struct {
	trigger Trigger

	mu              sync.Mutex
	lastAlertUnixMS int64
}

// Name returns the stable scenario identifier.
func (s *ScheduleMonitorScenario) Name() string { return "schedule_monitor" }

// Match records trigger metadata when schedule monitoring is requested.
func (s *ScheduleMonitorScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "schedule monitor", "schedule_monitor", "watch schedule") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules a follow-up check and notifies the source device.
func (s *ScheduleMonitorScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	checkUnixMS := time.Now().Add(5 * time.Minute).UnixMilli()
	if raw := strings.TrimSpace(s.trigger.Arguments["check_unix_ms"]); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			checkUnixMS = parsed
		}
	}
	if env.Scheduler != nil && s.trigger.SourceID != "" {
		if err := env.Scheduler.Schedule(ctx, "schedule_monitor:"+s.trigger.SourceID, checkUnixMS); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.lastAlertUnixMS = 0
	s.mu.Unlock()
	return notifySource(ctx, env, s.trigger.SourceID, "Schedule monitor active")
}

// Stop ends schedule monitor mode and currently has no side effects.
func (s *ScheduleMonitorScenario) Stop() error { return nil }

// HandleSensor consumes live sensor telemetry while schedule monitoring is
// active and raises an activity alert when movement exceeds threshold.
func (s *ScheduleMonitorScenario) HandleSensor(ctx context.Context, env *Environment, reading SensorReading) error {
	if env == nil || env.Broadcast == nil {
		return nil
	}
	if strings.TrimSpace(reading.DeviceID) == "" {
		return nil
	}
	monitorDeviceID := strings.TrimSpace(s.trigger.SourceID)
	if monitorDeviceID != "" && reading.DeviceID != monitorDeviceID {
		return nil
	}

	magnitude, ok := sensorMotionMagnitude(reading.Values)
	if !ok {
		return nil
	}
	threshold := parseFloatOrDefault(s.trigger.Arguments["motion_threshold"], 1.20)
	if magnitude < threshold {
		return nil
	}

	eventUnixMS := reading.UnixMS
	if eventUnixMS <= 0 {
		eventUnixMS = time.Now().UnixMilli()
	}
	cooldownMS := int64(parseFloatOrDefault(s.trigger.Arguments["cooldown_ms"], 60_000))
	if cooldownMS < 0 {
		cooldownMS = 0
	}

	s.mu.Lock()
	if s.lastAlertUnixMS > 0 && eventUnixMS-s.lastAlertUnixMS < cooldownMS {
		s.mu.Unlock()
		return nil
	}
	s.lastAlertUnixMS = eventUnixMS
	s.mu.Unlock()

	return notifySource(ctx, env, reading.DeviceID, fmt.Sprintf("Schedule monitor activity detected: magnitude=%.2f", magnitude))
}

func parseFloatOrDefault(raw string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func sensorMotionMagnitude(values map[string]float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	if scalar, ok := values["motion.magnitude"]; ok {
		return math.Abs(scalar), true
	}
	x, hasX := values["accelerometer.x"]
	y, hasY := values["accelerometer.y"]
	z, hasZ := values["accelerometer.z"]
	if hasX || hasY || hasZ {
		return math.Sqrt((x * x) + (y * y) + (z * z)), true
	}
	return 0, false
}

// RecentIMUAnomalyScenario answers "did you feel that?" from recent observations.
type RecentIMUAnomalyScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *RecentIMUAnomalyScenario) Name() string { return "recent_imu_anomaly" }

// Match records trigger metadata when retrospective IMU checks are requested.
func (s *RecentIMUAnomalyScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "recent_imu_anomaly", "did you feel that") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start summarizes recent IMU anomaly observations from the observation store.
func (s *RecentIMUAnomalyScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Observe == nil {
		return notifySource(ctx, env, s.trigger.SourceID, "No recent IMU anomaly data is available.")
	}
	zone := strings.TrimSpace(s.trigger.Arguments["zone"])
	observations := env.Observe.Recent(ctx, "imu_anomaly", zone, time.Now().Add(-30*time.Second))
	if len(observations) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "No unusual motion was recorded in the last 30 seconds.")
	}
	first := observations[0]
	zoneText := strings.TrimSpace(first.Zone)
	if zoneText == "" {
		zoneText = "the monitored area"
	}
	return notifySource(ctx, env, s.trigger.SourceID,
		fmt.Sprintf("Yes. Recent motion anomaly in %s at %.0f%% confidence.", zoneText, first.Confidence*100))
}

// Stop ends this one-shot scenario.
func (s *RecentIMUAnomalyScenario) Stop() error { return nil }

// SoundIdentificationScenario answers "what was that sound?" from recent observations.
type SoundIdentificationScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *SoundIdentificationScenario) Name() string { return "sound_identification" }

// Match records trigger metadata when retrospective sound labeling is requested.
func (s *SoundIdentificationScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "sound_identification", "what was that sound") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start returns the highest-confidence recent sound label.
func (s *SoundIdentificationScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Observe == nil {
		return notifySource(ctx, env, s.trigger.SourceID, "No recent sound data is available.")
	}
	observations := env.Observe.Recent(ctx, "sound", strings.TrimSpace(s.trigger.Arguments["zone"]), time.Now().Add(-20*time.Second))
	if len(observations) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "I did not find a recent sound event.")
	}
	best := observations[0]
	label := strings.TrimSpace(best.Subject)
	if label == "" {
		label = strings.TrimSpace(best.Attributes["label"])
	}
	if label == "" {
		label = "unknown sound"
	}
	return notifySource(ctx, env, s.trigger.SourceID,
		fmt.Sprintf("Most likely: %s (%.0f%% confidence).", label, best.Confidence*100))
}

// Stop ends this one-shot scenario.
func (s *SoundIdentificationScenario) Stop() error { return nil }

// SoundLocalizationScenario answers "where did that sound come from?".
type SoundLocalizationScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *SoundLocalizationScenario) Name() string { return "sound_localization" }

// Match records trigger metadata when retrospective sound localization is requested.
func (s *SoundLocalizationScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "sound_localization", "where did that sound come from") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start returns the best available recent location estimate for a sound.
func (s *SoundLocalizationScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Observe == nil {
		return notifySource(ctx, env, s.trigger.SourceID, "No recent localization data is available.")
	}
	observations := env.Observe.Recent(ctx, "sound", "", time.Now().Add(-20*time.Second))
	if len(observations) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "I do not have a recent localized sound.")
	}
	best := observations[0]
	zone := strings.TrimSpace(best.Zone)
	if zone == "" && best.Location != nil {
		zone = strings.TrimSpace(best.Location.Zone)
	}
	if zone == "" {
		zone = "an unknown zone"
	}
	return notifySource(ctx, env, s.trigger.SourceID,
		fmt.Sprintf("Likely source: %s (%.0f%% confidence).", zone, best.Confidence*100))
}

// Stop ends this one-shot scenario.
func (s *SoundLocalizationScenario) Stop() error { return nil }

// PresenceQueryScenario reports current person presence from world model.
type PresenceQueryScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *PresenceQueryScenario) Name() string { return "presence_query" }

// Match records trigger metadata when presence is requested.
func (s *PresenceQueryScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "presence_query", "who is in the house", "who is home") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start summarizes known person presence.
func (s *PresenceQueryScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.World == nil {
		return notifySource(ctx, env, s.trigger.SourceID, "Presence data is not available yet.")
	}
	people, err := env.World.WhoIsHome(ctx)
	if err != nil || len(people) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "I do not currently have confirmed occupants.")
	}
	parts := make([]string, 0, len(people))
	for _, person := range people {
		name := strings.TrimSpace(person.DisplayName)
		if name == "" {
			name = person.EntityID
		}
		zone := ""
		if person.LastKnown != nil {
			zone = strings.TrimSpace(person.LastKnown.Zone)
		}
		if zone == "" {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, name+" in "+zone)
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Currently detected: "+strings.Join(parts, ", ")+".")
}

// Stop ends this one-shot scenario.
func (s *PresenceQueryScenario) Stop() error { return nil }

// BluetoothInventoryScenario summarizes recent Bluetooth sightings.
type BluetoothInventoryScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *BluetoothInventoryScenario) Name() string { return "bluetooth_inventory" }

// Match records trigger metadata when Bluetooth inventory is requested.
func (s *BluetoothInventoryScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "bluetooth_inventory", "bluetooth inventory") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start summarizes recent Bluetooth observations.
func (s *BluetoothInventoryScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Observe == nil {
		return notifySource(ctx, env, s.trigger.SourceID, "Bluetooth inventory is unavailable.")
	}
	observations := env.Observe.Recent(ctx, "bluetooth", "", time.Now().Add(-5*time.Minute))
	if len(observations) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "No Bluetooth devices were recently observed.")
	}
	names := make([]string, 0, len(observations))
	seen := map[string]struct{}{}
	for _, ob := range observations {
		label := strings.TrimSpace(ob.Subject)
		if label == "" {
			label = strings.TrimSpace(ob.Attributes["device"])
		}
		if label == "" {
			label = strings.TrimSpace(ob.Attributes["mac"])
		}
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		names = append(names, label)
		if len(names) >= 5 {
			break
		}
	}
	if len(names) == 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "Bluetooth scans are active, but no identifiable devices were found.")
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Recent Bluetooth devices: "+strings.Join(names, ", ")+".")
}

// Stop ends this one-shot scenario.
func (s *BluetoothInventoryScenario) Stop() error { return nil }
