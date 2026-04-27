// Package capability provides typed in-memory services for REPL capability closure.
package capability

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Identity represents a user or system principal in the capability service.
type Identity struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name,omitempty"`
	Groups      []string          `json:"groups,omitempty"`
	Aliases     []string          `json:"aliases,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// InteractiveSession represents a collaborative session between participants.
type InteractiveSession struct {
	ID              string                  `json:"id"`
	Kind            string                  `json:"kind"`
	Target          string                  `json:"target"`
	Participants    []SessionParticipant    `json:"participants,omitempty"`
	AttachedDevices []string                `json:"attached_devices,omitempty"`
	ControlRequests []SessionControlRequest `json:"control_requests,omitempty"`
	ControlGrants   []SessionControlGrant   `json:"control_grants,omitempty"`
	Audit           []SessionAuditEvent     `json:"audit,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// SessionParticipant records a single identity's membership in a session.
type SessionParticipant struct {
	IdentityID string    `json:"identity_id"`
	JoinedAt   time.Time `json:"joined_at"`
}

// SessionControlRequest records a request from one participant to take control.
type SessionControlRequest struct {
	ParticipantID string    `json:"participant_id"`
	ControlType   string    `json:"control_type"`
	RequestedAt   time.Time `json:"requested_at"`
}

// SessionControlGrant records an approved control grant for one participant.
type SessionControlGrant struct {
	ParticipantID string    `json:"participant_id"`
	GrantedBy     string    `json:"granted_by"`
	ControlType   string    `json:"control_type"`
	GrantedAt     time.Time `json:"granted_at"`
}

