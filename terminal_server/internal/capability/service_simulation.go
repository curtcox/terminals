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

	result.Matched, result.Reason = s.matchSimExpectationLocked(deviceID, kind, selector)

	status := "failed"
	if result.Matched {
		status = "matched"
	}
	s.appendRecentLocked("sim", "expect "+deviceID+" "+kind+" "+status)
	return result, true
}

func (s *Service) matchSimExpectationLocked(deviceID, kind, selector string) (bool, string) {
	switch kind {
	case "ui":
		return s.matchUISimExpectationLocked(deviceID, selector)
	case "message":
		return s.matchMessageSimExpectationLocked(selector)
	default:
		return false, "unsupported expectation kind"
	}
}

func (s *Service) matchUISimExpectationLocked(deviceID, selector string) (bool, string) {
	snapshot, hasSnapshot := s.uiSnapshots[deviceID]
	if !hasSnapshot {
		return false, "no UI snapshot captured"
	}
	if selector == "" {
		return true, ""
	}
	haystack := strings.ToLower(snapshot.Descriptor + "\n" + snapshot.LastPatchDescriptor + "\n" + snapshot.LastTransition)
	if strings.Contains(haystack, strings.ToLower(selector)) {
		return true, ""
	}
	return false, "selector not found in captured UI payload"
}

func (s *Service) matchMessageSimExpectationLocked(selector string) (bool, string) {
	if len(s.bus) == 0 {
		return false, "no bus messages captured"
	}
	if selector == "" {
		return true, ""
	}
	needle := strings.ToLower(selector)
	for i := len(s.bus) - 1; i >= 0; i-- {
		event := s.bus[i]
		haystack := strings.ToLower(event.Kind + "\n" + event.Name + "\n" + event.Payload)
		if strings.Contains(haystack, needle) {
			return true, ""
		}
	}
	return false, "selector not found in captured bus messages"
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
