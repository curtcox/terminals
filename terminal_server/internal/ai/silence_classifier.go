package ai

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"time"
)

// SilenceAfterSoundLabel is the label emitted by SilenceClassifier the first
// time a PCM stream transitions from "loud" to sustained "quiet". Scenarios
// that want to react to this event can match on this label directly or use
// the substring "sound"/"silence" via AudioMonitorScenario target matching.
const SilenceAfterSoundLabel = "silence_after_sound"

const (
	silenceStateUnknown = iota
	silenceStateLoud
	silenceStateQuiet
)

type silenceClassifierRuntime struct {
	cfg         SilenceClassifierConfig
	out         chan<- SoundEvent
	audio       io.Reader
	windowBytes int
	holdFrames  int64
	state       int
	quietFrames int64
	totalFrames int64
}

// SilenceClassifierConfig configures the RMS-energy silence detector. All
// fields are optional; zero values are replaced with sensible defaults.
type SilenceClassifierConfig struct {
	// SampleRate is the PCM sample rate in Hz (default 16000).
	SampleRate int
	// Channels is the number of interleaved channels (default 1).
	Channels int
	// WindowDuration is the RMS analysis window (default 20ms).
	WindowDuration time.Duration
	// LoudThreshold is the normalized RMS (0..1) required to enter the
	// "loud" state (default 0.05).
	LoudThreshold float64
	// QuietThreshold is the normalized RMS (0..1) at or below which the
	// stream is considered "quiet". Must be less than LoudThreshold for
	// hysteresis; zero or out-of-range values default to LoudThreshold/2.
	QuietThreshold float64
	// HoldDuration is how long quiet must be sustained after a loud
	// segment before a silence event is emitted (default 2s).
	HoldDuration time.Duration
	// Label is the emitted SoundEvent label (default SilenceAfterSoundLabel).
	Label string
}

// SilenceClassifier is an ai.SoundClassifier that watches a 16-bit
// little-endian PCM stream and emits a single SoundEvent the first time the
// stream transitions from a loud segment to a sustained quiet segment.
type SilenceClassifier struct {
	cfg SilenceClassifierConfig
}

// NewSilenceClassifier returns a SilenceClassifier configured with cfg.
// Unset fields are populated with defaults.
func NewSilenceClassifier(cfg SilenceClassifierConfig) *SilenceClassifier {
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	if cfg.Channels <= 0 {
		cfg.Channels = 1
	}
	if cfg.WindowDuration <= 0 {
		cfg.WindowDuration = 20 * time.Millisecond
	}
	if cfg.LoudThreshold <= 0 {
		cfg.LoudThreshold = 0.05
	}
	if cfg.QuietThreshold <= 0 || cfg.QuietThreshold >= cfg.LoudThreshold {
		cfg.QuietThreshold = cfg.LoudThreshold / 2
	}
	if cfg.HoldDuration <= 0 {
		cfg.HoldDuration = 2 * time.Second
	}
	if cfg.Label == "" {
		cfg.Label = SilenceAfterSoundLabel
	}
	return &SilenceClassifier{cfg: cfg}
}

// Classify streams PCM bytes from audio, computes RMS energy per window,
// and emits one SoundEvent when a loud-to-quiet transition sustains for at
// least HoldDuration. The channel is closed when the first event is emitted,
// when the context is canceled, or when the audio stream ends.
func (c *SilenceClassifier) Classify(ctx context.Context, audio io.Reader) (<-chan SoundEvent, error) {
	out := make(chan SoundEvent, 1)

	windowFrames := silenceWindowFrames(c.cfg)
	windowBytes := windowFrames * 2 * c.cfg.Channels
	if windowBytes < 2 {
		windowBytes = 2
	}
	holdFrames := silenceHoldFrames(c.cfg)

	go func() {
		defer close(out)
		runtime := silenceClassifierRuntime{
			cfg:         c.cfg,
			out:         out,
			audio:       audio,
			windowBytes: windowBytes,
			holdFrames:  holdFrames,
			state:       silenceStateUnknown,
		}
		runtime.run(ctx)
	}()
	return out, nil
}

