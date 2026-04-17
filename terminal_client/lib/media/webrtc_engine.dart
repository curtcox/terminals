import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;

typedef OutboundSignalCallback = void Function(WebRTCSignal signal);

/// Client-side WebRTC control contract used by the control stream scaffold.
abstract class ClientMediaEngine {
  Future<void> startStream(iov1.StartStream start);
  Future<void> stopStream(String streamID);
  Future<void> handleSignal(WebRTCSignal signal);
  ValueListenable<MediaStream?> remoteStream(String streamID);
  ValueListenable<double> audioLevel(String streamID);
  Future<void> dispose();
}

ClientMediaEngine defaultClientMediaEngineFactory({
  required String localDeviceID,
  required OutboundSignalCallback onSignal,
}) {
  return FlutterWebRTCMediaEngine(
    localDeviceID: localDeviceID,
    onSignal: onSignal,
  );
}

class FlutterWebRTCMediaEngine implements ClientMediaEngine {
  FlutterWebRTCMediaEngine({
    required this.localDeviceID,
    required this.onSignal,
  });

  final String localDeviceID;
  final OutboundSignalCallback onSignal;
  final Map<String, _WebRTCSession> _sessions = <String, _WebRTCSession>{};
  final Map<String, ValueNotifier<MediaStream?>> _remoteStreamsByID =
      <String, ValueNotifier<MediaStream?>>{};
  final Map<String, ValueNotifier<double>> _audioLevelsByID =
      <String, ValueNotifier<double>>{};

  @override
  Future<void> startStream(iov1.StartStream start) async {
    final streamID = start.streamId.trim();
    if (streamID.isEmpty) {
      return;
    }
    if (_sessions.containsKey(streamID)) {
      return;
    }

    final sendLocalMedia = start.sourceDeviceId.trim() == localDeviceID;
    final wantsAudio = _kindHas(start.kind, 'audio');
    final wantsVideo = _kindHas(start.kind, 'video');
    final peerConnection = await createPeerConnection(<String, dynamic>{
      'iceServers': <Map<String, dynamic>>[
        <String, dynamic>{'urls': 'stun:stun.l.google.com:19302'},
      ],
    });

    final session = _WebRTCSession(
      streamID: streamID,
      peerConnection: peerConnection,
      wantsAudio: wantsAudio,
      wantsVideo: wantsVideo,
      remoteStream: _ensureRemoteStreamNotifier(streamID),
      audioLevel: _ensureAudioLevelNotifier(streamID),
    );
    _sessions[streamID] = session;

    peerConnection.onIceCandidate = (RTCIceCandidate candidate) {
      final payload = _encodeCandidate(candidate);
      if (payload.isEmpty) {
        return;
      }
      _emitSignal(streamID, 'candidate', payload);
    };
    peerConnection.onTrack = (RTCTrackEvent event) {
      final streams = event.streams;
      if (streams.isEmpty) {
        return;
      }
      session.remoteStream.value = streams.first;
      session.audioLevel.value =
          streams.first.getAudioTracks().isNotEmpty ? 1.0 : 0.0;
    };

    if (sendLocalMedia && (wantsAudio || wantsVideo)) {
      final mediaStream = await navigator.mediaDevices.getUserMedia(
        <String, dynamic>{
          'audio': wantsAudio,
          'video': wantsVideo,
        },
      );
      session.localStream = mediaStream;
      for (final track in mediaStream.getTracks()) {
        await peerConnection.addTrack(track, mediaStream);
      }
    }

    final offer = await peerConnection.createOffer(
      _sdpConstraints(wantsAudio: wantsAudio, wantsVideo: wantsVideo),
    );
    await peerConnection.setLocalDescription(offer);
    if ((offer.sdp ?? '').trim().isNotEmpty) {
      _emitSignal(streamID, 'offer', _encodeSDP(offer.sdp!));
    }
  }

  @override
  Future<void> stopStream(String streamID) async {
    final normalized = streamID.trim();
    final session = _sessions.remove(normalized);
    if (session == null) {
      return;
    }
    session.remoteStream.value = null;
    session.audioLevel.value = 0.0;
    await session.dispose();
  }

  @override
  Future<void> handleSignal(WebRTCSignal signal) async {
    final streamID = signal.streamId.trim();
    if (streamID.isEmpty) {
      return;
    }
    final session = _sessions[streamID];
    if (session == null) {
      return;
    }

    final signalType = signal.signalType.trim().toLowerCase();
    switch (signalType) {
      case 'offer':
        final remoteOffer = _decodeSessionDescription(
          signal.payload,
          'offer',
        );
        if (remoteOffer == null) {
          return;
        }
        await session.peerConnection.setRemoteDescription(remoteOffer);
        final answer = await session.peerConnection.createAnswer(
          _sdpConstraints(
            wantsAudio: session.wantsAudio,
            wantsVideo: session.wantsVideo,
          ),
        );
        await session.peerConnection.setLocalDescription(answer);
        if ((answer.sdp ?? '').trim().isNotEmpty) {
          _emitSignal(streamID, 'answer', _encodeSDP(answer.sdp!));
        }
        break;
      case 'answer':
        final remoteAnswer = _decodeSessionDescription(
          signal.payload,
          'answer',
        );
        if (remoteAnswer == null) {
          return;
        }
        await session.peerConnection.setRemoteDescription(remoteAnswer);
        break;
      case 'candidate':
        final candidate = _decodeCandidate(signal.payload);
        if (candidate == null) {
          return;
        }
        await session.peerConnection.addCandidate(candidate);
        break;
      default:
        return;
    }
  }

