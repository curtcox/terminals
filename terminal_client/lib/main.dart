import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:terminal_client/capabilities/probe.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/discovery/mdns_scanner.dart';
import 'package:terminal_client/edge/artifact_export.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/media/playback.dart';
import 'package:terminal_client/media/webrtc_engine.dart';
import 'package:terminal_client/util/browser_host.dart' as browser_host;
import 'package:terminal_client/util/speech.dart' as speech;

typedef TerminalControlClientFactory = TerminalControlClient Function({
  required String host,
  required int port,
});
typedef CapabilityProbeFactory = CapabilityProbe Function();
typedef ClientMediaEngineFactory = ClientMediaEngine Function({
  required String localDeviceID,
  required OutboundSignalCallback onSignal,
});
typedef AudioPlaybackFactory = AudioPlayback Function();
typedef UnixMsProvider = int Function();
typedef BugReportScreenshotCapture = Future<List<int>> Function();

int _systemNowUnixMs() => DateTime.now().toUtc().millisecondsSinceEpoch;

CapabilityProbe _defaultCapabilityProbeFactory() {
  final bindingType = WidgetsBinding.instance.runtimeType.toString();
  if (bindingType.contains('TestWidgetsFlutterBinding')) {
    return DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[],
    );
  }
  return DefaultCapabilityProbe();
}

AudioPlayback _defaultAudioPlaybackFactory() {
  final bindingType = WidgetsBinding.instance.runtimeType.toString();
  if (bindingType.contains('TestWidgetsFlutterBinding')) {
    return NoopAudioPlayback();
  }
  return AudioPlayerPlayback();
}

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
const String _defaultControlHost = String.fromEnvironment(
  'TERMINALS_CONTROL_HOST',
  defaultValue: '127.0.0.1',
);
const int _defaultGrpcPort = int.fromEnvironment(
  'TERMINALS_GRPC_PORT',
  defaultValue: 50051,
);
const int _defaultControlWSPort = int.fromEnvironment(
  'TERMINALS_CONTROL_WS_PORT',
  defaultValue: 50054,
);
const int _defaultControlPort =
    kIsWeb ? _defaultControlWSPort : _defaultGrpcPort;
const String _buildSha = String.fromEnvironment(
  'TERMINALS_BUILD_SHA',
  defaultValue: 'unknown',
);
const String _buildDate = String.fromEnvironment(
  'TERMINALS_BUILD_DATE',
  defaultValue: 'unknown',
);
const String _registerMetadataPhotoFrameBaseURLKey =
    'photo_frame_asset_base_url';
const String _registerMetadataServerBuildShaKey = 'server_build_sha';
const String _registerMetadataServerBuildDateKey = 'server_build_date';
const String _bugReportActionPrefix = 'bug_report';
const int _clientContextRecentUiCap = 32;
const int _clientContextRecentLogCap = 200;
const int _clientContextRecentErrorCap = 32;
const Duration _bugReportAckTimeout = Duration(seconds: 20);
const Duration _capabilityMonitorInterval = Duration(seconds: 2);
const Duration _registerAckRetryInterval = Duration(milliseconds: 400);
const Duration _registerAckTimeout = Duration(seconds: 20);

const Set<String> _loopbackHosts = <String>{
  'localhost',
  '127.0.0.1',
  '::1',
};

String resolveInitialControlHost({
  required bool isWebRuntime,
  required String configuredHost,
  required String pageHost,
}) {
  final trimmedConfiguredHost = configuredHost.trim();
  if (!isWebRuntime) {
    return trimmedConfiguredHost;
  }
  final trimmedPageHost = pageHost.trim();
  if (trimmedPageHost.isEmpty) {
    return trimmedConfiguredHost;
  }
  if (trimmedConfiguredHost.isEmpty) {
    return trimmedPageHost;
  }
  if (_loopbackHosts.contains(trimmedConfiguredHost.toLowerCase())) {
    return trimmedPageHost;
  }
  return trimmedConfiguredHost;
}

String resolvePageHost({
  required String browserLocationHost,
  required String uriBaseHost,
}) {
  final fromLocation = browserLocationHost.trim();
  if (fromLocation.isNotEmpty) {
    return fromLocation;
  }
  return uriBaseHost.trim();
}

String buildMetadataLabel(
    {required String buildDate, required String buildSha}) {
  final normalizedBuildDate = normalizeBuildValue(buildDate);
  final normalizedBuildSha = normalizeBuildValue(buildSha);
  return 'Build: $normalizedBuildDate | SHA: $normalizedBuildSha';
}

String normalizeBuildValue(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty) {
    return 'unknown';
  }
  return trimmed;
}

String buildVersionParityNote({
  required String clientBuildDate,
  required String clientBuildSha,
  required String serverBuildDate,
  required String serverBuildSha,
}) {
  final normalizedClientSha = normalizeBuildValue(clientBuildSha);
  final normalizedServerSha = normalizeBuildValue(serverBuildSha);
  final normalizedClientDate = normalizeBuildValue(clientBuildDate);
  final normalizedServerDate = normalizeBuildValue(serverBuildDate);

  if (normalizedServerSha == 'unknown' && normalizedServerDate == 'unknown') {
    return 'Build Match: unknown (awaiting server register ack)';
  }
  if (normalizedClientSha != 'unknown' && normalizedServerSha != 'unknown') {
    if (normalizedClientSha == normalizedServerSha) {
      if (normalizedClientDate != 'unknown' &&
          normalizedServerDate != 'unknown' &&
          normalizedClientDate != normalizedServerDate) {
        return 'Build Match: same SHA, different build date';
      }
      return 'Build Match: same SHA';
    }
    return 'Build Match: different SHA';
  }
  if (normalizedClientDate != 'unknown' && normalizedServerDate != 'unknown') {
    if (normalizedClientDate == normalizedServerDate) {
      return 'Build Match: same build date';
    }
    return 'Build Match: different build date';
  }
  return 'Build Match: unknown';
}

String buildServerBuildLine({
  required String serverBuildDate,
  required String serverBuildSha,
  required bool hasRegisterAck,
}) {
  final normalizedServerSha = normalizeBuildValue(serverBuildSha);
  final normalizedServerDate = normalizeBuildValue(serverBuildDate);
  if (!hasRegisterAck &&
      normalizedServerSha == 'unknown' &&
      normalizedServerDate == 'unknown') {
    return 'Server Build: awaiting register ack';
  }
  return 'Server ${buildMetadataLabel(
    buildDate: serverBuildDate,
    buildSha: serverBuildSha,
  )}';
}

String buildWebConnectionChipLabel({
  required bool hasRegisterAck,
  required bool isConnecting,
  required bool shouldStayConnected,
}) {
  if (hasRegisterAck) {
    return 'Connected';
  }
  if (isConnecting || shouldStayConnected) {
    return 'Connecting';
  }
  return 'Not connected';
}

String buildTransportDiagnosticsClipboardText({
  required String lastTransportDiagnostic,
  required List<String> recentAttempts,
}) {
  final lines = <String>['Transport Diagnostics'];
  final normalizedDiagnostic = lastTransportDiagnostic.trim();
  if (normalizedDiagnostic.isEmpty) {
    lines.add('No transport failures captured yet');
  } else {
    lines.add(normalizedDiagnostic);
  }
  if (recentAttempts.isNotEmpty) {
    lines.add('Recent Carrier Attempts');
    lines.addAll(
      recentAttempts
          .map((attempt) => attempt.trim())
          .where((a) => a.isNotEmpty),
    );
  }
  return lines.join('\n');
}

String buildControlStreamClipboardText({
  required String status,
  required String notification,
  required String transportDiagnostics,
}) {
  final lines = <String>['Control Stream: ${status.trim()}'];
  final normalizedNotification = notification.trim();
  if (normalizedNotification.isNotEmpty) {
    lines.add(normalizedNotification);
  }
  final normalizedDiagnostics = transportDiagnostics.trim();
  if (normalizedDiagnostics.isNotEmpty) {
    lines.add('Transport Diagnostics');
    lines.add(normalizedDiagnostics);
  }
  return lines.join('\n');
}

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

ControlCarrierKind? carrierKindFromPriorityName(String raw) {
  switch (raw.trim().toLowerCase()) {
    case 'grpc':
      return ControlCarrierKind.grpc;
    case 'websocket':
    case 'ws':
      return ControlCarrierKind.websocket;
    case 'tcp':
      return ControlCarrierKind.tcp;
    case 'http':
      return ControlCarrierKind.http;
    default:
      return null;
  }
}

bool isCarrierSupportedOnRuntime(
  ControlCarrierKind carrier, {
  required bool isWebRuntime,
}) {
  if (isWebRuntime) {
    return carrier == ControlCarrierKind.websocket;
  }
  return true;
}

List<ControlCarrierKind> buildCarrierPreference({
  required bool isWebRuntime,
  required List<String> serverPriority,
  ControlCarrierKind? lastSuccessfulCarrier,
}) {
  final defaults = isWebRuntime
      ? const <ControlCarrierKind>[ControlCarrierKind.websocket]
      : const <ControlCarrierKind>[
          ControlCarrierKind.grpc,
          ControlCarrierKind.websocket,
          ControlCarrierKind.tcp,
          ControlCarrierKind.http,
        ];
  if (serverPriority.isEmpty) {
    return defaults;
  }

  final ordered = <ControlCarrierKind>[];
  for (final raw in serverPriority) {
    final carrier = carrierKindFromPriorityName(raw);
    if (carrier == null) {
      continue;
    }
    if (isWebRuntime && carrier != ControlCarrierKind.websocket) {
      continue;
    }
    if (!ordered.contains(carrier)) {
      ordered.add(carrier);
    }
  }

  final preferred = ordered.isEmpty ? defaults : ordered;
  final filtered = preferred
      .where(
        (carrier) =>
            isCarrierSupportedOnRuntime(carrier, isWebRuntime: isWebRuntime),
      )
      .toSet()
      .toList(growable: true);
  if (filtered.isEmpty) {
    return defaults
        .where(
          (carrier) =>
              isCarrierSupportedOnRuntime(carrier, isWebRuntime: isWebRuntime),
        )
        .toList(growable: false);
  }
  if (lastSuccessfulCarrier != null &&
      filtered.contains(lastSuccessfulCarrier)) {
    filtered
      ..remove(lastSuccessfulCarrier)
      ..insert(0, lastSuccessfulCarrier);
  }
  return filtered;
}

