import 'control_client.dart';

TerminalControlClient createTerminalControlHttpClient({
  required Uri baseUri,
}) {
  return UnsupportedTerminalControlClient(
    'HTTP fallback control carrier is unavailable on this platform runtime.',
  );
}
