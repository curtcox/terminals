import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:terminal_client/app/client_dependencies.dart';
import 'package:terminal_client/app/terminal_client_shell.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/media/webrtc_engine.dart';

export 'package:terminal_client/app/client_dependencies.dart';

const bool _autoConnectOnStartupDefault = bool.fromEnvironment(
  'TERMINALS_AUTO_CONNECT_ON_STARTUP',
  defaultValue: kIsWeb,
);

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({
    super.key,
    this.clientFactory = defaultTerminalControlClientFactory,
    this.capabilityProbeFactory = defaultCapabilityProbeFactory,
    this.mediaEngineFactory = defaultClientMediaEngineFactory,
    this.audioPlaybackFactory = defaultAudioPlaybackFactory,
    this.alertDelivery = defaultAlertDelivery,
    this.heartbeatInterval = const Duration(seconds: 10),
    this.sensorTelemetryInterval = const Duration(seconds: 15),
    this.reconnectDelayBase = const Duration(seconds: 2),
    this.reconnectDelayMaxSeconds = 30,
    this.nowUnixMsProvider = defaultUnixMsProvider,
    this.autoConnectOnStartup = _autoConnectOnStartupDefault,
    this.wakeWordDetectorFactory = defaultWakeWordDetectorFactory,
    this.bugReportScreenshotCapture,
    this.screenMetricsProvider,
    this.screenMetricsChangeListenable,
    this.displayGeometryDebounceInterval = kDisplayGeometryDebounceInterval,
    this.mediaPermissionProbe = defaultMediaPermissionProbe,
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

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: TerminalClientShell(
        clientFactory: clientFactory,
        capabilityProbeFactory: capabilityProbeFactory,
        mediaEngineFactory: mediaEngineFactory,
        audioPlaybackFactory: audioPlaybackFactory,
        alertDelivery: alertDelivery,
        heartbeatInterval: heartbeatInterval,
        sensorTelemetryInterval: sensorTelemetryInterval,
        reconnectDelayBase: reconnectDelayBase,
        reconnectDelayMaxSeconds: reconnectDelayMaxSeconds,
        nowUnixMsProvider: nowUnixMsProvider,
        autoConnectOnStartup: autoConnectOnStartup,
        wakeWordDetectorFactory: wakeWordDetectorFactory,
        bugReportScreenshotCapture: bugReportScreenshotCapture,
        screenMetricsProvider: screenMetricsProvider,
        screenMetricsChangeListenable: screenMetricsChangeListenable,
        displayGeometryDebounceInterval: displayGeometryDebounceInterval,
        mediaPermissionProbe: mediaPermissionProbe,
      ),
    );
  }
}