// SessionAuditEvent records one control/share lifecycle event.
type SessionAuditEvent struct {
	Action    string    `json:"action"`
	Actor     string    `json:"actor,omitempty"`
	Target    string    `json:"target,omitempty"`
	Meta      string    `json:"meta,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageRoom represents a durable room for conversation history.
type MessageRoom struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Audience      string    `json:"audience,omitempty"`
	RetentionDays int       `json:"retention_days,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Message represents a posted message within a room.
type Message struct {
	ID              string    `json:"id"`
	Room            string    `json:"room"`
	TargetRef       string    `json:"target_ref,omitempty"`
	Text            string    `json:"text"`
	ThreadRootRef   string    `json:"thread_root_ref,omitempty"`
	ThreadParentRef string    `json:"thread_parent_ref,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// BoardItem represents a pinned item on a named board.
type BoardItem struct {
	ID        string    `json:"id"`
	Board     string    `json:"board"`
	Pinned    bool      `json:"pinned,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Artifact represents a stored artifact such as a document or media object.
type Artifact struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ArtifactVersion records one durable version entry for an artifact.
type ArtifactVersion struct {
	ArtifactID string    `json:"artifact_id"`
	Version    int       `json:"version"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Action     string    `json:"action"`
	CreatedAt  time.Time `json:"created_at"`
}

// ArtifactTemplate records a reusable artifact template keyed by name.
type ArtifactTemplate struct {
	Name             string    `json:"name"`
	SourceArtifactID string    `json:"source_artifact_id"`
	SourceKind       string    `json:"source_kind"`
	SourceTitle      string    `json:"source_title"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Annotation represents a user annotation attached to a canvas.
type Annotation struct {
	ID        string    `json:"id"`
	Canvas    string    `json:"canvas"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult represents a single item returned by a search query.
type SearchResult struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// MemoryEntry represents a stored memory item scoped to a named context.
type MemoryEntry struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Acknowledgement records that an identity has acknowledged a subject reference.
type Acknowledgement struct {
	IdentityID     string    `json:"identity_id"`
	SubjectRef     string    `json:"subject_ref"`
	ActorRef       string    `json:"actor_ref,omitempty"`
	Mode           string    `json:"mode,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

// RecentItem represents a recent activity entry in the capability service.
type RecentItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type searchableItem struct {
	ID        string
	Kind      string
	Text      string
	CreatedAt time.Time
}

// StoreRecord represents a key/value entry in a named namespace store.
type StoreRecord struct {
	Namespace string     `json:"namespace"`
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	Binding   string     `json:"binding,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// StoreNamespaceSummary represents aggregate inventory for one store namespace.
type StoreNamespaceSummary struct {
	Name        string `json:"name"`
	RecordCount int    `json:"record_count"`
}

// DeviceCohort represents a reusable named selector set for device targeting.
type DeviceCohort struct {
	Name      string    `json:"name"`
	Selectors []string  `json:"selectors,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UIView represents one authored UI view record.
type UIView struct {
	ViewID     string    `json:"view_id"`
	RootID     string    `json:"root_id,omitempty"`
	Descriptor string    `json:"descriptor,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UISnapshot represents current server-authored UI state for one device.
type UISnapshot struct {
	DeviceID                  string    `json:"device_id"`
	RootID                    string    `json:"root_id,omitempty"`
	Descriptor                string    `json:"descriptor,omitempty"`
	LastPatchComponentID      string    `json:"last_patch_component_id,omitempty"`
	LastPatchDescriptor       string    `json:"last_patch_descriptor,omitempty"`
	LastTransitionComponentID string    `json:"last_transition_component_id,omitempty"`
	LastTransition            string    `json:"last_transition,omitempty"`
	LastTransitionDurationMS  int       `json:"last_transition_duration_ms,omitempty"`
	Subscriptions             []string  `json:"subscriptions,omitempty"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// UIBroadcast represents one fan-out UI operation targeting a named cohort.
type UIBroadcast struct {
	Cohort     string    `json:"cohort"`
	Descriptor string    `json:"descriptor,omitempty"`
	PatchID    string    `json:"patch_id,omitempty"`
	Devices    []string  `json:"devices,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BusEvent represents a named event emitted on the internal event bus.
type BusEvent struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// HandlerRegistration represents one runtime input/event routing rule.
type HandlerRegistration struct {
	ID          string    `json:"id"`
	Selector    string    `json:"selector"`
	Action      string    `json:"action"`
	RunCommand  string    `json:"run_command,omitempty"`
	EmitKind    string    `json:"emit_kind,omitempty"`
	EmitName    string    `json:"emit_name,omitempty"`
	EmitPayload string    `json:"emit_payload,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Service provides typed in-memory storage for capability closure tools.
type Service struct {
	mu sync.RWMutex

	now func() time.Time
	seq uint64

	identities   []Identity
	sessions     []InteractiveSession
	messageRooms []MessageRoom
	messages     []Message
	boardItems   []BoardItem
	artifacts    []Artifact
	versions     map[string][]ArtifactVersion
	templates    map[string]ArtifactTemplate
	annotations  []Annotation
	memories     []MemoryEntry
	recent       []RecentItem
	store        map[string]StoreRecord
	bus          []BusEvent
	handlers     map[string]HandlerRegistration
	cohorts      map[string]DeviceCohort
	uiViews      map[string]UIView
	uiSnapshots  map[string]UISnapshot
	uiSubs       map[string][]string
	acks         map[string]Acknowledgement
}

// NewService creates a new Service with default seed data.
func NewService() *Service {
	now := time.Now
	s := &Service{
		now:         func() time.Time { return now().UTC() },
		store:       map[string]StoreRecord{},
		handlers:    map[string]HandlerRegistration{},
		cohorts:     map[string]DeviceCohort{},
		uiViews:     map[string]UIView{},
		uiSnapshots: map[string]UISnapshot{},
		uiSubs:      map[string][]string{},
		acks:        map[string]Acknowledgement{},
		versions:    map[string][]ArtifactVersion{},
		templates:   map[string]ArtifactTemplate{},
		identities: []Identity{
			{
				ID:          "system",
				DisplayName: "System",
				Groups:      []string{"family", "operators"},
				Aliases:     []string{"admin", "house"},
				Preferences: map[string]string{"notifications": "normal", "default_zone": "house"},
				CreatedAt:   now().UTC(),
			},
		},
	}
	s.createMessageRoomLocked("general")
	return s
}

// CreateMessageRoom creates one durable message room.
func (s *Service) CreateMessageRoom(name string) MessageRoom {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createMessageRoomLocked(name)
}

// ListMessageRooms returns all known message rooms.
func (s *Service) ListMessageRooms() []MessageRoom {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MessageRoom, 0, len(s.messageRooms))
	out = append(out, s.messageRooms...)
	return out
}

// GetMessageRoom returns one room by ID or by name.
func (s *Service) GetMessageRoom(roomRef string) (MessageRoom, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roomRef = strings.TrimSpace(roomRef)
	for _, room := range s.messageRooms {
		if strings.EqualFold(room.ID, roomRef) || strings.EqualFold(room.Name, roomRef) {
			return room, true
		}
	}
	return MessageRoom{}, false
}

// ListIdentities returns all registered identities sorted by ID.
func (s *Service) ListIdentities() []Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Identity, 0, len(s.identities))
	for _, identity := range s.identities {
		out = append(out, cloneIdentity(identity))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// GetIdentity returns one identity by ID or alias.
func (s *Service) GetIdentity(ref string) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Identity{}, false
	}

	for _, identity := range s.identities {
		if strings.EqualFold(identity.ID, ref) || sliceContainsFold(identity.Aliases, ref) {
			return cloneIdentity(identity), true
		}
	}
	return Identity{}, false
}

// ListGroups returns all known identity groups in deterministic order.
func (s *Service) ListGroups() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set := make(map[string]struct{})
	for _, identity := range s.identities {
		for _, group := range identity.Groups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			set[strings.ToLower(group)] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for group := range set {
		out = append(out, group)
	}
	sort.Strings(out)
	return out
}

// GetPreferences returns a copy of one identity's preferences.
func (s *Service) GetPreferences(ref string) (map[string]string, bool) {
	identity, ok := s.GetIdentity(ref)
	if !ok {
		return nil, false
	}
	return cloneStringMap(identity.Preferences), true
}

// ResolveAudience returns identities matching the given audience specifier.
func (s *Service) ResolveAudience(audience string) []Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	audience = strings.TrimSpace(audience)
	if audience == "" || strings.EqualFold(audience, "all") {
		out := make([]Identity, 0, len(s.identities))
		for _, identity := range s.identities {
			out = append(out, cloneIdentity(identity))
		}
		return out
	}

	key := audience
	value := audience
	if idx := strings.Index(audience, ":"); idx >= 0 {
		key = strings.TrimSpace(audience[:idx])
		value = strings.TrimSpace(audience[idx+1:])
	}
	if value == "" {
		return nil
	}

	out := make([]Identity, 0)
	for _, identity := range s.identities {
		switch strings.ToLower(key) {
		case "id":
			if strings.EqualFold(identity.ID, value) {
				out = append(out, cloneIdentity(identity))
			}
		case "group":
			if sliceContainsFold(identity.Groups, value) {
				out = append(out, cloneIdentity(identity))
			}
		case "alias":
			if sliceContainsFold(identity.Aliases, value) {
				out = append(out, cloneIdentity(identity))
			}
		}
	}
	return out
}

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
		nextReq := s.sessions[i].ControlRequests[:0]
		for _, req := range s.sessions[i].ControlRequests {
			if strings.EqualFold(req.ParticipantID, participant) {
				continue
			}
			nextReq = append(nextReq, req)
		}
		s.sessions[i].ControlRequests = nextReq
		grantUpdated := false
		for j := range s.sessions[i].ControlGrants {
			if strings.EqualFold(s.sessions[i].ControlGrants[j].ParticipantID, participant) {
				s.sessions[i].ControlGrants[j].GrantedBy = grantedBy
				s.sessions[i].ControlGrants[j].ControlType = controlType
				s.sessions[i].ControlGrants[j].GrantedAt = now
				grantUpdated = true
				break
			}
		}
		if !grantUpdated {
			s.sessions[i].ControlGrants = append(s.sessions[i].ControlGrants, SessionControlGrant{
				ParticipantID: participant,
				GrantedBy:     grantedBy,
				ControlType:   controlType,
				GrantedAt:     now,
			})
		}
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

// PostMessage posts a message to the given room.
func (s *Service) PostMessage(room, text string) Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	roomRef := strings.TrimSpace(room)
	if roomRef == "" {
		roomRef = "general"
	}
	roomRecord := s.ensureRoomLocked(roomRef)
	text = strings.TrimSpace(text)
	msg := Message{
		ID:        s.nextIDLocked("msg"),
		Room:      roomRecord.Name,
		Text:      text,
		CreatedAt: s.now(),
	}
	s.messages = append(s.messages, msg)
	s.appendRecentLocked("message", msg.ID+" room="+msg.Room+" "+msg.Text)
	return msg
}

// SendDirectMessage posts a direct message to one target actor.
func (s *Service) SendDirectMessage(targetRef, text string) Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetRef = normalizeTargetRef(targetRef)
	text = strings.TrimSpace(text)
	msg := Message{
		ID:        s.nextIDLocked("msg"),
		Room:      "dm:" + strings.ReplaceAll(targetRef, ":", "_"),
		TargetRef: targetRef,
		Text:      text,
		CreatedAt: s.now(),
	}
	s.messages = append(s.messages, msg)
	s.appendRecentLocked("message", msg.ID+" dm "+targetRef+" "+msg.Text)
	return msg
}

// ReplyMessageThread posts a reply anchored to one existing root message.
func (s *Service) ReplyMessageThread(rootRef, text string) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rootRef = strings.TrimSpace(rootRef)
	if rootRef == "" {
		return Message{}, false
	}
	var root Message
	found := false
	for _, item := range s.messages {
		if strings.EqualFold(item.ID, rootRef) {
			root = item
			found = true
			break
		}
	}
	if !found {
		return Message{}, false
	}
	rootID := root.ID
	if strings.TrimSpace(root.ThreadRootRef) != "" {
		rootID = root.ThreadRootRef
	}
	msg := Message{
		ID:              s.nextIDLocked("msg"),
		Room:            root.Room,
		TargetRef:       root.TargetRef,
		Text:            strings.TrimSpace(text),
		ThreadRootRef:   rootID,
		ThreadParentRef: root.ID,
		CreatedAt:       s.now(),
	}
	s.messages = append(s.messages, msg)
	s.appendRecentLocked("message", msg.ID+" thread="+msg.ThreadRootRef+" parent="+msg.ThreadParentRef+" "+msg.Text)
	return msg, true
}

