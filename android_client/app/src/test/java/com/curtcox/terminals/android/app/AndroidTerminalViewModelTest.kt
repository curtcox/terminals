package com.curtcox.terminals.android.app

import android.Manifest
import com.curtcox.terminals.android.capabilities.AndroidHardwareCapabilities
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ReconnectPolicy
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.diagnostics.DiagnosticClipboard
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AndroidLiveMediaSession
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.AndroidMediaPermissionState
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.platform.AndroidNetworkState
import com.curtcox.terminals.android.platform.AndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.platform.FireOsDeviceInfo
import com.curtcox.terminals.android.platform.FireOsDeviceInfoProvider
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.google.protobuf.ByteString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.control.v1.Control
import terminals.io.v1.Io
import java.io.IOException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest

@OptIn(ExperimentalCoroutinesApi::class)
class AndroidTerminalViewModelTest : AndroidTerminalViewModelTestBase() {

    @Test
    fun connectCreatesSessionAndMarksStateConnected() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(EndpointResolution("10.0.0.8", 8080), session.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("state=Connected"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("control_connected=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("control_endpoint=http://10.0.0.8:8080"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_connected=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_metered=false"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun bugReportServerDrivenActionSendsBugReportNotUiAction() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendUiAction(ServerDrivenAction("btn", "bug_report:other-device", ""))
        advanceUntilIdle()

        assertTrue(session.actions.isEmpty())
        assertEquals(1, session.bugReports.size)
        assertEquals("other-device", session.bugReports.first().subjectDeviceId)
        assertTrue(session.bugReports.first().description.contains("on-device"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun privacyToggleServerDrivenActionDoesNotSendUiAction() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendUiAction(ServerDrivenAction("act:main/privacy_toggle", "privacy.toggle", ""))
        advanceUntilIdle()

        assertTrue(session.actions.isEmpty())
        assertTrue(session.capabilityDeltaReasons.contains("privacy.toggle"))
        assertEquals(listOf(false, true), session.privacyModeCalls)
        assertTrue(viewModel.state.value.privacyModeEnabled)
        assertTrue(viewModel.state.value.diagnosticsText.contains("privacy_mode=true"))

        viewModel.sendUiAction(ServerDrivenAction("act:main/privacy_toggle", "privacy.toggle", ""))
        advanceUntilIdle()
        assertFalse(viewModel.state.value.privacyModeEnabled)
        assertTrue(viewModel.state.value.diagnosticsText.contains("privacy_mode=false"))
        assertEquals(listOf(false, true, false), session.privacyModeCalls)

        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun privacyToggleStopsLocalCaptureOnlyWhenEnablingPrivacy() = runTest(testDispatcher) {
        val disabledLive = AndroidLiveMediaSession.disabled()
        val countingLive =
            object : AndroidLiveMediaSession {
                var stopPrivacyCalls = 0

                override fun applyStartStream(start: Io.StartStream) = disabledLive.applyStartStream(start)

                override fun applyStopStream(streamId: String) = disabledLive.applyStopStream(streamId)

                override fun applyRouteStream(route: Io.RouteStream) = disabledLive.applyRouteStream(route)

                override fun applyWebRtcSignal(signal: Control.WebRTCSignal) =
                    disabledLive.applyWebRtcSignal(signal)

                override fun stopLocalCaptureStreamsForPrivacy() {
                    stopPrivacyCalls += 1
                    disabledLive.stopLocalCaptureStreamsForPrivacy()
                }
            }
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel =
            viewModel(
                session,
                networkStateProvider = AndroidNetworkStateProvider {
                    AndroidNetworkState(connected = true, metered = false)
                },
                mediaEngine = AndroidMediaEngine(liveMedia = countingLive),
            )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        assertEquals(0, countingLive.stopPrivacyCalls)

        viewModel.togglePrivacyMode()
        advanceUntilIdle()
        assertEquals(1, countingLive.stopPrivacyCalls)

        viewModel.togglePrivacyMode()
        advanceUntilIdle()
        assertEquals(1, countingLive.stopPrivacyCalls)

        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun connectAppliesPrivacyModeFromStateBeforeHandshake() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.togglePrivacyMode()
        advanceUntilIdle()
        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(listOf(true), session.privacyModeCalls)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun chromeBugReportQueuedWhenOffline() = runTest(testDispatcher) {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                deviceId = "dev-1",
                sessionFactory = { sink ->
                    FakeSession().also { it.sink = sink }
                },
            ),
        )

        viewModel.submitChromeBugReport()
        advanceUntilIdle()

        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Queued"))
    }

    @Test
    fun chromeBugReportSendsWhenConnected() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.submitChromeBugReport()
        advanceUntilIdle()

        assertEquals(1, session.bugReports.size)
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Sent"))
        assertEquals("android-native-terminal", session.bugReports.first().subjectDeviceId)
    }

    @Test
    fun chromeBugReportIncludesScreenshotWhenCaptureReturnsBytes() = runTest(testDispatcher) {
        val png = byteArrayOf(0x89.toByte(), 0x50, 0x4E, 0x47, 0x01)
        val session = FakeSession()
        val viewModel =
            viewModel(
                session,
                networkStateProvider = AndroidNetworkStateProvider {
                    AndroidNetworkState(connected = true, metered = false)
                },
                bugReportScreenshotCapture = { png },
            )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.submitChromeBugReport()
        advanceUntilIdle()

        val sent = session.bugReports.single()
        assertEquals(png.size.toString(), sent.sourceHintsMap["screenshot_byte_count"])
        assertEquals(ByteString.copyFrom(png), sent.screenshotPng)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun chromeBugReportQueuedThenFlushedOnConnect() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.submitChromeBugReport()
        advanceUntilIdle()
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Queued"))
        assertEquals(0, session.bugReports.size)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(1, session.bugReports.size)
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Sent queued bug report"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun chromeBugReportFlushMultipleQueuedAllSucceed() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.submitChromeBugReport()
        viewModel.submitChromeBugReport()
        advanceUntilIdle()
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Queued"))

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(2, session.bugReports.size)
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("Sent 2 queued bug reports"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun chromeBugReportFlushPartialFailureSummarizesCounts() = runTest(testDispatcher) {
        val session =
            FakeSession().apply {
                bugReportFailurePattern = listOf(true, false)
            }
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.submitChromeBugReport()
        viewModel.submitChromeBugReport()
        advanceUntilIdle()

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(1, session.bugReports.size)
        val status = viewModel.state.value.lastBugReportSubmitStatus!!
        assertTrue(status.contains("1 of 2"))
        assertTrue(status.contains("failed"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun chromeBugReportFlushAllFailRecordsFailure() = runTest(testDispatcher) {
        val session =
            FakeSession().apply {
                bugReportFailurePattern = listOf(true, true)
            }
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider {
                AndroidNetworkState(connected = true, metered = false)
            },
        )

        viewModel.submitChromeBugReport()
        viewModel.submitChromeBugReport()
        advanceUntilIdle()

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(0, session.bugReports.size)
        assertTrue(viewModel.state.value.lastBugReportSubmitStatus!!.contains("failed to send"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun refreshNetworkDiagnosticsSamplesCurrentNetworkState() {
        val states = ArrayDeque(
            listOf(
                AndroidNetworkState(connected = true, metered = false),
                AndroidNetworkState(connected = false, metered = true),
            ),
        )
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                networkStateProvider = AndroidNetworkStateProvider { states.removeFirst() },
            ),
        )

        viewModel.refreshNetworkDiagnostics("network-change")

        assertTrue(viewModel.state.value.diagnosticsText.contains("network_connected=false"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_metered=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_network_refresh=network-change"))
    }

    @Test
    fun diagnosticsIncludeFireOsDeviceInfoWhenAvailable() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                fireOsDeviceInfoProvider = FireOsDeviceInfoProvider {
                    FireOsDeviceInfo(
                        manufacturer = "Amazon",
                        model = "KFSUWI",
                        sdkInt = 28,
                    )
                },
            ),
        )

        assertTrue(viewModel.state.value.diagnosticsText.contains("device_manufacturer=Amazon"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("device_model=KFSUWI"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("device_sdk=28"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("device_likely_fire_os=true"))
    }

    @Test
    fun networkMonitorRefreshesDiagnosticsAndConnectedCapabilities() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        var networkState = AndroidNetworkState(connected = true, metered = false)
        val viewModel = viewModel(
            session,
            networkStateProvider = AndroidNetworkStateProvider { networkState },
            networkMonitor = monitor,
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.startNetworkMonitoring()
        networkState = AndroidNetworkState(connected = false, metered = true)
        monitor.emitChange()
        advanceUntilIdle()

        assertEquals(listOf("network-callback"), session.capabilityDeltaReasons)
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_connected=false"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_metered=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_capability_delta=network-callback"))

        viewModel.stopNetworkMonitoring()
        assertEquals(1, monitor.stopCount)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkMonitoringStartStopIsIdempotent() {
        val monitor = FakeNetworkMonitor()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                networkMonitor = monitor,
            ),
        )

        viewModel.startNetworkMonitoring()
        viewModel.startNetworkMonitoring()
        viewModel.stopNetworkMonitoring()
        viewModel.stopNetworkMonitoring()

        assertEquals(1, monitor.startCount)
        assertEquals(1, monitor.stopCount)
    }

    @Test
    fun networkMonitorRestartsDiscoveryWhenScanning() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                networkMonitor = monitor,
                discovery = discovery,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.startDiscovery()
        assertEquals(1, discovery.startCount)

        viewModel.startNetworkMonitoring()
        monitor.emitChange()
        advanceUntilIdle()

        assertEquals(1, discovery.stopCount)
        assertEquals(2, discovery.startCount)
        assertTrue(viewModel.state.value.diagnosticsText.contains("discovery_restart_reason=network-callback"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkMonitorDebouncesDiscoveryRestartWhenCallbacksBurst() = runTest(testDispatcher) {
        val monitor = FakeNetworkMonitor()
        val discovery = FakeDiscovery()
        var now = 1_000L
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                networkMonitor = monitor,
                discovery = discovery,
                discoveryRestartMinIntervalMillis = 1_500,
                nowMillis = { now },
            ),
        )

        viewModel.startDiscovery()
        assertEquals(1, discovery.startCount)
        viewModel.startNetworkMonitoring()

        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(1, discovery.stopCount)
        assertEquals(2, discovery.startCount)

        now += 500
        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(1, discovery.stopCount)
        assertEquals(2, discovery.startCount)
        assertTrue(viewModel.state.value.diagnosticsText.contains("discovery_restart_suppressed=network-callback"))

        now += 2_000
        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(2, discovery.stopCount)
        assertEquals(3, discovery.startCount)
    }

    @Test
    fun networkMonitorDebouncesCapabilityRefreshWhenCallbacksBurst() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        var now = 5_000L
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                networkMonitor = monitor,
                nowMillis = { now },
                networkCapabilityRefreshMinIntervalMillis = 1_500,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.startNetworkMonitoring()

        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(listOf("network-callback"), session.capabilityDeltaReasons)

        now += 500
        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(listOf("network-callback"), session.capabilityDeltaReasons)
        assertTrue(viewModel.state.value.diagnosticsText.contains("capability_refresh_suppressed=network-callback"))

        now += 2_000
        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(
            listOf("network-callback", "network-callback"),
            session.capabilityDeltaReasons,
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun copyDiagnosticsDelegatesCurrentDiagnosticsToClipboard() {
        var copied: String? = null
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                diagnosticClipboard = DiagnosticClipboard { copied = it },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.copyDiagnostics()

        assertEquals(viewModel.state.value.diagnosticsText, copied)
        assertEquals("copied", viewModel.state.value.lastDiagnosticsCopyStatus)
    }

    @Test
    fun copyDiagnosticsRecordsClipboardFailure() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                diagnosticClipboard = DiagnosticClipboard { error("clipboard unavailable") },
            ),
        )

        viewModel.copyDiagnostics()

        assertEquals("failed", viewModel.state.value.lastDiagnosticsCopyStatus)
        assertEquals("clipboard unavailable", viewModel.state.value.lastError)
    }

    @Test
    fun startDiscoveryRecordsDiscoveredServersAndSelectsEndpoint() {
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                discovery = discovery,
            ),
        )

        viewModel.startDiscovery()
        discovery.emit(
            DiscoveredServer(
                name = "Desk",
                host = "10.0.0.12",
                port = 50054,
                lastSeenMillis = 123,
                webSocketEndpoint = "ws://10.0.0.12:50054/control",
            ),
        )

        assertEquals(true, viewModel.state.value.discoveryState.scanning)
        assertEquals(1, viewModel.state.value.discoveryState.servers.size)
        assertTrue(viewModel.state.value.diagnosticsText.contains("discovered_servers=1"))

        viewModel.selectDiscoveredServer(viewModel.state.value.discoveryState.servers.single())

        assertEquals("ws://10.0.0.12:50054/control", viewModel.state.value.endpointText)
        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertEquals(false, viewModel.state.value.discoveryState.scanning)
        assertEquals(1, discovery.stopCount)
    }

    @Test
    fun discoveryErrorsFallbackToManualEndpointDiagnostics() {
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                discovery = discovery,
            ),
        )

        viewModel.startDiscovery()
        discovery.fail("mDNS blocked by network")

        assertEquals(false, viewModel.state.value.discoveryState.scanning)
        assertEquals("mDNS blocked by network", viewModel.state.value.discoveryState.lastError)
        assertTrue(viewModel.state.value.discoveryState.statusText.contains("Discovery unavailable"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("discovery=error"))
    }

    @Test
    fun initialStateIncludesPermissionEducationFromCapabilityProbe() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                capabilityProbe = FakeCapabilityProbe(
                    permissions = PermissionCapabilityState(
                        microphoneGranted = true,
                        cameraGranted = false,
                        notificationsGranted = false,
                    ),
                    hardware = AndroidHardwareCapabilities(
                        microphone = true,
                        frontCamera = true,
                    ),
                ),
            ),
        )

