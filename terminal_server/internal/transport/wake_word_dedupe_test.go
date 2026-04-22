package transport

import (
	"testing"
	"time"
)

func TestWakeWordDedupeStageDefaultPolicyIsFirstHeard(t *testing.T) {
	if got := defaultWakeWordWinnerPolicy(); got != wakeWordWinnerPolicyFirstHeard {
		t.Fatalf("default policy = %q, want %q", got, wakeWordWinnerPolicyFirstHeard)
	}
}

func TestWakeWordDedupeStageFirstHeardPolicyPrefersEarliest(t *testing.T) {
	base := time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC)
	events := []wakeWordCandidate{
		{DeviceID: "device-2", Spoken: "assistant lights off", HeardAt: base.Add(40 * time.Millisecond), Confidence: 0.91},
		{DeviceID: "device-1", Spoken: "assistant lights off", HeardAt: base.Add(10 * time.Millisecond), Confidence: 0.42},
	}

	winner, ok := selectWakeWordWinner(wakeWordWinnerPolicyFirstHeard, events)
	if !ok {
		t.Fatalf("winner missing for first-heard policy")
	}
	if winner.DeviceID != "device-1" {
		t.Fatalf("winner device = %q, want device-1", winner.DeviceID)
	}
}

func TestWakeWordDedupeStageHighestConfidencePolicyPrefersConfidence(t *testing.T) {
	base := time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC)
	events := []wakeWordCandidate{
		{DeviceID: "device-1", Spoken: "assistant lights off", HeardAt: base.Add(10 * time.Millisecond), Confidence: 0.65},
		{DeviceID: "device-2", Spoken: "assistant lights off", HeardAt: base.Add(40 * time.Millisecond), Confidence: 0.93},
	}

	winner, ok := selectWakeWordWinner(wakeWordWinnerPolicyHighestConfidence, events)
	if !ok {
		t.Fatalf("winner missing for highest-confidence policy")
	}
	if winner.DeviceID != "device-2" {
		t.Fatalf("winner device = %q, want device-2", winner.DeviceID)
	}
}

func TestWakeWordDedupeStageClosestTerminalPolicyPrefersNearest(t *testing.T) {
	base := time.Date(2026, 4, 22, 17, 0, 0, 0, time.UTC)
	events := []wakeWordCandidate{
		{DeviceID: "device-1", Spoken: "assistant lights off", HeardAt: base.Add(10 * time.Millisecond), Confidence: 0.88, DistanceMeters: 3.2},
		{DeviceID: "device-2", Spoken: "assistant lights off", HeardAt: base.Add(30 * time.Millisecond), Confidence: 0.77, DistanceMeters: 1.1},
	}

	winner, ok := selectWakeWordWinner(wakeWordWinnerPolicyClosestTerminal, events)
	if !ok {
		t.Fatalf("winner missing for closest-terminal policy")
	}
	if winner.DeviceID != "device-2" {
		t.Fatalf("winner device = %q, want device-2", winner.DeviceID)
	}
}
