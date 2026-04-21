import 'dart:async';
import 'dart:math' as math;

enum ConnectionPhase {
  disconnected,
  connecting,
  connectedUnregistered,
  registered,
  degraded,
}

ConnectionPhase deriveConnectionPhase({
  required bool shouldStayConnected,
  required bool isConnecting,
  required bool hasClient,
  required bool hasIncoming,
  required bool hasRegisterAck,
  required bool hasRecentTransportFailure,
}) {
  if (isConnecting) {
    return ConnectionPhase.connecting;
  }
  if (shouldStayConnected && hasClient && hasIncoming) {
    if (hasRegisterAck) {
      return ConnectionPhase.registered;
    }
    return ConnectionPhase.connectedUnregistered;
  }
  if (shouldStayConnected && hasRecentTransportFailure) {
    return ConnectionPhase.degraded;
  }
  if (shouldStayConnected) {
    return ConnectionPhase.connecting;
  }
  return ConnectionPhase.disconnected;
}

enum RetryBackoff {
  fixed,
  exponential,
}

class RetryPolicy {
  const RetryPolicy({
    required this.interval,
    required this.maxDuration,
    this.maxInterval,
    this.backoff = RetryBackoff.fixed,
  });

  final Duration interval;
  final Duration maxDuration;
  final Duration? maxInterval;
  final RetryBackoff backoff;

  Duration delayForAttempt(int attempt) {
    final normalizedAttempt = math.max(1, attempt);
    final baseMs = interval.inMilliseconds;
    if (backoff == RetryBackoff.fixed) {
      return interval;
    }
    final scaledMs = baseMs * (1 << (normalizedAttempt - 1));
    final capMs = maxInterval?.inMilliseconds;
    final boundedMs = capMs == null ? scaledMs : math.min(capMs, scaledMs);
    return Duration(milliseconds: math.max(1, boundedMs));
  }

  bool hasTimedOut(Duration elapsed) {
    return elapsed >= maxDuration;
  }
}

enum ReadinessResult {
  ready,
  timeout,
  failed,
}

class ConnectionReadinessGateway {
  ConnectionReadinessGateway({
    required this.currentPhase,
    required this.startConnection,
    required this.policy,
  });

  final ConnectionPhase Function() currentPhase;
  final Future<void> Function() startConnection;
  final RetryPolicy policy;

  Future<ReadinessResult> ensureConnectedAndRegistered() async {
    if (currentPhase() == ConnectionPhase.registered) {
      return ReadinessResult.ready;
    }
    try {
      await startConnection();
    } catch (_) {
      return ReadinessResult.failed;
    }

    final startedAt = DateTime.now().toUtc();
    var attempts = 0;
    while (true) {
      final phase = currentPhase();
      if (phase == ConnectionPhase.registered) {
        return ReadinessResult.ready;
      }
      final elapsed = DateTime.now().toUtc().difference(startedAt);
      if (policy.hasTimedOut(elapsed)) {
        return ReadinessResult.timeout;
      }
      attempts += 1;
      await Future<void>.delayed(policy.delayForAttempt(attempts));
    }
  }
}

enum SendMode {
  fireAndForget,
  queueUntilReady,
  requireAck,
}

enum SendResult {
  sent,
  notReady,
  failed,
  ackTimeout,
}

enum OutboundOperation {
  bootstrapHello,
  bootstrapRegister,
  bootstrapCapabilitySnapshot,
  heartbeat,
  sensorTelemetry,
  capabilityDelta,
  capabilitySnapshot,
  bugReport,
  launchApplication,
  runtimeQuery,
  deviceQuery,
  scenarioQuery,
  playbackArtifactsQuery,
  playbackMetadataQuery,
  uiAction,
  keyEvent,
  streamReady,
  webrtcSignal,
  artifactAvailable,
}

class OutboundRoutingRule {
  const OutboundRoutingRule({
    required this.mode,
    required this.safeToReplay,
    required this.requiresAck,
  });

  final SendMode mode;
  final bool safeToReplay;
  final bool requiresAck;
}

const Map<OutboundOperation, OutboundRoutingRule> kOutboundRoutingRules =
    <OutboundOperation, OutboundRoutingRule>{
  OutboundOperation.bootstrapHello: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.bootstrapRegister: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: true,
  ),
  OutboundOperation.bootstrapCapabilitySnapshot: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.heartbeat: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.sensorTelemetry: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.capabilityDelta: OutboundRoutingRule(
    mode: SendMode.queueUntilReady,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.capabilitySnapshot: OutboundRoutingRule(
    mode: SendMode.queueUntilReady,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.bugReport: OutboundRoutingRule(
    mode: SendMode.queueUntilReady,
    safeToReplay: true,
    requiresAck: true,
  ),
  OutboundOperation.launchApplication: OutboundRoutingRule(
    mode: SendMode.queueUntilReady,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.runtimeQuery: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.deviceQuery: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.scenarioQuery: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.playbackArtifactsQuery: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.playbackMetadataQuery: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.uiAction: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.keyEvent: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.streamReady: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
  OutboundOperation.webrtcSignal: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: false,
    requiresAck: false,
  ),
  OutboundOperation.artifactAvailable: OutboundRoutingRule(
    mode: SendMode.fireAndForget,
    safeToReplay: true,
    requiresAck: false,
  ),
};

class ReliableSendDispatcher<T> {
  ReliableSendDispatcher({
    required this.sendNow,
    required this.gateway,
  });

  final void Function(T request) sendNow;
  final ConnectionReadinessGateway gateway;
  final List<T> _queue = <T>[];

  Future<SendResult> sendWhenReady({
    required T request,
    required SendMode mode,
    Future<bool> Function()? waitForAck,
    Duration? ackTimeout,
  }) async {
    if (mode == SendMode.fireAndForget) {
      sendNow(request);
      return SendResult.sent;
    }

    _queue.add(request);
    final readiness = await gateway.ensureConnectedAndRegistered();
    if (readiness == ReadinessResult.failed) {
      return SendResult.failed;
    }
    if (readiness == ReadinessResult.timeout) {
      return SendResult.notReady;
    }

    _flushQueue();

    if (mode != SendMode.requireAck || waitForAck == null) {
      return SendResult.sent;
    }

    final timeout = ackTimeout ?? const Duration(seconds: 20);
    try {
      final acknowledged = await waitForAck().timeout(timeout);
      return acknowledged ? SendResult.sent : SendResult.failed;
    } on TimeoutException {
      return SendResult.ackTimeout;
    }
  }

  void _flushQueue() {
    if (_queue.isEmpty) {
      return;
    }
    final items = List<T>.from(_queue);
    _queue.clear();
    for (final item in items) {
      sendNow(item);
    }
  }
}
