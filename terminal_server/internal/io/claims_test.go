package io_test

import (
	"context"
	"testing"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func TestClaimManagerSharedAndExclusive(t *testing.T) {
	manager := iorouter.NewClaimManager()

	_, err := manager.Request(context.Background(), []iorouter.Claim{
		{ActivationID: "a1", DeviceID: "d1", Resource: "mic.analyze", Mode: iorouter.ClaimShared, Priority: 1},
		{ActivationID: "a2", DeviceID: "d1", Resource: "mic.analyze", Mode: iorouter.ClaimShared, Priority: 1},
	})
	if err != nil {
		t.Fatalf("Request(shared) error = %v", err)
	}

	if _, err := manager.Request(context.Background(), []iorouter.Claim{
		{ActivationID: "a3", DeviceID: "d1", Resource: "mic.analyze", Mode: iorouter.ClaimExclusive, Priority: 1},
	}); err != iorouter.ErrClaimConflict {
		t.Fatalf("exclusive claim conflict error = %v, want %v", err, iorouter.ErrClaimConflict)
	}
}

func TestClaimManagerPreemptAndRestore(t *testing.T) {
	manager := iorouter.NewClaimManager()
	_, _ = manager.Request(context.Background(), []iorouter.Claim{
		{ActivationID: "photo", DeviceID: "d1", Resource: "speaker.main", Mode: iorouter.ClaimExclusive, Priority: 1},
	})

	grant, err := manager.Request(context.Background(), []iorouter.Claim{
		{ActivationID: "pa", DeviceID: "d1", Resource: "speaker.main", Mode: iorouter.ClaimExclusive, Priority: 5},
	})
	if err != nil {
		t.Fatalf("Request(pa) error = %v", err)
	}
	if len(grant.Preempted) != 1 || grant.Preempted[0].ActivationID != "photo" {
		t.Fatalf("preempted = %+v, want photo", grant.Preempted)
	}

	if err := manager.Release(context.Background(), "pa"); err != nil {
		t.Fatalf("Release(pa) error = %v", err)
	}
	active := manager.Snapshot("d1")
	if len(active) != 1 || active[0].ActivationID != "photo" {
		t.Fatalf("active after restore = %+v, want photo restored", active)
	}
}

func TestClaimManagerComputeAndBufferClaims(t *testing.T) {
	manager := iorouter.NewClaimManager()

	_, err := manager.Request(context.Background(), []iorouter.Claim{
		{
			ActivationID: "sound-localize",
			DeviceID:     "edge-1",
			Resource:     iorouter.ResourceComputeCPUShared,
			Mode:         iorouter.ClaimShared,
			Priority:     2,
		},
		{
			ActivationID: "sound-localize",
			DeviceID:     "edge-1",
			Resource:     iorouter.ResourceBufferAudio,
			Mode:         iorouter.ClaimShared,
			Priority:     2,
		},
		{
			ActivationID: "ble-inventory",
			DeviceID:     "edge-1",
			Resource:     iorouter.ResourceRadioBLEScan,
			Mode:         iorouter.ClaimShared,
			Priority:     1,
		},
	})
	if err != nil {
		t.Fatalf("Request(edge resources) error = %v", err)
	}

	if _, err := manager.Request(context.Background(), []iorouter.Claim{{
		ActivationID: "exclusive-cpu",
		DeviceID:     "edge-1",
		Resource:     iorouter.ResourceComputeCPUShared,
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != iorouter.ErrClaimConflict {
		t.Fatalf("exclusive compute conflict error = %v, want %v", err, iorouter.ErrClaimConflict)
	}
}
