package io_test

import "testing"

import iorouter "github.com/curtcox/terminals/terminal_server/internal/io"

func TestConnectDisconnect(t *testing.T) {
	r := iorouter.NewRouter()
	if err := r.Connect("mic-a", "speaker-b", "audio"); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := r.Connect("mic-a", "speaker-b", "audio"); err != iorouter.ErrRouteExists {
		t.Fatalf("Connect() duplicate error = %v, want %v", err, iorouter.ErrRouteExists)
	}
	if err := r.Disconnect("mic-a", "speaker-b", "audio"); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if err := r.Disconnect("mic-a", "speaker-b", "audio"); err != iorouter.ErrRouteNotFound {
		t.Fatalf("Disconnect() missing error = %v, want %v", err, iorouter.ErrRouteNotFound)
	}
}
