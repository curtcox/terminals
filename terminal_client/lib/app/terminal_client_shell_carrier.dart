part of 'terminal_client_shell.dart';

extension _CarrierExtension on _TerminalClientShellState {
  List<ControlCarrierKind> _carrierPreference(DiscoveredServer? server) {
    return buildCarrierPreference(
      isWebRuntime: kIsWeb,
      serverPriority: server?.priority ?? const <String>[],
      lastSuccessfulCarrier: _lastSuccessfulCarrier,
    );
  }

  String _websocketPathFor(DiscoveredServer? server) {
    final endpoint = _endpointForCarrier(
      carrier: ControlCarrierKind.websocket,
      server: server,
    );
    return websocketPathFromEndpoint(endpoint);
  }

  String _endpointForCarrier({
    required ControlCarrierKind carrier,
    required DiscoveredServer? server,
  }) {
    switch (carrier) {
      case ControlCarrierKind.grpc:
        return resolvePreferredEndpoint(
          manualEndpoint: _grpcEndpointController.text,
          discoveredEndpoint: server?.grpcEndpoint ?? '',
        );
      case ControlCarrierKind.websocket:
        return resolvePreferredEndpoint(
          manualEndpoint: _websocketEndpointController.text,
          discoveredEndpoint: server?.websocketEndpoint ?? '',
        );
      case ControlCarrierKind.tcp:
        return resolvePreferredEndpoint(
          manualEndpoint: _tcpEndpointController.text,
          discoveredEndpoint: server?.tcpEndpoint ?? '',
        );
      case ControlCarrierKind.http:
        return resolvePreferredEndpoint(
          manualEndpoint: _httpEndpointController.text,
          discoveredEndpoint: server?.httpEndpoint ?? '',
        );
    }
  }

  int _grpcPortFor(DiscoveredServer? server, int fallbackPort) {
    return grpcPortFromEndpoint(
      endpoint: _endpointForCarrier(
        carrier: ControlCarrierKind.grpc,
        server: server,
      ),
      fallbackPort: fallbackPort,
    );
  }

  String _currentPageHost() {
    return resolvePageHost(
      browserLocationHost: browser_host.browserLocationHost(),
      uriBaseHost: Uri.base.host,
    );
  }

  void _resetCarrierCycle(List<ControlCarrierKind> carriers) {
    _activeCarrierCycle = carriers;
    _activeCarrierIndex = 0;
    _carrierAttemptLog.clear();
  }

  bool _moveToNextCarrierInCycle() {
    if (_activeCarrierIndex + 1 >= _activeCarrierCycle.length) {
      return false;
    }
    _activeCarrierIndex += 1;
    return true;
  }

  String _buildCarrierFailureSummary() {
    if (_carrierAttemptLog.isEmpty) {
      return 'No carriers were attempted.';
    }
    final lines = <String>[];
    for (final attempt in _carrierAttemptLog) {
      lines.add(formatCarrierAttempt(attempt));
    }
    return lines.join('\n');
  }

  DiscoveredServer? _selectedServerMetadata() {
    final selected = _selectedDiscoveredServer;
    if (selected == null || selected.isEmpty) {
      return null;
    }
    for (final server in _discoveredServers) {
      if ('${server.host}:${server.port}' == selected) {
        return server;
      }
    }
    return null;
  }
}
