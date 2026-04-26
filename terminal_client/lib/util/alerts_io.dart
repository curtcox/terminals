import 'package:flutter_local_notifications/flutter_local_notifications.dart';

final FlutterLocalNotificationsPlugin _notifications =
    FlutterLocalNotificationsPlugin();

bool _initialized = false;
int _nextNotificationId = 1;

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

  try {
    await _ensureInitialized();
    await _notifications.show(
      _nextNotificationId++,
      normalizedTitle.isEmpty ? 'Terminals' : normalizedTitle,
      normalizedBody.isEmpty ? normalizedTitle : normalizedBody,
      _notificationDetails(level),
    );
  } catch (_) {
    // Notification delivery is best-effort on unsupported or uninitialized hosts.
  }
}

Future<void> _ensureInitialized() async {
  if (_initialized) {
    return;
  }

  const darwin = DarwinInitializationSettings(
    requestAlertPermission: false,
    requestBadgePermission: false,
    requestSoundPermission: false,
  );

  const settings = InitializationSettings(
    android: AndroidInitializationSettings('@mipmap/ic_launcher'),
    iOS: darwin,
    macOS: darwin,
    linux: LinuxInitializationSettings(defaultActionName: 'Open notification'),
  );

  await _notifications.initialize(settings);

  await _notifications
      .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin>()
      ?.requestNotificationsPermission();

  await _notifications
      .resolvePlatformSpecificImplementation<
          IOSFlutterLocalNotificationsPlugin>()
      ?.requestPermissions(alert: true, badge: false, sound: true);

  await _notifications
      .resolvePlatformSpecificImplementation<
          MacOSFlutterLocalNotificationsPlugin>()
      ?.requestPermissions(alert: true, badge: false, sound: true);

  _initialized = true;
}

NotificationDetails _notificationDetails(String level) {
  final normalizedLevel = level.trim().toLowerCase();
  final highPriority = normalizedLevel == 'critical' ||
      normalizedLevel == 'error' ||
      normalizedLevel == 'warning';

  return NotificationDetails(
    android: AndroidNotificationDetails(
      'terminals_alerts',
      'Terminals Alerts',
      channelDescription: 'Server-originated alert notifications',
      importance: highPriority ? Importance.max : Importance.defaultImportance,
      priority: highPriority ? Priority.high : Priority.defaultPriority,
    ),
    iOS: const DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: false,
      presentSound: true,
    ),
    macOS: const DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: false,
      presentSound: true,
    ),
    linux: const LinuxNotificationDetails(),
  );
}
