package usecasevalidation

import (
	"context"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// FakeSoundClassifier is a test double for scenario.SoundClassifier that emits
// pre-configured events. The audio source passed to Classify is ignored; events
// are emitted immediately and the channel is closed. Use this to inject
// deterministic sound events into harness-based scenario tests.
type FakeSoundClassifier struct {
	Events []scenario.SoundEvent
}

// Classify returns a closed channel pre-loaded with the configured events.
func (f *FakeSoundClassifier) Classify(_ context.Context, _ scenario.AudioSource) (scenario.SoundEventStream, error) {
	ch := make(chan scenario.SoundEvent, len(f.Events))
	for _, ev := range f.Events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
