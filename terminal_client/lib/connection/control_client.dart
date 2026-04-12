import 'dart:async';

import 'package:fixnum/fixnum.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

/// Thin gRPC client wrapper around TerminalControlService.Connect.
class TerminalControlGrpcClient {
  TerminalControlGrpcClient({
    required this.host,
    required this.port,
  })  : _channel = ClientChannel(
          host,
          port: port,
          options: const ChannelOptions(
            credentials: ChannelCredentials.insecure(),
          ),
        ),
        _stub = _TerminalControlServiceClient(_channel);

  final String host;
  final int port;
  final ClientChannel _channel;
  final _TerminalControlServiceClient _stub;

  /// Starts the bidirectional control stream.
  ResponseStream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    return _stub.connect(requests, options: options);
  }

  /// Gracefully closes the underlying channel.
  Future<void> shutdown() => _channel.shutdown();

  /// Builds a canonical register message for session bootstrap.
  static ConnectRequest registerRequest({
    required String deviceId,
    required String deviceName,
    required String deviceType,
    required String platform,
  }) {
    return ConnectRequest()
      ..register = (RegisterDevice()
        ..capabilities = (DeviceCapabilities()
          ..deviceId = deviceId
          ..identity = (DeviceIdentity()
            ..deviceName = deviceName
            ..deviceType = deviceType
            ..platform = platform));
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
