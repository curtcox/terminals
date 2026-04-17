import 'dart:typed_data';

import 'package:audioplayers/audioplayers.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/util/speech.dart' as speech;

abstract class AudioPlayback {
  Future<void> play(iov1.PlayAudio playAudio);
  Future<void> dispose();
}

class NoopAudioPlayback implements AudioPlayback {
  @override
  Future<void> play(iov1.PlayAudio playAudio) async {}

  @override
  Future<void> dispose() async {}
}

class AudioPlayerPlayback implements AudioPlayback {
  AudioPlayerPlayback() : _player = AudioPlayer();

  final AudioPlayer _player;

  @override
  Future<void> play(iov1.PlayAudio playAudio) async {
    switch (playAudio.whichSource()) {
      case iov1.PlayAudio_Source.url:
        final url = playAudio.url.trim();
        if (url.isNotEmpty) {
          await _player.stop();
          await _player.play(UrlSource(url));
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
        await _player.stop();
        await _player.play(BytesSource(payload));
        return;
      case iov1.PlayAudio_Source.ttsText:
        final text = playAudio.ttsText.trim();
        if (text.isNotEmpty) {
          speech.speakText(text);
        }
        return;
      case iov1.PlayAudio_Source.notSet:
        return;
    }
  }

  @override
  Future<void> dispose() async {
    await _player.dispose();
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
