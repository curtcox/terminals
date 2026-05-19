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
		IntentV2: &IntentRecord{
			Action:  normalized,
			Slots:   map[string]string{},
			RawText: strings.TrimSpace(spoken),
			Source:  SourceVoice,
		},
	}

	applyVoiceIntentMatchers(&trigger, normalized, now)

	trigger.IntentV2.Action = trigger.Intent
	trigger.IntentV2.Slots = copyStringMap(trigger.Arguments)

	return trigger
}

func parseMultiWindowFocus(normalized string) (string, bool) {
	for _, prefix := range []string{
		"multi window focus ",
		"show all cameras focus ",
		"all cameras focus ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			focusDeviceID := strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
			if focusDeviceID == "" {
				return "", false
			}
			return focusDeviceID, true
		}
	}
	return "", false
}

// parseAudioMonitorTarget extracts the monitored subject from a
// "tell me when X …" or "notify me when X …" phrase. It strips one
// leading article ("the ", "a ", "an ", "my ") and one trailing
// action-verb phrase ("stops", "beeps", "is done", etc.) so the caller
// receives just the subject (e.g. "dishwasher", "dryer", "laundry").
// Returns false when no subject remains after trimming.
func parseAudioMonitorTarget(normalized string) (string, bool) {
	rest := normalized
	for _, prefix := range []string{"tell me when ", "notify me when "} {
		if strings.HasPrefix(rest, prefix) {
			rest = strings.TrimPrefix(rest, prefix)
			break
		}
	}
	for _, article := range []string{"the ", "a ", "an ", "my "} {
		if strings.HasPrefix(rest, article) {
			rest = strings.TrimPrefix(rest, article)
			break
		}
	}
	for _, suffix := range []string{
		" stops",
		" stop",
		" beeps",
		" beep",
		" is done",
		" is finished",
		" finishes",
		" finish",
		" goes off",
	} {
		if strings.HasSuffix(rest, suffix) {
			rest = strings.TrimSuffix(rest, suffix)
			break
		}
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	for _, article := range []string{"the", "a", "an", "my"} {
		if rest == article {
			return "", false
		}
	}
	return rest, true
}

func parseInternalVideoCallTarget(normalized string) (string, bool) {
	for _, prefix := range []string{
		"video call ",
		"start video call ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			targetDeviceID := strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
			if targetDeviceID == "" {
				return "", false
			}
			return targetDeviceID, true
		}
	}
	return "", false
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
	args["duration_seconds"] = strconv.Itoa(minutes * 60)
	args["fire_unix_ms"] = strconv.FormatInt(now.Add(time.Duration(minutes)*time.Minute).UnixMilli(), 10)
	labelParts := []string{}
	if len(parts) > 2 {
		labelParts = parts[2:]
	}
	if label := parseTimerLabel(labelParts); label != "" {
		args["label"] = label
	}
}

func parseTimerLabel(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "called", "named":
		return strings.TrimSpace(strings.Join(parts[1:], " "))
	case "for":
		return strings.TrimSpace(strings.Join(parts[1:], " "))
	default:
		return strings.TrimSpace(strings.Join(parts, " "))
	}
}

func parseBluetoothConnectTarget(normalized string) (string, bool) {
	for _, prefix := range []string{
		"bluetooth connect ",
		"connect bluetooth ",
		"connect ble ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			targetID := strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
			if targetID == "" {
				return "", false
			}
			return targetID, true
		}
	}
	return "", false
}

func parseUSBClaimVIDPID(normalized string) (string, string, bool) {
	const prefix = "usb claim "
	if !strings.HasPrefix(normalized, prefix) {
		return "", "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return "", "", false
	}
	vendorID := strings.TrimSpace(parts[0])
	productID := strings.TrimSpace(parts[1])
	if vendorID == "" || productID == "" {
		return "", "", false
	}
	return vendorID, productID, true
}

func parseTerminalVerification(normalized string) (string, string, bool) {
	const prefix = "verify terminal "
	if !strings.HasPrefix(normalized, prefix) {
		return "", "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
	if rest == "" {
		return "", "", false
	}
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return "", "", false
	}
	deviceID := strings.TrimSpace(parts[0])
	method := "manual"
	if len(parts) > 1 {
		method = strings.TrimSpace(parts[1])
	}
	if deviceID == "" {
		return "", "", false
	}
	return deviceID, method, true
}
