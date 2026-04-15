import 'dart:async';

import 'package:fixnum/fixnum.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

/// Transport contract used by the app for control-stream lifecycle.
abstract class TerminalControlClient {
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  });

  Future<void> shutdown();
}

/// Thin gRPC client wrapper around TerminalControlService.Connect.
class TerminalControlGrpcClient implements TerminalControlClient {
  TerminalControlGrpcClient({
    required this.host,
    required this.port,
  }) : _channel = ClientChannel(
          host,
          port: port,
          options: const ChannelOptions(
            credentials: ChannelCredentials.insecure(),
          ),
        );

  final String host;
  final int port;
  final ClientChannel _channel;
  late final _TerminalControlServiceClient _stub =
      _TerminalControlServiceClient(_channel);

  /// Starts the bidirectional control stream.
  @override
  ResponseStream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    return _stub.connect(requests, options: options);
  }

  /// Gracefully closes the underlying channel.
  @override
  Future<void> shutdown() => _channel.shutdown();

  /// Builds a canonical register message for session bootstrap.
  static ConnectRequest registerRequest({
    required String deviceId,
    required String deviceName,
    required String deviceType,
    required String platform,
    int? screenWidth,
    int? screenHeight,
    double? screenDensity,
    bool? screenTouch,
    bool keyboardPhysical = true,
    String keyboardLayout = 'en-US',
    String pointerType = 'touch_or_mouse',
    bool pointerHover = true,
    int speakerChannels = 2,
    List<int> speakerSampleRates = const [44100, 48000],
    int microphoneChannels = 1,
    List<int> microphoneSampleRates = const [16000, 44100, 48000],
    List<String> edgeRuntimes = const <String>[],
    List<String> edgeOperators = const <String>[],
    int edgeCPURealtime = 0,
    int edgeGPURealtime = 0,
    int edgeNPURealtime = 0,
    int edgeMemMB = 0,
    int edgeAudioRetentionSec = 0,
    int edgeVideoRetentionSec = 0,
    int edgeSensorRetentionSec = 0,
    int edgeRadioRetentionSec = 0,
    double edgeSyncErrorMs = 0,
    bool edgeMicArray = false,
    bool edgeCameraIntrinsics = false,
    bool edgeCompass = false,
  }) {
    final capabilities = DeviceCapabilities()
      ..deviceId = deviceId
      ..identity = (DeviceIdentity()
        ..deviceName = deviceName
        ..deviceType = deviceType
        ..platform = platform)
      ..keyboard = (KeyboardCapability()
        ..physical = keyboardPhysical
        ..layout = keyboardLayout)
      ..pointer = (PointerCapability()
        ..type = pointerType
        ..hover = pointerHover)
      ..speakers = (AudioOutputCapability()
        ..channels = speakerChannels
        ..sampleRates.addAll(speakerSampleRates))
      ..microphone = (AudioInputCapability()
        ..channels = microphoneChannels
        ..sampleRates.addAll(microphoneSampleRates))
      ..edge = (EdgeCapability()
        ..runtimes.addAll(edgeRuntimes)
        ..operators.addAll(edgeOperators)
        ..compute = (EdgeComputeCapability()
          ..cpuRealtime = edgeCPURealtime
          ..gpuRealtime = edgeGPURealtime
          ..npuRealtime = edgeNPURealtime
          ..memMb = edgeMemMB)
        ..retention = (EdgeRetentionCapability()
          ..audioSec = edgeAudioRetentionSec
          ..videoSec = edgeVideoRetentionSec
          ..sensorSec = edgeSensorRetentionSec
          ..radioSec = edgeRadioRetentionSec)
        ..timing = (EdgeTimingCapability()..syncErrorMs = edgeSyncErrorMs)
        ..geometry = (EdgeGeometryCapability()
          ..micArray = edgeMicArray
          ..cameraIntrinsics = edgeCameraIntrinsics
          ..compass = edgeCompass));

    if (screenWidth != null || screenHeight != null || screenDensity != null) {
      capabilities.screen = (ScreenCapability()
        ..width = screenWidth ?? 0
        ..height = screenHeight ?? 0
        ..density = screenDensity ?? 1.0
        ..touch = screenTouch ?? false);
    }

    return ConnectRequest()
      ..register = (RegisterDevice()..capabilities = capabilities);
  }

  /// Builds a heartbeat message.
  static ConnectRequest heartbeatRequest({
    required String deviceId,
    required int unixMs,
  }) {
    return ConnectRequest()
      ..heartbeat = (Heartbeat()
        ..deviceId = deviceId
        ..unixMs = Int64(unixMs));
  }
}

class _TerminalControlServiceClient extends Client {
  _TerminalControlServiceClient(super.channel);

  static final ClientMethod<ConnectRequest, ConnectResponse> _connectMethod =
      ClientMethod<ConnectRequest, ConnectResponse>(
    '/terminals.control.v1.TerminalControlService/Connect',
    (ConnectRequest value) => value.writeToBuffer(),
    (List<int> value) => ConnectResponse.fromBuffer(value),
  );

  ResponseStream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    return $createStreamingCall(
      _connectMethod,
      requests,
      options: options,
    );
  }
}
