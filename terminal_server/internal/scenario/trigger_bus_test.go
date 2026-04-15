package scenario

import (
	"testing"
	"time"
)

func TestSourceFromKindMapping(t *testing.T) {
	cases := []struct {
		name string
		kind TriggerKind
		want TriggerSource
	}{
		{name: "voice", kind: TriggerVoice, want: SourceVoice},
		{name: "schedule", kind: TriggerSchedule, want: SourceSchedule},
		{name: "event", kind: TriggerEvent, want: SourceEvent},
		{name: "cascade", kind: TriggerCascade, want: SourceCascade},
		{name: "manual", kind: TriggerManual, want: SourceManual},
		{name: "unknown defaults to manual", kind: TriggerKind("unknown"), want: SourceManual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourceFromKind(tc.kind); got != tc.want {
				t.Fatalf("sourceFromKind(%q) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestNormalizeTriggerDefaultsIntentAndEventSources(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	got := normalizeTrigger(Trigger{
		Kind:      TriggerSchedule,
		SourceID:  "  d1  ",
		Intent:    "  reminder  ",
		Arguments: map[string]string{"k": "v"},
		EventV2: &EventRecord{
			Kind: "  timer.fired  ",
		},
	}, now)

	if got.SourceID != "d1" {
		t.Fatalf("SourceID = %q, want d1", got.SourceID)
	}
	if got.Intent != "reminder" {
		t.Fatalf("Intent = %q, want reminder", got.Intent)
	}
	if got.IntentV2 == nil {
		t.Fatalf("expected IntentV2 to be populated")
	}
	if got.IntentV2.Source != SourceSchedule {
		t.Fatalf("IntentV2.Source = %q, want %q", got.IntentV2.Source, SourceSchedule)
	}
	if got.EventV2 == nil {
		t.Fatalf("expected EventV2 to remain populated")
	}
	if got.EventV2.Kind != "timer.fired" {
		t.Fatalf("EventV2.Kind = %q, want timer.fired", got.EventV2.Kind)
	}
	if got.EventV2.Source != SourceSchedule {
		t.Fatalf("EventV2.Source = %q, want %q", got.EventV2.Source, SourceSchedule)
	}
	if !got.EventV2.OccurredAt.Equal(now) {
		t.Fatalf("EventV2.OccurredAt = %v, want %v", got.EventV2.OccurredAt, now)
	}
}

func TestNormalizeTriggerPreservesExplicitSourcesAndOccurredAt(t *testing.T) {
	now := time.Date(2026, 4, 15, 13, 0, 0, 0, time.UTC)
	explicitAt := time.Date(2026, 4, 14, 7, 30, 0, 0, time.UTC)
	got := normalizeTrigger(Trigger{
		Kind: TriggerVoice,
		IntentV2: &IntentRecord{
			Action: "terminal",
			Source: SourceAgent,
		},
		EventV2: &EventRecord{
			Kind:       "sound.detected",
			Source:     SourceWebhook,
			OccurredAt: explicitAt,
		},
	}, now)

	if got.IntentV2 == nil || got.IntentV2.Source != SourceAgent {
		t.Fatalf("IntentV2.Source = %+v, want %q", got.IntentV2, SourceAgent)
	}
	if got.EventV2 == nil || got.EventV2.Source != SourceWebhook {
		t.Fatalf("EventV2.Source = %+v, want %q", got.EventV2, SourceWebhook)
	}
	if !got.EventV2.OccurredAt.Equal(explicitAt) {
		t.Fatalf("EventV2.OccurredAt = %v, want %v", got.EventV2.OccurredAt, explicitAt)
	}
}

func TestNormalizeTriggerCopiesArgumentsIntoIntentSlots(t *testing.T) {
	args := map[string]string{"device_id": "d1"}
	got := normalizeTrigger(Trigger{
		Kind:      TriggerManual,
		Intent:    "terminal",
		Arguments: args,
	}, time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC))

	if got.IntentV2 == nil {
		t.Fatalf("expected IntentV2 to be created")
	}
	if got.IntentV2.Slots == nil {
		t.Fatalf("expected IntentV2.Slots to be created")
	}
	if got.IntentV2.Slots["device_id"] != "d1" {
		t.Fatalf("IntentV2.Slots[device_id] = %q, want d1", got.IntentV2.Slots["device_id"])
	}

	// Mutating the original args map must not mutate normalized slots.
	args["device_id"] = "d2"
	if got.IntentV2.Slots["device_id"] != "d1" {
		t.Fatalf("slots mutated through shared map reference: %+v", got.IntentV2.Slots)
	}

	// Mutating slots must not mutate original args.
	got.IntentV2.Slots["device_id"] = "d3"
	if args["device_id"] != "d2" {
		t.Fatalf("arguments mutated through shared map reference: %+v", args)
	}
}

func TestNormalizeTriggerBackfillsIntentFromIntentV2Action(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:   TriggerManual,
		Intent: "   ",
		IntentV2: &IntentRecord{
			Action: "  terminal  ",
		},
	}, time.Date(2026, 4, 15, 15, 0, 0, 0, time.UTC))

	if got.IntentV2 == nil {
		t.Fatalf("expected IntentV2 to remain populated")
	}
	if got.IntentV2.Action != "terminal" {
		t.Fatalf("IntentV2.Action = %q, want terminal", got.IntentV2.Action)
	}
	if got.Intent != "terminal" {
		t.Fatalf("Intent = %q, want terminal", got.Intent)
	}
}

