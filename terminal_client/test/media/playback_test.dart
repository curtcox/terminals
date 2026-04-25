import 'dart:convert';

import 'package:audioplayers/audioplayers.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/media/playback.dart';

void main() {
  test('url source trims and plays URL audio', () async {
    final playedSources = <Source>[];
    var stopCalls = 0;
    final playback = AudioPlayerPlayback(
      stopPlayback: () async {
        stopCalls += 1;
      },
      playSource: (source) async {
        playedSources.add(source);
      },
    );

    await playback
        .play(iov1.PlayAudio()..url = '  https://example.test/a.mp3 ');

    expect(stopCalls, 1);
    expect(playedSources, hasLength(1));
    final source = playedSources.single;
    expect(source, isA<UrlSource>());
    expect((source as UrlSource).url, 'https://example.test/a.mp3');
  });

  test('pcm_data source wraps raw pcm16 bytes in a wav container', () async {
    final playedSources = <Source>[];
    final playback = AudioPlayerPlayback(
      stopPlayback: () async {},
      playSource: (source) async {
        playedSources.add(source);
      },
    );

    await playback.play(
      iov1.PlayAudio()
        ..pcmData = <int>[1, 2, 3, 4]
        ..format = 'pcm_s16le_16000_mono',
    );

    expect(playedSources, hasLength(1));
    final source = playedSources.single;
    expect(source, isA<BytesSource>());
    final bytes = (source as BytesSource).bytes;
    expect(ascii.decode(bytes.sublist(0, 4)), 'RIFF');
    expect(ascii.decode(bytes.sublist(8, 12)), 'WAVE');
    expect(bytes.sublist(bytes.length - 4), <int>[1, 2, 3, 4]);
  });

  test('pcm_data source preserves payload when format already wav', () async {
    final playedSources = <Source>[];
    final playback = AudioPlayerPlayback(
      stopPlayback: () async {},
      playSource: (source) async {
        playedSources.add(source);
      },
    );

    await playback.play(
      iov1.PlayAudio()
        ..pcmData = <int>[10, 20, 30, 40]
        ..format = 'audio/wav',
    );

    expect(playedSources, hasLength(1));
    final source = playedSources.single;
    expect(source, isA<BytesSource>());
    expect((source as BytesSource).bytes, <int>[10, 20, 30, 40]);
  });

  test('tts_text source speaks trimmed text without player operations',
      () async {
    final spoken = <String>[];
    var stopCalls = 0;
    var playCalls = 0;
    final playback = AudioPlayerPlayback(
      speechOutput: spoken.add,
      stopPlayback: () async {
        stopCalls += 1;
      },
      playSource: (source) async {
        playCalls += 1;
      },
    );

    await playback.play(iov1.PlayAudio()..ttsText = '  attention team  ');

    expect(spoken, <String>['attention team']);
    expect(stopCalls, 0);
    expect(playCalls, 0);
  });
}