func silenceWindowFrames(cfg SilenceClassifierConfig) int {
	windowFrames := int(int64(cfg.SampleRate) * int64(cfg.WindowDuration) / int64(time.Second))
	if windowFrames <= 0 {
		return 1
	}
	return windowFrames
}

func silenceHoldFrames(cfg SilenceClassifierConfig) int64 {
	holdFrames := int64(cfg.SampleRate) * int64(cfg.HoldDuration) / int64(time.Second)
	if holdFrames < 1 {
		return 1
	}
	return holdFrames
}

func (r *silenceClassifierRuntime) process(ctx context.Context, window []byte) bool {
	rms := computeRMS(window)
	frames := int64(len(window) / 2 / r.cfg.Channels)
	if frames < 1 {
		frames = 1
	}
	r.totalFrames += frames

	switch r.state {
	case silenceStateUnknown:
		return r.processUnknown(rms)
	case silenceStateLoud:
		return r.processLoud(rms, frames)
	case silenceStateQuiet:
		return r.processQuiet(ctx, rms, frames)
	}
	return false
}

func (r *silenceClassifierRuntime) run(ctx context.Context) {
	if r.audio == nil {
		return
	}
	accum := make([]byte, 0, r.windowBytes*2)
	tmp := make([]byte, r.windowBytes)
	for {
		if ctx.Err() != nil {
			return
		}
		if r.readNextWindow(ctx, &accum, tmp) {
			return
		}
	}
}

func (r *silenceClassifierRuntime) readNextWindow(ctx context.Context, accum *[]byte, tmp []byte) bool {
	n, err := r.audio.Read(tmp)
	if n > 0 {
		*accum = append(*accum, tmp[:n]...)
		if r.processFullWindows(ctx, accum) {
			return true
		}
	}
	if err == nil {
		return false
	}
	r.processFinalWindow(ctx, *accum)
	return true
}

func (r *silenceClassifierRuntime) processFullWindows(ctx context.Context, accum *[]byte) bool {
	for len(*accum) >= r.windowBytes {
		if r.process(ctx, (*accum)[:r.windowBytes]) {
			return true
		}
		*accum = (*accum)[r.windowBytes:]
	}
	return false
}

func (r *silenceClassifierRuntime) processFinalWindow(ctx context.Context, accum []byte) bool {
	if len(accum) < 2 {
		return false
	}
	trim := len(accum) - (len(accum) % 2)
	if trim < 2 {
		return false
	}
	return r.process(ctx, accum[:trim])
}

func (r *silenceClassifierRuntime) processUnknown(rms float64) bool {
	if rms >= r.cfg.LoudThreshold {
		r.state = silenceStateLoud
	}
	return false
}

func (r *silenceClassifierRuntime) processLoud(rms float64, frames int64) bool {
	if rms <= r.cfg.QuietThreshold {
		r.quietFrames = frames
		r.state = silenceStateQuiet
	}
	return false
}

func (r *silenceClassifierRuntime) processQuiet(ctx context.Context, rms float64, frames int64) bool {
	if rms > r.cfg.QuietThreshold {
		r.quietFrames = 0
		r.state = silenceStateLoud
		return false
	}
	r.quietFrames += frames
	if r.quietFrames < r.holdFrames {
		return false
	}
	r.emit(ctx)
	return true
}

func (r *silenceClassifierRuntime) emit(ctx context.Context) {
	atMS := int64(0)
	if r.cfg.SampleRate > 0 {
		atMS = r.totalFrames * 1000 / int64(r.cfg.SampleRate)
	}
	select {
	case r.out <- SoundEvent{
		Label:      r.cfg.Label,
		Confidence: 1.0,
		AtMS:       atMS,
	}:
	case <-ctx.Done():
	}
}

// computeRMS returns the root-mean-square energy of the given 16-bit
// little-endian PCM buffer, normalized to the [0, 1] range. Channels are
// collapsed to a single energy value (interleaving does not affect RMS).
func computeRMS(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	count := len(pcm) / 2
	if count == 0 {
		return 0
	}
	var sumSq float64
	for i := 0; i < count; i++ {
		s := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		norm := float64(s) / float64(math.MaxInt16)
		sumSq += norm * norm
	}
	return math.Sqrt(sumSq / float64(count))
}
