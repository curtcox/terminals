import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:terminal_client/app/client_dependencies.dart';
import 'package:terminal_client/app/terminal_client_view_state.dart';
import 'package:terminal_client/app/video_surface_view.dart';
import 'package:terminal_client/capabilities/capability_session.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/capabilities/probe.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:terminal_client/connection/carrier_preference.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/connection/control_response_dispatcher.dart';
import 'package:terminal_client/connection/control_session_controller.dart';
import 'package:terminal_client/connection/endpoint_resolution.dart';
import 'package:terminal_client/connection/reliability.dart';
import 'package:terminal_client/connection/transport_diagnostics.dart';
import 'package:terminal_client/diagnostics/bug_report_chrome.dart';
import 'package:terminal_client/diagnostics/client_chrome.dart';
import 'package:terminal_client/discovery/mdns_scanner.dart';
import 'package:terminal_client/edge/artifact_export.dart';
import 'package:terminal_client/edge/bundle_store.dart';
import 'package:terminal_client/edge/host.dart';
import 'package:terminal_client/edge/retention.dart';
import 'package:terminal_client/edge/scheduler.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/media/playback.dart';
import 'package:terminal_client/media/webrtc_engine.dart';
import 'package:terminal_client/ui/idle_main_layer_placeholder.dart';
import 'package:terminal_client/ui/server_driven_action.dart';
import 'package:terminal_client/ui/server_driven_node_key.dart';
import 'package:terminal_client/ui/server_driven_renderer.dart';
import 'package:terminal_client/util/browser_host.dart' as browser_host;
import 'package:terminal_client/util/speech.dart' as speech;

part 'terminal_client_shell_display.dart';
part 'terminal_client_shell_carrier.dart';
part 'terminal_client_shell_capabilities.dart';
part 'terminal_client_shell_connection.dart';
part 'terminal_client_shell_monitoring.dart';
part 'terminal_client_shell_diagnostics.dart';
part 'terminal_client_shell_bug_report.dart';
part 'terminal_client_shell_media.dart';
part 'terminal_client_shell_ui.dart';

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
const int _clientContextRecentUiCap = 32;
const int _clientContextRecentLogCap = 200;
const int _clientContextRecentErrorCap = 32;
const Duration _capabilityMonitorInterval = Duration(seconds: 2);
const RetryPolicy _bugReportAckRetryPolicy = RetryPolicy(
  interval: Duration(seconds: 1),
  maxDuration: Duration(seconds: 20),
);
const RetryPolicy _registerAckRetryPolicy = RetryPolicy(
  interval: Duration(milliseconds: 400),
  maxDuration: Duration(seconds: 20),
);
const RetryPolicy _readinessPolicy = RetryPolicy(
  interval: Duration(milliseconds: 120),
  maxDuration: Duration(seconds: 20),
);

class TerminalClientShell extends StatefulWidget {
  const TerminalClientShell({
    super.key,
    required this.clientFactory,
    required this.capabilityProbeFactory,
    required this.mediaEngineFactory,
    required this.audioPlaybackFactory,
    required this.alertDelivery,
    required this.heartbeatInterval,
    required this.sensorTelemetryInterval,
    required this.reconnectDelayBase,
    required this.reconnectDelayMaxSeconds,
    required this.nowUnixMsProvider,
    required this.autoConnectOnStartup,
    required this.wakeWordDetectorFactory,
    required this.bugReportScreenshotCapture,
    required this.screenMetricsProvider,
    required this.screenMetricsChangeListenable,
    required this.displayGeometryDebounceInterval,
    required this.mediaPermissionProbe,
    this.displaySurfaceMode = false,
  });

