package transport

import "strings"

func commandTargetDeviceIDsFromArgs(args map[string]string) []string {
	if len(args) == 0 {
		return nil
	}
	if rawList := strings.TrimSpace(args["device_ids"]); rawList != "" {
		parts := strings.Split(rawList, ",")
		out := make([]string, 0, len(parts))
		seen := map[string]struct{}{}
		for _, part := range parts {
			deviceID := strings.TrimSpace(part)
			if deviceID == "" {
				continue
			}
			if _, exists := seen[deviceID]; exists {
				continue
			}
			seen[deviceID] = struct{}{}
			out = append(out, deviceID)
		}
		if len(out) > 0 {
			return out
		}
	}
	if one := strings.TrimSpace(args["device_id"]); one != "" {
		return []string{one}
	}
	return nil
}

func (h *StreamHandler) commandTargetDeviceIDsFallback(cmd *CommandRequest) []string {
	if h.runtime != nil && h.runtime.Env != nil && h.runtime.Env.Devices != nil {
		all := h.runtime.Env.Devices.ListDeviceIDs()
		if len(all) > 0 {
			return all
		}
	}
	if source := strings.TrimSpace(cmd.DeviceID); source != "" {
		return []string{source}
	}
	return nil
}
