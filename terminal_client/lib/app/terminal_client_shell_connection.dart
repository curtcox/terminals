part of 'terminal_client_shell.dart';

extension _ConnectionExtension on _TerminalClientShellState {
  ConnectionPhase get _connectionPhase => deriveConnectionPhase(
        shouldStayConnected: _shouldStayConnected,
        isConnecting: _isConnecting,
        hasClient: _client != null,
        hasIncoming: _incoming != null,
        hasRegisterAck: _hasRegisterAck,
        hasRecentTransportFailure: _lastTransportDiagnostic.trim().isNotEmpty,
      );

  bool get _isConnectionRegistered =>
      _connectionPhase == ConnectionPhase.registered;

  bool get _hasActiveControlSession =>
      _shouldStayConnected && _incoming != null && _client != null;

  Future<void> _ensureConnectedForDispatch() async {
    if (_isConnectionRegistered || _hasActiveControlSession) {
      return;
    }
    _shouldStayConnected = true;
    if (!_isConnecting) {
      await _startStream(userInitiated: true);
    }
  }

  Future<SendResult> _sendWhenReady({
    required OutboundOperation operation,
    required ConnectRequest request,
    Future<bool> Function()? waitForAck,
    Duration? ackTimeout,
  }) async {
    final rule = kOutboundRoutingRules[operation] ??
        const OutboundRoutingRule(
          mode: SendMode.fireAndForget,
          safeToReplay: false,
          requiresAck: false,
        );
    return _reliableSender.sendWhenReady(
      request: request,
      mode: rule.mode,
      waitForAck: waitForAck,
      ackTimeout: ackTimeout,
    );
  }

