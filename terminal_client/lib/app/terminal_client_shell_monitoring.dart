part of 'terminal_client_shell.dart';

extension _MonitoringExtension on _TerminalClientShellState {
  void _startHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(widget.heartbeatInterval, (_) {
      if (!_hasActiveControlSession || _deviceId.isEmpty || !_appIsForeground) {
        return;
      }
      final unixMs = DateTime.now().toUtc().millisecondsSinceEpoch;
      _lastHeartbeatUnixMs = unixMs;
      _pendingHeartbeatUnixMs = unixMs;
      unawaited(
        _sendWhenReady(
          operation: OutboundOperation.heartbeat,
          request: TerminalControlGrpcClient.heartbeatRequest(
            deviceId: _deviceId,
            unixMs: unixMs,
          ),
        ),
      );
    });
  }

  void _stopHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  void _startSensorTelemetryLoop() {
    _sensorTimer?.cancel();
    _sensorTimer = Timer.periodic(widget.sensorTelemetryInterval, (_) {
      if (!_hasActiveControlSession || _deviceId.isEmpty || !_appIsForeground) {
        return;
      }
      _sendSensorTelemetry();
    });
  }

  void _stopSensorTelemetryLoop() {
    _sensorTimer?.cancel();
    _sensorTimer = null;
  }

  void _startCapabilityMonitorLoop() {
    _capabilityMonitorTimer?.cancel();
    _capabilityMonitorTimer = Timer.periodic(_capabilityMonitorInterval, (_) {
      unawaited(
          _probeAndPublishCapabilityChanges(reason: 'runtime_monitor_poll'));
    });
  }

  void _stopCapabilityMonitorLoop() {
    _capabilityMonitorTimer?.cancel();
    _capabilityMonitorTimer = null;
  }

  void _syncMonitoringLoops() {
    if (_hasActiveControlSession && _appIsForeground) {
      _startHeartbeatLoop();
      _startSensorTelemetryLoop();
      _startCapabilityMonitorLoop();
      return;
    }
    _stopHeartbeatLoop();
    _stopSensorTelemetryLoop();
    _stopCapabilityMonitorLoop();
  }

  ConnectRequest? _buildSensorTelemetryRequest() {
    return buildSensorTelemetryRequest(
      deviceID: _deviceId,
      capabilities: _capabilitySession.lastRegisteredCapabilities,
      unixMs: DateTime.now().toUtc().millisecondsSinceEpoch,
    );
  }

  void _sendSensorTelemetry() {
    if (_deviceId.isEmpty) {
      return;
    }
    final request = _buildSensorTelemetryRequest();
    if (request == null) {
      return;
    }
    _lastSensorSnapshot
      ..clear()
      ..addAll(request.sensor.values);
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.sensorTelemetry,
        request: request,
      ),
    );
    final unixMs = request.sensor.unixMs.toInt();
    if (mounted) {
      _rebuildState(() {
        _sensorSendCount += 1;
        _lastSensorSendUnixMs = unixMs;
      });
      return;
    }
    _sensorSendCount += 1;
    _lastSensorSendUnixMs = unixMs;
  }

  int _nowUnixMs() => widget.nowUnixMsProvider();
}
