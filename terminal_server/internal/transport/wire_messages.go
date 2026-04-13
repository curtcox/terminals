package transport

// WireRegisterRequest is a protobuf-adapter-friendly register payload.
type WireRegisterRequest struct {
	DeviceID     string
	DeviceName   string
	DeviceType   string
	Platform     string
	Capabilities []DataEntry
}

// WireCapabilityUpdateRequest is a protobuf-adapter-friendly capability payload.
type WireCapabilityUpdateRequest struct {
	DeviceID     string
	Capabilities []DataEntry
}

// WireHeartbeatRequest is a protobuf-adapter-friendly heartbeat payload.
type WireHeartbeatRequest struct {
	DeviceID string
}

// WireCommandRequest is a protobuf-adapter-friendly command payload.
type WireCommandRequest struct {
	RequestID string
	DeviceID  string
	Action    WireCommandAction
	Kind      WireCommandKind
	Text      string
	Intent    string
}

// WireCommandAction mirrors control.proto CommandAction.
type WireCommandAction int32

// WireCommandAction constants mirror control.proto CommandAction enum values.
const (
	// WireCommandActionUnspecified indicates the command action was not set.
	WireCommandActionUnspecified WireCommandAction = 0
	// WireCommandActionStart requests scenario/flow activation.
	WireCommandActionStart WireCommandAction = 1
	// WireCommandActionStop requests scenario/flow stop.
	WireCommandActionStop WireCommandAction = 2
)

// WireCommandKind mirrors control.proto command kind semantics.
// We keep string values stable to preserve existing system/manual/voice logic.
type WireCommandKind int32

// WireCommandKind constants mirror control.proto command kind semantics.
// String values are kept stable to preserve existing system/manual/voice logic.
const (
	// WireCommandKindUnspecified indicates the command kind was not set.
	WireCommandKindUnspecified WireCommandKind = 0
	// WireCommandKindVoice indicates speech-derived commands.
	WireCommandKindVoice WireCommandKind = 1
	// WireCommandKindManual indicates explicit manual commands.
	WireCommandKindManual WireCommandKind = 2
	// WireCommandKindSystem indicates introspection/admin commands.
	WireCommandKindSystem WireCommandKind = 3
)

// WireClientMessage is a protobuf-adapter-friendly oneof shape.
type WireClientMessage struct {
	Register   *WireRegisterRequest
	Capability *WireCapabilityUpdateRequest
	Heartbeat  *WireHeartbeatRequest
	Command    *WireCommandRequest
}

// WireRegisterResponse is a protobuf-adapter-friendly register response.
type WireRegisterResponse struct {
	ServerID string
	Message  string
}

// WireCommandResult is a protobuf-adapter-friendly command result payload.
type WireCommandResult struct {
	RequestID     string
	ScenarioStart string
	ScenarioStop  string
	Notification  string
	Data          []DataEntry
}

// WireControlError is a protobuf-adapter-friendly control error payload.
type WireControlError struct {
	Code    WireControlErrorCode
	Message string
}

// WireControlErrorCode mirrors control.proto ControlErrorCode semantics.
type WireControlErrorCode int32

const (
	// WireControlErrorCodeUnspecified indicates no explicit error classification.
	WireControlErrorCodeUnspecified WireControlErrorCode = 0
	// WireControlErrorCodeInvalidClientMessage indicates malformed top-level payload.
	WireControlErrorCodeInvalidClientMessage WireControlErrorCode = 1
	// WireControlErrorCodeInvalidCommandAction indicates unknown/unsupported action.
	WireControlErrorCodeInvalidCommandAction WireControlErrorCode = 2
	// WireControlErrorCodeInvalidCommandKind indicates unknown/unsupported command kind.
	WireControlErrorCodeInvalidCommandKind WireControlErrorCode = 3
	// WireControlErrorCodeMissingCommandIntent indicates missing required intent.
	WireControlErrorCodeMissingCommandIntent WireControlErrorCode = 4
	// WireControlErrorCodeMissingCommandText indicates missing required text.
	WireControlErrorCodeMissingCommandText WireControlErrorCode = 5
	// WireControlErrorCodeMissingCommandDeviceID indicates missing required device id.
	WireControlErrorCodeMissingCommandDeviceID WireControlErrorCode = 6
	// WireControlErrorCodeProtocolViolation indicates stream/session protocol misuse.
	WireControlErrorCodeProtocolViolation WireControlErrorCode = 7
	// WireControlErrorCodeUnknown indicates uncategorized server errors.
	WireControlErrorCodeUnknown WireControlErrorCode = 99
)

// WireServerMessage is a protobuf-adapter-friendly oneof response shape.
type WireServerMessage struct {
	RegisterAck   *WireRegisterResponse
	CommandResult *WireCommandResult
	SetUI         *uiWireDescriptor
	UpdateUI      *uiWireUpdate
	StartStream   *WireStartStream
	StopStream    *WireStopStream
	RouteStream   *WireRouteStream
	TransitionUI  *uiWireTransition
	Error         *WireControlError
}

// WireStartStream is a protobuf-adapter-friendly start stream payload.
type WireStartStream struct {
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	Metadata       []DataEntry
}

// WireStopStream is a protobuf-adapter-friendly stop stream payload.
type WireStopStream struct {
	StreamID string
}

// WireRouteStream is a protobuf-adapter-friendly route stream payload.
type WireRouteStream struct {
	StreamID       string
	SourceDeviceID string
	TargetDeviceID string
	Kind           string
}

// uiWireDescriptor is a compact wire representation for UI descriptors.
type uiWireDescriptor struct {
	ID       string
	Type     string
	Props    []DataEntry
	Children []uiWireDescriptor
}

// uiWireUpdate is a compact wire representation for UpdateUI payloads.
type uiWireUpdate struct {
	ComponentID string
	Node        uiWireDescriptor
}

// uiWireTransition is a compact wire representation for TransitionUI payloads.
type uiWireTransition struct {
	Transition string
	DurationMS int32
}
