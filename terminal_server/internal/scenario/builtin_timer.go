package scenario

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// TimerReminderScenario schedules a timer and confirms it via broadcast.
type TimerReminderScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *TimerReminderScenario) Name() string { return "timer_reminder" }

// Match records trigger arguments when this scenario should run.
func (s *TimerReminderScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "set timer", "timer_reminder", "cancel timer") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules the timer and confirms to the origin device.
func (s *TimerReminderScenario) Start(ctx context.Context, env *Environment) error {
	result, err := s.StartResult(ctx, env)
	if err != nil {
		return err
	}
	return ExecuteOperations(ctx, env, result.Ops, time.Now().UTC())
}

// StartResult returns scheduler and notification operations for a timer.
func (s *TimerReminderScenario) StartResult(ctx context.Context, env *Environment) (ScenarioResult, error) {
	_ = ctx
	if env == nil {
		return ScenarioResult{}, nil
	}
	if intentMatches(s.trigger.Intent, "cancel timer") {
		return s.cancelResult(ctx, env), nil
	}

	now := time.Now()
	durationSeconds := timerDurationSeconds(s.trigger.Arguments)
	fireUnixMS := now.Add(time.Duration(durationSeconds) * time.Second).UnixMilli()
	if raw := s.trigger.Arguments["fire_unix_ms"]; raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			fireUnixMS = parsed
		}
	}
	if durationSeconds <= 0 && fireUnixMS > 0 {
		delta := time.Until(time.UnixMilli(fireUnixMS))
		durationSeconds = int(delta.Round(time.Second).Seconds())
		if durationSeconds < 0 {
			durationSeconds = 0
		}
	}

	label := timerLabel(s.trigger.Arguments)
	targetID := timerTargetDevice(ctx, env, s.trigger.SourceID)
	timerKey := timerScheduleKey(s.trigger.SourceID, fireUnixMS, durationSeconds, label)
	ops := []Operation{
		{
			Kind:   OperationSchedulerAfter,
			Target: timerKey,
			Args: map[string]string{
				"unix_ms":          strconv.FormatInt(fireUnixMS, 10),
				"kind":             "timer",
				"device_id":        s.trigger.SourceID,
				"target_device_id": targetID,
				"subject":          label,
				"duration_seconds": strconv.Itoa(durationSeconds),
			},
		},
	}
	if env.UI != nil && targetID != "" {
		view := timerCountdownView(label, durationSeconds)
		ops = append([]Operation{{
			Kind:   OperationUISet,
			Target: targetID,
			Node:   &view,
		}}, ops...)
		if durationSeconds > 1 {
			tickUnixMS := now.Add(time.Second).UnixMilli()
			ops = append(ops, Operation{
				Kind:   OperationSchedulerAfter,
				Target: timerTickScheduleKey(s.trigger.SourceID, fireUnixMS, durationSeconds, label, tickUnixMS),
				Args: map[string]string{
					"unix_ms":          strconv.FormatInt(tickUnixMS, 10),
					"kind":             "timer.tick",
					"device_id":        s.trigger.SourceID,
					"target_device_id": targetID,
					"subject":          label,
					"duration_seconds": strconv.Itoa(durationSeconds),
					"expiry_unix_ms":   strconv.FormatInt(fireUnixMS, 10),
				},
			})
		}
	}
	if env.Broadcast != nil {
		ops = append(ops, Operation{
			Kind:   OperationBroadcastNotify,
			Target: s.trigger.SourceID,
			Args: map[string]string{
				"message": timerSetMessage(label, durationSeconds),
			},
		})
	}
	return ScenarioResult{Ops: ops}, nil
}

// Stop ends the scenario and currently has no side effects.
func (s *TimerReminderScenario) Stop() error { return nil }

func (s *TimerReminderScenario) cancelResult(ctx context.Context, env *Environment) ScenarioResult {
	_ = ctx
	ops := []Operation{}
	targetID := strings.TrimSpace(s.trigger.SourceID)
	if env != nil && env.Scheduler != nil {
		for _, record := range timerRecordsForDevice(env.Scheduler, s.trigger.SourceID) {
			ops = append(ops, Operation{
				Kind:   OperationSchedulerCancel,
				Target: record.Key,
			})
			if target := strings.TrimSpace(record.Payload["target_device_id"]); target != "" {
				targetID = target
			}
		}
	}
	if env != nil && env.UI != nil && targetID != "" {
		ops = append(ops, Operation{
			Kind:   OperationUIClear,
			Target: targetID,
			Args:   map[string]string{"root": "timer"},
		})
	}
	if env != nil && env.Broadcast != nil {
		ops = append(ops, Operation{
			Kind:   OperationBroadcastNotify,
			Target: strings.TrimSpace(s.trigger.SourceID),
			Args: map[string]string{
				"message": "Timer cancelled",
			},
		})
	}
	return ScenarioResult{Ops: ops, Done: true}
}

func timerRecordsForDevice(scheduler Scheduler, sourceID string) []storage.ScheduleRecord {
	if scheduler == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if structured, ok := scheduler.(interface {
		DueRecords(int64) []storage.ScheduleRecord
	}); ok {
		return filterTimerRecords(structured.DueRecords(math.MaxInt64), sourceID)
	}
	keys := scheduler.Due(math.MaxInt64)
	out := make([]storage.ScheduleRecord, 0, len(keys))
	for _, key := range keys {
		record := storage.ScheduleRecord{Key: key, Kind: scheduleKindFromKey(key)}
		if timerRecordMatchesDevice(record, sourceID) {
			out = append(out, record)
		}
	}
	return out
}

