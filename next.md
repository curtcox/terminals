1. Run Flutter widget tests in an environment with Flutter SDK installed (`terminal_client/test/widget_test.dart`).
2. Manually validate end-to-end carrier fallback sequence (gRPC → WebSocket → TCP → HTTP) with local server/client.
3. Confirm local diagnostic report quality when all carriers fail (status text + attempt breakdown).
