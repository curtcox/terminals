package scenario

import (
	"strings"
	"time"
)

func matchVoiceAlertIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "red alert":
		trigger.Intent = "red alert"
		return true
	case "stand down", "stop red alert":
		trigger.Intent = "red alert"
		return true
	default:
		return false
	}
}

func matchVoiceDisplayIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "photo frame":
		trigger.Intent = "photo frame"
		return true
	case "terminal", "open terminal":
		trigger.Intent = "terminal"
		return true
	default:
		return false
	}
}

func matchVoicePassthroughIntent(trigger *Trigger, normalized string) bool {
	switch {
	case normalized == "bluetooth scan" || normalized == "scan bluetooth" || normalized == "scan ble":
		trigger.Intent = "bluetooth_passthrough"
		trigger.Arguments["action"] = "scan"
		return true
	case strings.HasPrefix(normalized, "bluetooth connect ") || strings.HasPrefix(normalized, "connect bluetooth ") || strings.HasPrefix(normalized, "connect ble "):
		trigger.Intent = "bluetooth_passthrough"
		trigger.Arguments["action"] = "connect"
		if targetID, ok := parseBluetoothConnectTarget(normalized); ok {
			trigger.Arguments["target_id"] = targetID
		}
		return true
	case normalized == "usb enumerate" || normalized == "scan usb":
		trigger.Intent = "usb_passthrough"
		trigger.Arguments["action"] = "enumerate"
		return true
	case strings.HasPrefix(normalized, "usb claim "):
		trigger.Intent = "usb_passthrough"
		trigger.Arguments["action"] = "claim"
		if vendorID, productID, ok := parseUSBClaimVIDPID(normalized); ok {
			trigger.Arguments["vendor_id"] = vendorID
			trigger.Arguments["product_id"] = productID
		}
		return true
	default:
		return false
	}
}

func matchVoiceCommunicationIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "intercom", "start intercom":
		trigger.Intent = "intercom"
		return true
	case "announcement", "announce", "start announcement":
		trigger.Intent = "announcement"
		return true
	case "pa system", "pa mode", "end pa", "stop pa":
		trigger.Intent = "pa system"
		return true
	default:
		return false
	}
}

func matchVoiceMultiWindowIntent(trigger *Trigger, normalized string) bool {
	switch {
	case strings.HasPrefix(normalized, "multi window focus ") ||
		strings.HasPrefix(normalized, "show all cameras focus ") ||
		strings.HasPrefix(normalized, "all cameras focus "):
		trigger.Intent = "multi window"
		if focusDeviceID, ok := parseMultiWindowFocus(normalized); ok {
			trigger.Arguments["audio_focus_device_id"] = focusDeviceID
		}
		return true
	case normalized == "multi window", normalized == "show all cameras", normalized == "all cameras":
		trigger.Intent = "multi window"
		return true
	default:
		return false
	}
}

func matchVoiceAudioMonitorIntent(trigger *Trigger, normalized string) bool {
	switch {
	case strings.HasPrefix(normalized, "tell me when "), strings.HasPrefix(normalized, "notify me when "):
		if target, ok := parseAudioMonitorTarget(normalized); ok {
			trigger.Intent = "audio monitor"
			trigger.Arguments["target"] = target
			return true
		}
		return false
	case normalized == "audio monitor":
		trigger.Intent = "audio monitor"
		return true
	default:
		return false
	}
}

func matchVoiceSensingIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "did you feel that":
		trigger.Intent = "recent_imu_anomaly"
		return true
	case "what was that sound":
		trigger.Intent = "sound_identification"
		return true
	case "where did that sound come from":
		trigger.Intent = "sound_localization"
		return true
	case "who is in the house", "who is home", "who is in the house and where":
		trigger.Intent = "presence_query"
		return true
	case "bluetooth inventory", "what bluetooth devices are here":
		trigger.Intent = "bluetooth_inventory"
		return true
	default:
		return false
	}
}

func matchVoiceTerminalVerificationIntent(trigger *Trigger, normalized string) bool {
	if !strings.HasPrefix(normalized, "verify terminal ") {
		return false
	}
	trigger.Intent = "terminal_verification"
	if deviceID, method, ok := parseTerminalVerification(normalized); ok {
		trigger.Arguments["device_id"] = deviceID
		trigger.Arguments["method"] = method
	}
	return true
}

func matchVoiceCameraIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "camera monitor", "monitor camera", "watch camera":
		trigger.Intent = "camera monitor"
		return true
	case "vision analysis", "analyze camera", "detect packages", "watch door":
		trigger.Intent = "vision analysis"
		return true
	case "schedule monitor":
		trigger.Intent = "schedule monitor"
		return true
	default:
		return false
	}
}

func matchVoiceTimerIntent(trigger *Trigger, normalized string, now time.Time) bool {
	switch normalized {
	case "cancel timer", "cancel the timer", "stop timer", "stop the timer":
		trigger.Intent = "cancel timer"
		return true
	default:
		if strings.HasPrefix(normalized, "set a timer for ") {
			trigger.Intent = "set timer"
			parseTimerMinutes(trigger.Arguments, normalized, now)
			return true
		}
		return false
	}
}

func matchVoiceAssistantIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "voice assistant":
		trigger.Intent = "voice assistant"
		return true
	default:
		if strings.HasPrefix(normalized, "assistant ") {
			trigger.Intent = "voice assistant"
			trigger.Arguments["query"] = strings.TrimSpace(strings.TrimPrefix(normalized, "assistant "))
			return true
		}
		return false
	}
}

func matchVoiceCallIntent(trigger *Trigger, normalized string) bool {
	switch normalized {
	case "phone call":
		trigger.Intent = "phone call"
		return true
	default:
		if strings.HasPrefix(normalized, "call ") {
			trigger.Intent = "phone call"
			trigger.Arguments["target"] = strings.TrimSpace(strings.TrimPrefix(normalized, "call "))
			return true
		}
	}
	switch {
	case normalized == "video call", normalized == "start video call",
		strings.HasPrefix(normalized, "video call "), strings.HasPrefix(normalized, "start video call "):
		trigger.Intent = "internal video call"
		if targetDeviceID, ok := parseInternalVideoCallTarget(normalized); ok {
			trigger.Arguments["target_device_id"] = targetDeviceID
		}
		return true
	default:
		return false
	}
}

func applyVoiceIntentMatchers(trigger *Trigger, normalized string, now time.Time) {
	switch {
	case matchVoiceAlertIntent(trigger, normalized):
	case matchVoiceDisplayIntent(trigger, normalized):
	case matchVoicePassthroughIntent(trigger, normalized):
	case matchVoiceCommunicationIntent(trigger, normalized):
	case matchVoiceMultiWindowIntent(trigger, normalized):
	case matchVoiceAudioMonitorIntent(trigger, normalized):
	case matchVoiceSensingIntent(trigger, normalized):
	case matchVoiceTerminalVerificationIntent(trigger, normalized):
	case matchVoiceCameraIntent(trigger, normalized):
	case matchVoiceTimerIntent(trigger, normalized, now):
	case matchVoiceAssistantIntent(trigger, normalized):
	case matchVoiceCallIntent(trigger, normalized):
	default:
		// Keep full normalized text as fallback intent.
	}
}
