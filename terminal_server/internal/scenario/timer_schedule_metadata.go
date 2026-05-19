package scenario

import (
	"strconv"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/storage"
)

func timerMetadataFromScheduleRecord(record storage.ScheduleRecord) timerScheduleMetadata {
	if record.Kind == "timer.tick" || strings.HasPrefix(record.Key, "timer_tick:") {
		return timerTickMetadataFromRecord(record)
	}
	if record.Kind != "timer" && strings.HasPrefix(record.Key, "timer:") {
		return parseTimerScheduleKey(record.Key)
	}
	return timerScheduleMetadataFromRecord(record)
}

func timerTickMetadataFromRecord(record storage.ScheduleRecord) timerScheduleMetadata {
	legacy := timerScheduleMetadata{}
	if strings.HasPrefix(record.Key, "timer_tick:") {
		legacy = parseTimerTickScheduleKey(record.Key)
	}
	meta := timerScheduleMetadata{
		DeviceID:       strings.TrimSpace(record.DeviceID),
		TargetDeviceID: strings.TrimSpace(record.Payload["target_device_id"]),
		FireUnixMS:     legacy.FireUnixMS,
		Label:          strings.TrimSpace(record.Subject),
	}
	if meta.Label == "" {
		meta.Label = "timer"
	}
	if raw := strings.TrimSpace(record.Payload["expiry_unix_ms"]); raw != "" {
		meta.FireUnixMS, _ = strconv.ParseInt(raw, 10, 64)
	}
	if raw := strings.TrimSpace(record.Payload["duration_seconds"]); raw != "" {
		meta.DurationSeconds, _ = strconv.Atoi(raw)
	}
	mergeLegacyTimerMetadata(&meta, legacy)
	return meta
}

func timerScheduleMetadataFromRecord(record storage.ScheduleRecord) timerScheduleMetadata {
	legacy := timerScheduleMetadata{}
	if strings.HasPrefix(record.Key, "timer:") {
		legacy = parseTimerScheduleKey(record.Key)
	}
	meta := timerScheduleMetadata{
		DeviceID:       strings.TrimSpace(record.DeviceID),
		TargetDeviceID: strings.TrimSpace(record.Payload["target_device_id"]),
		FireUnixMS:     record.UnixMS,
		Label:          strings.TrimSpace(record.Subject),
	}
	if meta.Label == "" {
		meta.Label = "timer"
	}
	if raw := strings.TrimSpace(record.Payload["duration_seconds"]); raw != "" {
		meta.DurationSeconds, _ = strconv.Atoi(raw)
	}
	mergeLegacyTimerMetadata(&meta, legacy)
	return meta
}

func mergeLegacyTimerMetadata(meta *timerScheduleMetadata, legacy timerScheduleMetadata) {
	if meta.DeviceID == "" {
		meta.DeviceID = legacy.DeviceID
	}
	if meta.DurationSeconds == 0 {
		meta.DurationSeconds = legacy.DurationSeconds
	}
	if meta.Label == "timer" && legacy.Label != "" {
		meta.Label = legacy.Label
	}
}
