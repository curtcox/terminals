import 'package:flutter/widgets.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:terminal_client/capabilities/probe.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/media/playback.dart';
import 'package:terminal_client/media/webrtc_engine.dart';
import 'package:terminal_client/util/alert_delivery.dart';

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
typedef AlertDelivery = void Function({
  required String title,
  required String body,
  required String level,
});
typedef UnixMsProvider = int Function();
typedef BugReportScreenshotCapture = Future<List<int>> Function();
typedef WakeWordDetectorFactory = WakeWordDetectorController Function();
typedef MediaPermissionProbe = Future<void> Function({
  required bool audio,
  required bool video,
});

class WakeWordUtterance {
  const WakeWordUtterance({
    required this.audio,
    required this.sampleRate,
    required this.isFinal,
  });

  final List<int> audio;
  final int sampleRate;
  final bool isFinal;
}

abstract class WakeWordDetectorController {
  Future<void> setEnabled(bool enabled);
  void setOnUtterance(void Function(WakeWordUtterance utterance)? onUtterance);
  Future<void> dispose();
}

class NoopWakeWordDetectorController implements WakeWordDetectorController {
  @override
  Future<void> setEnabled(bool enabled) async {}

  @override
  void setOnUtterance(
      void Function(WakeWordUtterance utterance)? onUtterance) {}

  @override
  Future<void> dispose() async {}
}

int defaultUnixMsProvider() => DateTime.now().toUtc().millisecondsSinceEpoch;

CapabilityProbe defaultCapabilityProbeFactory() {
  final bindingType = WidgetsBinding.instance.runtimeType.toString();
  if (bindingType.contains('TestWidgetsFlutterBinding')) {
    return DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[],
    );
  }
  return DefaultCapabilityProbe();
}

AudioPlayback defaultAudioPlaybackFactory() {
  final bindingType = WidgetsBinding.instance.runtimeType.toString();
  if (bindingType.contains('TestWidgetsFlutterBinding')) {
    return NoopAudioPlayback();
  }
  return AudioPlayerPlayback();
}

final AlertDeliveryService _defaultAlertDeliveryService =
    AlertDeliveryService();

void defaultAlertDelivery({
  required String title,
  required String body,
  required String level,
}) {
  _defaultAlertDeliveryService.deliver(
    title: title,
    body: body,
    level: level,
  );
}

WakeWordDetectorController defaultWakeWordDetectorFactory() {
  return NoopWakeWordDetectorController();
}

Future<void> defaultMediaPermissionProbe({
  required bool audio,
  required bool video,
}) async {
  final stream = await navigator.mediaDevices.getUserMedia(
    <String, dynamic>{
      'audio': audio,
      'video': video,
    },
  );
  for (final track in stream.getTracks()) {
    track.stop();
  }
  await stream.dispose();
}

TerminalControlClient defaultTerminalControlClientFactory({
  required String host,
  required int port,
}) {
  return createTerminalControlClient(host: host, port: port);
}
