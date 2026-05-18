# capabilities

Client-side capability probing and negotiation.

`screen_metrics.dart` measures display resolution and pixel density for the capability hello message. `probe.dart` queries available media devices (cameras, microphones) via WebRTC APIs. Results feed the capability negotiation handshake in `connection/`.
