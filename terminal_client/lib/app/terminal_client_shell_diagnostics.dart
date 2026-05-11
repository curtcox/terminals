part of 'terminal_client_shell.dart';

extension _DiagnosticsExtension on _TerminalClientShellState {
  void _sendRuntimeStatusQuery() {
    final requestID = _nextDebugRequestID('debug-runtime-status');
    _pendingRuntimeStatusRequestID = requestID;
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.runtimeQuery,
        request: buildRuntimeStatusQueryRequest(requestID),
      ),
    );
  }

  void _sendDeviceStatusQuery() {
    final requestID = _nextDebugRequestID('debug-device-status');
    _pendingDeviceStatusRequestID = requestID;
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.deviceQuery,
        request: buildDeviceStatusQueryRequest(
          requestID: requestID,
          deviceID: _deviceId,
        ),
      ),
    );
  }

  void _sendScenarioRegistryQuery() {
    final requestID = _nextDebugRequestID('debug-scenario-registry');
    _pendingScenarioRegistryRequestID = requestID;
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.scenarioQuery,
        request: buildScenarioRegistryQueryRequest(requestID),
      ),
    );
  }

  void _sendPlaybackArtifactsQuery() {
    final requestID = _nextDebugRequestID('debug-playback-artifacts');
    _pendingPlaybackArtifactsRequestID = requestID;
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.playbackArtifactsQuery,
        request: buildPlaybackArtifactsQueryRequest(requestID),
      ),
    );
  }

  void _sendPlaybackMetadataQuery() {
    final artifactID = _playbackArtifactIdController.text.trim();
    if (artifactID.isEmpty) {
      _rebuildState(() {
        _status = 'Command error';
        _lastNotification = 'Playback artifact ID required';
      });
      return;
    }
    var targetDeviceID = _playbackTargetDeviceIdController.text.trim();
    if (targetDeviceID.isEmpty) {
      targetDeviceID = _deviceId;
      _playbackTargetDeviceIdController.text = targetDeviceID;
    }
    final requestID = _nextDebugRequestID('debug-playback-metadata');
    _pendingPlaybackMetadataRequestID = requestID;
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.playbackMetadataQuery,
        request: buildPlaybackMetadataQueryRequest(
          requestID: requestID,
          deviceID: _deviceId,
          artifactID: artifactID,
          targetDeviceID: targetDeviceID,
        ),
      ),
    );
  }

  void _applyDiagnosticsResponse(ConnectResponse response) {
    final diagnostics = commandDiagnosticsFromResponse(
      response: response,
      pendingRequestIDs: CommandDiagnosticsRequestIDs(
        runtimeStatus: _pendingRuntimeStatusRequestID,
        deviceStatus: _pendingDeviceStatusRequestID,
        scenarioRegistry: _pendingScenarioRegistryRequestID,
        playbackArtifacts: _pendingPlaybackArtifactsRequestID,
        playbackMetadata: _pendingPlaybackMetadataRequestID,
      ),
    );
    if (diagnostics == null) {
      return;
    }

    _diagnosticsTitle = diagnostics.title;
    _diagnosticsData = diagnostics.data;
    if (diagnostics.title == 'list_playback_artifacts') {
      final firstArtifactID = firstPlaybackArtifactID(diagnostics.data);
      if (firstArtifactID.isNotEmpty) {
        _playbackArtifactIdController.text = firstArtifactID;
      }
    } else if (diagnostics.title == 'scenario_registry') {
      _refreshAvailableApplications(diagnostics.data);
    }
  }

  void _applyRegisterMetadata(ConnectResponse response) {
    final metadata = registerMetadataFromResponse(response);
    if (metadata == null) {
      return;
    }
    _serverBuildSha = metadata.serverBuildSha;
    _serverBuildDate = metadata.serverBuildDate;
    if (metadata.hasDiagnosticsData) {
      _diagnosticsTitle = 'register_ack';
      _diagnosticsData = metadata.metadata;
    }
  }

  void _refreshAvailableApplications(Map<String, String> data) {
    _availableApplicationIntents = applicationIntentsFromDiagnostics(data);
    if (!_availableApplicationIntents.contains(_selectedApplicationIntent)) {
      _selectedApplicationIntent = _availableApplicationIntents.first;
    }
  }

  String _applicationLabel(String intent) {
    if (intent == 'terminal') {
      return 'terminal (REPL)';
    }
    return intent;
  }

  void _recordClientLog(String level, String message) {
    final entry = diagv1.LogEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..level = level
      ..message = message;
    debugPrint('[client][$level] $message');
    _appendBounded<diagv1.LogEntry>(
      _recentLogs,
      entry,
      _clientContextRecentLogCap,
    );
  }

  void _recordUiEvent({
    required String kind,
    required String componentId,
    required String detail,
    diagv1.UiEventKind kindEnum = diagv1.UiEventKind.UI_EVENT_KIND_UNSPECIFIED,
  }) {
    final entry = diagv1.UiEventEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..kind = kind
      ..componentId = componentId
      ..detail = detail;
    if (kindEnum != diagv1.UiEventKind.UI_EVENT_KIND_UNSPECIFIED) {
      entry.kindEnum = kindEnum;
    }
    _appendBounded<diagv1.UiEventEntry>(
      _recentUiEvents,
      entry,
      _clientContextRecentUiCap,
    );
  }

  void _recordUiAction({
    required String componentId,
    required String action,
    required String value,
  }) {
    final entry = diagv1.UiActionEntry()
      ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
      ..componentId = componentId
      ..action = action
      ..value = value;
    _appendBounded<diagv1.UiActionEntry>(
      _recentUiActions,
      entry,
      _clientContextRecentUiCap,
    );
  }

  diagv1.ClientContext _buildClientContext() {
    final dispatcher = WidgetsBinding.instance.platformDispatcher;
    final locale = dispatcher.locale;
    final timezone = dispatcher.locale.toLanguageTag().isNotEmpty
        ? DateTime.now().timeZoneName
        : '';
    final size = _currentLogicalSize();
    final devicePixelRatio = _currentDevicePixelRatio();
    final orientation = _lastKnownOrientation;

    final runtime = diagv1.RuntimeState()
      ..activeUiRoot = (_activeRoot?.deepCopy() ?? uiv1.Node())
      ..recentUiUpdates.addAll(_recentUiEvents.map((item) => item.deepCopy()))
      ..recentUiActions.addAll(_recentUiActions.map((item) => item.deepCopy()))
      ..recentLogs.addAll(_recentLogs.map((item) => item.deepCopy()));

    _activeStreamsByID.forEach((streamID, start) {
      final entry = diagv1.StreamEntry()
        ..streamId = streamID
        ..kind = start.kind
        ..sourceDeviceId = start.sourceDeviceId
        ..targetDeviceId = start.targetDeviceId;
      if (start.streamKind != iov1.StreamKind.STREAM_KIND_UNSPECIFIED) {
        entry.streamKind = start.streamKind;
      }
      runtime.activeStreams.add(entry);
    });
    _routesByStreamID.forEach((streamID, route) {
      final entry = diagv1.RouteEntry()
        ..streamId = streamID
        ..sourceDeviceId = route.sourceDeviceId
        ..targetDeviceId = route.targetDeviceId
        ..kind = route.kind;
      if (route.streamKind != iov1.StreamKind.STREAM_KIND_UNSPECIFIED) {
        entry.streamKind = route.streamKind;
      }
      runtime.activeRoutes.add(entry);
    });
    for (final signal in _recentWebRTCSignals) {
      final entry = diagv1.WebrtcSignalEntry()
        ..unixMs = Int64(DateTime.now().toUtc().millisecondsSinceEpoch)
        ..streamId = signal.streamId
        ..signalType = signal.signalType;
      final typed = iov1.WebRTCSignalType.valueOf(signal.signalTypeEnum.value);
      if (typed != null &&
          typed != iov1.WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_UNSPECIFIED) {
        entry.signalTypeEnum = typed;
      }
      runtime.recentWebrtcSignals.add(entry);
    }

    final connection = diagv1.ConnectionHealth()
      ..lastHeartbeatUnixMs = Int64(_lastHeartbeatUnixMs)
      ..reconnectAttempt = _reconnectAttempt
      ..lastStatus = _lastConnectionStatus
      ..lastRttMs = _lastRttMs
      ..online = _shouldStayConnected;
    connection.recentControlErrors
        .addAll(_recentControlErrors.map((item) => item.deepCopy()));

    final hardware = diagv1.HardwareState()
      ..screenWidthPx = size.width.round()
      ..screenHeightPx = size.height.round()
      ..devicePixelRatio = devicePixelRatio
      ..orientation = orientation;
    final batteryLevel = _lastSensorSnapshot['battery.level'];
    if (batteryLevel != null) {
      hardware.batteryLevel = batteryLevel;
    }
    final batteryCharging = _lastSensorSnapshot['battery.charging'];
    if (batteryCharging != null) {
      hardware.batteryCharging = batteryCharging >= 0.5;
    }
    hardware.sensorSnapshot.addAll(_lastSensorSnapshot);

    final contextProto = diagv1.ClientContext()
      ..identity = (diagv1.ClientIdentity()
        ..deviceId = _deviceId
        ..deviceName = _deviceNameController.text.trim()
        ..deviceType = _deviceTypeController.text.trim()
        ..platform = _platformController.text.trim()
        ..clientVersion = const String.fromEnvironment(
          'TERMINALS_CLIENT_VERSION',
        )
        ..clientGitSha = const String.fromEnvironment('TERMINALS_GIT_SHA')
        ..clientBuildUnixMs = Int64(
          int.tryParse(
                const String.fromEnvironment('TERMINALS_BUILD_UNIX_MS'),
              ) ??
              0,
        )
        ..osVersion = '${defaultTargetPlatform.name}${kIsWeb ? ':web' : ''}'
        ..locale = locale.toLanguageTag()
        ..timezone = timezone
        ..clockOffsetMs = Int64(0))
      ..runtime = runtime
      ..connection = connection
      ..hardware = hardware
      ..errorCapture = (diagv1.ErrorCapture()
        ..lastErrorMessage = _lastFlutterErrorMessage
        ..lastErrorStack = _lastFlutterErrorStack
        ..lastErrorUnixMs = Int64(_lastFlutterErrorUnixMs));
    final registeredCapabilities =
        _capabilitySession.lastRegisteredCapabilities;
    if (registeredCapabilities != null) {
      contextProto.capabilities = registeredCapabilities.deepCopy();
    }
    return contextProto;
  }

  String _nextDebugRequestID(String prefix) {
    _debugCommandSeq += 1;
    return '$prefix-$_debugCommandSeq';
  }

  void _appendBounded<T>(List<T> items, T next, int maxItems) {
    items.add(next);
    if (items.length > maxItems) {
      items.removeRange(0, items.length - maxItems);
    }
  }

  void _launchSelectedApplication() {
    final intent = _selectedApplicationIntent.trim();
    if (intent.isEmpty) {
      _rebuildState(() {
        _status = 'Command error';
        _lastNotification = 'Select an application before launching';
      });
      return;
    }
    if (!_isConnectionRegistered) {
      _rebuildState(() {
        _pendingLaunchApplicationIntent = intent;
        _status = 'Connecting';
        _lastNotification =
            'Connecting control stream to open application: $intent';
      });
      if (!_isConnecting) {
        unawaited(_startStream(userInitiated: true));
      }
      return;
    }
    _sendApplicationLaunchCommand(intent);
  }

  void _sendApplicationLaunchCommand(String intent) {
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.launchApplication,
        request: buildApplicationLaunchCommandRequest(
          requestID: _nextDebugRequestID('launch-app'),
          deviceID: _deviceId,
          intent: intent,
        ),
      ),
    );
    if (mounted) {
      _rebuildState(() {
        _lastNotification = 'Launching application: $intent';
      });
      return;
    }
    _lastNotification = 'Launching application: $intent';
  }

  void _installFlutterErrorHook() {
    _previousFlutterErrorHandler = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      _lastFlutterErrorMessage = details.exceptionAsString();
      _lastFlutterErrorStack = details.stack?.toString() ?? '';
      _lastFlutterErrorUnixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
      _recordClientLog('error', _lastFlutterErrorMessage);
      final prior = _previousFlutterErrorHandler;
      if (prior != null) {
        prior(details);
      } else {
        FlutterError.presentError(details);
      }
    };
  }

  void _restoreFlutterErrorHook() {
    if (FlutterError.onError == null) {
      return;
    }
    FlutterError.onError = _previousFlutterErrorHandler;
    _previousFlutterErrorHandler = null;
  }
}
