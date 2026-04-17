import 'dart:io';
import 'dart:typed_data';

import 'artifact_export_backend.dart';

class _IOArtifactExportBackend implements ArtifactExportBackend {
  _IOArtifactExportBackend() : _dir = _resolveDir();

  final Directory _dir;

  static Directory _resolveDir() {
    final root = Directory('${_edgeStorageRootDir().path}/artifacts');
    if (!root.existsSync()) {
      root.createSync(recursive: true);
    }
    return root;
  }

  @override
  Future<Uint8List?> read(String artifactId) async {
    final file = File('${_dir.path}/${Uri.encodeComponent(artifactId)}.bin');
    if (!file.existsSync()) {
      return null;
    }
    return file.readAsBytesSync();
  }

  @override
  Future<void> write(String artifactId, Uint8List payload) async {
    final file = File('${_dir.path}/${Uri.encodeComponent(artifactId)}.bin');
    file.writeAsBytesSync(payload, flush: true);
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _IOArtifactExportBackend();

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
