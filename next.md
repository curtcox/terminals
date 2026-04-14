Add the "rich response" visual companion to the voice-audio pipeline so the VoiceAssistant scenario answer shows on the source device's screen alongside the TTS audio (Phase 5 "Rich responses" milestone). Concretely:

1. In `terminal_server/internal/ui/descriptor.go`, add a `VoiceAssistantResponseView(deviceID, prompt, response string) Descriptor` (and a matching `VoiceAssistantResponsePatch(response string) Descriptor` for `UpdateUI`) that renders a centered card with the recognized prompt and the assistant's reply. Follow the same style conventions as the existing scenario views (fonts, layout primitives).

2. In `terminal_server/internal/transport/control_stream.go`, extend `handleVoiceAudio` so that on a successful TTS round-trip it also emits a `SetUI` (or `UpdateUI` targeting the global overlay slot used for notifications) carrying the `VoiceAssistantResponseView` for the source device — in addition to the `PlayAudio` message. Make the visual optional when TTS is unavailable: still render the response text, just without audio.

3. Teach `voice_pipeline_integration_test.go` to assert that the response slice now contains (a) the existing `PlayAudio`, and (b) a `SetUI`/`UpdateUI` whose descriptor text contains the LLM reply (`It is sunny in Test City`). Keep the `TestControlStreamVoiceCommandTextPath` and `TestControlStreamVoiceAudioSurfacesLLMError` tests in sync.

4. Add a `VoiceAssistantResponseView` descriptor test in `internal/ui/descriptor_test.go` that checks the prompt and response are embedded in the rendered primitives.

Keep the `Command{Kind:"voice", Text:...}` path working and still exercised by `TestControlStreamVoiceCommandTextPath`. Do not add client-side scenario logic — the visual must be composed entirely from existing server-driven UI primitives.
