package scenario

import (
	"context"
	"image"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// TriggerKind identifies how a scenario was requested.
type TriggerKind string

// Trigger kind constants identify how a scenario was activated.
const (
	// TriggerVoice indicates a spoken command initiated the scenario.
	TriggerVoice TriggerKind = "voice"
	// TriggerSchedule indicates a scheduled timer/reminder initiated the scenario.
	TriggerSchedule TriggerKind = "schedule"
	// TriggerEvent indicates an external event initiated the scenario.
	TriggerEvent TriggerKind = "event"
	// TriggerManual indicates a direct manual request initiated the scenario.
	TriggerManual TriggerKind = "manual"
	// TriggerCascade indicates another scenario initiated this scenario.
	TriggerCascade TriggerKind = "cascade"
)

// TriggerSource captures provenance for typed intents/events.
type TriggerSource string

const (
	// SourceVoice indicates a voice-produced intent/event.
	SourceVoice TriggerSource = "voice"
	// SourceUI indicates a UI-produced intent/event.
	SourceUI TriggerSource = "ui"
	// SourceSchedule indicates a scheduler-produced intent/event.
	SourceSchedule TriggerSource = "schedule"
	// SourceEvent indicates an analyzer or system event-produced record.
	SourceEvent TriggerSource = "event"
	// SourceCascade indicates a scenario-cascade-produced record.
	SourceCascade TriggerSource = "cascade"
	// SourceAgent indicates an automation agent-produced intent.
	SourceAgent TriggerSource = "agent"
	// SourceWebhook indicates a webhook-produced intent.
	SourceWebhook TriggerSource = "webhook"
	// SourceManual indicates a manually requested intent.
	SourceManual TriggerSource = "manual"
)

// Trigger contains routing metadata used for scenario matching.
type Trigger struct {
	Kind      TriggerKind
	SourceID  string
	Intent    string
	Arguments map[string]string
	IntentV2  *IntentRecord
	EventV2   *EventRecord
}

// IntentRecord is the typed trigger intent payload.
type IntentRecord struct {
	Action     string
	Object     string
	Slots      map[string]string
	Scope      TargetScope
	Confidence float64
	RawText    string
	Source     TriggerSource
}

// EventRecord is the typed trigger event payload.
type EventRecord struct {
	Kind       string
	Subject    string
	Attributes map[string]string
	Source     TriggerSource
	OccurredAt time.Time
}

// DeviceRef points at a concrete target device selected by placement.
type DeviceRef struct {
	DeviceID string
}

// TargetScope describes semantic targeting preferences.
type TargetScope struct {
	DeviceID  string
	Zone      string
	Role      string
	Nearest   bool
	Source    DeviceRef
	Broadcast bool
}

// PlacementQuery resolves semantic scope to concrete device targets.
type PlacementQuery struct {
	Scope         TargetScope
	RequiredCaps  []string
	PreferredCaps []string
	ExcludeBusy   bool
	Count         int
}

// PlacementEngine resolves semantic target scopes to concrete devices.
type PlacementEngine interface {
	Find(ctx context.Context, q PlacementQuery) ([]DeviceRef, error)
	NearestWith(ctx context.Context, source DeviceRef, capability string) (DeviceRef, error)
	DevicesInZone(ctx context.Context, zone string) ([]DeviceRef, error)
	DevicesWithRole(ctx context.Context, role string) ([]DeviceRef, error)
}

// ActivationRequest is the normalized request passed to definitions to
// determine match + activation construction.
type ActivationRequest struct {
	Trigger     Trigger
	Targets     []DeviceRef
	RequestedAt time.Time
}

// DeviceManager exposes device selection and command capabilities.
type DeviceManager interface {
	ListDeviceIDs() []string
}

// IORouter exposes stream-routing capability required by scenarios.
type IORouter interface {
	Connect(sourceID, targetID, streamKind string) error
	Disconnect(sourceID, targetID, streamKind string) error
	RouteCount() int
}

// AIBackend represents the legacy scenario-accessible AI service. It is
// retained while scenarios migrate to the capability-specific interfaces
// (LLM, SpeechToText, etc.) defined alongside it.
type AIBackend interface {
	Query(ctx context.Context, input string) (string, error)
}

// LLMMessage is a single LLM conversation entry. Mirrors ai.Message but
// kept in the scenario package so scenarios do not depend on the ai
// package's concrete types.
type LLMMessage struct {
	Role    string
	Content string
}

// LLMOptions configures a scenario-issued large-language-model query.
type LLMOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
}