// ListMessages returns messages in the given room, or all messages if room is empty.
func (s *Service) ListMessages(room string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roomRef := strings.TrimSpace(room)
	roomName := roomRef
	if roomRef != "" {
		for _, existingRoom := range s.messageRooms {
			if strings.EqualFold(existingRoom.ID, roomRef) || strings.EqualFold(existingRoom.Name, roomRef) {
				roomName = existingRoom.Name
				break
			}
		}
	}
	out := make([]Message, 0, len(s.messages))
	for _, item := range s.messages {
		if roomName == "" || item.Room == roomName {
			out = append(out, item)
		}
	}
	return out
}

// GetMessage returns one message by ID.
func (s *Service) GetMessage(messageID string) (Message, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	messageID = strings.TrimSpace(messageID)
	for _, item := range s.messages {
		if strings.EqualFold(item.ID, messageID) {
			return item, true
		}
	}
	return Message{}, false
}

// AcknowledgeMessage records an acknowledgement from an identity for a message.
func (s *Service) AcknowledgeMessage(identityID, messageID string) (Acknowledgement, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	identityID = strings.TrimSpace(identityID)
	messageID = strings.TrimSpace(messageID)
	if identityID == "" || messageID == "" {
		return Acknowledgement{}, false
	}

	found := false
	for _, item := range s.messages {
		if item.ID == messageID {
			found = true
			break
		}
	}
	if !found {
		return Acknowledgement{}, false
	}

	ack := Acknowledgement{
		IdentityID:     identityID,
		SubjectRef:     "message:" + messageID,
		ActorRef:       "person:" + identityID,
		Mode:           "read",
		AcknowledgedAt: s.now(),
	}
	s.acks[ackKey(ack.ActorRef, ack.SubjectRef)] = ack
	s.appendRecentLocked("message", messageID+" ack "+identityID)
	return ack, true
}