func TestNormalizeTriggerIntentV2SlotsFallbackFromArguments(t *testing.T) {
	args := map[string]string{"device_id": "d1"}
	got := normalizeTrigger(Trigger{
		Kind:      TriggerManual,
		Arguments: args,
		IntentV2: &IntentRecord{
			Action: "terminal",
			Slots:  nil,
		},
	}, time.Date(2026, 4, 15, 15, 5, 0, 0, time.UTC))

	if got.IntentV2 == nil || got.IntentV2.Slots == nil {
		t.Fatalf("expected IntentV2.Slots fallback from Arguments")
	}
	if got.IntentV2.Slots["device_id"] != "d1" {
		t.Fatalf("IntentV2.Slots[device_id] = %q, want d1", got.IntentV2.Slots["device_id"])
	}
	// Ensure copied map semantics are preserved here too.
	args["device_id"] = "d2"
	if got.IntentV2.Slots["device_id"] != "d1" {
		t.Fatalf("slots mutated through shared map reference: %+v", got.IntentV2.Slots)
	}
}

func TestNormalizeTriggerPreservesExplicitIntentV2Slots(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:      TriggerManual,
		Arguments: map[string]string{"device_id": "d1"},
		IntentV2: &IntentRecord{
			Action: "terminal",
			Slots:  map[string]string{"device_id": "override"},
		},
	}, time.Date(2026, 4, 15, 15, 8, 0, 0, time.UTC))

	if got.IntentV2 == nil || got.IntentV2.Slots == nil {
		t.Fatalf("expected IntentV2.Slots to be populated")
	}
	if got.IntentV2.Slots["device_id"] != "override" {
		t.Fatalf("IntentV2.Slots[device_id] = %q, want override", got.IntentV2.Slots["device_id"])
	}
}

func TestNormalizeTriggerInitializesArgumentsMapWhenNil(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:      TriggerManual,
		Intent:    "terminal",
		Arguments: nil,
	}, time.Date(2026, 4, 15, 15, 10, 0, 0, time.UTC))

	if got.Arguments == nil {
		t.Fatalf("expected Arguments map to be initialized")
	}
	if got.IntentV2 == nil || got.IntentV2.Slots == nil {
		t.Fatalf("expected IntentV2 slots to be initialized")
	}
	if len(got.Arguments) != 0 || len(got.IntentV2.Slots) != 0 {
		t.Fatalf("expected empty initialized maps, got arguments=%v slots=%v", got.Arguments, got.IntentV2.Slots)
	}
}

func TestNormalizeTriggerPrefersExplicitIntentOverIntentV2Action(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:   TriggerManual,
		Intent: "terminal",
		IntentV2: &IntentRecord{
			Action: "photo frame",
		},
	}, time.Date(2026, 4, 15, 15, 20, 0, 0, time.UTC))

	if got.Intent != "terminal" {
		t.Fatalf("Intent = %q, want terminal", got.Intent)
	}
	if got.IntentV2 == nil || got.IntentV2.Action != "photo frame" {
		t.Fatalf("IntentV2.Action = %+v, want photo frame", got.IntentV2)
	}
}

