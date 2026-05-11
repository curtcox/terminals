import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/app/client_dependencies.dart';
import 'package:terminal_client/capabilities/probe.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/media/playback.dart';
import 'package:terminal_client/media/webrtc_engine.dart';

Finder findClientChromePrivacyOrCaptureIndicators() {
  return find.byWidgetPredicate(
    (widget) {
      final key = widget.key;
      if (key is! ValueKey<String>) {
        return false;
      }
      final value = key.value;
      return value.contains('client.chrome.privacy') ||
          value.contains('client.chrome.capture');
    },
    description: 'client-chrome privacy/capture indicator',
  );
}

class FakeClientHarness {
  FakeClientHarness({
    this.failFirstConnectStream = false,
    this.failConnectAttempts = 0,
    this.requestSubscriptionDelay = Duration.zero,
    this.stopStreamBlocker,
  });

  final List<FakeTerminalControlClient> createdClients =
      <FakeTerminalControlClient>[];
  final List<FakeMediaEngine> createdMediaEngines = <FakeMediaEngine>[];
  final List<ControlCarrierKind> requestedCarriers = <ControlCarrierKind>[];
  final bool failFirstConnectStream;
  final int failConnectAttempts;
  final Duration requestSubscriptionDelay;
  final Completer<void>? stopStreamBlocker;

  FakeTerminalControlClient get lastClient => createdClients.last;
  FakeMediaEngine get lastMediaEngine => createdMediaEngines.last;

  TerminalControlClient createClient({
    required String host,
    required int port,
  }) {
    requestedCarriers.add(ControlClientTransportHint.preferredCarrier);
    final client = FakeTerminalControlClient(
      host: host,
      port: port,
      failOnConnectStream: createdClients.length < failConnectAttempts ||
          (failFirstConnectStream && createdClients.isEmpty),
      requestSubscriptionDelay: requestSubscriptionDelay,
    );
    createdClients.add(client);
    return client;
  }

  ClientMediaEngine createMediaEngine({
    required String localDeviceID,
    required OutboundSignalCallback onSignal,
  }) {
    final engine = FakeMediaEngine(
      localDeviceID: localDeviceID,
      onSignal: onSignal,
      stopStreamBlocker: stopStreamBlocker,
    );
    createdMediaEngines.add(engine);
    return engine;
  }
}

class StaticCapabilityProbe implements CapabilityProbe {
  StaticCapabilityProbe(this.capabilities);

  final capv1.DeviceCapabilities capabilities;

  @override
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context) async {
    return capabilities.deepCopy();
  }
}

class TestScreenMetricsController {
  TestScreenMetricsController(this._metrics);

  ScreenMetrics _metrics;
  final ValueNotifier<int> changes = ValueNotifier<int>(0);

  ScreenMetrics read() => _metrics;

  void update(ScreenMetrics metrics) {
    _metrics = metrics;
    changes.value = changes.value + 1;
  }
}

class FakeWakeWordDetectorController implements WakeWordDetectorController {
  final List<bool> enabledStates = <bool>[];
  void Function(WakeWordUtterance utterance)? _onUtterance;

  @override
  Future<void> setEnabled(bool enabled) async {
    enabledStates.add(enabled);
  }

  @override
  void setOnUtterance(void Function(WakeWordUtterance utterance)? onUtterance) {
    _onUtterance = onUtterance;
  }

  void simulateUtterance({
    required List<int> audio,
    required int sampleRate,
    required bool isFinal,
  }) {
    _onUtterance?.call(
      WakeWordUtterance(
        audio: audio,
        sampleRate: sampleRate,
        isFinal: isFinal,
      ),
    );
  }

  @override
  Future<void> dispose() async {}
}

class FakeAudioPlayback implements AudioPlayback {
  final List<iov1.PlayAudio> playedRequests = <iov1.PlayAudio>[];

  @override
  Future<void> play(iov1.PlayAudio playAudio) async {
    playedRequests.add(playAudio.deepCopy());
  }

  @override
  Future<void> dispose() async {}
}

class FakeMediaEngine implements ClientMediaEngine {
  FakeMediaEngine({
    required this.localDeviceID,
    required this.onSignal,
    this.stopStreamBlocker,
  });

  final String localDeviceID;
  final OutboundSignalCallback onSignal;
  final Completer<void>? stopStreamBlocker;
  final Set<String> _activeStreamIDs = <String>{};
  final List<String> stopStreamCalls = <String>[];
  final Map<String, ValueNotifier<MediaStream?>> _remoteStreamsByID =
      <String, ValueNotifier<MediaStream?>>{};
  final Map<String, ValueNotifier<bool>> _streamAttachedByID =
      <String, ValueNotifier<bool>>{};
  final Map<String, ValueNotifier<double>> _audioLevelsByID =
      <String, ValueNotifier<double>>{};

