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
	if patched.Version != 2 {
		t.Fatalf("PatchArtifact version = %d, want 2", patched.Version)
	}

	stored, ok := svc.GetArtifact(artifact.ID)
	if !ok {
		t.Fatalf("GetArtifact(%q) reported missing artifact", artifact.ID)
	}
	if stored.Version != 2 {
		t.Fatalf("GetArtifact version = %d, want 2", stored.Version)
	}

	history, ok := svc.ArtifactHistory(artifact.ID)
	if !ok {
		t.Fatalf("ArtifactHistory(%q) reported missing artifact", artifact.ID)
	}
	if len(history) != 2 {
		t.Fatalf("len(ArtifactHistory(%q)) = %d, want 2", artifact.ID, len(history))
	}
	if history[0].Action != "create" || history[0].Version != 1 {
		t.Fatalf("history[0] = %+v, want action=create version=1", history[0])
	}
	if history[1].Action != "patch" || history[1].Version != 2 {
		t.Fatalf("history[1] = %+v, want action=patch version=2", history[1])
	}

	results := svc.Search("advanced")
	if len(results) == 0 || results[0].ID != artifact.ID {
		t.Fatalf("Search(advanced) should include patched artifact id %q, got %+v", artifact.ID, results)
	}
}

func TestArtifactReplaceAndTemplateApply(t *testing.T) {
	svc := NewService()

	source := svc.CreateArtifact("lesson", "fractions basics")
	target := svc.CreateArtifact("note", "scratchpad")

	replaced, ok := svc.ReplaceArtifact(target.ID, "scratchpad v2")
	if !ok {
		t.Fatalf("ReplaceArtifact(%q) reported missing artifact", target.ID)
	}
	if replaced.Title != "scratchpad v2" || replaced.Version != 2 {
		t.Fatalf("ReplaceArtifact result = %+v, want title=scratchpad v2 version=2", replaced)
	}

	template, ok := svc.SaveArtifactTemplate("lesson-base", source.ID)
	if !ok {
		t.Fatalf("SaveArtifactTemplate(lesson-base,%q) failed", source.ID)
	}
	if template.SourceArtifactID != source.ID {
		t.Fatalf("template source = %q, want %q", template.SourceArtifactID, source.ID)
	}

	applied, ok := svc.ApplyArtifactTemplate("lesson-base", target.ID)
	if !ok {
		t.Fatalf("ApplyArtifactTemplate(lesson-base,%q) failed", target.ID)
	}
	if applied.Kind != source.Kind || applied.Title != source.Title {
		t.Fatalf("applied artifact = %+v, want kind=%q title=%q", applied, source.Kind, source.Title)
	}

	history, ok := svc.ArtifactHistory(target.ID)
	if !ok {
		t.Fatalf("ArtifactHistory(%q) reported missing artifact", target.ID)
	}
	if len(history) != 3 {
		t.Fatalf("len(ArtifactHistory(%q)) = %d, want 3", target.ID, len(history))
	}
	if history[1].Action != "replace" {
		t.Fatalf("history[1].Action = %q, want replace", history[1].Action)
	}
	if history[2].Action != "template.apply:lesson-base" {
		t.Fatalf("history[2].Action = %q, want template.apply:lesson-base", history[2].Action)
	}
}

func TestSearchTimelineRelatedRecentAndMemoryStream(t *testing.T) {
	svc := NewService()
	svc.PostMessage("kitchen", "buy milk and bread")
	svc.PinBoard("groceries", "milk list")
	svc.CreateArtifact("note", "family milk plan")
	memKitchen := svc.Remember("kitchen", "milk on shelf two")
	svc.Remember("garage", "replace bulb")

	timeline := svc.SearchTimeline("memory")
	if len(timeline) == 0 {
		t.Fatalf("SearchTimeline(memory) returned no rows")
	}
	if timeline[len(timeline)-1].Kind != "memory" {
		t.Fatalf("SearchTimeline(memory) last kind = %q, want memory", timeline[len(timeline)-1].Kind)
	}

	related := svc.SearchRelated("milk")
	if len(related) < 3 {
		t.Fatalf("SearchRelated(milk) = %+v, want at least 3 matches", related)
	}

	recent := svc.SearchRecent("memory", 1)
	if len(recent) != 1 {
		t.Fatalf("SearchRecent(memory,1) len = %d, want 1", len(recent))
	}
	if recent[0].Kind != "memory" {
		t.Fatalf("SearchRecent(memory,1) kind = %q, want memory", recent[0].Kind)
	}

	stream := svc.MemoryStream("kitchen")
	if len(stream) != 1 {
		t.Fatalf("MemoryStream(kitchen) len = %d, want 1", len(stream))
	}
	if stream[0].ID != memKitchen.ID {
		t.Fatalf("MemoryStream(kitchen)[0].ID = %q, want %q", stream[0].ID, memKitchen.ID)
	}
}
