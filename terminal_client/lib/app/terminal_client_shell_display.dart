part of 'terminal_client_shell.dart';

extension _DisplayExtension on _TerminalClientShellState {
  ScreenMetrics? _injectedScreenMetrics() {
    final provider = widget.screenMetricsProvider;
    if (provider == null) {
      return null;
    }
    return normalizeScreenMetrics(
      provider(),
      fallbackOrientation: _lastKnownOrientation,
    );
  }

  EdgeInsets _safeAreaInsetsFromView(ui.FlutterView view, double dpr) {
    return safeAreaInsetsFromView(view, dpr);
  }

  EdgeInsets _currentSafeAreaInsets() {
    final injected = _injectedScreenMetrics();
    if (injected != null) {
      return injected.safeAreaInsets;
    }
    return _lastKnownSafeAreaInsets;
  }

  Size _currentLogicalSize() {
    if (_lastKnownLogicalSize.width > 0 && _lastKnownLogicalSize.height > 0) {
      return _lastKnownLogicalSize;
    }
    final injected = _injectedScreenMetrics();
    if (injected != null) {
      return injected.logicalSize;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return Size.zero;
    }
    final view = views.first;
    final dpr = view.devicePixelRatio <= 0 ? 1.0 : view.devicePixelRatio;
    return Size(
      view.physicalSize.width / dpr,
      view.physicalSize.height / dpr,
    );
  }

  double _currentDevicePixelRatio() {
    if (_lastKnownDevicePixelRatio > 0) {
      return _lastKnownDevicePixelRatio;
    }
    final injected = _injectedScreenMetrics();
    if (injected != null) {
      return injected.devicePixelRatio;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return 1.0;
    }
    final dpr = views.first.devicePixelRatio;
    return dpr <= 0 ? 1.0 : dpr;
  }

  void _refreshDisplayMetrics() {
    final injected = _injectedScreenMetrics();
    if (injected != null) {
      _lastKnownLogicalSize = injected.logicalSize;
      _lastKnownDevicePixelRatio = injected.devicePixelRatio;
      _lastKnownSafeAreaInsets = injected.safeAreaInsets;
      _lastKnownOrientation = injected.orientation;
      return;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return;
    }
    final view = views.first;
    final dpr = view.devicePixelRatio <= 0 ? 1.0 : view.devicePixelRatio;
    final logicalSize = Size(
      view.physicalSize.width / dpr,
      view.physicalSize.height / dpr,
    );
    if (logicalSize.width <= 0 || logicalSize.height <= 0) {
      return;
    }
    _lastKnownLogicalSize = logicalSize;
    _lastKnownDevicePixelRatio = dpr;
    _lastKnownSafeAreaInsets = _safeAreaInsetsFromView(view, dpr);
    _lastKnownOrientation = _normalizedOrientationFromSize(logicalSize);
  }

  String _displaySignature({
    required Size logicalSize,
    required double devicePixelRatio,
    required EdgeInsets safeAreaInsets,
    required String orientation,
  }) {
    return displayGeometrySignature(
      logicalSize: logicalSize,
      devicePixelRatio: devicePixelRatio,
      safeAreaInsets: safeAreaInsets,
      orientation: orientation,
    );
  }

  void _scheduleDisplayGeometryCapabilityUpdate() {
    if (!_hasActiveControlSession) {
      return;
    }
    _displayGeometryDebounceTimer?.cancel();
    _displayGeometryDebounceTimer =
        Timer(widget.displayGeometryDebounceInterval, () {
      _sendLifecycleCapabilityUpdate(reason: 'display_geometry_change');
    });
  }

  String _normalizedOrientationFromSize(Size size) {
    return orientationForScreenSize(
      size,
      fallbackOrientation: _lastKnownOrientation,
    );
  }

  capv1.DeviceCapabilities _applyDisplayMetadata(
    capv1.DeviceCapabilities capabilities,
  ) {
    final size = _currentLogicalSize();
    return applyDisplayMetadataToCapabilities(
      capabilities,
      metrics: ScreenMetrics(
        logicalSize: size,
        devicePixelRatio: _currentDevicePixelRatio(),
        safeAreaInsets: _currentSafeAreaInsets(),
        orientation: _normalizedOrientationFromSize(size),
      ),
    );
  }
}
