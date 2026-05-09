package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.app.AndroidTerminalViewState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import com.google.protobuf.ByteString
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus
import terminals.io.v1.Io
import terminals.ui.v1.Ui

class ControlResponseDispatcherTest {
    private val dispatcher = ControlResponseDispatcher()

    @Test
    fun payloadNotSetLeavesStateUnchanged() {
        val seeded = AndroidTerminalViewState(
            serverRoot = textNode("x", "y"),
            lastTransition = "hold",
            lastOpaqueControlIoSummary = "type=stop_stream stream_id=z",
        )
        val next = dispatcher.dispatch(seeded, Control.ConnectResponse.getDefaultInstance())

        assertEquals(seeded, next)
    }

    @Test
    fun setUiReplacesRoot() {
        val root = textNode("title", "Ready")
        val response = Control.ConnectResponse.newBuilder()
            .setSetUi(Ui.SetUI.newBuilder().setDeviceId("device-1").setRoot(root))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(root, next.serverRoot)
    }

    @Test
    fun updateUiPatchesTargetWithoutReplacingSiblings() {
        val keep = textNode("keep", "Keep")
        val stale = textNode("replace", "Old")
        val fresh = textNode("replace", "New")
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(keep)
            .addChildren(stale)
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(Ui.UpdateUI.newBuilder().setDeviceId("device-1").setComponentId("replace").setNode(fresh))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(serverRoot = root), response)

