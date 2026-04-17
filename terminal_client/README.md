# Terminal Client

Flutter terminal client for the Terminals system.

## Supported Platforms

- Web
- macOS
- Android
- iOS
- Linux
- Windows

## Platform Notes

- Android and iOS include camera/microphone permission placeholders that are requested when media features are enabled.
- Linux and Windows runtime media capability/permission handling is platform-dependent and will be tightened in later remediation stages.
- Monitoring support is currently `foreground_only` on all platforms. The client pauses heartbeat/sensor monitoring when app lifecycle moves out of foreground and resumes on return.
- Android/iOS background monitoring guarantees (WorkManager/BGTask integration) are planned but not yet claimed by capabilities.

## Build Targets

From repository root:

- `make client-build-web`
- `make client-build-android`
- `make client-build-ios`
- `make client-build-linux`
- `make client-build-windows`
- `make client-build-macos`
- `make client-build-all`
