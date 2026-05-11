import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

class VideoSurfaceView extends StatefulWidget {
  const VideoSurfaceView({
    super.key,
    required this.streamListenable,
  });

  final ValueListenable<MediaStream?> streamListenable;

  @override
  State<VideoSurfaceView> createState() => _VideoSurfaceViewState();
}

class _VideoSurfaceViewState extends State<VideoSurfaceView> {
  final RTCVideoRenderer _renderer = RTCVideoRenderer();
  MediaStream? _boundStream;
  bool _rendererReady = false;

  @override
  void initState() {
    super.initState();
    widget.streamListenable.addListener(_syncStream);
    unawaited(_initializeRenderer());
  }

  Future<void> _initializeRenderer() async {
    await _renderer.initialize();
    _rendererReady = true;
    await _bind(widget.streamListenable.value);
    if (mounted) {
      setState(() {});
    }
  }

  Future<void> _syncStream() async {
    await _bind(widget.streamListenable.value);
    if (mounted) {
      setState(() {});
    }
  }

  Future<void> _bind(MediaStream? stream) async {
    if (!_rendererReady || identical(stream, _boundStream)) {
      return;
    }
    _boundStream = stream;
    _renderer.srcObject = stream;
  }

  @override
  void didUpdateWidget(covariant VideoSurfaceView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!identical(oldWidget.streamListenable, widget.streamListenable)) {
      oldWidget.streamListenable.removeListener(_syncStream);
      widget.streamListenable.addListener(_syncStream);
      unawaited(_syncStream());
    }
  }

  @override
  void dispose() {
    widget.streamListenable.removeListener(_syncStream);
    unawaited(_renderer.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final hasVideo = _boundStream?.getVideoTracks().isNotEmpty ?? false;
    if (!_rendererReady || !hasVideo) {
      return const Center(
        child: Icon(Icons.videocam_off_outlined),
      );
    }
    return RTCVideoView(
      _renderer,
      objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitContain,
    );
  }
}
