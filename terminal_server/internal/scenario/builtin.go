// Package scenario implements the scenario engine and built-in scenario types.
package scenario

import (
	"context"
	"strconv"
	"time"
)

// AlertScenario broadcasts a critical alert across targeted devices.
type AlertScenario struct{}

// Name returns the scenario identifier.
func (AlertScenario) Name() string { return "red_alert" }

// Match returns true when the trigger intent is "red alert".
func (AlertScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "red alert"
}

// Start broadcasts a red alert notification to all devices.
func (AlertScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "RED ALERT")
}

// Stop is a no-op for AlertScenario.
func (AlertScenario) Stop() error { return nil }

// PhotoFrameScenario marks a low-priority ambient mode.
type PhotoFrameScenario struct{}

// Name returns the scenario identifier.
func (PhotoFrameScenario) Name() string { return "photo_frame" }

// Match returns true when the trigger intent is "photo frame".
func (PhotoFrameScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "photo frame"
}

// Start broadcasts a photo frame activation notification to all devices.
func (PhotoFrameScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "Photo frame active")
}

// Stop is a no-op for PhotoFrameScenario.
func (PhotoFrameScenario) Stop() error { return nil }

// TimerReminderScenario schedules a timer and confirms it via broadcast.
type TimerReminderScenario struct {
	trigger Trigger
}

// Name returns the scenario identifier.
func (s *TimerReminderScenario) Name() string { return "timer_reminder" }

// Match returns true when the trigger intent is "set timer", storing the trigger for use in Start.
func (s *TimerReminderScenario) Match(trigger Trigger) bool {
	if trigger.Intent != "set timer" {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules a timer and notifies the source device via broadcast.
func (s *TimerReminderScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}

	fireUnixMS := time.Now().Add(10 * time.Minute).UnixMilli()
	if raw := s.trigger.Arguments["fire_unix_ms"]; raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			fireUnixMS = parsed
		}
	}

	timerKey := "timer:" + s.trigger.SourceID + ":" + strconv.FormatInt(fireUnixMS, 10)
	if env.Scheduler != nil {
		if err := env.Scheduler.Schedule(ctx, timerKey, fireUnixMS); err != nil {
			return err
		}
	}
	if env.Broadcast != nil {
		deviceIDs := []string{}
		if s.trigger.SourceID != "" {
			deviceIDs = []string{s.trigger.SourceID}
		}
		return env.Broadcast.Notify(ctx, deviceIDs, "Timer set")
	}
	return nil
}

// Stop is a no-op for TimerReminderScenario.
func (s *TimerReminderScenario) Stop() error { return nil }