  Future<void> _startStream({bool userInitiated = true}) async {
    if (_isConnecting) {
      return;
    }
    if (userInitiated) {
      _shouldStayConnected = true;
      _reconnectAttempt = 0;
      _cancelReconnectTimer();
    }

    final pageHost = _currentPageHost();
    final configuredHost = _hostController.text.trim();
    final host = resolveInitialControlHost(
      isWebRuntime: kIsWeb,
      configuredHost: configuredHost,
      pageHost: pageHost,
    );
    if (host != configuredHost) {
      _hostController.text = host;
    }
    final port = int.tryParse(_portController.text.trim());
    final selectedServer = _selectedServerMetadata();
    final size = _currentLogicalSize();
    if (host.isEmpty || port == null || port <= 0 || port > 65535) {
      _shouldStayConnected = false;
      _hasRegisterAck = false;
      _rebuildState(() {
        _status = 'Invalid host or port';
        _lastConnectionStatus = _status;
      });
      return;
    }

    if (userInitiated || _activeCarrierCycle.isEmpty) {
      _resetCarrierCycle(_carrierPreference(selectedServer));
    }
    if (_activeCarrierCycle.isEmpty) {
      _shouldStayConnected = false;
      _hasRegisterAck = false;
      _rebuildState(() {
        _status = 'No supported control carrier is available on this runtime';
        _lastConnectionStatus = _status;
      });
      return;
    }
    if (_activeCarrierIndex < 0 ||
        _activeCarrierIndex >= _activeCarrierCycle.length) {
      _activeCarrierIndex = 0;
    }

    final carrier = _activeCarrierCycle[_activeCarrierIndex];
    final rawTarget = resolveConnectionTargetForCarrier(
      carrier: carrier,
      endpoint: _endpointForCarrier(carrier: carrier, server: selectedServer),
      fallbackHost: host,
      fallbackPort: port,
    );
    final target = ConnectionTarget(
      host: resolveInitialControlHost(
        isWebRuntime: kIsWeb,
        configuredHost: rawTarget.host,
        pageHost: pageHost,
      ),
      port: rawTarget.port,
    );
    final carrierEndpoint = buildCarrierEndpointLabel(
      carrier: carrier,
      target: target,
      websocketPath: _websocketPathFor(selectedServer),
    );
    final attemptStartedAt = DateTime.now().toUtc();

    _isConnecting = true;
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    _capabilitySession.reset();
    if (mounted) {
      _rebuildState(() {
        final verb = userInitiated ? 'Connecting' : 'Reconnecting';
        _status = '$verb via ${controlCarrierLabel(carrier)}...';
        _lastConnectionStatus = _status;
      });
    }
    try {
      await _incoming?.cancel();
      _incoming = null;
      final existingClient = _client;
      if (existingClient != null) {
        await existingClient.shutdown();
      }
      ControlClientTransportHint.configure(
        carrier: carrier,
        wsPath: _websocketPathFor(selectedServer),
        tcp: _endpointForCarrier(
          carrier: ControlCarrierKind.tcp,
          server: selectedServer,
        ),
        http: _endpointForCarrier(
          carrier: ControlCarrierKind.http,
          server: selectedServer,
        ),
        desiredDeviceIdHint: _deviceId,
        resumeTokenHint: ControlClientTransportHint.resumeToken,
      );
      _client = widget.clientFactory(
        host: target.host,
        port: target.port,
      );

      final stream = _client!.connect(_outgoing.stream);
      _incoming = stream.listen(
        (ConnectResponse response) {
          if (!mounted) {
            return;
          }
          final nowUnixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
          var shouldFlushQueuedBugReports = false;
          _reconnectAttempt = 0;
          _rebuildState(() {
            if (_pendingHeartbeatUnixMs > 0) {
              _lastRttMs = (nowUnixMs - _pendingHeartbeatUnixMs).toDouble();
              _pendingHeartbeatUnixMs = 0;
            }
            _responses += 1;
            final responseStatus = statusFromConnectResponse(response);
            if (responseStatus.isNotEmpty) {
              _status = responseStatus;
              _lastConnectionStatus = responseStatus;
            }
            final uiUpdate = serverDrivenUiUpdateFromResponse(
              response: response,
              currentRoot: _activeRoot,
            );
            if (uiUpdate != null) {
              _activeRoot = uiUpdate.activeRoot;
              final transitionHint = uiUpdate.transitionHint;
              if (transitionHint != null) {
                _applyTransitionHint(transitionHint);
              }
              for (final event in uiUpdate.events) {
                _recordUiEvent(
                  kind: event.kind,
                  componentId: event.componentId,
                  detail: event.detail,
                  kindEnum: event.kindEnum,
                );
              }
            }
            if (uiUpdate?.uiChanged ?? false) {
              _activeRootRevision += 1;
            }
            if (response.hasNotification()) {
              _lastNotification = response.notification.body;
              widget.alertDelivery(
                title: response.notification.title,
                body: response.notification.body,
                level: response.notification.level,
              );
            }
            if (response.hasCommandResult() &&
                response.commandResult.notification.isNotEmpty) {
              _lastNotification = response.commandResult.notification;
            }
            _applyDiagnosticsResponse(response);
            if (response.hasError()) {
              _lastNotification = response.error.message;
              _appendBounded<diagv1.ControlErrorEntry>(
                _recentControlErrors,
                diagv1.ControlErrorEntry()
                  ..unixMs = Int64(nowUnixMs)
                  ..code = response.error.code.name
                  ..message = response.error.message,
                _clientContextRecentErrorCap,
              );
              _recordClientLog('error', response.error.message);
              _failPendingBugReportsForControlError(response.error);
              if (_isStaleCapabilityGenerationError(response.error)) {
                unawaited(
                  _probeAndPublishCapabilityChanges(
                    reason: 'stale_generation_rebaseline',
                    forceSnapshot: true,
                  ),
                );
              }
            }
            if (response.hasBugReportAck()) {
              _handleBugReportAck(response.bugReportAck);
            }
            if (response.hasRegisterAck()) {
              final firstRegisterAck = !_hasRegisterAck;
              _hasRegisterAck = true;
              _cancelRegisterAckRetry();
              _lastSuccessfulCarrier = carrier;
              _carrierAttemptLog.clear();
              if (firstRegisterAck) {
                shouldFlushQueuedBugReports = true;
                _sendScenarioRegistryQuery();
              }
              if (_pendingLaunchApplicationIntent.isNotEmpty) {
                final pendingIntent = _pendingLaunchApplicationIntent;
                _pendingLaunchApplicationIntent = '';
                _sendApplicationLaunchCommand(pendingIntent);
              }
            }
            if (response.hasCapabilityAck()) {
              _capabilitySession.observeAckGeneration(
                response.capabilityAck.acceptedGeneration.toInt(),
              );
            }
            _applyRegisterMetadata(response);
            if (_e2eEmitEvents && response.hasRegisterAck()) {
              debugPrint('E2E_EVENT: register_ack');
            }
            _applyMediaControlResponse(response);
          });
          if (shouldFlushQueuedBugReports) {
            _flushQueuedBugReports();
          }
        },
        onError: (Object error) {
          final diagnosis = diagnoseTransportError(error, isWeb: kIsWeb);
          unawaited(
            _handleCarrierAttemptFailure(
              carrier: carrier,
              endpoint: carrierEndpoint,
              stage: 'stream',
              status: diagnosis.statusText(),
              rawError: diagnosis.hasSummary
                  ? diagnosis.notificationText()
                  : diagnosis.rawError,
              elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
            ),
          );
        },
        onDone: () {
          if (!_shouldStayConnected) {
            return;
          }
          if (!_hasRegisterAck) {
            unawaited(
              _handleCarrierAttemptFailure(
                carrier: carrier,
                endpoint: carrierEndpoint,
                stage: 'stream_closed',
                status: 'Disconnected',
                rawError: 'stream closed before register acknowledgement',
                elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
              ),
            );
            return;
          }
          unawaited(
            _handleCarrierAttemptFailure(
              carrier: carrier,
              endpoint: carrierEndpoint,
              stage: 'stream_closed',
              status: 'Disconnected',
              rawError: 'stream closed after register acknowledgement',
              elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
            ),
          );
        },
      );

      _syncMonitoringLoops();
      final probedCapabilities = await _capabilityProbe.probe(
        CapabilityProbeContext(
          deviceId: _deviceId,
          deviceName: _deviceNameController.text.trim(),
          deviceType: _deviceTypeController.text.trim(),
          platform: _platformController.text.trim(),
          screenWidth: size.width.round(),
          screenHeight: size.height.round(),
          screenDensity: _currentDevicePixelRatio(),
          targetPlatform: defaultTargetPlatform,
        ),
      );
      final identity = probedCapabilities.hasIdentity()
          ? probedCapabilities.identity.deepCopy()
          : (capv1.DeviceIdentity()
            ..deviceName = _deviceNameController.text.trim()
            ..deviceType = _deviceTypeController.text.trim()
            ..platform = _platformController.text.trim());
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.bootstrapHello,
          request: TerminalControlGrpcClient.helloRequest(
            deviceId: _deviceId,
            identity: identity,
            clientVersion: 'terminal_client',
          ),
        ),
      );

