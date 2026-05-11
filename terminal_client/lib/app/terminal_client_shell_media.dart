part of 'terminal_client_shell.dart';

extension _MediaExtension on _TerminalClientShellState {
  void _applyMediaControlResponse(ConnectResponse response) {
    final synchronousUpdate =
        synchronousMediaControlUpdateFromResponse(response);
    if (response.hasStartStream()) {
      final start = response.startStream;
      if (synchronousUpdate.shouldAcknowledgeStartStream) {
        _activeStreamsByID[synchronousUpdate.startStreamID] = start.deepCopy();
        unawaited(
          _sendWhenReady(
            operation: OutboundOperation.streamReady,
            request: ConnectRequest()
              ..streamReady =
                  (StreamReady()..streamId = synchronousUpdate.startStreamID),
          ),
        );
        _streamReadyAckCount += 1;
        unawaited(_startMediaStream(start.deepCopy()));
      }
      if (synchronousUpdate.startStreamNotification.isNotEmpty) {
        _lastNotification = synchronousUpdate.startStreamNotification;
      }
    }
    if (response.hasStopStream()) {
      final streamID = synchronousUpdate.stopStreamID;
      if (streamID.isNotEmpty) {
        _activeStreamsByID.remove(streamID);
        _routesByStreamID.remove(streamID);
        unawaited(_mediaEngine.stopStream(streamID));
        _lastNotification = synchronousUpdate.stopStreamNotification;
      }
    }
    if (response.hasRouteStream()) {
      final route = response.routeStream;
      if (synchronousUpdate.routeStreamID.isNotEmpty) {
        _routesByStreamID[synchronousUpdate.routeStreamID] = route.deepCopy();
      }
      _lastNotification = synchronousUpdate.routeNotification;
    }
    if (response.hasWebrtcSignal()) {
      _recentWebRTCSignals.add(response.webrtcSignal.deepCopy());
      const maxSignals = 50;
      if (_recentWebRTCSignals.length > maxSignals) {
        _recentWebRTCSignals.removeRange(
          0,
          _recentWebRTCSignals.length - maxSignals,
        );
      }
      _lastNotification = synchronousUpdate.webrtcSignalNotification;
      unawaited(_mediaEngine.handleSignal(response.webrtcSignal.deepCopy()));
    }
    if (response.hasPlayAudio()) {
      unawaited(_executePlayAudio(response.playAudio.deepCopy()));
    }
    if (response.hasInstallBundle()) {
      unawaited(_handleInstallBundle(response.installBundle.deepCopy()));
    }
    if (response.hasRemoveBundle()) {
      unawaited(_handleRemoveBundle(response.removeBundle.deepCopy()));
    }
    if (response.hasStartFlow()) {
      unawaited(_handleStartFlow(response.startFlow.deepCopy()));
    }
    if (response.hasPatchFlow()) {
      unawaited(_handlePatchFlow(response.patchFlow.deepCopy()));
    }
    if (response.hasStopFlow()) {
      unawaited(_handleStopFlow(response.stopFlow.deepCopy()));
    }
    if (response.hasRequestArtifact()) {
      unawaited(_handleRequestArtifact(response.requestArtifact.deepCopy()));
    }
  }