// LLMResponse is the result of an LLM query as seen by scenarios.
type LLMResponse struct {
	Text         string
	FinishReason string
}

// LLM exposes a chat-style large language model to scenarios.
type LLM interface {
	Query(ctx context.Context, messages []LLMMessage, opts LLMOptions) (*LLMResponse, error)
}

// SpeechToText converts streaming audio into transcripts.
type SpeechToText interface {
	Transcribe(ctx context.Context, audio AudioSource) (TranscriptStream, error)
}

// AudioSource abstracts an inbound audio stream so scenarios do not need
// to depend on the io.Reader transport. Read semantics match io.Reader.
type AudioSource interface {
	Read(p []byte) (int, error)
}

// AudioSubscription is a scenario-facing handle to a live audio stream.
// Close releases the subscription and causes subsequent Reads to return EOF
// after any buffered data has been drained.
type AudioSubscription interface {
	AudioSource
	Close() error
}

// DeviceAudioSubscriber exposes live, per-device mic audio to scenarios.
// Scenarios that need to continuously analyze device audio (for example,
// AudioMonitorScenario) call SubscribeAudio to obtain an AudioSubscription
// that is fed by the transport layer as audio chunks arrive.
type DeviceAudioSubscriber interface {
	SubscribeAudio(ctx context.Context, deviceID string) (AudioSubscription, error)
}

// Transcript is a single recognition result.
type Transcript struct {
	Text       string
	Confidence float64
	IsFinal    bool
}

// TranscriptStream delivers transcripts as they become available.
type TranscriptStream <-chan Transcript

// WakeWordDetection is the result of checking recognized speech for a wake word.
type WakeWordDetection struct {
	Detected bool
	Command  string
}

// WakeWordDetector checks recognized speech for activation phrases and can
// optionally normalize the command text that should be routed to scenarios.
type WakeWordDetector interface {
	Detect(ctx context.Context, spoken string) (WakeWordDetection, error)
}

// TextToSpeech synthesizes audio from text.
type TextToSpeech interface {
	Synthesize(ctx context.Context, text string, opts TTSOptions) (AudioPlayback, error)
}

// TTSOptions configures a scenario-issued synthesis request.
type TTSOptions struct {
	Voice  string
	Format string
}

// AudioPlayback is the synthesized audio stream returned by TextToSpeech.
// Read semantics match io.Reader.
type AudioPlayback interface {
	Read(p []byte) (int, error)
}

// VisionAnalysis describes the result of analyzing a single frame.
type VisionAnalysis struct {
	Caption string
	Labels  []string
}

// VisionAnalyzer interprets images.
type VisionAnalyzer interface {
	Analyze(ctx context.Context, frame image.Image, prompt string) (*VisionAnalysis, error)
}

// SoundEvent describes a classified audio event.
type SoundEvent struct {
	Label      string
	Confidence float64
	AtMS       int64
}

// SoundEventStream delivers sound classification events as they become available.
type SoundEventStream <-chan SoundEvent

// SoundClassifier streams classified events from audio input.
type SoundClassifier interface {
	Classify(ctx context.Context, audio AudioSource) (SoundEventStream, error)
}

// SensorReading is one telemetry snapshot produced by a device client.
type SensorReading struct {
	DeviceID string
	UnixMS   int64
	Values   map[string]float64
}

// SensorConsumer is an optional hook implemented by scenarios that consume
// device telemetry snapshots while active.
type SensorConsumer interface {
	HandleSensor(ctx context.Context, env *Environment, reading SensorReading) error
}

// BluetoothCommand is a server-issued Bluetooth passthrough request.
type BluetoothCommand struct {
	DeviceID   string
	Action     string
	TargetID   string
	Parameters map[string]string
}

// USBCommand is a server-issued USB passthrough request.
type USBCommand struct {
	DeviceID   string
	Action     string
	VendorID   string
	ProductID  string
	Parameters map[string]string
}

// BluetoothEvent captures device-originated Bluetooth passthrough data.
type BluetoothEvent struct {
	DeviceID string
	Event    string
	Data     map[string]string
}

// USBEvent captures device-originated USB passthrough data.
type USBEvent struct {
	DeviceID string
	Event    string
	Data     map[string]string
}

// PassthroughBridge mediates server-directed Bluetooth/USB passthrough
// operations without coupling scenarios to client implementation details.
type PassthroughBridge interface {
	DispatchBluetoothCommand(ctx context.Context, cmd BluetoothCommand) error
	DispatchUSBCommand(ctx context.Context, cmd USBCommand) error
}

