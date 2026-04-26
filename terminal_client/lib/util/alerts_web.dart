// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:html' as html;

Future<void> showAlert({
  required String title,
  required String body,
  required String level,
}) async {
  final normalizedTitle = title.trim();
  final normalizedBody = body.trim();
  if (normalizedTitle.isEmpty && normalizedBody.isEmpty) {
    return;
  }

  if (!html.Notification.supported) {
    return;
  }

  var permission = html.Notification.permission;
  if (permission != 'granted') {
    permission = await html.Notification.requestPermission();
  }
  if (permission != 'granted') {
    return;
  }

  final fallbackTitle = normalizedTitle.isEmpty ? 'Terminals' : normalizedTitle;
  final fallbackBody =
      normalizedBody.isEmpty ? normalizedTitle : normalizedBody;
  final normalizedLevel = level.trim().toLowerCase();

  try {
    html.Notification(
      fallbackTitle,
      body: fallbackBody,
      tag: 'terminals-alert-$normalizedLevel',
    );
  } catch (_) {
    // Browser notification delivery is best-effort.
  }
}
