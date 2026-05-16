package scenario

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/storage"
)

// MorningRoutineMonitorScenario monitors a child's morning routine via camera
// activity. If no activity is seen before the configured alert time, it notifies
// the parent device. It also broadcasts a school-bus warning to the child's room
// at a configurable warning time.
//
// Trigger arguments:
//   - alert_time_ms: Unix ms when to check for absence and notify parent (required).
//   - warning_time_ms: Unix ms when to warn child (optional).
//   - alert_device_id: device ID to send parent alert (defaults to source device).
//   - warning_device_id: device ID of child's room terminal.
//   - alert_message: notification text (default "Morning routine: no activity detected").
//   - warning_message: warning text (default "The bus comes in 10 minutes").
type MorningRoutineMonitorScenario struct {
	trigger Trigger

	mu         sync.Mutex
	alertKey   string
	warningKey string
}

// Name returns the stable scenario identifier.
func (s *MorningRoutineMonitorScenario) Name() string { return "morning_routine_monitor" }

// Match records trigger metadata when morning routine monitoring is requested.
func (s *MorningRoutineMonitorScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "morning routine monitor", "morning_routine_monitor", "morning routine") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules the alert and optional warning jobs, then confirms activation.
func (s *MorningRoutineMonitorScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}

	alertTimeMS, err := strconv.ParseInt(strings.TrimSpace(s.trigger.Arguments["alert_time_ms"]), 10, 64)
	if err != nil || alertTimeMS <= 0 {
		return notifySource(ctx, env, s.trigger.SourceID, "morning_routine_monitor: alert_time_ms is required")
	}

	alertDeviceID := strings.TrimSpace(s.trigger.Arguments["alert_device_id"])
	if alertDeviceID == "" {
		alertDeviceID = strings.TrimSpace(s.trigger.SourceID)
	}
	alertMessage := strings.TrimSpace(s.trigger.Arguments["alert_message"])
	if alertMessage == "" {
		alertMessage = "Morning routine: no activity detected"
	}

	alertKey := fmt.Sprintf("morning_routine.alert:%s:%d", strings.TrimSpace(s.trigger.SourceID), alertTimeMS)
	s.mu.Lock()
	s.alertKey = alertKey
	s.mu.Unlock()

	if err := scheduleRecord(ctx, env, storage.ScheduleRecord{
		Key:      alertKey,
		Kind:     "morning_routine.alert",
		DeviceID: strings.TrimSpace(s.trigger.SourceID),
		UnixMS:   alertTimeMS,
		Payload: map[string]string{
			"device_id": alertDeviceID,
			"message":   alertMessage,
		},
	}); err != nil {
		return err
	}

	warningTimeMS, _ := strconv.ParseInt(strings.TrimSpace(s.trigger.Arguments["warning_time_ms"]), 10, 64)
	if warningTimeMS > 0 {
		warningDeviceID := strings.TrimSpace(s.trigger.Arguments["warning_device_id"])
		warningMessage := strings.TrimSpace(s.trigger.Arguments["warning_message"])
		if warningMessage == "" {
			warningMessage = "The bus comes in 10 minutes"
		}

		warningKey := fmt.Sprintf("morning_routine.warning:%s:%d", strings.TrimSpace(s.trigger.SourceID), warningTimeMS)
		s.mu.Lock()
		s.warningKey = warningKey
		s.mu.Unlock()

		if err := scheduleRecord(ctx, env, storage.ScheduleRecord{
			Key:      warningKey,
			Kind:     "morning_routine.warning",
			DeviceID: strings.TrimSpace(s.trigger.SourceID),
			UnixMS:   warningTimeMS,
			Payload: map[string]string{
				"device_id": warningDeviceID,
				"message":   warningMessage,
			},
		}); err != nil {
			return err
		}
	}

	return notifySource(ctx, env, s.trigger.SourceID, "Morning routine monitor active")
}

// HandleSensor tracks camera activity. If activity is detected before the alert
// fires, the alert is cancelled (child is already up).
func (s *MorningRoutineMonitorScenario) HandleSensor(ctx context.Context, env *Environment, reading SensorReading) error {
	if env == nil || env.Scheduler == nil {
		return nil
	}
	activity, ok := reading.Values["camera_activity"]
	if !ok || activity <= 0 {
		return nil
	}

	s.mu.Lock()
	key := s.alertKey
	s.alertKey = ""
	s.mu.Unlock()

	if key != "" {
		return env.Scheduler.Remove(ctx, key)
	}
	return nil
}

// Stop ends the morning routine monitor and has no persistent side effects.
func (s *MorningRoutineMonitorScenario) Stop() error { return nil }

// scheduleRecord writes a structured record to the scheduler when the
// StructuredScheduler extension is available, falling back to Schedule otherwise.
func scheduleRecord(ctx context.Context, env *Environment, record storage.ScheduleRecord) error {
	if env == nil || env.Scheduler == nil {
		return nil
	}
	if structured, ok := env.Scheduler.(StructuredScheduler); ok {
		return structured.ScheduleRecord(ctx, record)
	}
	return env.Scheduler.Schedule(ctx, record.Key, record.UnixMS)
}

// morningRoutineAlertMessage extracts the notification message and device ID
// from a morning_routine.alert schedule record payload.
func morningRoutineAlertMessage(record storage.ScheduleRecord) (deviceID, message string) {
	if record.Payload == nil {
		return "", ""
	}
	return strings.TrimSpace(record.Payload["device_id"]), strings.TrimSpace(record.Payload["message"])
}

// morningRoutineWarningMessage extracts the target device ID and message
// from a morning_routine.warning schedule record payload.
func morningRoutineWarningMessage(record storage.ScheduleRecord) (deviceID, message string) {
	if record.Payload == nil {
		return "", ""
	}
	return strings.TrimSpace(record.Payload["device_id"]), strings.TrimSpace(record.Payload["message"])
}

// isMorningRoutineKind returns true for both morning_routine.alert and
// morning_routine.warning schedule record kinds.
func isMorningRoutineKind(kind, key string) bool {
	return kind == "morning_routine.alert" || kind == "morning_routine.warning" ||
		strings.HasPrefix(key, "morning_routine.alert:") || strings.HasPrefix(key, "morning_routine.warning:")
}

// morningRoutineKind infers the specific sub-kind from record kind or key prefix.
func morningRoutineKind(kind, key string) string {
	if kind == "morning_routine.alert" || strings.HasPrefix(key, "morning_routine.alert:") {
		return "morning_routine.alert"
	}
	return "morning_routine.warning"
}

// processMorningRoutineRecord fires the alert or warning action for a due
// morning_routine schedule record and returns true if it was handled.
func processMorningRoutineRecord(ctx context.Context, env *Environment, record storage.ScheduleRecord) (bool, error) {
	if !isMorningRoutineKind(record.Kind, record.Key) {
		return false, nil
	}
	if env == nil || env.Broadcast == nil {
		return true, nil
	}
	kind := morningRoutineKind(record.Kind, record.Key)
	var deviceID, message string
	if kind == "morning_routine.alert" {
		deviceID, message = morningRoutineAlertMessage(record)
	} else {
		deviceID, message = morningRoutineWarningMessage(record)
	}
	if message == "" {
		return true, nil
	}
	var targets []string
	if deviceID != "" {
		targets = []string{deviceID}
	}
	return true, env.Broadcast.Notify(ctx, targets, message)
}
