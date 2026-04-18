import 'control_client.dart';

TerminalControlClient createTerminalControlTcpClient({
  required String host,
  required int port,
  String desiredDeviceId = '',
  String resumeToken = '',
  void Function(String token)? onResumeToken,
}) {
  return UnsupportedTerminalControlClient(
    'TCP control carrier is unavailable on this platform runtime.',
  );
}
