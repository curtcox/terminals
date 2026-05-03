import 'package:terminal_client/connection/control_client_factory.dart';

ControlCarrierKind? carrierKindFromPriorityName(String raw) {
  switch (raw.trim().toLowerCase()) {
    case 'grpc':
      return ControlCarrierKind.grpc;
    case 'websocket':
    case 'ws':
      return ControlCarrierKind.websocket;
    case 'tcp':
      return ControlCarrierKind.tcp;
    case 'http':
      return ControlCarrierKind.http;
    default:
      return null;
  }
}

bool isCarrierSupportedOnRuntime(
  ControlCarrierKind carrier, {
  required bool isWebRuntime,
}) {
  if (isWebRuntime) {
    return carrier == ControlCarrierKind.websocket;
  }
  return true;
}

List<ControlCarrierKind> buildCarrierPreference({
  required bool isWebRuntime,
  required List<String> serverPriority,
  ControlCarrierKind? lastSuccessfulCarrier,
}) {
  final defaults = isWebRuntime
      ? const <ControlCarrierKind>[ControlCarrierKind.websocket]
      : const <ControlCarrierKind>[
          ControlCarrierKind.grpc,
          ControlCarrierKind.websocket,
          ControlCarrierKind.tcp,
          ControlCarrierKind.http,
        ];
  if (serverPriority.isEmpty) {
    return defaults;
  }

  final ordered = <ControlCarrierKind>[];
  for (final raw in serverPriority) {
    final carrier = carrierKindFromPriorityName(raw);
    if (carrier == null) {
      continue;
    }
    if (isWebRuntime && carrier != ControlCarrierKind.websocket) {
      continue;
    }
    if (!ordered.contains(carrier)) {
      ordered.add(carrier);
    }
  }

  final preferred = ordered.isEmpty ? defaults : ordered;
  final filtered = preferred
      .where(
        (carrier) =>
            isCarrierSupportedOnRuntime(carrier, isWebRuntime: isWebRuntime),
      )
      .toSet()
      .toList(growable: true);
  if (filtered.isEmpty) {
    return defaults
        .where(
          (carrier) =>
              isCarrierSupportedOnRuntime(carrier, isWebRuntime: isWebRuntime),
        )
        .toList(growable: false);
  }
  if (lastSuccessfulCarrier != null &&
      filtered.contains(lastSuccessfulCarrier)) {
    filtered
      ..remove(lastSuccessfulCarrier)
      ..insert(0, lastSuccessfulCarrier);
  }
  return filtered;
}
