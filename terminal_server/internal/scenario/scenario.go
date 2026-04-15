package scenario

import (
	"context"
	"image"
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

// Trigger contains routing metadata used for scenario matching.
type Trigger struct {
	Kind      TriggerKind
	SourceID  string
	Intent    string
	Arguments map[string]string
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

// Broadcaster sends one-to-many notifications or commands.
type Broadcaster interface {
	Notify(ctx context.Context, deviceIDs []string, message string) error
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
	DeviceAudio DeviceAudioSubscriber
	Passthrough PassthroughBridge
}

// Scenario is the runtime contract for all server-side behaviors.
type Scenario interface {
	Name() string
	Match(trigger Trigger) bool
	Start(ctx context.Context, env *Environment) error
	Stop() error
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
