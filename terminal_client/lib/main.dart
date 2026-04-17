import 'dart:async';
import 'dart:math' as math;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/discovery/mdns_scanner.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/media/webrtc_engine.dart';
import 'package:terminal_client/util/speech.dart' as speech;

typedef TerminalControlClientFactory = TerminalControlClient Function({
  required String host,
  required int port,
});
typedef ClientMediaEngineFactory = ClientMediaEngine Function({
  required String localDeviceID,
  required OutboundSignalCallback onSignal,
});

const bool _e2eEmitEvents = bool.fromEnvironment(
  'TERMINALS_E2E_EMIT_EVENTS',
);
const bool _e2eAutoScanConnect = bool.fromEnvironment(
  'TERMINALS_E2E_AUTO_SCAN_CONNECT',
);
const String _mdnsServiceType = String.fromEnvironment(
  'TERMINALS_MDNS_SERVICE_TYPE',
  defaultValue: '_terminals._tcp.local',
);
const int _e2eStartupDelayMs = int.fromEnvironment(
  'TERMINALS_E2E_STARTUP_DELAY_MS',
  defaultValue: 600,
);
const String _bugReportActionPrefix = 'bug_report';
const int _clientContextRecentUiCap = 32;
const int _clientContextRecentLogCap = 200;
const int _clientContextRecentErrorCap = 32;
const Duration _bugReportAckTimeout = Duration(seconds: 20);
const List<String> _bugTokenWords = <String>[
  'ace',
  'actor',
  'adapt',
  'air',
  'alert',
  'anchor',
  'angle',
  'apple',
  'artist',
  'asset',
  'audio',
  'autumn',
  'badge',
  'balance',
  'beam',
  'berry',
  'beyond',
  'bicycle',
  'bird',
  'blossom',
  'blue',
  'board',
  'book',
  'breeze',
  'bridge',
  'bright',
  'buffer',
  'button',
  'cable',
  'calm',
  'camera',
  'canvas',
  'captain',
  'carbon',
  'care',
  'center',
  'chapter',
  'check',
  'choice',
  'circle',
  'city',
  'clarity',
  'clock',
  'cloud',
  'coast',
  'color',
  'comfort',
  'compass',
  'control',
  'copper',
  'corner',
  'cotton',
  'craft',
  'credit',
  'crisp',
  'current',
  'cycle',
  'daily',
  'dawn',
  'delta',
  'design',
  'detail',
  'device',
  'dialog',
  'dock',
  'domain',
  'dream',
  'drift',
  'drive',
  'echo',
  'edge',
  'ember',
  'energy',
  'engine',
  'entry',
  'equal',
  'estate',
  'evening',
  'event',
  'fabric',
  'factor',
  'field',
  'filter',
  'final',
  'flame',
  'flight',
  'flower',
  'focus',
  'forest',
  'frame',
  'fresh',
  'future',
  'garden',
  'gentle',
  'glide',
  'glow',
  'gold',
  'grain',
  'graph',
  'green',
  'group',
  'guide',
  'habit',
  'harbor',
  'harmony',
  'haven',
  'hero',
  'horizon',
  'house',
  'idea',
  'image',
  'index',
  'island',
  'item',
  'jewel',
  'journey',
  'keeper',
  'key',
  'kind',
  'kit',
  'ladder',
  'lake',
  'launch',
  'layer',
  'leaf',
  'legend',
  'level',
  'light',
  'limit',
  'linen',
  'list',
  'logic',
  'lucky',
  'lumen',
  'maker',
  'map',
  'market',
  'matrix',
  'meadow',
  'memory',
  'metal',
  'method',
  'metric',
  'midday',
  'mind',
  'mirror',
  'model',
  'moment',
  'monsoon',
  'morning',
  'motion',
  'mountain',
  'music',
  'native',
  'nature',
  'network',
  'nexus',
  'night',
  'noble',
  'north',
  'note',
  'novel',
  'number',
  'oak',
  'object',
  'ocean',
  'offer',
  'omega',
  'onward',
  'orbit',
  'origin',
  'output',
  'packet',
  'page',
  'panel',
  'paper',
  'path',
  'pearl',
  'pencil',
  'pepper',
  'photo',
  'pixel',
  'planet',
  'plate',
  'point',
  'portal',
  'power',
  'prairie',
  'prime',
  'pulse',
  'quiet',
  'rapid',
  'reader',
  'record',
  'reef',
  'render',
  'report',
  'ribbon',
  'river',
  'rocket',
  'root',
  'round',
  'route',
  'sail',
  'sample',
  'scale',
  'scene',
  'screen',
  'script',
  'sea',
  'seed',
  'shadow',
  'signal',
  'silver',
  'simple',
  'sky',
  'smile',
  'snow',
  'solar',
  'source',
  'spark',
  'spirit',
  'spring',
  'square',
  'stable',
  'stage',
  'star',
  'stone',
  'storm',
  'story',
  'stream',
  'street',
  'studio',
  'summer',
  'sun',
  'sunset',
  'switch',
  'table',
  'target',
  'task',
  'tempo',
  'text',
  'thread',
  'timber',
  'today',
  'token',
  'tower',
  'trace',
  'track',
  'travel',
  'tree',
  'trust',
  'tunnel',
  'union',
  'unit',
  'update',
  'urban',
  'value',
  'vector',
  'velvet',
  'view',
  'vivid',
  'voice',
  'wave',
  'weather',
  'window',
  'winter',
  'wisdom',
  'wood',
  'world',
  'writer',
  'yard',
  'year',
  'yield',
  'young',
  'zenith',
  'zone',
];

Duration calculateReconnectDelay({
  required int reconnectAttempt,
  required Duration reconnectDelayBase,
  required int reconnectDelayMaxSeconds,
}) {
  final scaledMs = reconnectDelayBase.inMilliseconds *
      math.pow(2, reconnectAttempt - 1).toInt();
  final maxMs = reconnectDelayMaxSeconds * 1000;
  final delayMs = math.min(maxMs, math.max(1, scaledMs));
  return Duration(milliseconds: delayMs);
}

void main() {
  runApp(const TerminalClientApp());
}

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({
    super.key,
    this.clientFactory = TerminalControlGrpcClient.new,
    this.mediaEngineFactory = defaultClientMediaEngineFactory,
    this.heartbeatInterval = const Duration(seconds: 10),
    this.sensorTelemetryInterval = const Duration(seconds: 15),
    this.reconnectDelayBase = const Duration(seconds: 2),
    this.reconnectDelayMaxSeconds = 30,
  });

  final TerminalControlClientFactory clientFactory;
  final ClientMediaEngineFactory mediaEngineFactory;
  final Duration heartbeatInterval;
  final Duration sensorTelemetryInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: _ControlStreamScaffold(
        clientFactory: clientFactory,
        mediaEngineFactory: mediaEngineFactory,
        heartbeatInterval: heartbeatInterval,
        sensorTelemetryInterval: sensorTelemetryInterval,
        reconnectDelayBase: reconnectDelayBase,
        reconnectDelayMaxSeconds: reconnectDelayMaxSeconds,
      ),
    );
  }
}

class _ControlStreamScaffold extends StatefulWidget {
  const _ControlStreamScaffold({
    required this.clientFactory,
    required this.mediaEngineFactory,
    required this.heartbeatInterval,
    required this.sensorTelemetryInterval,
    required this.reconnectDelayBase,
    required this.reconnectDelayMaxSeconds,
  });

  final TerminalControlClientFactory clientFactory;
  final ClientMediaEngineFactory mediaEngineFactory;
  final Duration heartbeatInterval;
  final Duration sensorTelemetryInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;

  @override
  State<_ControlStreamScaffold> createState() => _ControlStreamScaffoldState();
}

class _ControlStreamScaffoldState extends State<_ControlStreamScaffold> {
  final TextEditingController _hostController = TextEditingController(
    text: '127.0.0.1',
  );
  final TextEditingController _portController = TextEditingController(
    text: '50051',
  );
  final TextEditingController _deviceNameController = TextEditingController(
    text: 'Flutter Client',
  );
  final TextEditingController _deviceTypeController = TextEditingController(
    text: 'desktop',
  );
  final TextEditingController _platformController = TextEditingController(
    text: 'flutter',
  );
  final MdnsScanner _mdnsScanner = MdnsScanner(serviceType: _mdnsServiceType);
  TerminalControlClient? _client;
  final StreamController<ConnectRequest> _outgoing =
      StreamController<ConnectRequest>.broadcast();

