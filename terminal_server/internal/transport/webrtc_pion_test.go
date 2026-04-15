package transport

import (
	"context"
	"strings"
	"testing"
)

func TestParseSDPPayload(t *testing.T) {
	raw := "v=0\r\no=- 1 2 IN IP4 127.0.0.1"
	gotRaw, err := parseSDPPayload(raw)
	if err != nil {
		t.Fatalf("parseSDPPayload(raw) error = %v", err)
	}
	if gotRaw != raw {
		t.Fatalf("parseSDPPayload(raw) = %q, want %q", gotRaw, raw)
	}

	gotJSON, err := parseSDPPayload(`{"sdp":"v=0\\r\\na=group:BUNDLE 0"}`)
	if err != nil {
		t.Fatalf("parseSDPPayload(json) error = %v", err)
	}
	if !strings.Contains(gotJSON, "a=group:BUNDLE") {
		t.Fatalf("parseSDPPayload(json) = %q, want sdp body", gotJSON)
	}
}

func TestParseCandidatePayload(t *testing.T) {
	candidate, err := parseCandidatePayload(`{"candidate":"candidate:1 1 udp 2122252543 192.0.2.1 54321 typ host"}`)
	if err != nil {
		t.Fatalf("parseCandidatePayload(json) error = %v", err)
	}
	if !strings.HasPrefix(candidate.Candidate, "candidate:") {
		t.Fatalf("candidate.Candidate = %q, want candidate prefix", candidate.Candidate)
	}

	raw, err := parseCandidatePayload("candidate:2 1 udp 2122252543 192.0.2.2 54322 typ host")
	if err != nil {
		t.Fatalf("parseCandidatePayload(raw) error = %v", err)
	}
	if !strings.HasPrefix(raw.Candidate, "candidate:") {
		t.Fatalf("raw.Candidate = %q, want candidate prefix", raw.Candidate)
	}
}

func TestPionWebRTCSignalEngineRejectsInvalidOfferPayload(t *testing.T) {
	engine, err := NewPionWebRTCSignalEngine()
	if err != nil {
		t.Fatalf("NewPionWebRTCSignalEngine() error = %v", err)
	}

	_, err = engine.HandleSignal(context.Background(), WebRTCSignalEngineRequest{
		StreamID: "route:d1|d2|audio",
		DeviceID: "d1",
		Signal: WebRTCSignalRequest{
			StreamID:   "route:d1|d2|audio",
			SignalType: "offer",
			Payload:    `{"sdp":""}`,
		},
	})
	if err == nil {
		t.Fatalf("HandleSignal(offer) expected error for empty sdp payload")
	}
}

func TestPionWebRTCSignalEngineRemoveStream(t *testing.T) {
	engine, err := NewPionWebRTCSignalEngine()
	if err != nil {
		t.Fatalf("NewPionWebRTCSignalEngine() error = %v", err)
	}

	engine.RemoveStream("route:a|b|audio")
	engine.RemoveStream("route:a|b|audio")
}
