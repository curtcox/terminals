package capability

import "testing"

func TestResolveAudienceByGroupAndAlias(t *testing.T) {
	svc := NewService()

	groupMatches := svc.ResolveAudience("group:family")
	if len(groupMatches) == 0 {
		t.Fatalf("ResolveAudience(group:family) returned no identities")
	}

	aliasMatches := svc.ResolveAudience("alias:admin")
	if len(aliasMatches) != 1 || aliasMatches[0].ID != "system" {
		t.Fatalf("ResolveAudience(alias:admin) = %+v, want [system]", aliasMatches)
	}
}

func TestSessionJoinAndLeave(t *testing.T) {
	svc := NewService()
	session := svc.CreateSession("lesson", "math-room")

	joined, ok := svc.JoinSession(session.ID, "alice")
	if !ok {
		t.Fatalf("JoinSession(%q, alice) reported missing session", session.ID)
	}
	if len(joined.Participants) != 1 || joined.Participants[0].IdentityID != "alice" {
		t.Fatalf("JoinSession participants = %+v, want [alice]", joined.Participants)
	}

	joined, ok = svc.JoinSession(session.ID, "alice")
	if !ok {
		t.Fatalf("JoinSession(%q, alice) second call reported missing session", session.ID)
	}
	if len(joined.Participants) != 1 {
		t.Fatalf("JoinSession should be idempotent, participants = %+v", joined.Participants)
	}

	left, ok := svc.LeaveSession(session.ID, "alice")
	if !ok {
		t.Fatalf("LeaveSession(%q, alice) reported missing session", session.ID)
	}
	if len(left.Participants) != 0 {
		t.Fatalf("LeaveSession should remove participant, participants = %+v", left.Participants)
	}
}

func TestMessageAcknowledgeUnreadAndArtifactPatch(t *testing.T) {
	svc := NewService()

	message := svc.PostMessage("room-1", "remember the groceries")
	unread := svc.ListUnreadMessages("alice", "room-1")
	if len(unread) != 1 || unread[0].ID != message.ID {
		t.Fatalf("ListUnreadMessages before ack = %+v, want [%s]", unread, message.ID)
	}
	if _, ok := svc.AcknowledgeMessage("alice", message.ID); !ok {
		t.Fatalf("AcknowledgeMessage(%q,%q) = false, want true", "alice", message.ID)
	}
	unread = svc.ListUnreadMessages("alice", "room-1")
	if len(unread) != 0 {
		t.Fatalf("ListUnreadMessages after ack = %+v, want none", unread)
	}

	artifact := svc.CreateArtifact("lesson", "math lesson")
	patched, ok := svc.PatchArtifact(artifact.ID, "advanced math lesson")
	if !ok {
		t.Fatalf("PatchArtifact(%q) reported missing artifact", artifact.ID)
	}
	if patched.Title != "advanced math lesson" {
		t.Fatalf("PatchArtifact title = %q, want %q", patched.Title, "advanced math lesson")
	}
	results := svc.Search("advanced")
	if len(results) == 0 || results[0].ID != artifact.ID {
		t.Fatalf("Search(advanced) should include patched artifact id %q, got %+v", artifact.ID, results)
	}
}
