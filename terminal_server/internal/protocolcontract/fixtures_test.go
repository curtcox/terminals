package protocolcontract

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var updateFixtures = flag.Bool("update", false, "regenerate binary protocol contract fixtures")

func TestGoldenWireEnvelopeFixtures(t *testing.T) {
	cases := map[string]func(t *testing.T, envelope *controlv1.WireEnvelope){
		"hello_snapshot_v1":             assertHelloSnapshot,
		"capability_snapshot_v1":        assertCapabilitySnapshot,
		"register_ack_metadata_v1":      assertRegisterAckMetadata,
		"set_ui_basic_v1":               assertSetUIBasic,
		"set_ui_canvas_v1":              assertSetUICanvas,
		"start_stream_audio_v1":         assertStartStreamAudio,
		"start_stream_route_delta_v1":   assertStartStreamRouteDelta,
		"route_stream_route_delta_v1":   assertRouteStreamRouteDelta,
		"flow_plan_basic_v1":            assertFlowPlanBasic,
		"command_result_typed_data_v1":  assertCommandResultTypedData,
		"observation_sound_v1":          assertObservationSound,
		"flow_stats_v1":                 assertFlowStats,
		"unknown_metadata_key_v1":       assertUnknownMetadataKey,
		"deprecated_register_device_v1": assertDeprecatedRegisterDevice,
	}

	for name, assert := range cases {
		t.Run(name, func(t *testing.T) {
			textEnvelope := readTextFixture(t, name)
			if *updateFixtures {
				writeBinaryFixture(t, name, textEnvelope)
			}
			binaryEnvelope := readBinaryFixture(t, name)
			if !proto.Equal(textEnvelope, binaryEnvelope) {
				t.Fatalf("%s textproto and binpb differ:\ntext=%v\nbinary=%v", name, textEnvelope, binaryEnvelope)
			}
			roundTrip, err := proto.Marshal(binaryEnvelope)
			if err != nil {
				t.Fatalf("marshal %s: %v", name, err)
			}
			var decoded controlv1.WireEnvelope
			if err := proto.Unmarshal(roundTrip, &decoded); err != nil {
				t.Fatalf("round-trip decode %s: %v", name, err)
			}
			if !proto.Equal(binaryEnvelope, &decoded) {
				t.Fatalf("%s changed after binary round trip", name)
			}
			assert(t, binaryEnvelope)
		})
	}
}

func readTextFixture(t *testing.T, name string) *controlv1.WireEnvelope {
	t.Helper()
	data, err := os.ReadFile(fixturePath(name + ".textproto"))
	if err != nil {
		t.Fatalf("read text fixture %s: %v", name, err)
	}
	var envelope controlv1.WireEnvelope
	if err := prototext.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("parse text fixture %s: %v", name, err)
	}
	return &envelope
}

func writeBinaryFixture(t *testing.T, name string, envelope *controlv1.WireEnvelope) {
	t.Helper()
	data, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal fixture %s: %v", name, err)
	}
	if err := os.WriteFile(fixturePath(name+".binpb"), data, 0o644); err != nil {
		t.Fatalf("write binary fixture %s: %v", name, err)
	}
}

func readBinaryFixture(t *testing.T, name string) *controlv1.WireEnvelope {
	t.Helper()
	data, err := os.ReadFile(fixturePath(name + ".binpb"))
	if err != nil {
		t.Fatalf("read binary fixture %s: %v", name, err)
	}
	var envelope controlv1.WireEnvelope
	if err := proto.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("parse binary fixture %s: %v", name, err)
	}
	return &envelope
}

func fixturePath(name string) string {
	return filepath.Join("..", "..", "..", "api", "testdata", "envelopes", name)
}

func assertHelloSnapshot(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	hello := envelope.GetClientMessage().GetHello()
	if hello.GetDeviceId() != "terminal-kitchen" {
		t.Fatalf("device id = %q", hello.GetDeviceId())
	}
	if hello.GetIdentity().GetPlatform() != "flutter_test" {
		t.Fatalf("platform = %q", hello.GetIdentity().GetPlatform())
	}
}

func assertCapabilitySnapshot(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	snapshot := envelope.GetClientMessage().GetCapabilitySnapshot()
	if snapshot.GetGeneration() != 7 {
		t.Fatalf("generation = %d", snapshot.GetGeneration())
	}
	if snapshot.GetCapabilities().GetPointer().GetType() != "unknown_pointer_from_future" {
		t.Fatalf("pointer type was not preserved")
	}
}

func assertRegisterAckMetadata(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	ack := envelope.GetServerMessage().GetRegisterAck()
	metadata := ack.GetMetadata()
	if metadata["server_build_sha"] != "abc1234" {
		t.Fatalf("server_build_sha = %q", metadata["server_build_sha"])
	}
	if metadata["photo_frame_asset_base_url"] == "" {
		t.Fatalf("asset base URL missing")
	}
	if got := ack.GetServerMetadata().GetBuild().GetSha(); got != "abc1234" {
		t.Fatalf("server_metadata.build.sha = %q", got)
	}
	if got := ack.GetServerMetadata().GetBuild().GetDateRfc3339(); got != "2026-05-03T14:00:00Z" {
		t.Fatalf("server_metadata.build.date_rfc3339 = %q", got)
	}
	if got := ack.GetServerMetadata().GetPhotoFrameAssetBaseUrl(); got == "" {
		t.Fatalf("server_metadata.photo_frame_asset_base_url missing")
	}
}