  final TerminalControlClientFactory clientFactory;
  final CapabilityProbeFactory capabilityProbeFactory;
  final ClientMediaEngineFactory mediaEngineFactory;
  final AudioPlaybackFactory audioPlaybackFactory;
  final AlertDelivery alertDelivery;
  final Duration heartbeatInterval;
  final Duration sensorTelemetryInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;
  final UnixMsProvider nowUnixMsProvider;
  final bool autoConnectOnStartup;
  final WakeWordDetectorFactory wakeWordDetectorFactory;
  final BugReportScreenshotCapture? bugReportScreenshotCapture;
  final ScreenMetricsProvider? screenMetricsProvider;
  final Listenable? screenMetricsChangeListenable;
  final Duration displayGeometryDebounceInterval;
  final MediaPermissionProbe mediaPermissionProbe;
  /// When true, after registration and before the first server `SetUI`, show
  /// [idleMainLayerPlaceholderRoot] fullscreen (terminal-ui plan, Phase H).
  /// Defaults from [TerminalClientApp] via `TERMINALS_DISPLAY_SURFACE`.
  final bool displaySurfaceMode;

  @override
  State<TerminalClientShell> createState() => _TerminalClientShellState();
}

class _TerminalClientShellState extends State<TerminalClientShell>
    with WidgetsBindingObserver {
  final GlobalKey _bugReportScreenshotKey = GlobalKey();
  final TextEditingController _hostController = TextEditingController(
    text: _defaultControlHost,
  );
  final TextEditingController _portController = TextEditingController(
    text: _defaultControlPort.toString(),
  );
  final TextEditingController _grpcEndpointController = TextEditingController();
  final TextEditingController _websocketEndpointController =
      TextEditingController();
  final TextEditingController _tcpEndpointController = TextEditingController();
  final TextEditingController _httpEndpointController = TextEditingController();
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
  late final WakeWordDetectorController _wakeWordDetector;
  late final DurableArtifactExporter _artifactExporter;
  late final Future<EdgeHost> _edgeHostFuture;
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
  Timer? _displayGeometryDebounceTimer;
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
  String _serverBuildSha = 'unknown';
  String _serverBuildDate = 'unknown';
  final List<diagv1.UiEventEntry> _recentUiEvents = <diagv1.UiEventEntry>[];
  final List<diagv1.UiActionEntry> _recentUiActions = <diagv1.UiActionEntry>[];
  final List<diagv1.LogEntry> _recentLogs = <diagv1.LogEntry>[];
  final List<diagv1.ControlErrorEntry> _recentControlErrors =
      <diagv1.ControlErrorEntry>[];
  final Map<String, double> _lastSensorSnapshot = <String, double>{};
  final CapabilitySession _capabilitySession = CapabilitySession();

  /// Effective UI root: server `SetUI` tree, or the idle placeholder while
  /// [TerminalClientShell.displaySurfaceMode] is enabled and awaiting first UI.
  uiv1.Node? get _resolvedUiRoot {
    final serverRoot = _activeRoot;
    if (serverRoot != null) {
      return serverRoot;
    }
    if (widget.displaySurfaceMode && _isConnectionRegistered) {
      return idleMainLayerPlaceholderRoot();
    }
    return null;
  }
  int _lastHeartbeatUnixMs = 0;
  int _pendingHeartbeatUnixMs = 0;
  double _lastRttMs = 0;
  String _lastConnectionStatus = 'Idle';
  String _lastFlutterErrorMessage = '';
  String _lastFlutterErrorStack = '';
  int _lastFlutterErrorUnixMs = 0;
  String _lastTransportDiagnostic = '';
  final List<CarrierAttemptDiagnostic> _carrierAttemptLog =
      <CarrierAttemptDiagnostic>[];
  List<ControlCarrierKind> _activeCarrierCycle = <ControlCarrierKind>[];
  int _activeCarrierIndex = 0;
  ControlCarrierKind? _lastSuccessfulCarrier;
  String _lastBugTokenWord = '';
  String _lastBugTokenCode = '';
  BugReceiptChromeState _bugReceiptState = BugReceiptChromeState.none;
  String _bugReceiptDetail = '';
  String _bugReceiptReportId = '';
  final List<QueuedBugReport> _queuedBugReports = <QueuedBugReport>[];
  final List<PendingBugReport> _pendingBugReports = <PendingBugReport>[];
  Timer? _bugReportAckTimer;
  bool _hasRegisterAck = false;
  FlutterExceptionHandler? _previousFlutterErrorHandler;
  bool _appIsForeground = true;
  Size _lastKnownLogicalSize = Size.zero;
  double _lastKnownDevicePixelRatio = 1.0;
  EdgeInsets _lastKnownSafeAreaInsets = EdgeInsets.zero;
  String _lastKnownOrientation = 'unknown';
  TextDirection _lastKnownTextDirection = TextDirection.ltr;
  bool _capabilityPollInFlight = false;
  String _lastObservedDisplaySignature = '';
  bool _privacyModeEnabled = false;
  bool _wakeWordDetectorEnabled = false;
  late final ConnectionReadinessGateway _readinessGateway;
  late final ReliableSendDispatcher<ConnectRequest> _reliableSender;
  late final RetryController _registerAckRetryController;

  void _rebuildState(VoidCallback fn) => setState(fn);

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
    _wakeWordDetector = widget.wakeWordDetectorFactory();
    _wakeWordDetector.setOnUtterance(_handleWakeWordUtterance);
    _artifactExporter = DurableArtifactExporter();
    _edgeHostFuture = _createEdgeHost();
    _readinessGateway = ConnectionReadinessGateway(
      currentPhase: () => _connectionPhase,
      startConnection: _ensureConnectedForDispatch,
      policy: _readinessPolicy,
    );
    _reliableSender = ReliableSendDispatcher<ConnectRequest>(
      sendNow: (request) => _outgoing.add(request),
      gateway: _readinessGateway,
    );
    _registerAckRetryController = RetryController(
      policy: _registerAckRetryPolicy,
      nowUtc: () => DateTime.fromMillisecondsSinceEpoch(
        _nowUnixMs(),
        isUtc: true,
      ),
    );
    widget.screenMetricsChangeListenable
        ?.addListener(_handleInjectedScreenMetricsChanged);
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

  @override
  void didUpdateWidget(covariant TerminalClientShell oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.screenMetricsChangeListenable !=
        widget.screenMetricsChangeListenable) {
      oldWidget.screenMetricsChangeListenable
          ?.removeListener(_handleInjectedScreenMetricsChanged);
      widget.screenMetricsChangeListenable
          ?.addListener(_handleInjectedScreenMetricsChanged);
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
    if (_appIsForeground &&
        _shouldStayConnected &&
        !_hasActiveControlSession &&
        !_isConnecting) {
      _reconnectAttempt = 0;
      _scheduleReconnect();
    }
    _sendLifecycleCapabilityUpdate(reason: 'app_lifecycle_change');
  }

  @override
  void didChangeMetrics() {
    _refreshDisplayMetrics();
    _scheduleDisplayGeometryCapabilityUpdate();
  }

  void _handleInjectedScreenMetricsChanged() {
    _refreshDisplayMetrics();
    _scheduleDisplayGeometryCapabilityUpdate();
  }

  @override
  void dispose() {
    _shouldStayConnected = false;
    _cancelRegisterAckRetry();
    WidgetsBinding.instance.removeObserver(this);
    _cancelReconnectTimer();
    _displayGeometryDebounceTimer?.cancel();
    _displayGeometryDebounceTimer = null;
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
    _wakeWordDetector.setOnUtterance(null);
    unawaited(_wakeWordDetector.dispose());
    final existingClient = _client;
    if (existingClient != null) {
      unawaited(existingClient.shutdown());
    }
    _hostController.dispose();
    _portController.dispose();
    _grpcEndpointController.dispose();
    _websocketEndpointController.dispose();
    _tcpEndpointController.dispose();
    _httpEndpointController.dispose();
    _deviceNameController.dispose();
    _deviceTypeController.dispose();
    _platformController.dispose();
    _terminalInputController.dispose();
    _terminalInputFocusNode.dispose();
    _playbackArtifactIdController.dispose();
    _playbackTargetDeviceIdController.dispose();
    _restoreFlutterErrorHook();
    widget.screenMetricsChangeListenable
        ?.removeListener(_handleInjectedScreenMetricsChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final injectedMetrics = _injectedScreenMetrics();
    if (injectedMetrics != null) {
      _lastKnownLogicalSize = injectedMetrics.logicalSize;
      _lastKnownDevicePixelRatio = injectedMetrics.devicePixelRatio;
      _lastKnownSafeAreaInsets = injectedMetrics.safeAreaInsets;
      _lastKnownOrientation = injectedMetrics.orientation;
    } else {
      final mediaQuery = MediaQuery.of(context);
      _lastKnownLogicalSize = mediaQuery.size;
      _lastKnownDevicePixelRatio = mediaQuery.devicePixelRatio;
      _lastKnownSafeAreaInsets = mediaQuery.viewPadding;
      _lastKnownOrientation = mediaQuery.orientation.name;
    }
    final displaySignature = _displaySignature(
      logicalSize: _lastKnownLogicalSize,
      devicePixelRatio: _lastKnownDevicePixelRatio,
      safeAreaInsets: _lastKnownSafeAreaInsets,
      orientation: _lastKnownOrientation,
    );
    if (displaySignature != _lastObservedDisplaySignature) {
      _lastObservedDisplaySignature = displaySignature;
      _scheduleDisplayGeometryCapabilityUpdate();
    }
    final directionality = Directionality.maybeOf(context);
    if (directionality != null) {
      _lastKnownTextDirection = directionality;
    }
    final resolvedRoot = _resolvedUiRoot;
    final hideClientChrome = shouldHideClientChromeForRoot(resolvedRoot);
    if (hideClientChrome) {
      final root = resolvedRoot!;
      return RepaintBoundary(
        key: _bugReportScreenshotKey,
        child: Scaffold(
          resizeToAvoidBottomInset: false,
          floatingActionButton:
              BugReportButton(onPressed: _showBugReportDialog),
          bottomNavigationBar: _buildMetadataFooter(),
          body: SafeArea(
            child: Stack(
              children: [
                SizedBox.expand(
                  child: _buildServerDrivenRenderer(root),
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
        floatingActionButton: BugReportButton(onPressed: _showBugReportDialog),
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
                        ConnectionPhaseChip(
                          phase: _connectionPhase,
                          isRegistered: _isConnectionRegistered,
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
                  ExpansionTile(
                    title: const Text('Carrier Endpoint Overrides (Optional)'),
                    childrenPadding: const EdgeInsets.only(bottom: 8),
                    children: [
                      TextField(
                        controller: _grpcEndpointController,
                        decoration: const InputDecoration(
                          labelText: 'gRPC Endpoint Override',
                          hintText: 'host:50051',
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _websocketEndpointController,
                        decoration: const InputDecoration(
                          labelText: 'WebSocket Endpoint Override',
                          hintText: 'ws://host:50054/control',
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _tcpEndpointController,
                        decoration: const InputDecoration(
                          labelText: 'TCP Endpoint Override',
                          hintText: 'host:50055',
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _httpEndpointController,
                        decoration: const InputDecoration(
                          labelText: 'HTTP Endpoint Override',
                          hintText: 'http://host:50056',
                        ),
                      ),
                    ],
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
                    ],
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
                    'Sensor sends: $_sensorSendCount  Last sensor unix_ms: $_lastSensorSendUnixMs  Stream-ready acks: $_streamReadyAckCount  Capability ack gen: ${_capabilitySession.lastAckGeneration}',
                  ),
                  if (_playAudioCount > 0)
                    SelectableText(
                      'Play audio msgs: $_playAudioCount  Last play bytes: $_lastPlayAudioBytes  Last play target: $_lastPlayAudioDeviceID  Last play source: $_lastPlayAudioSource',
                    ),
                  if (_lastNotification.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Text('Notification: $_lastNotification'),
                  ],
                  if (_bugReceiptState != BugReceiptChromeState.none) ...[
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
                          child: _buildServerDrivenRenderer(_activeRoot!),
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
}
