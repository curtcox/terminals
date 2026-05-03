import 'dart:ui' as ui;

import 'package:flutter/material.dart';

typedef ScreenMetricsProvider = ScreenMetrics Function();

const Duration kDisplayGeometryDebounceInterval = Duration(milliseconds: 120);

class ScreenMetrics {
  ScreenMetrics({
    required this.logicalSize,
    required this.devicePixelRatio,
    this.safeAreaInsets = EdgeInsets.zero,
    String? orientation,
  }) : orientation = orientation ??
            orientationForScreenSize(
              logicalSize,
              fallbackOrientation: 'unknown',
            );

  final Size logicalSize;
  final double devicePixelRatio;
  final EdgeInsets safeAreaInsets;
  final String orientation;
}

String orientationForScreenSize(
  Size size, {
  required String fallbackOrientation,
}) {
  if (size.width <= 0 || size.height <= 0) {
    return fallbackOrientation;
  }
  return size.width >= size.height ? 'landscape' : 'portrait';
}

ScreenMetrics? normalizeScreenMetrics(
  ScreenMetrics metrics, {
  required String fallbackOrientation,
}) {
  final size = metrics.logicalSize;
  if (size.width <= 0 || size.height <= 0) {
    return null;
  }
  final dpr = metrics.devicePixelRatio <= 0 ? 1.0 : metrics.devicePixelRatio;
  return ScreenMetrics(
    logicalSize: size,
    devicePixelRatio: dpr,
    safeAreaInsets: metrics.safeAreaInsets,
    orientation: metrics.orientation.trim().isEmpty
        ? orientationForScreenSize(
            size,
            fallbackOrientation: fallbackOrientation,
          )
        : metrics.orientation.trim(),
  );
}

EdgeInsets safeAreaInsetsFromView(ui.FlutterView view, double dpr) {
  final ratio = dpr <= 0 ? 1.0 : dpr;
  final padding = view.padding;
  return EdgeInsets.fromLTRB(
    padding.left / ratio,
    padding.top / ratio,
    padding.right / ratio,
    padding.bottom / ratio,
  );
}

String displayGeometrySignature({
  required Size logicalSize,
  required double devicePixelRatio,
  required EdgeInsets safeAreaInsets,
  required String orientation,
}) {
  return '${logicalSize.width.round()}x${logicalSize.height.round()}@${devicePixelRatio.toStringAsFixed(3)}:${safeAreaInsets.left.toStringAsFixed(2)},${safeAreaInsets.top.toStringAsFixed(2)},${safeAreaInsets.right.toStringAsFixed(2)},${safeAreaInsets.bottom.toStringAsFixed(2)}:$orientation';
}
