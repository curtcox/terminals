package capability

import (
	"strings"
	"time"
)

// CreateSession creates a new interactive session of the given kind and target.
func (s *Service) CreateSession(kind, target string) InteractiveSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	kind = defaultIfBlank(kind, "generic")
	target = defaultIfBlank(target, "default")
	now := s.now()
	session := InteractiveSession{
		ID:        s.nextIDLocked("sess"),
		Kind:      kind,
		Target:    target,
		CreatedAt: now,
		UpdatedAt: now,
	}
	session.Audit = append(session.Audit, SessionAuditEvent{Action: "session.create", Meta: session.Kind + ":" + session.Target, CreatedAt: now})
	s.sessions = append(s.sessions, session)
	s.appendRecentLocked("session", session.ID+" "+session.Kind+" "+session.Target)
	return session
}

// ListSessions returns all active interactive sessions.
func (s *Service) ListSessions() []InteractiveSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSessions(s.sessions)
}

// GetSession returns the session with the given ID, or false if not found.
func (s *Service) GetSession(sessionID string) (InteractiveSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID = strings.TrimSpace(sessionID)
	for _, session := range s.sessions {
		if session.ID == sessionID {
			return cloneSession(session), true
		}
	}
	return InteractiveSession{}, false
}

// ListSessionParticipants returns participants for the given session ID.
func (s *Service) ListSessionParticipants(sessionID string) ([]SessionParticipant, bool) {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return nil, false
	}
	participants := append([]SessionParticipant(nil), session.Participants...)
	return participants, true
}

// JoinSession adds a participant to the session, returning the updated session.
func (s *Service) JoinSession(sessionID, participant string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	participant = strings.TrimSpace(participant)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if participant == "" {
			return cloneSession(s.sessions[i]), true
		}
		for _, existing := range s.sessions[i].Participants {
			if strings.EqualFold(existing.IdentityID, participant) {
				return cloneSession(s.sessions[i]), true
			}
		}
		s.sessions[i].Participants = append(s.sessions[i].Participants, SessionParticipant{
			IdentityID: participant,
			JoinedAt:   s.now(),
		})
		now := s.now()
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.join",
			Actor:     participant,
			Target:    s.sessions[i].ID,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" join "+participant)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// LeaveSession removes a participant from the session, returning the updated session.
func (s *Service) LeaveSession(sessionID, participant string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	participant = strings.TrimSpace(participant)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if participant != "" {
			next := s.sessions[i].Participants[:0]
			for _, existing := range s.sessions[i].Participants {
				if strings.EqualFold(existing.IdentityID, participant) {
					continue
				}
				next = append(next, existing)
			}
			s.sessions[i].Participants = next
			now := s.now()
			s.sessions[i].UpdatedAt = now
			s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
				Action:    "session.leave",
				Actor:     participant,
				Target:    s.sessions[i].ID,
				CreatedAt: now,
			})
			s.appendRecentLocked("session", s.sessions[i].ID+" leave "+participant)
		}
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// AttachDevice attaches a device reference to an existing session.
func (s *Service) AttachDevice(sessionID, deviceRef string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	deviceRef = strings.TrimSpace(deviceRef)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if deviceRef == "" {
			return cloneSession(s.sessions[i]), true
		}
		for _, existing := range s.sessions[i].AttachedDevices {
			if strings.EqualFold(existing, deviceRef) {
				return cloneSession(s.sessions[i]), true
			}
		}
		now := s.now()
		s.sessions[i].AttachedDevices = append(s.sessions[i].AttachedDevices, deviceRef)
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.attach_device",
			Target:    deviceRef,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" attach "+deviceRef)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// DetachDevice detaches a device reference from an existing session.
func (s *Service) DetachDevice(sessionID, deviceRef string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	deviceRef = strings.TrimSpace(deviceRef)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if deviceRef == "" {
			return cloneSession(s.sessions[i]), true
		}
		next := s.sessions[i].AttachedDevices[:0]
		for _, existing := range s.sessions[i].AttachedDevices {
			if strings.EqualFold(existing, deviceRef) {
				continue
			}
			next = append(next, existing)
		}
		now := s.now()
		s.sessions[i].AttachedDevices = next
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.detach_device",
			Target:    deviceRef,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" detach "+deviceRef)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// RequestControl records a participant's request for control in a session.
