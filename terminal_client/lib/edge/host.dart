import 'bundle_store.dart';
import 'retention.dart';
import 'scheduler.dart';

/// Generic edge operator host lifecycle surface.
class EdgeHost {
  EdgeHost({
    required this.bundleStore,
    required this.scheduler,
    required this.retention,
  });

  final BundleStore bundleStore;
  final EdgeScheduler scheduler;
  final RetentionBufferManager retention;

  final Set<String> _activeFlows = <String>{};

  Set<String> get activeFlows => Set<String>.unmodifiable(_activeFlows);

  void startFlow(String flowId) {
    _activeFlows.add(flowId);
  }

  void stopFlow(String flowId) {
    _activeFlows.remove(flowId);
  }
}
