import 'dart:convert';
import 'dart:io';

import 'host_state_backend.dart';

class _IOEdgeHostStateBackend implements EdgeHostStateBackend {
  _IOEdgeHostStateBackend({Directory? rootDir})
      : _stateFile = File('${_resolveRootDir(rootDir).path}/host_state.json');

  final File _stateFile;

  static Directory _resolveRootDir(Directory? rootDir) {
    final root = rootDir ?? _edgeStorageRootDir();
    if (!root.existsSync()) {
      root.createSync(recursive: true);
    }
    return root;
  }

  @override
  Future<List<EdgeHostFlowState>> load() async {
    if (!_stateFile.existsSync()) {
      return const <EdgeHostFlowState>[];
    }
    try {
      final decoded = jsonDecode(_stateFile.readAsStringSync());
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
      // Ignore one corrupt snapshot and start from an empty state.
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
    _stateFile.writeAsStringSync(jsonEncode(payload), flush: true);
  }
}

EdgeHostStateBackend createPlatformEdgeHostStateBackend() =>
    _IOEdgeHostStateBackend();

EdgeHostStateBackend createIOEdgeHostStateBackend({Directory? rootDir}) =>
    _IOEdgeHostStateBackend(rootDir: rootDir);

Directory _edgeStorageRootDir() {
  final home = Platform.environment['HOME'] ?? Directory.systemTemp.path;
  if (Platform.isMacOS) {
    return Directory('$home/Library/Application Support/Terminals/edge');
  }
  if (Platform.isWindows) {
    final appData = Platform.environment['APPDATA'];
    if (appData != null && appData.isNotEmpty) {
      return Directory('$appData\\Terminals\\edge');
    }
  }
  if (Platform.isLinux) {
    final xdgData = Platform.environment['XDG_DATA_HOME'];
    if (xdgData != null && xdgData.isNotEmpty) {
      return Directory('$xdgData/terminals/edge');
    }
    return Directory('$home/.local/share/terminals/edge');
  }
  return Directory('${Directory.systemTemp.path}/terminals/edge');
}
