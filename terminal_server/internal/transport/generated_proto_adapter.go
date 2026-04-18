package transport

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// GeneratedProtoAdapter maps generated protobuf messages to internal transport messages.
type GeneratedProtoAdapter struct{}

// ToInternal converts a generated protobuf request envelope into internal message form.
func (GeneratedProtoAdapter) ToInternal(env ProtoClientEnvelope) (ClientMessage, error) {
	switch typed := env.(type) {
	case *controlv1.ConnectRequest:
		return internalFromProtoRequest(typed)
	case controlv1.ConnectRequest:
		return internalFromProtoRequest(&typed)
	default:
		return ClientMessage{}, fmt.Errorf("unsupported proto client envelope %T", env)
	}
}

// FromInternal converts an internal server message to a generated protobuf response envelope.
func (GeneratedProtoAdapter) FromInternal(msg ServerMessage) (ProtoServerEnvelope, error) {
	return protoFromInternalServer(msg), nil
}

func internalFromProtoRequest(req *controlv1.ConnectRequest) (ClientMessage, error) {
	if req == nil {
		return ClientMessage{}, fmt.Errorf("nil connect request")
	}

	switch payload := req.GetPayload().(type) {
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
		caps := payload.Register.GetCapabilities()
		identity := caps.GetIdentity()
		return ClientMessage{
			Register: &RegisterRequest{
				DeviceID:     caps.GetDeviceId(),
				DeviceName:   identity.GetDeviceName(),
				DeviceType:   identity.GetDeviceType(),
				Platform:     identity.GetPlatform(),
				Capabilities: capabilitiesToDataMap(caps),
			},
		}, nil
	case *controlv1.ConnectRequest_Capability:
		caps := payload.Capability.GetCapabilities()
		return ClientMessage{
			Capability: &CapabilityUpdateRequest{
				DeviceID:     caps.GetDeviceId(),
				Capabilities: capabilitiesToDataMap(caps),
			},
		}, nil
	case *controlv1.ConnectRequest_Heartbeat:
		return ClientMessage{
			Heartbeat: &HeartbeatRequest{
				DeviceID: payload.Heartbeat.GetDeviceId(),
			},
		}, nil
	case *controlv1.ConnectRequest_Input:
		input := payload.Input
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
		return ClientMessage{Input: internal}, nil
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
				Arguments: command.GetArguments(),
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
				SignalType: signal.GetSignalType(),
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

func protoFromInternalServer(msg ServerMessage) *controlv1.ConnectResponse {
	switch {
	case msg.HelloAck != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_HelloAck{
				HelloAck: &controlv1.HelloAck{
					ServerId:            msg.HelloAck.ServerID,
					SessionId:           msg.HelloAck.SessionID,
					HeartbeatIntervalMs: msg.HelloAck.HeartbeatIntervalMS,
				},
			},
		}
	case msg.CapabilityAck != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_CapabilityAck{
				CapabilityAck: &controlv1.CapabilityAck{
					DeviceId:           msg.CapabilityAck.DeviceID,
					AcceptedGeneration: msg.CapabilityAck.AcceptedGeneration,
					SnapshotApplied:    msg.CapabilityAck.SnapshotApplied,
				},
			},
		}
	case msg.RegisterAck != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_RegisterAck{
				RegisterAck: &controlv1.RegisterAck{
					ServerId: msg.RegisterAck.ServerID,
					Message:  msg.RegisterAck.Message,
					Metadata: msg.RegisterAck.Metadata,
				},
			},
		}
	case msg.SetUI != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_SetUi{
				SetUi: &uiv1.SetUI{
					Root: descriptorToUINode(*msg.SetUI),
				},
			},
		}
	case msg.UpdateUI != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_UpdateUi{
				UpdateUi: &uiv1.UpdateUI{
					ComponentId: msg.UpdateUI.ComponentID,
					Node:        descriptorToUINode(msg.UpdateUI.Node),
				},
			},
		}
	case msg.StartStream != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_StartStream{
				StartStream: &iov1.StartStream{
					StreamId:       msg.StartStream.StreamID,
					Kind:           msg.StartStream.Kind,
					SourceDeviceId: msg.StartStream.SourceDeviceID,
					TargetDeviceId: msg.StartStream.TargetDeviceID,
					Metadata:       msg.StartStream.Metadata,
				},
			},
		}
	case msg.StopStream != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_StopStream{
				StopStream: &iov1.StopStream{
					StreamId: msg.StopStream.StreamID,
				},
			},
		}
	case msg.RouteStream != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_RouteStream{
				RouteStream: &iov1.RouteStream{
					StreamId:       msg.RouteStream.StreamID,
					SourceDeviceId: msg.RouteStream.SourceDeviceID,
					TargetDeviceId: msg.RouteStream.TargetDeviceID,
					Kind:           msg.RouteStream.Kind,
				},
			},
		}
	case msg.WebRTCSignal != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_WebrtcSignal{
				WebrtcSignal: &controlv1.WebRTCSignal{
					StreamId:   msg.WebRTCSignal.StreamID,
					SignalType: msg.WebRTCSignal.SignalType,
					Payload:    msg.WebRTCSignal.Payload,
				},
			},
		}
	case msg.TransitionUI != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_TransitionUi{
				TransitionUi: &uiv1.TransitionUI{
					Transition: msg.TransitionUI.Transition,
					DurationMs: msg.TransitionUI.DurationMS,
				},
			},
		}
	case msg.PlayAudio != nil:
		audio := append([]byte(nil), msg.PlayAudio.Audio...)
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_PlayAudio{
				PlayAudio: &iov1.PlayAudio{
					RequestId: msg.PlayAudio.RequestID,
					DeviceId:  msg.PlayAudio.DeviceID,
					Source: &iov1.PlayAudio_PcmData{
						PcmData: audio,
					},
					Format: msg.PlayAudio.Format,
				},
			},
		}
	case msg.StartFlow != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_StartFlow{
				StartFlow: &iov1.StartFlow{
					FlowId: msg.StartFlow.FlowID,
					Plan:   flowPlanToProto(msg.StartFlow.Plan),
				},
			},
		}
	case msg.PatchFlow != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_PatchFlow{
				PatchFlow: &iov1.PatchFlow{
					FlowId: msg.PatchFlow.FlowID,
					Plan:   flowPlanToProto(msg.PatchFlow.Plan),
				},
			},
		}
	case msg.StopFlow != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_StopFlow{
				StopFlow: &iov1.StopFlow{
					FlowId: msg.StopFlow.FlowID,
				},
			},
		}
	case msg.RequestArtifact != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_RequestArtifact{
				RequestArtifact: &iov1.RequestArtifact{
					ArtifactId: msg.RequestArtifact.ArtifactID,
				},
			},
		}
	case msg.BugReportAck != nil:
		ack := msg.BugReportAck
		if ack == nil {
			ack = &diagnosticsv1.BugReportAck{}
		}
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_BugReportAck{
				BugReportAck: ack,
			},
		}
	case msg.CommandAck != "" || msg.ScenarioStart != "" || msg.ScenarioStop != "" || msg.Notification != "" || len(msg.Data) > 0:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_CommandResult{
				CommandResult: &controlv1.CommandResult{
					RequestId:     msg.CommandAck,
					ScenarioStart: msg.ScenarioStart,
					ScenarioStop:  msg.ScenarioStop,
					Notification:  msg.Notification,
					Data:          msg.Data,
				},
			},
		}
	case msg.Notification != "":
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_Notification{
				Notification: &uiv1.Notification{
					Body:  msg.Notification,
					Level: "info",
				},
			},
		}
	case msg.Error != "" || msg.ErrorCode != "":
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_Error{
				Error: &controlv1.ControlError{
					Code:    protoErrorCodeFromInternal(msg.ErrorCode),
					Message: msg.Error,
				},
			},
		}
	default:
		return &controlv1.ConnectResponse{}
	}
}

