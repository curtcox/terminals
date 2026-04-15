package scenario

import (
	"context"
	"encoding/json"
	"strings"
)

type llmIntentEnvelope struct {
	Action string            `json:"action"`
	Object string            `json:"object"`
	Slots  map[string]string `json:"slots"`
	Scope  struct {
		DeviceID  string `json:"device_id"`
		Zone      string `json:"zone"`
		Role      string `json:"role"`
		Nearest   bool   `json:"nearest"`
		Broadcast bool   `json:"broadcast"`
	} `json:"scope"`
}

func shouldResolveWithLLM(spoken string, parsed Trigger) bool {
	normalized := strings.TrimSpace(strings.ToLower(spoken))
	if normalized == "" || parsed.Intent == "" {
		return false
	}
	// When ParseVoiceTrigger could not map to a known intent, it keeps the
	// normalized text as the fallback action string. This is our ambiguity gate.
	return parsed.Intent == normalized
}

func resolveVoiceIntentWithLLM(ctx context.Context, llm LLM, spoken string) (*IntentRecord, bool) {
	if llm == nil {
		return nil, false
	}
	resp, err := llm.Query(ctx, []LLMMessage{
		{
			Role: "system",
			Content: "Return ONLY JSON with keys: action, object, slots, scope." +
				" Use empty strings/objects when unknown.",
		},
		{
			Role:    "user",
			Content: spoken,
		},
	}, LLMOptions{})
	if err != nil || resp == nil {
		return nil, false
	}
	raw := strings.TrimSpace(resp.Text)
	if raw == "" {
		return nil, false
	}
	var decoded llmIntentEnvelope
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, false
	}
	action := strings.TrimSpace(strings.ToLower(decoded.Action))
	if action == "" {
		return nil, false
	}
	slots := decoded.Slots
	if slots == nil {
		slots = map[string]string{}
	}
	if scopeDeviceID := strings.TrimSpace(decoded.Scope.DeviceID); scopeDeviceID != "" {
		slots["placement_device_id"] = scopeDeviceID
	}
	if zone := strings.TrimSpace(decoded.Scope.Zone); zone != "" {
		slots["zone"] = zone
	}
	if role := strings.TrimSpace(decoded.Scope.Role); role != "" {
		slots["role"] = role
	}
	if decoded.Scope.Nearest {
		slots["nearest"] = "true"
	}
	if decoded.Scope.Broadcast {
		slots["broadcast"] = "true"
	}
	return &IntentRecord{
		Action:  action,
		Object:  strings.TrimSpace(decoded.Object),
		Slots:   slots,
		RawText: strings.TrimSpace(spoken),
		Source:  SourceVoice,
	}, true
}
