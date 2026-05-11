part of 'terminal_client_shell.dart';

extension _CapabilitiesExtension on _TerminalClientShellState {
  Future<void> _probeAndPublishCapabilityChanges({
    required String reason,
    bool forceSnapshot = false,
  }) async {
    if (!_hasActiveControlSession ||
        !_isConnectionRegistered ||
        _deviceId.isEmpty ||
        _capabilityPollInFlight) {
      return;
    }
    _capabilityPollInFlight = true;
    try {
      final probedCapabilities = await _capabilityProbe.probe(
        CapabilityProbeContext(
          deviceId: _deviceId,
          deviceName: _deviceNameController.text.trim(),
          deviceType: _deviceTypeController.text.trim(),
          platform: _platformController.text.trim(),
          screenWidth: _currentLogicalSize().width.round(),
          screenHeight: _currentLogicalSize().height.round(),
          screenDensity: _currentDevicePixelRatio(),
          targetPlatform: defaultTargetPlatform,
        ),
      );
      final nextCapabilities = _applyDisplayMetadata(
        probedCapabilities.deepCopy(),
      );
      if (_privacyModeEnabled) {
        nextCapabilities
          ..clearMicrophone()
          ..clearCamera();
      }
      _syncWakeWordDetector(nextCapabilities);
      final publication = _capabilitySession.publishChange(
        nextCapabilities,
        force: forceSnapshot,
      );
      if (publication == null) {
        return;
      }

      if (forceSnapshot) {
        unawaited(
          _sendWhenReady(
            operation: OutboundOperation.capabilitySnapshot,
            request: TerminalControlGrpcClient.capabilitySnapshotRequest(
              deviceId: _deviceId,
              generation: publication.generation,
              capabilities: publication.capabilities,
            ),
          ),
        );
        return;
      }
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.capabilityDelta,
          request: TerminalControlGrpcClient.capabilityDeltaRequest(
            deviceId: _deviceId,
            generation: publication.generation,
            capabilities: publication.capabilities,
            reason: reason,
          ),
        ),
      );
    } finally {
      _capabilityPollInFlight = false;
    }
  }

  Future<void> _handlePrivacyToggleAction() async {
    if (_privacyModeEnabled) {
      _privacyModeEnabled = false;
      await _probeAndPublishCapabilityChanges(reason: 'privacy.toggle');
      return;
    }
    await _stopLocalCaptureStreamsForPrivacyMode();
    final registeredCapabilities =
        _capabilitySession.lastRegisteredCapabilities;
    if (registeredCapabilities == null || _deviceId.isEmpty) {
      _privacyModeEnabled = true;
      _setWakeWordDetectorEnabled(false);
      return;
    }
    final nextCapabilities = registeredCapabilities.deepCopy()
      ..clearMicrophone()
      ..clearCamera();
    _privacyModeEnabled = true;
    _syncWakeWordDetector(nextCapabilities);
    final publication = _capabilitySession.publishChange(
      nextCapabilities,
      force: true,
    );
    if (publication == null) {
      return;
    }
    await _sendWhenReady(
      operation: OutboundOperation.capabilityDelta,
      request: TerminalControlGrpcClient.capabilityDeltaRequest(
        deviceId: _deviceId,
        generation: publication.generation,
        capabilities: publication.capabilities,
        reason: 'privacy.toggle',
      ),
    );
  }

  Future<void> _stopLocalCaptureStreamsForPrivacyMode() async {
    final streamIDs = _activeStreamsByID.entries
        .where((entry) {
          final sourceDeviceID = entry.value.sourceDeviceId.trim();
          if (sourceDeviceID != _deviceId) {
            return false;
          }
          final kind = entry.value.kind.trim().toLowerCase();
          return kind.contains('audio') || kind.contains('video');
        })
        .map((entry) => entry.key)
        .toList(growable: false);
    for (final streamID in streamIDs) {
      await _mediaEngine.stopStream(streamID);
      _activeStreamsByID.remove(streamID);
      _routesByStreamID.remove(streamID);
    }
  }

  void _syncWakeWordDetector(capv1.DeviceCapabilities capabilities) {
    _setWakeWordDetectorEnabled(
      capabilities.hasMicrophone() && !_privacyModeEnabled,
    );
  }

  void _setWakeWordDetectorEnabled(bool enabled) {
    if (_wakeWordDetectorEnabled == enabled) {
      return;
    }
    _wakeWordDetectorEnabled = enabled;
    unawaited(_wakeWordDetector.setEnabled(enabled));
  }

  bool _canEmitVoiceAudio() {
    if (_privacyModeEnabled || !_isConnectionRegistered) {
      return false;
    }
    final capabilities = _capabilitySession.lastRegisteredCapabilities;
    return capabilities != null && capabilities.hasMicrophone();
  }

  void _handleWakeWordUtterance(WakeWordUtterance utterance) {
    if (!_canEmitVoiceAudio()) {
      return;
    }
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.voiceAudio,
        request: ConnectRequest()
          ..voiceAudio = (VoiceAudio()
            ..deviceId = _deviceId
            ..audio = utterance.audio
            ..sampleRate = utterance.sampleRate
            ..isFinal = utterance.isFinal),
      ),
    );
  }

  bool _isStaleCapabilityGenerationError(ControlError error) {
    return isStaleCapabilityGenerationError(error);
  }

  void _sendLifecycleCapabilityUpdate({String reason = 'lifecycle_state'}) {
    if (!_hasActiveControlSession) {
      return;
    }
    unawaited(_probeAndPublishCapabilityChanges(reason: reason));
  }

  ConnectRequest? _buildBootstrapCapabilitySnapshotRequest() {
    final capabilities = _capabilitySession.lastRegisteredCapabilities;
    if (capabilities == null ||
        _deviceId.isEmpty ||
        _capabilitySession.generation <= 0) {
      return null;
    }
    return TerminalControlGrpcClient.capabilitySnapshotRequest(
      deviceId: _deviceId,
      generation: _capabilitySession.generation,
      capabilities: capabilities,
    );
  }
}
