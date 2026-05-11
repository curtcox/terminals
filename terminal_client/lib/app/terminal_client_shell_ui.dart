part of 'terminal_client_shell.dart';

extension _UiExtension on _TerminalClientShellState {
  bool _shouldShowFullscreenStatusOverlay() {
    return _bugReceiptState != BugReceiptChromeState.none ||
        _lastTransportDiagnostic.trim().isNotEmpty ||
        _lastNotification.trim().isNotEmpty;
  }

  Widget _buildTransportStatusCard() {
    return ControlStreamStatusCard(
      status: _status,
      notification: _lastNotification,
      transportDiagnostics: _lastTransportDiagnostic,
      bugReceiptState: _bugReceiptChromeState,
      bugReceiptReportId: _bugReceiptReportId,
      bugReceiptDetail: _bugReceiptDetail,
    );
  }

  Widget _buildMetadataFooter() {
    return ClientMetadataFooter(
      buildDate: _buildDate,
      buildSha: _buildSha,
    );
  }

  Widget _buildBuildParityPanel() {
    return BuildParityPanel(
      clientBuildDate: _buildDate,
      clientBuildSha: _buildSha,
      serverBuildDate: _serverBuildDate,
      serverBuildSha: _serverBuildSha,
      hasRegisterAck: _hasRegisterAck,
    );
  }

  Widget _buildDiagnosticsPanel() {
    return DiagnosticsPanel(
      title: _diagnosticsTitle,
      data: _diagnosticsData,
    );
  }

  Widget _buildBugReceiptPanel() {
    return BugReceiptPanel(
      state: _bugReceiptChromeState,
      reportId: _bugReceiptReportId,
      detail: _bugReceiptDetail,
    );
  }

  Widget _buildTransportDiagnosticsPanel() {
    final recentAttempts = _carrierAttemptLog.reversed.take(4).toList();
    return TransportDiagnosticsPanel(
      lastTransportDiagnostic: _lastTransportDiagnostic,
      recentAttempts: recentAttempts
          .map((attempt) => formatCarrierAttempt(attempt))
          .toList(),
    );
  }

  Widget _buildServerDrivenRenderer(uiv1.Node root) {
    return ServerDrivenRenderer(
      root: root,
      onAction: (action) {
        unawaited(_handleServerDrivenAction(action));
      },
      mediaSurfaceBuilder: (context, componentId, trackId) =>
          _buildVideoSurface(componentId: componentId, trackId: trackId),
      audioVisualizerBuilder: (context, componentId, streamId) =>
          _buildAudioVisualizer(componentId: componentId, streamId: streamId),
      textInputBindingResolver: _textInputBindingForComponent,
    );
  }

  Future<void> _handleServerDrivenAction(ServerDrivenAction action) async {
    await _sendUiAction(action);
  }

  ServerDrivenTextInputBinding? _textInputBindingForComponent(
    String componentId,
  ) {
    if (componentId != 'terminal_input') {
      return null;
    }
    if (_terminalInputController.text != _terminalInputShadow) {
      _recordClientLog(
        'warn',
        'terminal_input shadow mismatch on render: controller="${_terminalInputController.text}" shadow="$_terminalInputShadow"',
      );
      _terminalInputShadow = _terminalInputController.text;
    }
    return ServerDrivenTextInputBinding(
      controller: _terminalInputController,
      focusNode: _terminalInputFocusNode,
      autofocus: false,
      onChanged: _handleTerminalInputChanged,
      onSubmitted: (value) async {
        await _sendKeyText('\n');
        _terminalInputController.clear();
        _terminalInputShadow = '';
      },
    );
  }

  void _handleTerminalInputChanged(String value) {
    final previous = _terminalInputShadow;
    _recordClientLog(
      'info',
      'terminal_input.onChanged: prev="$previous" new="$value" controller.text="${_terminalInputController.text}"',
    );
    if (value.startsWith(previous) && value.length > previous.length) {
      final inserted = value.substring(previous.length);
      if (inserted.isNotEmpty) {
        _recordClientLog('info', 'detected insertion: "$inserted"');
        unawaited(_sendKeyText(inserted));
      }
    } else if (previous.startsWith(value) && previous.length > value.length) {
      final removed = previous.length - value.length;
      if (removed > 0) {
        _recordClientLog('info', 'detected $removed backspace(s)');
        unawaited(_sendKeyText(List<String>.filled(removed, '\b').join()));
      }
    } else if (value != previous) {
      _recordClientLog(
        'warn',
        'shadow sync lost: shadow="$previous" controller="$value" (no clear insertion/deletion)',
      );
    }
    _terminalInputShadow = value;
  }