func internalActionFromProto(action controlv1.CommandAction) string {
	switch action {
	case controlv1.CommandAction_COMMAND_ACTION_UNSPECIFIED:
		return ""
	case controlv1.CommandAction_COMMAND_ACTION_START:
		return CommandActionStart
	case controlv1.CommandAction_COMMAND_ACTION_STOP:
		return CommandActionStop
	default:
		return ""
	}
}

func internalKindFromProto(kind controlv1.CommandKind) string {
	switch kind {
	case controlv1.CommandKind_COMMAND_KIND_UNSPECIFIED:
		return ""
	case controlv1.CommandKind_COMMAND_KIND_VOICE:
		return CommandKindVoice
	case controlv1.CommandKind_COMMAND_KIND_MANUAL:
		return CommandKindManual
	case controlv1.CommandKind_COMMAND_KIND_SYSTEM:
		return CommandKindSystem
	default:
		return ""
	}
}

func protoErrorCodeFromInternal(code string) controlv1.ControlErrorCode {
	switch code {
	case ErrorCodeInvalidClientMessage:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_INVALID_CLIENT_MESSAGE
	case ErrorCodeInvalidCommandAction:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_INVALID_COMMAND_ACTION
	case ErrorCodeInvalidCommandKind:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_INVALID_COMMAND_KIND
	case ErrorCodeMissingIntent:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT
	case ErrorCodeMissingText:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_MISSING_COMMAND_TEXT
	case ErrorCodeMissingDeviceID:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_MISSING_COMMAND_DEVICE_ID
	case ErrorCodeProtocolViolation:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
	default:
		return controlv1.ControlErrorCode_CONTROL_ERROR_CODE_UNKNOWN
	}
}

