package transport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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
		if pointer := input.GetPointer(); pointer != nil {
			internal.Action = internalPointerActionFromProto(pointer.GetAction(), pointer.GetActionEnum())
		}
		if touch := input.GetTouch(); touch != nil {
			internal.Action = internalTouchActionFromProto(touch.GetAction(), touch.GetActionEnum())
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
					Invalidations:      capabilityInvalidationsToProto(msg.CapabilityAck.Invalidations),
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
					ServerMetadata: &controlv1.ServerMetadata{
						Build: &controlv1.BuildMetadata{
							Sha:         msg.RegisterAck.ServerMetadata.Build.SHA,
							DateRfc3339: msg.RegisterAck.ServerMetadata.Build.DateRFC3339,
						},
						PhotoFrameAssetBaseUrl: msg.RegisterAck.ServerMetadata.PhotoFrameAssetBaseURL,
					},
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
		routing := msg.StartStream.Routing
		if routing == nil {
			routing = streamRoutingFromMetadata(msg.StartStream.Metadata)
		}
		audioMetadata := msg.StartStream.AudioMetadata
		if audioMetadata == nil {
			audioMetadata = streamAudioMetadataFromLegacy(msg.StartStream.Metadata)
		}
		metadata := mergeLegacyRoutingMetadata(msg.StartStream.Metadata, routing)
		metadata = mergeLegacyAudioMetadata(metadata, audioMetadata)
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_StartStream{
				StartStream: &iov1.StartStream{
					StreamId:       msg.StartStream.StreamID,
					Kind:           msg.StartStream.Kind,
					SourceDeviceId: msg.StartStream.SourceDeviceID,
					TargetDeviceId: msg.StartStream.TargetDeviceID,
					Metadata:       metadata,
					StreamKind:     protoStreamKindFromInternal(msg.StartStream.Kind),
					Routing:        routing,
					AudioMetadata:  audioMetadata,
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
					StreamKind:     protoStreamKindFromInternal(msg.RouteStream.Kind),
					Routing:        msg.RouteStream.Routing,
				},
			},
		}
	case msg.WebRTCSignal != nil:
		return &controlv1.ConnectResponse{
			Payload: &controlv1.ConnectResponse_WebrtcSignal{
				WebrtcSignal: &controlv1.WebRTCSignal{
					StreamId:       msg.WebRTCSignal.StreamID,
					SignalType:     msg.WebRTCSignal.SignalType,
					Payload:        msg.WebRTCSignal.Payload,
					SignalTypeEnum: protoWebRTCSignalTypeFromInternal(msg.WebRTCSignal.SignalType),
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
					TypedData:     commandResultDataEntriesFromMap(msg.Data),
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

func capabilityInvalidationsToProto(in []CapabilityInvalidation) []*controlv1.ResourceInvalidation {
	if len(in) == 0 {
		return nil
	}
	out := make([]*controlv1.ResourceInvalidation, 0, len(in))
	for _, invalidation := range in {
		out = append(out, &controlv1.ResourceInvalidation{
			Resource: invalidation.Resource,
			Reason:   invalidation.Reason,
		})
	}
	return out
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

func internalPointerActionFromProto(legacy string, action iov1.PointerAction) string {
	switch action {
	case iov1.PointerAction_POINTER_ACTION_UNSPECIFIED:
		return strings.ToLower(strings.TrimSpace(legacy))
	case iov1.PointerAction_POINTER_ACTION_DOWN:
		return "down"
	case iov1.PointerAction_POINTER_ACTION_MOVE:
		return "move"
	case iov1.PointerAction_POINTER_ACTION_UP:
		return "up"
	case iov1.PointerAction_POINTER_ACTION_CANCEL:
		return "cancel"
	case iov1.PointerAction_POINTER_ACTION_SCROLL:
		return "scroll"
	default:
		return strings.ToLower(strings.TrimSpace(legacy))
	}
}

func internalTouchActionFromProto(legacy string, action iov1.TouchAction) string {
	switch action {
	case iov1.TouchAction_TOUCH_ACTION_UNSPECIFIED:
		return strings.ToLower(strings.TrimSpace(legacy))
	case iov1.TouchAction_TOUCH_ACTION_START:
		return "start"
	case iov1.TouchAction_TOUCH_ACTION_MOVE:
		return "move"
	case iov1.TouchAction_TOUCH_ACTION_END:
		return "end"
	case iov1.TouchAction_TOUCH_ACTION_CANCEL:
		return "cancel"
	default:
		return strings.ToLower(strings.TrimSpace(legacy))
	}
}

func commandArgumentsToInternalMap(
	typed []*controlv1.CommandArgumentEntry,
	legacy map[string]string,
) map[string]string {
	resolved := copyStringMap(legacy)
	for _, entry := range typed {
		if entry == nil {
			continue
		}
		key := strings.TrimSpace(entry.GetKey())
		if key == "" {
			continue
		}
		value, ok := commandTypedValueToString(entry.GetValue())
		if !ok {
			continue
		}
		resolved[key] = value
	}
	return resolved
}

func commandTypedValueToString(value *controlv1.CommandTypedValue) (string, bool) {
	if value == nil {
		return "", false
	}
	switch kind := value.GetKind().(type) {
	case *controlv1.CommandTypedValue_StringValue:
		return kind.StringValue, true
	case *controlv1.CommandTypedValue_Int64Value:
		return strconv.FormatInt(kind.Int64Value, 10), true
	case *controlv1.CommandTypedValue_BoolValue:
		return strconv.FormatBool(kind.BoolValue), true
	case *controlv1.CommandTypedValue_DoubleValue:
		return strconv.FormatFloat(kind.DoubleValue, 'f', -1, 64), true
	case *controlv1.CommandTypedValue_StringListValue:
		if kind.StringListValue == nil {
			return "", true
		}
		return strings.Join(kind.StringListValue.GetValues(), ","), true
	default:
		return "", false
	}
}

func commandResultDataEntriesFromMap(data map[string]string) []*controlv1.CommandResultDataEntry {
	if len(data) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*controlv1.CommandResultDataEntry, 0, len(data))
	for _, key := range keys {
		raw := data[key]
		entry := &controlv1.CommandResultDataEntry{Key: key}
		entry.Value = commandTypedValueFromLegacyString(key, raw)
		out = append(out, entry)
	}
	return out
}

func commandTypedValueFromLegacyString(key, raw string) *controlv1.CommandTypedValue {
	trimmed := strings.TrimSpace(raw)
	if shouldUseStringListValue(key, trimmed) {
		parts := strings.Split(trimmed, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			item := strings.TrimSpace(part)
			if item != "" {
				values = append(values, item)
			}
		}
		if len(values) > 0 {
			return &controlv1.CommandTypedValue{
				Kind: &controlv1.CommandTypedValue_StringListValue{
					StringListValue: &controlv1.CommandStringList{Values: values},
				},
			}
		}
	}
	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return &controlv1.CommandTypedValue{
			Kind: &controlv1.CommandTypedValue_Int64Value{Int64Value: parsed},
		}
	}
	switch strings.ToLower(trimmed) {
	case "true":
		return &controlv1.CommandTypedValue{
			Kind: &controlv1.CommandTypedValue_BoolValue{BoolValue: true},
		}
	case "false":
		return &controlv1.CommandTypedValue{
			Kind: &controlv1.CommandTypedValue_BoolValue{BoolValue: false},
		}
	}
	if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil && strings.Contains(trimmed, ".") {
		return &controlv1.CommandTypedValue{
			Kind: &controlv1.CommandTypedValue_DoubleValue{DoubleValue: parsed},
		}
	}
	return &controlv1.CommandTypedValue{
		Kind: &controlv1.CommandTypedValue_StringValue{StringValue: raw},
	}
}

func shouldUseStringListValue(key, value string) bool {
	if value == "" || strings.Contains(value, "|") || !strings.Contains(value, ",") {
		return false
	}
	normalizedKey := strings.TrimSpace(strings.ToLower(key))
	return normalizedKey == "device_ids" ||
		normalizedKey == "system_intents" ||
		normalizedKey == "command_kinds" ||
		normalizedKey == "command_actions" ||
		normalizedKey == "sensor_device_ids" ||
		normalizedKey == "recording_stream_ids"
}

func internalWebRTCSignalTypeFromProto(legacy string, signalType controlv1.WebRTCSignalType) string {
	switch signalType {
	case controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_UNSPECIFIED:
		return legacy
	case controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_OFFER:
		return "offer"
	case controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_ANSWER:
		return "answer"
	case controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_ICE_CANDIDATE:
		return "candidate"
	default:
		return legacy
	}
}

func protoWebRTCSignalTypeFromInternal(signalType string) controlv1.WebRTCSignalType {
	switch strings.ToLower(strings.TrimSpace(signalType)) {
	case "offer":
		return controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_OFFER
	case "answer":
		return controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_ANSWER
	case "candidate", "ice_candidate":
		return controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_ICE_CANDIDATE
	default:
		return controlv1.WebRTCSignalType_WEB_RTC_SIGNAL_TYPE_UNSPECIFIED
	}
}

func protoStreamKindFromInternal(kind string) iov1.StreamKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "audio":
		return iov1.StreamKind_STREAM_KIND_AUDIO
	case "video":
		return iov1.StreamKind_STREAM_KIND_VIDEO
	case "sensor":
		return iov1.StreamKind_STREAM_KIND_SENSOR
	case "data":
		return iov1.StreamKind_STREAM_KIND_DATA
	default:
		return iov1.StreamKind_STREAM_KIND_UNSPECIFIED
	}
}

