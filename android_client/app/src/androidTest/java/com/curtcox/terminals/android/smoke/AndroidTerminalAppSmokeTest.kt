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
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
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

    private class FakeSession : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        val actions = mutableListOf<ServerDrivenAction>()

        override suspend fun connect(endpoint: EndpointResolution) {
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() = Unit

        override suspend fun sendUiAction(action: ServerDrivenAction) {
            actions += action
        }

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean = false

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() = Unit

        override suspend fun close() = Unit
    }
}