  StreamSubscription<ConnectResponse>? _incoming;
  String _status = 'Idle';
  int _responses = 0;
  final String _deviceId = 'flutter-${DateTime.now().millisecondsSinceEpoch}';
  late final ClientMediaEngine _mediaEngine;
  uiv1.Node? _activeRoot;
  int _activeRootRevision = 0;
  String _activeTransition = 'none';
  Duration _activeTransitionDuration = Duration.zero;
  String _lastNotification = '';
  final TextEditingController _terminalInputController =
      TextEditingController();
  final TextEditingController _playbackArtifactIdController =
      TextEditingController();
  final TextEditingController _playbackTargetDeviceIdController =
      TextEditingController();
  String _terminalInputShadow = '';
  bool _isScanning = false;
  List<DiscoveredServer> _discoveredServers = [];
  String? _selectedDiscoveredServer;
  Timer? _heartbeatTimer;
  Timer? _sensorTimer;
  Timer? _reconnectTimer;
  bool _shouldStayConnected = false;
  bool _isConnecting = false;
  int _reconnectAttempt = 0;
  final Map<String, iov1.StartStream> _activeStreamsByID =
      <String, iov1.StartStream>{};
  final Map<String, iov1.RouteStream> _routesByStreamID =
      <String, iov1.RouteStream>{};
  final List<WebRTCSignal> _recentWebRTCSignals = <WebRTCSignal>[];
  int _sensorSendCount = 0;
  int _streamReadyAckCount = 0;
  int _lastSensorSendUnixMs = 0;
  int _playAudioCount = 0;
  int _lastPlayAudioBytes = 0;
  String _lastPlayAudioDeviceID = 'none';
  String _lastPlayAudioSource = 'none';
  int _debugCommandSeq = 0;
  String _pendingRuntimeStatusRequestID = '';
  String _pendingDeviceStatusRequestID = '';
  String _pendingPlaybackArtifactsRequestID = '';
  String _pendingPlaybackMetadataRequestID = '';
  String _diagnosticsTitle = 'none';
  Map<String, String> _diagnosticsData = <String, String>{};
  String _photoFrameAssetBaseURL = '';
  final List<diagv1.UiEventEntry> _recentUiEvents = <diagv1.UiEventEntry>[];
  final List<diagv1.UiActionEntry> _recentUiActions = <diagv1.UiActionEntry>[];
  final List<diagv1.LogEntry> _recentLogs = <diagv1.LogEntry>[];
  final List<diagv1.ControlErrorEntry> _recentControlErrors =
      <diagv1.ControlErrorEntry>[];
  final Map<String, double> _lastSensorSnapshot = <String, double>{};
  capv1.DeviceCapabilities? _lastRegisteredCapabilities;
  int _lastHeartbeatUnixMs = 0;
  int _pendingHeartbeatUnixMs = 0;
  double _lastRttMs = 0;
  String _lastConnectionStatus = 'Idle';
  String _lastFlutterErrorMessage = '';
  String _lastFlutterErrorStack = '';
  int _lastFlutterErrorUnixMs = 0;
  String _lastBugTokenWord = '';
  String _lastBugTokenCode = '';
  final List<_QueuedBugReport> _queuedBugReports = <_QueuedBugReport>[];
  final List<_PendingBugReport> _pendingBugReports = <_PendingBugReport>[];
  Timer? _bugReportAckTimer;
  bool _hasRegisterAck = false;
  FlutterExceptionHandler? _previousFlutterErrorHandler;

  @override
  void initState() {
    super.initState();
    _installFlutterErrorHook();
    _mediaEngine = widget.mediaEngineFactory(
      localDeviceID: _deviceId,
      onSignal: _sendWebRTCSignalMessage,
    );
    _recordClientLog('info', 'client started');
    if (_e2eEmitEvents) {
      debugPrint('E2E_EVENT: client_started');
    }
    if (_e2eAutoScanConnect) {
      unawaited(_runE2EAutoConnectFlow());
    }
  }

  Future<void> _runE2EAutoConnectFlow() async {
    await Future<void>.delayed(Duration(milliseconds: _e2eStartupDelayMs));
    if (!mounted) {
      return;
    }
    if (_e2eEmitEvents) {
      debugPrint('E2E_EVENT: scanning_started');
    }
    await _scanForServers();
    if (!mounted) {
      return;
    }
    if (_discoveredServers.isEmpty) {
      if (_e2eEmitEvents) {
        debugPrint('E2E_EVENT: no_servers_discovered');
      }
      return;
    }
    if (_e2eEmitEvents) {
      debugPrint('E2E_EVENT: discovered_servers=${_discoveredServers.length}');
    }
    await _startStream(userInitiated: true);
  }

  Future<void> _startStream({bool userInitiated = true}) async {
    if (_isConnecting) {
      return;
    }
    if (userInitiated) {
      _shouldStayConnected = true;
      _reconnectAttempt = 0;
      _cancelReconnectTimer();
    }

    final host = _hostController.text.trim();
    final port = int.tryParse(_portController.text.trim());
    final mediaQuery = MediaQuery.of(context);
    final size = mediaQuery.size;
    if (host.isEmpty || port == null || port <= 0 || port > 65535) {
      _shouldStayConnected = false;
      _hasRegisterAck = false;
      setState(() {
        _status = 'Invalid host or port';
        _lastConnectionStatus = _status;
      });
      return;
    }

    _isConnecting = true;
    _hasRegisterAck = false;
    if (mounted) {
      setState(() {
        _status = userInitiated ? 'Connecting...' : 'Reconnecting...';
        _lastConnectionStatus = _status;
      });
    }
    try {
      await _incoming?.cancel();
      _incoming = null;
      final existingClient = _client;
      if (existingClient != null) {
        await existingClient.shutdown();
      }
      _client = widget.clientFactory(host: host, port: port);

      final stream = _client!.connect(_outgoing.stream);
      _incoming = stream.listen(
        (ConnectResponse response) {
          if (!mounted) {
            return;
          }
          final nowUnixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
          var shouldFlushQueuedBugReports = false;
          _reconnectAttempt = 0;
          setState(() {
            if (_pendingHeartbeatUnixMs > 0) {
              _lastRttMs = (nowUnixMs - _pendingHeartbeatUnixMs).toDouble();
              _pendingHeartbeatUnixMs = 0;
            }
            _responses += 1;
            final responseStatus = _statusFromResponse(response);
            if (responseStatus.isNotEmpty) {
              _status = responseStatus;
              _lastConnectionStatus = responseStatus;
            }
            var nextRoot = _activeRoot;
            var uiChanged = false;
            if (response.hasSetUi() && response.setUi.hasRoot()) {
              nextRoot = response.setUi.root.deepCopy();
              uiChanged = true;
              _recordUiEvent(
                kind: 'set_ui',
                componentId: _nodeId(response.setUi.root),
                detail: 'root updated',
              );
            }
            if (response.hasUpdateUi()) {
              final updatedRoot = _applyUpdateUi(
                currentRoot: nextRoot,
                update: response.updateUi,
              );
              if (!identical(updatedRoot, nextRoot)) {
                uiChanged = true;
              }
              nextRoot = updatedRoot;
              _recordUiEvent(
                kind: 'update_ui',
                componentId: response.updateUi.componentId,
                detail: 'component patch',
              );
            }
            if (response.hasTransitionUi()) {
              _applyTransitionHint(response.transitionUi);
              if (nextRoot != null) {
                uiChanged = true;
              }
              _recordUiEvent(
                kind: 'transition_ui',
                componentId: _nodeId(nextRoot ?? uiv1.Node()),
                detail: response.transitionUi.transition,
              );
            }
            _activeRoot = nextRoot;
            if (uiChanged) {
              _activeRootRevision += 1;
            }
            if (response.hasNotification()) {
              _lastNotification = response.notification.body;
            }
            if (response.hasCommandResult() &&
                response.commandResult.notification.isNotEmpty) {
              _lastNotification = response.commandResult.notification;
            }
            _applyDiagnosticsResponse(response);
            if (response.hasError()) {
              _lastNotification = response.error.message;
              _appendBounded<diagv1.ControlErrorEntry>(
                _recentControlErrors,
                diagv1.ControlErrorEntry()
                  ..unixMs = Int64(nowUnixMs)
                  ..code = response.error.code.name
                  ..message = response.error.message,
                _clientContextRecentErrorCap,
              );
              _recordClientLog('error', response.error.message);
            }
            if (response.hasBugReportAck()) {
              _handleBugReportAck(response.bugReportAck);
            }
            if (response.hasRegisterAck()) {
              if (!_hasRegisterAck) {
                shouldFlushQueuedBugReports = true;
              }
              _hasRegisterAck = true;
            }
            _applyRegisterMetadata(response);
            if (_e2eEmitEvents && response.hasRegisterAck()) {
              debugPrint('E2E_EVENT: register_ack');
            }
            _applyMediaControlResponse(response);
          });
          if (shouldFlushQueuedBugReports) {
            _flushQueuedBugReports();
          }
        },
        onError: (Object error) {
          unawaited(_handleStreamClosed('Stream error: $error'));
        },
        onDone: () {
          unawaited(_handleStreamClosed('Disconnected'));
        },
      );

      _startHeartbeatLoop();
      _startSensorTelemetryLoop();
      final registerRequest = TerminalControlGrpcClient.registerRequest(
        deviceId: _deviceId,
        deviceName: _deviceNameController.text.trim(),
        deviceType: _deviceTypeController.text.trim(),
        platform: _platformController.text.trim(),
        screenWidth: size.width.round(),
        screenHeight: size.height.round(),
        screenDensity: mediaQuery.devicePixelRatio,
        screenTouch: true,
      );
      _lastRegisteredCapabilities = registerRequest.register.capabilities;
      _outgoing.add(registerRequest);
      final initialHeartbeatUnixMs =
          DateTime.now().toUtc().millisecondsSinceEpoch;
      _lastHeartbeatUnixMs = initialHeartbeatUnixMs;
      _pendingHeartbeatUnixMs = initialHeartbeatUnixMs;
      _outgoing.add(
        TerminalControlGrpcClient.heartbeatRequest(
          deviceId: _deviceId,
          unixMs: initialHeartbeatUnixMs,
        ),
      );
      _recordClientLog('info', 'control stream connected');
      _sendSensorTelemetry();
    } catch (error) {
      await _handleStreamClosed('Connection error: $error');
    } finally {
      _isConnecting = false;
    }
  }

