// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:html' as html;

void speakText(String text) {
  final trimmed = text.trim();
  if (trimmed.isEmpty) {
    return;
  }
  final synth = html.window.speechSynthesis;
  if (synth == null) {
    return;
  }
  synth.cancel();
  synth.speak(html.SpeechSynthesisUtterance(trimmed));
}
