import 'package:flutter/material.dart';

enum BugReceiptChromeState {
  none,
  waiting,
  received,
  error,
}

class BugReportButton extends StatelessWidget {
  const BugReportButton({
    super.key,
    required this.onPressed,
  });

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton.extended(
      onPressed: onPressed,
      icon: const Icon(Icons.bug_report_outlined),
      label: const Text('Report Bug'),
    );
  }
}

class BugReceiptPanel extends StatelessWidget {
  const BugReceiptPanel({
    super.key,
    required this.state,
    this.reportId = '',
    this.detail = '',
  });

  final BugReceiptChromeState state;
  final String reportId;
  final String detail;

  @override
  Widget build(BuildContext context) {
    late final Color borderColor;
    late final Color backgroundColor;
    late final IconData icon;
    late final String title;
    switch (state) {
      case BugReceiptChromeState.none:
        return const SizedBox.shrink();
      case BugReceiptChromeState.waiting:
        borderColor = Colors.amber.shade400;
        backgroundColor = Colors.amber.shade50;
        icon = Icons.schedule_outlined;
        title = 'Bug Report Receipt: Pending';
        break;
      case BugReceiptChromeState.received:
        borderColor = Colors.green.shade400;
        backgroundColor = Colors.green.shade50;
        icon = Icons.verified_outlined;
        title = 'Bug Report Receipt: Received';
        break;
      case BugReceiptChromeState.error:
        borderColor = Colors.red.shade400;
        backgroundColor = Colors.red.shade50;
        icon = Icons.error_outline;
        title = 'Bug Report Receipt: Error';
        break;
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: borderColor),
        borderRadius: BorderRadius.circular(8),
        color: backgroundColor,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                if (reportId.isNotEmpty)
                  Text(
                    'Receipt ID: $reportId',
                    style: const TextStyle(fontSize: 12),
                  ),
                if (detail.isNotEmpty)
                  Text(detail, style: const TextStyle(fontSize: 12)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
