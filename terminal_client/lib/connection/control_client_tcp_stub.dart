import 'control_client.dart';

TerminalControlClient createTerminalControlTcpClient({
  required String host,
  required int port,
}) {
  return UnsupportedTerminalControlClient(
    'TCP control carrier is unavailable on this platform runtime.',
  );
}
