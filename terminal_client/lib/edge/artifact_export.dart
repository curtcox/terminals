import 'dart:typed_data';

import 'artifact_export_backend.dart';

/// Artifact materialization contract for clip/frame/timeseries exports.
abstract class ArtifactExporter {
  Future<Uint8List> exportByID(String artifactId);
}

class DurableArtifactExporter implements ArtifactExporter {
  DurableArtifactExporter({ArtifactExportBackend? backend})
      : _backend = backend ?? createArtifactExportBackend();

  final ArtifactExportBackend _backend;

  @override
  Future<Uint8List> exportByID(String artifactId) async {
    final payload = _backend.read(artifactId);
    if (payload != null) {
      return payload;
    }
    throw StateError('artifact not found: $artifactId');
  }

  Future<void> save(String artifactId, Uint8List payload) async {
    _backend.write(artifactId, payload);
  }
}
