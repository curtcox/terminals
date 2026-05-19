package transport

import (
	"fmt"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
)

func internalFromProtoRequest(req *controlv1.ConnectRequest) (ClientMessage, error) {
	if req == nil {
		return ClientMessage{}, fmt.Errorf("nil connect request")
	}
	switch payload := req.GetPayload().(type) {
	case *controlv1.ConnectRequest_Hello,
		*controlv1.ConnectRequest_CapabilitySnapshot,
		*controlv1.ConnectRequest_CapabilityDelta,
		*controlv1.ConnectRequest_Register,
		*controlv1.ConnectRequest_Capability,
		*controlv1.ConnectRequest_Heartbeat:
		return internalFromProtoSessionPayload(payload)
	default:
		return internalFromProtoDataPayload(payload)
	}
}

func internalFromProtoSessionPayload(payload any) (ClientMessage, error) {
	switch payload := payload.(type) {
	case *controlv1.ConnectRequest_Hello:
		hello := payload.Hello
		identity := hello.GetIdentity()
		return ClientMessage{
			Hello: &HelloRequest{
				DeviceID:      hello.GetDeviceId(),
				DeviceName:    identity.GetDeviceName(),
				DeviceType:    identity.GetDeviceType(),
				Platform:      identity.GetPlatform(),
				ClientVersion: hello.GetClientVersion(),
			},
		}, nil
	case *controlv1.ConnectRequest_CapabilitySnapshot:
		caps := payload.CapabilitySnapshot.GetCapabilities()
		deviceID := payload.CapabilitySnapshot.GetDeviceId()
		if deviceID == "" {
			deviceID = caps.GetDeviceId()
		}
		return ClientMessage{
			CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:     deviceID,
				Generation:   payload.CapabilitySnapshot.GetGeneration(),
				Capabilities: capabilitiesToDataMap(caps),
			},
		}, nil
	case *controlv1.ConnectRequest_CapabilityDelta:
		caps := payload.CapabilityDelta.GetCapabilities()
		deviceID := payload.CapabilityDelta.GetDeviceId()
		if deviceID == "" {
			deviceID = caps.GetDeviceId()
		}
		return ClientMessage{
			CapabilityDelta: &CapabilityDeltaRequest{
				DeviceID:     deviceID,
				Generation:   payload.CapabilityDelta.GetGeneration(),
				Reason:       payload.CapabilityDelta.GetReason(),
				Capabilities: capabilitiesToDataMap(caps),
			},
		}, nil
	case *controlv1.ConnectRequest_Register:
		//nolint:staticcheck // Legacy payload accepted for compatibility but normalized to snapshot semantics.
		caps := payload.Register.GetCapabilities()
		return ClientMessage{
			CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:     caps.GetDeviceId(),
				Generation:   1,
				Capabilities: capabilitiesToDataMap(caps),
			},
		}, nil
	case *controlv1.ConnectRequest_Capability:
		return ClientMessage{}, fmt.Errorf("deprecated payload capability_update is not supported; use capability_snapshot/capability_delta")
	case *controlv1.ConnectRequest_Heartbeat:
		return ClientMessage{
			Heartbeat: &HeartbeatRequest{
				DeviceID: payload.Heartbeat.GetDeviceId(),
			},
		}, nil
	default:
		return ClientMessage{}, nil
	}
}

