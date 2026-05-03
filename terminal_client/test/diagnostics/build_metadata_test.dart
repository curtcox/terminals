import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/diagnostics/build_metadata.dart';

void main() {
  test('buildMetadataLabel renders date and sha', () {
    final label = buildMetadataLabel(
      buildDate: '2026-04-21T14:20:00Z',
      buildSha: 'abc123def456',
    );

    expect(label, 'Build: 2026-04-21T14:20:00Z | SHA: abc123def456');
  });

  test('buildVersionParityNote reports same SHA', () {
    final note = buildVersionParityNote(
      clientBuildDate: '2026-04-21T14:55:56Z',
      clientBuildSha: 'ea99b3f38658',
      serverBuildDate: '2026-04-21T14:56:01Z',
      serverBuildSha: 'ea99b3f38658',
    );

    expect(note, 'Build Match: same SHA, different build date');
  });

  test('buildVersionParityNote reports different SHA', () {
    final note = buildVersionParityNote(
      clientBuildDate: '2026-04-21T14:55:56Z',
      clientBuildSha: 'ea99b3f38658',
      serverBuildDate: '2026-04-21T14:56:01Z',
      serverBuildSha: 'server-sha',
    );

    expect(note, 'Build Match: different SHA');
  });

  test('buildServerBuildLine reports awaiting register ack before connect', () {
    final line = buildServerBuildLine(
      serverBuildDate: 'unknown',
      serverBuildSha: 'unknown',
      hasRegisterAck: false,
    );

    expect(line, 'Server Build: awaiting register ack');
  });

  test('buildWebConnectionChipLabel reports not connected before stream starts',
      () {
    final label = buildWebConnectionChipLabel(
      hasRegisterAck: false,
      isConnecting: false,
      shouldStayConnected: false,
    );

    expect(label, 'Not connected');
  });
}
