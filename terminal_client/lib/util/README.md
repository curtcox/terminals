# util

Platform-conditional exports for speech, alerts, and browser host.

Each file in this package is a thin conditional export that selects the right platform implementation at compile time:

- `speech.dart` — TTS/STT interface (IO vs. web)
- `alerts.dart` — system alert delivery (IO vs. web)
- `browser_host.dart` — browser window/tab host (stub vs. web)
- `alert_delivery.dart` — alert routing helpers