func capabilitiesToDataMap(caps *capabilitiesv1.DeviceCapabilities) map[string]string {
	if caps == nil {
		return map[string]string{}
	}

	out := map[string]string{
		"device_id": caps.GetDeviceId(),
	}
	identity := caps.GetIdentity()
	if identity != nil {
		out["device_name"] = identity.GetDeviceName()
		out["device_type"] = identity.GetDeviceType()
		out["platform"] = identity.GetPlatform()
	}
	if screen := caps.GetScreen(); screen != nil {
		out["screen.width"] = strconv.FormatInt(int64(screen.GetWidth()), 10)
		out["screen.height"] = strconv.FormatInt(int64(screen.GetHeight()), 10)
		out["screen.density"] = strconv.FormatFloat(screen.GetDensity(), 'f', -1, 64)
		out["screen.touch"] = strconv.FormatBool(screen.GetTouch())
		if orientation := strings.TrimSpace(screen.GetOrientation()); orientation != "" {
			out["screen.orientation"] = orientation
		}
		out["screen.fullscreen_supported"] = strconv.FormatBool(screen.GetFullscreenSupported())
		out["screen.multi_window_supported"] = strconv.FormatBool(screen.GetMultiWindowSupported())
		if safeArea := screen.GetSafeArea(); safeArea != nil {
			out["screen.safe_area.left"] = strconv.FormatInt(int64(safeArea.GetLeft()), 10)
			out["screen.safe_area.top"] = strconv.FormatInt(int64(safeArea.GetTop()), 10)
			out["screen.safe_area.right"] = strconv.FormatInt(int64(safeArea.GetRight()), 10)
			out["screen.safe_area.bottom"] = strconv.FormatInt(int64(safeArea.GetBottom()), 10)
		}
	}
	if displays := caps.GetDisplays(); len(displays) > 0 {
		out["display.count"] = strconv.FormatInt(int64(len(displays)), 10)
		for idx, display := range displays {
			prefix := "display." + strconv.Itoa(idx)
			if displayID := strings.TrimSpace(display.GetDisplayId()); displayID != "" {
				out[prefix+".id"] = displayID
			}
			if name := strings.TrimSpace(display.GetDisplayName()); name != "" {
				out[prefix+".name"] = name
			}
			out[prefix+".primary"] = strconv.FormatBool(display.GetPrimary())
			if screen := display.GetScreen(); screen != nil {
				out[prefix+".width"] = strconv.FormatInt(int64(screen.GetWidth()), 10)
				out[prefix+".height"] = strconv.FormatInt(int64(screen.GetHeight()), 10)
				out[prefix+".density"] = strconv.FormatFloat(screen.GetDensity(), 'f', -1, 64)
				if orientation := strings.TrimSpace(screen.GetOrientation()); orientation != "" {
					out[prefix+".orientation"] = orientation
				}
			}
		}
	}
	if keyboard := caps.GetKeyboard(); keyboard != nil {
		out["keyboard.physical"] = strconv.FormatBool(keyboard.GetPhysical())
		out["keyboard.layout"] = keyboard.GetLayout()
	}
	if pointer := caps.GetPointer(); pointer != nil {
		out["pointer.type"] = pointer.GetType()
		out["pointer.hover"] = strconv.FormatBool(pointer.GetHover())
	}
	if touch := caps.GetTouch(); touch != nil {
		out["touch.supported"] = strconv.FormatBool(touch.GetSupported())
		out["touch.max_points"] = strconv.FormatInt(int64(touch.GetMaxPoints()), 10)
	}
	if speakers := caps.GetSpeakers(); speakers != nil {
		out["speakers.present"] = "true"
		if speakers.GetChannels() > 0 {
			out["speakers.channels"] = strconv.FormatInt(int64(speakers.GetChannels()), 10)
		}
		if rates := joinInts(speakers.GetSampleRates()); rates != "" {
			out["speakers.sample_rates"] = rates
		}
		if endpoints := speakers.GetEndpoints(); len(endpoints) > 0 {
			out["speakers.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "speakers.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				out[prefix+".channels"] = strconv.FormatInt(int64(endpoint.GetChannels()), 10)
				if rates := joinInts(endpoint.GetSampleRates()); rates != "" {
					out[prefix+".sample_rates"] = rates
				}
				out[prefix+".available"] = strconv.FormatBool(endpoint.GetAvailable())
			}
		}
	}
	if mic := caps.GetMicrophone(); mic != nil {
		out["microphone.present"] = "true"
		if mic.GetChannels() > 0 {
			out["microphone.channels"] = strconv.FormatInt(int64(mic.GetChannels()), 10)
		}
		if rates := joinInts(mic.GetSampleRates()); rates != "" {
			out["microphone.sample_rates"] = rates
		}
		if endpoints := mic.GetEndpoints(); len(endpoints) > 0 {
			out["microphone.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "microphone.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				out[prefix+".channels"] = strconv.FormatInt(int64(endpoint.GetChannels()), 10)
				if rates := joinInts(endpoint.GetSampleRates()); rates != "" {
					out[prefix+".sample_rates"] = rates
				}
				out[prefix+".available"] = strconv.FormatBool(endpoint.GetAvailable())
			}
		}
	}
	if camera := caps.GetCamera(); camera != nil {
		out["camera.present"] = "true"
		if front := camera.GetFront(); front != nil {
			if front.GetWidth() > 0 {
				out["camera.front.width"] = strconv.FormatInt(int64(front.GetWidth()), 10)
			}
			if front.GetHeight() > 0 {
				out["camera.front.height"] = strconv.FormatInt(int64(front.GetHeight()), 10)
			}
			if front.GetFps() > 0 {
				out["camera.front.fps"] = strconv.FormatInt(int64(front.GetFps()), 10)
			}
		}
		if back := camera.GetBack(); back != nil {
			if back.GetWidth() > 0 {
				out["camera.back.width"] = strconv.FormatInt(int64(back.GetWidth()), 10)
			}
			if back.GetHeight() > 0 {
				out["camera.back.height"] = strconv.FormatInt(int64(back.GetHeight()), 10)
			}
			if back.GetFps() > 0 {
				out["camera.back.fps"] = strconv.FormatInt(int64(back.GetFps()), 10)
			}
		}
		if endpoints := camera.GetEndpoints(); len(endpoints) > 0 {
			out["camera.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "camera.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				if facing := strings.TrimSpace(endpoint.GetFacing()); facing != "" {
					out[prefix+".facing"] = facing
				}
				out[prefix+".available"] = strconv.FormatBool(endpoint.GetAvailable())
				if modes := endpoint.GetModes(); len(modes) > 0 {
					for modeIndex, mode := range modes {
						modePrefix := prefix + ".mode." + strconv.Itoa(modeIndex)
						out[modePrefix+".width"] = strconv.FormatInt(int64(mode.GetWidth()), 10)
						out[modePrefix+".height"] = strconv.FormatInt(int64(mode.GetHeight()), 10)
						out[modePrefix+".fps"] = strconv.FormatInt(int64(mode.GetFps()), 10)
					}
				}
			}
		}
	}
	if sensors := caps.GetSensors(); sensors != nil {
		out["sensors.accelerometer"] = strconv.FormatBool(sensors.GetAccelerometer())
		out["sensors.gyroscope"] = strconv.FormatBool(sensors.GetGyroscope())
		out["sensors.compass"] = strconv.FormatBool(sensors.GetCompass())
		out["sensors.ambient_light"] = strconv.FormatBool(sensors.GetAmbientLight())
		out["sensors.proximity"] = strconv.FormatBool(sensors.GetProximity())
		out["sensors.gps"] = strconv.FormatBool(sensors.GetGps())
	}
	if connectivity := caps.GetConnectivity(); connectivity != nil {
		out["connectivity.bluetooth_version"] = connectivity.GetBluetoothVersion()
		out["connectivity.wifi_signal_strength"] = strconv.FormatBool(connectivity.GetWifiSignalStrength())
		out["connectivity.usb_host"] = strconv.FormatBool(connectivity.GetUsbHost())
		out["connectivity.usb_ports"] = strconv.FormatInt(int64(connectivity.GetUsbPorts()), 10)
		out["connectivity.nfc"] = strconv.FormatBool(connectivity.GetNfc())
	}
	if battery := caps.GetBattery(); battery != nil {
		out["battery.level"] = strconv.FormatFloat(float64(battery.GetLevel()), 'f', -1, 32)
		out["battery.charging"] = strconv.FormatBool(battery.GetCharging())
	}
	if edge := caps.GetEdge(); edge != nil {
		out["edge.runtimes"] = strings.Join(edge.GetRuntimes(), ",")
		operators := edge.GetOperators()
		out["edge.operators"] = strings.Join(operators, ",")
		foregroundOnly := false
		backgroundCapable := false
		for _, operator := range operators {
			normalized := strings.TrimSpace(strings.ToLower(operator))
			switch normalized {
			case "monitor.foreground_only", "monitor.tier.foreground_only":
				foregroundOnly = true
				out["monitor.foreground_only"] = "true"
				out["monitor.support_tier"] = "foreground_only"
			case "monitor.background_capable", "monitor.tier.background_capable":
				backgroundCapable = true
				out["monitor.background_capable"] = "true"
				out["monitor.support_tier"] = "background_capable"
			case "monitor.lifecycle.foreground":
				out["monitor.runtime_state"] = "foreground"
			case "monitor.lifecycle.background":
				out["monitor.runtime_state"] = "background"
			}
		}
		if foregroundOnly && !backgroundCapable {
			out["monitor.background_capable"] = "false"
		}
		if compute := edge.GetCompute(); compute != nil {
			out["edge.compute.cpu_realtime"] = strconv.FormatInt(int64(compute.GetCpuRealtime()), 10)
			out["edge.compute.gpu_realtime"] = strconv.FormatInt(int64(compute.GetGpuRealtime()), 10)
			out["edge.compute.npu_realtime"] = strconv.FormatInt(int64(compute.GetNpuRealtime()), 10)
			out["edge.compute.mem_mb"] = strconv.FormatInt(int64(compute.GetMemMb()), 10)
		}
		if retention := edge.GetRetention(); retention != nil {
			out["edge.retention.audio_sec"] = strconv.FormatInt(int64(retention.GetAudioSec()), 10)
			out["edge.retention.video_sec"] = strconv.FormatInt(int64(retention.GetVideoSec()), 10)
			out["edge.retention.sensor_sec"] = strconv.FormatInt(int64(retention.GetSensorSec()), 10)
			out["edge.retention.radio_sec"] = strconv.FormatInt(int64(retention.GetRadioSec()), 10)
		}
		if timing := edge.GetTiming(); timing != nil {
			out["edge.timing.sync_error_ms"] = strconv.FormatFloat(timing.GetSyncErrorMs(), 'f', -1, 64)
		}
		if geometry := edge.GetGeometry(); geometry != nil {
			out["edge.geometry.mic_array"] = strconv.FormatBool(geometry.GetMicArray())
			out["edge.geometry.camera_intrinsics"] = strconv.FormatBool(geometry.GetCameraIntrinsics())
			out["edge.geometry.compass"] = strconv.FormatBool(geometry.GetCompass())
		}
	}
	return out
}

