import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:fixnum/fixnum.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

import 'control_client.dart';

class TerminalControlTcpClient implements TerminalControlClient {
  TerminalControlTcpClient({
    required this.host,
    required this.port,
    this.desiredDeviceId = '',
    this.resumeToken = '',
    this.onResumeToken,
  });

  final String host;
  final int port;
  final String desiredDeviceId;
  final String resumeToken;
  final void Function(String token)? onResumeToken;

  Socket? _socket;
  StreamSubscription<ConnectRequest>? _outgoingSub;
  StreamSubscription<List<int>>? _incomingSub;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    final controller = StreamController<ConnectResponse>();
    _connectAndBind(requests, controller);
    controller.onCancel = () async {
      await shutdown();
    };
    return controller.stream;
  }

  Future<void> _connectAndBind(
    Stream<ConnectRequest> requests,
    StreamController<ConnectResponse> controller,
  ) async {
    try {
      final socket = await Socket.connect(host, port);
      _socket = socket;
      var outgoingSequence = 1;

      final hello = WireEnvelope()
        ..protocolVersion = wireProtocolVersion
        ..sessionId = _newSessionId()
        ..transportHello = (TransportHello()
          ..protocolVersion = wireProtocolVersion
          ..supportedCarriers.add(CarrierKind.CARRIER_KIND_TCP)
          ..desiredDeviceId = desiredDeviceId
          ..resumeToken = resumeToken);
      _writeEnvelope(socket, hello);

      _incomingSub = socket.listen(
        (List<int> data) {
          _frameDecoder.add(data);
          while (_frameDecoder.hasFrame) {
            final frame = _frameDecoder.takeFrame();
            final envelope = WireEnvelope.fromBuffer(frame);
            if (envelope.hasTransportError()) {
              final error = envelope.transportError;
              controller.addError(
                StateError('transport error ${error.code}: ${error.message}'),
              );
              continue;
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
                continue;
              }
              final token = ack.resumeToken.trim();
              if (token.isNotEmpty) {
                onResumeToken?.call(token);
              }
              continue;
            }
            if (envelope.hasServerMessage()) {
              controller.add(envelope.serverMessage);
            }
          }
        },
        onError: controller.addError,
        onDone: () async {
          await controller.close();
        },
        cancelOnError: true,
      );

      _outgoingSub = requests.listen(
        (ConnectRequest request) {
          final envelope = WireEnvelope()
            ..protocolVersion = wireProtocolVersion
            ..sequence = Int64(outgoingSequence)
            ..clientMessage = request;
          outgoingSequence += 1;
          _writeEnvelope(socket, envelope);
        },
        onError: controller.addError,
        onDone: () {
          socket.destroy();
        },
      );
    } catch (error) {
      controller.addError(error);
      await controller.close();
    }
  }

  final _FrameDecoder _frameDecoder = _FrameDecoder();

  void _writeEnvelope(Socket socket, WireEnvelope envelope) {
    final payload = envelope.writeToBuffer();
    final sizeBytes = ByteData(4)..setUint32(0, payload.length, Endian.big);
    socket.add(sizeBytes.buffer.asUint8List());
    socket.add(payload);
  }

  String _newSessionId() {
    final epochMs = DateTime.now().toUtc().millisecondsSinceEpoch;
    return 'tcp-$epochMs';
  }

  @override
  Future<void> shutdown() async {
    await _outgoingSub?.cancel();
    await _incomingSub?.cancel();
    await _socket?.close();
    _outgoingSub = null;
    _incomingSub = null;
    _socket = null;
  }
}

TerminalControlClient createTerminalControlTcpClient({
  required String host,
  required int port,
  String desiredDeviceId = '',
  String resumeToken = '',
  void Function(String token)? onResumeToken,
}) {
  return TerminalControlTcpClient(
    host: host,
    port: port,
    desiredDeviceId: desiredDeviceId,
    resumeToken: resumeToken,
    onResumeToken: onResumeToken,
  );
}

class _FrameDecoder {
  final BytesBuilder _buffer = BytesBuilder(copy: false);
  int _readOffset = 0;

  bool get hasFrame {
    final data = _buffer.toBytes();
    final remaining = data.length - _readOffset;
    if (remaining < 4) {
      return false;
    }
    final frameLength = ByteData.sublistView(
            Uint8List.fromList(data), _readOffset, _readOffset + 4)
        .getUint32(0, Endian.big);
    return remaining >= 4 + frameLength;
  }

  void add(List<int> chunk) {
    if (_readOffset > 0) {
      final data = _buffer.toBytes();
      final unread = data.sublist(_readOffset);
      _buffer.clear();
      _buffer.add(unread);
      _readOffset = 0;
    }
    _buffer.add(chunk);
  }

  List<int> takeFrame() {
    final data = _buffer.toBytes();
    final bytes = Uint8List.fromList(data);
    final length = ByteData.sublistView(bytes, _readOffset, _readOffset + 4)
        .getUint32(0, Endian.big);
    final start = _readOffset + 4;
    final end = start + length;
    final frame = bytes.sublist(start, end);
    _readOffset = end;
    return frame;
  }
}
