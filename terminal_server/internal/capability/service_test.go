package capability

import (
	"strings"
	"testing"
	"time"
)

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

func TestCohortCRUDAndNormalization(t *testing.T) {
	svc := NewService()

	created := svc.CohortUpsert("Family-Screens", []string{"zone:kitchen", " role:screen ", "ZONE:KITCHEN"})
	if created.Name != "family-screens" {
		t.Fatalf("CohortUpsert name = %q, want family-screens", created.Name)
	}
	if len(created.Selectors) != 2 || created.Selectors[0] != "role:screen" || created.Selectors[1] != "zone:kitchen" {
		t.Fatalf("CohortUpsert selectors = %+v, want [role:screen zone:kitchen]", created.Selectors)
	}

	fetched, ok := svc.CohortGet("FAMILY-SCREENS")
	if !ok {
		t.Fatalf("CohortGet(FAMILY-SCREENS) = not found")
	}
	if fetched.Name != created.Name {
		t.Fatalf("CohortGet name = %q, want %q", fetched.Name, created.Name)
	}

	svc.CohortUpsert("kitchen-only", []string{"zone:kitchen"})
	list := svc.CohortList()
	if len(list) != 2 {
		t.Fatalf("len(CohortList()) = %d, want 2", len(list))
	}
	if list[0].Name != "family-screens" || list[1].Name != "kitchen-only" {
		t.Fatalf("CohortList order = %+v, want [family-screens kitchen-only]", []string{list[0].Name, list[1].Name})
	}

	if deleted := svc.CohortDelete("family-screens"); !deleted {
		t.Fatalf("CohortDelete(family-screens) = false, want true")
	}
	if _, ok := svc.CohortGet("family-screens"); ok {
		t.Fatalf("CohortGet(family-screens) should not exist after delete")
	}
	if deleted := svc.CohortDelete("family-screens"); deleted {
		t.Fatalf("second CohortDelete should return false")
	}
}

func TestUIViewCRUDAndNormalization(t *testing.T) {
	svc := NewService()

	created := svc.UIViewUpsert("Kitchen-Home", "root-main", `{\"type\":\"stack\"}`)
	if created.ViewID != "kitchen-home" {
		t.Fatalf("UIViewUpsert view_id = %q, want kitchen-home", created.ViewID)
	}
	if created.RootID != "root-main" {
		t.Fatalf("UIViewUpsert root_id = %q, want root-main", created.RootID)
	}

	fetched, ok := svc.UIViewGet("KITCHEN-HOME")
	if !ok {
		t.Fatalf("UIViewGet(KITCHEN-HOME) = not found")
	}
	if fetched.ViewID != created.ViewID {
		t.Fatalf("UIViewGet view_id = %q, want %q", fetched.ViewID, created.ViewID)
	}

	svc.UIViewUpsert("alerts", "root-alert", `{\"type\":\"banner\"}`)
	list := svc.UIViewList()
	if len(list) != 2 {
		t.Fatalf("len(UIViewList()) = %d, want 2", len(list))
	}
	if list[0].ViewID != "alerts" || list[1].ViewID != "kitchen-home" {
		t.Fatalf("UIViewList order = %+v, want [alerts kitchen-home]", []string{list[0].ViewID, list[1].ViewID})
	}

	if deleted := svc.UIViewDelete("kitchen-home"); !deleted {
		t.Fatalf("UIViewDelete(kitchen-home) = false, want true")
	}
	if _, ok := svc.UIViewGet("kitchen-home"); ok {
		t.Fatalf("UIViewGet(kitchen-home) should not exist after delete")
	}
	if deleted := svc.UIViewDelete("kitchen-home"); deleted {
		t.Fatalf("second UIViewDelete should return false")
	}
}

