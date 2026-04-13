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
	case normalized == "stand down" || normalized == "stop red alert":
		trigger.Intent = "red alert"
	case normalized == "photo frame":
		trigger.Intent = "photo frame"
	case normalized == "terminal" || normalized == "open terminal":
		trigger.Intent = "terminal"
	case normalized == "intercom" || normalized == "start intercom":
		trigger.Intent = "intercom"
	case normalized == "pa system" || normalized == "pa mode" || normalized == "end pa" || normalized == "stop pa":
		trigger.Intent = "pa system"
	case normalized == "multi window" || normalized == "show all cameras" || normalized == "all cameras":
		trigger.Intent = "multi window"
	case normalized == "audio monitor":
		trigger.Intent = "audio monitor"
	case normalized == "schedule monitor":
		trigger.Intent = "schedule monitor"
	case normalized == "voice assistant" || strings.HasPrefix(normalized, "assistant "):
		trigger.Intent = "voice assistant"
		if strings.HasPrefix(normalized, "assistant ") {
			trigger.Arguments["query"] = strings.TrimSpace(strings.TrimPrefix(normalized, "assistant "))
		}
	case normalized == "phone call" || strings.HasPrefix(normalized, "call "):
		trigger.Intent = "phone call"
		if strings.HasPrefix(normalized, "call ") {
			trigger.Arguments["target"] = strings.TrimSpace(strings.TrimPrefix(normalized, "call "))
		}
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
