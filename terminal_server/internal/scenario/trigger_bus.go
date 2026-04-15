package scenario

import (
	"strings"
	"sync"
	"time"
)

// IntentEventBus is a lightweight in-process pub/sub bus for typed triggers.
type IntentEventBus struct {
	mu        sync.RWMutex
	nextID    uint64
	listeners map[uint64]chan Trigger
}

// NewIntentEventBus constructs a ready-to-use trigger bus.
func NewIntentEventBus() *IntentEventBus {
	return &IntentEventBus{
		listeners: make(map[uint64]chan Trigger),
	}
}

// Subscribe registers a trigger listener with the requested buffer size.
// The returned cancel function unsubscribes and closes the channel.
func (b *IntentEventBus) Subscribe(buffer int) (<-chan Trigger, func()) {
	if buffer < 1 {
		buffer = 1
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	ch := make(chan Trigger, buffer)
	b.listeners[id] = ch
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		listener, ok := b.listeners[id]
		if !ok {
			return
		}
		delete(b.listeners, id)
		close(listener)
	}
}

// Publish fan-outs one normalized trigger to all listeners.
func (b *IntentEventBus) Publish(trigger Trigger) {
	if b == nil {
		return
	}
	trigger = normalizeTrigger(trigger, time.Now().UTC())

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.listeners {
		select {
		case ch <- trigger:
		default:
			// Drop when listener buffers are full to avoid backpressure.
		}
	}
}

func normalizeTrigger(trigger Trigger, now time.Time) Trigger {
	trigger.SourceID = strings.TrimSpace(trigger.SourceID)
	trigger.Intent = strings.TrimSpace(trigger.Intent)
	if trigger.Arguments == nil {
		trigger.Arguments = map[string]string{}
	}
	if trigger.IntentV2 != nil {
		trigger.IntentV2.Action = strings.TrimSpace(trigger.IntentV2.Action)
		if trigger.Intent == "" && trigger.IntentV2.Action != "" {
			trigger.Intent = trigger.IntentV2.Action
		}
		if trigger.IntentV2.Slots == nil {
			trigger.IntentV2.Slots = copyStringMap(trigger.Arguments)
		} else {
			trigger.IntentV2.Slots = copyStringMap(trigger.IntentV2.Slots)
		}
		if trigger.IntentV2.Source == "" {
			trigger.IntentV2.Source = sourceFromKind(trigger.Kind)
		}
	} else if trigger.Intent != "" {
		trigger.IntentV2 = &IntentRecord{
			Action:  trigger.Intent,
			Slots:   copyStringMap(trigger.Arguments),
			Source:  sourceFromKind(trigger.Kind),
			RawText: trigger.Intent,
		}
	}
	if trigger.EventV2 != nil {
		trigger.EventV2.Kind = strings.TrimSpace(trigger.EventV2.Kind)
		trigger.EventV2.Attributes = copyStringMap(trigger.EventV2.Attributes)
		if trigger.EventV2.OccurredAt.IsZero() {
			trigger.EventV2.OccurredAt = now
		}
		if trigger.EventV2.Source == "" {
			trigger.EventV2.Source = sourceFromKind(trigger.Kind)
		}
	}
	return trigger
}

func sourceFromKind(kind TriggerKind) TriggerSource {
	switch kind {
	case TriggerVoice:
		return SourceVoice
	case TriggerSchedule:
		return SourceSchedule
	case TriggerEvent:
		return SourceEvent
	case TriggerCascade:
		return SourceCascade
	case TriggerManual:
		return SourceManual
	default:
		return SourceManual
	}
}
