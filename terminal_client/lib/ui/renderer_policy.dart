import 'package:flutter/foundation.dart';

class RendererPolicy {
  const RendererPolicy({
    this.showFallbackDiagnostics = kDebugMode,
  });

  final bool showFallbackDiagnostics;
}