func filterTimerRecords(records []storage.ScheduleRecord, sourceID string) []storage.ScheduleRecord {
	out := make([]storage.ScheduleRecord, 0, len(records))
	for _, record := range records {
		if timerRecordMatchesDevice(record, sourceID) {
			out = append(out, record)
		}
	}
	return out
}

func timerRecordMatchesDevice(record storage.ScheduleRecord, sourceID string) bool {
	if record.Kind != "timer" && record.Kind != "timer.tick" &&
		!strings.HasPrefix(record.Key, "timer:") && !strings.HasPrefix(record.Key, "timer_tick:") {
		return false
	}
	meta := timerMetadataFromScheduleRecord(record)
	return sourceID == "" || meta.DeviceID == sourceID
}

func timerTargetDevice(ctx context.Context, env *Environment, sourceID string) string {
	sourceID = strings.TrimSpace(sourceID)
	if env == nil || env.Placement == nil {
		return sourceID
	}
	ref, err := env.Placement.NearestWith(ctx, DeviceRef{DeviceID: sourceID}, "screen")
	if err != nil || strings.TrimSpace(ref.DeviceID) == "" {
		return sourceID
	}
	return strings.TrimSpace(ref.DeviceID)
}

func timerDurationSeconds(args map[string]string) int {
	if seconds, ok := parsePositiveInt(args["duration_seconds"]); ok {
		return seconds
	}
	if minutes, ok := parsePositiveInt(args["minutes"]); ok {
		return minutes * 60
	}
	return 10 * 60
}

func parsePositiveInt(raw string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func timerLabel(args map[string]string) string {
	label := strings.TrimSpace(args["label"])
	if label == "" {
		label = "timer"
	}
	return label
}

func timerScheduleKey(sourceID string, fireUnixMS int64, durationSeconds int, label string) string {
	parts := []string{
		"timer",
		strings.TrimSpace(sourceID),
		strconv.FormatInt(fireUnixMS, 10),
	}
	if durationSeconds > 0 || strings.TrimSpace(label) != "" {
		parts = append(parts, strconv.Itoa(durationSeconds), url.PathEscape(strings.TrimSpace(label)))
	}
	return strings.Join(parts, ":")
}

func timerTickScheduleKey(sourceID string, expiryUnixMS int64, durationSeconds int, label string, tickUnixMS int64) string {
	parts := []string{
		"timer_tick",
		strings.TrimSpace(sourceID),
		strconv.FormatInt(expiryUnixMS, 10),
		strconv.Itoa(durationSeconds),
		url.PathEscape(strings.TrimSpace(label)),
		strconv.FormatInt(tickUnixMS, 10),
	}
	return strings.Join(parts, ":")
}

type timerScheduleMetadata struct {
	DeviceID        string
	TargetDeviceID  string
	FireUnixMS      int64
	DurationSeconds int
	Label           string
}

func parseTimerScheduleKey(key string) timerScheduleMetadata {
	parts := strings.Split(key, ":")
	meta := timerScheduleMetadata{Label: "timer"}
	if len(parts) >= 2 {
		meta.DeviceID = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		meta.FireUnixMS, _ = strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	}
	if len(parts) >= 4 {
		meta.DurationSeconds, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
	}
	if len(parts) >= 5 {
		if unescaped, err := url.PathUnescape(strings.TrimSpace(parts[4])); err == nil && strings.TrimSpace(unescaped) != "" {
			meta.Label = strings.TrimSpace(unescaped)
		}
	}
	return meta
}

func parseTimerTickScheduleKey(key string) timerScheduleMetadata {
	parts := strings.Split(key, ":")
	meta := timerScheduleMetadata{Label: "timer"}
	if len(parts) >= 2 {
		meta.DeviceID = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		meta.FireUnixMS, _ = strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	}
	if len(parts) >= 4 {
		meta.DurationSeconds, _ = strconv.Atoi(strings.TrimSpace(parts[3]))
	}
	if len(parts) >= 5 {
		if unescaped, err := url.PathUnescape(strings.TrimSpace(parts[4])); err == nil && strings.TrimSpace(unescaped) != "" {
			meta.Label = strings.TrimSpace(unescaped)
		}
	}
	return meta
}

func timerSetMessage(label string, durationSeconds int) string {
	if strings.TrimSpace(label) == "" || label == "timer" {
		return "Timer set"
	}
	if durationSeconds <= 0 {
		return "Timer set: " + label
	}
	return "Timer set: " + label + " " + mmss(durationSeconds)
}

func timerExpiredSpeech(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "timer"
	}
	return "Your " + label + " is ready."
}

func mmss(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func timerCountdownView(label string, remainingSeconds int) ui.Descriptor {
	return ui.New("stack", map[string]string{
		"id":         "timer",
		"background": "#17201A",
	}, ui.New("text", map[string]string{
		"id":    "timer_label",
		"value": strings.TrimSpace(label),
		"style": "headline",
		"color": "#F2F7EF",
	}), timerRemainingPatch(remainingSeconds), ui.New("text", map[string]string{
		"id":    "banner",
		"value": "",
		"style": "body",
		"color": "#D7E5D1",
	}), ui.GlobalOverlaySlot())
}

func timerRemainingPatch(remainingSeconds int) ui.Descriptor {
	return ui.New("text", map[string]string{
		"id":    "remaining",
		"value": mmss(remainingSeconds),
		"style": "headline",
		"color": "#F9D65C",
	})
}

func timerDoneBannerPatch() ui.Descriptor {
	return ui.New("text", map[string]string{
		"id":    "banner",
		"value": "Timer done!",
		"style": "alert",
		"color": "#FFFFFF",
	})
}
