# World Model and Calibration
See [masterplan.md](../masterplan.md) for overall system context. The existing [placement plan](placement.md) already gives devices zones and roles. This document extends that world model so the system can localize sounds, people, objects, radios, and terminals.

## Design Principle
Semantic placement still matters most. Users speak in room and role language. The spatial model exists to support:
- better placement decisions
- sound and object localization
- presence fusion
- verification of terminal location reports
- evidence quality and confidence tracking

The client still does **not** self-assign its zone. The server owns semantic placement metadata.

## Spatial Frames
The server maintains one house-wide world frame plus optional per-zone local frames.

```go
type Pose struct {
    X          float64
    Y          float64
    Z          float64
    Yaw        float64
    Pitch      float64
    Roll       float64
    Confidence float64
}

type DeviceGeometry struct {
    DeviceID          string
    Pose              Pose
    MicArray          *MicArrayGeometry
    CameraIntrinsics  *CameraIntrinsics
    CameraExtrinsics  *Pose
    RadioBias         map[string]float64
    ClockSyncErrorMS  float64
    VerificationState VerificationState
}
```

This geometry is optional but strongly preferred for localization-heavy scenarios.

## Verification State
Each fixed device carries a verification state.

```go
type VerificationState string

const (
    VerificationUnknown   VerificationState = "unknown"
    VerificationManual    VerificationState = "manual"
    VerificationMarker    VerificationState = "marker"
    VerificationAudioChirp VerificationState = "audio_chirp"
    VerificationRFFingerprint VerificationState = "rf_fingerprint"
    VerificationMixed     VerificationState = "mixed"
)
```

Verification state captures **how** the server believes the device pose is correct.

## Entity Records
The world model tracks more than devices.

```go
type EntityKind string

const (
    EntityPerson    EntityKind = "person"
    EntityObject    EntityKind = "object"
    EntityBluetooth EntityKind = "bluetooth_device"
)

type EntityRecord struct {
    EntityID        string
    Kind            EntityKind
    DisplayName     string
    StableAttrs     map[string]any
    LastKnown       *LocationEstimate
    LastSeenAt      time.Time
    Confidence      float64
}

type LocationEstimate struct {
    Zone       string
    Pose       *Pose
    RadiusM    float64
    Confidence float64
    Sources    []string
}
```

Examples:
- person: `Alice`
- object: `car_keys`
- bluetooth device: `AirPods Pro`, `watch`, `unknown_beacon`

## Presence Fusion
A person's location is usually not produced by one sensor alone. The world model fuses:
- camera detections and tracks
- device affinity and active UI interactions
- BLE sightings
- motion or sound observations near a device
- schedule context

The output is one best-effort presence graph that scenarios can query.

## Calibration Workflows
The admin UI should support several calibration methods for fixed terminals.

### Manual Placement
Admin chooses a zone, clicks a floor-plan position, and optionally enters facing direction.

### Marker Verification
The admin shows the camera a known visual marker. The server estimates camera pose and verifies or adjusts placement.

### Audio Chirp Verification
The server plays a known chirp on one or more speakers while nearby microphones timestamp the arrival. Multi-device timing plus existing poses verify a target device location.

### RF Fingerprint Verification
The server compares observed BLE/Wi-Fi fingerprints against previous baselines for a fixed terminal.

No one method is required. Verification confidence improves as more methods agree.

## Placement Queries Extended
Keep the existing [placement engine](placement.md) APIs, but add queries for people, objects, and location confidence.

```go
type EntityQuery struct {
    Person        string
    Object        string
    BluetoothMAC  string
    LastKnownOnly bool
    MinConfidence float64
}

type WorldModel interface {
    LocateEntity(ctx context.Context, q EntityQuery) (*LocationEstimate, error)
    WhoIsHome(ctx context.Context) ([]EntityRecord, error)
    VerifyDevice(ctx context.Context, deviceID string, method string) error
    RecentObservations(ctx context.Context, zone string, kind string, since time.Time) ([]Observation, error)
}
```

Examples:
- "Who is in the house and where are they?"
- "Where are my keys?"
- "What Bluetooth devices are in the house right now?"
- "Verify the kitchen tablet location."

## Administrative Views
The server-driven admin UI should include:
- floor-plan view with terminal poses and confidence rings
- camera and microphone geometry view
- person/object last-seen table
- Bluetooth inventory by zone and strength
- calibration history and verification status per device

## Confidence and Provenance
Any derived location must preserve:
- contributing devices
- calibration version
- last verification method
- timing quality
- confidence score

This is required for trust and debugging. "The system thinks the sound came from the hallway" must be explainable.

## Related Plans
- [placement.md](placement.md) — Base semantic placement model.
- [edge-execution.md](edge-execution.md) — Edge capability fields for timing and geometry.
- [observation-plane.md](observation-plane.md) — `LocationEstimate` and observation provenance.
- [sensing-use-case-flows.md](sensing-use-case-flows.md) — Flows that depend on spatial confidence.
