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
    fun registerAckPrefersTypedServerMetadataAndFallsBackToMetadataMap() {
        val responseTyped = Control.ConnectResponse.newBuilder()
            .setRegisterAck(
                Control.RegisterAck.newBuilder()
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
            .setError(Control.ControlError.newBuilder().setMessage("protocol violation"))
            .build()

        val afterNotification = dispatcher.dispatch(AndroidTerminalViewState(), notification)
        val afterError = dispatcher.dispatch(afterNotification, error)

        assertEquals("Timer", afterError.lastNotificationTitle)
        assertEquals("Done", afterError.lastNotificationBody)
        assertEquals("protocol violation", afterError.lastError)
    }

    private fun textNode(id: String, value: String): Ui.Node =
        Ui.Node.newBuilder()
            .setId(id)
            .setText(Ui.TextWidget.newBuilder().setValue(value))
            .build()
}