String classifyCarrierFailure({
  required String stage,
  required String rawError,
}) {
  final lower = rawError.toLowerCase();
  final trimmedStage = stage.trim().toLowerCase();

  if (lower.contains('unsupported_protocol_version') ||
      lower.contains('unsupported protocol version') ||
      lower.contains('transport hello rejected protocol version')) {
    return 'protocol_version';
  }
  if (lower.contains('unsupported_carrier') ||
      lower.contains('not declared in transport hello')) {
    return 'carrier_mismatch';
  }
  if (lower.contains('failed host lookup') ||
      lower.contains('name or service not known') ||
      lower.contains('nodename nor servname provided')) {
    return 'dns';
  }
  if (lower.contains('connection refused')) {
    return 'tcp_connect';
  }
  if (lower.contains('certificate') ||
      lower.contains('tls') ||
      lower.contains('ssl') ||
      lower.contains('handshake')) {
    return 'tls_or_handshake';
  }
  if (lower.contains('upgrade rejected') || lower.contains('http 403')) {
    return 'upgrade_rejected';
  }
  if (lower.contains('timed out') || lower.contains('timeout')) {
    return 'timeout';
  }
  if (trimmedStage == 'stream_closed' || lower.contains('stream closed')) {
    return 'stream_closed';
  }
  if (trimmedStage == 'connect') {
    return 'connect_error';
  }
  if (trimmedStage == 'stream') {
    return 'stream_error';
  }
  return 'unknown';
}

class TransportErrorDiagnosis {
  const TransportErrorDiagnosis({
    required this.summary,
    required this.guidance,
    required this.grpcCode,
    required this.grpcCodeName,
    required this.rawError,
  });

  final String summary;
  final String guidance;
  final int? grpcCode;
  final String grpcCodeName;
  final String rawError;

  bool get hasSummary => summary.isNotEmpty;

  String statusText() {
    if (hasSummary) {
      return 'Stream error: $summary';
    }
    return 'Stream error: $rawError';
  }

  String notificationText() {
    if (!hasSummary) {
      return '';
    }
    if (guidance.isEmpty) {
      return summary;
    }
    return '$summary $guidance';
  }
}

class _CarrierAttemptDiagnostic {
  const _CarrierAttemptDiagnostic({
    required this.carrier,
    required this.endpoint,
    required this.stage,
    required this.failureClass,
    required this.error,
    required this.elapsed,
  });

  final ControlCarrierKind carrier;
  final String endpoint;
  final String stage;
  final String failureClass;
  final String error;
  final Duration elapsed;

  String get carrierLabel {
    switch (carrier) {
      case ControlCarrierKind.grpc:
        return 'gRPC';
      case ControlCarrierKind.websocket:
        return 'WebSocket';
      case ControlCarrierKind.tcp:
        return 'TCP';
      case ControlCarrierKind.http:
        return 'HTTP';
    }
  }
}

String _formatCarrierAttempt(_CarrierAttemptDiagnostic attempt) {
  final elapsedMs = attempt.elapsed.inMilliseconds;
  return '${attempt.carrierLabel} failed at ${attempt.stage} '
      '[${attempt.failureClass}] (${attempt.endpoint}) '
      'after ${elapsedMs}ms: ${attempt.error}';
}

class _ConnectionTarget {
  const _ConnectionTarget({required this.host, required this.port});

  final String host;
  final int port;
}

TransportErrorDiagnosis diagnoseTransportError(
  Object error, {
  required bool isWeb,
}) {
  final raw = error.toString();
  final lower = raw.toLowerCase();
  final grpcCodeMatch =
      RegExp(r'code:\s*([0-9]+)', caseSensitive: false).firstMatch(raw);
  final grpcCode =
      grpcCodeMatch == null ? null : int.tryParse(grpcCodeMatch.group(1) ?? '');
  final grpcCodeNameMatch =
      RegExp(r'codeName:\s*([A-Z_]+)', caseSensitive: false).firstMatch(raw);
  final grpcCodeName = (grpcCodeNameMatch?.group(1) ?? '').trim().toUpperCase();
  final isGrpcError = lower.contains('grpc error');
  final isUnavailable = grpcCode == 14 ||
      grpcCodeName == 'UNAVAILABLE' ||
      lower.contains('unavailable');
  final hasSocketConstructorFailure =
      lower.contains('unsupported operation: socket constructor');

  if (isGrpcError && isUnavailable && hasSocketConstructorFailure && isWeb) {
    return TransportErrorDiagnosis(
      summary: 'gRPC UNAVAILABLE (14)',
      guidance:
          'Browser runtime cannot open raw gRPC sockets. Configure gRPC-Web via an HTTP proxy (for example Envoy) or use a non-web client target.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  if (isGrpcError && isUnavailable) {
    return TransportErrorDiagnosis(
      summary: 'gRPC UNAVAILABLE (14)',
      guidance:
          'Server is unreachable or transport is unavailable. Verify host/port, server process, and network/proxy configuration.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  if (isGrpcError && grpcCode != null) {
    final displayName = grpcCodeName.isEmpty ? '' : ' ($grpcCodeName)';
    return TransportErrorDiagnosis(
      summary: 'gRPC error $grpcCode$displayName',
      guidance: 'Check server logs and client/server protocol compatibility.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  return TransportErrorDiagnosis(
    summary: '',
    guidance: '',
    grpcCode: grpcCode,
    grpcCodeName: grpcCodeName,
    rawError: raw,
  );
}

void main() {
  runApp(const TerminalClientApp());
}

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({
    super.key,
    this.clientFactory = createTerminalControlClient,
    this.capabilityProbeFactory = _defaultCapabilityProbeFactory,
    this.mediaEngineFactory = defaultClientMediaEngineFactory,
    this.audioPlaybackFactory = _defaultAudioPlaybackFactory,
    this.heartbeatInterval = const Duration(seconds: 10),
    this.sensorTelemetryInterval = const Duration(seconds: 15),
    this.reconnectDelayBase = const Duration(seconds: 2),
    this.reconnectDelayMaxSeconds = 30,
    this.nowUnixMsProvider = _systemNowUnixMs,
    this.autoConnectOnStartup = kIsWeb,
    this.bugReportScreenshotCapture,
  });

  final TerminalControlClientFactory clientFactory;
  final CapabilityProbeFactory capabilityProbeFactory;
  final ClientMediaEngineFactory mediaEngineFactory;
  final AudioPlaybackFactory audioPlaybackFactory;
  final Duration heartbeatInterval;
  final Duration sensorTelemetryInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;
  final UnixMsProvider nowUnixMsProvider;
  final bool autoConnectOnStartup;
  final BugReportScreenshotCapture? bugReportScreenshotCapture;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: _ControlStreamScaffold(
        clientFactory: clientFactory,
        capabilityProbeFactory: capabilityProbeFactory,
        mediaEngineFactory: mediaEngineFactory,
        audioPlaybackFactory: audioPlaybackFactory,
        heartbeatInterval: heartbeatInterval,
        sensorTelemetryInterval: sensorTelemetryInterval,
        reconnectDelayBase: reconnectDelayBase,
        reconnectDelayMaxSeconds: reconnectDelayMaxSeconds,
        nowUnixMsProvider: nowUnixMsProvider,
        autoConnectOnStartup: autoConnectOnStartup,
        bugReportScreenshotCapture: bugReportScreenshotCapture,
      ),
    );
  }
}

class _ControlStreamScaffold extends StatefulWidget {
  const _ControlStreamScaffold({
    required this.clientFactory,
    required this.capabilityProbeFactory,
    required this.mediaEngineFactory,
    required this.audioPlaybackFactory,
    required this.heartbeatInterval,
    required this.sensorTelemetryInterval,
    required this.reconnectDelayBase,
    required this.reconnectDelayMaxSeconds,
    required this.nowUnixMsProvider,
    required this.autoConnectOnStartup,
    required this.bugReportScreenshotCapture,
  });

  final TerminalControlClientFactory clientFactory;
  final CapabilityProbeFactory capabilityProbeFactory;
  final ClientMediaEngineFactory mediaEngineFactory;
  final AudioPlaybackFactory audioPlaybackFactory;
  final Duration heartbeatInterval;
  final Duration sensorTelemetryInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;
  final UnixMsProvider nowUnixMsProvider;
  final bool autoConnectOnStartup;
  final BugReportScreenshotCapture? bugReportScreenshotCapture;

  @override
  State<_ControlStreamScaffold> createState() => _ControlStreamScaffoldState();
}

