package world

import (
	"context"
	"testing"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func TestModelLocateAndVerifyDevice(t *testing.T) {
	model := NewModel()
	model.UpsertGeometry(context.Background(), DeviceGeometry{
		DeviceID:          "kitchen-tablet",
		Zone:              "kitchen",
		MicArray:          &MicArrayGeometry{SpacingM: 0.12, Orientation: "landscape"},
		CameraIntrinsics:  &CameraIntrinsics{FocalLengthX: 1024, FocalLengthY: 1022},
		CameraExtrinsics:  &iorouter.Pose{X: 1.2, Y: 0.4, Z: 1.5, Yaw: 90},
		RadioBias:         map[string]float64{"ble": -2.4},
		VerificationState: VerificationManual,
	})
	model.UpsertEntity(context.Background(), EntityRecord{
		EntityID:    "alice",
		Kind:        EntityPerson,
		DisplayName: "Alice",
		Confidence:  0.92,
		LastSeenAt:  time.Now().UTC(),
		LastKnown: &iorouter.LocationEstimate{
			Zone:       "kitchen",
			Confidence: 0.92,
		},
	})

	location, err := model.LocateEntity(context.Background(), EntityQuery{Person: "alice", MinConfidence: 0.5})
	if err != nil {
		t.Fatalf("LocateEntity() error = %v", err)
	}
	if location.Zone != "kitchen" {
		t.Fatalf("zone = %q, want kitchen", location.Zone)
	}

	if err := model.VerifyDevice(context.Background(), "kitchen-tablet", "marker"); err != nil {
		t.Fatalf("VerifyDevice() error = %v", err)
	}
	geometry, ok := model.Geometry(context.Background(), "kitchen-tablet")
	if !ok {
		t.Fatalf("expected geometry for kitchen-tablet")
	}
	if geometry.VerificationState != VerificationMarker {
		t.Fatalf("verification state = %q, want marker", geometry.VerificationState)
	}
	if geometry.MicArray == nil || geometry.MicArray.SpacingM <= 0 {
		t.Fatalf("expected mic array geometry to be retained")
	}
	if geometry.CameraIntrinsics == nil || geometry.CameraIntrinsics.FocalLengthX <= 0 {
		t.Fatalf("expected camera intrinsics to be retained")
	}
	if geometry.CameraExtrinsics == nil || geometry.CameraExtrinsics.Z <= 0 {
		t.Fatalf("expected camera extrinsics to be retained")
	}
	if geometry.RadioBias["ble"] != -2.4 {
		t.Fatalf("radio bias ble = %v, want -2.4", geometry.RadioBias["ble"])
	}

	history, err := model.CalibrationHistory(context.Background(), "kitchen-tablet", 5)
	if err != nil {
		t.Fatalf("CalibrationHistory() error = %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].VerificationState != VerificationMarker {
		t.Fatalf("history verification state = %q, want marker", history[0].VerificationState)
	}
}

func TestModelRecentObservations(t *testing.T) {
	now := time.Now().UTC()
	model := NewModel()

	model.AddObservation(context.Background(), iorouter.Observation{
		Kind:       "sound.classifier",
		Zone:       "kitchen",
		Subject:    "dishwasher",
		OccurredAt: now,
	})
	model.AddObservation(context.Background(), iorouter.Observation{
		Kind:       "bluetooth",
		Zone:       "garage",
		Subject:    "headphones",
		OccurredAt: now.Add(-2 * time.Minute),
	})

	observations, err := model.RecentObservations(context.Background(), "kitchen", "sound", now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("RecentObservations() error = %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len(observations) = %d, want 1", len(observations))
	}
	if observations[0].Subject != "dishwasher" {
		t.Fatalf("subject = %q, want dishwasher", observations[0].Subject)
	}
}

func TestModelCalibrationHistoryRespectsLimit(t *testing.T) {
	model := NewModel()
	model.UpsertGeometry(context.Background(), DeviceGeometry{DeviceID: "hall-tablet"})

	if err := model.VerifyDevice(context.Background(), "hall-tablet", "manual"); err != nil {
		t.Fatalf("VerifyDevice(manual) error = %v", err)
	}
	if err := model.VerifyDevice(context.Background(), "hall-tablet", "marker"); err != nil {
		t.Fatalf("VerifyDevice(marker) error = %v", err)
	}
	if err := model.VerifyDevice(context.Background(), "hall-tablet", "audio_chirp"); err != nil {
		t.Fatalf("VerifyDevice(audio_chirp) error = %v", err)
	}

	history, err := model.CalibrationHistory(context.Background(), "hall-tablet", 2)
	if err != nil {
		t.Fatalf("CalibrationHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[0].VerificationState != VerificationMarker {
		t.Fatalf("history[0] = %q, want marker", history[0].VerificationState)
	}
	if history[1].VerificationState != VerificationAudioChirp {
		t.Fatalf("history[1] = %q, want audio_chirp", history[1].VerificationState)
	}
}

func TestModelListGeometriesSortedAndIsolated(t *testing.T) {
	model := NewModel()
	model.UpsertGeometry(context.Background(), DeviceGeometry{DeviceID: "zeta", RadioBias: map[string]float64{"ble": -4.0}})
	model.UpsertGeometry(context.Background(), DeviceGeometry{DeviceID: "alpha", RadioBias: map[string]float64{"ble": -1.0}})

	list := model.ListGeometries(context.Background())
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if list[0].DeviceID != "alpha" || list[1].DeviceID != "zeta" {
		t.Fatalf("device order = [%q, %q], want [alpha, zeta]", list[0].DeviceID, list[1].DeviceID)
	}

	list[0].RadioBias["ble"] = -99
	geometry, ok := model.Geometry(context.Background(), "alpha")
	if !ok {
		t.Fatalf("expected alpha geometry")
	}
	if geometry.RadioBias["ble"] != -1.0 {
		t.Fatalf("stored bias mutated to %v, want -1.0", geometry.RadioBias["ble"])
	}
}
