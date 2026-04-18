1. Add client-side monitors for display geometry changes, permission changes, and hot-plug endpoint changes, and emit capability deltas for those events.
2. Expand typed capability endpoint detail messages for multi-display and multi-endpoint audio/video inventory updates.
3. Add integration tests that assert server work is deferred until first accepted capability snapshot.
4. Add reconnect desynchronization tests that force snapshot re-baselining after stale delta rejection.
