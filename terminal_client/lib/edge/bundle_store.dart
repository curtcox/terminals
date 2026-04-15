import 'dart:typed_data';

/// In-memory bundle index used by the edge host scaffold.
class BundleStore {
  final Map<String, Uint8List> _bundles = <String, Uint8List>{};

  Iterable<String> get ids => _bundles.keys;

  void install(String bundleId, Uint8List payload) {
    _bundles[bundleId] = payload;
  }

  void remove(String bundleId) {
    _bundles.remove(bundleId);
  }

  Uint8List? get(String bundleId) => _bundles[bundleId];
}
