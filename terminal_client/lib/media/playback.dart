import 'dart:typed_data';

import 'package:audioplayers/audioplayers.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/util/speech.dart' as speech;

abstract class AudioPlayback {
  Future<void> play(iov1.PlayAudio playAudio);
  Future<void> dispose();
}

typedef SpeechOutput = void Function(String text);
typedef StopPlayback = Future<void> Function();
typedef PlaySource = Future<void> Function(Source source);
typedef DisposePlayback = Future<void> Function();

class NoopAudioPlayback implements AudioPlayback {
  @override
  Future<void> play(iov1.PlayAudio playAudio) async {}

  @override
  Future<void> dispose() async {}
}

class AudioPlayerPlayback implements AudioPlayback {
  AudioPlayerPlayback({
    SpeechOutput? speechOutput,
    StopPlayback? stopPlayback,
    PlaySource? playSource,
    DisposePlayback? disposePlayback,
  })  : _speechOutput = speechOutput ?? speech.speakText,
        _stopPlayback = stopPlayback,
        _playSource = playSource,
        _disposePlayback = disposePlayback;

  AudioPlayer? _player;
  final SpeechOutput _speechOutput;
  final StopPlayback? _stopPlayback;
  final PlaySource? _playSource;
  final DisposePlayback? _disposePlayback;

  AudioPlayer _resolvedPlayer() {
    return _player ??= AudioPlayer();
  }

  Future<void> _stop() async {
    final stopPlayback = _stopPlayback;
    if (stopPlayback != null) {
      await stopPlayback();
      return;
    }
    await _resolvedPlayer().stop();
  }

  Future<void> _play(Source source) async {
    final playSource = _playSource;
    if (playSource != null) {
      await playSource(source);
      return;
    }
    await _resolvedPlayer().play(source);
  }

  @override
  Future<void> play(iov1.PlayAudio playAudio) async {
    switch (playAudio.whichSource()) {
      case iov1.PlayAudio_Source.url:
        final url = playAudio.url.trim();
        if (url.isNotEmpty) {
          await _stop();
          await _play(UrlSource(url));
        }
        return;
      case iov1.PlayAudio_Source.pcmData:
        if (playAudio.pcmData.isEmpty) {
          return;
        }
        final format = playAudio.format.trim().toLowerCase();
        final payload = format.contains('wav')
            ? Uint8List.fromList(playAudio.pcmData)
            : _pcm16ToWav(
                pcmBytes: Uint8List.fromList(playAudio.pcmData),
                sampleRate: _sampleRateFromFormat(format),
                channels: _channelsFromFormat(format),
              );
        await _stop();
        await _play(BytesSource(payload));
        return;
      case iov1.PlayAudio_Source.ttsText:
        final text = playAudio.ttsText.trim();
        if (text.isNotEmpty) {
          _speechOutput(text);
        }
        return;
      case iov1.PlayAudio_Source.notSet:
        return;
    }
  }

  @override
  Future<void> dispose() async {
    final disposePlayback = _disposePlayback;
    if (disposePlayback != null) {
      await disposePlayback();
      return;
    }
    final player = _player;
    if (player != null) {
      await player.dispose();
    }
  }
}

int _sampleRateFromFormat(String format) {
  final match = RegExp(r'(\d{4,6})').firstMatch(format);
  if (match == null) {
    return 16000;
  }
  return int.tryParse(match.group(1) ?? '') ?? 16000;
}

int _channelsFromFormat(String format) {
  if (format.contains('stereo') || format.contains('2ch')) {
    return 2;
  }
  return 1;
}

Uint8List _pcm16ToWav({
  required Uint8List pcmBytes,
  required int sampleRate,
  required int channels,
}) {
  final byteRate = sampleRate * channels * 2;
  final blockAlign = channels * 2;
  final dataLength = pcmBytes.lengthInBytes;
  final fileSizeMinus8 = 36 + dataLength;

  final out = BytesBuilder(copy: false);
  out.add(_ascii('RIFF'));
  out.add(_le32(fileSizeMinus8));
  out.add(_ascii('WAVE'));
  out.add(_ascii('fmt '));
  out.add(_le32(16));
  out.add(_le16(1));
  out.add(_le16(channels));
  out.add(_le32(sampleRate));
  out.add(_le32(byteRate));
  out.add(_le16(blockAlign));
  out.add(_le16(16));
  out.add(_ascii('data'));
  out.add(_le32(dataLength));
  out.add(pcmBytes);
  return out.toBytes();
}

Uint8List _ascii(String value) => Uint8List.fromList(value.codeUnits);

Uint8List _le16(int value) {
  final out = ByteData(2);
  out.setUint16(0, value, Endian.little);
  return out.buffer.asUint8List();
}

Uint8List _le32(int value) {
  final out = ByteData(4);
  out.setUint32(0, value, Endian.little);
  return out.buffer.asUint8List();
}
