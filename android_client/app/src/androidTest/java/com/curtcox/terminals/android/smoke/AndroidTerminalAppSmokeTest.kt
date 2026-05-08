package com.curtcox.terminals.android.smoke

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import com.curtcox.terminals.android.app.AndroidClientDependencies
import com.curtcox.terminals.android.app.AndroidTerminalApp
import com.curtcox.terminals.android.app.AndroidTerminalViewModel
import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidHardwareCapabilities
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ReconnectPolicy
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.diagnostics.DiagnosticClipboard
import com.curtcox.terminals.android.discovery.AndroidNsdDiscovery
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.AndroidMediaPermissionState
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.ui.v1.Ui

class AndroidTerminalAppSmokeTest {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun invalidManualEndpointIsRejectedLocally() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("not-a-valid-endpoint")
        compose.onNodeWithTag("terminal-connect-button").assertIsNotEnabled()
        compose.onNodeWithText("Enter a host:port or http(s) URL.").assertIsDisplayed()
        assertEquals(ConnectionState.InvalidEndpoint, viewModel.state.value.connectionState)
    }

    @Test
    fun manualEndpointConnectsRendersServerUiAndDispatchesAction() {
        val session = FakeSession()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-connect-button").performClick()
        compose.waitUntil { viewModel.state.value.connectionState == ConnectionState.Connected }

        assertEquals(EndpointResolution("10.0.2.2", 8080), session.connectedEndpoint)

        runBlocking {
            session.sink.onResponse(
                Control.ConnectResponse.newBuilder()
                    .setSetUi(
                        Ui.SetUI.newBuilder()
                            .setDeviceId("device-1")
                            .setRoot(
                                Ui.Node.newBuilder()
                                    .setId("root")
                                    .setButton(
                                        Ui.ButtonWidget.newBuilder()
                                            .setLabel("Server action")
                                            .setAction("submit"),
                                    ),
                            ),
                    )
                    .build(),
            )
        }

        compose.onNodeWithText("Server action").assertIsDisplayed()
        compose.onNodeWithText("Server action").performClick()
        compose.waitUntil { session.actions.isNotEmpty() }

        assertEquals(listOf(ServerDrivenAction("root", "submit", "pressed")), session.actions)
    }

    @Test
    fun serverDrivenDeviceControlsReachPlatformAdapters() {
        val session = FakeSession()
        val keepAwakeValues = mutableListOf<Boolean>()
        val fullscreenValues = mutableListOf<Boolean>()
        val brightnessValues = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                keepAwakeController = AndroidKeepAwakeController { keepAwakeValues += it },
                fullscreenController = AndroidFullscreenController { fullscreenValues += it },
                brightnessController = AndroidBrightnessController { brightnessValues += it },
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-connect-button").performClick()
        compose.waitUntil { viewModel.state.value.connectionState == ConnectionState.Connected }

        runBlocking {
            session.sink.onResponse(
                Control.ConnectResponse.newBuilder()
                    .setSetUi(
                        Ui.SetUI.newBuilder()
                            .setDeviceId("device-1")
                            .setRoot(
                                Ui.Node.newBuilder()
                                    .setId("root")
                                    .setStack(Ui.StackWidget.newBuilder())
                                    .addChildren(
                                        Ui.Node.newBuilder()
                                            .setId("keep-awake")
                                            .setKeepAwake(Ui.KeepAwakeWidget.newBuilder().setEnabled(true)),
                                    )
                                    .addChildren(
                                        Ui.Node.newBuilder()
                                            .setId("fullscreen")
                                            .setFullscreen(Ui.FullscreenWidget.newBuilder().setEnabled(true)),
                                    )
                                    .addChildren(
                                        Ui.Node.newBuilder()
                                            .setId("brightness")
                                            .setBrightness(Ui.BrightnessWidget.newBuilder().setValue(0.42)),
                                    ),
                            ),
                    )
                    .build(),
            )
        }

        compose.onNodeWithText("keep_awake=true").assertIsDisplayed()
        compose.onNodeWithText("fullscreen=true").assertIsDisplayed()
        compose.onNodeWithText("brightness=0.42").assertIsDisplayed()
        compose.waitUntil {
            keepAwakeValues == listOf(true) &&
                fullscreenValues == listOf(true) &&
                brightnessValues == listOf(0.42)
        }
    }

    @Test
    fun diagnosticsCanBeCopiedFromTerminalChrome() {
        var copied: String? = null
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                diagnosticClipboard = DiagnosticClipboard { copied = it },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-copy-diagnostics-button").performClick()

        compose.onNodeWithText("Diagnostics copy: copied").assertIsDisplayed()
        assertEquals(viewModel.state.value.diagnosticsText, copied)
    }

    @Test
    fun localKeepAwakeCanBeToggledFromTerminalChrome() {
        val keepAwakeValues = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                keepAwakeController = AndroidKeepAwakeController { keepAwakeValues += it },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithText("Keep awake off").assertIsDisplayed()
        compose.onNodeWithTag("terminal-local-keep-awake-button").performClick()

        compose.onNodeWithText("Keep awake on").assertIsDisplayed()
        compose.onNodeWithText("local_keep_awake=true", substring = true).assertIsDisplayed()
        assertEquals(listOf(true), keepAwakeValues)
    }

    @Test
    fun localFullscreenCanBeToggledFromTerminalChrome() {
        val fullscreenValues = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                fullscreenController = AndroidFullscreenController { fullscreenValues += it },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithText("Fullscreen off").assertIsDisplayed()
        compose.onNodeWithTag("terminal-local-fullscreen-button").performClick()

        compose.onNodeWithText("Fullscreen on").assertIsDisplayed()
        compose.onNodeWithText("local_fullscreen=true", substring = true).assertIsDisplayed()
        assertEquals(listOf(true), fullscreenValues)
    }

    @Test
    fun localBrightDisplayCanBeToggledFromTerminalChrome() {
        val brightnessValues = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                brightnessController = AndroidBrightnessController { brightnessValues += it },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithText("Bright display off").assertIsDisplayed()
        compose.onNodeWithTag("terminal-local-bright-display-button").performClick()

        compose.onNodeWithText("Bright display on").assertIsDisplayed()
        compose.onNodeWithText("local_bright_display=true", substring = true).assertIsDisplayed()
        assertEquals(listOf(1.0), brightnessValues)
    }

    @Test
    fun liveMediaTransportStatusIsVisibleWithoutPermissionWarnings() {
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                capabilityProbe = object : AndroidCapabilityProbe {
                    override fun current(): AndroidCapabilitySnapshotInput =
                        AndroidCapabilitySnapshotInput(
                            identity = Capabilities.DeviceIdentity.newBuilder()
                                .setDeviceName("test-tablet")
                                .setDeviceType("tablet")
                                .setPlatform("android")
                                .build(),
                            screenMetrics = AndroidScreenMetrics(
                                widthPx = 1280,
                                heightPx = 800,
                                density = 1f,
                                orientation = "landscape",
                            ),
                            permissions = PermissionCapabilityState(notificationsGranted = true),
                        )
                },
                mediaPermissionProbe = AndroidMediaPermissionProbe {
                    AndroidMediaPermissionState(
                        microphoneGranted = true,
                        cameraGranted = true,
                    )
                },
                webRtcAdapter = AndroidWebRtcAdapter.disabled("fire-os-webrtc-not-enabled"),
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-live-media-status").assertIsDisplayed()
        compose.onNodeWithText("Live media transport is unavailable: fire-os-webrtc-not-enabled.").assertIsDisplayed()
    }

    @Test
    fun lifecycleCapabilityRefreshReachesConnectedSession() {
        val session = FakeSession(capabilityDeltaResult = true)
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-connect-button").performClick()
        compose.waitUntil { viewModel.state.value.connectionState == ConnectionState.Connected }

        viewModel.refreshCapabilities("configuration")

        compose.waitUntil { session.capabilityRefreshReasons == listOf("configuration") }
        compose.onNodeWithText("last_capability_delta=configuration", substring = true).assertIsDisplayed()
    }

    @Test
    fun capabilityRefreshReflectsUpdatedDisplayOrientationDiagnostics() {
        val session = FakeSession(capabilityDeltaResult = true)
        var currentMetrics = AndroidScreenMetrics(
            widthPx = 1280,
            heightPx = 800,
            density = 2f,
            orientation = "landscape",
        )
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                capabilityProbe = object : AndroidCapabilityProbe {
                    override fun current(): AndroidCapabilitySnapshotInput =
                        AndroidCapabilitySnapshotInput(
                            identity = Capabilities.DeviceIdentity.newBuilder()
                                .setDeviceName("test-tablet")
                                .setDeviceType("tablet")
                                .setPlatform("android")
                                .build(),
                            screenMetrics = currentMetrics,
                            permissions = PermissionCapabilityState(
                                notificationsGranted = true,
                            ),
                            hardware = AndroidHardwareCapabilities(touchSupported = true),
                        )
                },
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }
        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-connect-button").performClick()
        compose.waitUntil { viewModel.state.value.connectionState == ConnectionState.Connected }

        currentMetrics = AndroidScreenMetrics(
            widthPx = 800,
            heightPx = 1280,
            density = 2f,
            orientation = "portrait",
        )
        viewModel.refreshCapabilities("configuration")

        compose.waitUntil { session.capabilityRefreshReasons == listOf("configuration") }
        compose.onNodeWithText("cap_orientation=portrait", substring = true).assertIsDisplayed()
        compose.onNodeWithText("cap_display_px=800x1280", substring = true).assertIsDisplayed()
    }

    @Test
    fun permissionRefreshShowsWarningsAfterRuntimePermissionLoss() {
        var snapshotPermissions = PermissionCapabilityState(
            microphoneGranted = true,
            cameraGranted = true,
            notificationsGranted = true,
        )
        var mediaPermissions = AndroidMediaPermissionState(
            microphoneGranted = true,
            cameraGranted = true,
        )
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                capabilityProbe = object : AndroidCapabilityProbe {
                    override fun current(): AndroidCapabilitySnapshotInput =
                        AndroidCapabilitySnapshotInput(
                            identity = Capabilities.DeviceIdentity.newBuilder()
                                .setDeviceName("test-tablet")
                                .setDeviceType("tablet")
                                .setPlatform("android")
                                .build(),
                            screenMetrics = AndroidScreenMetrics(
                                widthPx = 1280,
                                heightPx = 800,
                                density = 1f,
                                orientation = "landscape",
                            ),
                            permissions = snapshotPermissions,
                            hardware = AndroidHardwareCapabilities(
                                touchSupported = true,
                                microphone = true,
                                frontCamera = true,
                            ),
                        )
                },
                mediaPermissionProbe = AndroidMediaPermissionProbe { mediaPermissions },
                webRtcAdapter = AndroidWebRtcAdapter.disabled("fire-os-webrtc-not-enabled"),
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }
        compose.onNodeWithTag("terminal-live-media-status").assertIsDisplayed()

        snapshotPermissions = PermissionCapabilityState(
            microphoneGranted = false,
            cameraGranted = false,
            notificationsGranted = false,
        )
        mediaPermissions = AndroidMediaPermissionState(
            microphoneGranted = false,
            cameraGranted = false,
        )
        viewModel.refreshPermissionEducation("runtime-permission-change")

        compose.onNodeWithText(
            "Notifications are disabled; server notifications will stay in terminal diagnostics.",
        ).assertIsDisplayed()
        compose.onNodeWithText(
            "Microphone capture is unavailable until hardware and permission are both present.",
        ).assertIsDisplayed()
        compose.onNodeWithText(
            "Camera capture is unavailable until hardware and permission are both present.",
        ).assertIsDisplayed()
        compose.onNodeWithText("last_permission_refresh=runtime-permission-change", substring = true).assertIsDisplayed()
    }

    @Test
    fun discoveryErrorFallsBackToManualAndDiscoveredServerSelectionUpdatesEndpoint() {
        val discovery = FakeDiscovery()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                discovery = discovery,
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }

        compose.onNodeWithTag("terminal-discovery-start-button").performClick()
        compose.waitUntil { viewModel.state.value.discoveryState.lastError != null }
        compose.onNodeWithText("Discovery unavailable: multicast blocked").assertIsDisplayed()

        discovery.publish(
            DiscoveredServer(
                name = "Kitchen Screen",
                host = "192.168.1.25",
                port = 8443,
                lastSeenMillis = 1L,
                webSocketEndpoint = "ws://192.168.1.25:8443/connect",
            ),
        )

        compose.waitUntil { viewModel.state.value.discoveryState.servers.isNotEmpty() }
        compose.onNodeWithTag("terminal-discovered-server-192.168.1.25-8443").performClick()
        compose.waitUntil { viewModel.state.value.endpointText == "ws://192.168.1.25:8443/connect" }

        compose.onNodeWithText("No discovered servers").assertIsDisplayed()
        assertEquals("ws://192.168.1.25:8443/connect", viewModel.state.value.endpointText)
        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertTrue(discovery.stopped)
    }

    @Test
    fun heartbeatFailureReconnectsAndUpdatesDiagnostics() {
        val firstSession = FakeSession(heartbeatError = IllegalStateException("simulated-network-loss"))
        val secondSession = FakeSession()
        var sessionFactoryCalls = 0
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 25,
                reconnectPolicy = ReconnectPolicy(initialDelayMillis = 1, maxDelayMillis = 1),
                maxReconnectAttempts = 1,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                sessionFactory = { sink ->
                    sessionFactoryCalls += 1
                    when (sessionFactoryCalls) {
                        1 -> firstSession.also { it.sink = sink }
                        else -> secondSession.also { it.sink = sink }
                    }
                },
            ),
        )

        compose.setContent { AndroidTerminalApp(viewModel) }
        compose.onNodeWithTag("terminal-endpoint-field").performTextInput("10.0.2.2:8080")
        compose.onNodeWithTag("terminal-connect-button").performClick()
        compose.waitUntil { viewModel.state.value.connectionState == ConnectionState.Connected }
        compose.waitUntil { secondSession.connectedEndpoint != null }

        assertTrue(firstSession.closed)
        assertEquals(EndpointResolution("10.0.2.2", 8080), secondSession.connectedEndpoint)
        assertTrue(viewModel.state.value.diagnosticsText.contains("reconnect_success_attempt=1"))
    }

    private class FakeSession(
        private val capabilityDeltaResult: Boolean = false,
        private val heartbeatError: Throwable? = null,
    ) : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        var closed: Boolean = false
        val actions = mutableListOf<ServerDrivenAction>()
        val capabilityRefreshReasons = mutableListOf<String>()

        override suspend fun connect(endpoint: EndpointResolution) {
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() {
            heartbeatError?.let { throw it }
        }

        override suspend fun sendUiAction(action: ServerDrivenAction) {
            actions += action
        }

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean {
            capabilityRefreshReasons += reason
            return capabilityDeltaResult
        }

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() = Unit

        override suspend fun close() {
            closed = true
        }
    }

    private class FakeDiscovery : AndroidNsdDiscovery {
        private var onServer: ((DiscoveredServer) -> Unit)? = null
        private var onError: ((String) -> Unit)? = null
        var stopped: Boolean = false

        override fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit) {
            stopped = false
            this.onServer = onServer
            this.onError = onError
            onError("multicast blocked")
        }

        override fun stop() {
            stopped = true
            onServer = null
            onError = null
        }

        fun publish(server: DiscoveredServer) {
            onServer?.invoke(server)
        }
    }
}
