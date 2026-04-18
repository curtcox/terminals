import 'dart:async';
import 'dart:typed_data';

import 'package:fixnum/fixnum.dart';
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
    this.desiredDeviceId = '',
    this.resumeToken = '',
    this.onResumeToken,
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
  final String desiredDeviceId;
  final String resumeToken;
  final void Function(String token)? onResumeToken;
  final WebSocketChannel _channel;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    _sendTransportHello();
    final controller = StreamController<ConnectResponse>();
    StreamSubscription<ConnectRequest>? outgoing;
    StreamSubscription<dynamic>? incoming;
    var outgoingSequence = 1;

    outgoing = requests.listen(
      (ConnectRequest message) {
        final envelope = WireEnvelope()
          ..protocolVersion = wireProtocolVersion
          ..sequence = Int64(outgoingSequence)
          ..clientMessage = message;
        outgoingSequence += 1;
        _channel.sink.add(Uint8List.fromList(envelope.writeToBuffer()));
      },
      onError: controller.addError,
      onDone: () {
        _channel.sink.close();
      },
    );

    incoming = _channel.stream.listen(
      (dynamic data) {
        late final List<int> bytes;
        if (data is Uint8List) {
          bytes = data;
        } else if (data is List<int>) {
          bytes = data;
        } else {
          controller.addError(StateError(
              'unexpected websocket frame type ${data.runtimeType}'));
          return;
        }
        final envelope = WireEnvelope.fromBuffer(bytes);
        if (envelope.hasTransportError()) {
          final error = envelope.transportError;
          controller.addError(
            StateError('transport error ${error.code}: ${error.message}'),
          );
          return;
        }
        if (envelope.hasTransportHelloAck()) {
          final ack = envelope.transportHelloAck;
          if (ack.acceptedProtocolVersion != wireProtocolVersion) {
            controller.addError(
              StateError(
                'transport hello rejected protocol version '
                '${ack.acceptedProtocolVersion}',
              ),
            );
            return;
          }
          final token = ack.resumeToken.trim();
          if (token.isNotEmpty) {
            onResumeToken?.call(token);
          }
          return;
        }
        if (envelope.hasServerMessage()) {
          controller.add(envelope.serverMessage);
        }
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

  void _sendTransportHello() {
    final helloEnvelope = WireEnvelope()
      ..protocolVersion = wireProtocolVersion
      ..sessionId = _newSessionId()
      ..transportHello = (TransportHello()
        ..protocolVersion = wireProtocolVersion
        ..supportedCarriers.add(CarrierKind.CARRIER_KIND_WEBSOCKET)
        ..desiredDeviceId = desiredDeviceId
        ..resumeToken = resumeToken);
    _channel.sink.add(Uint8List.fromList(helloEnvelope.writeToBuffer()));
  }

  String _newSessionId() {
    final epochMs = DateTime.now().toUtc().millisecondsSinceEpoch;
    return 'ws-$epochMs';
  }
}
