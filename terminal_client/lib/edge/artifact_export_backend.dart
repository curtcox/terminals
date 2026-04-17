import 'dart:typed_data';

import 'artifact_export_backend_stub.dart'
    if (dart.library.io) 'artifact_export_backend_io.dart'
    if (dart.library.html) 'artifact_export_backend_web.dart';

abstract class ArtifactExportBackend {
  Future<Uint8List?> read(String artifactId);
  Future<void> write(String artifactId, Uint8List payload);
}

ArtifactExportBackend createArtifactExportBackend() =>
    createPlatformArtifactExportBackend();
