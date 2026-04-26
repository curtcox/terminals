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
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name,omitempty"`
	Groups      []string  `json:"groups,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// InteractiveSession represents a collaborative session between participants.
type InteractiveSession struct {
	ID           string               `json:"id"`
	Kind         string               `json:"kind"`
	Target       string               `json:"target"`
	Participants []SessionParticipant `json:"participants,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
}

// SessionParticipant records a single identity's membership in a session.
type SessionParticipant struct {
	IdentityID string    `json:"identity_id"`
	JoinedAt   time.Time `json:"joined_at"`
}

// Message represents a posted message within a room.
type Message struct {
	ID        string    `json:"id"`
	Room      string    `json:"room"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// BoardItem represents a pinned item on a named board.
type BoardItem struct {
	ID        string    `json:"id"`
	Board     string    `json:"board"`
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
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BusEvent represents a named event emitted on the internal event bus.
type BusEvent struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Service provides typed in-memory storage for capability closure tools.
type Service struct {
	mu sync.RWMutex

	now func() time.Time
	seq uint64

	identities  []Identity
	sessions    []InteractiveSession
	messages    []Message
	boardItems  []BoardItem
	artifacts   []Artifact
	versions    map[string][]ArtifactVersion
	annotations []Annotation
	memories    []MemoryEntry
	recent      []RecentItem
	store       map[string]StoreRecord
	bus         []BusEvent
	acks        map[string]Acknowledgement
}

// NewService creates a new Service with default seed data.
func NewService() *Service {
	now := time.Now
	s := &Service{
		now:      func() time.Time { return now().UTC() },
		store:    map[string]StoreRecord{},
		acks:     map[string]Acknowledgement{},
		versions: map[string][]ArtifactVersion{},
		identities: []Identity{
			{
				ID:          "system",
				DisplayName: "System",
				Groups:      []string{"family", "operators"},
				Aliases:     []string{"admin", "house"},
				CreatedAt:   now().UTC(),
			},
		},
	}
	return s
}

// ListIdentities returns all registered identities sorted by ID.
func (s *Service) ListIdentities() []Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Identity(nil), s.identities...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ResolveAudience returns identities matching the given audience specifier.
func (s *Service) ResolveAudience(audience string) []Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	audience = strings.TrimSpace(audience)
	if audience == "" || strings.EqualFold(audience, "all") {
		return append([]Identity(nil), s.identities...)
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
				out = append(out, identity)
			}
		case "group":
			if sliceContainsFold(identity.Groups, value) {
				out = append(out, identity)
			}
		case "alias":
			if sliceContainsFold(identity.Aliases, value) {
				out = append(out, identity)
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
	session := InteractiveSession{
		ID:        s.nextIDLocked("sess"),
		Kind:      kind,
		Target:    target,
		CreatedAt: s.now(),
	}
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
			s.appendRecentLocked("session", s.sessions[i].ID+" leave "+participant)
		}
		return cloneSession(s.sessions[i]), true
	}
	return InteractiveSession{}, false
}

// PostMessage posts a message to the given room.
func (s *Service) PostMessage(room, text string) Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	room = defaultIfBlank(room, "general")
	text = strings.TrimSpace(text)
	msg := Message{
		ID:        s.nextIDLocked("msg"),
		Room:      room,
		Text:      text,
		CreatedAt: s.now(),
	}
	s.messages = append(s.messages, msg)
	s.appendRecentLocked("message", msg.ID+" "+msg.Text)
	return msg
}

// ListMessages returns messages in the given room, or all messages if room is empty.
func (s *Service) ListMessages(room string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room = strings.TrimSpace(room)
	out := make([]Message, 0, len(s.messages))
	for _, item := range s.messages {
		if room == "" || item.Room == room {
			out = append(out, item)
		}
	}
	return out
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
		AcknowledgedAt: s.now(),
	}
	s.acks[ackKey(identityID, ack.SubjectRef)] = ack
	s.appendRecentLocked("message", messageID+" ack "+identityID)
	return ack, true
}

// ListUnreadMessages returns messages that have not been acknowledged by the given identity.
func (s *Service) ListUnreadMessages(identityID, room string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	identityID = strings.TrimSpace(identityID)
	room = strings.TrimSpace(room)

	out := make([]Message, 0, len(s.messages))
	for _, item := range s.messages {
		if room != "" && item.Room != room {
			continue
		}
		if identityID != "" {
			if _, ok := s.acks[ackKey(identityID, "message:"+item.ID)]; ok {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

// PinBoard pins a text item to the named board.
func (s *Service) PinBoard(board, text string) BoardItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := BoardItem{
		ID:        s.nextIDLocked("pin"),
		Board:     defaultIfBlank(board, "default"),
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.boardItems = append(s.boardItems, item)
	s.appendRecentLocked("board", item.ID+" "+item.Text)
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

// StorePut sets a key/value record in the given namespace.
func (s *Service) StorePut(namespace, key, value string) StoreRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := StoreRecord{
		Namespace: defaultIfBlank(namespace, "default"),
		Key:       strings.TrimSpace(key),
		Value:     value,
		UpdatedAt: s.now(),
	}
	storeKey := record.Namespace + ":" + record.Key
	s.store[storeKey] = record
	s.appendRecentLocked("store", storeKey)
	return record
}

// StoreGet retrieves a record by namespace and key, returning false if not found.
func (s *Service) StoreGet(namespace, key string) (StoreRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storeKey := defaultIfBlank(namespace, "default") + ":" + strings.TrimSpace(key)
	record, ok := s.store[storeKey]
	return record, ok
}

// StoreList returns all records in the given namespace sorted by key.
func (s *Service) StoreList(namespace string) []StoreRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns := defaultIfBlank(namespace, "default")
	out := make([]StoreRecord, 0)
	for _, record := range s.store {
		if record.Namespace == ns {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
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

// BusTail returns all events emitted on the event bus.
func (s *Service) BusTail() []BusEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]BusEvent(nil), s.bus...)
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
	return item
}

func ackKey(identityID, subjectRef string) string {
	return strings.ToLower(strings.TrimSpace(identityID)) + "|" + strings.ToLower(strings.TrimSpace(subjectRef))
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
