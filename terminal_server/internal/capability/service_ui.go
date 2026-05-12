package capability

import (
	"sort"
	"strings"
)

// UIViewUpsert creates or updates one authored UI view record.
func (s *Service) UIViewUpsert(viewID, rootID, descriptor string) UIView {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewID = strings.ToLower(strings.TrimSpace(viewID))
	view := UIView{
		ViewID:     viewID,
		RootID:     strings.TrimSpace(rootID),
		Descriptor: strings.TrimSpace(descriptor),
		UpdatedAt:  s.now(),
	}
	s.uiViews[viewID] = view
	s.appendRecentLocked("ui", viewID+" upsert")
	return view
}

// UIViewGet returns one authored UI view record by id.
func (s *Service) UIViewGet(viewID string) (UIView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	view, ok := s.uiViews[strings.ToLower(strings.TrimSpace(viewID))]
	if !ok {
		return UIView{}, false
	}
	return view, true
}

// UIViewList returns all authored UI view records sorted by id.
func (s *Service) UIViewList() []UIView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	views := make([]UIView, 0, len(s.uiViews))
	for _, view := range s.uiViews {
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ViewID < views[j].ViewID })
	return views
}

// UIViewDelete removes one authored UI view record by id.
func (s *Service) UIViewDelete(viewID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewID = strings.ToLower(strings.TrimSpace(viewID))
	if _, ok := s.uiViews[viewID]; !ok {
		return false
	}
	delete(s.uiViews, viewID)
	s.appendRecentLocked("ui", viewID+" deleted")
	return true
}

// UIPush applies a full authored descriptor to one device snapshot.
func (s *Service) UIPush(deviceID, descriptor, rootID string) UISnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "unknown"
	}
	now := s.now()
	snapshot := s.uiSnapshots[deviceID]
	snapshot.DeviceID = deviceID
	snapshot.RootID = strings.TrimSpace(rootID)
	snapshot.Descriptor = strings.TrimSpace(descriptor)
	snapshot.UpdatedAt = now
	snapshot.Subscriptions = append([]string(nil), s.uiSubs[deviceID]...)
	s.uiSnapshots[deviceID] = snapshot
	s.appendRecentLocked("ui", "push "+deviceID)
	return snapshot
}

// UIPatch applies a patch descriptor to one device snapshot.
func (s *Service) UIPatch(deviceID, componentID, descriptor string) UISnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "unknown"
	}
	now := s.now()
	snapshot := s.uiSnapshots[deviceID]
	snapshot.DeviceID = deviceID
	snapshot.LastPatchComponentID = strings.TrimSpace(componentID)
	snapshot.LastPatchDescriptor = strings.TrimSpace(descriptor)
	snapshot.UpdatedAt = now
	snapshot.Subscriptions = append([]string(nil), s.uiSubs[deviceID]...)
	s.uiSnapshots[deviceID] = snapshot
	s.appendRecentLocked("ui", "patch "+deviceID)
	return snapshot
}

// UITransition applies a transition hint to one device snapshot.
func (s *Service) UITransition(deviceID, componentID, transition string, durationMS int) UISnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "unknown"
	}
	now := s.now()
	snapshot := s.uiSnapshots[deviceID]
	snapshot.DeviceID = deviceID
	snapshot.LastTransitionComponentID = strings.TrimSpace(componentID)
	snapshot.LastTransition = strings.TrimSpace(transition)
	snapshot.LastTransitionDurationMS = durationMS
	snapshot.UpdatedAt = now
	snapshot.Subscriptions = append([]string(nil), s.uiSubs[deviceID]...)
	s.uiSnapshots[deviceID] = snapshot
	s.appendRecentLocked("ui", "transition "+deviceID)
	return snapshot
}

// UIBroadcast fans out an authored descriptor or patch to the given device ids.
func (s *Service) UIBroadcast(cohort, descriptor, patchID string, deviceIDs []string) UIBroadcast {
	s.mu.Lock()
	defer s.mu.Unlock()

	cohort = strings.ToLower(strings.TrimSpace(cohort))
	devices := normalizeDeviceIDs(deviceIDs)
	descriptor = strings.TrimSpace(descriptor)
	patchID = strings.TrimSpace(patchID)
	now := s.now()
	for _, deviceID := range devices {
		snapshot := s.uiSnapshots[deviceID]
		snapshot.DeviceID = deviceID
		if patchID == "" {
			snapshot.Descriptor = descriptor
		} else {
			snapshot.LastPatchComponentID = patchID
			snapshot.LastPatchDescriptor = descriptor
		}
		snapshot.UpdatedAt = now
		snapshot.Subscriptions = append([]string(nil), s.uiSubs[deviceID]...)
		s.uiSnapshots[deviceID] = snapshot
	}
	broadcast := UIBroadcast{
		Cohort:     cohort,
		Descriptor: descriptor,
		PatchID:    patchID,
		Devices:    devices,
		UpdatedAt:  now,
	}
	s.appendRecentLocked("ui", "broadcast "+cohort)
	return broadcast
}

// UISubscribe records a device subscription target and returns the updated snapshot.
func (s *Service) UISubscribe(deviceID, to string) UISnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = "unknown"
	}
	to = strings.TrimSpace(to)
	if to != "" {
		existing := append([]string(nil), s.uiSubs[deviceID]...)
		if !sliceContainsFold(existing, to) {
			existing = append(existing, to)
		}
		sort.Slice(existing, func(i, j int) bool { return existing[i] < existing[j] })
		s.uiSubs[deviceID] = existing
	}
	now := s.now()
	snapshot := s.uiSnapshots[deviceID]
	snapshot.DeviceID = deviceID
	snapshot.Subscriptions = append([]string(nil), s.uiSubs[deviceID]...)
	snapshot.UpdatedAt = now
	s.uiSnapshots[deviceID] = snapshot
	s.appendRecentLocked("ui", "subscribe "+deviceID)
	return snapshot
}

// UISnapshot returns one device UI snapshot if any authored state exists.
func (s *Service) UISnapshot(deviceID string) (UISnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deviceID = strings.TrimSpace(deviceID)
	snapshot, ok := s.uiSnapshots[deviceID]
	subs := s.uiSubs[deviceID]
	if !ok && len(subs) == 0 {
		return UISnapshot{}, false
	}
	if !ok {
		snapshot = UISnapshot{DeviceID: deviceID}
	}
	snapshot.Subscriptions = append([]string(nil), subs...)
	return snapshot, true
}

func normalizeDeviceIDs(deviceIDs []string) []string {
	if len(deviceIDs) == 0 {
		return nil
	}
	out := make([]string, 0, len(deviceIDs))
	seen := make(map[string]struct{}, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		normalized := strings.TrimSpace(deviceID)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}
