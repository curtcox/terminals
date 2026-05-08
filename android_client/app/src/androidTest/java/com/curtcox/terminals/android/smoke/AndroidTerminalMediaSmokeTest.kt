package com.curtcox.terminals.android.smoke

import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
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
import com.curtcox.terminals.android.media.AndroidAudioPlayback
import com.curtcox.terminals.android.media.AndroidMediaDisplay
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import terminals.control.v1.Control
import terminals.io.v1.Io

class AndroidTerminalMediaSmokeTest {
    @get:Rule
    val compose = createComposeRule()

    @Test
    fun serverMediaCommandsDispatchThroughMediaEngineAndUpdateDiagnostics() {
        val session = FakeSession()
        val audioRequests = mutableListOf<String>()
        val mediaRequests = mutableListOf<String>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                buildMetadata = AndroidBuildMetadata("0.1.0-test", "sha", "date"),
                heartbeatIntervalMillis = 0,
                terminalSettings = AndroidTerminalSettings.inMemory(),
                mediaEngine = AndroidMediaEngine(
                    audioPlayback = AndroidAudioPlayback { command ->
                        audioRequests += command.requestId
                        AudioPlaybackResult.Played(command.requestId)
                    },
                    mediaDisplay = AndroidMediaDisplay { command ->
                        mediaRequests += command.requestId
                        MediaDisplayResult.Shown(command.requestId)
                    },
                ),
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
                    .setPlayAudio(
                        Io.PlayAudio.newBuilder()
                            .setRequestId("audio-123")
                            .setDeviceId("device-1")
                            .setTtsText("hello from server"),
                    )
                    .build(),
            )
            session.sink.onResponse(
                Control.ConnectResponse.newBuilder()
                    .setShowMedia(
                        Io.ShowMedia.newBuilder()
                            .setRequestId("media-456")
                            .setDeviceId("device-1")
                            .setMediaUrl("https://example.test/clip.mp4")
                            .setMediaType("video/mp4"),
                    )
                    .build(),
            )
        }

        compose.waitUntil { audioRequests == listOf("audio-123") && mediaRequests == listOf("media-456") }
        compose.waitUntil { viewModel.state.value.lastMediaRequestId == "media-456" }

        assertEquals(listOf("audio-123"), audioRequests)
        assertEquals(listOf("media-456"), mediaRequests)
        assertEquals("media-456", viewModel.state.value.lastMediaRequestId)
        assertEquals("shown", viewModel.state.value.lastMediaStatus)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_media=media-456:shown"))
    }

    private class FakeSession : AndroidControlSession {
        override var status: ControlSessionStatus = ControlSessionStatus()
        lateinit var sink: AndroidControlResponseSink

        override suspend fun connect(endpoint: EndpointResolution) {
            status = status.copy(connected = true, endpoint = endpoint)
        }

        override suspend fun sendHeartbeat() = Unit

        override suspend fun sendUiAction(action: ServerDrivenAction) = Unit

        override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean = false

        override suspend fun rebaselineCapabilitiesAfterStaleGeneration() = Unit

        override suspend fun close() = Unit
    }
}
