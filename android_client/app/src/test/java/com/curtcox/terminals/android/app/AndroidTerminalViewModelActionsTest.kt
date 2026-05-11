package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.control.v1.Control

@OptIn(ExperimentalCoroutinesApi::class)
class AndroidTerminalViewModelActionsTest : AndroidTerminalViewModelTestBase() {

    @Test
    fun uiActionIsSentThroughConnectedSession() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendUiAction(ServerDrivenAction("start", "tap"))
        advanceUntilIdle()

        assertEquals(ServerDrivenAction("start", "tap"), session.actions.single())
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun terminalKeyTextIsSentThroughConnectedSession() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendTerminalKeyText("ab")
        advanceUntilIdle()

        assertEquals(listOf("ab"), session.keyTexts)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun debugSystemQueriesAreSentThroughConnectedSession() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendRuntimeStatusQuery()
        viewModel.sendDeviceStatusQuery()
        advanceUntilIdle()

        assertEquals(
            listOf(
                "debug-runtime-status-1" to "runtime_status",
                "debug-device-status-2" to "device_status android-native-terminal",
            ),
            session.systemCommands,
        )
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_system_command=runtime_status:debug-runtime-status-1"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_system_command=device_status:debug-device-status-2"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun debugPlaybackQueriesMatchFlutterShell() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendPlaybackArtifactsQuery()
        viewModel.updatePlaybackArtifactId("artifact-a")
        viewModel.sendPlaybackMetadataQuery()
        advanceUntilIdle()

        assertEquals(
            listOf("debug-playback-artifacts-1" to "list_playback_artifacts"),
            session.systemCommands,
        )
        assertEquals(
            listOf(Triple("debug-playback-metadata-2", "artifact-a", "android-native-terminal")),
            session.playbackMetadataQueries,
        )
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_system_command=list_playback_artifacts:debug-playback-artifacts-1"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_manual_command=playback_metadata:debug-playback-metadata-2"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun playbackMetadataWithoutArtifactIdSetsLastError() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendPlaybackMetadataQuery()
        advanceUntilIdle()

        assertEquals("Playback artifact ID required", viewModel.state.value.lastError)
        assertTrue(session.playbackMetadataQueries.isEmpty())
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun playbackMetadataUsesExplicitTargetDeviceWhenProvided() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.updatePlaybackArtifactId("artifact-x")
        viewModel.updatePlaybackTargetDeviceId("subject-tablet-1")
        viewModel.sendPlaybackMetadataQuery()
        advanceUntilIdle()

        assertEquals(
            listOf(Triple("debug-playback-metadata-1", "artifact-x", "subject-tablet-1")),
            session.playbackMetadataQueries,
        )
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_manual_command=playback_metadata:debug-playback-metadata-1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun refreshCapabilitiesAsksConnectedSessionForDelta() = runTest(testDispatcher) {
        val session = FakeSession(capabilityDeltaSent = true)
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.refreshCapabilities("display_geometry_change")
        advanceUntilIdle()

        assertEquals(listOf("display_geometry_change"), session.capabilityDeltaReasons)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_capability_delta=display_geometry_change"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun staleGenerationProtocolErrorTriggersCapabilityRebaseline() = runTest(testDispatcher) {
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
    fun unrelatedProtocolErrorDoesNotRebaselineCapabilities() = runTest(testDispatcher) {
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
}
