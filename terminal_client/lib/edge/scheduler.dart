/// Simple admission control model for edge flows.
class EdgeScheduler {
  EdgeScheduler({
    required this.maxCPURealtime,
    required this.maxMemoryMB,
  });

  final int maxCPURealtime;
  final int maxMemoryMB;
  int _allocatedCPURealtime = 0;
  int _allocatedMemoryMB = 0;

  bool canAdmit({required int cpuRealtime, required int memoryMB}) {
    return _allocatedCPURealtime + cpuRealtime <= maxCPURealtime &&
        _allocatedMemoryMB + memoryMB <= maxMemoryMB;
  }

  bool admit({required int cpuRealtime, required int memoryMB}) {
    if (!canAdmit(cpuRealtime: cpuRealtime, memoryMB: memoryMB)) {
      return false;
    }
    _allocatedCPURealtime += cpuRealtime;
    _allocatedMemoryMB += memoryMB;
    return true;
  }

  void release({required int cpuRealtime, required int memoryMB}) {
    _allocatedCPURealtime =
        (_allocatedCPURealtime - cpuRealtime).clamp(0, maxCPURealtime);
    _allocatedMemoryMB = (_allocatedMemoryMB - memoryMB).clamp(0, maxMemoryMB);
  }
}
