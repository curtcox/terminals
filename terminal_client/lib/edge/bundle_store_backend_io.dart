import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'bundle_store_backend.dart';

class _IOBundleStoreBackend implements BundleStoreBackend {
  _IOBundleStoreBackend() : _dir = _resolveDir();

  final Directory _dir;

  static Directory _resolveDir() {
    final root = Directory('${_edgeStorageRootDir().path}/bundles');
    if (!root.existsSync()) {
      root.createSync(recursive: true);
    }
    return root;
  }

  @override
  Future<Map<String, Uint8List>> loadAll() async {
    final out = <String, Uint8List>{};
    if (!_dir.existsSync()) {
      return out;
    }
    for (final entity in _dir.listSync()) {
      if (entity is! File || !entity.path.endsWith('.b64')) {
        continue;
      }
      final id = entity.uri.pathSegments.isNotEmpty
          ? entity.uri.pathSegments.last.replaceAll('.b64', '')
          : '';
      if (id.isEmpty) {
        continue;
      }
      try {
        final encoded = entity.readAsStringSync();
        out[id] = base64Decode(encoded);
      } catch (_) {
        // Ignore one corrupt bundle and keep loading others.
      }
    }
    return out;
  }

  @override
  Future<void> put(String bundleId, Uint8List payload) async {
    final file = File('${_dir.path}/${Uri.encodeComponent(bundleId)}.b64');
    file.writeAsStringSync(base64Encode(payload), flush: true);
  }

  @override
  Future<void> remove(String bundleId) async {
    final file = File('${_dir.path}/${Uri.encodeComponent(bundleId)}.b64');
    if (file.existsSync()) {
      file.deleteSync();
    }
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _IOBundleStoreBackend();

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
