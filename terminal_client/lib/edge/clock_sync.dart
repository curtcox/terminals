/// Tracks coarse client/server timing error used by localization flows.
class ClockSyncState {
  ClockSyncState({required this.errorMs, required this.lastSampleUnixMs});

  final double errorMs;
  final int lastSampleUnixMs;
}
