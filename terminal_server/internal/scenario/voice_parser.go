package scenario

import (
	"strconv"
	"strings"
	"time"
)

// ParseVoiceTrigger converts recognized speech text into a trigger.
func ParseVoiceTrigger(sourceID, spoken string, now time.Time) Trigger {
	normalized := strings.TrimSpace(strings.ToLower(spoken))
	trigger := Trigger{
		Kind:      TriggerVoice,
		SourceID:  sourceID,
		Intent:    normalized,
		Arguments: map[string]string{},
	}

	switch {
	case normalized == "red alert":
		trigger.Intent = "red alert"
	case normalized == "photo frame":
		trigger.Intent = "photo frame"
	case strings.HasPrefix(normalized, "set a timer for "):
		trigger.Intent = "set timer"
		parseTimerMinutes(trigger.Arguments, normalized, now)
	default:
		// Keep full normalized text as fallback intent.
	}

	return trigger
}

func parseTimerMinutes(args map[string]string, normalized string, now time.Time) {
	const prefix = "set a timer for "
	rest := strings.TrimPrefix(normalized, prefix)
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return
	}
	minutes, err := strconv.Atoi(parts[0])
	if err != nil || minutes <= 0 {
		return
	}
	args["minutes"] = strconv.Itoa(minutes)
	args["fire_unix_ms"] = strconv.FormatInt(now.Add(time.Duration(minutes)*time.Minute).UnixMilli(), 10)
}
