// Package scenario contains server-side scenario matching and runtime flows.
package scenario

import (
	"context"
	"strconv"
	"time"
)

// AlertScenario broadcasts a critical alert across targeted devices.
type AlertScenario struct{}

// Name returns the stable scenario identifier.
func (AlertScenario) Name() string { return "red_alert" }

// Match checks whether the trigger intent activates this scenario.
func (AlertScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "red alert"
}

// Start broadcasts an alert notification.
func (AlertScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "RED ALERT")
}

// Stop ends the scenario and currently has no side effects.
func (AlertScenario) Stop() error { return nil }

// PhotoFrameScenario marks a low-priority ambient mode.
type PhotoFrameScenario struct{}

// Name returns the stable scenario identifier.
func (PhotoFrameScenario) Name() string { return "photo_frame" }

// Match checks whether the trigger intent activates this scenario.
func (PhotoFrameScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "photo frame"
}

// Start broadcasts a mode activation notification.
func (PhotoFrameScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "Photo frame active")
}

// Stop ends the scenario and currently has no side effects.
func (PhotoFrameScenario) Stop() error { return nil }

// TimerReminderScenario schedules a timer and confirms it via broadcast.
type TimerReminderScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *TimerReminderScenario) Name() string { return "timer_reminder" }

// Match records trigger arguments when this scenario should run.
func (s *TimerReminderScenario) Match(trigger Trigger) bool {
	if trigger.Intent != "set timer" {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules the timer and confirms to the origin device.
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

// Stop ends the scenario and currently has no side effects.
func (s *TimerReminderScenario) Stop() error { return nil }
