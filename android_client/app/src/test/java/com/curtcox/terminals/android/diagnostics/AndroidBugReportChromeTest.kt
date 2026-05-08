package com.curtcox.terminals.android.diagnostics

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus

class AndroidBugReportChromeTest {
    @Test
    fun formatIncludesCoreFieldsAndOmitsEmptyOptionals() {
        val ack = BugReportAck.newBuilder()
            .setReportId("rep-1")
            .setCorrelationId("corr-9")
            .setStatus(BugReportStatus.BUG_REPORT_STATUS_FILED)
            .setReportPath("logs/bug_reports/2026-05-08/rep-1.json")
            .build()

        val lines = AndroidBugReportChrome.formatDiagnosticsLines(ack)

        assertTrue(lines.contains("bug_report_id=rep-1"))
        assertTrue(lines.contains("bug_report_correlation_id=corr-9"))
        assertTrue(lines.contains("bug_report_status=filed"))
        assertTrue(lines.contains("bug_report_path=logs/bug_reports/2026-05-08/rep-1.json"))
        assertFalse(lines.contains("bug_report_merged_autodetect_id"))
        assertFalse(lines.contains("bug_report_message"))
    }

    @Test
    fun formatIncludesMergedIdAndMessageWhenPresent() {
        val ack = BugReportAck.newBuilder()
            .setReportId("rep-2")
            .setCorrelationId("")
            .setStatus(BugReportStatus.BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT)
            .setReportPath("path.json")
            .setMergedAutodetectReportId("auto-3")
            .setMessage("merged with autodetect intake")
            .build()

        val lines = AndroidBugReportChrome.formatDiagnosticsLines(ack)

        assertTrue(lines.contains("bug_report_merged_autodetect_id=auto-3"))
        assertTrue(lines.contains("bug_report_message=merged with autodetect intake"))
        assertTrue(lines.contains("bug_report_status=merged_with_autodetect"))
    }

    @Test
    fun formatMapsRejectedStatus() {
        val lines = AndroidBugReportChrome.formatDiagnosticsLines(
            BugReportAck.newBuilder()
                .setStatus(BugReportStatus.BUG_REPORT_STATUS_REJECTED)
                .build(),
        )
        assertTrue(lines.contains("bug_report_status=rejected"))
    }
}
