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
    required DeviceCapabilities capabilities,
  }) {
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
