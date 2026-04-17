import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'bundle_store_backend.dart';

class _IOBundleStoreBackend implements BundleStoreBackend {
  _IOBundleStoreBackend() : _dir = _resolveDir();

  final Directory _dir;

  static Directory _resolveDir() {
    final root =
        Directory('${Directory.systemTemp.path}/terminals_edge_bundles');
    if (!root.existsSync()) {
      root.createSync(recursive: true);
    }
    return root;
  }

  @override
  Map<String, Uint8List> loadAll() {
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
  void put(String bundleId, Uint8List payload) {
    final file = File('${_dir.path}/${Uri.encodeComponent(bundleId)}.b64');
    file.writeAsStringSync(base64Encode(payload), flush: true);
  }

  @override
  void remove(String bundleId) {
    final file = File('${_dir.path}/${Uri.encodeComponent(bundleId)}.b64');
    if (file.existsSync()) {
      file.deleteSync();
    }
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _IOBundleStoreBackend();
