Add a real silence-detecting `ai.SoundClassifier` implementation that
watches a PCM audio stream and emits a `SoundEvent{Label: "silence_after_sound"}`
(or similarly-named event) the first time the stream transitions from
"loud" to "quiet" for a configurable hold duration. This closes the Phase-6
sound-classification deliverable end-to-end: the monitor scenario already
consumes live hub audio, so a real silence detector lights up the
"tell me when the dishwasher stops" milestone without needing an external
model.

Implement in `internal/ai/silence_classifier.go` behind the existing
`ai.SoundClassifier` interface. RMS-energy threshold + hysteresis is fine;
keep the thresholds configurable so tests can drive transitions with short
PCM fixtures. Add focused unit tests for:

- loud-then-quiet transition emits one event and stops
- sustained quiet below the hold duration does not emit
- sustained loud never emits

Then add a scenario-level test that feeds PCM through the existing audio
hub and asserts `AudioMonitorScenario` notifies the source device.

Wiring: swap `ai.NewNoopBackends()` in `cmd/server/main.go` to plumb the
silence classifier as the `Sound` backend (keep noops for everything else
until those backends are chosen).
