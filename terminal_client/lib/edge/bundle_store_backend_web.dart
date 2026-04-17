// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:convert';
import 'dart:html' as html;
import 'dart:typed_data';

import 'bundle_store_backend.dart';

const String _bundlePrefix = 'terminals.bundle.';

class _WebBundleStoreBackend implements BundleStoreBackend {
  @override
  Map<String, Uint8List> loadAll() {
    final out = <String, Uint8List>{};
    final storage = html.window.localStorage;
    for (final key in storage.keys) {
      if (!key.startsWith(_bundlePrefix)) {
        continue;
      }
      final encoded = storage[key];
      if (encoded == null || encoded.isEmpty) {
        continue;
      }
      final id = key.substring(_bundlePrefix.length);
      try {
        out[id] = base64Decode(encoded);
      } catch (_) {
        // Keep going if one bundle payload is malformed.
      }
    }
    return out;
  }

  @override
  void put(String bundleId, Uint8List payload) {
    html.window.localStorage['$_bundlePrefix$bundleId'] = base64Encode(payload);
  }

  @override
  void remove(String bundleId) {
    html.window.localStorage.remove('$_bundlePrefix$bundleId');
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _WebBundleStoreBackend();
