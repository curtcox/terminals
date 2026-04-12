package io

import "testing"

func TestConnectDisconnect(t *testing.T) {
	r := NewRouter()
	if err := r.Connect("mic-a", "speaker-b", "audio"); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := r.Connect("mic-a", "speaker-b", "audio"); err != ErrRouteExists {
		t.Fatalf("Connect() duplicate error = %v, want %v", err, ErrRouteExists)
	}
	if err := r.Disconnect("mic-a", "speaker-b", "audio"); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if err := r.Disconnect("mic-a", "speaker-b", "audio"); err != ErrRouteNotFound {
		t.Fatalf("Disconnect() missing error = %v, want %v", err, ErrRouteNotFound)
	}
}
