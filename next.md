1. Run Flutter widget tests in an environment with Flutter SDK installed (`terminal_client/test/widget_test.dart`).
2. Manually validate fallback order and immediate failover (gRPC → WebSocket → TCP → HTTP) with local server/client.
3. Validate resume-token reuse across reconnects, including a carrier switch reconnect.
4. Confirm local diagnostic report quality when all carriers fail (status text + attempt breakdown).
