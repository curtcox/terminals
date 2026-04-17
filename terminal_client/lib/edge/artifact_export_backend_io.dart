import 'dart:io';
import 'dart:typed_data';

import 'artifact_export_backend.dart';

class _IOArtifactExportBackend implements ArtifactExportBackend {
  _IOArtifactExportBackend() : _dir = _resolveDir();

  final Directory _dir;

  static Directory _resolveDir() {
    final root =
        Directory('${Directory.systemTemp.path}/terminals_edge_artifacts');
    if (!root.existsSync()) {
      root.createSync(recursive: true);
    }
    return root;
  }

  @override
  Uint8List? read(String artifactId) {
    final file = File('${_dir.path}/${Uri.encodeComponent(artifactId)}.bin');
    if (!file.existsSync()) {
      return null;
    }
    return file.readAsBytesSync();
  }

  @override
  void write(String artifactId, Uint8List payload) {
    final file = File('${_dir.path}/${Uri.encodeComponent(artifactId)}.bin');
    file.writeAsBytesSync(payload, flush: true);
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _IOArtifactExportBackend();
