package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.media.AndroidAudioPlayback
import com.curtcox.terminals.android.media.AndroidLiveMediaSession
import com.curtcox.terminals.android.media.AndroidMediaDisplay
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.LiveMediaSessionResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.AndroidTerminalSpeech
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus
import terminals.io.v1.Io
import terminals.ui.v1.Ui

@OptIn(ExperimentalCoroutinesApi::class)
class AndroidTerminalViewModelServerResponseTest : AndroidTerminalViewModelTestBase() {

    @Test
    fun serverSetUiResponseUpdatesRenderedRoot() = runTest(testDispatcher) {
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
        assertEquals("UI updated", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=UI updated"))
        assertEquals(1, viewModel.state.value.inboundConnectResponseCount)
        assertTrue(viewModel.state.value.diagnosticsText.contains("inbound_connect_response_count=1"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun inboundConnectResponseCountIncrementsPerControlMessage() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setHeartbeat(Control.Heartbeat.newBuilder().build())
                .build(),
        )
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setHeartbeat(Control.Heartbeat.newBuilder().build())
                .build(),
        )
        assertEquals(2, viewModel.state.value.inboundConnectResponseCount)
        assertTrue(viewModel.state.value.diagnosticsText.contains("inbound_connect_response_count=2"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverRegisterAckMessageIsSurfacedInDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-reg")
                        .setMessage("capabilities accepted"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("capabilities accepted", viewModel.state.value.registerAckMessage)
        assertEquals("srv-reg", viewModel.state.value.registerAckServerId)
        assertTrue(viewModel.state.value.diagnosticsText.contains("register_ack_message=capabilities accepted"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("register_ack_server_id=srv-reg"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun registerAckTriggersScenarioRegistryQueryOnce() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.systemCommands.clear()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-reg")
                        .setMessage("capabilities accepted"),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals(1, session.systemCommands.count { it.second == "scenario_registry" })
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-reg")
                        .setMessage("again"),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals(1, session.systemCommands.count { it.second == "scenario_registry" })
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun scenarioRegistryCommandResultUpdatesApplicationIntents() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.sendScenarioRegistryQuery()
        advanceUntilIdle()
        val requestId = session.systemCommands.last { it.second == "scenario_registry" }.first
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setCommandResult(
                    Control.CommandResult.newBuilder()
                        .setRequestId(requestId)
                        .putData("photo_frame", "")
                        .putData("terminal", ""),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals(listOf("terminal", "photo_frame"), viewModel.state.value.availableApplicationIntents)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun submitApplicationLaunchCommandSendsManualStart() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-launch")
                        .setMessage("registered"),
                )
                .build(),
        )
        advanceUntilIdle()
        viewModel.updateSelectedApplicationIntent("photo_frame")
        viewModel.submitApplicationLaunchCommand()
        advanceUntilIdle()
        assertEquals(listOf("photo_frame"), session.applicationLaunchCommands.map { it.second })
        assertTrue(session.applicationLaunchCommands.single().first.startsWith("debug-launch-app-"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun applicationLaunchQueuesUntilRegisterAckThenSends() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        viewModel.updateSelectedApplicationIntent("photo_frame")
        viewModel.submitApplicationLaunchCommand()
        advanceUntilIdle()
        assertTrue(session.applicationLaunchCommands.isEmpty())
        assertEquals("photo_frame", viewModel.state.value.applicationLaunchQueuedIntent)
        assertTrue(
            viewModel.state.value.diagnosticsText.contains("application_launch_queued_until_register_ack=photo_frame"),
        )

        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-q")
                        .setMessage("registered"),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals(null, viewModel.state.value.applicationLaunchQueuedIntent)
        assertEquals(listOf("photo_frame"), session.applicationLaunchCommands.map { it.second })
        assertTrue(session.applicationLaunchCommands.single().first.startsWith("debug-launch-app-"))
        assertEquals(1, session.systemCommands.count { it.second == "scenario_registry" })
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun registerAckMessageRemainsInDiagnosticsAfterDisconnect() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-once")
                        .setMessage("registered once"),
                )
                .build(),
        )
        advanceUntilIdle()

        viewModel.disconnect()
        advanceUntilIdle()

        assertEquals(ConnectionState.ReadyToConnect, viewModel.state.value.connectionState)
        assertTrue(viewModel.state.value.diagnosticsText.contains("register_ack_message=registered once"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("register_ack_server_id=srv-once"))
    }

    @Test
    fun serverBugReportAckIsSurfacedInDiagnostics() = runTest(testDispatcher) {
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
    fun serverBugReportAckRemainsInDiagnosticsAfterDisconnect() = runTest(testDispatcher) {
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
    fun newConnectClearsBugReportAckFromPriorSession() = runTest(testDispatcher) {
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
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun newConnectClearsRegisterAckMessageFromPriorSession() = runTest(testDispatcher) {
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
        first.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRegisterAck(
                    Control.RegisterAck.newBuilder()
                        .setServerId("srv-first")
                        .setMessage("first session ack"),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals("first session ack", viewModel.state.value.registerAckMessage)
        assertEquals("srv-first", viewModel.state.value.registerAckServerId)

        viewModel.disconnect()
        advanceUntilIdle()

        viewModel.connect()
        advanceUntilIdle()
        assertNull(viewModel.state.value.registerAckMessage)
        assertNull(viewModel.state.value.registerAckServerId)
        assertTrue(!viewModel.state.value.diagnosticsText.contains("register_ack_message=first session ack"))
        assertTrue(!viewModel.state.value.diagnosticsText.contains("register_ack_server_id=srv-first"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun newConnectClearsControlActivityFromPriorSession() = runTest(testDispatcher) {
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
        first.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setSetUi(
                    Ui.SetUI.newBuilder()
                        .setDeviceId("device-1")
                        .setRoot(
                            Ui.Node.newBuilder()
                                .setId("root")
                                .setText(Ui.TextWidget.newBuilder().setValue("v1")),
                        ),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals("UI updated", viewModel.state.value.lastControlResponseActivity)

        viewModel.disconnect()
        advanceUntilIdle()

        viewModel.connect()
        advanceUntilIdle()
        assertNull(viewModel.state.value.lastControlResponseActivity)
        assertTrue(!viewModel.state.value.diagnosticsText.contains("last_control_activity=UI updated"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun controlActivityRemainsInDiagnosticsAfterDisconnect() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setCommandResult(
                    Control.CommandResult.newBuilder()
                        .setRequestId("req-1")
                        .setNotification("done"),
                )
                .build(),
        )
        advanceUntilIdle()
        assertEquals("Command response", viewModel.state.value.lastControlResponseActivity)

        viewModel.disconnect()
        advanceUntilIdle()

        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Command response"))
    }

    @Test
    fun serverTransitionUiIsSurfacedInDiagnostics() = runTest(testDispatcher) {
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
    fun serverTransitionUiRemainsInDiagnosticsAfterNetworkRefresh() = runTest(testDispatcher) {
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
    fun serverHeartbeatIsSurfacedInDiagnostics() = runTest(testDispatcher) {
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
    fun serverCommandResultIsSurfacedInDiagnostics() = runTest(testDispatcher) {
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
        assertEquals("Command response", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Command response"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverControlErrorIsSurfacedInDiagnostics() = runTest(testDispatcher) {
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
                        .setMessage("stale generation"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals("stale generation", viewModel.state.value.lastError)
        assertEquals(
            Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION.name,
            viewModel.state.value.lastControlErrorCode,
        )
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_error=stale generation"))
        assertTrue(
            viewModel.state.value.diagnosticsText.contains(
                "last_control_error_code=${Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION.name}",
            ),
        )
        assertEquals("Server error", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Server error"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverControlErrorClearsAfterSuccessfulSetUi() = runTest(testDispatcher) {
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
                        .setMessage("stale generation"),
                )
                .build(),
        )
        advanceUntilIdle()
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setText(Ui.TextWidget.newBuilder().setValue("Recovered"))
            .build()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setSetUi(Ui.SetUI.newBuilder().setDeviceId("device-1").setRoot(root))
                .build(),
        )
        advanceUntilIdle()

        assertEquals(root, viewModel.state.value.serverRoot)
        assertNull(viewModel.state.value.lastError)
        assertNull(viewModel.state.value.lastControlErrorCode)
        val text = viewModel.state.value.diagnosticsText
        assertTrue(!text.contains("last_error=stale generation"))
        assertTrue(!text.contains("last_control_error_code="))
        assertEquals("UI updated", viewModel.state.value.lastControlResponseActivity)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun capabilityAckInvalidationsAndSnapshotAppliedAreSurfacedInDiagnostics() = runTest(testDispatcher) {
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
        assertEquals("Connected", viewModel.state.value.lastControlResponseActivity)
        assertTrue(text.contains("last_control_activity=Connected"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun opaqueStartStreamSummaryIsSurfacedInDiagnosticsAndClearsOnDisconnect() = runTest(testDispatcher) {
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

        assertEquals(listOf("s-out"), session.streamReadyIds)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_opaque_control_io="))
        assertTrue(viewModel.state.value.diagnosticsText.contains("stream_id=s-out"))
        assertTrue(
            viewModel.state.value.diagnosticsText.contains(
                "last_live_media=start_stream:s-out:live-media-session-not-implemented",
            ),
        )
        assertEquals(
            "start_stream:s-out:live-media-session-not-implemented",
            viewModel.state.value.lastLiveMediaLine,
        )
        assertEquals("Stream started", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Stream started"))
        viewModel.disconnect()
        advanceUntilIdle()
        assertNull(viewModel.state.value.lastOpaqueControlIoSummary)
        assertNull(viewModel.state.value.lastLiveMediaLine)
    }

    @Test
    fun startStreamWithWebRtcDisabledSurfacesAdapterReasonInLiveMediaDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            webRtcAdapter = AndroidWebRtcAdapter.disabled("custom-webrtc-off"),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStartStream(
                    Io.StartStream.newBuilder()
                        .setStreamId("sx")
                        .setStreamKind(Io.StreamKind.STREAM_KIND_VIDEO),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(
            viewModel.state.value.diagnosticsText.contains(
                "last_live_media=start_stream:sx:custom-webrtc-off",
            ),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun startStreamWithBlankIdDoesNotSendStreamReady() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(session)

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStartStream(
                    Io.StartStream.newBuilder()
                        .setStreamId("  ")
                        .setStreamKind(Io.StreamKind.STREAM_KIND_VIDEO),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(session.streamReadyIds.isEmpty())
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverNotificationIsDeliveredThroughPlatformAdapter() = runTest(testDispatcher) {
        val session = FakeSession()
        val delivered = mutableListOf<Pair<String, String>>()
        val spoken = mutableListOf<String>()
        val viewModel = viewModel(
            session,
            notificationDelivery = AndroidNotificationDelivery { title, body -> delivered += title to body },
            speechDelivery = AndroidTerminalSpeech { spoken += it },
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
        assertEquals(listOf("Done"), spoken)
        assertEquals("Timer", viewModel.state.value.lastNotificationTitle)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_notification=Timer"))
        assertEquals("Notification", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Notification"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverNotificationTitleOnlySpeaksTitle() = runTest(testDispatcher) {
        val session = FakeSession()
        val delivered = mutableListOf<Pair<String, String>>()
        val spoken = mutableListOf<String>()
        val viewModel = viewModel(
            session,
            notificationDelivery = AndroidNotificationDelivery { title, body -> delivered += title to body },
            speechDelivery = AndroidTerminalSpeech { spoken += it },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setNotification(
                    Ui.Notification.newBuilder()
                        .setDeviceId("device-1")
                        .setTitle("Alert")
                        .setBody(""),
                )
                .build(),
        )
        advanceUntilIdle()

        assertEquals(listOf("Alert" to ""), delivered)
        assertEquals(listOf("Alert"), spoken)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun serverNotificationBlankDoesNotDeliverOrSpeak() = runTest(testDispatcher) {
        val session = FakeSession()
        val delivered = mutableListOf<Pair<String, String>>()
        val spoken = mutableListOf<String>()
        val viewModel = viewModel(
            session,
            notificationDelivery = AndroidNotificationDelivery { title, body -> delivered += title to body },
            speechDelivery = AndroidTerminalSpeech { spoken += it },
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setNotification(
                    Ui.Notification.newBuilder()
                        .setDeviceId("device-1")
                        .setTitle("   ")
                        .setBody("\t"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(delivered.isEmpty())
        assertTrue(spoken.isEmpty())
        assertFalse(viewModel.state.value.diagnosticsText.contains("last_notification="))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun playAudioResponseIsDelegatedThroughMediaEngine() = runTest(testDispatcher) {
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
        assertEquals("Play audio", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Play audio"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun unsupportedShowMediaResponseIsRecordedInDiagnostics() = runTest(testDispatcher) {
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
        assertEquals("Show media", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Show media"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun showMediaResponseCanBeDelegatedThroughMediaEngine() = runTest(testDispatcher) {
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
        assertEquals("Show media", viewModel.state.value.lastControlResponseActivity)
        assertTrue(viewModel.state.value.diagnosticsText.contains("last_control_activity=Show media"))
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun startStreamWithSupportedSessionRecordsAppliedInLiveMediaDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(liveMedia = alwaysAppliedSession()),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStartStream(
                    Io.StartStream.newBuilder()
                        .setStreamId("s-applied")
                        .setStreamKind(Io.StreamKind.STREAM_KIND_AUDIO),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(
            viewModel.state.value.diagnosticsText.contains(
                "last_live_media=start_stream:s-applied:applied",
            ),
        )
        assertEquals("start_stream:s-applied:applied", viewModel.state.value.lastLiveMediaLine)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun webRtcSignalWithSupportedSessionRecordsAppliedInLiveMediaDiagnostics() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(liveMedia = alwaysAppliedSession()),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setWebrtcSignal(
                    Control.WebRTCSignal.newBuilder()
                        .setStreamId("sig-1")
                        .setSignalTypeEnum(Control.WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_OFFER)
                        .setPayload("sdp-offer"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(
            viewModel.state.value.diagnosticsText.contains("last_live_media=webrtc_signal:sig-1:applied"),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun stopStreamWithSupportedSessionAppliesCleanly() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(liveMedia = alwaysAppliedSession()),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStartStream(Io.StartStream.newBuilder().setStreamId("s2").build())
                .build(),
        )
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setStopStream(Io.StopStream.newBuilder().setStreamId("s2").build())
                .build(),
        )
        advanceUntilIdle()

        assertTrue(
            viewModel.state.value.diagnosticsText.contains("last_live_media=start_stream:s2:applied"),
        )
        viewModel.disconnect()
        advanceUntilIdle()
    }

    @Test
    fun routeStreamWithSupportedSessionAppliesCleanly() = runTest(testDispatcher) {
        val session = FakeSession()
        val viewModel = viewModel(
            session,
            mediaEngine = AndroidMediaEngine(liveMedia = alwaysAppliedSession()),
        )

        viewModel.updateEndpoint("10.0.0.8:8080")
        viewModel.connect()
        advanceUntilIdle()
        session.sink.onResponse(
            Control.ConnectResponse.newBuilder()
                .setRouteStream(
                    Io.RouteStream.newBuilder()
                        .setStreamId("s3")
                        .setSourceDeviceId("src")
                        .setTargetDeviceId("tgt"),
                )
                .build(),
        )
        advanceUntilIdle()

        assertTrue(viewModel.state.value.inboundConnectResponseCount >= 1)
        viewModel.disconnect()
        advanceUntilIdle()
    }

    private fun alwaysAppliedSession(): AndroidLiveMediaSession =
        object : AndroidLiveMediaSession {
            override fun applyStartStream(start: Io.StartStream) = LiveMediaSessionResult.Applied
            override fun applyStopStream(streamId: String) = LiveMediaSessionResult.Applied
            override fun applyRouteStream(route: Io.RouteStream) = LiveMediaSessionResult.Applied
            override fun applyWebRtcSignal(signal: Control.WebRTCSignal) = LiveMediaSessionResult.Applied
            override fun stopLocalCaptureStreamsForPrivacy() = Unit
        }
}