func internalFromProtoDataPayload(payload any) (ClientMessage, error) {
	switch payload := payload.(type) {
	case *controlv1.ConnectRequest_Input:
		return clientMessageFromProtoInput(payload.Input), nil
	case *controlv1.ConnectRequest_Command:
		command := payload.Command
		return ClientMessage{
			Command: &CommandRequest{
				RequestID: command.GetRequestId(),
				DeviceID:  command.GetDeviceId(),
				Action:    internalActionFromProto(command.GetAction()),
				Kind:      internalKindFromProto(command.GetKind()),
				Text:      command.GetText(),
				Intent:    command.GetIntent(),
				Arguments: commandArgumentsToInternalMap(command.GetTypedArguments(), command.GetArguments()),
			},
		}, nil
	case *controlv1.ConnectRequest_Sensor:
		sensor := payload.Sensor
		values := map[string]float64{}
		for key, value := range sensor.GetValues() {
			values[key] = value
		}
		return ClientMessage{
			Sensor: &SensorDataRequest{
				DeviceID: sensor.GetDeviceId(),
				UnixMS:   sensor.GetUnixMs(),
				Values:   values,
			},
		}, nil
	case *controlv1.ConnectRequest_StreamReady:
		return ClientMessage{
			StreamReady: &StreamReadyRequest{
				StreamID: payload.StreamReady.GetStreamId(),
			},
		}, nil
	case *controlv1.ConnectRequest_WebrtcSignal:
		signal := payload.WebrtcSignal
		return ClientMessage{
			WebRTCSignal: &WebRTCSignalRequest{
				StreamID:   signal.GetStreamId(),
				SignalType: internalWebRTCSignalTypeFromProto(signal.GetSignalType(), signal.GetSignalTypeEnum()),
				Payload:    signal.GetPayload(),
			},
		}, nil
	case *controlv1.ConnectRequest_VoiceAudio:
		voice := payload.VoiceAudio
		audio := append([]byte(nil), voice.GetAudio()...)
		return ClientMessage{
			VoiceAudio: &VoiceAudioRequest{
				DeviceID:   voice.GetDeviceId(),
				Audio:      audio,
				SampleRate: voice.GetSampleRate(),
				IsFinal:    voice.GetIsFinal(),
			},
		}, nil
	case *controlv1.ConnectRequest_ObservationMessage:
		return ClientMessage{
			Observation: &ObservationRequest{
				Observation: observationFromProto(payload.ObservationMessage.GetObservation()),
			},
		}, nil
	case *controlv1.ConnectRequest_ArtifactAvailable:
		return ClientMessage{
			ArtifactReady: &ArtifactAvailableRequest{
				Artifact: artifactFromProto(payload.ArtifactAvailable.GetArtifact()),
			},
		}, nil
	case *controlv1.ConnectRequest_FlowStats:
		stats := payload.FlowStats
		return ClientMessage{
			FlowStats: &FlowStatsRequest{
				FlowID:        stats.GetFlowId(),
				CPUPct:        stats.GetCpuPct(),
				MemMB:         stats.GetMemMb(),
				DroppedFrames: stats.GetDroppedFrames(),
				State:         stats.GetState(),
				StateEnum:     stats.GetStateEnum(),
				Error:         stats.GetError(),
			},
		}, nil
	case *controlv1.ConnectRequest_ClockSample:
		sample := payload.ClockSample
		return ClientMessage{
			ClockSample: &ClockSampleRequest{
				DeviceID:     sample.GetDeviceId(),
				ClientUnixMS: sample.GetClientUnixMs(),
				ServerUnixMS: sample.GetServerUnixMs(),
				ErrorMS:      sample.GetErrorMs(),
			},
		}, nil
	case *controlv1.ConnectRequest_BugReport:
		return ClientMessage{
			BugReport: payload.BugReport,
		}, nil
	default:
		return ClientMessage{}, nil
	}
}

func clientMessageFromProtoInput(input *iov1.InputEvent) ClientMessage {
	internal := &InputRequest{
		DeviceID: input.GetDeviceId(),
	}
	if action := input.GetUiAction(); action != nil {
		internal.ComponentID = action.GetComponentId()
		internal.Action = action.GetAction()
		internal.Value = action.GetValue()
	}
	if key := input.GetKey(); key != nil {
		internal.KeyText = key.GetText()
	}
	if pointer := input.GetPointer(); pointer != nil {
		internal.Action = internalPointerActionFromProto(pointer.GetAction(), pointer.GetActionEnum())
	}
	if touch := input.GetTouch(); touch != nil {
		internal.Action = internalTouchActionFromProto(touch.GetAction(), touch.GetActionEnum())
	}
	return ClientMessage{Input: internal}
}
