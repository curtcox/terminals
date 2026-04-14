package ai

import (
	"bytes"
	"context"
	"image"
	"io"
	"testing"
)

func TestNoopBackendQueryReturnsSentinel(t *testing.T) {
	resp, err := NoopBackend{}.Query(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if resp != noopSentinel {
		t.Fatalf("Query() = %q, want %q", resp, noopSentinel)
	}
}

func TestNoopSpeechToTextDrainsAudioAndClosesChannel(t *testing.T) {
	audio := bytes.NewBufferString("pcm-bytes")
	ch, err := NoopSpeechToText{}.Transcribe(context.Background(), audio, STTOptions{Language: "en"})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("transcripts = %d, want 0", count)
	}
	if audio.Len() != 0 {
		t.Fatalf("audio buffer not drained; remaining = %d", audio.Len())
	}
}

func TestNoopTextToSpeechReturnsEmptyReader(t *testing.T) {
	r, err := NoopTextToSpeech{}.Synthesize(context.Background(), "hello", TTSOptions{})
	if err != nil {
		t.Fatalf("Synthesize() error = %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read synthesized audio: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("synthesized audio length = %d, want 0", len(got))
	}
}

func TestNoopLLMReturnsSentinelResponse(t *testing.T) {
	resp, err := NoopLLM{}.Query(
		context.Background(),
		[]Message{{Role: "user", Content: "anything"}},
		LLMOptions{},
	)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if resp == nil {
		t.Fatalf("Query() resp = nil, want non-nil")
	}
	if resp.Text != noopSentinel {
		t.Fatalf("resp.Text = %q, want %q", resp.Text, noopSentinel)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("resp.FinishReason = %q, want stop", resp.FinishReason)
	}
}

func TestNoopVisionAnalyzerReturnsSentinelCaption(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 1, 1))
	resp, err := NoopVisionAnalyzer{}.Analyze(context.Background(), frame, "describe")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if resp == nil || resp.Caption != noopSentinel {
		t.Fatalf("Analyze() resp = %+v, want caption %q", resp, noopSentinel)
	}
}

func TestNoopSoundClassifierDrainsAudioAndClosesChannel(t *testing.T) {
	audio := bytes.NewBufferString("pcm-bytes")
	ch, err := NoopSoundClassifier{}.Classify(context.Background(), audio)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("events = %d, want 0", count)
	}
	if audio.Len() != 0 {
		t.Fatalf("audio buffer not drained; remaining = %d", audio.Len())
	}
}

func TestNewNoopBackendsExposesEveryCapability(t *testing.T) {
	b := NewNoopBackends()
	if b.STT == nil {
		t.Fatalf("STT not set")
	}
	if b.TTS == nil {
		t.Fatalf("TTS not set")
	}
	if b.LLM == nil {
		t.Fatalf("LLM not set")
	}
	if b.Vision == nil {
		t.Fatalf("Vision not set")
	}
	if b.Sound == nil {
		t.Fatalf("Sound not set")
	}
}

func TestLLMQueryAdapterForwardsPromptAndReturnsResponseText(t *testing.T) {
	captured := struct {
		messages []Message
		opts     LLMOptions
	}{}
	llm := llmFunc(func(_ context.Context, msgs []Message, opts LLMOptions) (*LLMResponse, error) {
		captured.messages = msgs
		captured.opts = opts
		return &LLMResponse{Text: "ok"}, nil
	})
	adapter := LLMQueryAdapter{LLM: llm, Options: LLMOptions{Model: "test"}}
	got, err := adapter.Query(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if got != "ok" {
		t.Fatalf("Query() = %q, want ok", got)
	}
	if len(captured.messages) != 1 ||
		captured.messages[0].Role != "user" ||
		captured.messages[0].Content != "hi" {
		t.Fatalf("captured messages = %+v, want single user turn", captured.messages)
	}
	if captured.opts.Model != "test" {
		t.Fatalf("captured opts = %+v, want model=test", captured.opts)
	}
}

func TestLLMQueryAdapterReturnsSentinelWhenLLMNil(t *testing.T) {
	got, err := LLMQueryAdapter{}.Query(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if got != noopSentinel {
		t.Fatalf("Query() = %q, want %q", got, noopSentinel)
	}
}

type llmFunc func(context.Context, []Message, LLMOptions) (*LLMResponse, error)

func (f llmFunc) Query(ctx context.Context, msgs []Message, opts LLMOptions) (*LLMResponse, error) {
	return f(ctx, msgs, opts)
}