func joinInts(values []int32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatInt(int64(v), 10))
	}
	return strings.Join(parts, ",")
}

func descriptorToUINode(d ui.Descriptor) *uiv1.Node {
	children := make([]*uiv1.Node, 0, len(d.Children))
	for _, child := range d.Children {
		children = append(children, descriptorToUINode(child))
	}

	props := make(map[string]string, len(d.Props)+1)
	for k, v := range d.Props {
		props[k] = v
	}
	if _, ok := props["type"]; !ok && d.Type != "" {
		props["type"] = d.Type
	}

	node := &uiv1.Node{
		Id:       d.ID,
		Props:    props,
		Children: children,
	}
	applyWidgetFromDescriptor(node, d.Type, d.Props)
	return node
}

func applyWidgetFromDescriptor(node *uiv1.Node, nodeType string, props map[string]string) {
	switch nodeType {
	case "stack":
		node.Widget = &uiv1.Node_Stack{Stack: &uiv1.StackWidget{}}
	case "row":
		node.Widget = &uiv1.Node_Row{Row: &uiv1.RowWidget{}}
	case "grid":
		node.Widget = &uiv1.Node_Grid{Grid: &uiv1.GridWidget{Columns: parseInt32(props["columns"])}}
	case "scroll":
		node.Widget = &uiv1.Node_Scroll{Scroll: &uiv1.ScrollWidget{Direction: props["direction"]}}
	case "padding":
		node.Widget = &uiv1.Node_Padding{Padding: &uiv1.PaddingWidget{All: parseInt32(props["all"])}}
	case "center":
		node.Widget = &uiv1.Node_Center{Center: &uiv1.CenterWidget{}}
	case "expand":
		node.Widget = &uiv1.Node_Expand{Expand: &uiv1.ExpandWidget{}}
	case "text":
		node.Widget = &uiv1.Node_Text{
			Text: &uiv1.TextWidget{
				Value: props["value"],
				Style: props["style"],
				Color: props["color"],
			},
		}
	case "image":
		node.Widget = &uiv1.Node_Image{Image: &uiv1.ImageWidget{Url: props["url"]}}
	case "video_surface":
		node.Widget = &uiv1.Node_VideoSurface{VideoSurface: &uiv1.VideoSurfaceWidget{TrackId: props["track_id"]}}
	case "audio_visualizer":
		node.Widget = &uiv1.Node_AudioVisualizer{AudioVisualizer: &uiv1.AudioVisualizerWidget{StreamId: props["stream_id"]}}
	case "canvas":
		node.Widget = &uiv1.Node_Canvas{Canvas: &uiv1.CanvasWidget{DrawOpsJson: props["draw_ops_json"]}}
	case "text_input":
		node.Widget = &uiv1.Node_TextInput{
			TextInput: &uiv1.TextInputWidget{
				Placeholder: props["placeholder"],
				Autofocus:   parseBool(props["autofocus"]),
			},
		}
	case "button":
		node.Widget = &uiv1.Node_Button{Button: &uiv1.ButtonWidget{Label: props["label"], Action: props["action"]}}
	case "slider":
		node.Widget = &uiv1.Node_Slider{
			Slider: &uiv1.SliderWidget{
				Min:   parseFloat64(props["min"]),
				Max:   parseFloat64(props["max"]),
				Value: parseFloat64(props["value"]),
			},
		}
	case "toggle":
		node.Widget = &uiv1.Node_Toggle{Toggle: &uiv1.ToggleWidget{Value: parseBool(props["value"])}}
	case "dropdown":
		node.Widget = &uiv1.Node_Dropdown{Dropdown: &uiv1.DropdownWidget{Value: props["value"]}}
	case "gesture_area":
		node.Widget = &uiv1.Node_GestureArea{GestureArea: &uiv1.GestureAreaWidget{Action: props["action"]}}
	case "overlay":
		node.Widget = &uiv1.Node_Overlay{Overlay: &uiv1.OverlayWidget{}}
	case "progress":
		node.Widget = &uiv1.Node_Progress{Progress: &uiv1.ProgressWidget{Value: parseFloat64(props["value"])}}
	case "fullscreen":
		node.Widget = &uiv1.Node_Fullscreen{Fullscreen: &uiv1.FullscreenWidget{Enabled: parseBool(props["enabled"])}}
	case "keep_awake":
		node.Widget = &uiv1.Node_KeepAwake{KeepAwake: &uiv1.KeepAwakeWidget{Enabled: parseBool(props["enabled"])}}
	case "brightness":
		node.Widget = &uiv1.Node_Brightness{Brightness: &uiv1.BrightnessWidget{Value: parseFloat64(props["value"])}}
	default:
		node.Widget = nil
	}
}