// RecordAcknowledgement stores one acknowledgement for an actor/subject pair.
func (s *Service) RecordAcknowledgement(subjectRef, actorRef, mode string) (Acknowledgement, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subjectRef = strings.TrimSpace(subjectRef)
	actorRef = strings.TrimSpace(actorRef)
	mode = normalizeAckMode(mode)
	if subjectRef == "" || !isValidActorRef(actorRef) || mode == "" {
		return Acknowledgement{}, false
	}

	identityID := ""
	lowerActor := strings.ToLower(actorRef)
	if strings.HasPrefix(lowerActor, "person:") {
		identityID = strings.TrimSpace(actorRef[len("person:"):])
	}

	ack := Acknowledgement{
		IdentityID:     identityID,
		SubjectRef:     subjectRef,
		ActorRef:       actorRef,
		Mode:           mode,
		AcknowledgedAt: s.now(),
	}
	s.acks[ackKey(actorRef, subjectRef)] = ack
	s.appendRecentLocked("identity", subjectRef+" ack "+actorRef+" "+mode)
	return ack, true
}

// GetAcknowledgements returns all acknowledgements, optionally filtered by subject.
func (s *Service) GetAcknowledgements(subjectRef string) []Acknowledgement {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectRef = strings.TrimSpace(subjectRef)
	out := make([]Acknowledgement, 0, len(s.acks))
	for _, ack := range s.acks {
		if subjectRef != "" && !strings.EqualFold(ack.SubjectRef, subjectRef) {
			continue
		}
		out = append(out, ack)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AcknowledgedAt.Equal(out[j].AcknowledgedAt) {
			if out[i].SubjectRef == out[j].SubjectRef {
				return out[i].ActorRef < out[j].ActorRef
			}
			return out[i].SubjectRef < out[j].SubjectRef
		}
		return out[i].AcknowledgedAt.Before(out[j].AcknowledgedAt)
	})
	return out
}

