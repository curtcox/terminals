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

func TestConnectFanout(t *testing.T) {
	r := iorouter.NewRouter()

	added := r.ConnectFanout("mic-a", []string{"speaker-b", "speaker-c"}, "audio")
	if added != 2 {
		t.Fatalf("ConnectFanout() added = %d, want 2", added)
	}
	// Duplicate fanout should not add existing routes.
	added = r.ConnectFanout("mic-a", []string{"speaker-b", "speaker-c"}, "audio")
	if added != 0 {
		t.Fatalf("ConnectFanout() second added = %d, want 0", added)
	}
	if r.RouteCount() != 2 {
		t.Fatalf("RouteCount() = %d, want 2", r.RouteCount())
	}
}

func TestDisconnectDeviceAndRoutesForDevice(t *testing.T) {
	r := iorouter.NewRouter()
	_ = r.Connect("a", "b", "audio")
	_ = r.Connect("a", "c", "audio")
	_ = r.Connect("d", "a", "video")
	_ = r.Connect("x", "y", "audio")

	routesA := r.RoutesForDevice("a")
	if len(routesA) != 3 {
		t.Fatalf("len(RoutesForDevice(a)) = %d, want 3", len(routesA))
	}

	removed := r.DisconnectDevice("a")
	if removed != 3 {
		t.Fatalf("DisconnectDevice(a) removed = %d, want 3", removed)
	}
	if r.RouteCount() != 1 {
		t.Fatalf("RouteCount() = %d, want 1", r.RouteCount())
	}
}
