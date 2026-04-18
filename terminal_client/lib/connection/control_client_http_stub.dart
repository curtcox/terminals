import 'control_client.dart';

TerminalControlClient createTerminalControlHttpClient({
  required Uri baseUri,
  String desiredDeviceId = '',
  String resumeToken = '',
  void Function(String token)? onResumeToken,
}) {
  return UnsupportedTerminalControlClient(
    'HTTP fallback control carrier is unavailable on this platform runtime.',
  );
}
