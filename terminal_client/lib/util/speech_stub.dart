import 'package:flutter_tts/flutter_tts.dart';

final FlutterTts _tts = FlutterTts();

void speakText(String text) {
  final trimmed = text.trim();
  if (trimmed.isEmpty) {
    return;
  }
  _speakSafely(trimmed);
}

Future<void> _speakSafely(String text) async {
  try {
    await _tts.stop();
    await _tts.speak(text);
  } catch (_) {
    // Best-effort speech output on non-web platforms.
  }
}
