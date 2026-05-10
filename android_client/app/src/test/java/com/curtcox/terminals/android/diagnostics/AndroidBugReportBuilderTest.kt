package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.util.Clock
import java.util.TimeZone
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import terminals.capabilities.v1.Capabilities
import terminals.diagnostics.v1.Diagnostics
import terminals.ui.v1.Ui

class AndroidBugReportBuilderTest {
    private val fixedClock = Clock { 1_778_353_200_000L }
    private val buildMeta = AndroidBuildMetadata("9.9.9-test", "abc123", "2026-05-09")
    private var previousDefaultTimeZone: TimeZone? = null

    @Before
    fun pinDefaultTimeZone() {
        previousDefaultTimeZone = TimeZone.getDefault()
        TimeZone.setDefault(TimeZone.getTimeZone("America/Los_Angeles"))
    }

    @After
    fun restoreDefaultTimeZone() {
        previousDefaultTimeZone?.let { TimeZone.setDefault(it) }
    }

    @Test
    fun buildIncludesBugTagsHintsAndReportIdShape() {
        val cap =
            Capabilities.DeviceCapabilities.newBuilder()
                .setIdentity(
                    Capabilities.DeviceIdentity.newBuilder()
                        .setDeviceName("Test Tablet")
                        .setDeviceType("tablet")
                        .setPlatform("android"),
                )
                .build()
        val report =
            AndroidBugReportBuilder.build(
                description = "unit test",
                source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
                reporterDeviceId = "reporter-1",
                subjectDeviceId = "subject-1",
                extraSourceHints = mapOf("entry_point" to "test"),
                clock = fixedClock,
                buildMetadata = buildMeta,
                serverRoot = null,
                connectionState = ConnectionState.Connected,
                lastServerHeartbeatUnixMs = 99L,
                registeredCapabilities = cap,
                localeTag = "en-US",
                timezoneId = "America/Los_Angeles",
                osVersion = "11 (API 30)",
            )

        assertTrue(report.reportId.startsWith("clientbug-${fixedClock.nowMillis()}-"))
        assertTrue(report.tagsList.any { it.startsWith("bug_word:") })
        assertTrue(report.tagsList.any { it.startsWith("bug_code:") })
        assertEquals("square", report.sourceHintsMap["bug_token_word"])
        assertEquals("120000-square", report.sourceHintsMap["bug_token_code"])
        assertEquals(fixedClock.nowMillis().toString(), report.sourceHintsMap["bug_token_timestamp_unix_ms"])
        assertEquals("test", report.sourceHintsMap["entry_point"])
        assertEquals("reporter-1", report.reporterDeviceId)
        assertEquals("subject-1", report.subjectDeviceId)
        assertEquals("unit test", report.description)
    }

    @Test
    fun buildConnectionHealthReflectsDisconnectedState() {
        val report =
            AndroidBugReportBuilder.build(
                description = "d",
                source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
                reporterDeviceId = "r",
                subjectDeviceId = "s",
                extraSourceHints = emptyMap(),
                clock = fixedClock,
                buildMetadata = buildMeta,
                serverRoot = null,
                connectionState = ConnectionState.ReadyToConnect,
                lastServerHeartbeatUnixMs = null,
                registeredCapabilities = null,
                localeTag = "en",
                timezoneId = "UTC",
                osVersion = "1",
            )

        assertEquals(false, report.clientContext.connection.online)
    }

    @Test
    fun buildEmbedsActiveUiRootWhenPresent() {
        val root =
            Ui.Node.newBuilder()
                .setText(Ui.TextWidget.newBuilder().setValue("hello"))
                .build()
        val report =
            AndroidBugReportBuilder.build(
                description = "d",
                source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
                reporterDeviceId = "r",
                subjectDeviceId = "s",
                extraSourceHints = emptyMap(),
                clock = fixedClock,
                buildMetadata = buildMeta,
                serverRoot = root,
                connectionState = ConnectionState.Connected,
                lastServerHeartbeatUnixMs = null,
                registeredCapabilities = null,
                localeTag = "en",
                timezoneId = "UTC",
                osVersion = "1",
            )

        assertEquals("hello", report.clientContext.runtime.activeUiRoot.text.value)
    }
}
