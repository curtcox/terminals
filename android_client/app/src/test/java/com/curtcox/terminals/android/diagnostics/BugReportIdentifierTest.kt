package com.curtcox.terminals.android.diagnostics

import org.junit.Assert.assertEquals
import org.junit.Test
import java.util.TimeZone

class BugReportIdentifierTest {
    @Test
    fun buildBugIdentifierMatchesFlutterReferenceForLaNoon20260509() {
        val millis = 1_778_353_200_000L
        val id = buildBugIdentifier(millis, TimeZone.getTimeZone("America/Los_Angeles"))
        assertEquals("square", id.word)
        assertEquals("120000-square", id.code)
        assertEquals("terminals-bug://120000-square", id.qrPayload)
    }

    @Test
    fun buildLocalBugReportIdMatchesFlutterSanitizationShape() {
        val id = BugIdentifier("square", "120000-square", "terminals-bug://120000-square")
        val rid =
            buildLocalBugReportId(
                1_778_353_200_000L,
                id,
                reporterDeviceId = "Device A!",
                subjectDeviceId = "sub:ject",
            )
        assertEquals("clientbug-1778353200000-device-a-sub-ject-120000-square", rid)
    }
}
