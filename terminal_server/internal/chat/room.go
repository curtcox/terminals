// Package chat implements the in-memory chat room shared by all terminals.
//
// The room is a single-channel log of messages with a fixed retention window
// (by default 24h) and a per-device identity map. State is purely in-memory;
// it is lost on server restart. Callers coordinate UI broadcasting; this
// package only owns data.
package chat

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultRetention is how long messages are kept before being trimmed.
const DefaultRetention = 24 * time.Hour

// Message is one chat entry.
type Message struct {
	ID       string
	DeviceID string
	Name     string
	Text     string
	At       time.Time
}

// Room holds the shared chat state.
type Room struct {
	mu           sync.Mutex
	retention    time.Duration
	now          func() time.Time
	messages     []Message
	identities   map[string]string // deviceID -> display name
	participants map[string]time.Time
	seq          uint64
}

// NewRoom builds a room with the default retention.
func NewRoom() *Room {
	return NewRoomWithRetention(DefaultRetention)
}

// NewRoomWithRetention builds a room with a custom retention window.
func NewRoomWithRetention(retention time.Duration) *Room {
	if retention <= 0 {
		retention = DefaultRetention
	}
	return &Room{
		retention:    retention,
		now:          func() time.Time { return time.Now().UTC() },
		identities:   make(map[string]string),
		participants: make(map[string]time.Time),
	}
}

// Join registers a device as a participant. If no identity is set, a fallback
// is derived from the deviceID. Returns the current identity.
func (r *Room) Join(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.participants[deviceID] = r.now()
	if name, ok := r.identities[deviceID]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	return ""
}

// Leave removes a device from the active participant set. The identity and
// any messages they sent are retained until expiry.
func (r *Room) Leave(deviceID string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.participants, deviceID)
}

// Participants returns the current active participant device ids.
func (r *Room) Participants() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.participants))
	for deviceID := range r.participants {
		out = append(out, deviceID)
	}
	sort.Strings(out)
	return out
}

// SetName assigns a display name to a device. Empty names are ignored.
func (r *Room) SetName(deviceID, name string) string {
	deviceID = strings.TrimSpace(deviceID)
	name = strings.TrimSpace(name)
	if deviceID == "" || name == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.identities[deviceID] = name
	return name
}

// Name returns the display name assigned to a device, or "" if none.
func (r *Room) Name(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.identities[deviceID]
}

// Post appends a message. The returned message includes its assigned ID and
// timestamp. Empty text is rejected with ok=false. The caller is responsible
// for notifying participants; this method only records state.
func (r *Room) Post(deviceID, name, text string) (Message, bool) {
	deviceID = strings.TrimSpace(deviceID)
	text = strings.TrimSpace(text)
	if deviceID == "" || text == "" {
		return Message{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		r.identities[deviceID] = trimmed
		name = trimmed
	} else if stored, ok := r.identities[deviceID]; ok {
		name = stored
	}
	if strings.TrimSpace(name) == "" {
		name = deviceID
	}
	r.seq++
	now := r.now()
	msg := Message{
		ID:       formatSeq(r.seq),
		DeviceID: deviceID,
		Name:     name,
		Text:     text,
		At:       now,
	}
	r.messages = append(r.messages, msg)
	r.trimLocked(now)
	return msg, true
}

// Messages returns a copy of the current message log, oldest first, after
// trimming anything outside the retention window.
func (r *Room) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trimLocked(r.now())
	out := make([]Message, len(r.messages))
	copy(out, r.messages)
	return out
}

func (r *Room) trimLocked(now time.Time) {
	if len(r.messages) == 0 {
		return
	}
	cutoff := now.Add(-r.retention)
	idx := 0
	for idx < len(r.messages) && r.messages[idx].At.Before(cutoff) {
		idx++
	}
	if idx == 0 {
		return
	}
	r.messages = append(r.messages[:0], r.messages[idx:]...)
}

func formatSeq(n uint64) string {
	// Short, sortable ids sufficient for in-memory disambiguation.
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%36]
		n /= 36
	}
	return string(buf[i:])
}
