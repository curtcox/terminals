import 'dart:typed_data';

import 'bundle_store.dart';
import 'host_state_backend.dart';
import 'retention.dart';
import 'scheduler.dart';

/// Generic edge operator host lifecycle surface.
class EdgeHost {
  EdgeHost({
    required this.bundleStore,
    required this.scheduler,
    required this.retention,
    EdgeHostStateBackend? stateBackend,
  }) : _stateBackend = stateBackend ?? createEdgeHostStateBackend();

  static Future<EdgeHost> create({
    required BundleStore bundleStore,
    required EdgeScheduler scheduler,
    required RetentionBufferManager retention,
    EdgeHostStateBackend? stateBackend,
  }) async {
    final host = EdgeHost(
      bundleStore: bundleStore,
      scheduler: scheduler,
      retention: retention,
      stateBackend: stateBackend,
    );
    await host._hydrate();
    return host;
  }

  final BundleStore bundleStore;
  final EdgeScheduler scheduler;
  final RetentionBufferManager retention;
  final EdgeHostStateBackend _stateBackend;

  final Set<String> _activeFlows = <String>{};
  final Map<String, String> _flowBundleByID = <String, String>{};

  Set<String> get activeFlows => Set<String>.unmodifiable(_activeFlows);

  String? bundleForFlow(String flowId) => _flowBundleByID[flowId];

  Future<void> installBundle(String bundleId, List<int> payload) async {
    await bundleStore.install(bundleId, Uint8List.fromList(payload));
  }

  Future<void> removeBundle(String bundleId) async {
    await bundleStore.remove(bundleId);
    final affected = _flowBundleByID.entries
        .where((entry) => entry.value == bundleId)
        .map((entry) => entry.key)
        .toList(growable: false);
    for (final flowID in affected) {
      _activeFlows.remove(flowID);
      _flowBundleByID.remove(flowID);
    }
    await _persist();
  }

  Future<void> startFlow(String flowId, {String? bundleId}) async {
    if (bundleId != null && bundleStore.get(bundleId) == null) {
      throw StateError('bundle not installed: $bundleId');
    }
    _activeFlows.add(flowId);
    if (bundleId != null) {
      _flowBundleByID[flowId] = bundleId;
    } else {
      _flowBundleByID.remove(flowId);
    }
    await _persist();
  }

  Future<void> patchFlow(String flowId, {String? bundleId}) async {
    if (!_activeFlows.contains(flowId)) {
      throw StateError('flow not active: $flowId');
    }
    if (bundleId != null && bundleStore.get(bundleId) == null) {
      throw StateError('bundle not installed: $bundleId');
    }
    if (bundleId != null) {
      _flowBundleByID[flowId] = bundleId;
    }
    await _persist();
  }

  Future<void> stopFlow(String flowId) async {
    _activeFlows.remove(flowId);
    _flowBundleByID.remove(flowId);
    await _persist();
  }

  Future<void> _hydrate() async {
    final persisted = await _stateBackend.load();
    final installedBundles = bundleStore.ids.toSet();
    var changed = false;
    for (final flow in persisted) {
      final flowID = flow.flowID.trim();
      if (flowID.isEmpty) {
        changed = true;
        continue;
      }
      final bundleID = flow.bundleID?.trim();
      if (bundleID != null &&
          bundleID.isNotEmpty &&
          !installedBundles.contains(bundleID)) {
        changed = true;
        continue;
      }
      _activeFlows.add(flowID);
      if (bundleID != null && bundleID.isNotEmpty) {
        _flowBundleByID[flowID] = bundleID;
      }
    }
    if (changed) {
      await _persist();
    }
  }

  Future<void> _persist() {
    return _stateBackend.save(
      _activeFlows
          .map(
            (flowID) => EdgeHostFlowState(
              flowID: flowID,
              bundleID: _flowBundleByID[flowID],
            ),
          )
          .toList(growable: false),
    );
  }
}
