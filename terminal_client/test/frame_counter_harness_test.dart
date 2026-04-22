import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/testing/frame_counter_harness.dart';

void main() {
  test('encodes and decodes harness frame counter prefix', () {
    final encoded = encodeHarnessFrameCounterPayload(
      counter: 42,
      payload: const <int>[0x10, 0x20, 0x30],
    );

    final decoded = decodeHarnessFrameCounterPayload(encoded);
    expect(decoded, isNotNull);
    expect(decoded!.counter, 42);
    expect(decoded.payload, const <int>[0x10, 0x20, 0x30]);
  });

  test('decode returns null when payload is shorter than 8-byte prefix', () {
    final decoded = decodeHarnessFrameCounterPayload(const <int>[0x10, 0x20]);
    expect(decoded, isNull);
  });

  test('stamper emits monotonic counters starting at one', () {
    final stamper = HarnessFrameCounterStamper();

    final first = decodeHarnessFrameCounterPayload(
      stamper.stamp(const <int>[0x01]),
    );
    final second = decodeHarnessFrameCounterPayload(
      stamper.stamp(const <int>[0x02]),
    );

    expect(first, isNotNull);
    expect(second, isNotNull);
    expect(first!.counter, 1);
    expect(second!.counter, 2);
    expect(first.payload, const <int>[0x01]);
    expect(second.payload, const <int>[0x02]);
  });
}
