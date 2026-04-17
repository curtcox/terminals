import 'dart:typed_data';

import 'artifact_export_backend.dart';

class _MemoryArtifactExportBackend implements ArtifactExportBackend {
  final Map<String, Uint8List> _storage = <String, Uint8List>{};

  @override
  Uint8List? read(String artifactId) => _storage[artifactId];

  @override
  void write(String artifactId, Uint8List payload) {
    _storage[artifactId] = payload;
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _MemoryArtifactExportBackend();