func TestUIActiveOperationsAndSnapshot(t *testing.T) {
	svc := NewService()

	pushed := svc.UIPush("device-1", `{"type":"stack"}`, "root-main")
	if pushed.DeviceID != "device-1" {
		t.Fatalf("UIPush device_id = %q, want device-1", pushed.DeviceID)
	}
	if pushed.RootID != "root-main" {
		t.Fatalf("UIPush root_id = %q, want root-main", pushed.RootID)
	}

	patched := svc.UIPatch("device-1", "banner", `{"type":"text"}`)
	if patched.LastPatchComponentID != "banner" {
		t.Fatalf("UIPatch component_id = %q, want banner", patched.LastPatchComponentID)
	}

	transitioned := svc.UITransition("device-1", "banner", "fade", 150)
	if transitioned.LastTransition != "fade" || transitioned.LastTransitionDurationMS != 150 {
		t.Fatalf("UITransition = %+v, want transition fade with duration 150", transitioned)
	}

	broadcast := svc.UIBroadcast("family-screens", `{"type":"banner"}`, "alert-banner", []string{"device-1", "device-2", "device-1"})
	if len(broadcast.Devices) != 2 || broadcast.Devices[0] != "device-1" || broadcast.Devices[1] != "device-2" {
		t.Fatalf("UIBroadcast devices = %+v, want [device-1 device-2]", broadcast.Devices)
	}

	subscribed := svc.UISubscribe("device-1", "cohort:family-screens")
	if len(subscribed.Subscriptions) != 1 || subscribed.Subscriptions[0] != "cohort:family-screens" {
		t.Fatalf("UISubscribe subscriptions = %+v, want [cohort:family-screens]", subscribed.Subscriptions)
	}

	snapshot, ok := svc.UISnapshot("device-1")
	if !ok {
		t.Fatalf("UISnapshot(device-1) = not found")
	}
	if snapshot.LastPatchComponentID != "alert-banner" {
		t.Fatalf("UISnapshot last patch component = %q, want alert-banner", snapshot.LastPatchComponentID)
	}
	if !strings.Contains(strings.ToLower(snapshot.LastPatchDescriptor), "banner") {
		t.Fatalf("UISnapshot patch descriptor = %q, want banner descriptor", snapshot.LastPatchDescriptor)
	}
	if len(snapshot.Subscriptions) != 1 || snapshot.Subscriptions[0] != "cohort:family-screens" {
		t.Fatalf("UISnapshot subscriptions = %+v, want [cohort:family-screens]", snapshot.Subscriptions)
	}

	if _, ok := svc.UISnapshot("missing-device"); ok {
		t.Fatalf("UISnapshot(missing-device) should report not found")
	}
}