func assertSetUIBasic(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	root := envelope.GetServerMessage().GetSetUi().GetRoot()
	if root.GetId() != "root" || len(root.GetChildren()) != 2 {
		t.Fatalf("root = %+v", root)
	}
	if root.GetChildren()[0].GetText().GetStyle() != "title" {
		t.Fatalf("text style not preserved")
	}
}

func assertSetUICanvas(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	root := envelope.GetServerMessage().GetSetUi().GetRoot()
	if len(root.GetChildren()) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.GetChildren()))
	}
	canvas := root.GetChildren()[0].GetCanvas()
	if canvas == nil {
		t.Fatalf("canvas widget missing on first child")
	}
	if canvas.GetDrawOpsJson() == "" {
		t.Fatalf("legacy draw_ops_json empty; typed-vs-legacy coexistence requires both surfaces")
	}
	ops := canvas.GetDrawOps()
	if len(ops) != 2 {
		t.Fatalf("typed draw_ops len = %d, want 2", len(ops))
	}
	if rect := ops[0].GetRect(); rect == nil || rect.GetFill() != "#abc" || rect.GetWidth() != 3 {
		t.Fatalf("ops[0] = %+v, want rect fill=#abc width=3", ops[0])
	}
	if line := ops[1].GetLine(); line == nil || line.GetStroke() != "#000" || line.GetStrokeWidth() != 1.5 {
		t.Fatalf("ops[1] = %+v, want line stroke=#000 stroke_width=1.5", ops[1])
	}
}

func assertStartStreamAudio(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	stream := envelope.GetServerMessage().GetStartStream()
	if stream.GetKind() != "audio" {
		t.Fatalf("stream kind = %q", stream.GetKind())
	}
	if stream.GetStreamKind() != iov1.StreamKind_STREAM_KIND_AUDIO {
		t.Fatalf("stream stream_kind = %v", stream.GetStreamKind())
	}
	audio := stream.GetAudioMetadata()
	if audio == nil {
		t.Fatalf("typed audio metadata missing")
	}
	if got := audio.GetSampleRate(); got != 16000 {
		t.Fatalf("typed sample_rate = %d", got)
	}
	if got := audio.GetChannels(); got != 1 {
		t.Fatalf("typed channels = %d", got)
	}
	if got := audio.GetCodec(); got != "pcm_s16le" {
		t.Fatalf("typed codec = %q", got)
	}
	if stream.GetMetadata()["sample_rate"] != "16000" {
		t.Fatalf("sample_rate metadata = %q", stream.GetMetadata()["sample_rate"])
	}
	if stream.GetMetadata()["channels"] != "1" {
		t.Fatalf("channels metadata = %q", stream.GetMetadata()["channels"])
	}
	if stream.GetMetadata()["codec"] != "pcm_s16le" {
		t.Fatalf("codec metadata = %q", stream.GetMetadata()["codec"])
	}
}

func assertStartStreamRouteDelta(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	stream := envelope.GetServerMessage().GetStartStream()
	if stream.GetMetadata()["origin"] != "route_delta" {
		t.Fatalf("legacy origin = %q", stream.GetMetadata()["origin"])
	}
	if stream.GetMetadata()["webrtc_mode"] != "server_managed" {
		t.Fatalf("legacy webrtc_mode = %q", stream.GetMetadata()["webrtc_mode"])
	}
	routing := stream.GetRouting()
	if routing == nil {
		t.Fatalf("typed routing missing")
	}
	if got := routing.GetOrigin(); got != iov1.StreamOrigin_STREAM_ORIGIN_ROUTE_DELTA {
		t.Fatalf("typed routing origin = %v", got)
	}
	if got := routing.GetWebrtcMode(); got != iov1.WebRTCMode_WEB_RTC_MODE_SERVER_MANAGED {
		t.Fatalf("typed routing webrtc_mode = %v", got)
	}
}

func assertRouteStreamRouteDelta(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	route := envelope.GetServerMessage().GetRouteStream()
	if route.GetKind() != "audio" {
		t.Fatalf("legacy kind = %q", route.GetKind())
	}
	if got := route.GetStreamKind(); got != iov1.StreamKind_STREAM_KIND_AUDIO {
		t.Fatalf("typed stream_kind = %v", got)
	}
	routing := route.GetRouting()
	if routing == nil {
		t.Fatalf("typed routing missing")
	}
	if got := routing.GetOrigin(); got != iov1.StreamOrigin_STREAM_ORIGIN_ROUTE_DELTA {
		t.Fatalf("typed routing origin = %v", got)
	}
	if got := routing.GetWebrtcMode(); got != iov1.WebRTCMode_WEB_RTC_MODE_SERVER_MANAGED {
		t.Fatalf("typed routing webrtc_mode = %v", got)
	}
}

