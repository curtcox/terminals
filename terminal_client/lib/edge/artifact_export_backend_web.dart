// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:convert';
import 'dart:html' as html;
import 'dart:typed_data';

import 'artifact_export_backend.dart';

const String _artifactPrefix = 'terminals.artifact.';

class _WebArtifactExportBackend implements ArtifactExportBackend {
  @override
  Uint8List? read(String artifactId) {
    final encoded = html.window.localStorage['$_artifactPrefix$artifactId'];
    if (encoded == null || encoded.isEmpty) {
      return null;
    }
    try {
      return base64Decode(encoded);
    } catch (_) {
      return null;
    }
  }

  @override
  void write(String artifactId, Uint8List payload) {
    html.window.localStorage['$_artifactPrefix$artifactId'] =
        base64Encode(payload);
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _WebArtifactExportBackend();
