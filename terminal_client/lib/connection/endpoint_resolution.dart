const Set<String> _loopbackHosts = <String>{
  'localhost',
  '127.0.0.1',
  '::1',
};

String resolveInitialControlHost({
  required bool isWebRuntime,
  required String configuredHost,
  required String pageHost,
}) {
  final trimmedConfiguredHost = configuredHost.trim();
  if (!isWebRuntime) {
    return trimmedConfiguredHost;
  }
  final trimmedPageHost = pageHost.trim();
  if (trimmedPageHost.isEmpty) {
    return trimmedConfiguredHost;
  }
  if (trimmedConfiguredHost.isEmpty) {
    return trimmedPageHost;
  }
  if (_loopbackHosts.contains(trimmedConfiguredHost.toLowerCase())) {
    return trimmedPageHost;
  }
  return trimmedConfiguredHost;
}

String resolvePageHost({
  required String browserLocationHost,
  required String uriBaseHost,
}) {
  final fromLocation = browserLocationHost.trim();
  if (fromLocation.isNotEmpty) {
    return fromLocation;
  }
  return uriBaseHost.trim();
}

String resolvePreferredEndpoint({
  required String manualEndpoint,
  required String discoveredEndpoint,
}) {
  final manual = manualEndpoint.trim();
  if (manual.isNotEmpty) {
    return manual;
  }
  return discoveredEndpoint.trim();
}

String websocketPathFromEndpoint(String endpoint) {
  final trimmed = endpoint.trim();
  if (trimmed.isEmpty) {
    return '/control';
  }
  final parsed = Uri.tryParse(trimmed);
  if (parsed == null || !parsed.hasScheme || parsed.host.isEmpty) {
    return '/control';
  }
  if (parsed.path.trim().isEmpty) {
    return '/control';
  }
  return parsed.path;
}