  void _applyTransitionHint(ServerDrivenTransitionHint transitionHint) {
    _activeTransition = transitionHint.transition;
    _activeTransitionDuration = transitionHint.duration;
    _lastNotification = transitionHint.notification;
  }

  Widget _buildTransition(Widget child, Animation<double> animation) {
    switch (_activeTransition) {
      case 'fade':
        return FadeTransition(opacity: animation, child: child);
      case 'pa_source_enter':
        return ScaleTransition(scale: animation, child: child);
      case 'pa_source_exit':
        return FadeTransition(opacity: animation, child: child);
      case 'scale':
        return ScaleTransition(scale: animation, child: child);
      case 'slide':
      case 'slide_left':
      case 'slide-left':
      case 'pa_receive_enter':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0.15, 0),
        );
      case 'slide_right':
      case 'slide-right':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(-0.15, 0),
        );
      case 'slide_up':
      case 'slide-up':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, 0.15),
        );
      case 'slide_down':
      case 'slide-down':
      case 'pa_receive_exit':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, -0.15),
        );
      default:
        return child;
    }
  }

  Widget _buildSlideTransition({
    required Widget child,
    required Animation<double> animation,
    required Offset beginOffset,
  }) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: beginOffset,
        end: Offset.zero,
      ).animate(animation),
      child: child,
    );
  }

  Future<void> _sendUiAction(ServerDrivenAction action) async {
    if (_deviceId.isEmpty) {
      return;
    }
    _recordUiAction(
      componentId: action.componentId,
      action: action.action,
      value: action.value,
    );
    if (action.action == 'privacy.toggle') {
      await _handlePrivacyToggleAction();
      return;
    }
    if (action.action.startsWith(bugReportActionPrefix)) {
      await _submitBugReportFromAction(
        componentId: action.componentId,
        action: action.action,
        value: action.value,
      );
      return;
    }
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.uiAction,
        request: buildUiActionInputRequest(
          deviceID: _deviceId,
          action: action,
        ),
      ),
    );
  }

  Future<void> _sendKeyText(String text) async {
    if (_deviceId.isEmpty || text.isEmpty) {
      _recordClientLog('warn',
          '_sendKeyText called with deviceId.isEmpty=${_deviceId.isEmpty} text.isEmpty=${text.isEmpty}');
      return;
    }
    _recordClientLog('info',
        'sending key text: ${text.replaceAll(String.fromCharCode(127), '<DEL>')}');
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.keyEvent,
        request: buildKeyInputRequest(deviceID: _deviceId, text: text),
      ),
    );
  }

  Widget _buildVideoSurface({
    required String componentId,
    required String trackId,
  }) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 6),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Colors.black,
        border: Border.all(color: Colors.blueGrey.shade700),
        borderRadius: BorderRadius.circular(8),
      ),
      child: SizedBox(
        height: 160,
        child: Stack(
          children: [
            Positioned.fill(
              child: VideoSurfaceView(
                streamListenable: _mediaEngine.remoteStream(trackId),
              ),
            ),
            Align(
              alignment: Alignment.topLeft,
              child: ValueListenableBuilder<bool>(
                valueListenable: _mediaEngine.streamAttached(trackId),
                builder: (context, attached, _) {
                  return Container(
                    key: ValueKey<String>(
                      'ui-video-surface-state-$componentId',
                    ),
                    margin: const EdgeInsets.all(6),
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.black54,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      attached ? 'Attached' : 'Waiting for media',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 11,
                      ),
                    ),
                  );
                },
              ),
            ),
            if (trackId.isNotEmpty)
              Align(
                alignment: Alignment.bottomRight,
                child: Container(
                  margin: const EdgeInsets.all(6),
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black54,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    trackId,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 11,
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildAudioVisualizer({
    required String componentId,
    required String streamId,
  }) {
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 6),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: ValueListenableBuilder<bool>(
        valueListenable: _mediaEngine.streamAttached(streamId),
        builder: (context, attached, _) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  const Text('Audio level'),
                  const Spacer(),
                  Container(
                    key: ValueKey<String>(
                      'ui-audio-visualizer-state-$componentId',
                    ),
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.blueGrey.shade50,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      attached ? 'Attached' : 'Waiting for media',
                      style: const TextStyle(fontSize: 11),
                    ),
                  ),
                ],
              ),
              if (streamId.isNotEmpty)
                Text(streamId, style: const TextStyle(fontSize: 12)),
              const SizedBox(height: 8),
              ValueListenableBuilder<double>(
                valueListenable: _mediaEngine.audioLevel(streamId),
                builder: (context, level, _) {
                  return LinearProgressIndicator(
                    value: attached ? level.clamp(0.0, 1.0).toDouble() : null,
                  );
                },
              ),
            ],
          );
        },
      ),
    );
  }
}
