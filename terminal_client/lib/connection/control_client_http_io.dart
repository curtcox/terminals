import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:fixnum/fixnum.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

import 'control_client.dart';

class TerminalControlHttpClient implements TerminalControlClient {
  TerminalControlHttpClient({required this.baseUri});

  final Uri baseUri;
  final HttpClient _http = HttpClient();

  StreamSubscription<ConnectRequest>? _outgoingSub;
  Timer? _pollTimer;
  bool _closed = false;
  int _outgoingSequence = 1;
  String? _sessionId;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    final controller = StreamController<ConnectResponse>();
    _start(requests, controller);
    controller.onCancel = () async {
      await shutdown();
    };
    return controller.stream;
  }

  Future<void> _start(
    Stream<ConnectRequest> requests,
    StreamController<ConnectResponse> controller,
  ) async {
    _sessionId = _newSessionId();
    try {
      final hello = WireEnvelope()
        ..protocolVersion = wireProtocolVersion
        ..sessionId = _sessionId!
        ..transportHello = (TransportHello()
          ..protocolVersion = wireProtocolVersion
          ..supportedCarriers.add(CarrierKind.CARRIER_KIND_HTTP));
      await _postPoll(hello);
    } catch (error) {
      controller.addError(error);
      await controller.close();
      return;
    }

    _outgoingSub = requests.listen(
      (ConnectRequest request) async {
        if (_closed) {
          return;
        }
        final envelope = WireEnvelope()
          ..protocolVersion = wireProtocolVersion
          ..sessionId = _sessionId!
          ..sequence = Int64(_outgoingSequence)
          ..clientMessage = request;
        _outgoingSequence += 1;
        try {
          await _postPoll(envelope);
        } catch (error) {
          controller.addError(error);
        }
      },
      onError: controller.addError,
      onDone: () async {
        await controller.close();
      },
    );

    _pollTimer = Timer.periodic(const Duration(milliseconds: 200), (_) {
      unawaited(_pollStream(controller));
    });
  }

  Future<void> _pollStream(StreamController<ConnectResponse> controller) async {
    if (_closed || _sessionId == null) {
      return;
    }
    final uri = baseUri.resolve('/v1/control/stream/$_sessionId?wait_ms=20000');
    final request = await _http.getUrl(uri);
    final response = await request.close();
    if (response.statusCode == HttpStatus.noContent) {
      await response.drain<void>();
      return;
    }
    if (response.statusCode != HttpStatus.ok) {
      final body = await utf8.decoder.bind(response).join();
      throw StateError(
          'http control stream failed (${response.statusCode}): $body');
    }
    final payload = await response.fold<List<int>>(<int>[], (acc, chunk) {
      acc.addAll(chunk);
      return acc;
    });
    final envelope = WireEnvelope.fromBuffer(payload);
    if (envelope.hasServerMessage()) {
      controller.add(envelope.serverMessage);
    }
  }

  Future<void> _postPoll(WireEnvelope envelope) async {
    if (_sessionId == null) {
      throw StateError('missing http control session id');
    }
    final uri = baseUri.resolve('/v1/control/poll/$_sessionId');
    final request = await _http.postUrl(uri);
    request.headers.contentType = ContentType('application', 'x-protobuf');
    request.add(envelope.writeToBuffer());
    final response = await request.close();
    if (response.statusCode != HttpStatus.accepted &&
        response.statusCode != HttpStatus.ok) {
      final body = await utf8.decoder.bind(response).join();
      throw StateError(
          'http control poll failed (${response.statusCode}): $body');
    }
    await response.drain<void>();
  }

  String _newSessionId() {
    final epochMs = DateTime.now().toUtc().millisecondsSinceEpoch;
    return 'http-$epochMs';
  }

  @override
  Future<void> shutdown() async {
    _closed = true;
    _pollTimer?.cancel();
    await _outgoingSub?.cancel();
    _http.close(force: true);
  }
}

TerminalControlClient createTerminalControlHttpClient({
  required Uri baseUri,
}) {
  return TerminalControlHttpClient(baseUri: baseUri);
}