func flowNodeTypedArgsFromArgs(args map[string]string) *iov1.FlowNodeArgs {
	if len(args) == 0 {
		return nil
	}
	deviceID := strings.TrimSpace(args["device_id"])
	resource := strings.TrimSpace(args["resource"])
	streamKind := strings.TrimSpace(args["stream_kind"])
	name := strings.TrimSpace(args["name"])
	if deviceID == "" && resource == "" && streamKind == "" && name == "" {
		return nil
	}
	return &iov1.FlowNodeArgs{
		DeviceId:       deviceID,
		Resource:       resource,
		StreamKind:     streamKind,
		StreamKindEnum: protoStreamKindFromInternal(streamKind),
		Name:           name,
	}
}

func protoExecPolicyFromInternal(exec iorouter.ExecPolicy) iov1.ExecPolicy {
	switch exec {
	case iorouter.ExecAuto:
		return iov1.ExecPolicy_EXEC_POLICY_AUTO
	case iorouter.ExecPreferClient:
		return iov1.ExecPolicy_EXEC_POLICY_PREFER_CLIENT
	case iorouter.ExecRequireClient:
		return iov1.ExecPolicy_EXEC_POLICY_REQUIRE_CLIENT
	case iorouter.ExecServerOnly:
		return iov1.ExecPolicy_EXEC_POLICY_SERVER_ONLY
	default:
		return iov1.ExecPolicy_EXEC_POLICY_UNSPECIFIED
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
	applyWidgetFromDescriptor(node, d)
	return node
}

func applyWidgetFromDescriptor(node *uiv1.Node, d ui.Descriptor) {
	nodeType := d.Type
	props := d.Props
	switch nodeType {
	case "stack":
		node.Widget = &uiv1.Node_Stack{Stack: &uiv1.StackWidget{}}
	case "row":
		node.Widget = &uiv1.Node_Row{Row: &uiv1.RowWidget{}}
	case "grid":
		node.Widget = &uiv1.Node_Grid{Grid: &uiv1.GridWidget{Columns: parseInt32(props["columns"])}}
	case "scroll":
		direction := props["direction"]
		node.Widget = &uiv1.Node_Scroll{Scroll: &uiv1.ScrollWidget{
			Direction:     direction,
			DirectionEnum: scrollDirectionFromString(direction),
		}}
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
		drawOpsJSON := props["draw_ops_json"]
		var typedOps []*uiv1.DrawOp
		if len(d.CanvasOps) > 0 {
			typedOps = canvasDrawOpsFromUI(d.CanvasOps)
			if drawOpsJSON == "" {
				drawOpsJSON = ui.CanvasOpsToJSON(d.CanvasOps)
			}
		} else {
			typedOps = canvasDrawOpsFromJSON(drawOpsJSON)
		}
		node.Widget = &uiv1.Node_Canvas{Canvas: &uiv1.CanvasWidget{
			DrawOpsJson: drawOpsJSON,
			DrawOps:     typedOps,
		}}
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

// observationAttributesFromProto resolves Observation.attributes preferring
// the typed mirror message when populated, then merging in any legacy map
// keys not already covered by typed fields. This implements the typed-first
// fallback policy from plans/features/protocol/evolution-rules.md.
func observationAttributesFromProto(ob *iov1.Observation) map[string]string {
	legacy := ob.GetAttributes()
	typed := ob.GetTypedAttributes()
	if typed == nil {
		return cloneStringMapAdapter(legacy)
	}
	out := make(map[string]string, len(legacy)+4)
	for k, v := range legacy {
		out[k] = v
	}
	if v := typed.GetLabel(); v != "" {
		out["label"] = v
	}
	if v := typed.GetDevice(); v != "" {
		out["device"] = v
	}
	if v := typed.GetMac(); v != "" {
		out["mac"] = v
	}
	if v := typed.GetDurationSeconds(); v != "" {
		out["duration_seconds"] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// observationTypedAttributesFromInternal builds the typed mirror message from
// an internal Attributes map. Returns nil if no known typed keys are present.
func observationTypedAttributesFromInternal(attrs map[string]string) *iov1.ObservationAttributes {
	if len(attrs) == 0 {
		return nil
	}
	label := strings.TrimSpace(attrs["label"])
	device := strings.TrimSpace(attrs["device"])
	mac := strings.TrimSpace(attrs["mac"])
	duration := strings.TrimSpace(attrs["duration_seconds"])
	if label == "" && device == "" && mac == "" && duration == "" {
		return nil
	}
	return &iov1.ObservationAttributes{
		Label:           label,
		Device:          device,
		Mac:             mac,
		DurationSeconds: duration,
	}
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
		Attributes: observationAttributesFromProto(ob),
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
			Id:         node.ID,
			Kind:       string(node.Kind),
			Args:       cloneStringMapAdapter(node.Args),
			Exec:       string(node.Exec),
			ExecPolicy: protoExecPolicyFromInternal(node.Exec),
			TypedArgs:  flowNodeTypedArgsFromArgs(node.Args),
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

func scrollDirectionFromString(value string) uiv1.ScrollDirection {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "horizontal":
		return uiv1.ScrollDirection_SCROLL_DIRECTION_HORIZONTAL
	case "vertical":
		return uiv1.ScrollDirection_SCROLL_DIRECTION_VERTICAL
	default:
		return uiv1.ScrollDirection_SCROLL_DIRECTION_UNSPECIFIED
	}
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
