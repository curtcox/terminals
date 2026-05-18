package capability

import (
	"sort"
	"strings"
)

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
		return cloneIdentities(s.identities)
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
		if identityMatchesAudience(identity, key, value) {
			out = append(out, cloneIdentity(identity))
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

func ackKey(identityID, subjectRef string) string {
	return strings.ToLower(strings.TrimSpace(identityID)) + "|" + strings.ToLower(strings.TrimSpace(subjectRef))
}

func cloneIdentity(identity Identity) Identity {
	identity.Groups = append([]string(nil), identity.Groups...)
	identity.Aliases = append([]string(nil), identity.Aliases...)
	identity.Preferences = cloneStringMap(identity.Preferences)
	return identity
}

func cloneIdentities(identities []Identity) []Identity {
	out := make([]Identity, 0, len(identities))
	for _, identity := range identities {
		out = append(out, cloneIdentity(identity))
	}
	return out
}

func identityMatchesAudience(identity Identity, key, value string) bool {
	switch strings.ToLower(key) {
	case "id":
		return strings.EqualFold(identity.ID, value)
	case "group":
		return sliceContainsFold(identity.Groups, value)
	case "alias":
		return sliceContainsFold(identity.Aliases, value)
	default:
		return false
	}
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
