package com.curtcox.terminals.android.diagnostics

import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus

/**
 * Formats server [BugReportAck] payloads for terminal diagnostics chrome.
 * Generic terminal behavior only — no scenario-specific interpretation.
 */
object AndroidBugReportChrome {
    fun formatDiagnosticsLines(ack: BugReportAck): String = buildString {
        appendLine("bug_report_id=${ack.reportId}")
        appendLine("bug_report_correlation_id=${ack.correlationId}")
        appendLine("bug_report_status=${formatStatus(ack.status)}")
        appendLine("bug_report_path=${ack.reportPath}")
        if (ack.mergedAutodetectReportId.isNotBlank()) {
            appendLine("bug_report_merged_autodetect_id=${ack.mergedAutodetectReportId}")
        }
        if (ack.message.isNotBlank()) {
            appendLine("bug_report_message=${ack.message}")
        }
    }.trimEnd()

    private fun formatStatus(status: BugReportStatus): String =
        when (status) {
            BugReportStatus.BUG_REPORT_STATUS_UNSPECIFIED -> "unspecified"
            BugReportStatus.BUG_REPORT_STATUS_FILED -> "filed"
            BugReportStatus.BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT -> "merged_with_autodetect"
            BugReportStatus.BUG_REPORT_STATUS_REJECTED -> "rejected"
            BugReportStatus.UNRECOGNIZED -> "unrecognized"
        }
}
