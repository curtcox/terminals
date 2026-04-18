import 'dart:async';

import 'package:multicast_dns/multicast_dns.dart';

class DiscoveredServer {
  DiscoveredServer({
    required this.name,
    required this.host,
    required this.port,
    required this.grpcEndpoint,
    required this.websocketEndpoint,
    required this.tcpEndpoint,
    required this.httpEndpoint,
    required this.priority,
  });

  final String name;
  final String host;
  final int port;
  final String grpcEndpoint;
  final String websocketEndpoint;
  final String tcpEndpoint;
  final String httpEndpoint;
  final List<String> priority;
}

class MdnsScanner {
  MdnsScanner({this.serviceType = '_terminals._tcp.local'});

  final String serviceType;

  Future<List<DiscoveredServer>> scan({
    Duration timeout = const Duration(seconds: 2),
  }) async {
    final client = MDnsClient();
    final discovered = <String, DiscoveredServer>{};

    await client.start();
    try {
      final ptrQuery = client.lookup<PtrResourceRecord>(
        ResourceRecordQuery.serverPointer(serviceType),
      );

      final ptrDeadline = DateTime.now().add(timeout);
      await for (final ptr in ptrQuery.timeout(timeout, onTimeout: (_) {})) {
        final srvName = ptr.domainName;
        final srvQuery = client.lookup<SrvResourceRecord>(
          ResourceRecordQuery.service(srvName),
        );

        final remaining = ptrDeadline.difference(DateTime.now());
        if (remaining <= Duration.zero) {
          break;
        }
        await for (final srv
            in srvQuery.timeout(remaining, onTimeout: (_) {})) {
          final txt = await _loadTxtMetadata(
            client: client,
            serviceName: srvName,
            timeout: remaining,
          );
          final host = _normalizeHost(srv.target);
          final grpcEndpoint = _nonEmptyOrFallback(
            txt['grpc'],
            '$host:${srv.port}',
          );
          final websocketEndpoint = _nonEmptyOrFallback(
            txt['ws'],
            'ws://$host:50054/control',
          );
          final tcpEndpoint = _nonEmptyOrFallback(
            txt['tcp'],
            '$host:50055',
          );
          final httpEndpoint = _nonEmptyOrFallback(
            txt['http'],
            'http://$host:50056',
          );
          final priority = _parsePriority(txt['priority']);
          final key = '${srv.target}:${srv.port}';
          discovered[key] = DiscoveredServer(
            name: _instanceName(srvName),
            host: host,
            port: srv.port,
            grpcEndpoint: grpcEndpoint,
            websocketEndpoint: websocketEndpoint,
            tcpEndpoint: tcpEndpoint,
            httpEndpoint: httpEndpoint,
            priority: priority,
          );
        }
      }
    } finally {
      client.stop();
    }

    final out = discovered.values.toList()
      ..sort((a, b) => a.name.compareTo(b.name));
    return out;
  }

  String _instanceName(String srvName) {
    final firstDot = srvName.indexOf('.');
    if (firstDot <= 0) {
      return srvName;
    }
    return srvName.substring(0, firstDot);
  }

  String _normalizeHost(String host) {
    if (host.endsWith('.')) {
      return host.substring(0, host.length - 1);
    }
    return host;
  }

  Future<Map<String, String>> _loadTxtMetadata({
    required MDnsClient client,
    required String serviceName,
    required Duration timeout,
  }) async {
    final query = client.lookup<TxtResourceRecord>(
      ResourceRecordQuery.text(serviceName),
    );
    final metadata = <String, String>{};
    await for (final txt in query.timeout(timeout, onTimeout: (_) {})) {
      final value = txt.text.trim();
      if (value.isEmpty) {
        continue;
      }
      final eq = value.indexOf('=');
      if (eq <= 0 || eq == value.length - 1) {
        continue;
      }
      final key = value.substring(0, eq).trim().toLowerCase();
      final textValue = value.substring(eq + 1).trim();
      if (key.isEmpty || textValue.isEmpty) {
        continue;
      }
      metadata[key] = textValue;
    }
    return metadata;
  }

  String _nonEmptyOrFallback(String? value, String fallback) {
    final trimmed = value?.trim() ?? '';
    if (trimmed.isNotEmpty) {
      return trimmed;
    }
    return fallback;
  }

  List<String> _parsePriority(String? raw) {
    final value = raw?.trim() ?? '';
    if (value.isEmpty) {
      return const <String>['grpc', 'websocket', 'tcp', 'http'];
    }
    final out = <String>[];
    for (final part in value.split(',')) {
      final normalized = part.trim().toLowerCase();
      if (normalized.isEmpty) {
        continue;
      }
      out.add(normalized);
    }
    if (out.isEmpty) {
      return const <String>['grpc', 'websocket', 'tcp', 'http'];
    }
    return out;
  }
}