  @override
  Future<void> dispose() async {
    final sessions = _sessions.values.toList(growable: false);
    _sessions.clear();
    for (final session in sessions) {
      await session.dispose();
    }
    for (final notifier in _remoteStreamsByID.values) {
      notifier.value = null;
    }
    for (final notifier in _audioLevelsByID.values) {
      notifier.value = 0.0;
    }
  }

  @override
  ValueListenable<MediaStream?> remoteStream(String streamID) {
    return _ensureRemoteStreamNotifier(streamID.trim());
  }

  @override
  ValueListenable<double> audioLevel(String streamID) {
    return _ensureAudioLevelNotifier(streamID.trim());
  }

  void _emitSignal(String streamID, String signalType, String payload) {
    onSignal(
      WebRTCSignal()
        ..streamId = streamID
        ..signalType = signalType
        ..payload = payload,
    );
  }

  ValueNotifier<MediaStream?> _ensureRemoteStreamNotifier(String streamID) {
    return _remoteStreamsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<MediaStream?>(null),
    );
  }

  ValueNotifier<double> _ensureAudioLevelNotifier(String streamID) {
    return _audioLevelsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<double>(0.0),
    );
  }
}

class _WebRTCSession {
  _WebRTCSession({
    required this.streamID,
    required this.peerConnection,
    required this.wantsAudio,
    required this.wantsVideo,
    required this.remoteStream,
    required this.audioLevel,
  });

  final String streamID;
  final RTCPeerConnection peerConnection;
  final bool wantsAudio;
  final bool wantsVideo;
  final ValueNotifier<MediaStream?> remoteStream;
  final ValueNotifier<double> audioLevel;
  MediaStream? localStream;

  Future<void> dispose() async {
    final stream = localStream;
    localStream = null;
    if (stream != null) {
      for (final track in stream.getTracks()) {
        track.stop();
      }
      await stream.dispose();
    }
    await peerConnection.close();
  }
}

bool _kindHas(String kind, String token) {
  final normalized = kind.trim().toLowerCase();
  if (normalized.isEmpty) {
    return true;
  }
  return normalized.contains(token);
}

String _encodeSDP(String sdp) {
  return jsonEncode(<String, String>{'sdp': sdp});
}

String _encodeCandidate(RTCIceCandidate candidate) {
  final candidateValue = candidate.candidate?.trim() ?? '';
  if (candidateValue.isEmpty) {
    return '';
  }
  return jsonEncode(<String, dynamic>{
    'candidate': candidateValue,
    'sdpMid': candidate.sdpMid,
    'sdpMLineIndex': candidate.sdpMLineIndex,
  });
}

RTCSessionDescription? _decodeSessionDescription(
  String raw,
  String type,
) {
  final payload = raw.trim();
  if (payload.isEmpty) {
    return null;
  }
  var sdp = payload;
  if (payload.startsWith('{')) {
    try {
      final decoded = jsonDecode(payload);
      if (decoded is Map<String, dynamic>) {
        final value = decoded['sdp'];
        if (value is String && value.trim().isNotEmpty) {
          sdp = value;
        } else {
          return null;
        }
      }
    } catch (_) {
      return null;
    }
  }
  return RTCSessionDescription(sdp, type);
}

Map<String, dynamic> _sdpConstraints({
  required bool wantsAudio,
  required bool wantsVideo,
}) {
  return <String, dynamic>{
    'mandatory': <String, dynamic>{
      'OfferToReceiveAudio': wantsAudio,
      'OfferToReceiveVideo': wantsVideo,
    },
    'optional': <dynamic>[],
  };
}

RTCIceCandidate? _decodeCandidate(String raw) {
  final payload = raw.trim();
  if (payload.isEmpty) {
    return null;
  }
  if (!payload.startsWith('{')) {
    return RTCIceCandidate(payload, null, null);
  }
  try {
    final decoded = jsonDecode(payload);
    if (decoded is! Map<String, dynamic>) {
      return null;
    }
    final candidate = decoded['candidate'];
    if (candidate is! String || candidate.trim().isEmpty) {
      return null;
    }
    final sdpMid = decoded['sdpMid'];
    final sdpMLineIndex = decoded['sdpMLineIndex'];
    return RTCIceCandidate(
      candidate,
      sdpMid is String ? sdpMid : null,
      sdpMLineIndex is int ? sdpMLineIndex : null,
    );
  } catch (_) {
    return null;
  }
}