// ListUnreadMessages returns messages that have not been acknowledged by the given identity.
func (s *Service) ListUnreadMessages(identityID, room string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	identityID = strings.TrimSpace(identityID)
	roomRef := strings.TrimSpace(room)
	roomName := roomRef
	if roomRef != "" {
		for _, existingRoom := range s.messageRooms {
			if strings.EqualFold(existingRoom.ID, roomRef) || strings.EqualFold(existingRoom.Name, roomRef) {
				roomName = existingRoom.Name
				break
			}
		}
	}

	out := make([]Message, 0, len(s.messages))
	for _, item := range s.messages {
		if roomName != "" && item.Room != roomName {
			continue
		}
		if identityID != "" {
			if _, ok := s.acks[ackKey("person:"+identityID, "message:"+item.ID)]; ok {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

// PostBoard posts a non-pinned entry to a named board.
func (s *Service) PostBoard(board, text string) BoardItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.postBoardLocked(board, text, false)
}

// PinBoard pins a text item to the named board.
func (s *Service) PinBoard(board, text string) BoardItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.postBoardLocked(board, text, true)
}

func (s *Service) postBoardLocked(board, text string, pinned bool) BoardItem {
	item := BoardItem{
		ID:        s.nextIDLocked("pin"),
		Board:     defaultIfBlank(board, "default"),
		Pinned:    pinned,
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.boardItems = append(s.boardItems, item)
	action := "post"
	if pinned {
		action = "pin"
	}
	s.appendRecentLocked("board", item.ID+" "+action+" "+item.Text)
	return item
}

// ListBoard returns all items pinned to the given board.
func (s *Service) ListBoard(board string) []BoardItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	board = strings.TrimSpace(board)
	out := make([]BoardItem, 0, len(s.boardItems))
	for _, item := range s.boardItems {
		if board == "" || item.Board == board {
			out = append(out, item)
		}
	}
	return out
}

// CreateArtifact creates a new artifact of the given kind and title.
func (s *Service) CreateArtifact(kind, title string) Artifact {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	item := Artifact{
		ID:        s.nextIDLocked("art"),
		Kind:      defaultIfBlank(kind, "document"),
		Title:     strings.TrimSpace(title),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.artifacts = append(s.artifacts, item)
	s.appendArtifactVersionLocked(item, "create")
	s.appendRecentLocked("artifact", item.ID+" "+item.Title)
	return item
}

// PatchArtifact updates the title of the artifact with the given ID.
func (s *Service) PatchArtifact(artifactID, title string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	artifactID = strings.TrimSpace(artifactID)
	title = strings.TrimSpace(title)
	for i := range s.artifacts {
		if s.artifacts[i].ID != artifactID {
			continue
		}
		if title == "" {
			return s.artifacts[i], true
		}
		s.artifacts[i].Title = title
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "patch")
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" patch "+s.artifacts[i].Title)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// ReplaceArtifact replaces the artifact title and records a full replacement version.
func (s *Service) ReplaceArtifact(artifactID, title string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	artifactID = strings.TrimSpace(artifactID)
	title = strings.TrimSpace(title)
	for i := range s.artifacts {
		if s.artifacts[i].ID != artifactID {
			continue
		}
		s.artifacts[i].Title = title
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "replace")
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" replace "+s.artifacts[i].Title)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// SaveArtifactTemplate stores one template by name using an existing artifact as source.
func (s *Service) SaveArtifactTemplate(name, sourceArtifactID string) (ArtifactTemplate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	sourceArtifactID = strings.TrimSpace(sourceArtifactID)
	if name == "" || sourceArtifactID == "" {
		return ArtifactTemplate{}, false
	}
	artifact, ok := s.getArtifactLocked(sourceArtifactID)
	if !ok {
		return ArtifactTemplate{}, false
	}
	now := s.now()
	template := ArtifactTemplate{
		Name:             name,
		SourceArtifactID: artifact.ID,
		SourceKind:       artifact.Kind,
		SourceTitle:      artifact.Title,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if existing, ok := s.templates[name]; ok {
		template.CreatedAt = existing.CreatedAt
	}
	s.templates[name] = template
	s.appendRecentLocked("artifact", "template save "+template.Name+" -> "+template.SourceArtifactID)
	return template, true
}

// ApplyArtifactTemplate applies a saved template to an existing target artifact.
func (s *Service) ApplyArtifactTemplate(name, targetArtifactID string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	targetArtifactID = strings.TrimSpace(targetArtifactID)
	template, ok := s.templates[name]
	if !ok || targetArtifactID == "" {
		return Artifact{}, false
	}
	for i := range s.artifacts {
		if s.artifacts[i].ID != targetArtifactID {
			continue
		}
		s.artifacts[i].Kind = template.SourceKind
		s.artifacts[i].Title = template.SourceTitle
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "template.apply:"+template.Name)
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" template "+template.Name)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// GetArtifact returns one artifact by ID.
func (s *Service) GetArtifact(artifactID string) (Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artifactID = strings.TrimSpace(artifactID)
	for _, item := range s.artifacts {
		if item.ID == artifactID {
			return item, true
		}
	}
	return Artifact{}, false
}

// ListArtifacts returns all stored artifacts.
func (s *Service) ListArtifacts() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Artifact(nil), s.artifacts...)
}

func (s *Service) getArtifactLocked(artifactID string) (Artifact, bool) {
	for _, item := range s.artifacts {
		if item.ID == artifactID {
			return item, true
		}
	}
	return Artifact{}, false
}

// ArtifactHistory returns version history for an artifact in creation order.
func (s *Service) ArtifactHistory(artifactID string) ([]ArtifactVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artifactID = strings.TrimSpace(artifactID)
	versions, ok := s.versions[artifactID]
	if !ok {
		return nil, false
	}
	return append([]ArtifactVersion(nil), versions...), true
}

// AnnotateCanvas appends an annotation to the named canvas.
func (s *Service) AnnotateCanvas(canvas, text string) Annotation {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := Annotation{
		ID:        s.nextIDLocked("ann"),
		Canvas:    defaultIfBlank(canvas, "default"),
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.annotations = append(s.annotations, item)
	s.appendRecentLocked("canvas", item.ID+" "+item.Text)
	return item
}

// ListCanvas returns annotations on the given canvas.
func (s *Service) ListCanvas(canvas string) []Annotation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canvas = strings.TrimSpace(canvas)
	out := make([]Annotation, 0, len(s.annotations))
	for _, item := range s.annotations {
		if canvas == "" || item.Canvas == canvas {
			out = append(out, item)
		}
	}
	return out
}

// Search returns items whose text matches the query across messages, board, artifacts and memories.
func (s *Service) Search(query string) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil
	}
	out := make([]SearchResult, 0)
	for _, item := range s.searchCorpusLocked() {
		if strings.Contains(strings.ToLower(item.Text), needle) {
			out = append(out, SearchResult{ID: item.ID, Kind: item.Kind, Text: item.Text})
		}
	}
	return out
}