        assertEquals(keep, next.serverRoot!!.childrenList[0])
        assertEquals(fresh, next.serverRoot!!.childrenList[1])
    }

    @Test
    fun updateUiWithoutRootIsIgnored() {
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("missing")
                    .setNode(textNode("missing", "Ignored")),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertNull(next.serverRoot)
        assertEquals("UI patched", next.lastControlResponseActivity)
    }

    @Test
    fun updateUiWithBlankComponentIdReplacesEntireRoot() {
        val oldRoot = textNode("root", "Old")
        val fresh = textNode("root", "New")
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("   ")
                    .setNode(fresh),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(serverRoot = oldRoot), response)

        assertEquals(fresh, next.serverRoot)
    }

    @Test
    fun updateUiWithBlankComponentIdSetsRootWhenPreviouslyNull() {
        val fresh = textNode("solo", "Only")
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("")
                    .setNode(fresh),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(fresh, next.serverRoot)
        assertEquals("UI patched", next.lastControlResponseActivity)
    }

    @Test
    fun updateUiWithoutNodeLeavesExistingRootUnchanged() {
        val root = textNode("keep", "Value")
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("any"),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(serverRoot = root), response)

        assertEquals(root, next.serverRoot)
        assertEquals("UI patched", next.lastControlResponseActivity)
    }

    @Test
    fun updateUiPatchesChildTargetedByPropsIdWhenProtobufIdBlank() {
        val target = Ui.Node.newBuilder()
            .putProps("id", "target")
            .setText(Ui.TextWidget.newBuilder().setValue("Old").build())
            .build()
        val keep = textNode("keep", "Keep")
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(keep)
            .addChildren(target)
            .build()
        val fresh = Ui.Node.newBuilder()
            .putProps("id", "target")
            .setText(Ui.TextWidget.newBuilder().setValue("New").build())
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("target")
                    .setNode(fresh),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(serverRoot = root), response)

        assertEquals(keep, next.serverRoot!!.childrenList[0])
        assertEquals(fresh, next.serverRoot.childrenList[1])
    }

    @Test
    fun updateUiWithUnknownTargetLeavesTreeUnchanged() {
        val keep = textNode("keep", "Keep")
        val root = Ui.Node.newBuilder()
            .setId("root")
            .setStack(Ui.StackWidget.getDefaultInstance())
            .addChildren(keep)
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setUpdateUi(
                Ui.UpdateUI.newBuilder()
                    .setDeviceId("device-1")
                    .setComponentId("ghost")
                    .setNode(textNode("ghost", "Ignored")),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(serverRoot = root), response)

        assertEquals(root, next.serverRoot)
    }

    @Test
    fun bugReportAckRecordsDiagnosticsChrome() {
        val ack = BugReportAck.newBuilder()
            .setReportId("br-7")
            .setCorrelationId("c1")
            .setStatus(BugReportStatus.BUG_REPORT_STATUS_FILED)
            .setReportPath("logs/bug_reports/x.json")
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setBugReportAck(ack)
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastBugReportAckDiagnostics!!.contains("bug_report_id=br-7"))
        assertTrue(next.lastBugReportAckDiagnostics.contains("bug_report_status=filed"))
    }

    @Test
    fun transitionUiRecordsLastTransition() {
        val response = Control.ConnectResponse.newBuilder()
            .setTransitionUi(
                Ui.TransitionUI.newBuilder()
                    .setDeviceId("device-1")
                    .setTransition("fade")
                    .setDurationMs(120),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("fade", next.lastTransition)
        assertEquals(120L, next.lastTransitionDurationMs)
    }

    @Test
    fun transitionUiWithBlankTransitionClearsPriorTransition() {
        val seeded = AndroidTerminalViewState(
            lastTransition = "fade",
            lastTransitionDurationMs = 99L,
        )
        val response = Control.ConnectResponse.newBuilder()
            .setTransitionUi(
                Ui.TransitionUI.newBuilder()
                    .setDeviceId("device-1")
                    .setTransition("")
                    .setDurationMs(0),
            )
            .build()

        val next = dispatcher.dispatch(seeded, response)

        assertNull(next.lastTransition)
        assertNull(next.lastTransitionDurationMs)
    }

    @Test
    fun helloAckRecordsHandshakeFields() {
        val response = Control.ConnectResponse.newBuilder()
            .setHelloAck(
                Control.HelloAck.newBuilder()
                    .setServerId("srv-1")
                    .setSessionId("sess-9")
                    .setHeartbeatIntervalMs(12_000),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("srv-1", next.controlServerId)
        assertEquals("sess-9", next.controlSessionId)
        assertEquals(12_000L, next.serverHeartbeatIntervalMs)
    }

    @Test
    fun helloAckWithNonPositiveHeartbeatDoesNotRecordServerInterval() {
        val response = Control.ConnectResponse.newBuilder()
            .setHelloAck(
                Control.HelloAck.newBuilder()
                    .setServerId("srv-2")
                    .setSessionId("sess-2")
                    .setHeartbeatIntervalMs(0),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("srv-2", next.controlServerId)
        assertEquals("sess-2", next.controlSessionId)
        assertNull(next.serverHeartbeatIntervalMs)
    }

    @Test
    fun registerAckPrefersTypedServerMetadataAndFallsBackToMetadataMap() {
        val responseTyped = Control.ConnectResponse.newBuilder()
            .setRegisterAck(
                Control.RegisterAck.newBuilder()
                    .setServerId("reg-srv-typed")
                    .setMessage("device registered")
                    .setServerMetadata(
                        Control.ServerMetadata.newBuilder()
                            .setBuild(
                                Control.BuildMetadata.newBuilder()
                                    .setSha("abc")
                                    .setDateRfc3339("2026-05-08T12:00:00Z"),
                            )
                            .setPhotoFrameAssetBaseUrl("https://example/static/"),
                    ),
            )
            .build()
        val afterTyped = dispatcher.dispatch(AndroidTerminalViewState(), responseTyped)
        assertEquals("abc", afterTyped.serverBuildSha)
        assertEquals("2026-05-08T12:00:00Z", afterTyped.serverBuildDate)
        assertEquals("device registered", afterTyped.registerAckMessage)
        assertEquals("reg-srv-typed", afterTyped.registerAckServerId)
        assertEquals("https://example/static/", afterTyped.registerAckAssetBaseUrl)

        val responseMap = Control.ConnectResponse.newBuilder()
            .setRegisterAck(
                Control.RegisterAck.newBuilder()
                    .putMetadata("server_build_sha", "from-map")
                    .putMetadata("server_build_date", "2026-01-01T00:00:00Z"),
            )
            .build()
        val afterMap = dispatcher.dispatch(AndroidTerminalViewState(), responseMap)
        assertEquals("from-map", afterMap.serverBuildSha)
        assertEquals("2026-01-01T00:00:00Z", afterMap.serverBuildDate)
    }

    @Test
    fun registerAckBlankMessagePreservesPriorMessage() {
        val first = Control.ConnectResponse.newBuilder()
            .setRegisterAck(
                Control.RegisterAck.newBuilder()
                    .setMessage("first ack"),
            )
            .build()
        val afterFirst = dispatcher.dispatch(AndroidTerminalViewState(), first)
        assertEquals("first ack", afterFirst.registerAckMessage)

        val second = Control.ConnectResponse.newBuilder()
            .setRegisterAck(Control.RegisterAck.newBuilder())
            .build()
        val afterSecond = dispatcher.dispatch(afterFirst, second)
        assertEquals("first ack", afterSecond.registerAckMessage)
    }

    @Test
    fun registerAckBlankServerIdPreservesPriorServerId() {
        val first = Control.ConnectResponse.newBuilder()
            .setRegisterAck(
                Control.RegisterAck.newBuilder()
                    .setServerId("srv-a"),
            )
            .build()
        val afterFirst = dispatcher.dispatch(AndroidTerminalViewState(), first)
        assertEquals("srv-a", afterFirst.registerAckServerId)

        val second = Control.ConnectResponse.newBuilder()
            .setRegisterAck(Control.RegisterAck.newBuilder().setMessage("follow-up"))
            .build()
        val afterSecond = dispatcher.dispatch(afterFirst, second)
        assertEquals("srv-a", afterSecond.registerAckServerId)
        assertEquals("follow-up", afterSecond.registerAckMessage)
    }

    @Test
    fun capabilityAckRecordsAcceptedGeneration() {
        val response = Control.ConnectResponse.newBuilder()
            .setCapabilityAck(
                Control.CapabilityAck.newBuilder()
                    .setDeviceId("device-1")
                    .setAcceptedGeneration(42),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(42L, next.lastCapabilityAckGeneration)
        assertEquals(false, next.lastCapabilityAckSnapshotApplied)
        assertNull(next.lastCapabilityInvalidationsSummary)
    }

    @Test
    fun capabilityAckRecordsSnapshotAppliedAndInvalidationsSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setCapabilityAck(
                Control.CapabilityAck.newBuilder()
                    .setDeviceId("device-1")
                    .setAcceptedGeneration(3)
                    .setSnapshotApplied(true)
                    .addInvalidations(
                        Control.ResourceInvalidation.newBuilder()
                            .setResource("mic.capture")
                            .setReason("capability_lost"),
                    )
                    .addInvalidations(
                        Control.ResourceInvalidation.newBuilder()
                            .setResource("camera.capture")
                            .setReason("capability_lost"),
                    ),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(3L, next.lastCapabilityAckGeneration)
        assertTrue(next.lastCapabilityAckSnapshotApplied)
        assertEquals(
            "mic.capture:capability_lost; camera.capture:capability_lost",
            next.lastCapabilityInvalidationsSummary,
        )
    }

    @Test
    fun capabilityAckInvalidationsSummaryTruncatesLongLists() {
        val builder = Control.CapabilityAck.newBuilder()
            .setDeviceId("device-1")
            .setAcceptedGeneration(1)
        repeat(6) { i ->
            builder.addInvalidations(
                Control.ResourceInvalidation.newBuilder()
                    .setResource("res-$i")
                    .setReason("gone"),
            )
        }
        val response = Control.ConnectResponse.newBuilder().setCapabilityAck(builder).build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastCapabilityInvalidationsSummary!!.contains("res-0:gone"))
        assertTrue(next.lastCapabilityInvalidationsSummary.contains("+2 more"))
    }

    @Test
    fun serverHeartbeatRecordsLastServerUnixMs() {
        val response = Control.ConnectResponse.newBuilder()
            .setHeartbeat(
                Control.Heartbeat.newBuilder()
                    .setDeviceId("device-1")
                    .setUnixMs(1_700_000_000_000L),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(1_700_000_000_000L, next.lastServerHeartbeatUnixMs)
    }

    @Test
    fun serverHeartbeatWithoutTimestampClearsRecordedValue() {
        val seeded = AndroidTerminalViewState(lastServerHeartbeatUnixMs = 42L)
        val response = Control.ConnectResponse.newBuilder()
            .setHeartbeat(Control.Heartbeat.newBuilder().setDeviceId("device-1"))
            .build()

        val next = dispatcher.dispatch(seeded, response)

        assertNull(next.lastServerHeartbeatUnixMs)
    }

    @Test
    fun commandResultRecordsRequestIdAndNotification() {
        val response = Control.ConnectResponse.newBuilder()
            .setCommandResult(
                Control.CommandResult.newBuilder()
                    .setRequestId("cmd-7")
                    .setNotification("Started timer"),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("cmd-7", next.lastCommandResultRequestId)
        assertEquals("Started timer", next.lastCommandResultNotification)
    }

    @Test
    fun commandResultWithEmptyFieldsClearsPriorDiagnosticsFields() {
        val seeded = AndroidTerminalViewState(
            lastCommandResultRequestId = "old",
            lastCommandResultNotification = "old-note",
        )
        val response = Control.ConnectResponse.newBuilder()
            .setCommandResult(Control.CommandResult.newBuilder())
            .build()

        val next = dispatcher.dispatch(seeded, response)

        assertNull(next.lastCommandResultRequestId)
        assertNull(next.lastCommandResultNotification)
    }

    @Test
    fun startStreamRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setStartStream(
                Io.StartStream.newBuilder()
                    .setStreamId("stream-1")
                    .setStreamKind(Io.StreamKind.STREAM_KIND_VIDEO)
                    .setKind("custom"),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastOpaqueControlIoSummary!!.contains("type=start_stream"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("stream_id=stream-1"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("STREAM_KIND_VIDEO"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("kind=custom"))
    }

    @Test
    fun stopStreamRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setStopStream(Io.StopStream.newBuilder().setStreamId("s9"))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("type=stop_stream stream_id=s9", next.lastOpaqueControlIoSummary)
    }

    @Test
    fun routeStreamRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setRouteStream(
                Io.RouteStream.newBuilder()
                    .setStreamId("r1")
                    .setStreamKind(Io.StreamKind.STREAM_KIND_AUDIO)
                    .setKind("upstream"),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastOpaqueControlIoSummary!!.contains("type=route_stream"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("stream_id=r1"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("STREAM_KIND_AUDIO"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("kind=upstream"))
    }

    @Test
    fun removeBundleRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setRemoveBundle(Io.RemoveBundle.newBuilder().setBundleId("bundle-z"))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("type=remove_bundle bundle_id=bundle-z", next.lastOpaqueControlIoSummary)
    }

    @Test
    fun startFlowRecordsOpaqueSummaryWithPlanCounts() {
        val plan = Io.FlowPlan.newBuilder()
            .addNodes(Io.FlowNode.newBuilder().setId("n1"))
            .addEdges(Io.FlowEdge.newBuilder().setFrom("n1").setTo("n2"))
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setStartFlow(
                Io.StartFlow.newBuilder()
                    .setFlowId("flow-a")
                    .setPlan(plan),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(
            "type=start_flow flow_id=flow-a nodes=1 edges=1",
            next.lastOpaqueControlIoSummary,
        )
    }

    @Test
    fun patchFlowRecordsOpaqueSummaryWithPlanCounts() {
        val plan = Io.FlowPlan.newBuilder()
            .addNodes(Io.FlowNode.newBuilder().setId("a"))
            .addNodes(Io.FlowNode.newBuilder().setId("b"))
            .build()
        val response = Control.ConnectResponse.newBuilder()
            .setPatchFlow(
                Io.PatchFlow.newBuilder()
                    .setFlowId("flow-b")
                    .setPlan(plan),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals(
            "type=patch_flow flow_id=flow-b nodes=2 edges=0",
            next.lastOpaqueControlIoSummary,
        )
    }

    @Test
    fun stopFlowRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setStopFlow(Io.StopFlow.newBuilder().setFlowId("flow-c"))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("type=stop_flow flow_id=flow-c", next.lastOpaqueControlIoSummary)
    }

    @Test
    fun requestArtifactRecordsOpaqueSummary() {
        val response = Control.ConnectResponse.newBuilder()
            .setRequestArtifact(Io.RequestArtifact.newBuilder().setArtifactId("art-99"))
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertEquals("type=request_artifact artifact_id=art-99", next.lastOpaqueControlIoSummary)
    }

    @Test
    fun webRtcSignalRecordsOpaqueSummaryWithoutPayloadBody() {
        val response = Control.ConnectResponse.newBuilder()
            .setWebrtcSignal(
                Control.WebRTCSignal.newBuilder()
                    .setStreamId("webrtc-1")
                    .setSignalTypeEnum(Control.WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_OFFER)
                    .setPayload("secret-sdp-should-not-appear-in-summary"),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastOpaqueControlIoSummary!!.contains("type=webrtc_signal"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("stream_id=webrtc-1"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("WEB_RTC_SIGNAL_TYPE_OFFER"))
        assertTrue(!next.lastOpaqueControlIoSummary.contains("secret-sdp"))
    }

    @Test
    fun installBundleRecordsOpaqueSummaryWithTarSize() {
        val response = Control.ConnectResponse.newBuilder()
            .setInstallBundle(
                Io.InstallBundle.newBuilder()
                    .setBundleId("b1")
                    .setVersion("1.0.0")
                    .setSha256("abcdef0123456789abcdef0123456789")
                    .setTarGz(ByteString.copyFrom(ByteArray(500))),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertTrue(next.lastOpaqueControlIoSummary!!.contains("type=install_bundle"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("bundle_id=b1"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("version=1.0.0"))
        assertTrue(next.lastOpaqueControlIoSummary.contains("sha256_prefix=abcdef012345..."))
        assertTrue(next.lastOpaqueControlIoSummary.contains("tar_gz_bytes=500"))
    }

    @Test
    fun playAudioClearsOpaqueSummary() {
        val seeded = AndroidTerminalViewState(
            lastOpaqueControlIoSummary = "type=start_stream stream_id=x",
        )
        val response = Control.ConnectResponse.newBuilder()
            .setPlayAudio(Io.PlayAudio.newBuilder().setRequestId("a1").setUrl("https://example/a.mp3"))
            .build()

        val next = dispatcher.dispatch(seeded, response)

        assertNull(next.lastOpaqueControlIoSummary)
        assertEquals("Play audio", next.lastControlResponseActivity)
    }

    @Test
    fun showMediaClearsOpaqueSummary() {
        val seeded = AndroidTerminalViewState(
            lastOpaqueControlIoSummary = "type=webrtc_signal stream_id=x",
        )
        val response = Control.ConnectResponse.newBuilder()
            .setShowMedia(
                Io.ShowMedia.newBuilder()
                    .setRequestId("m1")
                    .setMediaUrl("https://example/v.mp4"),
            )
            .build()

        val next = dispatcher.dispatch(seeded, response)

        assertNull(next.lastOpaqueControlIoSummary)
        assertEquals("Show media", next.lastControlResponseActivity)
    }

    @Test
    fun dispatchRecordsControlResponseActivityForOpaqueIoAndHandshake() {
        val start = Control.ConnectResponse.newBuilder()
            .setStartStream(Io.StartStream.newBuilder().setStreamId("s1"))
            .build()
        val hello = Control.ConnectResponse.newBuilder()
            .setHelloAck(
                Control.HelloAck.newBuilder()
                    .setServerId("srv")
                    .setSessionId("sess")
                    .setHeartbeatIntervalMs(5_000),
            )
            .build()

        val afterStart = dispatcher.dispatch(AndroidTerminalViewState(), start)
        assertEquals("Stream started", afterStart.lastControlResponseActivity)

        val afterHello = dispatcher.dispatch(afterStart, hello)
        assertEquals("Connected", afterHello.lastControlResponseActivity)
    }

    @Test
    fun notificationAndErrorUpdateGenericTerminalState() {
        val notification = Control.ConnectResponse.newBuilder()
            .setNotification(
                Ui.Notification.newBuilder()
                    .setDeviceId("device-1")
                    .setTitle("Timer")
                    .setBody("Done"),
            )
            .build()
        val error = Control.ConnectResponse.newBuilder()
            .setError(
                Control.ControlError.newBuilder()
                    .setCode(Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION)
                    .setMessage("protocol violation"),
            )
            .build()

        val afterNotification = dispatcher.dispatch(AndroidTerminalViewState(), notification)
        val afterError = dispatcher.dispatch(afterNotification, error)

        assertEquals("Timer", afterError.lastNotificationTitle)
        assertEquals("Done", afterError.lastNotificationBody)
        assertEquals("protocol violation", afterError.lastError)
        assertEquals(
            Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION.name,
            afterError.lastControlErrorCode,
        )
        assertEquals("Notification", afterNotification.lastControlResponseActivity)
        assertEquals("Server error", afterError.lastControlResponseActivity)
    }

    @Test
    fun errorWithEmptyMessageStillRecordsControlErrorCode() {
        val response = Control.ConnectResponse.newBuilder()
            .setError(
                Control.ControlError.newBuilder()
                    .setCode(Control.ControlErrorCode.CONTROL_ERROR_CODE_UNKNOWN),
            )
            .build()

        val next = dispatcher.dispatch(AndroidTerminalViewState(), response)

        assertNull(next.lastError)
        assertEquals(Control.ControlErrorCode.CONTROL_ERROR_CODE_UNKNOWN.name, next.lastControlErrorCode)
    }

    private fun textNode(id: String, value: String): Ui.Node =
        Ui.Node.newBuilder()
            .setId(id)
            .setText(Ui.TextWidget.newBuilder().setValue(value))
            .build()
}
