package com.curtcox.terminals.android.app

import android.Manifest
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ReconnectPolicy
import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidHardwareCapabilities
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.diagnostics.DiagnosticClipboard
import com.curtcox.terminals.android.discovery.AndroidNsdDiscovery
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AndroidAudioPlayback
import com.curtcox.terminals.android.media.AndroidMediaDisplay
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.AndroidMediaPermissionState
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.media.AndroidWebRtcSupport
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidNetworkMonitor
import com.curtcox.terminals.android.platform.AndroidNetworkState
import com.curtcox.terminals.android.platform.AndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.AndroidPermissionRequester
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.platform.FireOsDeviceInfo
import com.curtcox.terminals.android.platform.FireOsDeviceInfoProvider
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.setMain
import kotlinx.coroutines.CompletableDeferred
import org.junit.After
import org.junit.Assume.assumeTrue
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus
import terminals.io.v1.Io
import terminals.ui.v1.Ui

@OptIn(ExperimentalCoroutinesApi::class)
class AndroidTerminalViewModelTest {
    private val dispatcher = StandardTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun connectCreatesSessionAndMarksStateConnected() = runTest(dispatcher) {
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
    fun networkMonitorRefreshesDiagnosticsAndConnectedCapabilities() = runTest(dispatcher) {
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
    fun networkMonitorRestartsDiscoveryWhenScanning() = runTest(dispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
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
    fun networkMonitorDebouncesDiscoveryRestartWhenCallbacksBurst() = runTest(dispatcher) {
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
    fun networkMonitorDebouncesCapabilityRefreshWhenCallbacksBurst() = runTest(dispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val monitor = FakeNetworkMonitor()
        var now = 5_000L
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
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
    fun requestMicrophonePermissionRefreshesPermissionEducationAndCapabilities() = runTest(dispatcher) {
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
    fun requestMissingPermissionsRequestsPresentMissingMediaPermissions() = runTest(dispatcher) {
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
    fun requestMissingPermissionsRequestsNotificationPermissionWhenRuntimePromptIsSupported() = runTest(dispatcher) {
        assumeTrue(android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU)
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
    fun connectFailureReturnsToReadyStateWithDiagnostics() = runTest(dispatcher) {
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
    fun reconnectClosesPreviousSessionBeforeOpeningNext() = runTest(dispatcher) {
        val first = FakeSession()
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
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
    fun secondConnectCancelsInFlightConnectAttemptAndClosesProvisionalSession() = runTest(dispatcher) {
        val firstConnectGate = CompletableDeferred<Unit>()
        val first = FakeSession(connectGate = firstConnectGate)
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
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
    fun disconnectWithValidEndpointReturnsToReadyStateDiagnostics() = runTest(dispatcher) {
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
    fun connectedSessionSendsPeriodicHeartbeats() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session, heartbeatIntervalMillis = 100)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(250)
        runCurrent()

        assertEquals(2, session.heartbeatCount)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun heartbeatFailureReconnectsWithBoundedBackoff() = runTest(dispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 2,
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        runCurrent()
        advanceTimeBy(100)
        runCurrent()
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_pending=true"))

        advanceTimeBy(50)
        runCurrent()

        assertEquals(1, first.closeCount)
        assertEquals(EndpointResolution("10.0.0.8", 8080), second.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_success_attempt=1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun heartbeatFailureStopsAfterReconnectAttemptsAreExhausted() = runTest(dispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession(connectError = IllegalStateException("still offline"))
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
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
    fun networkRestoreRetriesConnectAfterReconnectIsExhausted() = runTest(dispatcher) {
        val first = FakeSession(heartbeatError = IllegalStateException("socket closed"))
        val second = FakeSession(connectError = IllegalStateException("still offline"))
        val third = FakeSession()
        val sessions = ArrayDeque(listOf(first, second, third))
        val monitor = FakeNetworkMonitor()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 100,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 50, maxDelayMillis = 50),
                maxReconnectAttempts = 1,
                networkMonitor = monitor,
                networkReconnectRestoreMinIntervalMillis = 0,
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
        advanceUntilIdle()

        assertEquals(EndpointResolution("10.0.0.8", 8080), third.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_cause=network-restore:network-callback"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun networkRestoreDoesNotRetryAfterUserDisconnect() = runTest(dispatcher) {
        val session = FakeSession()
        val sessions = ArrayDeque(listOf(session))
        val monitor = FakeNetworkMonitor()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
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
    fun networkRestoreDebouncesReconnectAttempts() = runTest(dispatcher) {
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
        advanceUntilIdle()
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_cause=network-restore:network-callback"))

        now += 1_000
        monitor.emitChange()
        advanceUntilIdle()
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_reconnect_restore_suppressed=network-callback"))
        assertEquals(true, sessions.size == 1)

        now += 5_000
        monitor.emitChange()
        advanceUntilIdle()
        assertEquals(EndpointResolution("10.0.0.8", 8080), fourth.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverSetUiResponseUpdatesRenderedRoot() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setText(Ui.TextWidget.newBuilder().setValue("Ready"))
            .build()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setSetUi(Ui.SetUI.newBuilder().setDeviceId("device-1").setRoot(root))
                .build(),
        )

        assertEquals(root, viewModel.state.value.serverRoot)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverBugReportAckIsSurfacedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setBugReportAck(
                    BugReportAck.newBuilder()
                        .setReportId("rep-native-1")
                        .setCorrelationId("cor-2")
                        .setStatus(BugReportStatus.BUG_REPORT_STATUS_FILED)
                        .setReportPath("logs/bug_reports/2026-05-08/rep-native-1.json")
                        .setMessage("stored")
                        .build(),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(viewModel.state.value.diagnosticsText.contains("bug_report_id=rep-native-1"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("bug_report_status=filed"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("bug_report_message=stored"))
        assertTrue(viewModel.state.value.lastBugReportAckDiagnostics!!.contains("rep-native-1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverBugReportAckRemainsInDiagnosticsAfterDisconnect() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setBugReportAck(
                    BugReportAck.newBuilder()
                        .setReportId("rep-native-1")
                        .setCorrelationId("cor-2")
                        .setStatus(BugReportStatus.BUG_REPORT_STATUS_FILED)
                        .setReportPath("logs/bug_reports/2026-05-08/rep-native-1.json")
                        .setMessage("stored")
                        .build(),
                )
                .build(),
        )
        advanceUntilIdle()

        viewModel.disconnect()
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("bug_report_id=rep-native-1"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("bug_report_status=filed"))
        assertTrue(viewModel.state.value.lastBugReportAckDiagnostics!!.contains("rep-native-1"))
    }

    @Test
    fun newConnectClearsBugReportAckFromPriorSession() = runTest(dispatcher) {
        val first = FakeSession()
        val second = FakeSession()
        val sessions = ArrayDeque(listOf(first, second))
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                sessionFactory = { sink ->
                    sessions.removeFirst().also { it.sink = sink }
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        first.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setBugReportAck(
                    BugReportAck.newBuilder()
                        .setReportId("rep-native-1")
                        .setStatus(BugReportStatus.BUG_REPORT_STATUS_FILED)
                        .build(),
                )
                .build(),
        )
        advanceUntilIdle()
        assertTrue(viewModel.state.value.lastBugReportAckDiagnostics!!.contains("rep-native-1"))

        viewModel.disconnect()
        advanceUntilIdle()

        viewModel.connect()
        advanceUntilIdle()
        assertNull(viewModel.state.value.lastBugReportAckDiagnostics)
    }

    @Test
    fun serverTransitionUiIsSurfacedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setTransitionUi(
                    Ui.TransitionUI.newBuilder()
                        .setDeviceId("device-1")
                        .setTransition("slide_left")
                        .setDurationMs(200),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("slide_left", viewModel.state.value.lastTransition)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_transition=slide_left"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_transition_duration_ms=200"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverTransitionUiRemainsInDiagnosticsAfterNetworkRefresh() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setTransitionUi(
                    Ui.TransitionUI.newBuilder()
                        .setDeviceId("device-1")
                        .setTransition("slide_left")
                        .setDurationMs(200),
                )
                .build(),
        )
        advanceUntilIdle()

        viewModel.refreshNetworkDiagnostics("network-change")
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_transition=slide_left"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_transition_duration_ms=200"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_network_refresh=network-change"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverHeartbeatIsSurfacedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setHeartbeat(
                    Control.Heartbeat.newBuilder()
                        .setDeviceId("device-1")
                        .setUnixMs(1_700_000_000_000L),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals(1_700_000_000_000L, viewModel.state.value.lastServerHeartbeatUnixMs)
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("last_server_heartbeat_unix_ms=1700000000000"),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverCommandResultIsSurfacedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setCommandResult(
                    Control.CommandResult.newBuilder()
                        .setRequestId("cmd-7")
                        .setNotification("Started timer"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("cmd-7", viewModel.state.value.lastCommandResultRequestId)
        assertEquals("Started timer", viewModel.state.value.lastCommandResultNotification)
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("last_command_result_request_id=cmd-7"),
        )
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("last_command_result_notification=Started timer"),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun capabilityAckInvalidationsAndSnapshotAppliedAreSurfacedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setCapabilityAck(
                    Control.CapabilityAck.newBuilder()
                        .setDeviceId("device-1")
                        .setAcceptedGeneration(9)
                        .setSnapshotApplied(true)
                        .addInvalidations(
                            Control.ResourceInvalidation.newBuilder()
                                .setResource("mic.capture")
                                .setReason("capability_lost"),
                        ),
                )
                .build(),
        )
        advanceUntilIdle()

        val text = viewModel.state.value.diagnosticsText
        assertTrue(text.contains("last_capability_ack_generation=9"))
        assertTrue(text.contains("capability_ack_snapshot_applied=true"))
        assertTrue(text.contains("last_capability_invalidations=mic.capture:capability_lost"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun opaqueStartStreamSummaryIsSurfacedInDiagnosticsAndClearsOnDisconnect() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStartStream(
                    Io.StartStream.newBuilder()
                        .setStreamId("s-out")
                        .setStreamKind(Io.StreamKind.STREAM_KIND_AUDIO),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(viewModel.state.value.diagnosticsText.contains("last_opaque_control_io="))
        assertTrue(viewModel.state.value.diagnosticsText.contains("stream_id=s-out"))
        viewModel.disconnect()
        advanceUntilIdle()
        assertNull(viewModel.state.value.lastOpaqueControlIoSummary)
    }

    @Test
    fun serverNotificationIsDeliveredThroughPlatformAdapter() = runTest(dispatcher) {
        val session = FakeSession()
        val delivered = mutableListOf<Pair<String, String>>()
        val viewModel = viewModel(
            session,
            notificationDelivery = AndroidNotificationDelivery { title, body -> delivered += title to body },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setNotification(
                    Ui.Notification.newBuilder()
                        .setDeviceId("device-1")
                        .setTitle("Timer")
                        .setBody("Done"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals(listOf("Timer" to "Done"), delivered)
        assertEquals("Timer", viewModel.state.value.lastNotificationTitle)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_notification=Timer"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun playAudioResponseIsDelegatedThroughMediaEngine() = runTest(dispatcher) {
        val session = FakeSession()
        val played = mutableListOf<Io.PlayAudio>()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(
                audioPlayback = AndroidAudioPlayback { command ->
                    played += command
                    AudioPlaybackResult.Played(command.requestId)
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setPlayAudio(
                    Io.PlayAudio.newBuilder()
                        .setRequestId("audio-1")
                        .setDeviceId("device-1")
                        .setPcmData(com.google.protobuf.ByteString.copyFrom(byteArrayOf(1, 2, 3)))
                        .setFormat("audio/pcm"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("audio-1", played.single().requestId)
        assertEquals("audio-1", viewModel.state.value.lastMediaRequestId)
        assertEquals("played", viewModel.state.value.lastMediaStatus)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_media=audio-1:played"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun unsupportedShowMediaResponseIsRecordedInDiagnostics() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setShowMedia(
                    Io.ShowMedia.newBuilder()
                        .setRequestId("media-1")
                        .setDeviceId("device-1")
                        .setMediaUrl("https://example.test/clip.mp4")
                        .setMediaType("video/mp4"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("media-1", viewModel.state.value.lastMediaRequestId)
        assertEquals("unsupported-media:video/mp4", viewModel.state.value.lastMediaStatus)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_media=media-1:unsupported-media:video/mp4"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun showMediaResponseCanBeDelegatedThroughMediaEngine() = runTest(dispatcher) {
        val session = FakeSession()
        val shown = mutableListOf<Io.ShowMedia>()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(
                mediaDisplay = AndroidMediaDisplay { command ->
                    shown += command
                    MediaDisplayResult.Shown(command.requestId)
                },
            ),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setShowMedia(
                    Io.ShowMedia.newBuilder()
                        .setRequestId("media-2")
                        .setDeviceId("device-1")
                        .setMediaUrl("https://example.test/image.png")
                        .setMediaType("image/png"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("media-2", shown.single().requestId)
        assertEquals("shown", viewModel.state.value.lastMediaStatus)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun uiActionIsSentThroughConnectedSession() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendUiAction(ServerDrivenAction("start", "tap", "pressed"))
        advanceUntilIdle()

        assertEquals(ServerDrivenAction("start", "tap", "pressed"), session.actions.single())
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun refreshCapabilitiesAsksConnectedSessionForDelta() = runTest(dispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.refreshCapabilities("configuration")
        advanceUntilIdle()

        assertEquals(listOf("configuration"), session.capabilityDeltaReasons)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_capability_delta=configuration"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun staleGenerationProtocolErrorTriggersCapabilityRebaseline() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setError(
                    Control.ControlError.newBuilder()
                        .setCode(Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION)
                        .setMessage("stale capability generation"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals(1, session.rebaselineCount)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_capability_rebaseline=stale-generation"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun unrelatedProtocolErrorDoesNotRebaselineCapabilities() = runTest(dispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setError(
                    Control.ControlError.newBuilder()
                        .setCode(Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION)
                        .setMessage("malformed input"),
                )
                .build(),
        )

        assertEquals(0, session.rebaselineCount)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun keepAwakeDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(keepAwakeController = AndroidKeepAwakeController { calls.add(it) }),
        )

        viewModel.setKeepAwake(true)
        viewModel.setKeepAwake(false)

        assertEquals(listOf(true, false), calls)
    }

    @Test
    fun localKeepAwakeSettingIsRestoredAndApplied() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialKeepAwakeEnabled = true),
                keepAwakeController = AndroidKeepAwakeController { calls.add(it) },
            ),
        )

        assertEquals(true, viewModel.state.value.localKeepAwakeEnabled)
        assertEquals(listOf(true), calls)
    }

    @Test
    fun localKeepAwakeTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Boolean>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                keepAwakeController = AndroidKeepAwakeController { calls.add(it) },
            ),
        )

        viewModel.setLocalKeepAwake(true)

        assertEquals(true, settings.keepAwakeEnabled())
        assertEquals(true, viewModel.state.value.localKeepAwakeEnabled)
        assertEquals(listOf(true), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_keep_awake=true"))
    }

    @Test
    fun localFullscreenSettingIsRestoredAndApplied() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialFullscreenEnabled = true),
                fullscreenController = AndroidFullscreenController { calls.add(it) },
            ),
        )

        assertEquals(true, viewModel.state.value.localFullscreenEnabled)
        assertEquals(listOf(true), calls)
    }

    @Test
    fun localFullscreenTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Boolean>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                fullscreenController = AndroidFullscreenController { calls.add(it) },
            ),
        )

        viewModel.setLocalFullscreen(true)

        assertEquals(true, settings.fullscreenEnabled())
        assertEquals(true, viewModel.state.value.localFullscreenEnabled)
        assertEquals(listOf(true), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_fullscreen=true"))
    }

    @Test
    fun fullscreenDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(fullscreenController = AndroidFullscreenController { calls.add(it) }),
        )

        viewModel.setFullscreen(true)
        viewModel.setFullscreen(false)

        assertEquals(listOf(true, false), calls)
    }

    @Test
    fun brightnessDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(brightnessController = AndroidBrightnessController { calls.add(it) }),
        )

        viewModel.setBrightness(0.25)
        viewModel.setBrightness(1.0)

        assertEquals(listOf(0.25, 1.0), calls)
    }

    @Test
    fun localBrightDisplaySettingIsRestoredAndApplied() {
        val calls = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialBrightDisplayEnabled = true),
                brightnessController = AndroidBrightnessController { calls.add(it) },
            ),
        )

        assertEquals(true, viewModel.state.value.localBrightDisplayEnabled)
        assertEquals(listOf(1.0), calls)
    }

    @Test
    fun localBrightDisplayTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Double>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                brightnessController = AndroidBrightnessController { calls.add(it) },
            ),
        )

        viewModel.setLocalBrightDisplay(true)

        assertEquals(true, settings.brightDisplayEnabled())
        assertEquals(true, viewModel.state.value.localBrightDisplayEnabled)
        assertEquals(listOf(1.0), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_bright_display=true"))
    }

    private fun viewModel(
        session: FakeSession,
        notificationDelivery: AndroidNotificationDelivery = AndroidNotificationDelivery.none(),
        mediaEngine: AndroidMediaEngine = AndroidMediaEngine.unsupported(),
        networkStateProvider: AndroidNetworkStateProvider = AndroidNetworkStateProvider.unknown(),
        mediaPermissionProbe: AndroidMediaPermissionProbe = AndroidMediaPermissionProbe.unavailable(),
        webRtcAdapter: AndroidWebRtcAdapter = AndroidWebRtcAdapter { AndroidWebRtcSupport(supported = true) },
        permissionRequester: AndroidPermissionRequester = AndroidPermissionRequester.none(),
        networkMonitor: AndroidNetworkMonitor = AndroidNetworkMonitor.none(),
        heartbeatIntervalMillis: Long = 0,
    ): AndroidTerminalViewModel =
        AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                notificationDelivery = notificationDelivery,
                mediaEngine = mediaEngine,
                networkStateProvider = networkStateProvider,
                networkMonitor = networkMonitor,
                mediaPermissionProbe = mediaPermissionProbe,
                webRtcAdapter = webRtcAdapter,
                permissionRequester = permissionRequester,
                heartbeatIntervalMillis = heartbeatIntervalMillis,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

    private class FakeNetworkMonitor : AndroidNetworkMonitor {
        private var onChanged: (() -> Unit)? = null
        var startCount = 0
        var stopCount = 0

        override fun start(onChanged: () -> Unit) {
            startCount += 1
            this.onChanged = onChanged
        }

        override fun stop() {
            stopCount += 1
            onChanged = null
        }

        fun emitChange() {
            onChanged?.invoke()
        }
    }

    private class FakePermissionRequester(
        val grantedPermissions: MutableSet<String> = mutableSetOf(),
    ) : AndroidPermissionRequester {
        val requests = mutableListOf<String>()
        var nextGrant: Boolean = false

        override fun hasPermission(permission: String): Boolean = grantedPermissions.contains(permission)

        override fun requestPermission(permission: String, onResult: (Boolean) -> Unit) {
            requests += permission
            onResult(nextGrant)
        }
    }

    private class FakeSession(
        private val connectError: Throwable? = null,
        private val capabilityDeltaSent: Boolean = false,
        private val heartbeatError: Throwable? = null,
        private val connectGate: CompletableDeferred<Unit>? = null,
    ) : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        val actions = mutableListOf<ServerDrivenAction>()
        val capabilityDeltaReasons = mutableListOf<String>()
        var rebaselineCount = 0
        var heartbeatCount = 0
        var closeCount = 0

        override suspend fun connect(endpoint: EndpointResolution) {
            connectGate?.await()
            connectError?.let { throw it }
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() {
            heartbeatError?.let { throw it }
            heartbeatCount += 1
        }

        override suspend fun sendUiAction(action: ServerDrivenAction) {
            actions += action
        }

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean {
            capabilityDeltaReasons += reason
            return capabilityDeltaSent
        }

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() {
            rebaselineCount += 1
        }

        override suspend fun close() {
            closeCount += 1
        }
    }

    private class FakeDiscovery : AndroidNsdDiscovery {
        private var onServer: ((DiscoveredServer) -> Unit)? = null
        private var onError: ((String) -> Unit)? = null
        var startCount = 0
        var stopCount = 0

        override fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit) {
            startCount += 1
            this.onServer = onServer
            this.onError = onError
        }

        override fun stop() {
            stopCount += 1
        }

        fun emit(server: DiscoveredServer) {
            onServer?.invoke(server)
        }

        fun fail(message: String) {
            onError?.invoke(message)
        }
    }

    private class FakeCapabilityProbe(
        var permissions: PermissionCapabilityState = PermissionCapabilityState(),
        var hardware: AndroidHardwareCapabilities = AndroidHardwareCapabilities(),
    ) : AndroidCapabilityProbe {
        override fun current(): AndroidCapabilitySnapshotInput =
            AndroidCapabilitySnapshotInput(
                identity = terminals.capabilities.v1.Capabilities.DeviceIdentity.newBuilder()
                    .setDeviceName("test-terminal")
                    .setDeviceType("tablet")
                    .setPlatform("android")
                    .build(),
                screenMetrics = AndroidScreenMetrics(
                    widthPx = 1280,
                    heightPx = 800,
                    density = 1.0f,
                    orientation = "landscape",
                ),
                permissions = permissions,
                hardware = hardware,
            )
    }
}