// SearchTimeline returns activity records in timeline order optionally filtered by scope.
func (s *Service) SearchTimeline(scope string) []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	out := make([]RecentItem, 0, len(s.recent))
	for _, item := range s.recent {
		if needle != "" && !strings.EqualFold(item.Kind, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// SearchRelated returns indexed items related to the given subject reference or phrase.
func (s *Service) SearchRelated(subjectRef string) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subjectRef = strings.TrimSpace(subjectRef)
	if subjectRef == "" {
		return nil
	}
	tokens := normalizedTokens(subjectRef)
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		result    SearchResult
		score     int
		createdAt time.Time
	}
	matches := make([]scored, 0)
	subjectLower := strings.ToLower(subjectRef)
	for _, item := range s.searchCorpusLocked() {
		score := 0
		idLower := strings.ToLower(item.ID)
		textLower := strings.ToLower(item.Text)
		if strings.EqualFold(item.ID, subjectRef) {
			score += 3
		}
		for _, token := range tokens {
			if strings.Contains(textLower, token) || strings.Contains(idLower, token) || strings.Contains(subjectLower, idLower) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		matches = append(matches, scored{
			result:    SearchResult{ID: item.ID, Kind: item.Kind, Text: item.Text},
			score:     score,
			createdAt: item.CreatedAt,
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if !matches[i].createdAt.Equal(matches[j].createdAt) {
			return matches[i].createdAt.After(matches[j].createdAt)
		}
		return matches[i].result.ID < matches[j].result.ID
	})
	out := make([]SearchResult, 0, len(matches))
	for _, item := range matches {
		out = append(out, item.result)
	}
	return out
}

// SearchRecent returns the newest timeline entries for a scope.
func (s *Service) SearchRecent(scope string, limit int) []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	if limit <= 0 {
		limit = 20
	}
	buffer := make([]RecentItem, 0, limit)
	for _, item := range s.recent {
		if needle != "" && !strings.EqualFold(item.Kind, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		buffer = append(buffer, item)
		if len(buffer) > limit {
			buffer = append([]RecentItem(nil), buffer[len(buffer)-limit:]...)
		}
	}
	return buffer
}

// Remember stores a memory entry in the given scope.
func (s *Service) Remember(scope, text string) MemoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := MemoryEntry{
		ID:        s.nextIDLocked("mem"),
		Scope:     defaultIfBlank(scope, "general"),
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.memories = append(s.memories, item)
	s.appendRecentLocked("memory", item.ID+" "+item.Text)
	return item
}

// Recall returns memory entries whose text or scope matches the query.
func (s *Service) Recall(query string) []MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(query))
	out := make([]MemoryEntry, 0, len(s.memories))
	for _, item := range s.memories {
		if needle == "" || strings.Contains(strings.ToLower(item.Text), needle) || strings.Contains(strings.ToLower(item.Scope), needle) {
			out = append(out, item)
		}
	}
	return out
}

// MemoryStream returns memory entries in insertion order with optional scope filtering.
func (s *Service) MemoryStream(scope string) []MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	out := make([]MemoryEntry, 0, len(s.memories))
	for _, item := range s.memories {
		if needle != "" && !strings.EqualFold(item.Scope, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// ListRecent returns the most recent activity entries in insertion order.
func (s *Service) ListRecent() []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]RecentItem(nil), s.recent...)
}

// CohortUpsert creates or updates one named device cohort.
func (s *Service) CohortUpsert(name string, selectors []string) DeviceCohort {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	cohort := DeviceCohort{
		Name:      name,
		Selectors: normalizeSelectors(selectors),
		UpdatedAt: s.now(),
	}
	s.cohorts[name] = cohort
	s.appendRecentLocked("cohort", name+" upsert")
	return cohort
}

// CohortGet returns one cohort by name.
func (s *Service) CohortGet(name string) (DeviceCohort, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cohort, ok := s.cohorts[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return DeviceCohort{}, false
	}
	cohort.Selectors = append([]string(nil), cohort.Selectors...)
	return cohort, true
}

// CohortList returns all cohorts sorted by name.
func (s *Service) CohortList() []DeviceCohort {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cohorts := make([]DeviceCohort, 0, len(s.cohorts))
	for _, cohort := range s.cohorts {
		copyCohort := cohort
		copyCohort.Selectors = append([]string(nil), cohort.Selectors...)
		cohorts = append(cohorts, copyCohort)
	}
	sort.Slice(cohorts, func(i, j int) bool { return cohorts[i].Name < cohorts[j].Name })
	return cohorts
}

// CohortDelete removes one cohort by name.
func (s *Service) CohortDelete(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	if _, ok := s.cohorts[name]; !ok {
		return false
	}
	delete(s.cohorts, name)
	s.appendRecentLocked("cohort", name+" deleted")
	return true
}

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

// StorePut sets a key/value record in the given namespace.
// ttl <= 0 means the record does not expire.
func (s *Service) StorePut(namespace, key, value string, ttl time.Duration) StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	trimmedKey := strings.TrimSpace(key)
	storeKey := ns + ":" + trimmedKey
	existing, hasExisting := s.store[storeKey]
	record := StoreRecord{
		Namespace: ns,
		Key:       trimmedKey,
		Value:     value,
		UpdatedAt: s.now(),
	}
	if hasExisting {
		record.Binding = existing.Binding
	}
	if ttl > 0 {
		expiresAt := s.now().Add(ttl)
		record.ExpiresAt = &expiresAt
	}
	s.store[storeKey] = record
	s.appendRecentLocked("store", storeKey)
	return record
}

// StoreGet retrieves a record by namespace and key, returning false if not found.
func (s *Service) StoreGet(namespace, key string) (StoreRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	record, ok := s.store[storeKey]
	if !ok {
		return StoreRecord{}, false
	}
	if storeRecordExpired(record, s.now()) {
		delete(s.store, storeKey)
		return StoreRecord{}, false
	}
	return record, ok
}

