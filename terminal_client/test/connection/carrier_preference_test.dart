import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/carrier_preference.dart';
import 'package:terminal_client/connection/control_client_factory.dart';

void main() {
  test('carrierKindFromPriorityName maps known names and aliases', () {
    expect(carrierKindFromPriorityName('grpc'), ControlCarrierKind.grpc);
    expect(carrierKindFromPriorityName('ws'), ControlCarrierKind.websocket);
    expect(carrierKindFromPriorityName(' websocket '),
        ControlCarrierKind.websocket);
    expect(carrierKindFromPriorityName('tcp'), ControlCarrierKind.tcp);
    expect(carrierKindFromPriorityName('http'), ControlCarrierKind.http);
    expect(carrierKindFromPriorityName('bogus'), isNull);
  });

  test('buildCarrierPreference respects priority and last successful carrier',
      () {
    final ordered = buildCarrierPreference(
      isWebRuntime: false,
      serverPriority: const <String>['http', 'tcp', 'websocket', 'grpc'],
      lastSuccessfulCarrier: ControlCarrierKind.websocket,
    );

    expect(
      ordered,
      <ControlCarrierKind>[
        ControlCarrierKind.websocket,
        ControlCarrierKind.http,
        ControlCarrierKind.tcp,
        ControlCarrierKind.grpc,
      ],
    );
  });

  test('buildCarrierPreference filters unsupported web carriers', () {
    final ordered = buildCarrierPreference(
      isWebRuntime: true,
      serverPriority: const <String>['grpc', 'websocket', 'tcp'],
    );

    expect(ordered, <ControlCarrierKind>[ControlCarrierKind.websocket]);
  });
}
