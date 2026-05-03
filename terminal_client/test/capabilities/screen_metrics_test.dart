import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';

void main() {
  test('orientationForScreenSize derives landscape and portrait', () {
    expect(
      orientationForScreenSize(
        const Size(800, 600),
        fallbackOrientation: 'unknown',
      ),
      'landscape',
    );
    expect(
      orientationForScreenSize(
        const Size(390, 844),
        fallbackOrientation: 'unknown',
      ),
      'portrait',
    );
  });

  test('orientationForScreenSize returns fallback for invalid sizes', () {
    expect(
      orientationForScreenSize(
        Size.zero,
        fallbackOrientation: 'previous',
      ),
      'previous',
    );
  });

  test('normalizeScreenMetrics rejects invalid sizes', () {
    final metrics = normalizeScreenMetrics(
      ScreenMetrics(
        logicalSize: Size.zero,
        devicePixelRatio: 2,
      ),
      fallbackOrientation: 'portrait',
    );

    expect(metrics, isNull);
  });

  test('normalizeScreenMetrics clamps device pixel ratio and orientation', () {
    final metrics = normalizeScreenMetrics(
      ScreenMetrics(
        logicalSize: const Size(1024, 768),
        devicePixelRatio: 0,
        safeAreaInsets: const EdgeInsets.fromLTRB(1, 2, 3, 4),
        orientation: '   ',
      ),
      fallbackOrientation: 'portrait',
    );

    expect(metrics, isNotNull);
    expect(metrics!.devicePixelRatio, 1.0);
    expect(metrics.orientation, 'landscape');
    expect(metrics.safeAreaInsets, const EdgeInsets.fromLTRB(1, 2, 3, 4));
  });

  test('displayGeometrySignature formats stable display identity', () {
    final signature = displayGeometrySignature(
      logicalSize: const Size(390.4, 844.2),
      devicePixelRatio: 3,
      safeAreaInsets: const EdgeInsets.fromLTRB(0, 47, 0, 34),
      orientation: 'portrait',
    );

    expect(signature, '390x844@3.000:0.00,47.00,0.00,34.00:portrait');
  });
}