// StoreList returns all records in the given namespace sorted by key.
func (s *Service) StoreList(namespace string) []StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	out := make([]StoreRecord, 0)
	now := s.now()
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		if record.Namespace == ns {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// StoreDelete removes a record by namespace and key.
func (s *Service) StoreDelete(namespace, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	if _, ok := s.store[storeKey]; !ok {
		return false
	}
	delete(s.store, storeKey)
	s.appendRecentLocked("store", storeKey+" deleted")
	return true
}

// StoreNamespaces returns namespace inventory sorted by namespace.
func (s *Service) StoreNamespaces() []StoreNamespaceSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	counts := map[string]int{}
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		counts[record.Namespace]++
	}
	namespaces := make([]StoreNamespaceSummary, 0, len(counts))
	for namespace, count := range counts {
		namespaces = append(namespaces, StoreNamespaceSummary{Name: namespace, RecordCount: count})
	}
	sort.Slice(namespaces, func(i, j int) bool { return namespaces[i].Name < namespaces[j].Name })
	return namespaces
}

// StoreWatch returns records in a namespace with an optional key prefix.
func (s *Service) StoreWatch(namespace, prefix string) []StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := defaultIfBlank(namespace, "default")
	prefix = strings.TrimSpace(prefix)
	out := make([]StoreRecord, 0)
	now := s.now()
	for key, record := range s.store {
		if storeRecordExpired(record, now) {
			delete(s.store, key)
			continue
		}
		if record.Namespace != ns {
			continue
		}
		if prefix != "" && !strings.HasPrefix(record.Key, prefix) {
			continue
		}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// StoreBind binds an existing record to a device:scenario selector.
func (s *Service) StoreBind(namespace, key, binding string) (StoreRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	record, ok := s.store[storeKey]
	if !ok {
		return StoreRecord{}, false
	}
	if storeRecordExpired(record, s.now()) {
		delete(s.store, storeKey)
		return StoreRecord{}, false
	}
	record.Binding = strings.TrimSpace(binding)
	record.UpdatedAt = s.now()
	s.store[storeKey] = record
	s.appendRecentLocked("store", storeKey+" bound")
	return record, true
}

func storeRecordExpired(record StoreRecord, now time.Time) bool {
	return record.ExpiresAt != nil && !record.ExpiresAt.After(now)
}

// BusEmit emits a named event with an optional payload on the event bus.
func (s *Service) BusEmit(kind, name, payload string) BusEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := BusEvent{
		ID:        s.nextIDLocked("bus"),
		Kind:      defaultIfBlank(kind, "event"),
		Name:      defaultIfBlank(name, "unnamed"),
		Payload:   strings.TrimSpace(payload),
		CreatedAt: s.now(),
	}
	s.bus = append(s.bus, event)
	s.appendRecentLocked("bus", event.ID+" "+event.Name)
	return event
}

// BusTail returns events emitted on the event bus with optional filtering.
func (s *Service) BusTail(kind, name string, limit int) []BusEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterBusEvents(s.bus, kind, name, limit)
}

// BusReplay returns events within an inclusive ID window with optional filtering.
func (s *Service) BusReplay(fromID, toID, kind, name string, limit int) []BusEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	window := busWindowByID(s.bus, strings.TrimSpace(fromID), strings.TrimSpace(toID))
	return filterBusEvents(window, kind, name, limit)
}

// HandlerList returns all registered runtime handlers sorted by id.
func (s *Service) HandlerList() []HandlerRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HandlerRegistration, 0, len(s.handlers))
	for _, handler := range s.handlers {
		out = append(out, handler)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// HandlerOnRun registers a routing handler that executes a REPL command when matched.
func (s *Service) HandlerOnRun(selector, action, command string) HandlerRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	handler := HandlerRegistration{
		ID:         s.nextIDLocked("handler"),
		Selector:   normalizeHandlerSelector(selector),
		Action:     normalizeHandlerAction(action),
		RunCommand: strings.TrimSpace(command),
		UpdatedAt:  s.now(),
	}
	s.handlers[handler.ID] = handler
	s.appendRecentLocked("handler", handler.ID+" on")
	return handler
}

// HandlerOnEmit registers a routing handler that emits a bus event or intent when matched.
func (s *Service) HandlerOnEmit(selector, action, emitKind, emitName, emitPayload string) HandlerRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	handler := HandlerRegistration{
		ID:          s.nextIDLocked("handler"),
		Selector:    normalizeHandlerSelector(selector),
		Action:      normalizeHandlerAction(action),
		EmitKind:    defaultIfBlank(strings.ToLower(strings.TrimSpace(emitKind)), "intent"),
		EmitName:    strings.TrimSpace(emitName),
		EmitPayload: strings.TrimSpace(emitPayload),
		UpdatedAt:   s.now(),
	}
	s.handlers[handler.ID] = handler
	s.appendRecentLocked("handler", handler.ID+" on")
	return handler
}

// HandlerOff removes one registered handler by id.
func (s *Service) HandlerOff(handlerID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	handlerID = strings.TrimSpace(handlerID)
	if _, ok := s.handlers[handlerID]; !ok {
		return false
	}
	delete(s.handlers, handlerID)
	s.appendRecentLocked("handler", handlerID+" off")
	return true
}