class _ControlStreamScaffoldState extends State<_ControlStreamScaffold>
    with WidgetsBindingObserver {
  final GlobalKey _bugReportScreenshotKey = GlobalKey();
  final TextEditingController _hostController = TextEditingController(
    text: _defaultControlHost,
  );
  final TextEditingController _portController = TextEditingController(
    text: _defaultControlPort.toString(),
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
  late final CapabilityProbe _capabilityProbe;
  late final ClientMediaEngine _mediaEngine;
  late final AudioPlayback _audioPlayback;
  late final DurableArtifactExporter _artifactExporter;
  uiv1.Node? _activeRoot;
  int _activeRootRevision = 0;
  String _activeTransition = 'none';
  Duration _activeTransitionDuration = Duration.zero;
  String _lastNotification = '';
  final TextEditingController _terminalInputController =
      TextEditingController();
  final FocusNode _terminalInputFocusNode = FocusNode(
    debugLabel: 'terminal_input',
  );
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
  Timer? _capabilityMonitorTimer;
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
  String _pendingScenarioRegistryRequestID = '';
  String _pendingPlaybackArtifactsRequestID = '';
  String _pendingPlaybackMetadataRequestID = '';
  List<String> _availableApplicationIntents = <String>['terminal'];
  String _selectedApplicationIntent = 'terminal';
  String _pendingLaunchApplicationIntent = '';
  String _diagnosticsTitle = 'none';
  Map<String, String> _diagnosticsData = <String, String>{};
  String _photoFrameAssetBaseURL = '';
  String _serverBuildSha = 'unknown';
  String _serverBuildDate = 'unknown';
  final List<diagv1.UiEventEntry> _recentUiEvents = <diagv1.UiEventEntry>[];
  final List<diagv1.UiActionEntry> _recentUiActions = <diagv1.UiActionEntry>[];
  final List<diagv1.LogEntry> _recentLogs = <diagv1.LogEntry>[];
  final List<diagv1.ControlErrorEntry> _recentControlErrors =
      <diagv1.ControlErrorEntry>[];
  final Map<String, double> _lastSensorSnapshot = <String, double>{};
  capv1.DeviceCapabilities? _lastRegisteredCapabilities;
  int _capabilityGeneration = 0;
  int _lastCapabilityAckGeneration = 0;
  int _lastHeartbeatUnixMs = 0;
  int _pendingHeartbeatUnixMs = 0;
  double _lastRttMs = 0;
  String _lastConnectionStatus = 'Idle';
  String _lastFlutterErrorMessage = '';
  String _lastFlutterErrorStack = '';
  int _lastFlutterErrorUnixMs = 0;
  String _lastTransportDiagnostic = '';
  final List<_CarrierAttemptDiagnostic> _carrierAttemptLog =
      <_CarrierAttemptDiagnostic>[];
  List<ControlCarrierKind> _activeCarrierCycle = <ControlCarrierKind>[];
  int _activeCarrierIndex = 0;
  ControlCarrierKind? _lastSuccessfulCarrier;
  String _lastBugTokenWord = '';
  String _lastBugTokenCode = '';
  _BugReceiptState _bugReceiptState = _BugReceiptState.none;
  String _bugReceiptDetail = '';
  String _bugReceiptReportId = '';
  final List<_QueuedBugReport> _queuedBugReports = <_QueuedBugReport>[];
  final List<_PendingBugReport> _pendingBugReports = <_PendingBugReport>[];
  Timer? _bugReportAckTimer;
  Timer? _registerAckRetryTimer;
  int _registerAckRetryAttempts = 0;
  int _registerAckRetryStartedUnixMs = 0;
  bool _hasRegisterAck = false;
  FlutterExceptionHandler? _previousFlutterErrorHandler;
  bool _appIsForeground = true;
  Size _lastKnownLogicalSize = Size.zero;
  double _lastKnownDevicePixelRatio = 1.0;
  String _lastKnownOrientation = 'unknown';
  TextDirection _lastKnownTextDirection = TextDirection.ltr;
  bool _capabilityPollInFlight = false;
  String _lastCapabilitySignature = '';

  int _nowUnixMs() => widget.nowUnixMsProvider();

  Size _currentLogicalSize() {
    if (_lastKnownLogicalSize.width > 0 && _lastKnownLogicalSize.height > 0) {
      return _lastKnownLogicalSize;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return Size.zero;
    }
    final view = views.first;
    final dpr = view.devicePixelRatio <= 0 ? 1.0 : view.devicePixelRatio;
    return Size(
      view.physicalSize.width / dpr,
      view.physicalSize.height / dpr,
    );
  }

  double _currentDevicePixelRatio() {
    if (_lastKnownDevicePixelRatio > 0) {
      return _lastKnownDevicePixelRatio;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return 1.0;
    }
    final dpr = views.first.devicePixelRatio;
    return dpr <= 0 ? 1.0 : dpr;
  }

  bool get _hasActiveControlSession =>
      _shouldStayConnected && _incoming != null && _client != null;

  String _capabilitySignature(capv1.DeviceCapabilities capabilities) {
    return capabilities.writeToBuffer().join(',');
  }

  String _normalizedOrientationFromSize(Size size) {
    if (size.width <= 0 || size.height <= 0) {
      return _lastKnownOrientation;
    }
    return size.width >= size.height ? 'landscape' : 'portrait';
  }

  capv1.DeviceCapabilities _applyDisplayMetadata(
    capv1.DeviceCapabilities capabilities,
  ) {
    final size = _currentLogicalSize();
    final dpr = _currentDevicePixelRatio();
    final orientation = _normalizedOrientationFromSize(size);
    final screen = capabilities.hasScreen()
        ? capabilities.screen
        : (capabilities.screen = capv1.ScreenCapability());
    screen
      ..width = size.width.round()
      ..height = size.height.round()
      ..density = dpr
      ..orientation = orientation
      ..fullscreenSupported = true
      ..multiWindowSupported = true
      ..safeArea = (screen.hasSafeArea() ? screen.safeArea : capv1.Insets())
      ..safeArea.left = 0
      ..safeArea.top = 0
      ..safeArea.right = 0
      ..safeArea.bottom = 0;

    if (capabilities.displays.isEmpty) {
      capabilities.displays.add(capv1.DisplayCapability()..displayId = 'main');
    }
    final display = capabilities.displays.first;
    display
      ..displayId = display.displayId.isEmpty ? 'main' : display.displayId
      ..displayName =
          display.displayName.isEmpty ? 'Primary Display' : display.displayName
      ..primary = true
      ..screen = screen.deepCopy();
    return capabilities;
  }

  capv1.DeviceCapabilities _applyLifecycleOperator(
    capv1.DeviceCapabilities capabilities,
  ) {
    final edge = capabilities.hasEdge()
        ? capabilities.edge
        : (capabilities.edge = capv1.EdgeCapability());
    final lifecycleOperator = _appIsForeground
        ? 'monitor.lifecycle.foreground'
        : 'monitor.lifecycle.background';
    final nextOperators = edge.operators
        .where(
          (operator) =>
              operator != 'monitor.lifecycle.foreground' &&
              operator != 'monitor.lifecycle.background',
        )
        .toList(growable: true)
      ..add(lifecycleOperator);
    edge.operators
      ..clear()
      ..addAll(_dedupeOperators(nextOperators));
    return capabilities;
  }

  Future<void> _probeAndPublishCapabilityChanges({
    required String reason,
    bool forceSnapshot = false,
  }) async {
    if (!_hasActiveControlSession ||
        _deviceId.isEmpty ||
        _capabilityPollInFlight) {
      return;
    }
    _capabilityPollInFlight = true;
    try {
      final touchInputLikely = switch (defaultTargetPlatform) {
        TargetPlatform.android => true,
        TargetPlatform.iOS => true,
        TargetPlatform.fuchsia => true,
        TargetPlatform.macOS => false,
        TargetPlatform.linux => false,
        TargetPlatform.windows => false,
      };
      final probedCapabilities = await _capabilityProbe.probe(
        CapabilityProbeContext(
          deviceId: _deviceId,
          deviceName: _deviceNameController.text.trim(),
          deviceType: _deviceTypeController.text.trim(),
          platform: _platformController.text.trim(),
          screenWidth: _currentLogicalSize().width.round(),
          screenHeight: _currentLogicalSize().height.round(),
          screenDensity: _currentDevicePixelRatio(),
          touchInputLikely: touchInputLikely,
          targetPlatform: defaultTargetPlatform,
        ),
      );
      final nextCapabilities = _applyLifecycleOperator(
        _applyDisplayMetadata(probedCapabilities.deepCopy()),
      );
      final nextSignature = _capabilitySignature(nextCapabilities);
      final changed = nextSignature != _lastCapabilitySignature;
      if (!changed && !forceSnapshot) {
        return;
      }

      _lastRegisteredCapabilities = nextCapabilities;
      _lastCapabilitySignature = nextSignature;
      final nextGeneration = math.max(
        _capabilityGeneration + 1,
        _lastCapabilityAckGeneration + 1,
      );
      _capabilityGeneration = nextGeneration;
      if (forceSnapshot) {
        _outgoing.add(
          TerminalControlGrpcClient.capabilitySnapshotRequest(
            deviceId: _deviceId,
            generation: _capabilityGeneration,
            capabilities: nextCapabilities,
          ),
        );
        return;
      }
      _outgoing.add(
        TerminalControlGrpcClient.capabilityDeltaRequest(
          deviceId: _deviceId,
          generation: _capabilityGeneration,
          capabilities: nextCapabilities,
          reason: reason,
        ),
      );
    } finally {
      _capabilityPollInFlight = false;
    }
  }

  bool _isStaleCapabilityGenerationError(ControlError error) {
    if (error.code != ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) {
      return false;
    }
    final message = error.message.toLowerCase();
    return message.contains('stale capability generation') ||
        (message.contains('generation') && message.contains('stale'));
  }

  DiscoveredServer? _selectedServerMetadata() {
    final selected = _selectedDiscoveredServer;
    if (selected == null || selected.isEmpty) {
      return null;
    }
    for (final server in _discoveredServers) {
      if ('${server.host}:${server.port}' == selected) {
        return server;
      }
    }
    return null;
  }

  List<ControlCarrierKind> _carrierPreference(DiscoveredServer? server) {
    return buildCarrierPreference(
      isWebRuntime: kIsWeb,
      serverPriority: server?.priority ?? const <String>[],
      lastSuccessfulCarrier: _lastSuccessfulCarrier,
    );
  }

  String _websocketPathFor(DiscoveredServer? server) {
    final endpoint = server?.websocketEndpoint.trim() ?? '';
    if (endpoint.isEmpty) {
      return '/control';
    }
    final parsed = Uri.tryParse(endpoint);
    if (parsed != null && parsed.path.isNotEmpty) {
      return parsed.path;
    }
    return '/control';
  }

  int _grpcPortFor(DiscoveredServer? server, int fallbackPort) {
    final endpoint = server?.grpcEndpoint.trim() ?? '';
    if (endpoint.isEmpty) {
      return fallbackPort;
    }
    final parsed = Uri.tryParse('tcp://$endpoint');
    if (parsed == null || parsed.port <= 0) {
      return fallbackPort;
    }
    return parsed.port;
  }

  _ConnectionTarget _targetForCarrier({
    required ControlCarrierKind carrier,
    required DiscoveredServer? server,
    required String fallbackHost,
    required int fallbackPort,
  }) {
    switch (carrier) {
      case ControlCarrierKind.grpc:
        final endpoint = server?.grpcEndpoint.trim() ?? '';
        if (endpoint.isEmpty) {
          return _ConnectionTarget(host: fallbackHost, port: fallbackPort);
        }
        final parsed = Uri.tryParse('tcp://$endpoint');
        if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
          return _ConnectionTarget(host: fallbackHost, port: fallbackPort);
        }
        return _ConnectionTarget(host: parsed.host, port: parsed.port);
      case ControlCarrierKind.websocket:
        final endpoint = server?.websocketEndpoint.trim() ?? '';
        if (endpoint.isEmpty) {
          return _ConnectionTarget(host: fallbackHost, port: fallbackPort);
        }
        final parsed = Uri.tryParse(endpoint);
        if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
          return _ConnectionTarget(host: fallbackHost, port: fallbackPort);
        }
        return _ConnectionTarget(host: parsed.host, port: parsed.port);
      case ControlCarrierKind.tcp:
        final endpoint = server?.tcpEndpoint.trim() ?? '';
        if (endpoint.isEmpty) {
          return _ConnectionTarget(host: fallbackHost, port: 50055);
        }
        final parsed = Uri.tryParse('tcp://$endpoint');
        if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
          return _ConnectionTarget(host: fallbackHost, port: 50055);
        }
        return _ConnectionTarget(host: parsed.host, port: parsed.port);
      case ControlCarrierKind.http:
        final endpoint = server?.httpEndpoint.trim() ?? '';
        if (endpoint.isEmpty) {
          return _ConnectionTarget(host: fallbackHost, port: 50056);
        }
        final parsed = Uri.tryParse(endpoint);
        if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
          return _ConnectionTarget(host: fallbackHost, port: 50056);
        }
        return _ConnectionTarget(host: parsed.host, port: parsed.port);
    }
  }

  String _carrierName(ControlCarrierKind carrier) {
    switch (carrier) {
      case ControlCarrierKind.grpc:
        return 'gRPC';
      case ControlCarrierKind.websocket:
        return 'WebSocket';
      case ControlCarrierKind.tcp:
        return 'TCP';
      case ControlCarrierKind.http:
        return 'HTTP';
    }
  }

  String _carrierEndpointLabelForTarget({
    required ControlCarrierKind carrier,
    required _ConnectionTarget target,
    required DiscoveredServer? server,
  }) {
    switch (carrier) {
      case ControlCarrierKind.grpc:
        return '${target.host}:${target.port}';
      case ControlCarrierKind.websocket:
        return 'ws://${target.host}:${target.port}${_websocketPathFor(server)}';
      case ControlCarrierKind.tcp:
        return '${target.host}:${target.port}';
      case ControlCarrierKind.http:
        return 'http://${target.host}:${target.port}';
    }
  }

  String _currentPageHost() {
    return resolvePageHost(
      browserLocationHost: browser_host.browserLocationHost(),
      uriBaseHost: Uri.base.host,
    );
  }

  void _resetCarrierCycle(List<ControlCarrierKind> carriers) {
    _activeCarrierCycle = carriers;
    _activeCarrierIndex = 0;
    _carrierAttemptLog.clear();
  }

  bool _moveToNextCarrierInCycle() {
    if (_activeCarrierIndex + 1 >= _activeCarrierCycle.length) {
      return false;
    }
    _activeCarrierIndex += 1;
    return true;
  }

  String _buildCarrierFailureSummary() {
    if (_carrierAttemptLog.isEmpty) {
      return 'No carriers were attempted.';
    }
    final lines = <String>[];
    for (final attempt in _carrierAttemptLog) {
      lines.add(_formatCarrierAttempt(attempt));
    }
    return lines.join('\n');
  }

  String _sanitizeBugReportIdComponent(String raw) {
    final normalized = raw
        .trim()
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
        .replaceAll(RegExp(r'^-+'), '')
        .replaceAll(RegExp(r'-+$'), '');
    if (normalized.isEmpty) {
      return 'unknown';
    }
    return normalized;
  }

  String _buildLocalBugReportId({
    required DateTime now,
    required _BugIdentifier identifier,
    required String subjectDeviceID,
  }) {
    final reporter = _sanitizeBugReportIdComponent(_deviceId);
    final subject = _sanitizeBugReportIdComponent(subjectDeviceID);
    final code = _sanitizeBugReportIdComponent(identifier.code);
    return 'clientbug-${now.toUtc().millisecondsSinceEpoch}-$reporter-$subject-$code';
  }

  String _bugTransportContextSuffix() {
    final diagnostic = _lastTransportDiagnostic.trim();
    if (diagnostic.isEmpty) {
      return '';
    }
    return ' Last transport diagnostic: $diagnostic';
  }

  void _requeuePendingBugReportsForRetry(String reason) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final pending = List<_PendingBugReport>.from(_pendingBugReports);
    _pendingBugReports.clear();
    _queuedBugReports.insertAll(
      0,
      pending
          .map(
            (item) => _QueuedBugReport(
              bugReport: item.bugReport.deepCopy(),
              identifier: item.identifier,
              firstQueuedUnixMs: item.firstQueuedUnixMs,
              dispatchAttempts: item.dispatchAttempts,
            ),
          )
          .toList(),
    );
    final first = pending.first;
    _lastBugTokenWord = first.identifier.word;
    _lastBugTokenCode = first.identifier.code;
    _recordClientLog(
      'warn',
      're-queued ${pending.length} pending bug report(s) after transport failure: $reason',
    );
    if (!mounted) {
      return;
    }
    setState(() {
      _status = 'Bug report retry pending';
      _lastNotification =
          'Retrying bug report after transport recovery (word: ${first.identifier.word}, code: ${first.identifier.code}).';
      _bugReceiptState = _BugReceiptState.waiting;
      _bugReceiptReportId = '';
      _bugReceiptDetail =
          'Still waiting for a positive server receipt for word ${first.identifier.word}, code ${first.identifier.code}. Last failure: $reason';
    });
  }

  bool _shouldShowFullscreenStatusOverlay() {
    return _bugReceiptState != _BugReceiptState.none ||
        _lastTransportDiagnostic.trim().isNotEmpty ||
        _lastNotification.trim().isNotEmpty;
  }

  Widget _buildTransportStatusCard() {
    final blockText = buildControlStreamClipboardText(
      status: _status,
      notification: _lastNotification,
      transportDiagnostics: _lastTransportDiagnostic,
    );
    return Material(
      elevation: 4,
      borderRadius: BorderRadius.circular(12),
      color: Colors.white,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          border: Border.all(color: Colors.blueGrey.shade200),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            SelectableText(blockText, style: const TextStyle(fontSize: 12)),
            if (_bugReceiptState != _BugReceiptState.none) ...[
              const SizedBox(height: 8),
              _buildBugReceiptPanel(),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataFooter() {
    return SafeArea(
      top: false,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.end,
          mainAxisSize: MainAxisSize.max,
          children: [
            Flexible(
              child: SelectableText(
                buildMetadataLabel(buildDate: _buildDate, buildSha: _buildSha),
                textAlign: TextAlign.right,
                style: TextStyle(fontSize: 11, color: Colors.grey.shade700),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildBuildParityPanel() {
    final clientLabel =
        'Client ${buildMetadataLabel(buildDate: _buildDate, buildSha: _buildSha)}';
    final serverLabel = buildServerBuildLine(
      serverBuildDate: _serverBuildDate,
      serverBuildSha: _serverBuildSha,
      hasRegisterAck: _hasRegisterAck,
    );
    final parityLabel = buildVersionParityNote(
      clientBuildDate: _buildDate,
      clientBuildSha: _buildSha,
      serverBuildDate: _serverBuildDate,
      serverBuildSha: _serverBuildSha,
    );
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
          const SelectableText(
            'Client / Server Build',
            style: TextStyle(fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 4),
          SelectableText(clientLabel, style: const TextStyle(fontSize: 12)),
          SelectableText(serverLabel, style: const TextStyle(fontSize: 12)),
          SelectableText(parityLabel, style: const TextStyle(fontSize: 12)),
        ],
      ),
    );
  }

  @override
  void initState() {
    super.initState();
    _hostController.text = resolveInitialControlHost(
      isWebRuntime: kIsWeb,
      configuredHost: _hostController.text,
      pageHost: _currentPageHost(),
    );
    WidgetsBinding.instance.addObserver(this);
    final lifecycleState = WidgetsBinding.instance.lifecycleState;
    _appIsForeground =
        lifecycleState == null || lifecycleState == AppLifecycleState.resumed;
    _installFlutterErrorHook();
    _capabilityProbe = widget.capabilityProbeFactory();
    _mediaEngine = widget.mediaEngineFactory(
      localDeviceID: _deviceId,
      onSignal: _sendWebRTCSignalMessage,
    );
    _audioPlayback = widget.audioPlaybackFactory();
    _artifactExporter = DurableArtifactExporter();
    _recordClientLog('info', 'client started');
    if (_e2eEmitEvents) {
      debugPrint('E2E_EVENT: client_started');
    }
    if (_e2eAutoScanConnect) {
      unawaited(_runE2EAutoConnectFlow());
    }
    if (widget.autoConnectOnStartup) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || _hasActiveControlSession || _isConnecting) {
          return;
        }
        unawaited(_startStream(userInitiated: true));
      });
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

    final pageHost = _currentPageHost();
    final configuredHost = _hostController.text.trim();
    final host = resolveInitialControlHost(
      isWebRuntime: kIsWeb,
      configuredHost: configuredHost,
      pageHost: pageHost,
    );
    if (host != configuredHost) {
      _hostController.text = host;
    }
    final port = int.tryParse(_portController.text.trim());
    final selectedServer = _selectedServerMetadata();
    final size = _currentLogicalSize();
    if (host.isEmpty || port == null || port <= 0 || port > 65535) {
      _shouldStayConnected = false;
      _hasRegisterAck = false;
      setState(() {
        _status = 'Invalid host or port';
        _lastConnectionStatus = _status;
      });
      return;
    }

    if (userInitiated || _activeCarrierCycle.isEmpty) {
      _resetCarrierCycle(_carrierPreference(selectedServer));
    }
    if (_activeCarrierCycle.isEmpty) {
      _shouldStayConnected = false;
      _hasRegisterAck = false;
      setState(() {
        _status = 'No supported control carrier is available on this runtime';
        _lastConnectionStatus = _status;
      });
      return;
    }
    if (_activeCarrierIndex < 0 ||
        _activeCarrierIndex >= _activeCarrierCycle.length) {
      _activeCarrierIndex = 0;
    }

    final carrier = _activeCarrierCycle[_activeCarrierIndex];
    final rawTarget = _targetForCarrier(
      carrier: carrier,
      server: selectedServer,
      fallbackHost: host,
      fallbackPort: port,
    );
    final target = _ConnectionTarget(
      host: resolveInitialControlHost(
        isWebRuntime: kIsWeb,
        configuredHost: rawTarget.host,
        pageHost: pageHost,
      ),
      port: rawTarget.port,
    );
    final carrierEndpoint = _carrierEndpointLabelForTarget(
      carrier: carrier,
      target: target,
      server: selectedServer,
    );
    final attemptStartedAt = DateTime.now().toUtc();

    _isConnecting = true;
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    _capabilityGeneration = 0;
    _lastCapabilityAckGeneration = 0;
    if (mounted) {
      setState(() {
        final verb = userInitiated ? 'Connecting' : 'Reconnecting';
        _status = '$verb via ${_carrierName(carrier)}...';
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
      ControlClientTransportHint.configure(
        carrier: carrier,
        wsPath: _websocketPathFor(selectedServer),
        tcp: selectedServer?.tcpEndpoint,
        http: selectedServer?.httpEndpoint,
        desiredDeviceIdHint: _deviceId,
        resumeTokenHint: ControlClientTransportHint.resumeToken,
      );
      _client = widget.clientFactory(
        host: target.host,
        port: target.port,
      );

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
              _failPendingBugReportsForControlError(response.error);
              if (_isStaleCapabilityGenerationError(response.error)) {
                unawaited(
                  _probeAndPublishCapabilityChanges(
                    reason: 'stale_generation_rebaseline',
                    forceSnapshot: true,
                  ),
                );
              }
            }
            if (response.hasBugReportAck()) {
              _handleBugReportAck(response.bugReportAck);
            }
            if (response.hasRegisterAck()) {
              final firstRegisterAck = !_hasRegisterAck;
              if (firstRegisterAck) {
                shouldFlushQueuedBugReports = true;
                _sendScenarioRegistryQuery();
              }
              _hasRegisterAck = true;
              _cancelRegisterAckRetry();
              if (_pendingLaunchApplicationIntent.isNotEmpty) {
                final pendingIntent = _pendingLaunchApplicationIntent;
                _pendingLaunchApplicationIntent = '';
                _sendApplicationLaunchCommand(pendingIntent);
              }
              _lastSuccessfulCarrier = carrier;
              _carrierAttemptLog.clear();
            }
            if (response.hasCapabilityAck()) {
              _lastCapabilityAckGeneration =
                  response.capabilityAck.acceptedGeneration.toInt();
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
          final diagnosis = diagnoseTransportError(error, isWeb: kIsWeb);
          unawaited(
            _handleCarrierAttemptFailure(
              carrier: carrier,
              endpoint: carrierEndpoint,
              stage: 'stream',
              status: diagnosis.statusText(),
              rawError: diagnosis.hasSummary
                  ? diagnosis.notificationText()
                  : diagnosis.rawError,
              elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
            ),
          );
        },
        onDone: () {
          if (!_shouldStayConnected) {
            return;
          }
          if (!_hasRegisterAck) {
            unawaited(
              _handleCarrierAttemptFailure(
                carrier: carrier,
                endpoint: carrierEndpoint,
                stage: 'stream_closed',
                status: 'Disconnected',
                rawError: 'stream closed before register acknowledgement',
                elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
              ),
            );
            return;
          }
          unawaited(
            _handleCarrierAttemptFailure(
              carrier: carrier,
              endpoint: carrierEndpoint,
              stage: 'stream_closed',
              status: 'Disconnected',
              rawError: 'stream closed after register acknowledgement',
              elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
            ),
          );
        },
      );

      _syncMonitoringLoops();
      final touchInputLikely = switch (defaultTargetPlatform) {
        TargetPlatform.android => true,
        TargetPlatform.iOS => true,
        TargetPlatform.fuchsia => true,
        TargetPlatform.macOS => false,
        TargetPlatform.linux => false,
        TargetPlatform.windows => false,
      };
      final probedCapabilities = await _capabilityProbe.probe(
        CapabilityProbeContext(
          deviceId: _deviceId,
          deviceName: _deviceNameController.text.trim(),
          deviceType: _deviceTypeController.text.trim(),
          platform: _platformController.text.trim(),
          screenWidth: size.width.round(),
          screenHeight: size.height.round(),
          screenDensity: _currentDevicePixelRatio(),
          touchInputLikely: touchInputLikely,
          targetPlatform: defaultTargetPlatform,
        ),
      );
      final identity = probedCapabilities.hasIdentity()
          ? probedCapabilities.identity.deepCopy()
          : (capv1.DeviceIdentity()
            ..deviceName = _deviceNameController.text.trim()
            ..deviceType = _deviceTypeController.text.trim()
            ..platform = _platformController.text.trim());
      _outgoing.add(
        TerminalControlGrpcClient.helloRequest(
          deviceId: _deviceId,
          identity: identity,
          clientVersion: 'terminal_client',
        ),
      );

      _capabilityGeneration = 1;
      _lastRegisteredCapabilities = _applyLifecycleOperator(
        _applyDisplayMetadata(probedCapabilities.deepCopy()),
      );
      _lastCapabilitySignature = _capabilitySignature(
        _lastRegisteredCapabilities!,
      );
      _outgoing.add(
        TerminalControlGrpcClient.registerRequest(
          capabilities: _lastRegisteredCapabilities!,
        ),
      );
      _outgoing.add(
        TerminalControlGrpcClient.capabilitySnapshotRequest(
          deviceId: _deviceId,
          generation: _capabilityGeneration,
          capabilities: _lastRegisteredCapabilities!,
        ),
      );
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
      _registerAckRetryAttempts = 0;
      _scheduleRegisterAckRetry();
      _recordClientLog('info', 'control stream connected');
      _sendSensorTelemetry();
    } catch (error) {
      final diagnosis = diagnoseTransportError(error, isWeb: kIsWeb);
      await _handleCarrierAttemptFailure(
        carrier: carrier,
        endpoint: carrierEndpoint,
        stage: 'connect',
        status: diagnosis.hasSummary
            ? 'Connection error: ${diagnosis.summary}'
            : 'Connection error: $error',
        rawError: diagnosis.hasSummary
            ? diagnosis.notificationText()
            : diagnosis.rawError,
        elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
      );
    } finally {
      _isConnecting = false;
    }
  }

  void _startHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(widget.heartbeatInterval, (_) {
      if (!_hasActiveControlSession || _deviceId.isEmpty || !_appIsForeground) {
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
      if (!_hasActiveControlSession || _deviceId.isEmpty || !_appIsForeground) {
        return;
      }
      _sendSensorTelemetry();
    });
  }

  void _startCapabilityMonitorLoop() {
    _capabilityMonitorTimer?.cancel();
    _capabilityMonitorTimer = Timer.periodic(_capabilityMonitorInterval, (_) {
      unawaited(
          _probeAndPublishCapabilityChanges(reason: 'runtime_monitor_poll'));
    });
  }

  void _syncMonitoringLoops() {
    if (_hasActiveControlSession && _appIsForeground) {
      _startHeartbeatLoop();
      _startSensorTelemetryLoop();
      _startCapabilityMonitorLoop();
      return;
    }
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    _stopCapabilityMonitorLoop();
  }

  void _sendLifecycleCapabilityUpdate({String reason = 'lifecycle_state'}) {
    if (!_hasActiveControlSession) {
      return;
    }
    unawaited(_probeAndPublishCapabilityChanges(reason: reason));
  }

  List<String> _dedupeOperators(List<String> operators) {
    final seen = <String>{};
    final deduped = <String>[];
    for (final raw in operators) {
      final normalized = raw.trim();
      if (normalized.isEmpty || seen.contains(normalized)) {
        continue;
      }
      seen.add(normalized);
      deduped.add(normalized);
    }
    return deduped;
  }

  ConnectRequest _buildSensorTelemetryRequest() {
    final now = DateTime.now().toUtc();
    final values = <String, double>{
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

  void _sendScenarioRegistryQuery() {
    final requestID = _nextDebugRequestID('debug-scenario-registry');
    _pendingScenarioRegistryRequestID = requestID;
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = requestID
          ..kind = CommandKind.COMMAND_KIND_SYSTEM
          ..intent = 'scenario_registry'),
    );
  }

  void _cancelRegisterAckRetry() {
    _registerAckRetryTimer?.cancel();
    _registerAckRetryTimer = null;
    _registerAckRetryAttempts = 0;
    _registerAckRetryStartedUnixMs = 0;
  }

  void _scheduleRegisterAckRetry() {
    _registerAckRetryTimer?.cancel();
    if (_hasRegisterAck ||
        !_hasActiveControlSession ||
        _lastRegisteredCapabilities == null) {
      return;
    }
    if (_registerAckRetryStartedUnixMs <= 0) {
      _registerAckRetryStartedUnixMs = _nowUnixMs();
    }
    if (_nowUnixMs() - _registerAckRetryStartedUnixMs >=
        _registerAckTimeout.inMilliseconds) {
      _recordClientLog(
        'warn',
        'register acknowledgement still pending after '
            '${_registerAckTimeout.inSeconds}s and $_registerAckRetryAttempts retry attempts',
      );
      return;
    }
    _registerAckRetryTimer = Timer(_registerAckRetryInterval, () {
      if (_hasRegisterAck ||
          !_hasActiveControlSession ||
          _lastRegisteredCapabilities == null) {
        _cancelRegisterAckRetry();
        return;
      }
      _registerAckRetryAttempts += 1;
      _outgoing.add(
        TerminalControlGrpcClient.registerRequest(
          capabilities: _lastRegisteredCapabilities!,
        ),
      );
      _recordClientLog(
        'warn',
        'register acknowledgement pending; retrying bootstrap register '
            'attempt=$_registerAckRetryAttempts',
      );
      _scheduleRegisterAckRetry();
    });
  }

  void _launchSelectedApplication() {
    final intent = _selectedApplicationIntent.trim();
    if (intent.isEmpty) {
      setState(() {
        _status = 'Command error';
        _lastNotification = 'Select an application before launching';
      });
      return;
    }
    if (!_hasActiveControlSession || !_hasRegisterAck) {
      setState(() {
        _pendingLaunchApplicationIntent = intent;
        _status = 'Connecting';
        _lastNotification =
            'Connecting control stream to open application: $intent';
      });
      if (!_hasActiveControlSession && !_isConnecting) {
        unawaited(_startStream(userInitiated: true));
      }
      return;
    }
    _sendApplicationLaunchCommand(intent);
  }

  void _sendApplicationLaunchCommand(String intent) {
    _outgoing.add(
      ConnectRequest()
        ..command = (CommandRequest()
          ..requestId = _nextDebugRequestID('launch-app')
          ..deviceId = _deviceId
          ..action = CommandAction.COMMAND_ACTION_START
          ..kind = CommandKind.COMMAND_KIND_MANUAL
          ..intent = intent),
    );
    if (mounted) {
      setState(() {
        _lastNotification = 'Launching application: $intent';
      });
      return;
    }
    _lastNotification = 'Launching application: $intent';
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

  void _refreshAvailableApplications(Map<String, String> data) {
    final discovered = data.keys
        .map((key) => key.trim())
        .where((key) => key.isNotEmpty)
        .toSet();
    discovered.remove('terminal');
    final sorted = discovered.toList()..sort();
    _availableApplicationIntents = <String>['terminal', ...sorted];
    if (!_availableApplicationIntents.contains(_selectedApplicationIntent)) {
      _selectedApplicationIntent = _availableApplicationIntents.first;
    }
  }

  String _applicationLabel(String intent) {
    if (intent == 'terminal') {
      return 'terminal (REPL)';
    }
    return intent;
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
        requestID == _pendingScenarioRegistryRequestID) {
      diagnosticsTitle = 'scenario_registry';
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
    } else if (result.notification == 'System query: scenario_registry') {
      diagnosticsTitle = 'scenario_registry';
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
    } else if (diagnosticsTitle == 'scenario_registry') {
      _refreshAvailableApplications(data);
    }
  }

  void _applyRegisterMetadata(ConnectResponse response) {
    if (!response.hasRegisterAck()) {
      return;
    }
    final metadata = Map<String, String>.from(response.registerAck.metadata);
    _serverBuildSha = normalizeBuildValue(
      metadata[_registerMetadataServerBuildShaKey] ?? '',
    );
    _serverBuildDate = normalizeBuildValue(
      metadata[_registerMetadataServerBuildDateKey] ?? '',
    );
    if (metadata.isNotEmpty) {
      _diagnosticsTitle = 'register_ack';
      _diagnosticsData = metadata;
      final photoBaseURL =
          metadata[_registerMetadataPhotoFrameBaseURLKey]?.trim() ?? '';
      if (photoBaseURL.isNotEmpty) {
        _photoFrameAssetBaseURL = photoBaseURL;
        _lastNotification = 'Photo frame asset base URL configured';
      }
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
    debugPrint('[client][$level] $message');
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
    final size = _currentLogicalSize();
    final devicePixelRatio = _currentDevicePixelRatio();
    final orientation = _lastKnownOrientation;

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
      ..screenWidthPx = size.width.round()
      ..screenHeightPx = size.height.round()
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
    final screenshotPng = await _captureBugReportScreenshot();
    final bugReport = diagv1.BugReport()
      ..reportId = _buildLocalBugReportId(
        now: now,
        identifier: identifier,
        subjectDeviceID: subjectDeviceID,
      )
      ..reporterDeviceId = _deviceId
      ..subjectDeviceId = subjectDeviceID
      ..source = source
      ..description = description
      ..timestampUnixMs = Int64(now.toUtc().millisecondsSinceEpoch)
      ..clientContext = _buildClientContext();
    if (screenshotPng.isNotEmpty) {
      bugReport.screenshotPng = screenshotPng;
      bugReport.sourceHints['screenshot_byte_count'] =
          screenshotPng.length.toString();
    }
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
    final firstQueuedUnixMs = _nowUnixMs();
    if (_isBugReportTransportReady()) {
      _dispatchBugReport(
        bugReport: bugReport,
        identifier: identifier,
        subjectDeviceID: subjectDeviceID,
        firstQueuedUnixMs: firstQueuedUnixMs,
        previousDispatchAttempts: 0,
        replay: false,
      );
      setState(() {
        _lastBugTokenWord = identifier.word;
        _lastBugTokenCode = identifier.code;
        _status = 'Bug report pending';
        _lastNotification =
            'Submitting bug report (word: ${identifier.word}, code: ${identifier.code}) and waiting for server ack...';
        _bugReceiptState = _BugReceiptState.waiting;
        _bugReceiptReportId = '';
        _bugReceiptDetail =
            'Waiting for server receipt for word ${identifier.word}, code ${identifier.code}.';
      });
      return;
    }

    _queuedBugReports.add(
      _QueuedBugReport(
        bugReport: bugReport.deepCopy(),
        identifier: identifier,
        firstQueuedUnixMs: firstQueuedUnixMs,
        dispatchAttempts: 0,
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
          'word=${identifier.word} code=${identifier.code} report_id=${bugReport.reportId}',
    );
    setState(() {
      _lastBugTokenWord = identifier.word;
      _lastBugTokenCode = identifier.code;
      _status = 'Bug report queued';
      _lastNotification =
          'Bug report queued (word: ${identifier.word}, code: ${identifier.code}). Connecting and will send automatically.';
      _bugReceiptState = _BugReceiptState.waiting;
      _bugReceiptReportId = '';
      _bugReceiptDetail =
          'Queued and waiting for a positive server receipt for word ${identifier.word}, code ${identifier.code}.';
    });
    _ensureBugReportAckWatchdog();
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

  Future<List<int>> _captureBugReportScreenshot() async {
    try {
      final overrideCapture = widget.bugReportScreenshotCapture;
      if (overrideCapture != null) {
        return await overrideCapture();
      }
      final bindingType = WidgetsBinding.instance.runtimeType.toString();
      if (bindingType.contains('TestWidgetsFlutterBinding')) {
        return const <int>[];
      }
      if (!mounted) {
        return const <int>[];
      }
      final screenshotContext = _bugReportScreenshotKey.currentContext;
      if (screenshotContext == null) {
        return const <int>[];
      }
      final renderObject = screenshotContext.findRenderObject();
      if (renderObject is! RenderRepaintBoundary) {
        return const <int>[];
      }
      final pixelRatio = math.min(
        1.0,
        View.maybeOf(screenshotContext)?.devicePixelRatio ??
            _lastKnownDevicePixelRatio,
      );
      final image = await renderObject.toImage(
        pixelRatio: pixelRatio > 0 ? pixelRatio : 1.0,
      );
      try {
        final byteData = await image.toByteData(
          format: ui.ImageByteFormat.png,
        );
        if (byteData == null) {
          return const <int>[];
        }
        return Uint8List.sublistView(
          byteData.buffer.asUint8List(
            byteData.offsetInBytes,
            byteData.lengthInBytes,
          ),
        );
      } finally {
        image.dispose();
      }
    } catch (error) {
      _recordClientLog(
        'warn',
        'bug report screenshot capture failed error=$error',
      );
      return const <int>[];
    }
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
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return;
    }
    SemanticsService.sendAnnouncement(
      views.first,
      'Bug reference word ${identifier.word}. Code ${identifier.code}',
      _lastKnownTextDirection,
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
      'last_transport_diagnostic': _lastTransportDiagnostic,
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
    final receiptID = ack.reportId.trim();
    if (receiptID.isEmpty) {
      _status = 'Bug report receipt error';
      _lastNotification = tokenWord.isEmpty
          ? 'Bug report failed: no positive receipt from server.'
          : 'Bug report failed: no positive receipt from server (word: $tokenWord).';
      _bugReceiptState = _BugReceiptState.error;
      _bugReceiptReportId = '';
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'No positive receipt was generated by the server.'
          : 'No positive receipt was generated by the server for word $tokenWord, code $tokenCode.';
      _recordClientLog(
        'error',
        'bug report ack missing report_id status=${ack.status.name} '
            'word=$tokenWord code=$tokenCode',
      );
      return;
    }
    _status = 'Bug report filed';
    _lastNotification = tokenWord.isEmpty
        ? 'Bug report filed: $receiptID'
        : 'Bug report filed: $receiptID (word: $tokenWord)';
    _bugReceiptState = _BugReceiptState.received;
    _bugReceiptReportId = receiptID;
    final ackMessage = ack.message.trim();
    if (ackMessage == 'ack_replayed') {
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'Positive receipt recovered after transport failover. The server replayed the original acknowledgement.'
          : 'Positive receipt recovered after transport failover for word $tokenWord, code $tokenCode.';
    } else {
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'Positive receipt received from server.'
          : 'Positive receipt received for word $tokenWord, code $tokenCode.';
    }
    _recordClientLog(
      'info',
      'bug report ack status=${ack.status.name} id=$receiptID '
          'word=$tokenWord code=$tokenCode message=${ack.message}',
    );
  }

  void _failPendingBugReportsForControlError(ControlError error) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final failed = _pendingBugReports.removeAt(0);
    _lastBugTokenWord = failed.identifier.word;
    _lastBugTokenCode = failed.identifier.code;
    final reason = error.message.trim().isNotEmpty
        ? error.message.trim()
        : error.code.name;
    _status = 'Bug report receipt error';
    _lastNotification =
        'Bug report failed: $reason (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
    _bugReceiptState = _BugReceiptState.error;
    _bugReceiptReportId = '';
    _bugReceiptDetail =
        'No positive receipt could be generated: $reason.${_bugTransportContextSuffix()}';
    _recordClientLog(
      'error',
      'bug report rejected by control error code=${error.code.name} '
          'message=$reason word=${failed.identifier.word} code=${failed.identifier.code}',
    );
    _pendingBugReports.clear();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
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
    required int firstQueuedUnixMs,
    required int previousDispatchAttempts,
    required bool replay,
  }) {
    _outgoing.add(
      ConnectRequest()..bugReport = bugReport.deepCopy(),
    );
    final nextAttempt = previousDispatchAttempts + 1;
    _pendingBugReports.add(
      _PendingBugReport(
        bugReport: bugReport.deepCopy(),
        identifier: identifier,
        firstQueuedUnixMs: firstQueuedUnixMs,
        submittedUnixMs: _nowUnixMs(),
        dispatchAttempts: nextAttempt,
      ),
    );
    _ensureBugReportAckWatchdog();
    _recordClientLog(
      'info',
      '${replay ? 'replayed' : 'submitted'} bug report for subject=$subjectDeviceID '
          'word=${identifier.word} code=${identifier.code} '
          'report_id=${bugReport.reportId} attempt=$nextAttempt',
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
        bugReport: item.bugReport.deepCopy(),
        identifier: item.identifier,
        subjectDeviceID: item.bugReport.subjectDeviceId,
        firstQueuedUnixMs: item.firstQueuedUnixMs,
        previousDispatchAttempts: item.dispatchAttempts,
        replay: true,
      );
    }
    if (mounted) {
      setState(() {
        _status = 'Queued bug reports sent';
        _lastNotification =
            'Sent ${queued.length} queued bug report(s); waiting for server ack.';
        _bugReceiptState = _BugReceiptState.waiting;
        _bugReceiptReportId = '';
        _bugReceiptDetail = queued.length == 1
            ? 'Queued bug report sent. Waiting for positive server receipt.'
            : 'Sent ${queued.length} queued bug reports. Waiting for positive server receipts.';
      });
    }
  }

  void _ensureBugReportAckWatchdog() {
    if (_bugReportAckTimer != null) {
      return;
    }
    _bugReportAckTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      final nowUnixMs = _nowUnixMs();
      if (_pendingBugReports.isNotEmpty) {
        final first = _pendingBugReports.first;
        if (nowUnixMs - first.firstQueuedUnixMs >=
            _bugReportAckTimeout.inMilliseconds) {
          final failed = _pendingBugReports.removeAt(0);
          _lastBugTokenWord = failed.identifier.word;
          _lastBugTokenCode = failed.identifier.code;
          if (mounted) {
            setState(() {
              _status = 'Bug report receipt error';
              _lastNotification =
                  'Bug report failed: no positive receipt from server (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
              _bugReceiptState = _BugReceiptState.error;
              _bugReceiptReportId = '';
              _bugReceiptDetail =
                  'No positive receipt was generated by the server for word ${failed.identifier.word}, code ${failed.identifier.code}.${_bugTransportContextSuffix()}';
            });
          }
          _recordClientLog(
            'error',
            'bug report ack timeout word=${failed.identifier.word} code=${failed.identifier.code} '
                'attempts=${failed.dispatchAttempts} transport_diagnostic=$_lastTransportDiagnostic',
          );
        }
      } else if (_queuedBugReports.isNotEmpty &&
          _bugReceiptState == _BugReceiptState.waiting) {
        final firstQueued = _queuedBugReports.first;
        if (nowUnixMs - firstQueued.firstQueuedUnixMs >=
            _bugReportAckTimeout.inMilliseconds) {
          _lastBugTokenWord = firstQueued.identifier.word;
          _lastBugTokenCode = firstQueued.identifier.code;
          if (mounted) {
            setState(() {
              _status = 'Bug report receipt error';
              _lastNotification =
                  'Bug report failed: no positive receipt while queued (word: ${firstQueued.identifier.word}, code: ${firstQueued.identifier.code}).';
              _bugReceiptState = _BugReceiptState.error;
              _bugReceiptReportId = '';
              _bugReceiptDetail =
                  'No positive receipt could be generated because the report remained queued for more than ${_bugReportAckTimeout.inSeconds}s.${_bugTransportContextSuffix()}';
            });
          }
          _recordClientLog(
            'error',
            'queued bug report receipt timeout word=${firstQueued.identifier.word} code=${firstQueued.identifier.code} '
                'transport_diagnostic=$_lastTransportDiagnostic',
          );
        }
      }
      if (_pendingBugReports.isEmpty &&
          (_queuedBugReports.isEmpty ||
              _bugReceiptState != _BugReceiptState.waiting)) {
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
        _status = 'Bug report receipt error';
        _lastNotification =
            'Bug report failed: no positive receipt (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
        _bugReceiptState = _BugReceiptState.error;
        _bugReceiptReportId = '';
        _bugReceiptDetail = 'No positive receipt could be generated: $reason.';
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

  void _stopCapabilityMonitorLoop() {
    _capabilityMonitorTimer?.cancel();
    _capabilityMonitorTimer = null;
  }

  void _cancelReconnectTimer() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }

  Future<void> _handleCarrierAttemptFailure({
    required ControlCarrierKind carrier,
    required String endpoint,
    required String stage,
    required String status,
    required String rawError,
    required Duration elapsed,
  }) async {
    final failureClass =
        classifyCarrierFailure(stage: stage, rawError: rawError);
    _carrierAttemptLog.add(
      _CarrierAttemptDiagnostic(
        carrier: carrier,
        endpoint: endpoint,
        stage: stage,
        failureClass: failureClass,
        error: rawError,
        elapsed: elapsed,
      ),
    );
    final attemptSummary = _formatCarrierAttempt(_carrierAttemptLog.last);
    _lastTransportDiagnostic = attemptSummary;
    _recordClientLog(
      'warn',
      'control carrier failure carrier=${_carrierName(carrier)} stage=$stage '
          'class=$failureClass endpoint=$endpoint elapsed_ms=${elapsed.inMilliseconds} '
          'error=$rawError',
    );
    if (_shouldStayConnected) {
      _requeuePendingBugReportsForRetry(attemptSummary);
    }

    _incoming = null;
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    final existingClient = _client;
    _client = null;
    if (existingClient != null) {
      await existingClient.shutdown();
    }
    _syncMonitoringLoops();

    if (_moveToNextCarrierInCycle()) {
      final nextCarrier = _activeCarrierCycle[_activeCarrierIndex];
      _recordClientLog(
        'info',
        'switching control carrier from ${_carrierName(carrier)} to ${_carrierName(nextCarrier)}',
      );
      if (mounted) {
        setState(() {
          _status =
              '${_carrierName(carrier)} failed, trying ${_carrierName(nextCarrier)}...';
          _lastConnectionStatus = _status;
          _lastNotification = _lastTransportDiagnostic;
        });
      }
      if (_shouldStayConnected) {
        unawaited(
          Future<void>.microtask(() => _startStream(userInitiated: false)),
        );
      }
      return;
    }

    final summary = _buildCarrierFailureSummary();
    _lastTransportDiagnostic = summary;
    _recordClientLog('error', 'all control carriers failed\n$summary');
    if (mounted) {
      setState(() {
        _status = 'All control carriers failed';
        _lastConnectionStatus = _status;
        _lastNotification = summary;
      });
    }
    _activeCarrierCycle = <ControlCarrierKind>[];
    _activeCarrierIndex = 0;
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
    if (kIsWeb) {
      setState(() {
        _status = 'Connected (web origin); LAN scan not required';
        _lastConnectionStatus = _status;
      });
      return;
    }
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
          final first = found.first;
          _selectedDiscoveredServer = '${first.host}:${first.port}';
          _hostController.text = first.host;
          _portController.text = _grpcPortFor(first, first.port).toString();
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
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    _capabilityGeneration = 0;
    _lastCapabilityAckGeneration = 0;
    _reconnectAttempt = 0;
    _activeCarrierCycle = <ControlCarrierKind>[];
    _activeCarrierIndex = 0;
    _cancelReconnectTimer();
    _syncMonitoringLoops();
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
  void didChangeAppLifecycleState(AppLifecycleState state) {
    final wasForeground = _appIsForeground;
    switch (state) {
      case AppLifecycleState.resumed:
        _appIsForeground = true;
        break;
      case AppLifecycleState.inactive:
      case AppLifecycleState.hidden:
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
        _appIsForeground = false;
        break;
    }
    if (_appIsForeground == wasForeground) {
      return;
    }
    _syncMonitoringLoops();
    _sendLifecycleCapabilityUpdate(reason: 'app_lifecycle_change');
  }

  @override
  void didChangeMetrics() {
    _sendLifecycleCapabilityUpdate(reason: 'display_geometry_change');
  }

  @override
  void dispose() {
    _shouldStayConnected = false;
    _cancelRegisterAckRetry();
    WidgetsBinding.instance.removeObserver(this);
    _cancelReconnectTimer();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    _stopCapabilityMonitorLoop();
    final incoming = _incoming;
    if (incoming != null) {
      unawaited(incoming.cancel());
    }
    _outgoing.close();
    unawaited(_mediaEngine.dispose());
    unawaited(_audioPlayback.dispose());
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
    _terminalInputFocusNode.dispose();
    _playbackArtifactIdController.dispose();
    _playbackTargetDeviceIdController.dispose();
    _restoreFlutterErrorHook();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    _lastKnownLogicalSize = mediaQuery.size;
    _lastKnownDevicePixelRatio = mediaQuery.devicePixelRatio;
    _lastKnownOrientation = mediaQuery.orientation.name;
    final directionality = Directionality.maybeOf(context);
    if (directionality != null) {
      _lastKnownTextDirection = directionality;
    }
    final showTerminalFullscreen = _isTerminalRoot(_activeRoot);
    if (showTerminalFullscreen) {
      return RepaintBoundary(
        key: _bugReportScreenshotKey,
        child: Scaffold(
          floatingActionButton: FloatingActionButton.extended(
            onPressed: _showBugReportDialog,
            icon: const Icon(Icons.bug_report_outlined),
            label: const Text('Report Bug'),
          ),
          bottomNavigationBar: _buildMetadataFooter(),
          body: SafeArea(
            child: Stack(
              children: [
                SizedBox.expand(
                  child: _renderNode(_activeRoot!),
                ),
                if (_shouldShowFullscreenStatusOverlay())
                  Positioned(
                    top: 12,
                    right: 12,
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 420),
                      child: _buildTransportStatusCard(),
                    ),
                  ),
              ],
            ),
          ),
        ),
      );
    }
    return RepaintBoundary(
      key: _bugReportScreenshotKey,
      child: Scaffold(
        floatingActionButton: FloatingActionButton.extended(
          onPressed: _showBugReportDialog,
          icon: const Icon(Icons.bug_report_outlined),
          label: const Text('Report Bug'),
        ),
        bottomNavigationBar: _buildMetadataFooter(),
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
                      if (kIsWeb)
                        Chip(
                          avatar: Icon(
                            _hasRegisterAck
                                ? Icons.check_circle_outline
                                : Icons.sync_outlined,
                          ),
                          label: Text(
                            buildWebConnectionChipLabel(
                              hasRegisterAck: _hasRegisterAck,
                              isConnecting: _isConnecting,
                              shouldStayConnected: _shouldStayConnected,
                            ),
                          ),
                        )
                      else
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
                          final selected = _selectedServerMetadata();
                          final parsedPort = int.tryParse(parts[1]);
                          _portController.text = _grpcPortFor(
                            selected,
                            parsedPort ?? _defaultGrpcPort,
                          ).toString();
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
                  const SizedBox(height: 12),
                  _buildBuildParityPanel(),
                  const SizedBox(height: 20),
                  SelectableText('Control Stream: $_status'),
                  const SizedBox(height: 12),
                  SelectableText('Responses: $_responses'),
                  SelectableText(
                    'Media routes: ${_routesByStreamID.length}  Active streams: ${_activeStreamsByID.length}  Signals: ${_recentWebRTCSignals.length}',
                  ),
                  SelectableText(
                    'Sensor sends: $_sensorSendCount  Last sensor unix_ms: $_lastSensorSendUnixMs  Stream-ready acks: $_streamReadyAckCount  Capability ack gen: $_lastCapabilityAckGeneration',
                  ),
                  if (_playAudioCount > 0)
                    SelectableText(
                      'Play audio msgs: $_playAudioCount  Last play bytes: $_lastPlayAudioBytes  Last play target: $_lastPlayAudioDeviceID  Last play source: $_lastPlayAudioSource',
                    ),
                  if (_photoFrameAssetBaseURL.isNotEmpty)
                    SelectableText(
                        'Photo frame assets: $_photoFrameAssetBaseURL'),
                  if (_lastNotification.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    SelectableText('Notification: $_lastNotification'),
                  ],
                  if (_bugReceiptState != _BugReceiptState.none) ...[
                    const SizedBox(height: 12),
                    _buildBugReceiptPanel(),
                  ],
                  if (_lastTransportDiagnostic.isNotEmpty ||
                      _carrierAttemptLog.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _buildTransportDiagnosticsPanel(),
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
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    initialValue: _availableApplicationIntents
                            .contains(_selectedApplicationIntent)
                        ? _selectedApplicationIntent
                        : _availableApplicationIntents.first,
                    decoration: const InputDecoration(
                      labelText: 'Available Application',
                    ),
                    items: _availableApplicationIntents
                        .map(
                          (intent) => DropdownMenuItem<String>(
                            value: intent,
                            child: Text(_applicationLabel(intent)),
                          ),
                        )
                        .toList(),
                    onChanged: (value) {
                      if (value == null) {
                        return;
                      }
                      setState(() {
                        _selectedApplicationIntent = value;
                      });
                    },
                  ),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 12,
                    children: [
                      OutlinedButton(
                        onPressed: _launchSelectedApplication,
                        child: const Text('Open Application'),
                      ),
                      OutlinedButton(
                        onPressed: _sendScenarioRegistryQuery,
                        child: const Text('Refresh Applications'),
                      ),
                    ],
                  ),
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

  Widget _buildBugReceiptPanel() {
    late final Color borderColor;
    late final Color backgroundColor;
    late final IconData icon;
    late final String title;
    switch (_bugReceiptState) {
      case _BugReceiptState.none:
        return const SizedBox.shrink();
      case _BugReceiptState.waiting:
        borderColor = Colors.amber.shade400;
        backgroundColor = Colors.amber.shade50;
        icon = Icons.schedule_outlined;
        title = 'Bug Report Receipt: Pending';
        break;
      case _BugReceiptState.received:
        borderColor = Colors.green.shade400;
        backgroundColor = Colors.green.shade50;
        icon = Icons.verified_outlined;
        title = 'Bug Report Receipt: Received';
        break;
      case _BugReceiptState.error:
        borderColor = Colors.red.shade400;
        backgroundColor = Colors.red.shade50;
        icon = Icons.error_outline;
        title = 'Bug Report Receipt: Error';
        break;
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: borderColor),
        borderRadius: BorderRadius.circular(8),
        color: backgroundColor,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: const TextStyle(fontWeight: FontWeight.w600)),
                if (_bugReceiptReportId.isNotEmpty)
                  Text(
                    'Receipt ID: $_bugReceiptReportId',
                    style: const TextStyle(fontSize: 12),
                  ),
                if (_bugReceiptDetail.isNotEmpty)
                  Text(_bugReceiptDetail, style: const TextStyle(fontSize: 12)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTransportDiagnosticsPanel() {
    final recentAttempts = _carrierAttemptLog.reversed.take(4).toList();
    final blockText = buildTransportDiagnosticsClipboardText(
      lastTransportDiagnostic: _lastTransportDiagnostic,
      recentAttempts: recentAttempts
          .map((attempt) => _formatCarrierAttempt(attempt))
          .toList(),
    );
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
          SelectableText(blockText, style: const TextStyle(fontSize: 12)),
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
    if (response.hasRequestArtifact()) {
      return 'Artifact requested';
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
        unawaited(_startMediaStream(start.deepCopy()));
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
      unawaited(_executePlayAudio(response.playAudio.deepCopy()));
    }
    if (response.hasRequestArtifact()) {
      unawaited(_handleRequestArtifact(response.requestArtifact.deepCopy()));
    }
  }

  Future<void> _startMediaStream(iov1.StartStream start) async {
    try {
      await _ensureMediaPermissionsForStart(start);
      await _mediaEngine.startStream(start);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Media permission required';
        _lastConnectionStatus = _status;
        _lastNotification =
            'Unable to start media stream ${start.streamId}: $error';
      });
    }
  }

  Future<void> _ensureMediaPermissionsForStart(iov1.StartStream start) async {
    final sourceDevice = start.sourceDeviceId.trim();
    if (sourceDevice != _deviceId) {
      return;
    }
    final wantsAudio = start.kind.toLowerCase().contains('audio');
    final wantsVideo = start.kind.toLowerCase().contains('video');
    if (!wantsAudio && !wantsVideo) {
      return;
    }
    final stream = await navigator.mediaDevices.getUserMedia(
      <String, dynamic>{
        'audio': wantsAudio,
        'video': wantsVideo,
      },
    );
    for (final track in stream.getTracks()) {
      track.stop();
    }
    await stream.dispose();
  }

  Future<void> _executePlayAudio(iov1.PlayAudio playAudio) async {
    final source = switch (playAudio.whichSource()) {
      iov1.PlayAudio_Source.pcmData => 'pcm_data',
      iov1.PlayAudio_Source.url => 'url',
      iov1.PlayAudio_Source.ttsText => 'tts_text',
      iov1.PlayAudio_Source.notSet => 'not_set',
    };
    final bytes = playAudio.whichSource() == iov1.PlayAudio_Source.pcmData
        ? playAudio.pcmData.length
        : 0;
    try {
      await _audioPlayback.play(playAudio);
      if (playAudio.whichSource() == iov1.PlayAudio_Source.pcmData &&
          playAudio.pcmData.isNotEmpty &&
          playAudio.requestId.trim().isNotEmpty) {
        await _artifactExporter.save(
          'play_audio/${playAudio.requestId.trim()}',
          Uint8List.fromList(playAudio.pcmData),
        );
      }
      if (!mounted) {
        return;
      }
      setState(() {
        _playAudioCount += 1;
        _lastPlayAudioBytes = bytes;
        _lastPlayAudioDeviceID =
            playAudio.deviceId.isNotEmpty ? playAudio.deviceId : 'unknown';
        _lastPlayAudioSource = source;
        _lastNotification =
            'Play audio: $_lastPlayAudioDeviceID ($source, $bytes bytes)';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _lastNotification = 'Play audio failed: $error';
      });
    }
  }

  Future<void> _handleRequestArtifact(iov1.RequestArtifact request) async {
    final artifactID = request.artifactId.trim();
    if (artifactID.isEmpty) {
      return;
    }
    try {
      final payload = await _artifactExporter.exportByID(artifactID);
      final nowUnixMs = _nowUnixMs();
      _outgoing.add(
        ConnectRequest()
          ..artifactAvailable = (iov1.ArtifactAvailable()
            ..artifact = (iov1.ArtifactRef()
              ..id = artifactID
              ..kind = 'artifact.binary'
              ..source = (iov1.DeviceRef()..deviceId = _deviceId)
              ..startUnixMs = Int64(nowUnixMs)
              ..endUnixMs = Int64(nowUnixMs)
              ..uri = 'local://artifact/$artifactID?bytes=${payload.length}')),
      );
      if (mounted) {
        setState(() {
          _lastNotification = 'Artifact available: $artifactID';
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _lastNotification = 'Artifact request failed: $artifactID ($error)';
        });
      }
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
      _recordClientLog('warn',
          '_sendKeyText called with deviceId.isEmpty=${_deviceId.isEmpty} text.isEmpty=${text.isEmpty}');
      return;
    }
    _recordClientLog('info',
        'sending key text: ${text.replaceAll(String.fromCharCode(127), '<DEL>')}');
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

  bool _isTerminalRoot(uiv1.Node? node) {
    if (node == null) {
      return false;
    }
    return _nodeId(node).trim() == 'terminal_root';
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
        if (isTerminalInput &&
            _terminalInputController.text != _terminalInputShadow) {
          _recordClientLog('warn',
              'terminal_input shadow mismatch on render: controller="${_terminalInputController.text}" shadow="$_terminalInputShadow"');
          _terminalInputShadow = _terminalInputController.text;
        }
        return TextField(
          controller: _terminalInputController,
          focusNode: isTerminalInput ? _terminalInputFocusNode : null,
          decoration: InputDecoration(
            hintText: node.textInput.placeholder,
          ),
          // Terminal input receives frequent server-driven output patches.
          // Re-applying autofocus on each rebuild can cause single-character
          // replace behavior on web, so keep autofocus off for this field.
          autofocus: isTerminalInput ? false : node.textInput.autofocus,
          onChanged: (value) {
            if (isTerminalInput) {
              final previous = _terminalInputShadow;
              _recordClientLog('info',
                  'terminal_input.onChanged: prev="$previous" new="$value" controller.text="${_terminalInputController.text}"');
              if (value.startsWith(previous) &&
                  value.length > previous.length) {
                final inserted = value.substring(previous.length);
                if (inserted.isNotEmpty) {
                  _recordClientLog('info', 'detected insertion: "$inserted"');
                  unawaited(_sendKeyText(inserted));
                }
              } else if (previous.startsWith(value) &&
                  previous.length > value.length) {
                final removed = previous.length - value.length;
                if (removed > 0) {
                  _recordClientLog('info', 'detected $removed backspace(s)');
                  unawaited(
                      _sendKeyText(List<String>.filled(removed, '\b').join()));
                }
              } else if (value != previous) {
                _recordClientLog('warn',
                    'shadow sync lost: shadow="$previous" controller="$value" (no clear insertion/deletion)');
              }
              _terminalInputShadow = value;
            }
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
        final streamID = node.videoSurface.trackId.trim();
        return Container(
          key: ValueKey<String>('ui-video-surface-$componentId'),
          margin: const EdgeInsets.symmetric(vertical: 6),
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: Colors.black,
            border: Border.all(color: Colors.blueGrey.shade700),
            borderRadius: BorderRadius.circular(8),
          ),
          child: SizedBox(
            height: 160,
            child: Stack(
              children: [
                Positioned.fill(
                  child: _VideoSurfaceView(
                    streamListenable: _mediaEngine.remoteStream(streamID),
                  ),
                ),
                if (streamID.isNotEmpty)
                  Align(
                    alignment: Alignment.bottomRight,
                    child: Container(
                      margin: const EdgeInsets.all(6),
                      padding: const EdgeInsets.symmetric(
                        horizontal: 6,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: Colors.black54,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        streamID,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 11,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        );
      case uiv1.Node_Widget.audioVisualizer:
        final componentId = _nodeId(node);
        final streamID = node.audioVisualizer.streamId.trim();
        return Container(
          key: ValueKey<String>('ui-audio-visualizer-$componentId'),
          margin: const EdgeInsets.symmetric(vertical: 6),
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            border: Border.all(color: Colors.blueGrey.shade200),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Audio level'),
              if (streamID.isNotEmpty)
                Text(streamID, style: const TextStyle(fontSize: 12)),
              const SizedBox(height: 8),
              ValueListenableBuilder<double>(
                valueListenable: _mediaEngine.audioLevel(streamID),
                builder: (context, level, _) {
                  return LinearProgressIndicator(
                    value: level > 0 ? level.clamp(0.0, 1.0) : null,
                  );
                },
              ),
            ],
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

class _VideoSurfaceView extends StatefulWidget {
  const _VideoSurfaceView({
    required this.streamListenable,
  });

  final ValueListenable<MediaStream?> streamListenable;

  @override
  State<_VideoSurfaceView> createState() => _VideoSurfaceViewState();
}

class _VideoSurfaceViewState extends State<_VideoSurfaceView> {
  final RTCVideoRenderer _renderer = RTCVideoRenderer();
  MediaStream? _boundStream;
  bool _rendererReady = false;

  @override
  void initState() {
    super.initState();
    widget.streamListenable.addListener(_syncStream);
    unawaited(_initializeRenderer());
  }

  Future<void> _initializeRenderer() async {
    await _renderer.initialize();
    _rendererReady = true;
    await _bind(widget.streamListenable.value);
    if (mounted) {
      setState(() {});
    }
  }

  Future<void> _syncStream() async {
    await _bind(widget.streamListenable.value);
    if (mounted) {
      setState(() {});
    }
  }

  Future<void> _bind(MediaStream? stream) async {
    if (!_rendererReady || identical(stream, _boundStream)) {
      return;
    }
    _boundStream = stream;
    _renderer.srcObject = stream;
  }

  @override
  void didUpdateWidget(covariant _VideoSurfaceView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!identical(oldWidget.streamListenable, widget.streamListenable)) {
      oldWidget.streamListenable.removeListener(_syncStream);
      widget.streamListenable.addListener(_syncStream);
      unawaited(_syncStream());
    }
  }

  @override
  void dispose() {
    widget.streamListenable.removeListener(_syncStream);
    unawaited(_renderer.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final hasVideo = _boundStream?.getVideoTracks().isNotEmpty ?? false;
    if (!_rendererReady || !hasVideo) {
      return const Center(
        child: Icon(Icons.videocam_off_outlined),
      );
    }
    return RTCVideoView(
      _renderer,
      objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitContain,
    );
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

enum _BugReceiptState {
  none,
  waiting,
  received,
  error,
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
    required this.bugReport,
    required this.identifier,
    required this.firstQueuedUnixMs,
    required this.submittedUnixMs,
    required this.dispatchAttempts,
  });

  final diagv1.BugReport bugReport;
  final _BugIdentifier identifier;
  final int firstQueuedUnixMs;
  final int submittedUnixMs;
  final int dispatchAttempts;
}

class _QueuedBugReport {
  const _QueuedBugReport({
    required this.bugReport,
    required this.identifier,
    required this.firstQueuedUnixMs,
    required this.dispatchAttempts,
  });

  final diagv1.BugReport bugReport;
  final _BugIdentifier identifier;
  final int firstQueuedUnixMs;
  final int dispatchAttempts;
}
