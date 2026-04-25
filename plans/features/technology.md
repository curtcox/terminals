---
title: "Technology Choices"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Technology Choices

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

## Server

| Component       | Technology                | Rationale                                    |
|-----------------|---------------------------|----------------------------------------------|
| Language        | Go                        | Specified requirement                        |
| RPC Framework   | gRPC + protobuf           | Bidirectional streaming, strong typing, codegen for Go + Dart |
| Media Transport | WebRTC (Pion)             | Go-native WebRTC. SFU capability, adaptive bitrate |
| SIP/VoIP        | Pion/SIP or go-ozzo/sip   | External phone calls                         |
| Database        | SQLite (via modernc.org)  | Zero-dependency, sufficient for single-server |
| mDNS            | hashicorp/mdns            | Service discovery on LAN                     |
| AI Interface    | Plugin-based (Go interfaces) | Swap implementations without changing server core |
| Config          | YAML                      | Human-readable server configuration          |

## Client

| Component         | Technology                 | Rationale                                    |
|-------------------|----------------------------|----------------------------------------------|
| Framework         | Flutter                    | Specified requirement. Single codebase for all platforms |
| gRPC              | grpc-dart                  | Matches server gRPC. Protobuf codegen for Dart |
| WebRTC            | flutter_webrtc             | Mature, cross-platform WebRTC                |
| mDNS              | multicast_dns / nsd        | Platform-aware mDNS scanning                 |
| Sensors           | sensors_plus               | Accelerometer, gyroscope, compass            |
| Camera            | camera                     | Cross-platform camera access                 |
| Audio             | Built into WebRTC + record | Recording and playback                       |
| Bluetooth         | flutter_blue_plus          | BLE scanning and connection                  |
| Platform channels | Custom per-platform        | USB, NFC, and other platform-specific IO     |

## AI Backend (Pluggable)

The server defines Go interfaces for each AI capability. Implementations can target local or cloud services:

```go
type SpeechToText interface {
    Transcribe(ctx context.Context, audio io.Reader, opts STTOptions) (<-chan Transcript, error)
}

type TextToSpeech interface {
    Synthesize(ctx context.Context, text string, opts TTSOptions) (io.Reader, error)
}

type LLM interface {
    Query(ctx context.Context, messages []Message, opts LLMOptions) (*Response, error)
}

type VisionAnalyzer interface {
    Analyze(ctx context.Context, frame image.Image, prompt string) (*Analysis, error)
}

type SoundClassifier interface {
    Classify(ctx context.Context, audio io.Reader) (<-chan SoundEvent, error)
}
```

Potential implementations:

- **Local**: Whisper (STT), Piper (TTS), llama.cpp (LLM), ONNX Runtime (vision/sound)
- **Cloud**: OpenAI, Google Cloud, Anthropic, ElevenLabs, Deepgram

The choice of implementation is a server configuration decision, not an architectural one.

## Related Plans

- [architecture-server.md](architecture-server.md) — How these map to server modules.
- [architecture-client.md](architecture-client.md) — How these map to client modules.
- [ci.md](ci.md) — Build/test/lint toolchain for these stacks.
