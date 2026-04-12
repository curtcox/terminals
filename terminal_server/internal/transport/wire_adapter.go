package transport

import "fmt"

// WireProtoAdapter maps ProtoEnvelope values through wire message structs.
// This mirrors the shape a generated protobuf adapter will use.
type WireProtoAdapter struct{}

// ToInternal converts a wire client envelope into internal message form.
func (WireProtoAdapter) ToInternal(env ProtoClientEnvelope) (ClientMessage, error) {
	switch typed := env.(type) {
	case WireClientMessage:
		return InternalFromWireClient(typed)
	case *WireClientMessage:
		return InternalFromWireClient(*typed)
	default:
		return ClientMessage{}, fmt.Errorf("unsupported wire client envelope %T", env)
	}
}

// FromInternal converts an internal server message into wire envelope form.
func (WireProtoAdapter) FromInternal(msg ServerMessage) (ProtoServerEnvelope, error) {
	return WireFromInternalServer(msg), nil
}
