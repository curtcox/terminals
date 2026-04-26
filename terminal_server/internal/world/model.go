// Package world stores calibration data and fused world-model entities.
package world

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// VerificationState captures how a device pose was verified.
type VerificationState string

const (
	// VerificationUnknown means no verification method has been recorded yet.
	VerificationUnknown VerificationState = "unknown"
	// VerificationManual indicates a user manually confirmed the location.
	VerificationManual VerificationState = "manual"
	// VerificationMarker indicates marker-based visual verification.
	VerificationMarker VerificationState = "marker"
	// VerificationAudioChirp indicates calibration by emitted/recorded chirps.
	VerificationAudioChirp VerificationState = "audio_chirp"
	// VerificationRFFingerprint indicates RF-based verification.
	VerificationRFFingerprint VerificationState = "rf_fingerprint"
	// VerificationMixed indicates multiple verification methods were combined.
	VerificationMixed VerificationState = "mixed"
)

// DeviceGeometry tracks calibrated device placement and sensor metadata.
type DeviceGeometry struct {
	DeviceID          string
	Zone              string
	Pose              iorouter.Pose
	MicArray          *MicArrayGeometry
	CameraIntrinsics  *CameraIntrinsics
	CameraExtrinsics  *iorouter.Pose
	RadioBias         map[string]float64
	ClockSyncErrorMS  float64
	VerificationState VerificationState
	CalibrationTag    string
	UpdatedAt         time.Time
}

// MicArrayGeometry describes an optional fixed microphone array layout.
type MicArrayGeometry struct {
	ElementPoses []iorouter.Pose
	SpacingM     float64
	Orientation  string
}

// CameraIntrinsics captures lens/model parameters for localization.
type CameraIntrinsics struct {
	FocalLengthX float64
	FocalLengthY float64
	PrincipalX   float64
	PrincipalY   float64
	Distortion   []float64
}

// CalibrationEvent records one verification update for a fixed device.
type CalibrationEvent struct {
	DeviceID          string
	Method            string
	VerificationState VerificationState
	CalibrationTag    string
	OccurredAt        time.Time
}

// EntityKind describes tracked entities.
type EntityKind string

const (
	// EntityPerson tracks people presence/location.
	EntityPerson EntityKind = "person"
	// EntityObject tracks household objects.
	EntityObject EntityKind = "object"
	// EntityBluetooth tracks Bluetooth devices.
	EntityBluetooth EntityKind = "bluetooth_device"
)

// EntityRecord tracks one world-model entity.
type EntityRecord struct {
	EntityID    string
	Kind        EntityKind
	DisplayName string
	StableAttrs map[string]string
	LastKnown   *iorouter.LocationEstimate
	LastSeenAt  time.Time
	Confidence  float64
}

// EntityQuery filters world-model lookup operations.
type EntityQuery struct {
	Person        string
	Object        string
	BluetoothMAC  string
	LastKnownOnly bool
	MinConfidence float64
}

// ErrNotFound indicates no matching world-model record.
var ErrNotFound = errors.New("world model record not found")

// Model is an in-memory world model with calibration and entity records.
type Model struct {
	mu              sync.RWMutex
	geometries      map[string]DeviceGeometry
	calibration     map[string][]CalibrationEvent
	entities        map[string]EntityRecord
	observations    []iorouter.Observation
	maxObservations int
}

// NewModel returns an empty world model.
func NewModel() *Model {
	return &Model{
		geometries:      make(map[string]DeviceGeometry),
		calibration:     make(map[string][]CalibrationEvent),
		entities:        make(map[string]EntityRecord),
		maxObservations: 2048,
	}
}

