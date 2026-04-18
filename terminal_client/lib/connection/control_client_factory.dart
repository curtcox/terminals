import 'package:flutter/foundation.dart';

import 'control_client.dart';
import 'control_client_http_stub.dart'
    if (dart.library.io) 'control_client_http_io.dart' as http_client;
import 'control_client_tcp_stub.dart'
    if (dart.library.io) 'control_client_tcp_io.dart' as tcp_client;
import 'control_client_ws.dart';

enum ControlCarrierKind {
  grpc,
  websocket,
  tcp,
  http,
}

class ControlClientTransportHint {
  static ControlCarrierKind preferredCarrier = ControlCarrierKind.grpc;
  static String websocketPath = '/control';
  static String? tcpEndpoint;
  static String? httpEndpoint;

  static void configure({
    required ControlCarrierKind carrier,
    String wsPath = '/control',
    String? tcp,
    String? http,
  }) {
    preferredCarrier = carrier;
    websocketPath = wsPath;
    tcpEndpoint = tcp;
    httpEndpoint = http;
  }
}

/// Creates the correct control transport for the current platform.
TerminalControlClient createTerminalControlClient({
  required String host,
  required int port,
  ControlCarrierKind? preferredCarrier,
  String? websocketPath,
  String? tcpEndpoint,
  String? httpEndpoint,
}) {
  final preferred =
      preferredCarrier ?? ControlClientTransportHint.preferredCarrier;
  final wsPath = websocketPath ?? ControlClientTransportHint.websocketPath;
  final tcpHint = tcpEndpoint ?? ControlClientTransportHint.tcpEndpoint;
  final httpHint = httpEndpoint ?? ControlClientTransportHint.httpEndpoint;
  if (preferred == ControlCarrierKind.websocket || kIsWeb) {
    return TerminalControlWebSocketClient(
      host: host,
      port: port,
      path: wsPath,
      secure: Uri.base.scheme == 'https',
    );
  }
  if (preferred == ControlCarrierKind.tcp) {
    final endpoint = _parseHostPort(tcpHint, host, 50055);
    return tcp_client.createTerminalControlTcpClient(
      host: endpoint.$1,
      port: endpoint.$2,
    );
  }
  if (preferred == ControlCarrierKind.http) {
    final uri = _parseHttpEndpoint(httpHint, host);
    return http_client.createTerminalControlHttpClient(baseUri: uri);
  }
  return TerminalControlGrpcClient(host: host, port: port);
}

(String, int) _parseHostPort(
  String? endpoint,
  String fallbackHost,
  int fallbackPort,
) {
  final raw = (endpoint ?? '').trim();
  if (raw.isEmpty) {
    return (fallbackHost, fallbackPort);
  }
  final uri = Uri.tryParse('tcp://$raw');
  if (uri == null || uri.host.isEmpty || uri.port <= 0) {
    return (fallbackHost, fallbackPort);
  }
  return (uri.host, uri.port);
}

Uri _parseHttpEndpoint(String? endpoint, String fallbackHost) {
  final raw = (endpoint ?? '').trim();
  if (raw.isNotEmpty) {
    final parsed = Uri.tryParse(raw);
    if (parsed != null && parsed.hasScheme && parsed.host.isNotEmpty) {
      return parsed;
    }
  }
  return Uri.parse('http://$fallbackHost:50056');
}
