package scenario

import "context"

// TriggerKind identifies how a scenario was requested.
type TriggerKind string

const (
	TriggerVoice    TriggerKind = "voice"
	TriggerSchedule TriggerKind = "schedule"
	TriggerEvent    TriggerKind = "event"
	TriggerManual   TriggerKind = "manual"
	TriggerCascade  TriggerKind = "cascade"
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

// AIBackend represents scenario-accessible AI services.
type AIBackend interface {
	Query(ctx context.Context, input string) (string, error)
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
	Devices   DeviceManager
	IO        IORouter
	AI        AIBackend
	Telephony TelephonyBridge
	Storage   StorageManager
	Scheduler Scheduler
	Broadcast Broadcaster
}

// Scenario is the runtime contract for all server-side behaviors.
type Scenario interface {
	Name() string
	Match(trigger Trigger) bool
	Start(ctx context.Context, env *Environment) error
	Stop() error
}
