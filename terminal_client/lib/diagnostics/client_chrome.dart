import 'package:flutter/material.dart';
import 'package:terminal_client/connection/reliability.dart';
import 'package:terminal_client/diagnostics/bug_report_chrome.dart';
import 'package:terminal_client/diagnostics/build_metadata.dart';
import 'package:terminal_client/diagnostics/diagnostic_clipboard.dart';

class ClientMetadataFooter extends StatelessWidget {
  const ClientMetadataFooter({
    super.key,
    required this.buildDate,
    required this.buildSha,
  });

  final String buildDate;
  final String buildSha;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.end,
          mainAxisSize: MainAxisSize.max,
          children: [
            Flexible(
              child: SelectableText(
                buildMetadataLabel(buildDate: buildDate, buildSha: buildSha),
                textAlign: TextAlign.right,
                style: TextStyle(fontSize: 11, color: Colors.grey.shade700),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class BuildParityPanel extends StatelessWidget {
  const BuildParityPanel({
    super.key,
    required this.clientBuildDate,
    required this.clientBuildSha,
    required this.serverBuildDate,
    required this.serverBuildSha,
    required this.hasRegisterAck,
  });

  final String clientBuildDate;
  final String clientBuildSha;
  final String serverBuildDate;
  final String serverBuildSha;
  final bool hasRegisterAck;

  @override
  Widget build(BuildContext context) {
    final clientLabel =
        'Client ${buildMetadataLabel(buildDate: clientBuildDate, buildSha: clientBuildSha)}';
    final serverLabel = buildServerBuildLine(
      serverBuildDate: serverBuildDate,
      serverBuildSha: serverBuildSha,
      hasRegisterAck: hasRegisterAck,
    );
    final parityLabel = buildVersionParityNote(
      clientBuildDate: clientBuildDate,
      clientBuildSha: clientBuildSha,
      serverBuildDate: serverBuildDate,
      serverBuildSha: serverBuildSha,
    );
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SelectableText(
            'Client / Server Build',
            style: TextStyle(fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 4),
          SelectableText(clientLabel, style: const TextStyle(fontSize: 12)),
          SelectableText(serverLabel, style: const TextStyle(fontSize: 12)),
          SelectableText(parityLabel, style: const TextStyle(fontSize: 12)),
        ],
      ),
    );
  }
}

class ControlStreamStatusCard extends StatelessWidget {
  const ControlStreamStatusCard({
    super.key,
    required this.status,
    required this.notification,
    required this.transportDiagnostics,
    required this.bugReceiptState,
    this.bugReceiptReportId = '',
    this.bugReceiptDetail = '',
  });

  final String status;
  final String notification;
  final String transportDiagnostics;
  final BugReceiptChromeState bugReceiptState;
  final String bugReceiptReportId;
  final String bugReceiptDetail;

  @override
  Widget build(BuildContext context) {
    final blockText = buildControlStreamClipboardText(
      status: status,
      notification: notification,
      transportDiagnostics: transportDiagnostics,
    );
    return Material(
      elevation: 4,
      borderRadius: BorderRadius.circular(12),
      color: Colors.white,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          border: Border.all(color: Colors.blueGrey.shade200),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            SelectableText(blockText, style: const TextStyle(fontSize: 12)),
            if (bugReceiptState != BugReceiptChromeState.none) ...[
              const SizedBox(height: 8),
              BugReceiptPanel(
                state: bugReceiptState,
                reportId: bugReceiptReportId,
                detail: bugReceiptDetail,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class DiagnosticsPanel extends StatelessWidget {
  const DiagnosticsPanel({
    super.key,
    required this.title,
    required this.data,
  });

  final String title;
  final Map<String, String> data;

  @override
  Widget build(BuildContext context) {
    final keys = data.keys.toList()..sort();
    final displayKeys = keys.take(16).toList();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Diagnostics: $title'),
          if (displayKeys.isEmpty)
            const Text('No diagnostics data yet')
          else
            ...displayKeys.map(
              (key) => Text(
                '$key=${data[key]}',
                style: const TextStyle(fontSize: 12),
              ),
            ),
        ],
      ),
    );
  }
}

class TransportDiagnosticsPanel extends StatelessWidget {
  const TransportDiagnosticsPanel({
    super.key,
    required this.lastTransportDiagnostic,
    required this.recentAttempts,
  });

  final String lastTransportDiagnostic;
  final List<String> recentAttempts;

  @override
  Widget build(BuildContext context) {
    final blockText = buildTransportDiagnosticsClipboardText(
      lastTransportDiagnostic: lastTransportDiagnostic,
      recentAttempts: recentAttempts,
    );
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SelectableText(blockText, style: const TextStyle(fontSize: 12)),
        ],
      ),
    );
  }
}

class ConnectionPhaseChip extends StatelessWidget {
  const ConnectionPhaseChip({
    super.key,
    required this.phase,
    required this.isRegistered,
  });

  final ConnectionPhase phase;
  final bool isRegistered;

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(
        isRegistered ? Icons.check_circle_outline : Icons.sync_outlined,
      ),
      label: Text(buildConnectionPhaseLabel(phase)),
    );
  }
}
