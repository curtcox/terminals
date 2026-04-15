/// Rolling retention windows for retrospective sensing queries.
class RetentionBufferManager {
  RetentionBufferManager({
    required this.audioSec,
    required this.videoSec,
    required this.sensorSec,
    required this.radioSec,
  });

  final int audioSec;
  final int videoSec;
  final int sensorSec;
  final int radioSec;
}
