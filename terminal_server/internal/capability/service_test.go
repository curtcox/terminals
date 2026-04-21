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
