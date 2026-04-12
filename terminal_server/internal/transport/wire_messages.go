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
	WireCommandActionUnspecified WireCommandAction = 0
	WireCommandActionStart       WireCommandAction = 1
	WireCommandActionStop        WireCommandAction = 2
)

// WireCommandKind mirrors control.proto command kind semantics.
// We keep string values stable to preserve existing system/manual/voice logic.
type WireCommandKind int32

// WireCommandKind constants mirror control.proto command kind semantics.
// String values are kept stable to preserve existing system/manual/voice logic.
const (
	WireCommandKindUnspecified WireCommandKind = 0
	WireCommandKindVoice       WireCommandKind = 1
	WireCommandKindManual      WireCommandKind = 2
	WireCommandKindSystem      WireCommandKind = 3
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
	Code    string
	Message string
}

// WireServerMessage is a protobuf-adapter-friendly oneof response shape.
type WireServerMessage struct {
	RegisterAck   *WireRegisterResponse
	CommandResult *WireCommandResult
	SetUI         *uiWireDescriptor
	Error         *WireControlError
}

// uiWireDescriptor is a compact wire representation for UI descriptors.
type uiWireDescriptor struct {
	ID       string
	Type     string
	Props    []DataEntry
	Children []uiWireDescriptor
}