      final bootstrapPublication = _capabilitySession.startBootstrap(
        _applyDisplayMetadata(
          probedCapabilities.deepCopy(),
        ),
      );
      _syncWakeWordDetector(bootstrapPublication.capabilities);
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.bootstrapCapabilitySnapshot,
          request: TerminalControlGrpcClient.capabilitySnapshotRequest(
            deviceId: _deviceId,
            generation: bootstrapPublication.generation,
            capabilities: bootstrapPublication.capabilities,
          ),
        ),
      );
      final initialHeartbeatUnixMs =
          DateTime.now().toUtc().millisecondsSinceEpoch;
      _lastHeartbeatUnixMs = initialHeartbeatUnixMs;
      _pendingHeartbeatUnixMs = initialHeartbeatUnixMs;
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.heartbeat,
          request: TerminalControlGrpcClient.heartbeatRequest(
            deviceId: _deviceId,
            unixMs: initialHeartbeatUnixMs,
          ),
        ),
      );
      _scheduleRegisterAckRetry();
      _recordClientLog('info', 'control stream connected');
      _sendSensorTelemetry();
    } catch (error) {
      final diagnosis = diagnoseTransportError(error, isWeb: kIsWeb);
      await _handleCarrierAttemptFailure(
        carrier: carrier,
        endpoint: carrierEndpoint,
        stage: 'connect',
        status: diagnosis.hasSummary
            ? 'Connection error: ${diagnosis.summary}'
            : 'Connection error: $error',
        rawError: diagnosis.hasSummary
            ? diagnosis.notificationText()
            : diagnosis.rawError,
        elapsed: DateTime.now().toUtc().difference(attemptStartedAt),
      );
    } finally {
      _isConnecting = false;
    }
  }

  Future<void> _stopStream() async {
    _shouldStayConnected = false;
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    _capabilitySession.reset();
    _reconnectAttempt = 0;
    _activeCarrierCycle = <ControlCarrierKind>[];
    _activeCarrierIndex = 0;
    _cancelReconnectTimer();
    _syncMonitoringLoops();
    _drainPendingBugReportsAsFailed('stream stopped before bug report ack');
    await _incoming?.cancel();
    _incoming = null;
    for (final streamID in _activeStreamsByID.keys.toList(growable: false)) {
      await _mediaEngine.stopStream(streamID);
    }
    _activeStreamsByID.clear();
    _routesByStreamID.clear();
    final existingClient = _client;
    if (existingClient != null) {
      await existingClient.shutdown();
      _client = null;
    }
    if (mounted) {
      _rebuildState(() {
        _status = 'Disconnected';
        _lastConnectionStatus = _status;
        _activeRoot = null;
      });
    }
  }

  Future<void> _handleCarrierAttemptFailure({
    required ControlCarrierKind carrier,
    required String endpoint,
    required String stage,
    required String status,
    required String rawError,
    required Duration elapsed,
  }) async {
    final failureClass =
        classifyCarrierFailure(stage: stage, rawError: rawError);
    _carrierAttemptLog.add(
      CarrierAttemptDiagnostic(
        carrier: carrier,
        endpoint: endpoint,
        stage: stage,
        failureClass: failureClass,
        error: rawError,
        elapsed: elapsed,
      ),
    );
    final attemptSummary = formatCarrierAttempt(_carrierAttemptLog.last);
    _lastTransportDiagnostic = attemptSummary;
    _recordClientLog(
      'warn',
      'control carrier failure carrier=${controlCarrierLabel(carrier)} stage=$stage '
          'class=$failureClass endpoint=$endpoint elapsed_ms=${elapsed.inMilliseconds} '
          'error=$rawError',
    );
    if (_shouldStayConnected) {
      _requeuePendingBugReportsForRetry(attemptSummary);
    }

    _incoming = null;
    _cancelRegisterAckRetry();
    _hasRegisterAck = false;
    final existingClient = _client;
    _client = null;
    if (existingClient != null) {
      await existingClient.shutdown();
    }
    _syncMonitoringLoops();

    if (_moveToNextCarrierInCycle()) {
      final nextCarrier = _activeCarrierCycle[_activeCarrierIndex];
      _recordClientLog(
        'info',
        'switching control carrier from ${controlCarrierLabel(carrier)} to ${controlCarrierLabel(nextCarrier)}',
      );
      if (mounted) {
        _rebuildState(() {
          _status =
              '${controlCarrierLabel(carrier)} failed, trying ${controlCarrierLabel(nextCarrier)}...';
          _lastConnectionStatus = _status;
          _lastNotification = _lastTransportDiagnostic;
        });
      }
      if (_shouldStayConnected) {
        unawaited(
          Future<void>.microtask(() => _startStream(userInitiated: false)),
        );
      }
      return;
    }

    final summary = _buildCarrierFailureSummary();
    _lastTransportDiagnostic = summary;
    _recordClientLog('error', 'all control carriers failed\n$summary');
    if (mounted) {
      _rebuildState(() {
        _status = 'All control carriers failed';
        _lastConnectionStatus = _status;
        _lastNotification = summary;
      });
    }
    _activeCarrierCycle = <ControlCarrierKind>[];
    _activeCarrierIndex = 0;
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (!_shouldStayConnected) {
      return;
    }
    if (_isConnecting) {
      if (_reconnectTimer?.isActive ?? false) {
        return;
      }
      _reconnectTimer = Timer(const Duration(milliseconds: 50), () {
        _reconnectTimer = null;
        _scheduleReconnect();
      });
      return;
    }
    if (_reconnectTimer?.isActive ?? false) {
      return;
    }
    _reconnectAttempt += 1;
    final reconnectDelay = calculateReconnectDelay(
      reconnectAttempt: _reconnectAttempt,
      reconnectDelayBase: widget.reconnectDelayBase,
      reconnectDelayMaxSeconds: widget.reconnectDelayMaxSeconds,
    );
    final displaySeconds = reconnectDelay.inMilliseconds <= 1000
        ? 1
        : (reconnectDelay.inMilliseconds / 1000).ceil();
    if (mounted) {
      _rebuildState(() {
        _status = 'Connection lost, retrying in ${displaySeconds}s...';
        _lastConnectionStatus = _status;
      });
    }
    _reconnectTimer = Timer(reconnectDelay, () {
      _reconnectTimer = null;
      if (!_shouldStayConnected || !mounted) {
        return;
      }
      unawaited(_startStream(userInitiated: false));
    });
  }

  Future<void> _scanForServers() async {
    if (kIsWeb) {
      _rebuildState(() {
        _status = 'Connected (web origin); LAN scan not required';
        _lastConnectionStatus = _status;
      });
      return;
    }
    if (_isScanning) {
      return;
    }
    _rebuildState(() {
      _isScanning = true;
      _status = 'Scanning LAN for server...';
      _lastConnectionStatus = _status;
    });
    try {
      final found = await _mdnsScanner.scan();
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _discoveredServers = found;
        if (found.isNotEmpty) {
          final first = found.first;
          _selectedDiscoveredServer = '${first.host}:${first.port}';
          _hostController.text = first.host;
          _portController.text = _grpcPortFor(first, first.port).toString();
          _status = 'Found ${found.length} server(s)';
          _lastConnectionStatus = _status;
        } else {
          _selectedDiscoveredServer = null;
          _status = 'No servers discovered';
          _lastConnectionStatus = _status;
        }
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      _rebuildState(() {
        _status = 'Discovery error: $error';
        _lastConnectionStatus = _status;
      });
    } finally {
      if (mounted) {
        _rebuildState(() {
          _isScanning = false;
        });
      }
    }
  }

  void _scheduleRegisterAckRetry() {
    _registerAckRetryController.start(
      shouldContinue: () =>
          !_hasRegisterAck &&
          _hasActiveControlSession &&
          _capabilitySession.lastRegisteredCapabilities != null,
      onRetry: (attempt) {
        final request = _buildBootstrapCapabilitySnapshotRequest();
        if (request == null) {
          return;
        }
        unawaited(
          _sendWhenReady(
            operation: OutboundOperation.bootstrapCapabilitySnapshot,
            request: request,
          ),
        );
        _recordClientLog(
          'warn',
          'register acknowledgement pending; retrying bootstrap capability snapshot '
              'attempt=$attempt',
        );
      },
      onTimeout: (attempts, elapsed) {
        _recordClientLog(
          'warn',
          'register acknowledgement still pending after '
              '${_registerAckRetryPolicy.maxDuration.inSeconds}s and $attempts retry attempts',
        );
      },
    );
  }

  void _cancelRegisterAckRetry() {
    _registerAckRetryController.stop();
  }

  void _cancelReconnectTimer() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }
}
