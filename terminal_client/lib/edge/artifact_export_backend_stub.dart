import 'dart:typed_data';

import 'artifact_export_backend.dart';

class _MemoryArtifactExportBackend implements ArtifactExportBackend {
  final Map<String, Uint8List> _storage = <String, Uint8List>{};

  @override
  Future<Uint8List?> read(String artifactId) async => _storage[artifactId];

  @override
  Future<void> write(String artifactId, Uint8List payload) async {
    _storage[artifactId] = payload;
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _MemoryArtifactExportBackend();
