import 'dart:typed_data';

/// Artifact materialization contract for clip/frame/timeseries exports.
abstract class ArtifactExporter {
  Future<Uint8List> exportByID(String artifactId);
}
