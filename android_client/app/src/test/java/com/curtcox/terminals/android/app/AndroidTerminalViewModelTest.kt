package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import terminals.control.v1.Control
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
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()

        assertEquals(EndpointResolution("10.0.0.8", 8080), session.connectedEndpoint)
        assertEquals(ConnectionState.Connected, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("state=Connected"))
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

    private fun viewModel(session: FakeSession): AndroidTerminalViewModel =
        AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                sessionFactory = { sink ->
                    session.sink = sink
                    session
                },
            ),
        )

    private class FakeSession(
        private val connectError: Throwable? = null,
        private val capabilityDeltaSent: Boolean = false,
    ) : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink
        var connectedEndpoint: EndpointResolution? = null
        val actions = mutableListOf<ServerDrivenAction>()
        val capabilityDeltaReasons = mutableListOf<String>()
        var rebaselineCount = 0

        override suspend fun connect(endpoint: EndpointResolution) {
            connectError?.let { throw it }
            connectedEndpoint = endpoint
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() = Unit

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

        override suspend fun close() = Unit
    }
}
