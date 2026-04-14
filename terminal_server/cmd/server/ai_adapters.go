package main

import (
	"context"
	"image"
	"io"

	"github.com/curtcox/terminals/terminal_server/internal/ai"
	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// scenarioDeviceAudio adapts an *audio.Hub to scenario.DeviceAudioSubscriber
// so scenarios can subscribe to live device mic audio without depending on
// the audio package directly.
type scenarioDeviceAudio struct {
	hub *audio.Hub
}

func (a scenarioDeviceAudio) SubscribeAudio(
	ctx context.Context,
	deviceID string,
) (scenario.AudioSubscription, error) {
	return a.hub.Subscribe(ctx, deviceID), nil
}

// scenarioLLM adapts an ai.LLM to scenario.LLM so the runtime can invoke
// the configured backend without depending on the ai package.
type scenarioLLM struct {
	backend ai.LLM
}

func (a scenarioLLM) Query(
	ctx context.Context,
	messages []scenario.LLMMessage,
	opts scenario.LLMOptions,
) (*scenario.LLMResponse, error) {
	aiMsgs := make([]ai.Message, len(messages))
	for i, m := range messages {
		aiMsgs[i] = ai.Message{Role: m.Role, Content: m.Content}
	}
	resp, err := a.backend.Query(ctx, aiMsgs, ai.LLMOptions{
		Model:       opts.Model,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return &scenario.LLMResponse{Text: resp.Text, FinishReason: resp.FinishReason}, nil
}

// scenarioSTT adapts an ai.SpeechToText to scenario.SpeechToText.
type scenarioSTT struct {
	backend ai.SpeechToText
}

func (a scenarioSTT) Transcribe(
	ctx context.Context,
	audio scenario.AudioSource,
) (scenario.TranscriptStream, error) {
	reader := audioSourceReader{src: audio}
	ch, err := a.backend.Transcribe(ctx, reader, ai.STTOptions{})
	if err != nil {
		return nil, err
	}
	out := make(chan scenario.Transcript)
	go func() {
		defer close(out)
		for t := range ch {
			out <- scenario.Transcript{
				Text:       t.Text,
				Confidence: t.Confidence,
				IsFinal:    t.IsFinal,
			}
		}
	}()
	return out, nil
}

type audioSourceReader struct {
	src scenario.AudioSource
}

func (r audioSourceReader) Read(p []byte) (int, error) {
	if r.src == nil {
		return 0, io.EOF
	}
	return r.src.Read(p)
}

// scenarioTTS adapts an ai.TextToSpeech to scenario.TextToSpeech.
type scenarioTTS struct {
	backend ai.TextToSpeech
}

func (a scenarioTTS) Synthesize(
	ctx context.Context,
	text string,
	opts scenario.TTSOptions,
) (scenario.AudioPlayback, error) {
	r, err := a.backend.Synthesize(ctx, text, ai.TTSOptions{
		Voice:  opts.Voice,
		Format: opts.Format,
	})
	if err != nil {
		return nil, err
	}
	return audioPlaybackReader{r: r}, nil
}

type audioPlaybackReader struct {
	r io.Reader
}

func (a audioPlaybackReader) Read(p []byte) (int, error) {
	if a.r == nil {
		return 0, io.EOF
	}
	return a.r.Read(p)
}

// scenarioVisionAnalyzer adapts an ai.VisionAnalyzer to scenario.VisionAnalyzer.
type scenarioVisionAnalyzer struct {
	backend ai.VisionAnalyzer
}

func (a scenarioVisionAnalyzer) Analyze(
	ctx context.Context,
	frame image.Image,
	prompt string,
) (*scenario.VisionAnalysis, error) {
	resp, err := a.backend.Analyze(ctx, frame, prompt)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	labels := make([]string, len(resp.Labels))
	copy(labels, resp.Labels)
	return &scenario.VisionAnalysis{Caption: resp.Caption, Labels: labels}, nil
}

// scenarioSoundClassifier adapts an ai.SoundClassifier to scenario.SoundClassifier.
type scenarioSoundClassifier struct {
	backend ai.SoundClassifier
}

func (a scenarioSoundClassifier) Classify(
	ctx context.Context,
	audio scenario.AudioSource,
) (scenario.SoundEventStream, error) {
	reader := audioSourceReader{src: audio}
	ch, err := a.backend.Classify(ctx, reader)
	if err != nil {
		return nil, err
	}
	out := make(chan scenario.SoundEvent)
	go func() {
		defer close(out)
		for event := range ch {
			out <- scenario.SoundEvent{
				Label:      event.Label,
				Confidence: event.Confidence,
				AtMS:       event.AtMS,
			}
		}
	}()
	return out, nil
}
