Wire the AI voice pipeline into the control stream end-to-end so a client device's mic audio drives a `voice_assistant` scenario without the test having to call STT/TTS by hand. Concretely:

1. Extend the proto / wire model with a `voice_audio` client message (`device_id`, `audio` bytes, `sample_rate`, `is_final`) and a `play_audio` server message (`device_id`, `audio` bytes, `format`).
2. Have the control stream collect inbound `voice_audio` chunks per device, run the configured `scenario.SpeechToText` on the assembled buffer when `is_final` is set, and forward the transcript as a synthesized voice command into `Runtime.HandleVoiceText`.
3. After the `voice_assistant` scenario broadcasts its response, run the configured `scenario.TextToSpeech` on the response text and emit a `play_audio` server message back to the source device.
4. Replace the manual STT/TTS hops in `internal/transport/voice_pipeline_integration_test.go` with a test that pushes raw `voice_audio` through the control stream and asserts a `play_audio` reply with the expected bytes.

Keep the existing text-only `Command{Kind:"voice", Text:...}` path working for tests that don't need the audio round-trip.
