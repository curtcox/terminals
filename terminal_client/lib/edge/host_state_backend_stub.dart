import 'host_state_backend.dart';

class _MemoryEdgeHostStateBackend implements EdgeHostStateBackend {
  List<EdgeHostFlowState> _flows = const <EdgeHostFlowState>[];

  @override
  Future<List<EdgeHostFlowState>> load() async {
    return List<EdgeHostFlowState>.from(_flows, growable: false);
  }

  @override
  Future<void> save(List<EdgeHostFlowState> flows) async {
    _flows = List<EdgeHostFlowState>.from(flows, growable: false);
  }
}

EdgeHostStateBackend createPlatformEdgeHostStateBackend() =>
    _MemoryEdgeHostStateBackend();
