// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:convert';
import 'dart:html' as html;

import 'host_state_backend.dart';

const String _storageKey = 'terminals.edge.host_state';

class _WebEdgeHostStateBackend implements EdgeHostStateBackend {
  @override
  Future<List<EdgeHostFlowState>> load() async {
    final raw = html.window.localStorage[_storageKey];
    if (raw == null || raw.trim().isEmpty) {
      return const <EdgeHostFlowState>[];
    }
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        return const <EdgeHostFlowState>[];
      }
      final flows = decoded['flows'];
      if (flows is! List) {
        return const <EdgeHostFlowState>[];
      }
      final out = <EdgeHostFlowState>[];
      for (final item in flows) {
        if (item is! Map<String, dynamic>) {
          continue;
        }
        final flowID = (item['flow_id'] as String? ?? '').trim();
        if (flowID.isEmpty) {
          continue;
        }
        final rawBundleID = (item['bundle_id'] as String? ?? '').trim();
        out.add(
          EdgeHostFlowState(
            flowID: flowID,
            bundleID: rawBundleID.isEmpty ? null : rawBundleID,
          ),
        );
      }
      return out;
    } catch (_) {
      return const <EdgeHostFlowState>[];
    }
  }

  @override
  Future<void> save(List<EdgeHostFlowState> flows) async {
    final payload = <String, dynamic>{
      'flows': flows
          .map(
            (flow) => <String, dynamic>{
              'flow_id': flow.flowID,
              if (flow.bundleID != null) 'bundle_id': flow.bundleID,
            },
          )
          .toList(growable: false),
    };
    html.window.localStorage[_storageKey] = jsonEncode(payload);
  }
}

EdgeHostStateBackend createPlatformEdgeHostStateBackend() =>
    _WebEdgeHostStateBackend();
