import 'host_state_backend_stub.dart'
    if (dart.library.io) 'host_state_backend_io.dart'
    if (dart.library.html) 'host_state_backend_web.dart';

class EdgeHostFlowState {
  const EdgeHostFlowState({
    required this.flowID,
    this.bundleID,
  });

  final String flowID;
  final String? bundleID;
}

abstract class EdgeHostStateBackend {
  Future<List<EdgeHostFlowState>> load();
  Future<void> save(List<EdgeHostFlowState> flows);
}

EdgeHostStateBackend createEdgeHostStateBackend() =>
    createPlatformEdgeHostStateBackend();