func busWindowByID(events []BusEvent, fromID, toID string) []BusEvent {
	if len(events) == 0 {
		return nil
	}
	start := 0
	if fromID != "" {
		for i, event := range events {
			if event.ID == fromID {
				start = i
				break
			}
		}
	}
	end := len(events) - 1
	if toID != "" {
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].ID == toID {
				end = i
				break
			}
		}
	}
	if start > end {
		return nil
	}
	return append([]BusEvent(nil), events[start:end+1]...)
}

func filterBusEvents(events []BusEvent, kind, name string, limit int) []BusEvent {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	filtered := make([]BusEvent, 0, len(events))
	for _, event := range events {
		if kind != "" && !strings.EqualFold(event.Kind, kind) {
			continue
		}
		if name != "" && !strings.EqualFold(event.Name, name) {
			continue
		}
		filtered = append(filtered, event)
	}
	if limit > 0 && len(filtered) > limit {
		return append([]BusEvent(nil), filtered[len(filtered)-limit:]...)
	}
	return filtered
}

func (s *Service) nextIDLocked(prefix string) string {
	s.seq++
	return prefix + "-" + strconv64(s.seq)
}

func (s *Service) appendRecentLocked(kind, text string) {
	item := RecentItem{
		ID:        s.nextIDLocked("recent"),
		Kind:      kind,
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.recent = append(s.recent, item)
	if len(s.recent) > 200 {
		s.recent = append([]RecentItem(nil), s.recent[len(s.recent)-200:]...)
	}
}

func defaultIfBlank(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeTargetRef(targetRef string) string {
	targetRef = strings.TrimSpace(targetRef)
	if targetRef == "" {
		return "person:unknown"
	}
	if strings.Contains(targetRef, ":") {
		return targetRef
	}
	return "person:" + targetRef
}

func (s *Service) createMessageRoomLocked(name string) MessageRoom {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "general"
	}
	for _, room := range s.messageRooms {
		if strings.EqualFold(room.Name, name) {
			return room
		}
	}
	room := MessageRoom{
		ID:            s.nextIDLocked("msgroom"),
		Name:          name,
		Audience:      "household:all",
		RetentionDays: 30,
		CreatedAt:     s.now(),
	}
	s.messageRooms = append(s.messageRooms, room)
	s.appendRecentLocked("message", room.ID+" room.create "+room.Name)
	return room
}

func (s *Service) ensureRoomLocked(roomRef string) MessageRoom {
	roomRef = strings.TrimSpace(roomRef)
	for _, room := range s.messageRooms {
		if strings.EqualFold(room.ID, roomRef) || strings.EqualFold(room.Name, roomRef) {
			return room
		}
	}
	return s.createMessageRoomLocked(roomRef)
}

func normalizedTokens(value string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		field = strings.Trim(field, " .,;:!?()[]{}\"'")
		if len(field) < 2 {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func normalizeSelectors(selectors []string) []string {
	if len(selectors) == 0 {
		return nil
	}
	out := make([]string, 0, len(selectors))
	seen := make(map[string]struct{}, len(selectors))
	for _, selector := range selectors {
		normalized := strings.ToLower(strings.TrimSpace(selector))
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

func normalizeHandlerSelector(selector string) string {
	return strings.ToLower(strings.TrimSpace(selector))
}

func normalizeHandlerAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}

func (s *Service) searchCorpusLocked() []searchableItem {
	out := make([]searchableItem, 0, len(s.messages)+len(s.boardItems)+len(s.artifacts)+len(s.memories))
	for _, item := range s.messages {
		out = append(out, searchableItem{ID: item.ID, Kind: "message", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.boardItems {
		out = append(out, searchableItem{ID: item.ID, Kind: "board", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.artifacts {
		out = append(out, searchableItem{ID: item.ID, Kind: "artifact", Text: item.Title, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.memories {
		out = append(out, searchableItem{ID: item.ID, Kind: "memory", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	return out
}

func sliceContainsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

func strconv64(v uint64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
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

func ackKey(identityID, subjectRef string) string {
	return strings.ToLower(strings.TrimSpace(identityID)) + "|" + strings.ToLower(strings.TrimSpace(subjectRef))
}

func cloneIdentity(identity Identity) Identity {
	identity.Groups = append([]string(nil), identity.Groups...)
	identity.Aliases = append([]string(nil), identity.Aliases...)
	identity.Preferences = cloneStringMap(identity.Preferences)
	return identity
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func normalizeAckMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		return "read"
	}
	return mode
}

func isValidActorRef(actorRef string) bool {
	actorRef = strings.TrimSpace(actorRef)
	if actorRef == "" {
		return false
	}
	lower := strings.ToLower(actorRef)
	for _, prefix := range []string{"person:", "device:", "agent:", "anonymous:"} {
		if strings.HasPrefix(lower, prefix) && strings.TrimSpace(actorRef[len(prefix):]) != "" {
			return true
		}
	}
	return false
}

func (s *Service) appendArtifactVersionLocked(item Artifact, action string) {
	s.versions[item.ID] = append(s.versions[item.ID], ArtifactVersion{
		ArtifactID: item.ID,
		Version:    item.Version,
		Kind:       item.Kind,
		Title:      item.Title,
		Action:     strings.TrimSpace(action),
		CreatedAt:  item.UpdatedAt,
	})
}
