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

	switch {
	case normalized == "red alert":
		trigger.Intent = "red alert"
	case normalized == "stand down" || normalized == "stop red alert":
		trigger.Intent = "red alert"
	case normalized == "photo frame":
		trigger.Intent = "photo frame"
	case normalized == "terminal" || normalized == "open terminal":
		trigger.Intent = "terminal"
	case normalized == "bluetooth scan" || normalized == "scan bluetooth" || normalized == "scan ble":
		trigger.Intent = "bluetooth_passthrough"
		trigger.Arguments["action"] = "scan"
	case strings.HasPrefix(normalized, "bluetooth connect ") || strings.HasPrefix(normalized, "connect bluetooth ") || strings.HasPrefix(normalized, "connect ble "):
		trigger.Intent = "bluetooth_passthrough"
		trigger.Arguments["action"] = "connect"
		if targetID, ok := parseBluetoothConnectTarget(normalized); ok {
			trigger.Arguments["target_id"] = targetID
		}
	case normalized == "usb enumerate" || normalized == "scan usb":
		trigger.Intent = "usb_passthrough"
		trigger.Arguments["action"] = "enumerate"
	case strings.HasPrefix(normalized, "usb claim "):
		trigger.Intent = "usb_passthrough"
		trigger.Arguments["action"] = "claim"
		if vendorID, productID, ok := parseUSBClaimVIDPID(normalized); ok {
			trigger.Arguments["vendor_id"] = vendorID
			trigger.Arguments["product_id"] = productID
		}
	case normalized == "intercom" || normalized == "start intercom":
		trigger.Intent = "intercom"
	case normalized == "announcement" || normalized == "announce" || normalized == "start announcement":
		trigger.Intent = "announcement"
	case normalized == "pa system" || normalized == "pa mode" || normalized == "end pa" || normalized == "stop pa":
		trigger.Intent = "pa system"
	case strings.HasPrefix(normalized, "multi window focus ") ||
		strings.HasPrefix(normalized, "show all cameras focus ") ||
		strings.HasPrefix(normalized, "all cameras focus "):
		trigger.Intent = "multi window"
		if focusDeviceID, ok := parseMultiWindowFocus(normalized); ok {
			trigger.Arguments["audio_focus_device_id"] = focusDeviceID
		}
	case normalized == "multi window" || normalized == "show all cameras" || normalized == "all cameras":
		trigger.Intent = "multi window"
	case strings.HasPrefix(normalized, "tell me when ") ||
		strings.HasPrefix(normalized, "notify me when "):
		if target, ok := parseAudioMonitorTarget(normalized); ok {
			trigger.Intent = "audio monitor"
			trigger.Arguments["target"] = target
		}
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
	case normalized == "video call" || normalized == "start video call" ||
		strings.HasPrefix(normalized, "video call ") || strings.HasPrefix(normalized, "start video call "):
		trigger.Intent = "internal video call"
		if targetDeviceID, ok := parseInternalVideoCallTarget(normalized); ok {
			trigger.Arguments["target_device_id"] = targetDeviceID
		}
	case strings.HasPrefix(normalized, "set a timer for "):
		trigger.Intent = "set timer"
		parseTimerMinutes(trigger.Arguments, normalized, now)
	default:
		// Keep full normalized text as fallback intent.
	}

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
	args["fire_unix_ms"] = strconv.FormatInt(now.Add(time.Duration(minutes)*time.Minute).UnixMilli(), 10)
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
