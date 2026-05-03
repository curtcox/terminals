package protocolcontract

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
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
		"start_stream_audio_v1":         assertStartStreamAudio,
		"flow_plan_basic_v1":            assertFlowPlanBasic,
		"observation_sound_v1":          assertObservationSound,
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

func assertStartStreamAudio(t *testing.T, envelope *controlv1.WireEnvelope) {
	t.Helper()
	stream := envelope.GetServerMessage().GetStartStream()
	if stream.GetKind() != "audio" {
		t.Fatalf("stream kind = %q", stream.GetKind())
	}
	if stream.GetMetadata()["sample_rate"] != "16000" {
		t.Fatalf("sample_rate metadata = %q", stream.GetMetadata()["sample_rate"])
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
	if len(observation.GetEvidence()) != 1 {
		t.Fatalf("evidence count = %d", len(observation.GetEvidence()))
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