func parseInt32(raw string) int32 {
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}

func parseFloat64(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseBool(raw string) bool {
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return v
}

func observationFromProto(ob *iov1.Observation) iorouter.Observation {
	if ob == nil {
		return iorouter.Observation{}
	}
	out := iorouter.Observation{
		Kind:       ob.GetKind(),
		Subject:    ob.GetSubject(),
		OccurredAt: unixMSTime(ob.GetOccurredUnixMs()),
		Confidence: ob.GetConfidence(),
		Zone:       ob.GetZone(),
		TrackID:    ob.GetTrackId(),
		Attributes: cloneStringMapAdapter(ob.GetAttributes()),
		Provenance: iorouter.ObservationProvenance{
			FlowID:             ob.GetProvenance().GetFlowId(),
			NodeID:             ob.GetProvenance().GetNodeId(),
			ExecSite:           ob.GetProvenance().GetExecSite(),
			ModelID:            ob.GetProvenance().GetModelId(),
			CalibrationVersion: ob.GetProvenance().GetCalibrationVersion(),
		},
	}
	out.SourceDevice = iorouter.DeviceRef{DeviceID: ob.GetSourceDevice().GetDeviceId()}
	if loc := ob.GetLocation(); loc != nil {
		var pose *iorouter.Pose
		if p := loc.GetPose(); p != nil {
			pose = &iorouter.Pose{
				X:          p.GetX(),
				Y:          p.GetY(),
				Z:          p.GetZ(),
				Yaw:        p.GetYaw(),
				Pitch:      p.GetPitch(),
				Roll:       p.GetRoll(),
				Confidence: p.GetConfidence(),
			}
		}
		out.Location = &iorouter.LocationEstimate{
			Zone:       loc.GetZone(),
			Pose:       pose,
			RadiusM:    loc.GetRadiusM(),
			Confidence: loc.GetConfidence(),
			Sources:    append([]string(nil), loc.GetSources()...),
		}
	}
	for _, artifact := range ob.GetEvidence() {
		out.Evidence = append(out.Evidence, artifactFromProto(artifact))
	}
	return out
}

func artifactFromProto(artifact *iov1.ArtifactRef) iorouter.ArtifactRef {
	if artifact == nil {
		return iorouter.ArtifactRef{}
	}
	return iorouter.ArtifactRef{
		ID:        artifact.GetId(),
		Kind:      artifact.GetKind(),
		Source:    iorouter.DeviceRef{DeviceID: artifact.GetSource().GetDeviceId()},
		StartTime: unixMSTime(artifact.GetStartUnixMs()),
		EndTime:   unixMSTime(artifact.GetEndUnixMs()),
		URI:       artifact.GetUri(),
	}
}

func flowPlanToProto(plan iorouter.FlowPlan) *iov1.FlowPlan {
	nodes := make([]*iov1.FlowNode, 0, len(plan.Nodes))
	for _, node := range plan.Nodes {
		nodes = append(nodes, &iov1.FlowNode{
			Id:   node.ID,
			Kind: string(node.Kind),
			Args: cloneStringMapAdapter(node.Args),
			Exec: string(node.Exec),
		})
	}
	edges := make([]*iov1.FlowEdge, 0, len(plan.Edges))
	for _, edge := range plan.Edges {
		edges = append(edges, &iov1.FlowEdge{
			From: edge.From,
			To:   edge.To,
		})
	}
	return &iov1.FlowPlan{
		Nodes: nodes,
		Edges: edges,
	}
}

func unixMSTime(unixMS int64) time.Time {
	if unixMS <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(unixMS).UTC()
}

func cloneStringMapAdapter(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
