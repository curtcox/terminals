package capability

import (
	"sort"
	"strings"
	"time"
)

// SimDeviceUpsert creates or updates one virtual simulation device.
func (s *Service) SimDeviceUpsert(deviceID string, caps []string) SimDevice {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	now := s.now()
	existing, ok := s.simDevices[deviceID]
	createdAt := now
	if ok {
		createdAt = existing.CreatedAt
	}
	device := SimDevice{
		DeviceID:  deviceID,
		Caps:      normalizeSimCaps(caps),
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
	s.simDevices[deviceID] = device
	s.appendRecentLocked("sim", "device upsert "+deviceID)
	return device
}

// SimDeviceGet returns one simulation device by id.
func (s *Service) SimDeviceGet(deviceID string) (SimDevice, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.simDevices[strings.ToLower(strings.TrimSpace(deviceID))]
	if !ok {
		return SimDevice{}, false
	}
	device.Caps = append([]string(nil), device.Caps...)
	return device, true
}

// SimDeviceList returns all simulation devices sorted by id.
func (s *Service) SimDeviceList() []SimDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SimDevice, 0, len(s.simDevices))
	for _, device := range s.simDevices {
		copyDevice := device
		copyDevice.Caps = append([]string(nil), device.Caps...)
		out = append(out, copyDevice)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeviceID < out[j].DeviceID })
	return out
}

// SimDeviceDelete removes one simulation device and its buffered inputs.
func (s *Service) SimDeviceDelete(deviceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	if _, ok := s.simDevices[deviceID]; !ok {
		return false
	}
	delete(s.simDevices, deviceID)
	delete(s.simInputs, deviceID)
	s.appendRecentLocked("sim", "device delete "+deviceID)
	return true
}

// SimRecordInput stores one synthetic input event for a simulation device.
func (s *Service) SimRecordInput(deviceID, componentID, action, value string) (SimInputRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	if _, ok := s.simDevices[deviceID]; !ok {
		return SimInputRecord{}, false
	}
	record := SimInputRecord{
		ID:          s.nextIDLocked("simin"),
		DeviceID:    deviceID,
		ComponentID: strings.TrimSpace(componentID),
		Action:      strings.TrimSpace(action),
		Value:       strings.TrimSpace(value),
		CreatedAt:   s.now(),
	}
	s.simInputs[deviceID] = append(s.simInputs[deviceID], record)
	if len(s.simInputs[deviceID]) > 200 {
		s.simInputs[deviceID] = append([]SimInputRecord(nil), s.simInputs[deviceID][len(s.simInputs[deviceID])-200:]...)
	}
	s.appendRecentLocked("sim", "input "+deviceID+" "+record.Action)
	return record, true
}

// SimInputs returns buffered synthetic input events for one simulation device.
func (s *Service) SimInputs(deviceID string) []SimInputRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.simInputs[strings.ToLower(strings.TrimSpace(deviceID))]
	return append([]SimInputRecord(nil), items...)
}

// SimExpect checks one expectation against captured simulation state.
func (s *Service) SimExpect(deviceID, kind, selector string, within time.Duration) (SimExpectationResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	if _, ok := s.simDevices[deviceID]; !ok {
		return SimExpectationResult{}, false
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	selector = strings.TrimSpace(selector)
	checkedAt := s.now()
	result := SimExpectationResult{
		DeviceID:  deviceID,
		Kind:      kind,
		Selector:  selector,
		Matched:   false,
		CheckedAt: checkedAt,
	}
	if within > 0 {
		result.Within = within.String()
	}

	switch kind {
	case "ui":
		snapshot, hasSnapshot := s.uiSnapshots[deviceID]
		if !hasSnapshot {
			result.Reason = "no UI snapshot captured"
			break
		}
		if selector == "" {
			result.Matched = true
			break
		}
		haystack := strings.ToLower(snapshot.Descriptor + "\n" + snapshot.LastPatchDescriptor + "\n" + snapshot.LastTransition)
		result.Matched = strings.Contains(haystack, strings.ToLower(selector))
		if !result.Matched {
			result.Reason = "selector not found in captured UI payload"
		}
	case "message":
		if len(s.bus) == 0 {
			result.Reason = "no bus messages captured"
			break
		}
		if selector == "" {
			result.Matched = true
			break
		}
		needle := strings.ToLower(selector)
		for i := len(s.bus) - 1; i >= 0; i-- {
			event := s.bus[i]
			haystack := strings.ToLower(event.Kind + "\n" + event.Name + "\n" + event.Payload)
			if strings.Contains(haystack, needle) {
				result.Matched = true
				break
			}
		}
		if !result.Matched {
			result.Reason = "selector not found in captured bus messages"
		}
	default:
		result.Reason = "unsupported expectation kind"
	}

	status := "failed"
	if result.Matched {
		status = "matched"
	}
	s.appendRecentLocked("sim", "expect "+deviceID+" "+kind+" "+status)
	return result, true
}

// SimRecord returns simulation captures for one device and optional lookback duration.
func (s *Service) SimRecord(deviceID string, duration time.Duration) (SimRecordResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	if _, ok := s.simDevices[deviceID]; !ok {
		return SimRecordResult{}, false
	}

	endedAt := s.now()
	startedAt := endedAt
	if duration > 0 {
		startedAt = endedAt.Add(-duration)
	}

	snapshot, hasSnapshot := s.uiSnapshots[deviceID]
	if !hasSnapshot {
		snapshot = UISnapshot{DeviceID: deviceID}
	}

	inputs := append([]SimInputRecord(nil), s.simInputs[deviceID]...)
	messages := make([]BusEvent, 0, len(s.bus))
	for _, event := range s.bus {
		if duration > 0 && event.CreatedAt.Before(startedAt) {
			continue
		}
		messages = append(messages, event)
	}

	result := SimRecordResult{
		DeviceID:  deviceID,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		Snapshot:  snapshot,
		Inputs:    inputs,
		Messages:  messages,
	}
	if duration > 0 {
		result.Duration = duration.String()
	}

	s.appendRecentLocked("sim", "record "+deviceID)
	return result, true
}

func normalizeSimCaps(caps []string) []string {
	out := make([]string, 0, len(caps))
	seen := map[string]struct{}{}
	for _, raw := range caps {
		for _, part := range strings.Split(raw, ",") {
			capValue := strings.ToLower(strings.TrimSpace(part))
			if capValue == "" {
				continue
			}
			if _, exists := seen[capValue]; exists {
				continue
			}
			seen[capValue] = struct{}{}
			out = append(out, capValue)
		}
	}
	sort.Strings(out)
	return out
}
