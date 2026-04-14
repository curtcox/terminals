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

	windowFrames := int(int64(c.cfg.SampleRate) * int64(c.cfg.WindowDuration) / int64(time.Second))
	if windowFrames <= 0 {
		windowFrames = 1
	}
	windowBytes := windowFrames * 2 * c.cfg.Channels
	if windowBytes < 2 {
		windowBytes = 2
	}
	holdFrames := int64(c.cfg.SampleRate) * int64(c.cfg.HoldDuration) / int64(time.Second)
	if holdFrames < 1 {
		holdFrames = 1
	}

	go func() {
		defer close(out)
		if audio == nil {
			return
		}
		accum := make([]byte, 0, windowBytes*2)
		tmp := make([]byte, windowBytes)

		state := silenceStateUnknown
		var quietFrames, totalFrames int64

		process := func(window []byte) bool {
			rms := computeRMS(window)
			frames := int64(len(window) / 2 / c.cfg.Channels)
			if frames < 1 {
				frames = 1
			}
			totalFrames += frames
			switch state {
			case silenceStateUnknown:
				if rms >= c.cfg.LoudThreshold {
					state = silenceStateLoud
				}
			case silenceStateLoud:
				if rms <= c.cfg.QuietThreshold {
					quietFrames = frames
					state = silenceStateQuiet
				}
			case silenceStateQuiet:
				if rms > c.cfg.QuietThreshold {
					quietFrames = 0
					state = silenceStateLoud
					return false
				}
				quietFrames += frames
				if quietFrames >= holdFrames {
					atMS := int64(0)
					if c.cfg.SampleRate > 0 {
						atMS = totalFrames * 1000 / int64(c.cfg.SampleRate)
					}
					select {
					case out <- SoundEvent{
						Label:      c.cfg.Label,
						Confidence: 1.0,
						AtMS:       atMS,
					}:
					case <-ctx.Done():
					}
					return true
				}
			}
			return false
		}

		for {
			if err := ctx.Err(); err != nil {
				return
			}
			n, err := audio.Read(tmp)
			if n > 0 {
				accum = append(accum, tmp[:n]...)
				for len(accum) >= windowBytes {
					if process(accum[:windowBytes]) {
						return
					}
					accum = accum[windowBytes:]
				}
			}
			if err != nil {
				// Process any remaining bytes as a final short window.
				if len(accum) >= 2 {
					trim := len(accum) - (len(accum) % 2)
					if trim >= 2 {
						if process(accum[:trim]) {
							return
						}
					}
				}
				return
			}
		}
	}()
	return out, nil
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