func assertFlowPlanBasic(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	plan := envelope.GetServerMessage().GetStartFlow().GetPlan()
	if len(plan.GetNodes()) != 2 || len(plan.GetEdges()) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.GetNodes()[0].GetArgs()["stream_id"] != "stream-audio-1" {
		t.Fatalf("flow args not preserved")
	}
	if got := plan.GetNodes()[0].GetExec(); got != "edge" {
		t.Fatalf("legacy exec[0] = %q", got)
	}
	if got := plan.GetNodes()[0].GetExecPolicy(); got != iov1.ExecPolicy_EXEC_POLICY_PREFER_CLIENT {
		t.Fatalf("typed exec_policy[0] = %v", got)
	}
	if got := plan.GetNodes()[1].GetExec(); got != "server" {
		t.Fatalf("legacy exec[1] = %q", got)
	}
	if got := plan.GetNodes()[1].GetExecPolicy(); got != iov1.ExecPolicy_EXEC_POLICY_SERVER_ONLY {
		t.Fatalf("typed exec_policy[1] = %v", got)
	}
	capture := plan.GetNodes()[0]
	if got := capture.GetTypedArgs().GetDeviceId(); got != "kitchen-terminal" {
		t.Fatalf("typed_args.device_id = %q", got)
	}
	if got := capture.GetTypedArgs().GetResource(); got != "microphone" {
		t.Fatalf("typed_args.resource = %q", got)
	}
	if got := capture.GetTypedArgs().GetStreamKind(); got != "audio" {
		t.Fatalf("typed_args.stream_kind = %q", got)
	}
	if got := capture.GetTypedArgs().GetStreamKindEnum(); got != iov1.StreamKind_STREAM_KIND_AUDIO {
		t.Fatalf("typed_args.stream_kind_enum = %v", got)
	}
	if got := capture.GetArgs()["device_id"]; got != "kitchen-terminal" {
		t.Fatalf("legacy args[device_id] = %q", got)
	}
}

func assertCommandResultTypedData(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	result := envelope.GetServerMessage().GetCommandResult()
	if result.GetRequestId() != "runtime-status-1" {
		t.Fatalf("request_id = %q", result.GetRequestId())
	}
	if result.GetData()["processed"] != "legacy-processed" {
		t.Fatalf("legacy processed data = %q", result.GetData()["processed"])
	}
	typed := map[string]*controlv1.CommandTypedValue{}
	for _, entry := range result.GetTypedData() {
		typed[entry.GetKey()] = entry.GetValue()
	}
	if got := typed["processed"].GetInt64Value(); got != 3 {
		t.Fatalf("typed processed = %d, want 3", got)
	}
	if got := typed["ok"].GetBoolValue(); !got {
		t.Fatalf("typed ok = %v, want true", got)
	}
	if got := typed["command_kinds"].GetStringListValue().GetValues(); len(got) != 2 || got[0] != "voice" || got[1] != "manual" {
		t.Fatalf("typed command_kinds = %v, want [voice manual]", got)
	}
	if got := typed["detail"].GetStringValue(); got != "typed values win" {
		t.Fatalf("typed detail = %q", got)
	}
}

func assertObservationSound(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	observation := envelope.GetClientMessage().GetObservationMessage().GetObservation()
	if observation.GetKind() != "sound.detected" {
		t.Fatalf("observation kind = %q", observation.GetKind())
	}
	if observation.GetAttributes()["loudness_db"] != "72.5" {
		t.Fatalf("attributes not preserved")
	}
	if got := observation.GetTypedAttributes().GetLabel(); got != "whistle" {
		t.Fatalf("typed_attributes.label = %q", got)
	}
	if got := observation.GetTypedAttributes().GetDevice(); got != "kettle" {
		t.Fatalf("typed_attributes.device = %q", got)
	}
	if got := observation.GetAttributes()["label"]; got != "whistle" {
		t.Fatalf("legacy attributes[label] = %q", got)
	}
	if len(observation.GetEvidence()) != 1 {
		t.Fatalf("evidence count = %d", len(observation.GetEvidence()))
	}
}

func assertFlowStats(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	stats := envelope.GetClientMessage().GetFlowStats()
	if stats.GetFlowId() != "flow-edge-1" {
		t.Fatalf("flow_id = %q", stats.GetFlowId())
	}
	if stats.GetState() != "running" {
		t.Fatalf("legacy state = %q", stats.GetState())
	}
	if stats.GetStateEnum() != iov1.FlowState_FLOW_STATE_RUNNING {
		t.Fatalf("state_enum = %v", stats.GetStateEnum())
	}
}

func assertUnknownMetadataKey(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	metadata := envelope.GetServerMessage().GetRegisterAck().GetMetadata()
	if metadata["future.experimental_key"] != "preserve-but-ignore" {
		t.Fatalf("unknown metadata key was not preserved: %+v", metadata)
	}
}

func assertDeprecatedRegisterDevice(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	register := envelope.GetClientMessage().GetRegister()
	if register.GetCapabilities().GetDeviceId() != "legacy-terminal" {
		t.Fatalf("legacy register payload not decodable")
	}
}
