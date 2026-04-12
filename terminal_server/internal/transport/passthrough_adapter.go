package transport

import "fmt"

// PassthroughProtoAdapter is a temporary adapter used before protobuf codegen is wired.
// It treats proto envelopes as internal transport messages.
type PassthroughProtoAdapter struct{}

// ToInternal maps passthrough client envelopes to internal messages.
func (PassthroughProtoAdapter) ToInternal(env ProtoClientEnvelope) (ClientMessage, error) {
	switch typed := env.(type) {
	case ClientMessage:
		return typed, nil
	case *ClientMessage:
		return *typed, nil
	default:
		return ClientMessage{}, fmt.Errorf("unsupported passthrough client envelope %T", env)
	}
}

// FromInternal maps internal server messages to passthrough proto envelopes.
func (PassthroughProtoAdapter) FromInternal(msg ServerMessage) (ProtoServerEnvelope, error) {
	return msg, nil
}
