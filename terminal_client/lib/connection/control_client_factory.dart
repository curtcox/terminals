import 'package:flutter/foundation.dart';

import 'control_client.dart';
import 'control_client_ws.dart';

/// Creates the correct control transport for the current platform.
TerminalControlClient createTerminalControlClient({
  required String host,
  required int port,
}) {
  if (kIsWeb) {
    return TerminalControlWebSocketClient(
      host: host,
      port: port,
      secure: Uri.base.scheme == 'https',
    );
  }
  return TerminalControlGrpcClient(host: host, port: port);
}