func (s *Service) RequestControl(sessionID, participant, controlType string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	participant = strings.TrimSpace(participant)
	controlType = normalizeControlType(controlType)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if participant == "" {
			return cloneSession(s.sessions[i]), true
		}
		for _, existing := range s.sessions[i].ControlRequests {
			if strings.EqualFold(existing.ParticipantID, participant) {
				return cloneSession(s.sessions[i]), true
			}
		}
		now := s.now()
		s.sessions[i].ControlRequests = append(s.sessions[i].ControlRequests, SessionControlRequest{
			ParticipantID: participant,
			ControlType:   controlType,
			RequestedAt:   now,
		})
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.control.request",
			Actor:     participant,
			Target:    s.sessions[i].ID,
			Meta:      controlType,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" control.request "+participant)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// GrantControl approves control for one participant and records an audit entry.
func (s *Service) GrantControl(sessionID, participant, grantedBy, controlType string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	participant = strings.TrimSpace(participant)
	grantedBy = defaultIfBlank(grantedBy, "system")
	controlType = normalizeControlType(controlType)
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if participant == "" {
			return cloneSession(s.sessions[i]), true
		}
		now := s.now()
		s.sessions[i].ControlRequests = controlRequestsWithoutParticipant(s.sessions[i].ControlRequests, participant)
		s.sessions[i].ControlGrants = upsertControlGrant(s.sessions[i].ControlGrants, participant, grantedBy, controlType, now)
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.control.grant",
			Actor:     grantedBy,
			Target:    participant,
			Meta:      controlType,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" control.grant "+participant)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

func controlRequestsWithoutParticipant(requests []SessionControlRequest, participant string) []SessionControlRequest {
	next := requests[:0]
	for _, req := range requests {
		if !strings.EqualFold(req.ParticipantID, participant) {
			next = append(next, req)
		}
	}
	return next
}

func upsertControlGrant(grants []SessionControlGrant, participant, grantedBy, controlType string, now time.Time) []SessionControlGrant {
	for i := range grants {
		if strings.EqualFold(grants[i].ParticipantID, participant) {
			grants[i].GrantedBy = grantedBy
			grants[i].ControlType = controlType
			grants[i].GrantedAt = now
			return grants
		}
	}
	return append(grants, SessionControlGrant{
		ParticipantID: participant,
		GrantedBy:     grantedBy,
		ControlType:   controlType,
		GrantedAt:     now,
	})
}

// RevokeControl removes control grants for one participant and records an audit entry.
func (s *Service) RevokeControl(sessionID, participant, revokedBy string) (InteractiveSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID = strings.TrimSpace(sessionID)
	participant = strings.TrimSpace(participant)
	revokedBy = defaultIfBlank(revokedBy, "system")
	for i := range s.sessions {
		if s.sessions[i].ID != sessionID {
			continue
		}
		if participant == "" {
			return cloneSession(s.sessions[i]), true
		}
		next := s.sessions[i].ControlGrants[:0]
		for _, grant := range s.sessions[i].ControlGrants {
			if strings.EqualFold(grant.ParticipantID, participant) {
				continue
			}
			next = append(next, grant)
		}
		now := s.now()
		s.sessions[i].ControlGrants = next
		s.sessions[i].UpdatedAt = now
		s.sessions[i].Audit = append(s.sessions[i].Audit, SessionAuditEvent{
			Action:    "session.control.revoke",
			Actor:     revokedBy,
			Target:    participant,
			CreatedAt: now,
		})
		s.appendRecentLocked("session", s.sessions[i].ID+" control.revoke "+participant)
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

func cloneSessions(input []InteractiveSession) []InteractiveSession {
	out := make([]InteractiveSession, 0, len(input))
	for _, item := range input {
		out = append(out, cloneSession(item))
	}
	return out
}

func cloneSession(item InteractiveSession) InteractiveSession {
	item.Participants = append([]SessionParticipant(nil), item.Participants...)
	item.AttachedDevices = append([]string(nil), item.AttachedDevices...)
	item.ControlRequests = append([]SessionControlRequest(nil), item.ControlRequests...)
	item.ControlGrants = append([]SessionControlGrant(nil), item.ControlGrants...)
	item.Audit = append([]SessionAuditEvent(nil), item.Audit...)
	return item
}

func normalizeControlType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "interactive"
	}
	return value
}