  @override
  Future<void> startStream(iov1.StartStream start) async {
    if (start.streamId.isEmpty || _activeStreamIDs.contains(start.streamId)) {
      return;
    }
    _activeStreamIDs.add(start.streamId);
    _remoteStreamsByID.putIfAbsent(
      start.streamId,
      () => ValueNotifier<MediaStream?>(null),
    );
    _streamAttachedByID
        .putIfAbsent(
          start.streamId,
          () => ValueNotifier<bool>(false),
        )
        .value = true;
    _audioLevelsByID.putIfAbsent(
      start.streamId,
      () => ValueNotifier<double>(0.5),
    );
    onSignal(
      WebRTCSignal()
        ..streamId = start.streamId
        ..signalType = 'offer'
        ..payload = '{"sdp":"fake-offer-for-${start.streamId}"}',
    );
    onSignal(
      WebRTCSignal()
        ..streamId = start.streamId
        ..signalType = 'candidate'
        ..payload = '{"candidate":"fake-local-candidate-1"}',
    );
  }

  @override
  Future<void> stopStream(String streamID) async {
    stopStreamCalls.add(streamID);
    final blocker = stopStreamBlocker;
    if (blocker != null) {
      await blocker.future;
    }
    _activeStreamIDs.remove(streamID);
    _remoteStreamsByID
        .putIfAbsent(
          streamID,
          () => ValueNotifier<MediaStream?>(null),
        )
        .value = null;
    _streamAttachedByID
        .putIfAbsent(
          streamID,
          () => ValueNotifier<bool>(false),
        )
        .value = false;
    _audioLevelsByID
        .putIfAbsent(
          streamID,
          () => ValueNotifier<double>(0.0),
        )
        .value = 0.0;
  }

  @override
  Future<void> handleSignal(WebRTCSignal signal) async {
    if (signal.signalType.trim().toLowerCase() != 'offer') {
      return;
    }
    onSignal(
      WebRTCSignal()
        ..streamId = signal.streamId
        ..signalType = 'answer'
        ..payload = '{"sdp":"fake-answer-for-${signal.streamId}"}',
    );
    onSignal(
      WebRTCSignal()
        ..streamId = signal.streamId
        ..signalType = 'candidate'
        ..payload = '{"candidate":"fake-local-candidate-2"}',
    );
  }

  @override
  Future<void> dispose() async {}

  @override
  ValueListenable<MediaStream?> remoteStream(String streamID) {
    return _remoteStreamsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<MediaStream?>(null),
    );
  }

  @override
  ValueListenable<bool> streamAttached(String streamID) {
    return _streamAttachedByID.putIfAbsent(
      streamID,
      () => ValueNotifier<bool>(false),
    );
  }

  @override
  ValueListenable<double> audioLevel(String streamID) {
    return _audioLevelsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<double>(0.0),
    );
  }
}

class FakeTerminalControlClient implements TerminalControlClient {
  FakeTerminalControlClient({
    required this.host,
    required this.port,
    required this.failOnConnectStream,
    required this.requestSubscriptionDelay,
  });

  final String host;
  final int port;
  final bool failOnConnectStream;
  final Duration requestSubscriptionDelay;
  final List<ConnectRequest> requests = <ConnectRequest>[];
  final StreamController<ConnectResponse> _responses =
      StreamController<ConnectResponse>.broadcast();
  StreamSubscription<ConnectRequest>? _requestSubscription;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    if (requestSubscriptionDelay > Duration.zero) {
      unawaited(
        Future<void>.delayed(requestSubscriptionDelay).then((_) {
          _requestSubscription = requests.listen(this.requests.add);
        }),
      );
    } else {
      _requestSubscription = requests.listen(this.requests.add);
    }
    if (failOnConnectStream) {
      return Stream<ConnectResponse>.error(StateError('stream dropped'));
    }
    return _responses.stream;
  }

  @override
  Future<void> shutdown() async {
    await _requestSubscription?.cancel();
    if (!_responses.isClosed) {
      await _responses.close();
    }
  }

  void emitResponse(ConnectResponse response) {
    _responses.add(response);
  }

  Future<void> closeStream() async {
    if (!_responses.isClosed) {
      await _responses.close();
    }
  }
}
