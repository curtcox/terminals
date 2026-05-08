package com.curtcox.terminals.android.smoke

import androidx.compose.ui.test.assertIsDisplayed
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
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.diagnostics.DiagnosticClipboard
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
import org.junit.Rule
import org.junit.Test
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.ui.v1.Ui

class AndroidTerminalAppSmokeTest {
    @get:Rule
    val compose = createComposeRule()

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

    private class FakeSession(
        private val capabilityDeltaResult: Boolean = false,
    ) : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        val actions = mutableListOf<ServerDrivenAction>()
        val capabilityRefreshReasons = mutableListOf<String>()

        override suspend fun connect(endpoint: EndpointResolution) {
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() = Unit

        override suspend fun sendUiAction(action: ServerDrivenAction) {
            actions += action
        }

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean {
            capabilityRefreshReasons += reason
            return capabilityDeltaResult
        }

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() = Unit

        override suspend fun close() = Unit
    }
}
