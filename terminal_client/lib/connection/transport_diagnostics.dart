String classifyCarrierFailure({
  required String stage,
  required String rawError,
}) {
  final lower = rawError.toLowerCase();
  final trimmedStage = stage.trim().toLowerCase();

  if (lower.contains('unsupported_protocol_version') ||
      lower.contains('unsupported protocol version') ||
      lower.contains('transport hello rejected protocol version')) {
    return 'protocol_version';
  }
  if (lower.contains('unsupported_carrier') ||
      lower.contains('not declared in transport hello')) {
    return 'carrier_mismatch';
  }
  if (lower.contains('failed host lookup') ||
      lower.contains('name or service not known') ||
      lower.contains('nodename nor servname provided')) {
    return 'dns';
  }
  if (lower.contains('connection refused')) {
    return 'tcp_connect';
  }
  if (lower.contains('certificate') ||
      lower.contains('tls') ||
      lower.contains('ssl') ||
      lower.contains('handshake')) {
    return 'tls_or_handshake';
  }
  if (lower.contains('upgrade rejected') || lower.contains('http 403')) {
    return 'upgrade_rejected';
  }
  if (lower.contains('timed out') || lower.contains('timeout')) {
    return 'timeout';
  }
  if (trimmedStage == 'stream_closed' || lower.contains('stream closed')) {
    return 'stream_closed';
  }
  if (trimmedStage == 'connect') {
    return 'connect_error';
  }
  if (trimmedStage == 'stream') {
    return 'stream_error';
  }
  return 'unknown';
}

class TransportErrorDiagnosis {
  const TransportErrorDiagnosis({
    required this.summary,
    required this.guidance,
    required this.grpcCode,
    required this.grpcCodeName,
    required this.rawError,
  });

  final String summary;
  final String guidance;
  final int? grpcCode;
  final String grpcCodeName;
  final String rawError;

  bool get hasSummary => summary.isNotEmpty;

  String statusText() {
    if (hasSummary) {
      return 'Stream error: $summary';
    }
    return 'Stream error: $rawError';
  }

  String notificationText() {
    if (!hasSummary) {
      return '';
    }
    if (guidance.isEmpty) {
      return summary;
    }
    return '$summary $guidance';
  }
}

TransportErrorDiagnosis diagnoseTransportError(
  Object error, {
  required bool isWeb,
}) {
  final raw = error.toString();
  final lower = raw.toLowerCase();
  final grpcCodeMatch =
      RegExp(r'code:\s*([0-9]+)', caseSensitive: false).firstMatch(raw);
  final grpcCode =
      grpcCodeMatch == null ? null : int.tryParse(grpcCodeMatch.group(1) ?? '');
  final grpcCodeNameMatch =
      RegExp(r'codeName:\s*([A-Z_]+)', caseSensitive: false).firstMatch(raw);
  final grpcCodeName = (grpcCodeNameMatch?.group(1) ?? '').trim().toUpperCase();
  final isGrpcError = lower.contains('grpc error');
  final isUnavailable = grpcCode == 14 ||
      grpcCodeName == 'UNAVAILABLE' ||
      lower.contains('unavailable');
  final hasSocketConstructorFailure =
      lower.contains('unsupported operation: socket constructor');

  if (isGrpcError && isUnavailable && hasSocketConstructorFailure && isWeb) {
    return TransportErrorDiagnosis(
      summary: 'gRPC UNAVAILABLE (14)',
      guidance:
          'Browser runtime cannot open raw gRPC sockets. Configure gRPC-Web via an HTTP proxy (for example Envoy) or use a non-web client target.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  if (isGrpcError && isUnavailable) {
    return TransportErrorDiagnosis(
      summary: 'gRPC UNAVAILABLE (14)',
      guidance:
          'Server is unreachable or transport is unavailable. Verify host/port, server process, and network/proxy configuration.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  if (isGrpcError && grpcCode != null) {
    final displayName = grpcCodeName.isEmpty ? '' : ' ($grpcCodeName)';
    return TransportErrorDiagnosis(
      summary: 'gRPC error $grpcCode$displayName',
      guidance: 'Check server logs and client/server protocol compatibility.',
      grpcCode: grpcCode,
      grpcCodeName: grpcCodeName,
      rawError: raw,
    );
  }

  return TransportErrorDiagnosis(
    summary: '',
    guidance: '',
    grpcCode: grpcCode,
    grpcCodeName: grpcCodeName,
    rawError: raw,
  );
}
