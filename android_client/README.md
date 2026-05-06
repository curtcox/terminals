# Terminals Native Android Client

This is the native Android client scaffold for Terminals. It is a generic
terminal shell for Android and Kindle Fire tablets.

The client must remain scenario-agnostic:

- server behavior belongs in `terminal_server/`,
- wire contracts come from `api/terminals/**` protobufs,
- Android platform APIs stay behind adapters,
- Fire OS compatibility means no Google Play Services dependency.

Useful commands:

```bash
../scripts/check-android-client-boundary.sh
./gradlew testDebugUnitTest
./gradlew lintDebug
./gradlew assembleDebug
```

See `../docs/client-android.md` and
`../plans/features/android-client/plan.md` for the implementation sequence.
