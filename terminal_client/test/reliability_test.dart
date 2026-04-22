import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/reliability.dart';

void main() {
  group('connection phase', () {
    test('maps disconnected state', () {
      expect(
        deriveConnectionPhase(
          shouldStayConnected: false,
          isConnecting: false,
          hasClient: false,
          hasIncoming: false,
          hasRegisterAck: false,
          hasRecentTransportFailure: false,
        ),
        ConnectionPhase.disconnected,
      );
    });

    test('maps connecting state', () {
      expect(
        deriveConnectionPhase(
          shouldStayConnected: true,
          isConnecting: true,
          hasClient: false,
          hasIncoming: false,
          hasRegisterAck: false,
          hasRecentTransportFailure: false,
        ),
        ConnectionPhase.connecting,
      );
    });

    test('maps connected_unregistered state', () {
      expect(
        deriveConnectionPhase(
          shouldStayConnected: true,
          isConnecting: false,
          hasClient: true,
          hasIncoming: true,
          hasRegisterAck: false,
          hasRecentTransportFailure: false,
        ),
        ConnectionPhase.connectedUnregistered,
      );
    });

    test('maps registered state', () {
      expect(
        deriveConnectionPhase(
          shouldStayConnected: true,
          isConnecting: false,
          hasClient: true,
          hasIncoming: true,
          hasRegisterAck: true,
          hasRecentTransportFailure: false,
        ),
        ConnectionPhase.registered,
      );
    });

    test(
        'maps degraded state when transport failed but reconnect intent remains',
        () {
      expect(
        deriveConnectionPhase(
          shouldStayConnected: true,
          isConnecting: false,
          hasClient: false,
          hasIncoming: false,
          hasRegisterAck: false,
          hasRecentTransportFailure: true,
        ),
        ConnectionPhase.degraded,
      );
    });
  });

  group('retry policy', () {
    test('fixed backoff uses constant interval', () {
      const policy = RetryPolicy(
        interval: Duration(milliseconds: 50),
        maxDuration: Duration(seconds: 1),
      );
      expect(policy.delayForAttempt(1), const Duration(milliseconds: 50));
      expect(policy.delayForAttempt(4), const Duration(milliseconds: 50));
    });

    test('exponential backoff doubles until max interval', () {
      const policy = RetryPolicy(
        interval: Duration(milliseconds: 25),
        maxDuration: Duration(seconds: 5),
        maxInterval: Duration(milliseconds: 200),
        backoff: RetryBackoff.exponential,
      );
      expect(policy.delayForAttempt(1), const Duration(milliseconds: 25));
      expect(policy.delayForAttempt(2), const Duration(milliseconds: 50));
      expect(policy.delayForAttempt(3), const Duration(milliseconds: 100));
      expect(policy.delayForAttempt(4), const Duration(milliseconds: 200));
      expect(policy.delayForAttempt(6), const Duration(milliseconds: 200));
    });

    test('times out once elapsed exceeds max duration', () {
      const policy = RetryPolicy(
        interval: Duration(milliseconds: 20),
        maxDuration: Duration(milliseconds: 80),
      );
      expect(policy.hasTimedOut(const Duration(milliseconds: 79)), isFalse);
      expect(policy.hasTimedOut(const Duration(milliseconds: 80)), isTrue);
    });

    test('retry controller executes retries until stop condition', () async {
      final attempts = <int>[];
      final reachedThree = Completer<void>();
      final controller = RetryController(
        policy: const RetryPolicy(
          interval: Duration(milliseconds: 10),
          maxDuration: Duration(milliseconds: 200),
        ),
      );
      controller.start(
        shouldContinue: () => attempts.length < 3,
        onRetry: (attempt) {
          attempts.add(attempt);
          if (attempt == 3 && !reachedThree.isCompleted) {
            reachedThree.complete();
          }
        },
      );

      await reachedThree.future.timeout(const Duration(milliseconds: 200));
      controller.stop();

      expect(attempts, <int>[1, 2, 3]);
    });

    test('retry controller reports timeout', () async {
      final timeoutResult = Completer<(int, Duration)>();
      final controller = RetryController(
        policy: const RetryPolicy(
          interval: Duration(milliseconds: 10),
          maxDuration: Duration(milliseconds: 35),
        ),
      );
      controller.start(
        shouldContinue: () => true,
        onRetry: (_) {},
        onTimeout: (attempts, elapsed) {
          if (!timeoutResult.isCompleted) {
            timeoutResult.complete((attempts, elapsed));
          }
        },
      );

      final result = await timeoutResult.future.timeout(
        const Duration(milliseconds: 250),
      );
      controller.stop();

      expect(result.$1, greaterThanOrEqualTo(1));
      expect(result.$2, greaterThanOrEqualTo(const Duration(milliseconds: 35)));
    });
  });

  group('readiness gateway', () {
    test('returns ready once phase reaches registered', () async {
      var phase = ConnectionPhase.disconnected;
      final gateway = ConnectionReadinessGateway(
        currentPhase: () => phase,
        startConnection: () async {
          phase = ConnectionPhase.connecting;
          Future<void>.delayed(const Duration(milliseconds: 15), () {
            phase = ConnectionPhase.registered;
          });
        },
        policy: const RetryPolicy(
          interval: Duration(milliseconds: 10),
          maxDuration: Duration(milliseconds: 200),
        ),
      );

      final result = await gateway.ensureConnectedAndRegistered();
      expect(result, ReadinessResult.ready);
    });

    test('returns timeout when registration never arrives', () async {
      var phase = ConnectionPhase.disconnected;
      final gateway = ConnectionReadinessGateway(
        currentPhase: () => phase,
        startConnection: () async {
          phase = ConnectionPhase.connecting;
        },
        policy: const RetryPolicy(
          interval: Duration(milliseconds: 10),
          maxDuration: Duration(milliseconds: 60),
        ),
      );

      final result = await gateway.ensureConnectedAndRegistered();
      expect(result, ReadinessResult.timeout);
    });

    test('returns failed when start connection throws', () async {
      final gateway = ConnectionReadinessGateway(
        currentPhase: () => ConnectionPhase.disconnected,
        startConnection: () async {
          throw StateError('connect failed');
        },
        policy: const RetryPolicy(
          interval: Duration(milliseconds: 10),
          maxDuration: Duration(milliseconds: 60),
        ),
      );

      final result = await gateway.ensureConnectedAndRegistered();
      expect(result, ReadinessResult.failed);
    });
  });

  group('reliable send dispatcher', () {
    test('sends fire-and-forget immediately', () async {
      final sent = <String>[];
      final dispatcher = ReliableSendDispatcher<String>(
        sendNow: sent.add,
        gateway: ConnectionReadinessGateway(
          currentPhase: () => ConnectionPhase.disconnected,
          startConnection: () async {},
          policy: const RetryPolicy(
            interval: Duration(milliseconds: 10),
            maxDuration: Duration(milliseconds: 30),
          ),
        ),
      );

      final result = await dispatcher.sendWhenReady(
        request: 'one',
        mode: SendMode.fireAndForget,
      );

      expect(result, SendResult.sent);
      expect(sent, <String>['one']);
    });

    test('queues until ready then flushes in order', () async {
      var phase = ConnectionPhase.disconnected;
      final sent = <String>[];
      final dispatcher = ReliableSendDispatcher<String>(
        sendNow: sent.add,
        gateway: ConnectionReadinessGateway(
          currentPhase: () => phase,
          startConnection: () async {
            phase = ConnectionPhase.registered;
          },
          policy: const RetryPolicy(
            interval: Duration(milliseconds: 10),
            maxDuration: Duration(milliseconds: 60),
          ),
        ),
      );

      final first = await dispatcher.sendWhenReady(
        request: 'a',
        mode: SendMode.queueUntilReady,
      );
      final second = await dispatcher.sendWhenReady(
        request: 'b',
        mode: SendMode.queueUntilReady,
      );

      expect(first, SendResult.sent);
      expect(second, SendResult.sent);
      expect(sent, <String>['a', 'b']);
    });

    test('require_ack succeeds when ack future resolves true', () async {
      final sent = <String>[];
      final ack = Completer<bool>();
      final dispatcher = ReliableSendDispatcher<String>(
        sendNow: sent.add,
        gateway: ConnectionReadinessGateway(
          currentPhase: () => ConnectionPhase.registered,
          startConnection: () async {},
          policy: const RetryPolicy(
            interval: Duration(milliseconds: 10),
            maxDuration: Duration(milliseconds: 60),
          ),
        ),
      );

      final pending = dispatcher.sendWhenReady(
        request: 'ack-me',
        mode: SendMode.requireAck,
        waitForAck: () => ack.future,
        ackTimeout: const Duration(milliseconds: 80),
      );
      ack.complete(true);

      expect(await pending, SendResult.sent);
      expect(sent, <String>['ack-me']);
    });

    test('require_ack times out when ack future does not resolve', () async {
      final sent = <String>[];
      final dispatcher = ReliableSendDispatcher<String>(
        sendNow: sent.add,
        gateway: ConnectionReadinessGateway(
          currentPhase: () => ConnectionPhase.registered,
          startConnection: () async {},
          policy: const RetryPolicy(
            interval: Duration(milliseconds: 10),
            maxDuration: Duration(milliseconds: 60),
          ),
        ),
      );

      final result = await dispatcher.sendWhenReady(
        request: 'ack-timeout',
        mode: SendMode.requireAck,
        waitForAck: () => Completer<bool>().future,
        ackTimeout: const Duration(milliseconds: 20),
      );

      expect(result, SendResult.ackTimeout);
      expect(sent, <String>['ack-timeout']);
    });
  });

  group('outbound routing rules', () {
    test('routes user input actions through queue-until-ready reliability', () {
      expect(
        kOutboundRoutingRules[OutboundOperation.uiAction]?.mode,
        SendMode.queueUntilReady,
      );
      expect(
        kOutboundRoutingRules[OutboundOperation.keyEvent]?.mode,
        SendMode.queueUntilReady,
      );
    });
  });
}