  void _startHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(widget.heartbeatInterval, (_) {
      if (!_shouldStayConnected || _deviceId.isEmpty) {
        return;
      }
      final unixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
      _lastHeartbeatUnixMs = unixMs;
      _pendingHeartbeatUnixMs = unixMs;
      _outgoing.add(
        TerminalControlGrpcClient.heartbeatRequest(
          deviceId: _deviceId,
          unixMs: unixMs,
        ),
      );
    });
  }

  void _startSensorTelemetryLoop() {
    _sensorTimer?.cancel();
    _sensorTimer = Timer.periodic(widget.sensorTelemetryInterval, (_) {
      if (!_shouldStayConnected || _deviceId.isEmpty) {
        return;
      }
      _sendSensorTelemetry();
    });
  }

  ConnectRequest _buildSensorTelemetryRequest() {
    final now = DateTime.now().toUtc();
    final values = <String, double>{
      'battery.level': 1.0,
      'battery.charging': 1.0,
      'connectivity.online': 1.0,
      'connectivity.reconnect_attempt': _reconnectAttempt.toDouble(),
      'time.utc_hour': now.hour.toDouble(),
      'time.utc_weekday': now.weekday.toDouble(),
      'time.utc_minute': now.minute.toDouble(),
    };
    return ConnectRequest()
      ..sensor = (iov1.SensorData()
        ..deviceId = _deviceId
        ..unixMs = Int64(now.millisecondsSinceEpoch)
        ..values.addAll(values));
  }

  void _sendSensorTelemetry() {
    if (_deviceId.isEmpty) {
      return;
    }
    final request = _buildSensorTelemetryRequest();
    _lastSensorSnapshot
      ..clear()
      ..addAll(request.sensor.values);
    _outgoing.add(request);
    final unixMs = request.sensor.unixMs.toInt();
    if (mounted) {
      setState(() {
        _sensorSendCount += 1;
        _lastSensorSendUnixMs = unixMs;
      });
      return;
    }
    _sensorSendCount += 1;
    _lastSensorSendUnixMs = unixMs;
  }

  String _nextDebugRequestID(String prefix) {
    _debugCommandSeq += 1;
    return '$prefix-$_debugCommandSeq';
  }

  void _sendRuntimeStatusQuery() {
    final requestID = _nextDebugRequestID('debug-runtime-status');
    _pendingRuntimeStatusRequestID = requestID;
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = requestID
          ..kind = CommandKind.COMMAND_KIND_SYSTEM
          ..intent = 'runtime_status'),
    );
  }

  void _sendDeviceStatusQuery() {
    final requestID = _nextDebugRequestID('debug-device-status');
    _pendingDeviceStatusRequestID = requestID;
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = requestID
          ..kind = CommandKind.COMMAND_KIND_SYSTEM
          ..intent = 'device_status $_deviceId'),
    );
  }

  void _sendPlaybackArtifactsQuery() {
    final requestID = _nextDebugRequestID('debug-playback-artifacts');
    _pendingPlaybackArtifactsRequestID = requestID;
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = requestID
          ..kind = CommandKind.COMMAND_KIND_SYSTEM
          ..intent = 'list_playback_artifacts'),
    );
  }

  void _sendPlaybackMetadataQuery() {
    final artifactID = _playbackArtifactIdController.text.trim();
    if (artifactID.isEmpty) {
      setState(() {
        _status = 'Command error';
        _lastNotification = 'Playback artifact ID required';
      });
      return;
    }
    var targetDeviceID = _playbackTargetDeviceIdController.text.trim();
    if (targetDeviceID.isEmpty) {
      targetDeviceID = _deviceId;
      _playbackTargetDeviceIdController.text = targetDeviceID;
    }
    final requestID = _nextDebugRequestID('debug-playback-metadata');
    _pendingPlaybackMetadataRequestID = requestID;
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = requestID
          ..deviceId = _deviceId
          ..kind = CommandKind.COMMAND_KIND_MANUAL
          ..intent = 'playback_metadata'
          ..arguments['artifact_id'] = artifactID
          ..arguments['target_device_id'] = targetDeviceID),
    );
  }

  String _firstPlaybackArtifactID(Map<String, String> data) {
    final keys = data.keys.toList()..sort();
    for (final key in keys) {
      final parts = data[key]?.split('|') ?? const <String>[];
      if (parts.isNotEmpty && parts.first.trim().isNotEmpty) {
        return parts.first.trim();
      }
    }
    return '';
  }

  void _applyDiagnosticsResponse(ConnectResponse response) {
    if (!response.hasCommandResult()) {
      return;
    }
    final result = response.commandResult;
    if (result.data.isEmpty) {
      return;
    }

    final requestID = result.requestId;
    var diagnosticsTitle = '';
    if (requestID.isNotEmpty && requestID == _pendingRuntimeStatusRequestID) {
      diagnosticsTitle = 'runtime_status';
    } else if (requestID.isNotEmpty &&
        requestID == _pendingDeviceStatusRequestID) {
      diagnosticsTitle = 'device_status';
    } else if (requestID.isNotEmpty &&
        requestID == _pendingPlaybackArtifactsRequestID) {
      diagnosticsTitle = 'list_playback_artifacts';
    } else if (requestID.isNotEmpty &&
        requestID == _pendingPlaybackMetadataRequestID) {
      diagnosticsTitle = 'playback_metadata';
    } else if (result.notification == 'System query: runtime_status') {
      diagnosticsTitle = 'runtime_status';
    } else if (result.notification == 'System query: device_status') {
      diagnosticsTitle = 'device_status';
    } else if (result.notification == 'System query: list_playback_artifacts') {
      diagnosticsTitle = 'list_playback_artifacts';
    } else if (result.notification == 'Playback metadata ready') {
      diagnosticsTitle = 'playback_metadata';
    } else {
      return;
    }

    final data = Map<String, String>.from(result.data);
    _diagnosticsTitle = diagnosticsTitle;
    _diagnosticsData = data;
    if (diagnosticsTitle == 'list_playback_artifacts') {
      final firstArtifactID = _firstPlaybackArtifactID(data);
      if (firstArtifactID.isNotEmpty) {
        _playbackArtifactIdController.text = firstArtifactID;
      }
    }
  }

  void _applyRegisterMetadata(ConnectResponse response) {
    if (!response.hasRegisterAck()) {
      return;
    }
    final metadata = Map<String, String>.from(response.registerAck.metadata);
    if (metadata.isEmpty) {
      return;
    }
    _diagnosticsTitle = 'register_ack';
    _diagnosticsData = metadata;
    final photoBaseURL = metadata['photo_frame_asset_base_url']?.trim() ?? '';
    if (photoBaseURL.isNotEmpty) {
      _photoFrameAssetBaseURL = photoBaseURL;
      _lastNotification = 'Photo frame asset base URL configured';
    }
  }

  void _installFlutterErrorHook() {
    _previousFlutterErrorHandler = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      _lastFlutterErrorMessage = details.exceptionAsString();
      _lastFlutterErrorStack = details.stack?.toString() ?? '';
      _lastFlutterErrorUnixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
      _recordClientLog('error', _lastFlutterErrorMessage);
      final prior = _previousFlutterErrorHandler;
      if (prior != null) {
        prior(details);
      } else {
        FlutterError.presentError(details);
      }
    };
  }

  void _restoreFlutterErrorHook() {
    if (FlutterError.onError == null) {
      return;
    }
    FlutterError.onError = _previousFlutterErrorHandler;
    _previousFlutterErrorHandler = null;
  }

  void _appendBounded<T>(List<T> items, T next, int maxItems) {
    items.add(next);
    if (items.length > maxItems) {
      items.removeRange(0, items.length - maxItems);
    }
  }

  void _recordClientLog(String level, String message) {
    final entry = diagv1.LogEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..level = level
      ..message = message;
    _appendBounded<diagv1.LogEntry>(
      _recentLogs,
      entry,
      _clientContextRecentLogCap,
    );
  }

  void _recordUiEvent({
    required String kind,
    required String componentId,
    required String detail,
  }) {
    final entry = diagv1.UiEventEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..kind = kind
      ..componentId = componentId
      ..detail = detail;
    _appendBounded<diagv1.UiEventEntry>(
      _recentUiEvents,
      entry,
      _clientContextRecentUiCap,
    );
  }

  void _recordUiAction({
    required String componentId,
    required String action,
    required String value,
  }) {
    final entry = diagv1.UiActionEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..componentId = componentId
      ..action = action
      ..value = value;
    _appendBounded<diagv1.UiActionEntry>(
      _recentUiActions,
      entry,
      _clientContextRecentUiCap,
    );
  }

  diagv1.ClientContext _buildClientContext() {
    final dispatcher = WidgetsBinding.instance.platformDispatcher;
    final locale = dispatcher.locale;
    final timezone = dispatcher.locale.toLanguageTag().isNotEmpty
        ? DateTime.now().timeZoneName
        : '';
    final mediaQuery = MediaQuery.maybeOf(context);
    final size = mediaQuery?.size;
    final devicePixelRatio = mediaQuery?.devicePixelRatio ?? 1.0;
    final orientation = mediaQuery?.orientation.name ?? 'unknown';

    final runtime = diagv1.RuntimeState()
      ..activeUiRoot = (_activeRoot?.deepCopy() ?? uiv1.Node())
      ..recentUiUpdates.addAll(_recentUiEvents.map((item) => item.deepCopy()))
      ..recentUiActions.addAll(_recentUiActions.map((item) => item.deepCopy()))
      ..recentLogs.addAll(_recentLogs.map((item) => item.deepCopy()));

    _activeStreamsByID.forEach((streamID, start) {
      runtime.activeStreams.add(
        diagv1.StreamEntry()
          ..streamId = streamID
          ..kind = start.kind
          ..sourceDeviceId = start.sourceDeviceId
          ..targetDeviceId = start.targetDeviceId,
      );
    });
    _routesByStreamID.forEach((streamID, route) {
      runtime.activeRoutes.add(
        diagv1.RouteEntry()
          ..streamId = streamID
          ..sourceDeviceId = route.sourceDeviceId
          ..targetDeviceId = route.targetDeviceId
          ..kind = route.kind,
      );
    });
    for (final signal in _recentWebRTCSignals) {
      runtime.recentWebrtcSignals.add(
        diagv1.WebrtcSignalEntry()
          ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
          ..streamId = signal.streamId
          ..signalType = signal.signalType,
      );
    }

    final connection = diagv1.ConnectionHealth()
      ..lastHeartbeatUnixMs = Int64(_lastHeartbeatUnixMs)
      ..reconnectAttempt = _reconnectAttempt
      ..lastStatus = _lastConnectionStatus
      ..lastRttMs = _lastRttMs
      ..online = _shouldStayConnected;
    connection.recentControlErrors
        .addAll(_recentControlErrors.map((item) => item.deepCopy()));

    final hardware = diagv1.HardwareState()
      ..batteryLevel = (_lastSensorSnapshot['battery.level'] ?? 0).toDouble()
      ..batteryCharging =
          (_lastSensorSnapshot['battery.charging'] ?? 0).toDouble() >= 0.5
      ..screenWidthPx = size?.width.round() ?? 0
      ..screenHeightPx = size?.height.round() ?? 0
      ..devicePixelRatio = devicePixelRatio
      ..orientation = orientation;
    hardware.sensorSnapshot.addAll(_lastSensorSnapshot);

    final contextProto = diagv1.ClientContext()
      ..identity = (diagv1.ClientIdentity()
        ..deviceId = _deviceId
        ..deviceName = _deviceNameController.text.trim()
        ..deviceType = _deviceTypeController.text.trim()
        ..platform = _platformController.text.trim()
        ..clientVersion = const String.fromEnvironment(
          'TERMINALS_CLIENT_VERSION',
        )
        ..clientGitSha = const String.fromEnvironment('TERMINALS_GIT_SHA')
        ..clientBuildUnixMs = Int64(
          int.tryParse(
                const String.fromEnvironment('TERMINALS_BUILD_UNIX_MS'),
              ) ??
              0,
        )
        ..osVersion = '${defaultTargetPlatform.name}${kIsWeb ? ':web' : ''}'
        ..locale = locale.toLanguageTag()
        ..timezone = timezone
        ..clockOffsetMs = Int64(0))
      ..runtime = runtime
      ..connection = connection
      ..hardware = hardware
      ..errorCapture = (diagv1.ErrorCapture()
        ..lastErrorMessage = _lastFlutterErrorMessage
        ..lastErrorStack = _lastFlutterErrorStack
        ..lastErrorUnixMs = Int64(_lastFlutterErrorUnixMs));
    if (_lastRegisteredCapabilities != null) {
      contextProto.capabilities = _lastRegisteredCapabilities!.deepCopy();
    }
    return contextProto;
  }

  Future<void> _submitBugReportFromAction({
    required String componentId,
    required String action,
    required String value,
  }) async {
    var subjectDeviceID = _deviceId;
    final parts = action.split(':');
    if (parts.length > 1) {
      final explicit = parts.sublist(1).join(':').trim();
      if (explicit.isNotEmpty) {
        subjectDeviceID = explicit;
      }
    } else if (value.trim().isNotEmpty) {
      subjectDeviceID = value.trim();
    }

    await _submitBugReport(
      subjectDeviceID: subjectDeviceID,
      description: 'Filed from on-device bug report button',
      source: diagv1.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
      sourceHints: <String, String>{
        'component_id': componentId,
        'action': action,
      },
    );
  }

  Future<void> _submitBugReport({
    required String subjectDeviceID,
    required String description,
    required diagv1.BugReportSource source,
    Map<String, String> sourceHints = const <String, String>{},
    List<String> tags = const <String>[],
    _BugIdentifier? bugIdentifier,
  }) async {
    final now = DateTime.now().toLocal();
    final identifier = bugIdentifier ?? _buildBugIdentifier(now);
    final bugReport = diagv1.BugReport()
      ..reporterDeviceId = _deviceId
      ..subjectDeviceId = subjectDeviceID
      ..source = source
      ..description = description
      ..timestampUnixMs = Int64(now.toUtc().millisecondsSinceEpoch)
      ..clientContext = _buildClientContext();
    bugReport.tags.addAll(<String>[
      ...tags,
      'bug_word:${identifier.word}',
      'bug_code:${identifier.code}',
    ]);
    bugReport.sourceHints['bug_token_word'] = identifier.word;
    bugReport.sourceHints['bug_token_code'] = identifier.code;
    bugReport.sourceHints['bug_token_timestamp_unix_ms'] =
        now.toUtc().millisecondsSinceEpoch.toString();
    _buildAutomaticBugSourceHints().forEach((key, value) {
      if (value.isEmpty) {
        return;
      }
      bugReport.sourceHints[key] = value;
    });
    sourceHints.forEach((key, value) {
      bugReport.sourceHints[key] = value;
    });

    if (!mounted) {
      return;
    }
    if (_isBugReportTransportReady()) {
      _dispatchBugReport(
        bugReport: bugReport,
        identifier: identifier,
        subjectDeviceID: subjectDeviceID,
        replay: false,
      );
      setState(() {
        _lastBugTokenWord = identifier.word;
        _lastBugTokenCode = identifier.code;
        _status = 'Bug report pending';
        _lastNotification =
            'Submitting bug report (word: ${identifier.word}, code: ${identifier.code}) and waiting for server ack...';
      });
      return;
    }

    _queuedBugReports.add(
      _QueuedBugReport(
        bugReport: bugReport,
        identifier: identifier,
      ),
    );
    final hasActiveStreamAttempt =
        _shouldStayConnected && _client != null && _incoming != null;
    if (!hasActiveStreamAttempt && !_isConnecting) {
      unawaited(_startStream(userInitiated: true));
    }
    _recordClientLog(
      'warn',
      'queued bug report while transport not ready for subject=$subjectDeviceID '
          'word=${identifier.word} code=${identifier.code}',
    );
    setState(() {
      _lastBugTokenWord = identifier.word;
      _lastBugTokenCode = identifier.code;
      _status = 'Bug report queued';
      _lastNotification =
          'Bug report queued (word: ${identifier.word}, code: ${identifier.code}). Connecting and will send automatically.';
    });
  }

  Future<void> _showBugReportDialog() async {
    final descriptionController = TextEditingController();
    final tagsController = TextEditingController();
    var draftIdentifier = _buildBugIdentifier(DateTime.now().toLocal());
    final draft = await showDialog<_BugReportDraft>(
      context: context,
      builder: (context) {
        return StatefulBuilder(
          builder: (context, setDialogState) {
            return AlertDialog(
              title: const Text('Report a bug'),
              content: SizedBox(
                width: 440,
                child: SingleChildScrollView(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('No fields are required.'),
                      const SizedBox(height: 12),
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          border: Border.all(color: Colors.blueGrey.shade200),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'Reference word',
                              style: TextStyle(fontWeight: FontWeight.w600),
                            ),
                            const SizedBox(height: 6),
                            Text(
                              draftIdentifier.word,
                              style: Theme.of(context).textTheme.headlineSmall,
                            ),
                            SelectableText(draftIdentifier.code),
                            const SizedBox(height: 8),
                            Center(
                              child: QrImageView(
                                data: draftIdentifier.qrPayload,
                                size: 140,
                              ),
                            ),
                            const SizedBox(height: 8),
                            Wrap(
                              spacing: 8,
                              children: [
                                OutlinedButton.icon(
                                  onPressed: () =>
                                      _announceBugIdentifier(draftIdentifier),
                                  icon: const Icon(Icons.volume_up_outlined),
                                  label: const Text('Speak'),
                                ),
                                OutlinedButton.icon(
                                  onPressed: () {
                                    setDialogState(() {
                                      draftIdentifier = _buildBugIdentifier(
                                        DateTime.now().toLocal(),
                                      );
                                    });
                                  },
                                  icon: const Icon(Icons.refresh_outlined),
                                  label: const Text('Refresh'),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: descriptionController,
                        minLines: 2,
                        maxLines: 4,
                        decoration: const InputDecoration(
                          labelText: 'Description (optional)',
                          hintText: 'What happened? (optional)',
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: tagsController,
                        decoration: const InputDecoration(
                          labelText: 'Tags (optional)',
                          hintText: 'ui, playback, audio',
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('Cancel'),
                ),
                FilledButton(
                  onPressed: () {
                    Navigator.of(context).pop(
                      _BugReportDraft(
                        description: descriptionController.text.trim(),
                        tags: tagsController.text
                            .split(',')
                            .map((value) => value.trim())
                            .where((value) => value.isNotEmpty)
                            .toList(),
                        identifier: draftIdentifier,
                      ),
                    );
                  },
                  child: const Text('Submit'),
                ),
              ],
            );
          },
        );
      },
    );

    descriptionController.dispose();
    tagsController.dispose();
    if (draft == null) {
      return;
    }
    _announceBugIdentifier(draft.identifier);
    await _submitBugReport(
      subjectDeviceID: _deviceId,
      description: draft.description.isNotEmpty
          ? draft.description
          : 'Filed from terminal client bug-report dialog',
      source: diagv1.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
      sourceHints: const <String, String>{
        'entry_point': 'manual_bug_report_dialog',
      },
      tags: draft.tags,
      bugIdentifier: draft.identifier,
    );
  }

  _BugIdentifier _buildBugIdentifier(DateTime nowLocal) {
    final secondsFromMidnight =
        nowLocal.hour * 3600 + nowLocal.minute * 60 + nowLocal.second;
    final daySalt = nowLocal.year * 10000 + nowLocal.month * 100 + nowLocal.day;
    final index =
        ((secondsFromMidnight ~/ 17) + daySalt) % _bugTokenWords.length;
    final word = _bugTokenWords[index];
    final hh = nowLocal.hour.toString().padLeft(2, '0');
    final mm = nowLocal.minute.toString().padLeft(2, '0');
    final ss = nowLocal.second.toString().padLeft(2, '0');
    final code = '$hh$mm$ss-$word';
    return _BugIdentifier(
      word: word,
      code: code,
      qrPayload: 'terminals-bug://$code',
    );
  }

  void _announceBugIdentifier(_BugIdentifier identifier) {
    speech.speakText(
        'Bug reference word ${identifier.word}. Code ${identifier.code}');
    if (!mounted) {
      return;
    }
    final direction = Directionality.of(context);
    SemanticsService.announce(
      'Bug reference word ${identifier.word}. Code ${identifier.code}',
      direction,
    );
  }

  Map<String, String> _buildAutomaticBugSourceHints() {
    return <String, String>{
      'host': _hostController.text.trim(),
      'port': _portController.text.trim(),
      'status': _status,
      'last_connection_status': _lastConnectionStatus,
      'active_ui_root': _activeRoot == null ? '' : _nodeId(_activeRoot!),
      'active_stream_count': _activeStreamsByID.length.toString(),
      'route_count': _routesByStreamID.length.toString(),
      'reconnect_attempt': _reconnectAttempt.toString(),
      'last_notification': _lastNotification,
      'page_url': Uri.base.toString(),
      'active_root_revision': _activeRootRevision.toString(),
      'responses_seen': _responses.toString(),
      'queued_bug_reports': _queuedBugReports.length.toString(),
      'pending_bug_reports': _pendingBugReports.length.toString(),
    };
  }

  void _handleBugReportAck(diagv1.BugReportAck ack) {
    _PendingBugReport? pending;
    if (_pendingBugReports.isNotEmpty) {
      pending = _pendingBugReports.removeAt(0);
    }
    final tokenWord = pending?.identifier.word ?? _lastBugTokenWord;
    final tokenCode = pending?.identifier.code ?? _lastBugTokenCode;
    _lastBugTokenWord = tokenWord;
    _lastBugTokenCode = tokenCode;
    if (_pendingBugReports.isEmpty) {
      _bugReportAckTimer?.cancel();
      _bugReportAckTimer = null;
    }
    _status = 'Bug report filed';
    _lastNotification = tokenWord.isEmpty
        ? 'Bug report filed: ${ack.reportId}'
        : 'Bug report filed: ${ack.reportId} (word: $tokenWord)';
    _recordClientLog(
      'info',
      'bug report ack status=${ack.status.name} id=${ack.reportId} '
          'word=$tokenWord code=$tokenCode',
    );
  }

  bool _isBugReportTransportReady() {
    return _client != null &&
        _incoming != null &&
        _shouldStayConnected &&
        _hasRegisterAck;
  }

  void _dispatchBugReport({
    required diagv1.BugReport bugReport,
    required _BugIdentifier identifier,
    required String subjectDeviceID,
    required bool replay,
  }) {
    _outgoing.add(
      ConnectRequest()..bugReport = bugReport,
    );
    _pendingBugReports.add(
      _PendingBugReport(
        identifier: identifier,
        submittedUnixMs: DateTime.now().toUtc().millisecondsSinceEpoch,
      ),
    );
    _ensureBugReportAckWatchdog();
    _recordClientLog(
      'info',
      '${replay ? 'replayed' : 'submitted'} bug report for subject=$subjectDeviceID '
          'word=${identifier.word} code=${identifier.code}',
    );
  }

  void _flushQueuedBugReports() {
    if (!_isBugReportTransportReady()) {
      return;
    }
    if (_queuedBugReports.isEmpty) {
      return;
    }
    final queued = List<_QueuedBugReport>.from(_queuedBugReports);
    _queuedBugReports.clear();
    for (final item in queued) {
      _dispatchBugReport(
        bugReport: item.bugReport,
        identifier: item.identifier,
        subjectDeviceID: item.bugReport.subjectDeviceId,
        replay: true,
      );
    }
    if (mounted) {
      setState(() {
        _status = 'Queued bug reports sent';
        _lastNotification =
            'Sent ${queued.length} queued bug report(s); waiting for server ack.';
      });
    }
  }

  void _ensureBugReportAckWatchdog() {
    if (_bugReportAckTimer != null) {
      return;
    }
    _bugReportAckTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (_pendingBugReports.isEmpty) {
        _bugReportAckTimer?.cancel();
        _bugReportAckTimer = null;
        return;
      }
      final nowUnixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
      final first = _pendingBugReports.first;
      if (nowUnixMs - first.submittedUnixMs <
          _bugReportAckTimeout.inMilliseconds) {
        return;
      }
      final failed = _pendingBugReports.removeAt(0);
      _lastBugTokenWord = failed.identifier.word;
      _lastBugTokenCode = failed.identifier.code;
      if (mounted) {
        setState(() {
          _status = 'Bug report not confirmed';
          _lastNotification =
              'Bug report not confirmed by server (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
        });
      }
      _recordClientLog(
        'error',
        'bug report ack timeout word=${failed.identifier.word} code=${failed.identifier.code}',
      );
      if (_pendingBugReports.isEmpty) {
        _bugReportAckTimer?.cancel();
        _bugReportAckTimer = null;
      }
    });
  }

  void _drainPendingBugReportsAsFailed(String reason) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final failed = _pendingBugReports.removeAt(0);
    _lastBugTokenWord = failed.identifier.word;
    _lastBugTokenCode = failed.identifier.code;
    _recordClientLog(
      'error',
      'bug report failed word=${failed.identifier.word} code=${failed.identifier.code} reason=$reason',
    );
    if (mounted) {
      setState(() {
        _status = 'Bug report not confirmed';
        _lastNotification =
            'Bug report not confirmed (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
      });
    }
    _pendingBugReports.clear();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
  }

  void _stopHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  void _stopSensorTelemetryLoop() {
    _sensorTimer?.cancel();
    _sensorTimer = null;
  }

  void _cancelReconnectTimer() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }

  Future<void> _handleStreamClosed(String status) async {
    _recordClientLog('warn', 'control stream closed: $status');
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    _incoming = null;
    _hasRegisterAck = false;
    _drainPendingBugReportsAsFailed('stream closed before bug report ack');
    final existingClient = _client;
    _client = null;
    if (existingClient != null) {
      await existingClient.shutdown();
    }
    if (mounted) {
      setState(() {
        _status = status;
        _lastConnectionStatus = status;
      });
    }
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (!_shouldStayConnected) {
      return;
    }
    if (_isConnecting) {
      if (_reconnectTimer?.isActive ?? false) {
        return;
      }
      _reconnectTimer = Timer(const Duration(milliseconds: 50), () {
        _reconnectTimer = null;
        _scheduleReconnect();
      });
      return;
    }
    if (_reconnectTimer?.isActive ?? false) {
      return;
    }
    _reconnectAttempt += 1;
    final reconnectDelay = calculateReconnectDelay(
      reconnectAttempt: _reconnectAttempt,
      reconnectDelayBase: widget.reconnectDelayBase,
      reconnectDelayMaxSeconds: widget.reconnectDelayMaxSeconds,
    );
    final displaySeconds = reconnectDelay.inMilliseconds <= 1000
        ? 1
        : (reconnectDelay.inMilliseconds / 1000).ceil();
    if (mounted) {
      setState(() {
        _status = 'Connection lost, retrying in ${displaySeconds}s...';
        _lastConnectionStatus = _status;
      });
    }
    _reconnectTimer = Timer(reconnectDelay, () {
      _reconnectTimer = null;
      if (!_shouldStayConnected || !mounted) {
        return;
      }
      unawaited(_startStream(userInitiated: false));
    });
  }

  Future<void> _scanForServers() async {
    if (_isScanning) {
      return;
    }
    setState(() {
      _isScanning = true;
      _status = 'Scanning LAN for server...';
      _lastConnectionStatus = _status;
    });
    try {
      final found = await _mdnsScanner.scan();
      if (!mounted) {
        return;
      }
      setState(() {
        _discoveredServers = found;
        if (found.isNotEmpty) {
          _selectedDiscoveredServer = '${found.first.host}:${found.first.port}';
          _hostController.text = found.first.host;
          _portController.text = found.first.port.toString();
          _status = 'Found ${found.length} server(s)';
          _lastConnectionStatus = _status;
        } else {
          _selectedDiscoveredServer = null;
          _status = 'No servers discovered';
          _lastConnectionStatus = _status;
        }
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Discovery error: $error';
        _lastConnectionStatus = _status;
      });
    } finally {
      if (mounted) {
        setState(() {
          _isScanning = false;
        });
      }
    }
  }

  Future<void> _stopStream() async {
    _shouldStayConnected = false;
    _hasRegisterAck = false;
    _reconnectAttempt = 0;
    _cancelReconnectTimer();
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    _drainPendingBugReportsAsFailed('stream stopped before bug report ack');
    await _incoming?.cancel();
    _incoming = null;
    for (final streamID in _activeStreamsByID.keys.toList(growable: false)) {
      await _mediaEngine.stopStream(streamID);
    }
    _activeStreamsByID.clear();
    _routesByStreamID.clear();
    final existingClient = _client;
    if (existingClient != null) {
      await existingClient.shutdown();
      _client = null;
    }
    if (mounted) {
      setState(() {
        _status = 'Disconnected';
        _lastConnectionStatus = _status;
        _activeRoot = null;
      });
    }
  }

  @override
  void dispose() {
    _shouldStayConnected = false;
    _cancelReconnectTimer();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    final incoming = _incoming;
    if (incoming != null) {
      unawaited(incoming.cancel());
    }
    _outgoing.close();
    unawaited(_mediaEngine.dispose());
    final existingClient = _client;
    if (existingClient != null) {
      unawaited(existingClient.shutdown());
    }
    _hostController.dispose();
    _portController.dispose();
    _deviceNameController.dispose();
    _deviceTypeController.dispose();
    _platformController.dispose();
    _terminalInputController.dispose();
    _playbackArtifactIdController.dispose();
    _playbackTargetDeviceIdController.dispose();
    _restoreFlutterErrorHook();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _showBugReportDialog,
        icon: const Icon(Icons.bug_report_outlined),
        label: const Text('Report Bug'),
      ),
      body: Align(
        alignment: Alignment.topCenter,
        child: SingleChildScrollView(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: _hostController,
                  decoration: const InputDecoration(
                    labelText: 'Server Host',
                    hintText: '127.0.0.1',
                  ),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    ElevatedButton(
                      onPressed: _isScanning ? null : _scanForServers,
                      child: Text(_isScanning ? 'Scanning...' : 'Scan LAN'),
                    ),
                  ],
                ),
                if (_discoveredServers.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    key: ValueKey<String?>(_selectedDiscoveredServer),
                    initialValue: _selectedDiscoveredServer,
                    decoration:
                        const InputDecoration(labelText: 'Discovered Server'),
                    items: _discoveredServers
                        .map(
                          (server) => DropdownMenuItem<String>(
                            value: '${server.host}:${server.port}',
                            child: Text(
                              '${server.name} (${server.host}:${server.port})',
                            ),
                          ),
                        )
                        .toList(),
                    onChanged: (value) {
                      if (value == null) {
                        return;
                      }
                      final parts = value.split(':');
                      if (parts.length != 2) {
                        return;
                      }
                      setState(() {
                        _selectedDiscoveredServer = value;
                        _hostController.text = parts[0];
                        _portController.text = parts[1];
                      });
                    },
                  ),
                ],
                const SizedBox(height: 12),
                TextField(
                  controller: _portController,
                  decoration: const InputDecoration(labelText: 'Server Port'),
                  keyboardType: TextInputType.number,
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _deviceNameController,
                  decoration: const InputDecoration(labelText: 'Device Name'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _deviceTypeController,
                  decoration: const InputDecoration(labelText: 'Device Type'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _platformController,
                  decoration: const InputDecoration(labelText: 'Platform'),
                ),
                const SizedBox(height: 20),
                Text('Control Stream: $_status'),
                const SizedBox(height: 12),
                Text('Responses: $_responses'),
                Text(
                  'Media routes: ${_routesByStreamID.length}  Active streams: ${_activeStreamsByID.length}  Signals: ${_recentWebRTCSignals.length}',
                ),
                Text(
                  'Sensor sends: $_sensorSendCount  Last sensor unix_ms: $_lastSensorSendUnixMs  Stream-ready acks: $_streamReadyAckCount',
                ),
                if (_playAudioCount > 0)
                  Text(
                    'Play audio msgs: $_playAudioCount  Last play bytes: $_lastPlayAudioBytes  Last play target: $_lastPlayAudioDeviceID  Last play source: $_lastPlayAudioSource',
                  ),
                if (_photoFrameAssetBaseURL.isNotEmpty)
                  Text('Photo frame assets: $_photoFrameAssetBaseURL'),
                if (_lastNotification.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Text('Notification: $_lastNotification'),
                ],
                const SizedBox(height: 20),
                Wrap(
                  spacing: 12,
                  children: [
                    ElevatedButton(
                      onPressed: _startStream,
                      child: const Text('Connect Stream'),
                    ),
                    OutlinedButton(
                      onPressed: _stopStream,
                      child: const Text('Disconnect'),
                    ),
                    OutlinedButton(
                      onPressed: _sendRuntimeStatusQuery,
                      child: const Text('Runtime Status'),
                    ),
                    OutlinedButton(
                      onPressed: _sendDeviceStatusQuery,
                      child: const Text('Device Status'),
                    ),
                    OutlinedButton(
                      onPressed: _sendPlaybackArtifactsQuery,
                      child: const Text('List Playback Artifacts'),
                    ),
                    OutlinedButton(
                      onPressed: _sendPlaybackMetadataQuery,
                      child: const Text('Playback Metadata'),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _playbackArtifactIdController,
                  decoration: const InputDecoration(
                    labelText: 'Playback Artifact ID',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _playbackTargetDeviceIdController,
                  decoration: const InputDecoration(
                    labelText: 'Playback Target Device ID',
                    hintText: 'Defaults to this device',
                  ),
                ),
                const SizedBox(height: 12),
                _buildDiagnosticsPanel(),
                if (_activeRoot != null) ...[
                  const SizedBox(height: 24),
                  const Divider(),
                  const SizedBox(height: 12),
                  SizedBox(
                    height: 320,
                    child: AnimatedSwitcher(
                      duration: _activeTransitionDuration,
                      switchInCurve: Curves.easeOut,
                      switchOutCurve: Curves.easeIn,
                      transitionBuilder: _buildTransition,
                      child: KeyedSubtree(
                        key: ValueKey<int>(_activeRootRevision),
                        child: _renderNode(_activeRoot!),
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildDiagnosticsPanel() {
    final keys = _diagnosticsData.keys.toList()..sort();
    final displayKeys = keys.take(16).toList();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Diagnostics: $_diagnosticsTitle'),
          if (displayKeys.isEmpty)
            const Text('No diagnostics data yet')
          else
            ...displayKeys.map(
              (key) => Text(
                '$key=${_diagnosticsData[key]}',
                style: const TextStyle(fontSize: 12),
              ),
            ),
        ],
      ),
    );
  }

  String _statusFromResponse(ConnectResponse response) {
    if (response.hasError()) {
      return 'Server error';
    }
    if (response.hasTransitionUi()) {
      return 'UI transition';
    }
    if (response.hasStartStream()) {
      return 'Stream started';
    }
    if (response.hasStopStream()) {
      return 'Stream stopped';
    }
    if (response.hasRouteStream()) {
      return 'Route updated';
    }
    if (response.hasWebrtcSignal()) {
      return 'WebRTC signal';
    }
    if (response.hasPlayAudio()) {
      return 'Play audio';
    }
    if (response.hasBugReportAck()) {
      return 'Bug report filed';
    }
    if (response.hasUpdateUi()) {
      return 'UI patched';
    }
    if (response.hasRegisterAck()) {
      return 'Registered';
    }
    if (response.hasCommandResult()) {
      return 'Command response';
    }
    if (response.hasSetUi()) {
      return 'UI updated';
    }
    return 'Connected';
  }

  void _applyMediaControlResponse(ConnectResponse response) {
    if (response.hasStartStream()) {
      final start = response.startStream;
      if (start.streamId.isNotEmpty) {
        _activeStreamsByID[start.streamId] = start.deepCopy();
        _outgoing.add(
          ConnectRequest()
            ..streamReady = (StreamReady()..streamId = start.streamId),
        );
        _streamReadyAckCount += 1;
        unawaited(_mediaEngine.startStream(start.deepCopy()));
      }
      if (start.kind.isNotEmpty) {
        _lastNotification = 'Start stream: ${start.kind} (${start.streamId})';
      }
    }
    if (response.hasStopStream()) {
      final streamID = response.stopStream.streamId;
      if (streamID.isNotEmpty) {
        _activeStreamsByID.remove(streamID);
        _routesByStreamID.remove(streamID);
        unawaited(_mediaEngine.stopStream(streamID));
        _lastNotification = 'Stop stream: $streamID';
      }
    }
    if (response.hasRouteStream()) {
      final route = response.routeStream;
      if (route.streamId.isNotEmpty) {
        _routesByStreamID[route.streamId] = route.deepCopy();
      }
      _lastNotification =
          'Route: ${route.sourceDeviceId} -> ${route.targetDeviceId} (${route.kind})';
    }
    if (response.hasWebrtcSignal()) {
      _recentWebRTCSignals.add(response.webrtcSignal.deepCopy());
      const maxSignals = 50;
      if (_recentWebRTCSignals.length > maxSignals) {
        _recentWebRTCSignals.removeRange(
          0,
          _recentWebRTCSignals.length - maxSignals,
        );
      }
      _lastNotification =
          'WebRTC signal: ${response.webrtcSignal.signalType} (${response.webrtcSignal.streamId})';
      unawaited(_mediaEngine.handleSignal(response.webrtcSignal.deepCopy()));
    }
    if (response.hasPlayAudio()) {
      final playAudio = response.playAudio;
      var source = 'unknown';
      var bytes = 0;
      switch (playAudio.whichSource()) {
        case iov1.PlayAudio_Source.pcmData:
          source = 'pcm_data';
          bytes = playAudio.pcmData.length;
          break;
        case iov1.PlayAudio_Source.url:
          source = 'url';
          bytes = 0;
          break;
        case iov1.PlayAudio_Source.ttsText:
          source = 'tts_text';
          bytes = 0;
          break;
        case iov1.PlayAudio_Source.notSet:
          source = 'not_set';
          bytes = 0;
          break;
      }
      _playAudioCount += 1;
      _lastPlayAudioBytes = bytes;
      _lastPlayAudioDeviceID =
          playAudio.deviceId.isNotEmpty ? playAudio.deviceId : 'unknown';
      _lastPlayAudioSource = source;
      _lastNotification =
          'Play audio: $_lastPlayAudioDeviceID ($source, $bytes bytes)';
    }
  }

  void _applyTransitionHint(uiv1.TransitionUI transitionUi) {
    final transition = transitionUi.transition.trim().toLowerCase();
    final hasTransition = transition.isNotEmpty && transition != 'none';
    final defaultDuration = hasTransition ? 250 : 0;
    final durationMs =
        transitionUi.durationMs > 0 ? transitionUi.durationMs : defaultDuration;
    _activeTransition = transition;
    _activeTransitionDuration = Duration(milliseconds: durationMs);
    _lastNotification =
        'Transition: ${transitionUi.transition} (${transitionUi.durationMs}ms)';
  }

  Widget _buildTransition(Widget child, Animation<double> animation) {
    switch (_activeTransition) {
      case 'fade':
        return FadeTransition(opacity: animation, child: child);
      case 'pa_source_enter':
        return ScaleTransition(scale: animation, child: child);
      case 'pa_source_exit':
        return FadeTransition(opacity: animation, child: child);
      case 'scale':
        return ScaleTransition(scale: animation, child: child);
      case 'slide':
      case 'slide_left':
      case 'slide-left':
      case 'pa_receive_enter':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0.15, 0),
        );
      case 'slide_right':
      case 'slide-right':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(-0.15, 0),
        );
      case 'slide_up':
      case 'slide-up':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, 0.15),
        );
      case 'slide_down':
      case 'slide-down':
      case 'pa_receive_exit':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, -0.15),
        );
      default:
        return child;
    }
  }

  Widget _buildSlideTransition({
    required Widget child,
    required Animation<double> animation,
    required Offset beginOffset,
  }) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: beginOffset,
        end: Offset.zero,
      ).animate(animation),
      child: child,
    );
  }

  uiv1.Node? _applyUpdateUi({
    required uiv1.Node? currentRoot,
    required uiv1.UpdateUI update,
  }) {
    if (!update.hasNode()) {
      return currentRoot;
    }
    final targetID = update.componentId.trim();
    final replacement = update.node.deepCopy();
    if (targetID.isEmpty) {
      return replacement;
    }
    if (currentRoot == null) {
      return null;
    }

    final root = currentRoot.deepCopy();
    if (_nodeId(root) == targetID) {
      return replacement;
    }
    final replaced = _replaceNodeByID(
      current: root,
      targetID: targetID,
      replacement: replacement,
    );
    if (!replaced) {
      return currentRoot;
    }
    return root;
  }

  bool _replaceNodeByID({
    required uiv1.Node current,
    required String targetID,
    required uiv1.Node replacement,
  }) {
    for (var i = 0; i < current.children.length; i++) {
      final child = current.children[i];
      if (_nodeId(child) == targetID) {
        current.children[i] = replacement.deepCopy();
        return true;
      }
      if (_replaceNodeByID(
        current: child,
        targetID: targetID,
        replacement: replacement,
      )) {
        return true;
      }
    }
    return false;
  }

  Future<void> _sendUiAction({
    required String componentId,
    required String action,
    required String value,
  }) async {
    if (_deviceId.isEmpty) {
      return;
    }
    _recordUiAction(
      componentId: componentId,
      action: action,
      value: value,
    );
    if (action.startsWith(_bugReportActionPrefix)) {
      await _submitBugReportFromAction(
        componentId: componentId,
        action: action,
        value: value,
      );
      return;
    }
    _outgoing.add(
      ConnectRequest()
        ..input = (iov1.InputEvent()
          ..deviceId = _deviceId
          ..uiAction = (iov1.UIAction()
            ..componentId = componentId
            ..action = action
            ..value = value)),
    );
  }

  Future<void> _sendKeyText(String text) async {
    if (_deviceId.isEmpty || text.isEmpty) {
      return;
    }
    _outgoing.add(
      ConnectRequest()
        ..input = (iov1.InputEvent()
          ..deviceId = _deviceId
          ..key = (iov1.KeyEvent()..text = text)),
    );
  }

  void _sendWebRTCSignalMessage(WebRTCSignal signal) {
    final streamID = signal.streamId.trim();
    final signalType = signal.signalType.trim();
    if (streamID.isEmpty || signalType.isEmpty) {
      return;
    }
    _outgoing.add(
      ConnectRequest()..webrtcSignal = signal.deepCopy(),
    );
  }

  String _nodeId(uiv1.Node node) {
    if (node.id.isNotEmpty) {
      return node.id;
    }
    return node.props['id'] ?? '';
  }

  Widget _renderNode(uiv1.Node node) {
    switch (node.whichWidget()) {
      case uiv1.Node_Widget.stack:
        return Container(
          color: _parseHexColor(node.props['background']),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: node.children.map(_renderNode).toList(),
          ),
        );
      case uiv1.Node_Widget.row:
        return Row(children: node.children.map(_renderNode).toList());
      case uiv1.Node_Widget.grid:
        final columns = node.grid.columns > 0 ? node.grid.columns : 1;
        return LayoutBuilder(
          builder: (context, constraints) {
            const spacing = 8.0;
            final maxWidth = constraints.maxWidth.isFinite
                ? constraints.maxWidth
                : MediaQuery.of(context).size.width;
            final totalSpacing = spacing * (columns - 1);
            final itemWidth =
                columns <= 1 ? maxWidth : (maxWidth - totalSpacing) / columns;
            return Wrap(
              spacing: spacing,
              runSpacing: spacing,
              children: node.children
                  .map(
                    (child) => SizedBox(
                      width: itemWidth,
                      child: _renderNode(child),
                    ),
                  )
                  .toList(),
            );
          },
        );
      case uiv1.Node_Widget.scroll:
        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: node.children.map(_renderNode).toList(),
          ),
        );
      case uiv1.Node_Widget.padding:
        return Padding(
          padding: EdgeInsets.all(node.padding.all.toDouble()),
          child: _renderNodeChildren(node.children),
        );
      case uiv1.Node_Widget.center:
        return Center(child: _renderNodeChildren(node.children));
      case uiv1.Node_Widget.expand:
        return Expanded(child: _renderNodeChildren(node.children));
      case uiv1.Node_Widget.text:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: SelectableText(
            node.text.value,
            style: TextStyle(
              color: _parseHexColor(node.text.color),
              fontFamily: node.text.style == 'monospace' ? 'monospace' : null,
            ),
          ),
        );
      case uiv1.Node_Widget.textInput:
        final componentId = _nodeId(node);
        final isTerminalInput = componentId == 'terminal_input';
        return TextField(
          controller: _terminalInputController,
          decoration: InputDecoration(
            hintText: node.textInput.placeholder,
          ),
          autofocus: node.textInput.autofocus,
          onChanged: (value) {
            if (!isTerminalInput) {
              return;
            }
            final previous = _terminalInputShadow;
            if (value.startsWith(previous) && value.length > previous.length) {
              final inserted = value.substring(previous.length);
              if (inserted.isNotEmpty) {
                unawaited(_sendKeyText(inserted));
              }
            } else if (previous.startsWith(value) &&
                previous.length > value.length) {
              final removed = previous.length - value.length;
              if (removed > 0) {
                unawaited(
                    _sendKeyText(List<String>.filled(removed, '\b').join()));
              }
            }
            _terminalInputShadow = value;
          },
          onSubmitted: (value) async {
            if (isTerminalInput) {
              await _sendKeyText('\n');
            } else {
              await _sendUiAction(
                componentId:
                    componentId.isNotEmpty ? componentId : 'text_input',
                action: 'submit',
                value: value,
              );
            }
            _terminalInputController.clear();
            _terminalInputShadow = '';
          },
        );
      case uiv1.Node_Widget.button:
        final componentId = _nodeId(node);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: ElevatedButton(
            onPressed: () {
              unawaited(
                _sendUiAction(
                  componentId: componentId.isNotEmpty ? componentId : 'button',
                  action: node.button.action.isNotEmpty
                      ? node.button.action
                      : 'tap',
                  value: '',
                ),
              );
            },
            child: Text(node.button.label),
          ),
        );
      case uiv1.Node_Widget.slider:
        final componentId = _nodeId(node);
        final min = node.slider.min;
        final max = node.slider.max > min ? node.slider.max : min + 1;
        final value = node.slider.value.clamp(min, max).toDouble();
        return Slider(
          value: value,
          min: min,
          max: max,
          onChanged: (nextValue) {
            unawaited(
              _sendUiAction(
                componentId: componentId.isNotEmpty ? componentId : 'slider',
                action: 'change',
                value: nextValue.toString(),
              ),
            );
          },
        );
      case uiv1.Node_Widget.toggle:
        final componentId = _nodeId(node);
        return SwitchListTile(
          value: node.toggle.value,
          onChanged: (nextValue) {
            unawaited(
              _sendUiAction(
                componentId: componentId.isNotEmpty ? componentId : 'toggle',
                action: 'toggle',
                value: nextValue.toString(),
              ),
            );
          },
        );
      case uiv1.Node_Widget.dropdown:
        final componentId = _nodeId(node);
        final options = node.dropdown.options;
        final selected = options.contains(node.dropdown.value)
            ? node.dropdown.value
            : (options.isNotEmpty ? options.first : null);
        return DropdownButton<String>(
          isExpanded: true,
          value: selected,
          hint: const Text('Select option'),
          items: options
              .map(
                (option) => DropdownMenuItem<String>(
                  value: option,
                  child: Text(option),
                ),
              )
              .toList(),
          onChanged: options.isEmpty
              ? null
              : (nextValue) {
                  if (nextValue == null) {
                    return;
                  }
                  unawaited(
                    _sendUiAction(
                      componentId:
                          componentId.isNotEmpty ? componentId : 'dropdown',
                      action: 'select',
                      value: nextValue,
                    ),
                  );
                },
        );
      case uiv1.Node_Widget.gestureArea:
        final componentId = _nodeId(node);
        final action = node.gestureArea.action.isNotEmpty
            ? node.gestureArea.action
            : 'tap';
        final child = _renderNodeChildren(node.children);
        return GestureDetector(
          key: ValueKey<String>('ui-gesture-$componentId'),
          behavior: HitTestBehavior.opaque,
          onTap: () {
            unawaited(
              _sendUiAction(
                componentId:
                    componentId.isNotEmpty ? componentId : 'gesture_area',
                action: action,
                value: '',
              ),
            );
          },
          child: node.children.isEmpty
              ? const SizedBox(width: 48, height: 48)
              : child,
        );
      case uiv1.Node_Widget.overlay:
        final componentId = _nodeId(node);
        return Stack(
          key: ValueKey<String>('ui-overlay-$componentId'),
          fit: StackFit.loose,
          children: node.children.map(_renderNode).toList(),
        );
      case uiv1.Node_Widget.videoSurface:
        final componentId = _nodeId(node);
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-video-surface-$componentId'),
          title: 'Video surface',
          detail: node.videoSurface.trackId,
          child: const SizedBox(
            height: 120,
            child: Center(
              child: Icon(Icons.videocam_outlined),
            ),
          ),
        );
      case uiv1.Node_Widget.audioVisualizer:
        final componentId = _nodeId(node);
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-audio-visualizer-$componentId'),
          title: 'Audio visualizer',
          detail: node.audioVisualizer.streamId,
          child: const Padding(
            padding: EdgeInsets.only(top: 8),
            child: LinearProgressIndicator(),
          ),
        );
      case uiv1.Node_Widget.canvas:
        final componentId = _nodeId(node);
        final drawOps = node.canvas.drawOpsJson.trim();
        final drawOpsPreview = drawOps.isEmpty
            ? 'No draw ops'
            : (drawOps.length > 64 ? '${drawOps.substring(0, 64)}…' : drawOps);
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-canvas-$componentId'),
          title: 'Canvas',
          detail: drawOpsPreview,
        );
      case uiv1.Node_Widget.fullscreen:
        final componentId = _nodeId(node);
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-fullscreen-$componentId'),
          title:
              'Fullscreen ${node.fullscreen.enabled ? 'enabled' : 'disabled'}',
          child: _renderNodeChildren(node.children),
        );
      case uiv1.Node_Widget.keepAwake:
        final componentId = _nodeId(node);
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-keep-awake-$componentId'),
          title:
              'Keep awake ${node.keepAwake.enabled ? 'enabled' : 'disabled'}',
          child: _renderNodeChildren(node.children),
        );
      case uiv1.Node_Widget.brightness:
        final componentId = _nodeId(node);
        final brightness = node.brightness.value.clamp(0.0, 1.0).toDouble();
        return _placeholderPrimitive(
          key: ValueKey<String>('ui-brightness-$componentId'),
          title: 'Brightness hint',
          detail: brightness.toStringAsFixed(2),
          child: _renderNodeChildren(node.children),
        );
      case uiv1.Node_Widget.image:
        return Image.network(
          node.image.url,
          fit: BoxFit.cover,
          errorBuilder: (context, error, stackTrace) {
            return const Icon(Icons.broken_image_outlined);
          },
        );
      case uiv1.Node_Widget.progress:
        return LinearProgressIndicator(
          value: node.progress.value.clamp(0.0, 1.0).toDouble(),
        );
      case uiv1.Node_Widget.notSet:
        break;
    }
    return const SizedBox.shrink();
  }

  Widget _renderNodeChildren(List<uiv1.Node> children) {
    if (children.isEmpty) {
      return const SizedBox.shrink();
    }
    if (children.length == 1) {
      return _renderNode(children.first);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: children.map(_renderNode).toList(),
    );
  }

  Widget _placeholderPrimitive({
    required Key key,
    required String title,
    String? detail,
    Widget? child,
  }) {
    return Container(
      key: key,
      margin: const EdgeInsets.symmetric(vertical: 6),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title),
          if (detail != null && detail.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(detail, style: const TextStyle(fontSize: 12)),
          ],
          if (child != null) ...[
            const SizedBox(height: 6),
            child,
          ],
        ],
      ),
    );
  }

  Color? _parseHexColor(String? raw) {
    if (raw == null || raw.isEmpty) {
      return null;
    }
    var value = raw.trim();
    if (value.startsWith('#')) {
      value = value.substring(1);
    }
    if (value.length == 6) {
      value = 'FF$value';
    }
    if (value.length != 8) {
      return null;
    }
    final parsed = int.tryParse(value, radix: 16);
    if (parsed == null) {
      return null;
    }
    return Color(parsed);
  }
}

class _BugIdentifier {
  const _BugIdentifier({
    required this.word,
    required this.code,
    required this.qrPayload,
  });

  final String word;
  final String code;
  final String qrPayload;
}

class _BugReportDraft {
  const _BugReportDraft({
    required this.description,
    required this.tags,
    required this.identifier,
  });

  final String description;
  final List<String> tags;
  final _BugIdentifier identifier;
}

class _PendingBugReport {
  const _PendingBugReport({
    required this.identifier,
    required this.submittedUnixMs,
  });

  final _BugIdentifier identifier;
  final int submittedUnixMs;
}

class _QueuedBugReport {
  const _QueuedBugReport({
    required this.bugReport,
    required this.identifier,
  });

  final diagv1.BugReport bugReport;
  final _BugIdentifier identifier;
}
