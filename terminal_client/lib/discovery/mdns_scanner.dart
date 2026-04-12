import 'dart:async';

import 'package:multicast_dns/multicast_dns.dart';

class DiscoveredServer {
  DiscoveredServer({
    required this.name,
    required this.host,
    required this.port,
  });

  final String name;
  final String host;
  final int port;
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
          final key = '${srv.target}:${srv.port}';
          discovered[key] = DiscoveredServer(
            name: _instanceName(srvName),
            host: _normalizeHost(srv.target),
            port: srv.port,
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
}