func TestNormalizeTriggerEventKindTrimDoesNotRewriteExistingIntent(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:   TriggerEvent,
		Intent: "terminal",
		EventV2: &EventRecord{
			Kind: " sound.detected ",
		},
	}, time.Date(2026, 4, 15, 15, 25, 0, 0, time.UTC))

	if got.Intent != "terminal" {
		t.Fatalf("Intent = %q, want terminal", got.Intent)
	}
	if got.EventV2 == nil || got.EventV2.Kind != "sound.detected" {
		t.Fatalf("EventV2.Kind = %+v, want sound.detected", got.EventV2)
	}
}

func TestNormalizeTriggerWithoutIntentOrEventDoesNotSynthesizeIntentV2(t *testing.T) {
	got := normalizeTrigger(Trigger{
		Kind:      TriggerManual,
		SourceID:  "  d1  ",
		Intent:    "   ",
		Arguments: nil,
	}, time.Date(2026, 4, 15, 15, 30, 0, 0, time.UTC))

	if got.SourceID != "d1" {
		t.Fatalf("SourceID = %q, want d1", got.SourceID)
	}
	if got.Intent != "" {
		t.Fatalf("Intent = %q, want empty", got.Intent)
	}
	if got.IntentV2 != nil {
		t.Fatalf("expected IntentV2 to remain nil when no intent payload is provided, got %+v", got.IntentV2)
	}
	if got.EventV2 != nil {
		t.Fatalf("expected EventV2 to remain nil, got %+v", got.EventV2)
	}
	if got.Arguments == nil {
		t.Fatalf("expected Arguments map to be initialized")
	}
}

func TestIntentEventBusPublishNormalizesBeforeFanout(t *testing.T) {
	bus := NewIntentEventBus()
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	bus.Publish(Trigger{
		Kind:     TriggerVoice,
		SourceID: " d1 ",
		Intent:   " terminal ",
		EventV2: &EventRecord{
			Kind: " voice.detected ",
		},
	})

	select {
	case got := <-ch:
		if got.SourceID != "d1" {
			t.Fatalf("SourceID = %q, want d1", got.SourceID)
		}
		if got.Intent != "terminal" {
			t.Fatalf("Intent = %q, want terminal", got.Intent)
		}
		if got.IntentV2 == nil || got.IntentV2.Source != SourceVoice {
			t.Fatalf("IntentV2 source = %+v, want %q", got.IntentV2, SourceVoice)
		}
		if got.EventV2 == nil || got.EventV2.Source != SourceVoice {
			t.Fatalf("EventV2 source = %+v, want %q", got.EventV2, SourceVoice)
		}
		if got.EventV2.OccurredAt.IsZero() {
			t.Fatalf("expected EventV2.OccurredAt to be set")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for published trigger")
	}
}

func TestIntentEventBusPublishDropsWhenListenerBufferIsFull(t *testing.T) {
	bus := NewIntentEventBus()
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	// Fill the only slot in the listener channel.
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "first"})

	done := make(chan struct{})
	go func() {
		bus.Publish(Trigger{Kind: TriggerManual, Intent: "second"})
		close(done)
	}()

	select {
	case <-done:
		// Expected: publish must never block on a full listener buffer.
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Publish blocked on full listener channel")
	}

	select {
	case got := <-ch:
		if got.Intent != "first" {
			t.Fatalf("received intent = %q, want first", got.Intent)
		}
	default:
		t.Fatalf("expected first message in channel")
	}

	select {
	case got := <-ch:
		t.Fatalf("unexpected second message delivered: %+v", got)
	default:
	}
}

func TestIntentEventBusCancelUnsubscribesAndClosesChannel(t *testing.T) {
	bus := NewIntentEventBus()
	ch, cancel := bus.Subscribe(1)

	cancel()
	cancel() // idempotent

	_, ok := <-ch
	if ok {
		t.Fatalf("expected closed channel after cancel")
	}

	// Publishing after cancel should be a no-op for the closed subscriber.
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "ignored"})
}

func TestIntentEventBusSubscribeClampsBufferToAtLeastOne(t *testing.T) {
	bus := NewIntentEventBus()
	ch, cancel := bus.Subscribe(0)
	defer cancel()

	bus.Publish(Trigger{Kind: TriggerManual, Intent: "first"})
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "second"})

	select {
	case got := <-ch:
		if got.Intent != "first" {
			t.Fatalf("received intent = %q, want first", got.Intent)
		}
	default:
		t.Fatalf("expected one buffered message")
	}

	select {
	case got := <-ch:
		t.Fatalf("unexpected second message delivered: %+v", got)
	default:
	}
}

