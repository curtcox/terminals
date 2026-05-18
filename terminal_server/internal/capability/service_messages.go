package capability

import (
	"strings"
)

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

// ListUnreadMessages returns messages that have not been acknowledged by the given identity.
func (s *Service) ListUnreadMessages(identityID, room string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	identityID = strings.TrimSpace(identityID)
	roomName := s.messageRoomNameLocked(room)

	out := make([]Message, 0, len(s.messages))
	for _, item := range s.messages {
		if s.messageIsUnreadForLocked(item, identityID, roomName) {
			out = append(out, item)
		}
	}
	return out
}

func (s *Service) messageRoomNameLocked(room string) string {
	roomRef := strings.TrimSpace(room)
	if roomRef == "" {
		return ""
	}
	for _, existingRoom := range s.messageRooms {
		if strings.EqualFold(existingRoom.ID, roomRef) || strings.EqualFold(existingRoom.Name, roomRef) {
			return existingRoom.Name
		}
	}
	return roomRef
}

func (s *Service) messageIsUnreadForLocked(item Message, identityID, roomName string) bool {
	if roomName != "" && item.Room != roomName {
		return false
	}
	if identityID == "" {
		return true
	}
	_, acknowledged := s.acks[ackKey("person:"+identityID, "message:"+item.ID)]
	return !acknowledged
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