        assertEquals(false, viewModel.state.value.permissionEducation.notificationsGranted)
        assertEquals(true, viewModel.state.value.permissionEducation.microphonePresent)
        assertEquals(true, viewModel.state.value.permissionEducation.microphoneAvailable)
        assertEquals(true, viewModel.state.value.permissionEducation.cameraPresent)
        assertEquals(false, viewModel.state.value.permissionEducation.cameraAvailable)
        assertTrue(
            viewModel.state.value.permissionEducation.messages.any {
                it.contains("Notifications are disabled")
            },
        )
    }

    @Test
    fun refreshPermissionEducationSamplesCurrentCapabilityProbe() {
        val probe = FakeCapabilityProbe(
            permissions = PermissionCapabilityState(notificationsGranted = true),
            hardware = AndroidHardwareCapabilities(),
        )
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                capabilityProbe = probe,
            ),
        )

        probe.permissions = PermissionCapabilityState(
            microphoneGranted = true,
            cameraGranted = true,
            notificationsGranted = true,
        )
        probe.hardware = AndroidHardwareCapabilities(
            microphone = true,
            backCamera = true,
        )
        viewModel.refreshPermissionEducation("permission-result")

        assertEquals(true, viewModel.state.value.permissionEducation.notificationsGranted)
        assertEquals(true, viewModel.state.value.permissionEducation.microphonePresent)
        assertEquals(true, viewModel.state.value.permissionEducation.microphoneAvailable)
        assertEquals(true, viewModel.state.value.permissionEducation.cameraPresent)
        assertEquals(true, viewModel.state.value.permissionEducation.cameraAvailable)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_permission_refresh=permission-result"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("permission_camera_present=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("permission_camera_available=true"))
    }

    @Test
    fun refreshShellDiagnosticsAndCapabilitiesKeepsNetworkAndPermissionRefreshLines() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
            ),
        )

        viewModel.refreshShellDiagnosticsAndCapabilities(
            networkRefreshReason = "configuration",
            permissionRefreshReason = "configuration",
            capabilityDeltaReason = "display_geometry_change",
        )

        val text = viewModel.state.value.diagnosticsText
        assertTrue(text.contains("last_network_refresh=configuration"))
        assertTrue(text.contains("last_permission_refresh=configuration"))
    }

    @Test
    fun refreshPermissionEducationIncludesMediaPermissionAndWebRtcDiagnostics() {
        var mediaPermissions = AndroidMediaPermissionState(
            microphoneGranted = false,
            cameraGranted = true,
        )
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                mediaPermissionProbe = AndroidMediaPermissionProbe { mediaPermissions },
                webRtcAdapter = AndroidWebRtcAdapter.disabled("fire-os-webrtc-not-enabled"),
            ),
        )

        mediaPermissions = AndroidMediaPermissionState(
            microphoneGranted = true,
            cameraGranted = true,
        )
        viewModel.refreshPermissionEducation("media-permission-result")

        assertEquals(true, viewModel.state.value.mediaSupport.microphonePermissionGranted)
        assertEquals(true, viewModel.state.value.mediaSupport.cameraPermissionGranted)
        assertEquals(false, viewModel.state.value.mediaSupport.webRtcSupported)
        assertEquals("fire-os-webrtc-not-enabled", viewModel.state.value.mediaSupport.webRtcReason)
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_microphone_permission=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_camera_permission=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_webrtc_supported=false"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_webrtc_reason=fire-os-webrtc-not-enabled"))
    }

    @Test
    fun baselineDiagnosticsAlwaysIncludePermissionAndMediaStatus() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                mediaPermissionProbe = AndroidMediaPermissionProbe {
                    AndroidMediaPermissionState(
                        microphoneGranted = false,
                        cameraGranted = true,
                    )
                },
                webRtcAdapter = AndroidWebRtcAdapter.disabled("fire-os-webrtc-not-enabled"),
            ),
        )

        assertTrue(viewModel.state.value.diagnosticsText.contains("permission_notifications="))
        assertTrue(viewModel.state.value.diagnosticsText.contains("permission_microphone_present="))
        assertTrue(viewModel.state.value.diagnosticsText.contains("permission_camera_available="))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_microphone_permission=false"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_camera_permission=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("media_webrtc_reason=fire-os-webrtc-not-enabled"))
    }

    @Test
    fun requestMicrophonePermissionRefreshesPermissionEducationAndCapabilities() = runTest(testDispatcher) {
        val probe = FakeCapabilityProbe(
            permissions = PermissionCapabilityState(
                microphoneGranted = false,
                cameraGranted = true,
                notificationsGranted = true,
            ),
            hardware = AndroidHardwareCapabilities(
                microphone = true,
            ),
        )
        val permissionRequester = FakePermissionRequester(
            grantedPermissions = mutableSetOf(),
        )
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                capabilityProbe = probe,
                permissionRequester = permissionRequester,
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        permissionRequester.nextGrant = true
        probe.permissions = PermissionCapabilityState(
            microphoneGranted = true,
            cameraGranted = true,
            notificationsGranted = true,
        )
        viewModel.requestMicrophonePermission()
        advanceUntilIdle()

        assertEquals(listOf("android.permission.RECORD_AUDIO"), permissionRequester.requests)
        assertEquals(true, viewModel.state.value.permissionEducation.microphoneAvailable)
        assertEquals(listOf("microphone-permission"), session.capabilityDeltaReasons)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_permission_refresh=microphone-permission-result"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun requestMissingPermissionsRequestsPresentMissingMediaPermissions() = runTest(testDispatcher) {
        val probe = FakeCapabilityProbe(
            permissions = PermissionCapabilityState(
                microphoneGranted = false,
                cameraGranted = false,
                notificationsGranted = true,
            ),
            hardware = AndroidHardwareCapabilities(
                microphone = true,
                frontCamera = true,
            ),
        )
        val permissionRequester = FakePermissionRequester(
            grantedPermissions = mutableSetOf(),
        )
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                capabilityProbe = probe,
                permissionRequester = permissionRequester,
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        permissionRequester.nextGrant = true
        probe.permissions = PermissionCapabilityState(
            microphoneGranted = true,
            cameraGranted = true,
            notificationsGranted = true,
        )
        viewModel.requestMissingPermissions()
        advanceUntilIdle()

        assertTrue(permissionRequester.requests.contains(Manifest.permission.RECORD_AUDIO))
        assertTrue(permissionRequester.requests.contains(Manifest.permission.CAMERA))
        assertEquals(
            listOf("microphone-permission", "camera-permission"),
            session.capabilityDeltaReasons,
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun requestMissingPermissionsRequestsNotificationPermissionWhenRuntimePromptIsSupported() = runTest(testDispatcher) {
        val probe = FakeCapabilityProbe(
            permissions = PermissionCapabilityState(
                microphoneGranted = false,
                cameraGranted = false,
                notificationsGranted = false,
            ),
            hardware = AndroidHardwareCapabilities(
                microphone = true,
                frontCamera = true,
            ),
        )
        val permissionRequester = FakePermissionRequester(
            grantedPermissions = mutableSetOf(),
        )
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                capabilityProbe = probe,
                permissionRequester = permissionRequester,
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                runtimeNotificationPermissionPromptSupported = true,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        permissionRequester.nextGrant = true
        probe.permissions = PermissionCapabilityState(
            microphoneGranted = true,
            cameraGranted = true,
            notificationsGranted = true,
        )
        viewModel.requestMissingPermissions()
        advanceUntilIdle()

        assertTrue(permissionRequester.requests.contains(Manifest.permission.POST_NOTIFICATIONS))
        assertTrue(permissionRequester.requests.contains(Manifest.permission.RECORD_AUDIO))
        assertTrue(permissionRequester.requests.contains(Manifest.permission.CAMERA))
        assertTrue(session.capabilityDeltaReasons.contains("notification-permission"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun connectFailureReturnsToReadyStateWithDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession(connectError = IllegalStateException("no route"))
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertEquals("no route", viewModel.state.value.lastError)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_error=no route"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("control_connected=false"))
    }

    @Test
    fun validEndpointUpdatesAreRememberedAsManualEndpoint() {
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                terminalSettings = settings,
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")

        assertEquals("10.0.0.8:8080", settings.lastManualEndpoint())
    }

    @Test
    fun initialStateRestoresRememberedManualEndpoint() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                terminalSettings = AndroidTerminalSettings.inMemory("10.0.0.8:8080"),
            ),
        )

        assertEquals("10.0.0.8:8080", viewModel.state.value.endpointText)
        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
    }

    @Test
    fun reconnectClosesPreviousSessionBeforeOpeningNext() = runTest(testDispatcher) {
        val first = FakeSession()
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(1, first.closeCount)
        assertEquals(EndpointResolution("10.0.0.8", 8080), second.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun secondConnectCancelsInFlightConnectAttemptAndClosesProvisionalSession() = runTest(testDispatcher) {
        val firstConnectGate = CompletableDeferred<Unit>()
        val first = FakeSession(connectGate = firstConnectGate)
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        assertEquals(ConnectionState.Connecting, viewModel.state.value.connectionState)

        viewModel.connect()
        runCurrent()
        firstConnectGate.complete(Unit)
        advanceUntilIdle()

        assertEquals(1, first.closeCount)
        assertEquals(EndpointResolution("10.0.0.8", 8080), second.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun disconnectWithValidEndpointReturnsToReadyStateDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session, heartbeatIntervalMillis = 0)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        viewModel.disconnect()
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("state=ReadyToConnect"))
    }

    @Test
    fun connectedSessionSendsPeriodicHeartbeats() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session, heartbeatIntervalMillis = 100)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        assertEquals(3, session.heartbeatCount)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun connectedSessionSendsPeriodicSensorTelemetry() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session, sensorTelemetryIntervalMillis = 100)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        assertEquals(3, session.sensorTelemetryCount)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun connectedSessionSendsPeriodicRuntimeMonitorPollCapabilityProbe() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session, capabilityMonitorIntervalMillis = 100)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        assertEquals(2, session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" })
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun backgroundPausesHeartbeatSensorTelemetryAndCapabilityMonitorLoops() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            heartbeatIntervalMillis = 100,
            sensorTelemetryIntervalMillis = 100,
            capabilityMonitorIntervalMillis = 100,
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        val heartbeatBefore = session.heartbeatCount
        val sensorBefore = session.sensorTelemetryCount
        val pollBefore = session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" }
        assertTrue(heartbeatBefore >= 1)
        assertTrue(sensorBefore >= 1)
        assertTrue(pollBefore >= 1)

        viewModel.setAppForegrounded(false)
        runCurrent()
        advanceTimeBy(400)
        runCurrent()

        assertEquals(heartbeatBefore, session.heartbeatCount)
        assertEquals(sensorBefore, session.sensorTelemetryCount)
        assertEquals(pollBefore, session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" })

        viewModel.setAppForegrounded(true)
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        assertTrue(session.heartbeatCount > heartbeatBefore)
        assertTrue(session.sensorTelemetryCount > sensorBefore)
        assertTrue(session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" } > pollBefore)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun connectWhileBackgroundedDoesNotStartLoopsUntilForegrounded() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            heartbeatIntervalMillis = 100,
            sensorTelemetryIntervalMillis = 100,
            capabilityMonitorIntervalMillis = 100,
        )

        viewModel.setAppForegrounded(false)
        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        advanceTimeBy(300)
        runCurrent()
        assertEquals(0, session.heartbeatCount)
        assertEquals(0, session.sensorTelemetryCount)
        assertEquals(0, session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" })

        viewModel.setAppForegrounded(true)
        runCurrent()
        advanceTimeBy(250)
        runCurrent()
        assertTrue(session.heartbeatCount >= 1)
        assertTrue(session.sensorTelemetryCount >= 1)
        assertTrue(session.capabilityDeltaReasons.count { it == "runtime_monitor_poll" } >= 1)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun appLifecycleChangeSendsCapabilityDeltaWhenConnected() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = viewModel(
            session,
            heartbeatIntervalMillis = 0,
            sensorTelemetryIntervalMillis = 0,
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        viewModel.setAppForegrounded(false)
        advanceUntilIdle()
        assertEquals(listOf("app_lifecycle_change"), session.capabilityDeltaReasons)

        viewModel.setAppForegrounded(true)
        advanceUntilIdle()
        assertEquals(
            listOf("app_lifecycle_change", "app_lifecycle_change"),
            session.capabilityDeltaReasons,
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkMonitorSkipsCapabilityRefreshWhenBackgrounded() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        val viewModel = viewModel(
            session,
            heartbeatIntervalMillis = 0,
            sensorTelemetryIntervalMillis = 0,
            networkMonitor = monitor,
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.startNetworkMonitoring()

        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(listOf("network-callback"), session.capabilityDeltaReasons)

        viewModel.setAppForegrounded(false)
        advanceUntilIdle()
        assertEquals(
            listOf("network-callback", "app_lifecycle_change"),
            session.capabilityDeltaReasons,
        )
        monitor.emitChange()
        advanceUntilIdle()

        assertEquals(
            listOf("network-callback", "app_lifecycle_change"),
            session.capabilityDeltaReasons,
        )
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("capability_refresh_suppressed=app-background"),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkMonitorSkipsDiscoveryRestartWhenBackgrounded() = runTest(testDispatcher) {
        val monitor = FakeNetworkMonitor()
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                networkMonitor = monitor,
                discovery = discovery,
            ),
        )

        viewModel.startDiscovery()
        assertEquals(1, discovery.startCount)
        viewModel.startNetworkMonitoring()

        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(1, discovery.stopCount)
        assertEquals(2, discovery.startCount)

        viewModel.setAppForegrounded(false)
        advanceUntilIdle()
        monitor.emitChange()
        advanceUntilIdle()

        assertEquals(1, discovery.stopCount)
        assertEquals(2, discovery.startCount)
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("discovery_restart_suppressed=app-background"),
        )
    }

    @Test
    fun heartbeatFailureReconnectsWithBoundedBackoff() = runTest(testDispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 2,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        try {
            viewModel.updateEndpoint("10.0.0.8:8080")
            viewModel.connect()
            runCurrent()
            advanceTimeBy(100)
            runCurrent()

            advanceTimeBy(50)
            runCurrent()

            assertEquals(1, first.closeCount)
            assertEquals(EndpointResolution("10.0.0.8", 8080), second.connectedEndpoint)
            assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        } finally {
            viewModel.disconnect()
            advanceUntilIdle()
        }
    }

    @Test
    fun reconnectAttemptCounterTracksLoopAndResetsOnSuccess() = runTest(testDispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val secondAttemptGate = CompletableDeferred<Unit>()
        val second = FakeSession(
            connectError = IllegalStateException("still offline"),
            connectGate = secondAttemptGate,
        )
        val third = FakeSession()
        val sessions = ArrayDeque(listOf(first, second, third))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 5,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        try {
            viewModel.updateEndpoint("10.0.0.8:8080")
            viewModel.connect()
            runCurrent()
            advanceTimeBy(100)
            runCurrent()

            // First reconnect attempt fails, exposing attempt=1 on view state during the loop.
            advanceTimeBy(50)
            runCurrent()
            assertEquals(1, viewModel.state.value.reconnectAttempt)
            assertEquals(ConnectionState.Connecting, viewModel.state.value.connectionState)
            secondAttemptGate.complete(Unit)
            runCurrent()

            // Second reconnect attempt succeeds; counter resets to 0 on success.
            advanceTimeBy(50)
            runCurrent()
            assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
            assertEquals(0, viewModel.state.value.reconnectAttempt)
        } finally {
            viewModel.disconnect()
            advanceUntilIdle()
        }
    }

    @Test
    fun transportTerminationTriggersReconnectWithBackoff() = runTest(testDispatcher) {
        val first = FakeSession()
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 2,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)

        first.sink.onTransportTerminated(IOException("stream reset"))
        runCurrent()

        assertEquals(ConnectionState.Connecting, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_pending=true"))
        assertEquals("stream reset", viewModel.state.value.lastError)
        assertEquals(1, first.closeCount)

        advanceTimeBy(50)
        runCurrent()

        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        assertEquals(EndpointResolution("10.0.0.8", 8080), second.connectedEndpoint)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_success_attempt=1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun heartbeatFailureStopsAfterReconnectAttemptsAreExhausted() = runTest(testDispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession(connectError = IllegalStateException("still offline"))
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 1,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(150)
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertEquals("still offline", viewModel.state.value.lastError)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_exhausted=1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkRestoreRetriesConnectAfterReconnectIsExhausted() = runTest(testDispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession(connectError = IllegalStateException("still offline"))
        val third = FakeSession()
        val sessions = ArrayDeque(listOf(first, second, third))
        val monitor = FakeNetworkMonitor()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 1,
                networkMonitor = monitor,
                networkReconnectRestoreMinIntervalMillis = 0,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        try {
            viewModel.updateEndpoint("10.0.0.8:8080")
            viewModel.connect()
            runCurrent()
            advanceTimeBy(150)
            advanceUntilIdle()
            assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_exhausted=1"))

            viewModel.startNetworkMonitoring()
            monitor.emitChange()
            advanceTimeBy(400)
            runCurrent()

            assertEquals(EndpointResolution("10.0.0.8", 8080), third.connectedEndpoint)
            assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        } finally {
            viewModel.disconnect()
            advanceUntilIdle()
        }
    }

    @Test
    fun networkRestoreDoesNotRetryAfterUserDisconnect() = runTest(testDispatcher) {
        val session = FakeSession()
        val sessions = ArrayDeque(listOf(session))
        val monitor = FakeNetworkMonitor()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                sensorTelemetryIntervalMillis = 0,
                networkMonitor = monitor,
                networkReconnectRestoreMinIntervalMillis = 0,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.disconnect()
        advanceUntilIdle()
        viewModel.startNetworkMonitoring()
        monitor.emitChange()
        advanceUntilIdle()

        assertEquals(1, session.closeCount)
        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertTrue(!viewModel.state.value.diagnosticsText.contains("network-restore:"))
    }

    @Test
    fun networkRestoreDebouncesReconnectAttempts() = runTest(testDispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession(connectError = IllegalStateException("still offline"))
        val third = FakeSession(connectError = IllegalStateException("still offline"))
        val fourth = FakeSession()
        val sessions = ArrayDeque(listOf(first, second, third, fourth))
        val monitor = FakeNetworkMonitor()
        var now = 1_000L
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                sensorTelemetryIntervalMillis = 0,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 1,
                networkMonitor = monitor,
                networkReconnectRestoreMinIntervalMillis = 5_000,
                nowMillis = { now },
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(150)
        advanceUntilIdle()
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_exhausted=1"))

        viewModel.startNetworkMonitoring()
        monitor.emitChange()
        advanceTimeBy(400)
        runCurrent()
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_cause=network-restore:network-callback"))

        now += 1_000
        monitor.emitChange()
        advanceTimeBy(400)
        runCurrent()
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_reconnect_restore_suppressed=network-callback"))
        assertEquals(true, sessions.size == 1)

        now += 5_000
        monitor.emitChange()
        advanceTimeBy(400)
        runCurrent()
        assertEquals(EndpointResolution("10.0.0.8", 8080), fourth.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        viewModel.disconnect()
        advanceUntilIdle()
    }
}