func TestIdentityLookupGroupsPreferencesAndAcknowledgements(t *testing.T) {
	svc := NewService()

	identity, ok := svc.GetIdentity("admin")
	if !ok {
		t.Fatalf("GetIdentity(admin) = not found, want system")
	}
	if identity.ID != "system" {
		t.Fatalf("GetIdentity(admin).ID = %q, want system", identity.ID)
	}

	groups := svc.ListGroups()
	if len(groups) != 2 || groups[0] != "family" || groups[1] != "operators" {
		t.Fatalf("ListGroups() = %+v, want [family operators]", groups)
	}

	prefs, ok := svc.GetPreferences("system")
	if !ok {
		t.Fatalf("GetPreferences(system) = not found")
	}
	if prefs["notifications"] != "normal" {
		t.Fatalf("GetPreferences(system)[notifications] = %q, want normal", prefs["notifications"])
	}

	ack, ok := svc.RecordAcknowledgement("message:msg-1", "device:kitchen-screen", "dismissed")
	if !ok {
		t.Fatalf("RecordAcknowledgement(...) returned false")
	}
	if ack.ActorRef != "device:kitchen-screen" || ack.Mode != "dismissed" {
		t.Fatalf("ack = %+v, want actor=device:kitchen-screen mode=dismissed", ack)
	}
	if _, ok := svc.RecordAcknowledgement("message:msg-1", "invalid", "read"); ok {
		t.Fatalf("RecordAcknowledgement should reject invalid actor refs")
	}

	acks := svc.GetAcknowledgements("message:msg-1")
	if len(acks) != 1 {
		t.Fatalf("GetAcknowledgements(message:msg-1) len = %d, want 1", len(acks))
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

func TestSessionAttachDetachAndControlLifecycle(t *testing.T) {
	svc := NewService()
	session := svc.CreateSession("shared_view", "kitchen")

	attached, ok := svc.AttachDevice(session.ID, "device:kiosk-1")
	if !ok {
		t.Fatalf("AttachDevice(%q,device:kiosk-1) reported missing session", session.ID)
	}
	if len(attached.AttachedDevices) != 1 || attached.AttachedDevices[0] != "device:kiosk-1" {
		t.Fatalf("AttachDevice attached devices = %+v, want [device:kiosk-1]", attached.AttachedDevices)
	}

	detached, ok := svc.DetachDevice(session.ID, "device:kiosk-1")
	if !ok {
		t.Fatalf("DetachDevice(%q,device:kiosk-1) reported missing session", session.ID)
	}
	if len(detached.AttachedDevices) != 0 {
		t.Fatalf("DetachDevice should remove attached device, got %+v", detached.AttachedDevices)
	}

	requested, ok := svc.RequestControl(session.ID, "alice", "keyboard")
	if !ok {
		t.Fatalf("RequestControl(%q,alice) reported missing session", session.ID)
	}
	if len(requested.ControlRequests) != 1 {
		t.Fatalf("RequestControl requests = %+v, want one request", requested.ControlRequests)
	}
	if requested.ControlRequests[0].ParticipantID != "alice" || requested.ControlRequests[0].ControlType != "keyboard" {
		t.Fatalf("RequestControl request[0] = %+v, want participant=alice control_type=keyboard", requested.ControlRequests[0])
	}

	granted, ok := svc.GrantControl(session.ID, "alice", "moderator", "keyboard")
	if !ok {
		t.Fatalf("GrantControl(%q,alice) reported missing session", session.ID)
	}
	if len(granted.ControlRequests) != 0 {
		t.Fatalf("GrantControl should clear request, got %+v", granted.ControlRequests)
	}
	if len(granted.ControlGrants) != 1 {
		t.Fatalf("GrantControl grants = %+v, want one grant", granted.ControlGrants)
	}
	if granted.ControlGrants[0].ParticipantID != "alice" || granted.ControlGrants[0].GrantedBy != "moderator" {
		t.Fatalf("GrantControl grant[0] = %+v, want participant=alice granted_by=moderator", granted.ControlGrants[0])
	}

	revoked, ok := svc.RevokeControl(session.ID, "alice", "moderator")
	if !ok {
		t.Fatalf("RevokeControl(%q,alice) reported missing session", session.ID)
	}
	if len(revoked.ControlGrants) != 0 {
		t.Fatalf("RevokeControl should clear grants, got %+v", revoked.ControlGrants)
	}

	audit := revoked.Audit
	if len(audit) < 6 {
		t.Fatalf("audit events = %+v, want at least create/attach/detach/request/grant/revoke", audit)
	}
	last := audit[len(audit)-1]
	if last.Action != "session.control.revoke" || last.Actor != "moderator" || last.Target != "alice" {
		t.Fatalf("last audit event = %+v, want control revoke by moderator for alice", last)
	}
}

func TestMessageAcknowledgeUnreadAndArtifactPatch(t *testing.T) {
	svc := NewService()

	room := svc.CreateMessageRoom("room-1")
	if strings.TrimSpace(room.ID) == "" || room.Name != "room-1" {
		t.Fatalf("CreateMessageRoom(room-1) = %+v, want non-empty id and name room-1", room)
	}
	if gotRoom, ok := svc.GetMessageRoom(room.ID); !ok || gotRoom.ID != room.ID {
		t.Fatalf("GetMessageRoom(%s) = (%+v,%v), want id=%s,true", room.ID, gotRoom, ok, room.ID)
	}
	if gotRoom, ok := svc.GetMessageRoom(room.Name); !ok || gotRoom.ID != room.ID {
		t.Fatalf("GetMessageRoom(%s) = (%+v,%v), want id=%s,true", room.Name, gotRoom, ok, room.ID)
	}

	message := svc.PostMessage("room-1", "remember the groceries")
	messageGet, ok := svc.GetMessage(message.ID)
	if !ok || messageGet.ID != message.ID {
		t.Fatalf("GetMessage(%q) = (%+v,%v), want id=%s,true", message.ID, messageGet, ok, message.ID)
	}

	reply, ok := svc.ReplyMessageThread(message.ID, "adding eggs")
	if !ok {
		t.Fatalf("ReplyMessageThread(%q, adding eggs) returned false", message.ID)
	}
	if reply.ThreadRootRef != message.ID || reply.ThreadParentRef != message.ID {
		t.Fatalf("ReplyMessageThread thread refs = root:%q parent:%q, want both %q", reply.ThreadRootRef, reply.ThreadParentRef, message.ID)
	}
	if reply.Room != message.Room {
		t.Fatalf("ReplyMessageThread room = %q, want %q", reply.Room, message.Room)
	}

	direct := svc.SendDirectMessage("mom", "come downstairs")
	if direct.TargetRef != "person:mom" {
		t.Fatalf("SendDirectMessage target = %q, want person:mom", direct.TargetRef)
	}
	if !strings.HasPrefix(direct.Room, "dm:") {
		t.Fatalf("SendDirectMessage room = %q, want dm:*", direct.Room)
	}

	boardPost := svc.PostBoard("family", "Need milk")
	if boardPost.Pinned {
		t.Fatalf("PostBoard should create non-pinned entries")
	}
	boardPin := svc.PinBoard("family", "Dinner in 10")
	if !boardPin.Pinned {
		t.Fatalf("PinBoard should create pinned entries")
	}

	unread := svc.ListUnreadMessages("alice", "room-1")
	if len(unread) != 2 || unread[0].ID != message.ID || unread[1].ID != reply.ID {
		t.Fatalf("ListUnreadMessages before ack = %+v, want [%s %s]", unread, message.ID, reply.ID)
	}
	if _, ok := svc.AcknowledgeMessage("alice", message.ID); !ok {
		t.Fatalf("AcknowledgeMessage(%q,%q) = false, want true", "alice", message.ID)
	}
	ackEntries := svc.GetAcknowledgements("message:" + message.ID)
	if len(ackEntries) != 1 {
		t.Fatalf("GetAcknowledgements(message:%s) len = %d, want 1", message.ID, len(ackEntries))
	}
	if ackEntries[0].ActorRef != "person:alice" || ackEntries[0].Mode != "read" {
		t.Fatalf("message ack entry = %+v, want actor=person:alice mode=read", ackEntries[0])
	}
	unread = svc.ListUnreadMessages("alice", "room-1")
	if len(unread) != 1 || unread[0].ID != reply.ID {
		t.Fatalf("ListUnreadMessages after first ack = %+v, want [%s]", unread, reply.ID)
	}
	if _, ok := svc.AcknowledgeMessage("alice", reply.ID); !ok {
		t.Fatalf("AcknowledgeMessage(%q,%q) for reply = false, want true", "alice", reply.ID)
	}
	unread = svc.ListUnreadMessages("alice", "room-1")
	if len(unread) != 0 {
		t.Fatalf("ListUnreadMessages after acking thread reply = %+v, want none", unread)
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

func TestStoreTTLExpirationAndPruning(t *testing.T) {
	svc := NewService()
	now := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	stored := svc.StorePut("notes", "timer", "active", 2*time.Second)
	if stored.ExpiresAt == nil {
		t.Fatalf("StorePut with ttl should set expires_at")
	}

	record, ok := svc.StoreGet("notes", "timer")
	if !ok {
		t.Fatalf("StoreGet(notes,timer) before expiry = false, want true")
	}
	if record.ExpiresAt == nil || !record.ExpiresAt.Equal(now.Add(2*time.Second)) {
		t.Fatalf("StoreGet(notes,timer) expires_at = %v, want %v", record.ExpiresAt, now.Add(2*time.Second))
	}

	now = now.Add(3 * time.Second)
	if _, ok := svc.StoreGet("notes", "timer"); ok {
		t.Fatalf("StoreGet(notes,timer) after expiry = true, want false")
	}
	if records := svc.StoreList("notes"); len(records) != 0 {
		t.Fatalf("StoreList(notes) after expiry = %+v, want empty", records)
	}
}

func TestStorePutWithoutTTLHasNoExpiration(t *testing.T) {
	svc := NewService()
	record := svc.StorePut("notes", "persist", "value", 0)
	if record.ExpiresAt != nil {
		t.Fatalf("StorePut without ttl should not set expires_at, got %v", record.ExpiresAt)
	}
}

func TestStoreNamespacesDeleteWatchAndBind(t *testing.T) {
	svc := NewService()
	svc.StorePut("notes", "alpha", "one", 0)
	svc.StorePut("notes", "beta", "two", 0)
	svc.StorePut("alerts", "fire", "active", 0)

	namespaces := svc.StoreNamespaces()
	if len(namespaces) != 2 {
		t.Fatalf("len(StoreNamespaces()) = %d, want 2", len(namespaces))
	}
	if namespaces[0].Name != "alerts" || namespaces[0].RecordCount != 1 {
		t.Fatalf("StoreNamespaces()[0] = %+v, want alerts with 1 record", namespaces[0])
	}
	if namespaces[1].Name != "notes" || namespaces[1].RecordCount != 2 {
		t.Fatalf("StoreNamespaces()[1] = %+v, want notes with 2 records", namespaces[1])
	}

	watched := svc.StoreWatch("notes", "a")
	if len(watched) != 1 || watched[0].Key != "alpha" {
		t.Fatalf("StoreWatch(notes,a) = %+v, want only alpha", watched)
	}

	record, ok := svc.StoreBind("notes", "alpha", "device-1:chat")
	if !ok {
		t.Fatalf("StoreBind(notes,alpha,...) = false, want true")
	}
	if record.Binding != "device-1:chat" {
		t.Fatalf("StoreBind binding = %q, want device-1:chat", record.Binding)
	}

	deleted := svc.StoreDelete("notes", "beta")
	if !deleted {
		t.Fatalf("StoreDelete(notes,beta) = false, want true")
	}
	if _, ok := svc.StoreGet("notes", "beta"); ok {
		t.Fatalf("StoreGet(notes,beta) after delete = true, want false")
	}
}

func TestBusTailFilterAndReplayWindow(t *testing.T) {
	svc := NewService()
	one := svc.BusEmit("event", "alarm", "ring")
	two := svc.BusEmit("event", "door", "open")
	three := svc.BusEmit("intent", "assist", "kitchen")

	filtered := svc.BusTail("event", "", 1)
	if len(filtered) != 1 || filtered[0].ID != two.ID {
		t.Fatalf("BusTail(event,limit=1) = %+v, want only %s", filtered, two.ID)
	}

	replay := svc.BusReplay(one.ID, three.ID, "event", "door", 0)
	if len(replay) != 1 || replay[0].ID != two.ID {
		t.Fatalf("BusReplay window/filter = %+v, want only %s", replay, two.ID)
	}
}

func TestHandlerRegisterListAndOff(t *testing.T) {
	svc := NewService()

	runHandler := svc.HandlerOnRun(" scenario=chat ", " submit ", "store put chat last hello")
	if runHandler.ID == "" {
		t.Fatalf("HandlerOnRun should assign id")
	}
	if runHandler.Selector != "scenario=chat" {
		t.Fatalf("HandlerOnRun selector = %q, want scenario=chat", runHandler.Selector)
	}
	if runHandler.Action != "submit" {
		t.Fatalf("HandlerOnRun action = %q, want submit", runHandler.Action)
	}
	if runHandler.RunCommand != "store put chat last hello" {
		t.Fatalf("HandlerOnRun run_command = %q, want command", runHandler.RunCommand)
	}

	emitHandler := svc.HandlerOnEmit("device=d1", "tap", "intent", "alert_ack", "device=d1")
	if emitHandler.EmitKind != "intent" || emitHandler.EmitName != "alert_ack" {
		t.Fatalf("HandlerOnEmit = %+v, want intent/alert_ack", emitHandler)
	}

	list := svc.HandlerList()
	if len(list) != 2 {
		t.Fatalf("len(HandlerList()) = %d, want 2", len(list))
	}
	if list[0].ID != runHandler.ID || list[1].ID != emitHandler.ID {
		t.Fatalf("HandlerList order = [%s %s], want [%s %s]", list[0].ID, list[1].ID, runHandler.ID, emitHandler.ID)
	}

	if deleted := svc.HandlerOff(runHandler.ID); !deleted {
		t.Fatalf("HandlerOff(%q) = false, want true", runHandler.ID)
	}
	if deleted := svc.HandlerOff(runHandler.ID); deleted {
		t.Fatalf("second HandlerOff(%q) should be false", runHandler.ID)
	}

	remaining := svc.HandlerList()
	if len(remaining) != 1 || remaining[0].ID != emitHandler.ID {
		t.Fatalf("remaining handlers = %+v, want only %s", remaining, emitHandler.ID)
	}
}

func TestScenarioDefineListGetAndUndefine(t *testing.T) {
	svc := NewService()

	defined := svc.ScenarioDefine(InlineScenarioDefinition{
		Name:         " Red_Alert ",
		MatchIntents: []string{"red alert", " red alert ", "all hands"},
		MatchEvents:  []string{"alarm.triggered"},
		Priority:     "HIGH",
		OnStart:      "ui broadcast all_screens '{\"type\":\"banner\"}'",
		OnInput:      "handlers on scenario=red_alert submit --emit intent alert_ack",
		OnEvents: []InlineScenarioEventHook{
			{Kind: "alarm.triggered", Command: "bus emit event alarm.ack"},
			{Kind: "", Command: "ignored"},
		},
		OnSuspend: "store put alerts red_alert suspended",
		OnResume:  "store put alerts red_alert resumed",
		OnStop:    "ui broadcast all_screens '{\"type\":\"clear\"}'",
	})
	if defined.Name != "red_alert" {
		t.Fatalf("ScenarioDefine name = %q, want red_alert", defined.Name)
	}
	if defined.Priority != "high" {
		t.Fatalf("ScenarioDefine priority = %q, want high", defined.Priority)
	}
	if len(defined.MatchIntents) != 2 {
		t.Fatalf("ScenarioDefine intents = %+v, want 2 unique intents", defined.MatchIntents)
	}
	if len(defined.OnEvents) != 1 {
		t.Fatalf("ScenarioDefine on_events = %+v, want 1 valid event hook", defined.OnEvents)
	}

	listed := svc.ScenarioList()
	if len(listed) != 1 {
		t.Fatalf("len(ScenarioList()) = %d, want 1", len(listed))
	}
	if listed[0].Name != "red_alert" {
		t.Fatalf("ScenarioList()[0].Name = %q, want red_alert", listed[0].Name)
	}

	found, ok := svc.ScenarioGet("RED_ALERT")
	if !ok {
		t.Fatalf("ScenarioGet(RED_ALERT) = false, want true")
	}
	if found.OnStart == "" || found.OnStop == "" {
		t.Fatalf("ScenarioGet hooks missing expected values: %+v", found)
	}

	updated := svc.ScenarioDefine(InlineScenarioDefinition{Name: "red_alert", Priority: "unknown"})
	if updated.Priority != "normal" {
		t.Fatalf("ScenarioDefine unknown priority = %q, want normal", updated.Priority)
	}

	if deleted := svc.ScenarioUndefine("red_alert"); !deleted {
		t.Fatalf("ScenarioUndefine(red_alert) = false, want true")
	}
	if deleted := svc.ScenarioUndefine("red_alert"); deleted {
		t.Fatalf("second ScenarioUndefine(red_alert) should be false")
	}
	if _, ok := svc.ScenarioGet("red_alert"); ok {
		t.Fatalf("ScenarioGet(red_alert) after undefine = true, want false")
	}
}

func TestSimDeviceInputAndScriptDryRunLifecycle(t *testing.T) {
	svc := NewService()

	device := svc.SimDeviceUpsert("Kitchen-Sim", []string{"display", "keyboard", "display"})
	if device.DeviceID != "kitchen-sim" {
		t.Fatalf("SimDeviceUpsert device id = %q, want kitchen-sim", device.DeviceID)
	}
	if len(device.Caps) != 2 {
		t.Fatalf("SimDeviceUpsert caps = %+v, want deduped caps", device.Caps)
	}

	listed := svc.SimDeviceList()
	if len(listed) != 1 || listed[0].DeviceID != "kitchen-sim" {
		t.Fatalf("SimDeviceList = %+v, want kitchen-sim", listed)
	}

	if _, ok := svc.SimRecordInput("missing", "banner", "tap", ""); ok {
		t.Fatalf("SimRecordInput for missing device should return false")
	}

	record, ok := svc.SimRecordInput("kitchen-sim", "banner", "tap", "ack")
	if !ok {
		t.Fatalf("SimRecordInput(kitchen-sim,...) = false, want true")
	}
	if record.ID == "" || record.Action != "tap" {
		t.Fatalf("SimRecordInput result = %+v, want id and tap action", record)
	}

	inputs := svc.SimInputs("kitchen-sim")
	if len(inputs) != 1 || inputs[0].ID != record.ID {
		t.Fatalf("SimInputs(kitchen-sim) = %+v, want [%s]", inputs, record.ID)
	}

	svc.UIPush("kitchen-sim", `{"type":"stack","children":[{"type":"text","text":"hello"}]}`, "sim-root")
	if _, ok := svc.SimExpect("missing", "ui", "hello", 0); ok {
		t.Fatalf("SimExpect for missing device should return false")
	}
	uiExpectation, ok := svc.SimExpect("kitchen-sim", "ui", "hello", 2*time.Second)
	if !ok {
		t.Fatalf("SimExpect(kitchen-sim,ui,hello) = false, want true")
	}
	if !uiExpectation.Matched {
		t.Fatalf("SimExpect(kitchen-sim,ui,hello) matched = false, want true (%+v)", uiExpectation)
	}
	if uiExpectation.Within != "2s" {
		t.Fatalf("SimExpect within = %q, want 2s", uiExpectation.Within)
	}

	svc.BusEmit("event", "alarm", "kitchen-sim:alert")
	messageExpectation, ok := svc.SimExpect("kitchen-sim", "message", "alert", 0)
	if !ok || !messageExpectation.Matched {
		t.Fatalf("SimExpect(kitchen-sim,message,alert) = %+v, ok=%v, want matched true", messageExpectation, ok)
	}

	recording, ok := svc.SimRecord("kitchen-sim", 30*time.Second)
	if !ok {
		t.Fatalf("SimRecord(kitchen-sim) = false, want true")
	}
	if recording.Snapshot.DeviceID != "kitchen-sim" {
		t.Fatalf("SimRecord snapshot device = %q, want kitchen-sim", recording.Snapshot.DeviceID)
	}
	if len(recording.Inputs) == 0 {
		t.Fatalf("SimRecord inputs = %+v, want at least one input", recording.Inputs)
	}
	if len(recording.Messages) == 0 {
		t.Fatalf("SimRecord messages = %+v, want at least one message", recording.Messages)
	}
	if recording.Duration != "30s" {
		t.Fatalf("SimRecord duration = %q, want 30s", recording.Duration)
	}

	dryRun := svc.ScriptDryRun("fixtures/smoke.term", "# comment\n\nstore put notes k v\nui push d1 banner")
	if dryRun.Path != "fixtures/smoke.term" {
		t.Fatalf("ScriptDryRun path = %q, want fixtures/smoke.term", dryRun.Path)
	}
	if dryRun.CommandCount != 2 || dryRun.SkippedCount != 2 {
		t.Fatalf("ScriptDryRun counts = commands:%d skipped:%d, want 2/2", dryRun.CommandCount, dryRun.SkippedCount)
	}

	run := svc.ScriptRun("fixtures/smoke.term", "# comment\n\nstore put notes k v\nui push d1 banner\nmessage post phase12-room fixture-layer2-mutating\nboard post phase12-board fixture-board-mutating\nartifact create lesson fixture-artifact-mutating\ncanvas annotate phase12-canvas fixture-canvas-mutating\nsession create lesson phase12-session\nsession join latest fixture-session-member\nmemory remember phase12-memory fixture-memory-mutating\nmessage ls phase12-room\nboard ls phase12-board\nartifact history latest\ncanvas ls phase12-canvas\nsession members latest\nmemory recall fixture-memory-mutating\nmessage rooms")
	if run.Path != "fixtures/smoke.term" {
		t.Fatalf("ScriptRun path = %q, want fixtures/smoke.term", run.Path)
	}
	if run.CommandCount != 16 || run.SkippedCount != 2 || run.ExecutedCount != 16 || run.FailedCount != 0 {
		t.Fatalf("ScriptRun counts = commands:%d skipped:%d executed:%d failed:%d, want 16/2/16/0", run.CommandCount, run.SkippedCount, run.ExecutedCount, run.FailedCount)
	}
	stored, ok := svc.StoreGet("notes", "k")
	if !ok || stored.Value != "v" {
		t.Fatalf("ScriptRun store side effect missing: ok=%v record=%+v", ok, stored)
	}
	messages := svc.ListMessages("phase12-room")
	if len(messages) != 1 || messages[0].Text != "fixture-layer2-mutating" {
		t.Fatalf("ScriptRun message side effect missing: %+v", messages)
	}
	boards := svc.ListBoard("phase12-board")
	if len(boards) != 1 || boards[0].Text != "fixture-board-mutating" {
		t.Fatalf("ScriptRun board side effect missing: %+v", boards)
	}
	artifacts := svc.ListArtifacts()
	artifactID := ""
	for _, artifact := range artifacts {
		if artifact.Title == "fixture-artifact-mutating" {
			artifactID = artifact.ID
			break
		}
	}
	if strings.TrimSpace(artifactID) == "" {
		t.Fatalf("ScriptRun artifact side effect missing: %+v", artifacts)
	}
	versions, ok := svc.ArtifactHistory(artifactID)
	if !ok || len(versions) != 1 || versions[0].Action != "create" {
		t.Fatalf("ScriptRun artifact history side effect missing: ok=%v versions=%+v", ok, versions)
	}
	annotations := svc.ListCanvas("phase12-canvas")
	if len(annotations) != 1 || annotations[0].Text != "fixture-canvas-mutating" {
		t.Fatalf("ScriptRun canvas side effect missing: %+v", annotations)
	}
	sessions := svc.ListSessions()
	if len(sessions) != 1 || sessions[0].Kind != "lesson" || sessions[0].Target != "phase12-session" {
		t.Fatalf("ScriptRun session create side effect missing: %+v", sessions)
	}
	participants, ok := svc.ListSessionParticipants(sessions[0].ID)
	if !ok || len(participants) != 1 || participants[0].IdentityID != "fixture-session-member" {
		t.Fatalf("ScriptRun session join side effect missing: ok=%v participants=%+v", ok, participants)
	}
	memories := svc.Recall("fixture-memory-mutating")
	if len(memories) != 1 || memories[0].Scope != "phase12-memory" || memories[0].Text != "fixture-memory-mutating" {
		t.Fatalf("ScriptRun memory side effect missing: %+v", memories)
	}
	snapshot, ok := svc.UISnapshot("d1")
	if !ok || snapshot.DeviceID != "d1" {
		t.Fatalf("ScriptRun ui push side effect missing: ok=%v snapshot=%+v", ok, snapshot)
	}

	failed := svc.ScriptRun("fixtures/fail.term", "sim device rm missing")
	if failed.CommandCount != 1 || failed.ExecutedCount != 0 || failed.FailedCount != 1 {
		t.Fatalf("failed ScriptRun counts = commands:%d executed:%d failed:%d, want 1/0/1", failed.CommandCount, failed.ExecutedCount, failed.FailedCount)
	}
	if len(failed.Issues) != 1 || !strings.Contains(failed.Issues[0], "sim device not found") {
		t.Fatalf("failed ScriptRun issues = %+v, want sim device not found", failed.Issues)
	}

	if deleted := svc.SimDeviceDelete("kitchen-sim"); !deleted {
		t.Fatalf("SimDeviceDelete(kitchen-sim) = false, want true")
	}
	if deleted := svc.SimDeviceDelete("kitchen-sim"); deleted {
		t.Fatalf("second SimDeviceDelete(kitchen-sim) should be false")
	}
	if got := svc.SimInputs("kitchen-sim"); len(got) != 0 {
		t.Fatalf("SimInputs(kitchen-sim) after delete = %+v, want empty", got)
	}
}

func TestMessageRoomThreadUnreadAcknowledgeLifecycle(t *testing.T) {
	svc := NewService()

	room := svc.CreateMessageRoom("kitchen")
	if room.Name != "kitchen" {
		t.Fatalf("CreateMessageRoom(kitchen).Name = %q, want kitchen", room.Name)
	}

	root := svc.PostMessage("kitchen", "Dinner in 10")
	if root.Room != "kitchen" {
		t.Fatalf("PostMessage room = %q, want kitchen", root.Room)
	}

	reply, ok := svc.ReplyMessageThread(root.ID, "On my way")
	if !ok {
		t.Fatalf("ReplyMessageThread(%q) returned false", root.ID)
	}
	if reply.ThreadRootRef != root.ID || reply.ThreadParentRef != root.ID {
		t.Fatalf("thread refs = root:%q parent:%q, want both %q", reply.ThreadRootRef, reply.ThreadParentRef, root.ID)
	}

	unread := svc.ListUnreadMessages("alice", "kitchen")
	if len(unread) != 2 {
		t.Fatalf("ListUnreadMessages(alice,kitchen) len = %d, want 2", len(unread))
	}

	if _, ok := svc.AcknowledgeMessage("alice", root.ID); !ok {
		t.Fatalf("AcknowledgeMessage(alice,%q) returned false", root.ID)
	}
	if _, ok := svc.AcknowledgeMessage("alice", reply.ID); !ok {
		t.Fatalf("AcknowledgeMessage(alice,%q) returned false", reply.ID)
	}

	unread = svc.ListUnreadMessages("alice", "kitchen")
	if len(unread) != 0 {
		t.Fatalf("ListUnreadMessages(alice,kitchen) after ack = %+v, want empty", unread)
	}
}