func TestIntentEventBusPublishFansOutToMultipleSubscribers(t *testing.T) {
	bus := NewIntentEventBus()
	ch1, cancel1 := bus.Subscribe(1)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(1)
	defer cancel2()

	bus.Publish(Trigger{Kind: TriggerVoice, Intent: "terminal"})

	for i, ch := range []<-chan Trigger{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Intent != "terminal" {
				t.Fatalf("subscriber %d intent = %q, want terminal", i+1, got.Intent)
			}
			if got.IntentV2 == nil || got.IntentV2.Source != SourceVoice {
				t.Fatalf("subscriber %d intent source = %+v, want %q", i+1, got.IntentV2, SourceVoice)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timed out waiting for subscriber %d", i+1)
		}
	}
}

func TestIntentEventBusPublishToReadySubscriberWhenAnotherIsFull(t *testing.T) {
	bus := NewIntentEventBus()
	fullCh, cancelFull := bus.Subscribe(1)
	defer cancelFull()
	readyCh, cancelReady := bus.Subscribe(2)
	defer cancelReady()

	// Fill one subscriber to force drop behavior on subsequent publish.
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "first"})

	bus.Publish(Trigger{Kind: TriggerVoice, Intent: "second"})

	got1 := waitTrigger(t, readyCh, "ready subscriber first delivery")
	got2 := waitTrigger(t, readyCh, "ready subscriber second delivery")
	if got1.Intent != "first" || got2.Intent != "second" {
		t.Fatalf("ready subscriber intents = [%q, %q], want [first, second]", got1.Intent, got2.Intent)
	}

	// The full subscriber should still only have one buffered message.
	select {
	case got := <-fullCh:
		if got.Intent != "first" {
			t.Fatalf("full subscriber first intent = %q, want first", got.Intent)
		}
	default:
		t.Fatalf("expected first buffered message on full subscriber")
	}
	select {
	case got := <-fullCh:
		t.Fatalf("unexpected second message on full subscriber: %+v", got)
	default:
	}
}

func TestIntentEventBusCancelOneSubscriberKeepsOthersActive(t *testing.T) {
	bus := NewIntentEventBus()
	ch1, cancel1 := bus.Subscribe(1)
	ch2, cancel2 := bus.Subscribe(1)
	defer cancel2()

	cancel1()
	_, ok := <-ch1
	if ok {
		t.Fatalf("expected canceled subscriber channel to be closed")
	}

	bus.Publish(Trigger{Kind: TriggerManual, Intent: "terminal"})

	got := waitTrigger(t, ch2, "active subscriber delivery after peer cancel")
	if got.Intent != "terminal" {
		t.Fatalf("active subscriber intent = %q, want terminal", got.Intent)
	}
}

func TestIntentEventBusPublishStillWorksAfterPartialUnsubscribe(t *testing.T) {
	bus := NewIntentEventBus()
	ch1, cancel1 := bus.Subscribe(1)
	defer cancel1()
	_, cancel2 := bus.Subscribe(1)
	cancel2()

	bus.Publish(Trigger{Kind: TriggerVoice, Intent: "first"})
	bus.Publish(Trigger{Kind: TriggerVoice, Intent: "second"})

	got1 := waitTrigger(t, ch1, "first publish after partial unsubscribe")
	if got1.Intent != "first" {
		t.Fatalf("first intent = %q, want first", got1.Intent)
	}

	// Buffer is size 1, so a second immediate publish can be dropped; ensure no panic/block and
	// channel remains readable statefully by publishing again after draining.
	bus.Publish(Trigger{Kind: TriggerVoice, Intent: "third"})
	got3 := waitTrigger(t, ch1, "third publish after drain")
	if got3.Intent != "third" {
		t.Fatalf("third intent = %q, want third", got3.Intent)
	}
}

func TestIntentEventBusPublishNilReceiverIsSafe(_ *testing.T) {
	var bus *IntentEventBus
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "ignored"})
}

func TestIntentEventBusPublishWithoutSubscribersIsSafe(_ *testing.T) {
	bus := NewIntentEventBus()
	bus.Publish(Trigger{Kind: TriggerManual, Intent: "ignored"})
}

func waitTrigger(t *testing.T, ch <-chan Trigger, context string) Trigger {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for %s", context)
		return Trigger{}
	}
}
