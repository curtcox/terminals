import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/util/alert_delivery.dart';

void main() {
  test('delivers notification and speaks body text', () async {
    final shown = <String>[];
    final spoken = <String>[];
    final service = AlertDeliveryService(
      notificationDispatcher: ({
        required String title,
        required String body,
        required String level,
      }) async {
        shown.add('$title|$body|$level');
      },
      speaker: spoken.add,
    );

    service.deliver(
      title: 'Timer',
      body: 'Dishwasher finished',
      level: 'info',
    );
    await Future<void>.delayed(Duration.zero);

    expect(shown, <String>['Timer|Dishwasher finished|info']);
    expect(spoken, <String>['Dishwasher finished']);
  });

  test('uses title as spoken fallback when body is empty', () async {
    final spoken = <String>[];
    final service = AlertDeliveryService(
      notificationDispatcher: ({
        required String title,
        required String body,
        required String level,
      }) async {},
      speaker: spoken.add,
    );

    service.deliver(title: 'Doorbell', body: '  ', level: 'warning');
    await Future<void>.delayed(Duration.zero);

    expect(spoken, <String>['Doorbell']);
  });

  test('ignores delivery when both title and body are empty', () async {
    var notificationCalls = 0;
    var speechCalls = 0;
    final service = AlertDeliveryService(
      notificationDispatcher: ({
        required String title,
        required String body,
        required String level,
      }) async {
        notificationCalls += 1;
      },
      speaker: (_) {
        speechCalls += 1;
      },
    );

    service.deliver(title: '  ', body: ' ', level: 'info');
    await Future<void>.delayed(Duration.zero);

    expect(notificationCalls, 0);
    expect(speechCalls, 0);
  });

  test('swallows notification errors and still speaks', () async {
    final spoken = <String>[];
    final service = AlertDeliveryService(
      notificationDispatcher: ({
        required String title,
        required String body,
        required String level,
      }) async {
        throw StateError('notification channel unavailable');
      },
      speaker: spoken.add,
    );

    service.deliver(title: 'Alert', body: 'Fallback speech', level: 'error');
    await Future<void>.delayed(Duration.zero);

    expect(spoken, <String>['Fallback speech']);
  });
}
