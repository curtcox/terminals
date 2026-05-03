package transport

import (
	"bytes"
	"context"
	"testing"
)

type voicePipelinePublisherStub struct {
	deviceID string
	chunks   [][]byte
}

func (p *voicePipelinePublisherStub) Publish(deviceID string, chunk []byte) {
	p.deviceID = deviceID
	p.chunks = append(p.chunks, append([]byte(nil), chunk...))
}

func TestVoicePipelineBuffersPartialAudioAndClearsOnFinal(t *testing.T) {
	pipeline := NewVoicePipeline(NewStreamHandler(nil))

	first, _ := pipeline.captureChunk("device-1", []byte("hello "), false)
	if !bytes.Equal(first, []byte("hello ")) {
		t.Fatalf("first buffer = %q, want hello", first)
	}
	if got := string(pipeline.buffers["device-1"]); got != "hello " {
		t.Fatalf("stored partial buffer = %q, want hello", got)
	}

	final, _ := pipeline.captureChunk("device-1", []byte("world"), true)
	if !bytes.Equal(final, []byte("hello world")) {
		t.Fatalf("final buffer = %q, want hello world", final)
	}
	if _, ok := pipeline.buffers["device-1"]; ok {
		t.Fatal("final chunk did not clear device buffer")
	}
}

func TestVoicePipelineCopiesBufferedChunks(t *testing.T) {
	pipeline := NewVoicePipeline(NewStreamHandler(nil))
	chunk := []byte("abc")

	pipeline.captureChunk("device-1", chunk, false)
	chunk[0] = 'z'

	if got := string(pipeline.buffers["device-1"]); got != "abc" {
		t.Fatalf("stored buffer = %q, want abc", got)
	}
}

func TestVoicePipelinePublishesOnlyNonEmptyChunks(t *testing.T) {
	pipeline := NewVoicePipeline(NewStreamHandler(nil))
	publisher := &voicePipelinePublisherStub{}
	pipeline.SetDeviceAudioPublisher(publisher)

	if _, err := pipeline.HandleAudio(context.Background(), &VoiceAudioRequest{
		DeviceID: "device-1",
		Audio:    []byte("abc"),
		IsFinal:  false,
	}); err != nil {
		t.Fatalf("HandleAudio(non-final) error = %v", err)
	}
	if _, err := pipeline.HandleAudio(context.Background(), &VoiceAudioRequest{
		DeviceID: "device-1",
		IsFinal:  false,
	}); err != nil {
		t.Fatalf("HandleAudio(empty non-final) error = %v", err)
	}

	if publisher.deviceID != "device-1" {
		t.Fatalf("published device = %q, want device-1", publisher.deviceID)
	}
	if len(publisher.chunks) != 1 || !bytes.Equal(publisher.chunks[0], []byte("abc")) {
		t.Fatalf("published chunks = %+v, want one abc chunk", publisher.chunks)
	}
}

func TestVoicePipelineFinalClearsBufferBeforeRuntimeError(t *testing.T) {
	pipeline := NewVoicePipeline(NewStreamHandler(nil))
	pipeline.captureChunk("device-1", []byte("partial"), false)

	_, err := pipeline.HandleAudio(context.Background(), &VoiceAudioRequest{
		DeviceID: "device-1",
		Audio:    []byte("-final"),
		IsFinal:  true,
	})
	if err == nil || err.Error() != "scenario runtime not configured" {
		t.Fatalf("HandleAudio(final) error = %v, want scenario runtime not configured", err)
	}
	if _, ok := pipeline.buffers["device-1"]; ok {
		t.Fatal("final chunk did not clear device buffer after runtime error")
	}
}
