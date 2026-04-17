/// Rolling retention windows for retrospective sensing queries.
class RetentionBufferManager {
  RetentionBufferManager({
    required this.audioSec,
    required this.videoSec,
    required this.sensorSec,
    required this.radioSec,
    int Function()? nowUnixMs,
  }) : _nowUnixMs =
            nowUnixMs ?? (() => DateTime.now().toUtc().millisecondsSinceEpoch);

  final int Function() _nowUnixMs;

  final int audioSec;
  final int videoSec;
  final int sensorSec;
  final int radioSec;

  final Map<String, List<_RetentionSample>> _samplesByKind =
      <String, List<_RetentionSample>>{};

  void addSample({
    required String kind,
    required List<int> payload,
    int? unixMs,
  }) {
    final timestamp = unixMs ?? _nowUnixMs();
    final bucket = _samplesByKind.putIfAbsent(kind, () => <_RetentionSample>[]);
    bucket.add(
      _RetentionSample(
        unixMs: timestamp,
        payload: List<int>.from(payload, growable: false),
      ),
    );
    _evict(kind);
  }

  List<List<int>> recentSamples(String kind) {
    _evict(kind);
    final bucket = _samplesByKind[kind] ?? const <_RetentionSample>[];
    return bucket
        .map((item) => List<int>.from(item.payload, growable: false))
        .toList(growable: false);
  }

  int sampleCount(String kind) {
    _evict(kind);
    return (_samplesByKind[kind] ?? const <_RetentionSample>[]).length;
  }

  void _evict(String kind) {
    final bucket = _samplesByKind[kind];
    if (bucket == null || bucket.isEmpty) {
      return;
    }
    final retentionMs = _retentionWindowMs(kind);
    if (retentionMs <= 0) {
      bucket.clear();
      return;
    }
    final threshold = _nowUnixMs() - retentionMs;
    var firstKept = 0;
    while (firstKept < bucket.length && bucket[firstKept].unixMs < threshold) {
      firstKept += 1;
    }
    if (firstKept > 0) {
      bucket.removeRange(0, firstKept);
    }
  }

  int _retentionWindowMs(String kind) {
    switch (kind) {
      case 'audio':
        return audioSec * 1000;
      case 'video':
        return videoSec * 1000;
      case 'sensor':
        return sensorSec * 1000;
      case 'radio':
        return radioSec * 1000;
      default:
        return 0;
    }
  }
}

class _RetentionSample {
  const _RetentionSample({
    required this.unixMs,
    required this.payload,
  });

  final int unixMs;
  final List<int> payload;
}
