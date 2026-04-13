package transport

import (
	"errors"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// ErrInvalidWireMessage indicates the wire message had no payload.
var ErrInvalidWireMessage = errors.New("invalid wire message")

// InternalFromWireClient converts adapter-level wire messages to internal messages.
func InternalFromWireClient(w WireClientMessage) (ClientMessage, error) {
	switch {
	case w.Register != nil:
		return ClientMessage{
			Register: &RegisterRequest{
				DeviceID:     w.Register.DeviceID,
				DeviceName:   w.Register.DeviceName,
				DeviceType:   w.Register.DeviceType,
				Platform:     w.Register.Platform,
				Capabilities: DecodeDataEntries(w.Register.Capabilities),
			},
		}, nil
	case w.Capability != nil:
		return ClientMessage{
			Capability: &CapabilityUpdateRequest{
				DeviceID:     w.Capability.DeviceID,
				Capabilities: DecodeDataEntries(w.Capability.Capabilities),
			},
		}, nil
	case w.Heartbeat != nil:
		return ClientMessage{
			Heartbeat: &HeartbeatRequest{
				DeviceID: w.Heartbeat.DeviceID,
			},
		}, nil
	case w.Command != nil:
		return ClientMessage{
			Command: &CommandRequest{
				RequestID: w.Command.RequestID,
				DeviceID:  w.Command.DeviceID,
				Action:    internalActionFromWire(w.Command.Action),
				Kind:      internalKindFromWire(w.Command.Kind),
				Text:      w.Command.Text,
				Intent:    w.Command.Intent,
			},
		}, nil
	default:
		return ClientMessage{}, ErrInvalidWireMessage
	}
}

func internalActionFromWire(action WireCommandAction) string {
	switch action {
	case WireCommandActionUnspecified:
		return ""
	case WireCommandActionStart:
		return CommandActionStart
	case WireCommandActionStop:
		return CommandActionStop
	default:
		return ""
	}
}

func internalKindFromWire(kind WireCommandKind) string {
	switch kind {
	case WireCommandKindUnspecified:
		return ""
	case WireCommandKindVoice:
		return CommandKindVoice
	case WireCommandKindManual:
		return CommandKindManual
	case WireCommandKindSystem:
		return CommandKindSystem
	default:
		return ""
	}
}

func wireErrorCodeFromInternal(code string) WireControlErrorCode {
	switch code {
	case ErrorCodeInvalidClientMessage:
		return WireControlErrorCodeInvalidClientMessage
	case ErrorCodeInvalidCommandAction:
		return WireControlErrorCodeInvalidCommandAction
	case ErrorCodeInvalidCommandKind:
		return WireControlErrorCodeInvalidCommandKind
	case ErrorCodeMissingIntent:
		return WireControlErrorCodeMissingCommandIntent
	case ErrorCodeMissingText:
		return WireControlErrorCodeMissingCommandText
	case ErrorCodeMissingDeviceID:
		return WireControlErrorCodeMissingCommandDeviceID
	case ErrorCodeProtocolViolation:
		return WireControlErrorCodeProtocolViolation
	default:
		return WireControlErrorCodeUnknown
	}
}

// WireFromInternalServer converts internal server messages to adapter-level wire messages.
func WireFromInternalServer(msg ServerMessage) WireServerMessage {
	out := WireServerMessage{}
	if msg.RegisterAck != nil {
		out.RegisterAck = &WireRegisterResponse{
			ServerID: msg.RegisterAck.ServerID,
			Message:  msg.RegisterAck.Message,
		}
	}
	if msg.CommandAck != "" || msg.Notification != "" || msg.ScenarioStart != "" || msg.ScenarioStop != "" || len(msg.Data) > 0 {
		out.CommandResult = &WireCommandResult{
			RequestID:     msg.CommandAck,
			ScenarioStart: msg.ScenarioStart,
			ScenarioStop:  msg.ScenarioStop,
			Notification:  msg.Notification,
			Data:          EncodeDataMap(msg.Data),
		}
	}
	if msg.SetUI != nil {
		uiNode := wireDescriptorFromUI(*msg.SetUI)
		out.SetUI = &uiNode
	}
	if msg.UpdateUI != nil {
		out.UpdateUI = &uiWireUpdate{
			ComponentID: msg.UpdateUI.ComponentID,
			Node:        wireDescriptorFromUI(msg.UpdateUI.Node),
		}
	}
	if msg.StartStream != nil {
		out.StartStream = &WireStartStream{
			StreamID:       msg.StartStream.StreamID,
			Kind:           msg.StartStream.Kind,
			SourceDeviceID: msg.StartStream.SourceDeviceID,
			TargetDeviceID: msg.StartStream.TargetDeviceID,
			Metadata:       EncodeDataMap(msg.StartStream.Metadata),
		}
	}
	if msg.StopStream != nil {
		out.StopStream = &WireStopStream{
			StreamID: msg.StopStream.StreamID,
		}
	}
	if msg.RouteStream != nil {
		out.RouteStream = &WireRouteStream{
			StreamID:       msg.RouteStream.StreamID,
			SourceDeviceID: msg.RouteStream.SourceDeviceID,
			TargetDeviceID: msg.RouteStream.TargetDeviceID,
			Kind:           msg.RouteStream.Kind,
		}
	}
	if msg.TransitionUI != nil {
		out.TransitionUI = &uiWireTransition{
			Transition: msg.TransitionUI.Transition,
			DurationMS: msg.TransitionUI.DurationMS,
		}
	}
	if msg.ErrorCode != "" || msg.Error != "" {
		out.Error = &WireControlError{
			Code:    wireErrorCodeFromInternal(msg.ErrorCode),
			Message: msg.Error,
		}
	}
	return out
}

func wireDescriptorFromUI(d ui.Descriptor) uiWireDescriptor {
	children := make([]uiWireDescriptor, 0, len(d.Children))
	for _, child := range d.Children {
		children = append(children, wireDescriptorFromUI(child))
	}
	return uiWireDescriptor{
		ID:       d.ID,
		Type:     d.Type,
		Props:    EncodeDataMap(d.Props),
		Children: children,
	}
}
