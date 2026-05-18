# edge

Client-side durable storage, scheduling, and host state.

`bundle_store.dart` / `bundle_store_backend_*.dart` provide platform-adaptive durable key-value storage (native IO vs. web stubs). `artifact_export.dart` / `artifact_export_backend_*.dart` export diagnostic artifacts to platform storage. `host.dart` assembles the `EdgeHost` from its storage and scheduling dependencies. `host_state_backend_*.dart` persist host configuration across restarts. `retention.dart` enforces size limits on stored data. `scheduler.dart` provides simple admission control for edge flows. `clock_sync.dart` tracks server/client clock offset.
