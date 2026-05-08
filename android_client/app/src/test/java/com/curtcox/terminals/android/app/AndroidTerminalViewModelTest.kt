package com.curtcox.terminals.android.app

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
import com.curtcox.terminals.android.platform.AndroidNetworkState
import com.curtcox.terminals.android.platform.AndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
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
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import terminals.control.v1.Control
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
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_connected=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("network_metered=false"))
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
    fun connectFailureReturnsToReadyStateWithDiagnostics() = runTest(dispatcher) {
        val session = FakeSession(connectError = IllegalStateException("no route"))
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertEquals("no route", viewModel.state.value.lastError)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_error=no route"))
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
        heartbeatIntervalMillis: Long = 0,
    ): AndroidTerminalViewModel =
        AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                notificationDelivery = notificationDelivery,
                mediaEngine = mediaEngine,
                networkStateProvider = networkStateProvider,
                mediaPermissionProbe = mediaPermissionProbe,
                webRtcAdapter = webRtcAdapter,
                heartbeatIntervalMillis = heartbeatIntervalMillis,
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

    private class FakeSession(
        private val connectError: Throwable? = null,
        private val capabilityDeltaSent: Boolean = false,
        private val heartbeatError: Throwable? = null,
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
        var stopCount = 0

        override fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit) {
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