  Future<void> _handleInstallBundle(iov1.InstallBundle request) async {
    final bundleID = request.bundleId.trim();
    if (bundleID.isEmpty) {
      return;
    }
    try {
      final host = await _edgeHostFuture;
      await host.installBundle(bundleID, request.tarGz);
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Bundle installed: $bundleID';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Bundle install failed: $bundleID ($error)';
      });
    }
  }

  Future<void> _handleRemoveBundle(iov1.RemoveBundle request) async {
    final bundleID = request.bundleId.trim();
    if (bundleID.isEmpty) {
      return;
    }
    try {
      final host = await _edgeHostFuture;
      await host.removeBundle(bundleID);
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Bundle removed: $bundleID';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Bundle removal failed: $bundleID ($error)';
      });
    }
  }

  Future<void> _handleStartFlow(iov1.StartFlow request) async {
    final flowID = request.flowId.trim();
    if (flowID.isEmpty) {
      return;
    }
    final bundleID = bundleIDFromFlowPlan(request.plan);
    try {
      final host = await _edgeHostFuture;
      await host.startFlow(flowID, bundleId: bundleID);
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow started: $flowID';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow start failed: $flowID ($error)';
      });
    }
  }

  Future<void> _handlePatchFlow(iov1.PatchFlow request) async {
    final flowID = request.flowId.trim();
    if (flowID.isEmpty) {
      return;
    }
    final bundleID = bundleIDFromFlowPlan(request.plan);
    try {
      final host = await _edgeHostFuture;
      await host.patchFlow(flowID, bundleId: bundleID);
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow patched: $flowID';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow patch failed: $flowID ($error)';
      });
    }
  }

  Future<void> _handleStopFlow(iov1.StopFlow request) async {
    final flowID = request.flowId.trim();
    if (flowID.isEmpty) {
      return;
    }
    try {
      final host = await _edgeHostFuture;
      await host.stopFlow(flowID);
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow stopped: $flowID';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Flow stop failed: $flowID ($error)';
      });
    }
  }

  Future<void> _startMediaStream(iov1.StartStream start) async {
    try {
      await _ensureMediaPermissionsForStart(start);
      await _mediaEngine.startStream(start);
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _status = 'Media permission required';
        _lastConnectionStatus = _status;
        _lastNotification =
            'Unable to start media stream ${start.streamId}: $error';
      });
    }
  }

  Future<void> _ensureMediaPermissionsForStart(iov1.StartStream start) async {
    final sourceDevice = start.sourceDeviceId.trim();
    if (sourceDevice != _deviceId) {
      return;
    }
    final wantsAudio = start.kind.toLowerCase().contains('audio');
    final wantsVideo = start.kind.toLowerCase().contains('video');
    if (!wantsAudio && !wantsVideo) {
      return;
    }
    await widget.mediaPermissionProbe(
      audio: wantsAudio,
      video: wantsVideo,
    );
  }

  Future<void> _executePlayAudio(iov1.PlayAudio playAudio) async {
    final source = playAudioSourceLabel(playAudio);
    final bytes = playAudioPcmByteCount(playAudio);
    try {
      await _audioPlayback.play(playAudio);
      if (playAudio.whichSource() == iov1.PlayAudio_Source.pcmData &&
          playAudio.pcmData.isNotEmpty &&
          playAudio.requestId.trim().isNotEmpty) {
        await _artifactExporter.save(
          'play_audio/${playAudio.requestId.trim()}',
          Uint8List.fromList(playAudio.pcmData),
        );
      }
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _playAudioCount += 1;
        _lastPlayAudioBytes = bytes;
        _lastPlayAudioDeviceID =
            playAudio.deviceId.isNotEmpty ? playAudio.deviceId : 'unknown';
        _lastPlayAudioSource = source;
        _lastNotification =
            'Play audio: $_lastPlayAudioDeviceID ($source, $bytes bytes)';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _lastNotification = 'Play audio failed: $error';
      });
    }
  }

  Future<void> _handleRequestArtifact(iov1.RequestArtifact request) async {
    final artifactID = request.artifactId.trim();
    if (artifactID.isEmpty) {
      return;
    }
    try {
      final payload = await _artifactExporter.exportByID(artifactID);
      final nowUnixMs = _nowUnixMs();
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.artifactAvailable,
          request: ConnectRequest()
            ..artifactAvailable = (iov1.ArtifactAvailable()
              ..artifact = (iov1.ArtifactRef()
                ..id = artifactID
                ..kind = 'artifact.binary'
                ..source = (iov1.DeviceRef()..deviceId = _deviceId)
                ..startUnixMs = Int64(nowUnixMs)
                ..endUnixMs = Int64(nowUnixMs)
                ..uri =
                    'local://artifact/$artifactID?bytes=${payload.length}')),
        ),
      );
      if (mounted) {
        _rebuildState(() {
          _lastNotification = 'Artifact available: $artifactID';
        });
      }
    } catch (error) {
      if (mounted) {
        _rebuildState(() {
          _lastNotification = 'Artifact request failed: $artifactID ($error)';
        });
      }
    }
  }

  void _sendWebRTCSignalMessage(WebRTCSignal signal) {
    final streamID = signal.streamId.trim();
    final signalType = signal.signalType.trim();
    if (streamID.isEmpty || signalType.isEmpty) {
      return;
    }
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.webrtcSignal,
        request: ConnectRequest()..webrtcSignal = signal.deepCopy(),
      ),
    );
  }

  Future<EdgeHost> _createEdgeHost() async {
    final bundleStore = await BundleStore.create();
    return EdgeHost.create(
      bundleStore: bundleStore,
      scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 512),
      retention: RetentionBufferManager(
        audioSec: 20,
        videoSec: 10,
        sensorSec: 600,
        radioSec: 300,
      ),
    );
  }
}
