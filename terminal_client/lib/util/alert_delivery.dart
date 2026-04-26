import 'dart:async';

import 'package:terminal_client/util/alerts.dart' as alerts;
import 'package:terminal_client/util/speech.dart' as speech;

typedef AlertNotificationDispatcher = Future<void> Function({
  required String title,
  required String body,
  required String level,
});
typedef AlertSpeaker = void Function(String text);

class AlertDeliveryService {
  AlertDeliveryService({
    AlertNotificationDispatcher? notificationDispatcher,
    AlertSpeaker? speaker,
  })  : _notificationDispatcher = notificationDispatcher ?? alerts.showAlert,
        _speaker = speaker ?? speech.speakText;

  final AlertNotificationDispatcher _notificationDispatcher;
  final AlertSpeaker _speaker;

  void deliver({
    required String title,
    required String body,
    required String level,
  }) {
    final normalizedTitle = title.trim();
    final normalizedBody = body.trim();
    if (normalizedTitle.isEmpty && normalizedBody.isEmpty) {
      return;
    }

    unawaited(
      _dispatchNotification(
        title: normalizedTitle,
        body: normalizedBody,
        level: level.trim(),
      ),
    );

    final spoken = normalizedBody.isNotEmpty ? normalizedBody : normalizedTitle;
    _speaker(spoken);
  }

  Future<void> _dispatchNotification({
    required String title,
    required String body,
    required String level,
  }) async {
    try {
      await _notificationDispatcher(title: title, body: body, level: level);
    } catch (_) {
      // Keep alert delivery best-effort; speech still provides fallback output.
    }
  }
}
