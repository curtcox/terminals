import 'dart:typed_data';

const int _harnessFrameCounterPrefixBytes = 8;

class DecodedHarnessFrameCounterPayload {
  const DecodedHarnessFrameCounterPayload({
    required this.counter,
    required this.payload,
  });

  final int counter;
  final List<int> payload;
}

class HarnessFrameCounterStamper {
  int _nextCounter = 0;

  List<int> stamp(List<int> payload) {
    _nextCounter += 1;
    return encodeHarnessFrameCounterPayload(
      counter: _nextCounter,
      payload: payload,
    );
  }
}

List<int> encodeHarnessFrameCounterPayload({
  required int counter,
  required List<int> payload,
}) {
  final prefix = ByteData(_harnessFrameCounterPrefixBytes)
    ..setUint64(0, counter, Endian.big);
  return <int>[
    ...prefix.buffer.asUint8List(),
    ...payload,
  ];
}

DecodedHarnessFrameCounterPayload? decodeHarnessFrameCounterPayload(
  List<int> frame,
) {
  if (frame.length < _harnessFrameCounterPrefixBytes) {
    return null;
  }
  final typed = Uint8List.fromList(frame);
  final view = ByteData.sublistView(typed);
  final counter = view.getUint64(0, Endian.big);
  return DecodedHarnessFrameCounterPayload(
    counter: counter,
    payload: typed.sublist(_harnessFrameCounterPrefixBytes),
  );
}
