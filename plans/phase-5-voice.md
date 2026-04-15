# Phase 5 — Voice Assistant

See [masterplan.md](../masterplan.md) for overall system context.

Add AI-powered voice interaction.

## Prerequisites

- [phase-3-media.md](phase-3-media.md) complete — mic audio can be streamed client → server.
- Clear AI backend decision (local vs. cloud) for STT, LLM, TTS — see [technology.md](technology.md#ai-backend-pluggable).

## Deliverables

- [x] **AI backend interfaces**: Define and implement the pluggable AI interfaces (`SpeechToText`, `TextToSpeech`, `LLM`, `VisionAnalyzer`, `SoundClassifier`). See [technology.md](technology.md#ai-backend-pluggable).
- [x] **Wake word detection**: Continuous low-power audio monitoring on idle devices.
- [x] **STT pipeline**: Mic → server → speech-to-text.
- [x] **LLM query pipeline**: Transcribed text → LLM → response text.
- [x] **TTS pipeline**: Response text → TTS → speaker.
- [x] **Rich responses**: Voice response + accompanying visual UI on the device screen. See [use-case-flows.md](use-case-flows.md#smart-speaker--voice-assistant).
- [ ] **Intent/Event bus**: Ship the typed trigger bus. Voice transcripts are parsed into `Intent` records (with `Action`, `Object`, `Slots`, `Scope`, `Source: voice`); existing UI-triggered flows also emit through the bus. The scenario engine matches on `Intent`/`Event` instead of stringly-typed triggers. See [scenario-engine.md](scenario-engine.md#triggers-intents-and-events).
- [ ] **LLM intent resolution**: Optional path where ambiguous utterances go through the LLM to produce a structured `Intent`; the LLM is a producer on the same bus, not a side path.
- [ ] **Voice assistant media plan**: Re-express the STT/LLM/TTS pipelines as `mic → fork → [STT, optional recorder]`, `TTS → speaker`, with shared `mic.analyze` + exclusive `speaker.main` claims so the assistant overlays without evicting ambient scenarios.
- [ ] **Overlay-scoped rich responses**: Render the visual companion UI on the overlay layer with a `screen.overlay` claim so the assistant never replaces the main scenario.

## Milestone

Say a wake word, ask a question, get a spoken and visual answer.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Voice assistant scenario priority.
- [io-abstraction.md](io-abstraction.md) — Consume/produce/fork primitives used by the pipelines.
- [server-driven-ui.md](server-driven-ui.md) — Visual companion UI.
- [phase-6-monitoring.md](phase-6-monitoring.md) — Next phase.
