import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/edge/retention.dart';

void main() {
  test('evicts samples outside retention window', () {
    var now = 10 * 1000;
    final retention = RetentionBufferManager(
      audioSec: 5,
      videoSec: 5,
      sensorSec: 5,
      radioSec: 5,
      nowUnixMs: () => now,
    );

    retention.addSample(kind: 'audio', payload: const <int>[1], unixMs: 0);
    retention.addSample(kind: 'audio', payload: const <int>[2], unixMs: 6000);
    retention.addSample(kind: 'audio', payload: const <int>[3], unixMs: 9000);

    expect(retention.sampleCount('audio'), 2);
    expect(
      retention.recentSamples('audio'),
      <List<int>>[
        const <int>[2],
        const <int>[3],
      ],
    );

    now = 14000;
    expect(retention.sampleCount('audio'), 1);
    expect(retention.recentSamples('audio'), <List<int>>[
      const <int>[3]
    ]);
  });
}
