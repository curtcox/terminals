package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.util.Clock
import com.google.protobuf.ByteString
import terminals.capabilities.v1.Capabilities
import terminals.diagnostics.v1.Diagnostics
import terminals.ui.v1.Ui

/**
 * Builds protobuf [Diagnostics.BugReport] messages for the control stream.
 * Generic terminal behavior only — mirrors the Flutter shell’s on-device filing path.
 */
object AndroidBugReportBuilder {
    fun build(
        description: String,
        source: Diagnostics.BugReportSource,
        reporterDeviceId: String,
        subjectDeviceId: String,
        extraSourceHints: Map<String, String>,
        clock: Clock,
        buildMetadata: AndroidBuildMetadata,
        serverRoot: Ui.Node?,
        connectionState: ConnectionState,
        lastServerHeartbeatUnixMs: Long?,
        registeredCapabilities: Capabilities.DeviceCapabilities?,
        localeTag: String,
        timezoneId: String,
        osVersion: String,
        reconnectAttempt: Int = 0,
        lastStatus: String? = null,
        screenshotPng: ByteArray? = null,
    ): Diagnostics.BugReport {
        val nowMillis = clock.nowMillis()
        val identifier = buildBugIdentifier(nowMillis)
        val reportId =
            buildLocalBugReportId(
                nowMillis,
                identifier,
                reporterDeviceId,
                subjectDeviceId,
            )
        val clientContext =
            Diagnostics.ClientContext.newBuilder()
                .setIdentity(identity(reporterDeviceId, buildMetadata, registeredCapabilities, localeTag, timezoneId, osVersion))
                .setRuntime(runtimeState(serverRoot))
                .setConnection(
                    connectionState(
                        connectionState,
                        lastServerHeartbeatUnixMs,
                        reconnectAttempt,
                        lastStatus,
                    ),
                )
                .setHardware(hardwareFrom(registeredCapabilities))
                .setErrorCapture(Diagnostics.ErrorCapture.getDefaultInstance())
                .apply {
                    if (registeredCapabilities != null) {
                        setCapabilities(registeredCapabilities)
                    }
                }
                .build()

        val hints =
            mutableMapOf(
                "bug_token_word" to identifier.word,
                "bug_token_code" to identifier.code,
                "bug_token_timestamp_unix_ms" to nowMillis.toString(),
            )
        hints.putAll(extraSourceHints)
        if (screenshotPng != null && screenshotPng.isNotEmpty()) {
            hints["screenshot_byte_count"] = screenshotPng.size.toString()
        }

        val reportBuilder =
            Diagnostics.BugReport.newBuilder()
                .setReportId(reportId)
                .setReporterDeviceId(reporterDeviceId)
                .setSubjectDeviceId(subjectDeviceId)
                .setSource(source)
                .setDescription(description)
                .setTimestampUnixMs(nowMillis)
                .addTags("bug_word:${identifier.word}")
                .addTags("bug_code:${identifier.code}")
                .putAllSourceHints(hints)
                .setClientContext(clientContext)
        if (screenshotPng != null && screenshotPng.isNotEmpty()) {
            reportBuilder.screenshotPng = ByteString.copyFrom(screenshotPng)
        }
        return reportBuilder.build()
    }

    private fun identity(
        reporterDeviceId: String,
        buildMetadata: AndroidBuildMetadata,
        registeredCapabilities: Capabilities.DeviceCapabilities?,
        localeTag: String,
        timezoneId: String,
        osVersion: String,
    ): Diagnostics.ClientIdentity {
        val b =
            Diagnostics.ClientIdentity.newBuilder()
                .setDeviceId(reporterDeviceId)
                .setClientVersion(buildMetadata.versionName)
                .setClientGitSha(buildMetadata.buildSha)
                .setClientBuildUnixMs(0)
                .setOsVersion(osVersion)
                .setLocale(localeTag)
                .setTimezone(timezoneId)
                .setClockOffsetMs(0)
        if (registeredCapabilities != null && registeredCapabilities.hasIdentity()) {
            b.setDeviceName(registeredCapabilities.identity.deviceName)
            b.setDeviceType(registeredCapabilities.identity.deviceType)
            b.setPlatform(registeredCapabilities.identity.platform)
        } else {
            b.setPlatform("android")
        }
        return b.build()
    }

    private fun runtimeState(serverRoot: Ui.Node?): Diagnostics.RuntimeState =
        Diagnostics.RuntimeState.newBuilder()
            .setActiveUiRoot(serverRoot ?: Ui.Node.getDefaultInstance())
            .build()

    private fun connectionState(
        state: ConnectionState,
        lastServerHeartbeatUnixMs: Long?,
        reconnectAttempt: Int,
        lastStatus: String?,
    ): Diagnostics.ConnectionHealth {
        val online = state == ConnectionState.Connected
        val b =
            Diagnostics.ConnectionHealth.newBuilder()
                .setOnline(online)
        lastServerHeartbeatUnixMs?.takeIf { it > 0 }?.let { b.setLastHeartbeatUnixMs(it) }
        if (reconnectAttempt > 0) {
            b.reconnectAttempt = reconnectAttempt
        }
        lastStatus?.takeIf { it.isNotBlank() }?.let { b.lastStatus = it }
        return b.build()
    }

    private fun hardwareFrom(cap: Capabilities.DeviceCapabilities?): Diagnostics.HardwareState {
        val b = Diagnostics.HardwareState.newBuilder()
        if (cap != null && cap.hasScreen()) {
            b.screenWidthPx = cap.screen.width
            b.screenHeightPx = cap.screen.height
            b.devicePixelRatio = cap.screen.density
            b.orientation = cap.screen.orientation
        }
        if (cap != null && cap.hasBattery()) {
            b.batteryLevel = cap.battery.level
            b.batteryCharging = cap.battery.charging
            b.putSensorSnapshot("battery.level", cap.battery.level.toDouble())
            b.putSensorSnapshot("battery.charging", if (cap.battery.charging) 1.0 else 0.0)
        }
        return b.build()
    }
}