// BluetoothEventConsumer optionally receives Bluetooth passthrough events.
type BluetoothEventConsumer interface {
	HandleBluetoothEvent(ctx context.Context, env *Environment, event BluetoothEvent) error
}

// USBEventConsumer optionally receives USB passthrough events.
type USBEventConsumer interface {
	HandleUSBEvent(ctx context.Context, env *Environment, event USBEvent) error
}

// TelephonyBridge exposes external call controls.
type TelephonyBridge interface {
	Call(ctx context.Context, target string) error
	Hangup(ctx context.Context, sessionID string) error
}

// StorageManager provides persistence for scenario state.
type StorageManager interface {
	Put(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
}

// Scheduler provides timer/reminder and recurring activation hooks.
type Scheduler interface {
	Schedule(ctx context.Context, key string, unixMS int64) error
	Due(unixMS int64) []string
	Remove(ctx context.Context, key string) error
}

// StructuredScheduler optionally supports typed scheduler records.
type StructuredScheduler interface {
	ScheduleRecord(ctx context.Context, record storage.ScheduleRecord) error
	DueRecords(unixMS int64) []storage.ScheduleRecord
}

// Broadcaster sends one-to-many notifications or commands.
type Broadcaster interface {
	Notify(ctx context.Context, deviceIDs []string, message string) error
}

// UIHost applies server-driven UI descriptors to terminals.
type UIHost interface {
	Set(ctx context.Context, deviceID string, root ui.Descriptor) error
	Patch(ctx context.Context, deviceID, componentID string, node ui.Descriptor) error
	Clear(ctx context.Context, deviceID, root string) error
}

// Environment is the dependency bag scenarios receive at runtime.
type Environment struct {
	Devices     DeviceManager
	IO          IORouter
	AI          AIBackend
	LLM         LLM
	Vision      VisionAnalyzer
	Sound       SoundClassifier
	STT         SpeechToText
	WakeWord    WakeWordDetector
	TTS         TextToSpeech
	Telephony   TelephonyBridge
	Storage     StorageManager
	Scheduler   Scheduler
	Broadcast   Broadcaster
	UI          UIHost
	DeviceAudio DeviceAudioSubscriber
	Passthrough PassthroughBridge
	Placement   PlacementEngine
	TriggerBus  *IntentEventBus
	Observe     ObservationStore
	World       WorldModel
}

// Scenario is the runtime contract for all server-side behaviors.
type Scenario interface {
	Name() string
	Match(trigger Trigger) bool
	Start(ctx context.Context, env *Environment) error
	Stop() error
}

// ScenarioDefinition is a stateless singleton that can match a request and
// construct per-run activation instances.
type ScenarioDefinition interface { //nolint:revive
	Name() string
	Match(req ActivationRequest) bool
	NewActivation(req ActivationRequest) (Scenario, error)
}

// Suspendable is an optional hook implemented by scenarios that need to
// release live resources when a higher-priority scenario preempts their IO.
type Suspendable interface {
	Suspend() error
}

// Resumable is an optional hook implemented by scenarios that need to
// reacquire resources after preemption is lifted.
type Resumable interface {
	Resume(ctx context.Context, env *Environment) error
}

// EventConsumer is an optional hook implemented by scenarios that want
// typed runtime events delivered while active.
type EventConsumer interface {
	HandleEvent(ctx context.Context, env *Environment, event EventRecord) error
}

// ObservationStore exposes typed observation and artifact history.
type ObservationStore interface {
	Recent(ctx context.Context, kind, zone string, since time.Time) []iorouter.Observation
	Artifact(ctx context.Context, artifactID string) (iorouter.ArtifactRef, bool)
}

// EntityQuery filters world-model lookup operations.
type EntityQuery struct {
	Person        string
	Object        string
	BluetoothMAC  string
	LastKnownOnly bool
	MinConfidence float64
}

// EntityRecord represents one person/object/device world-model entry.
type EntityRecord struct {
	EntityID    string
	Kind        string
	DisplayName string
	LastKnown   *iorouter.LocationEstimate
	LastSeenAt  time.Time
	Confidence  float64
}

// WorldModel provides calibration, entity location, and verification hooks.
type WorldModel interface {
	LocateEntity(ctx context.Context, q EntityQuery) (*iorouter.LocationEstimate, error)
	WhoIsHome(ctx context.Context) ([]EntityRecord, error)
	VerifyDevice(ctx context.Context, deviceID string, method string) error
	RecentObservations(ctx context.Context, zone string, kind string, since time.Time) ([]iorouter.Observation, error)
}
