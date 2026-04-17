import 'dart:async';
import 'dart:typed_data';

import 'package:grpc/grpc.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'control_client.dart';

/// Websocket control-stream client for browser sessions.
class TerminalControlWebSocketClient implements TerminalControlClient {
  TerminalControlWebSocketClient({
    required this.host,
    required this.port,
    this.path = '/control',
    this.secure = false,
  }) : _channel = WebSocketChannel.connect(
          Uri(
            scheme: secure ? 'wss' : 'ws',
            host: host,
            port: port,
            path: path,
          ),
        );

  final String host;
  final int port;
  final String path;
  final bool secure;
  final WebSocketChannel _channel;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    final controller = StreamController<ConnectResponse>();
    StreamSubscription<ConnectRequest>? outgoing;
    StreamSubscription<dynamic>? incoming;

    outgoing = requests.listen(
      (ConnectRequest message) {
        _channel.sink.add(Uint8List.fromList(message.writeToBuffer()));
      },
      onError: controller.addError,
      onDone: () {
        _channel.sink.close();
      },
    );

    incoming = _channel.stream.listen(
      (dynamic data) {
        if (data is Uint8List) {
          controller.add(ConnectResponse.fromBuffer(data));
          return;
        }
        if (data is List<int>) {
          controller.add(ConnectResponse.fromBuffer(data));
          return;
        }
        controller.addError(
            StateError('unexpected websocket frame type ${data.runtimeType}'));
      },
      onError: controller.addError,
      onDone: () async {
        await outgoing?.cancel();
        await controller.close();
      },
    );

    controller.onCancel = () async {
      await outgoing?.cancel();
      await incoming?.cancel();
      await _channel.sink.close();
    };

    return controller.stream;
  }

  @override
  Future<void> shutdown() => _channel.sink.close();
}