// UpsertGeometry stores or updates one device geometry record.
func (m *Model) UpsertGeometry(_ context.Context, geometry DeviceGeometry) {
	if m == nil || strings.TrimSpace(geometry.DeviceID) == "" {
		return
	}
	if geometry.UpdatedAt.IsZero() {
		geometry.UpdatedAt = time.Now().UTC()
	}
	if geometry.VerificationState == "" {
		geometry.VerificationState = VerificationUnknown
	}
	if geometry.RadioBias == nil {
		geometry.RadioBias = map[string]float64{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.geometries[geometry.DeviceID] = cloneDeviceGeometry(geometry)
}

// Geometry returns one geometry record if available.
func (m *Model) Geometry(_ context.Context, deviceID string) (DeviceGeometry, bool) {
	if m == nil {
		return DeviceGeometry{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	geometry, ok := m.geometries[strings.TrimSpace(deviceID)]
	if ok {
		geometry = cloneDeviceGeometry(geometry)
	}
	return geometry, ok
}

// ListGeometries returns all known device geometry records sorted by device ID.
func (m *Model) ListGeometries(_ context.Context) []DeviceGeometry {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]DeviceGeometry, 0, len(m.geometries))
	for _, geometry := range m.geometries {
		out = append(out, cloneDeviceGeometry(geometry))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DeviceID < out[j].DeviceID
	})
	return out
}

// UpsertEntity stores or updates an entity record.
func (m *Model) UpsertEntity(_ context.Context, entity EntityRecord) {
	if m == nil || strings.TrimSpace(entity.EntityID) == "" {
		return
	}
	if entity.StableAttrs == nil {
		entity.StableAttrs = map[string]string{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entities[entity.EntityID] = entity
}

// AddObservation stores one recent observation for world-model queries.
func (m *Model) AddObservation(_ context.Context, observation iorouter.Observation) {
	if m == nil {
		return
	}
	if observation.OccurredAt.IsZero() {
		observation.OccurredAt = time.Now().UTC()
	}
	if observation.Attributes == nil {
		observation.Attributes = map[string]string{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.observations = append(m.observations, observation)
	if over := len(m.observations) - m.maxObservations; over > 0 {
		m.observations = append([]iorouter.Observation(nil), m.observations[over:]...)
	}
}

// RecentObservations returns observations filtered by zone/kind/since.
func (m *Model) RecentObservations(_ context.Context, zone string, kind string, since time.Time) ([]iorouter.Observation, error) {
	if m == nil {
		return nil, nil
	}
	zone = strings.TrimSpace(strings.ToLower(zone))
	kind = strings.TrimSpace(strings.ToLower(kind))

	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]iorouter.Observation, 0, len(m.observations))
	for i := len(m.observations) - 1; i >= 0; i-- {
		ob := m.observations[i]
		if !since.IsZero() && ob.OccurredAt.Before(since) {
			continue
		}
		if zone != "" && strings.ToLower(strings.TrimSpace(ob.Zone)) != zone {
			continue
		}
		if kind != "" && !strings.Contains(strings.ToLower(ob.Kind), kind) {
			continue
		}
		out = append(out, ob)
	}
	return out, nil
}

// LocateEntity resolves an entity to its best-known location.
func (m *Model) LocateEntity(_ context.Context, query EntityQuery) (*iorouter.LocationEstimate, error) {
	if m == nil {
		return nil, ErrNotFound
	}
	target := strings.TrimSpace(strings.ToLower(query.Person))
	if target == "" {
		target = strings.TrimSpace(strings.ToLower(query.Object))
	}
	if target == "" {
		target = strings.TrimSpace(strings.ToLower(query.BluetoothMAC))
	}
	if target == "" {
		return nil, ErrNotFound
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, entity := range m.entities {
		if strings.ToLower(strings.TrimSpace(entity.EntityID)) != target &&
			strings.ToLower(strings.TrimSpace(entity.DisplayName)) != target {
			continue
		}
		if entity.Confidence < query.MinConfidence || entity.LastKnown == nil {
			continue
		}
		location := *entity.LastKnown
		return &location, nil
	}
	return nil, ErrNotFound
}

// WhoIsHome returns person entities with confidence over zero.
func (m *Model) WhoIsHome(_ context.Context) ([]EntityRecord, error) {
	if m == nil {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]EntityRecord, 0, len(m.entities))
	for _, entity := range m.entities {
		if entity.Kind != EntityPerson || entity.Confidence <= 0 {
			continue
		}
		out = append(out, entity)
	}
	return out, nil
}

// VerifyDevice updates a device's verification state and calibration tag.
func (m *Model) VerifyDevice(ctx context.Context, deviceID string, method string) error {
	if m == nil {
		return ErrNotFound
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ErrNotFound
	}
	state := verificationStateFromMethod(method)

	m.mu.Lock()
	defer m.mu.Unlock()
	geometry, ok := m.geometries[deviceID]
	if !ok {
		return ErrNotFound
	}
	geometry.VerificationState = state
	geometry.CalibrationTag = strings.TrimSpace(method)
	geometry.UpdatedAt = time.Now().UTC()
	m.geometries[deviceID] = geometry
	m.calibration[deviceID] = append(m.calibration[deviceID], CalibrationEvent{
		DeviceID:          deviceID,
		Method:            strings.TrimSpace(method),
		VerificationState: state,
		CalibrationTag:    geometry.CalibrationTag,
		OccurredAt:        geometry.UpdatedAt,
	})
	_ = ctx
	return nil
}

// CalibrationHistory returns recent verification events for one device.
func (m *Model) CalibrationHistory(_ context.Context, deviceID string, limit int) ([]CalibrationEvent, error) {
	if m == nil {
		return nil, ErrNotFound
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, ErrNotFound
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	events, ok := m.calibration[deviceID]
	if !ok {
		return nil, ErrNotFound
	}

	start := 0
	if limit > 0 && len(events) > limit {
		start = len(events) - limit
	}
	out := append([]CalibrationEvent(nil), events[start:]...)
	return out, nil
}

func verificationStateFromMethod(method string) VerificationState {
	switch strings.TrimSpace(strings.ToLower(method)) {
	case "manual":
		return VerificationManual
	case "marker":
		return VerificationMarker
	case "audio_chirp", "chirp":
		return VerificationAudioChirp
	case "rf", "rf_fingerprint":
		return VerificationRFFingerprint
	case "mixed":
		return VerificationMixed
	default:
		return VerificationUnknown
	}
}

func cloneDeviceGeometry(in DeviceGeometry) DeviceGeometry {
	out := in
	if len(in.RadioBias) == 0 {
		out.RadioBias = map[string]float64{}
		return out
	}
	bias := make(map[string]float64, len(in.RadioBias))
	for key, value := range in.RadioBias {
		bias[key] = value
	}
	out.RadioBias = bias
	return out
}
