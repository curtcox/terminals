package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Test
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics
import terminals.io.v1.Io
import terminals.ui.v1.Ui

/**
 * Human-readable activity labels must stay aligned with Flutter
 * `statusFromConnectResponse` in `terminal_client/lib/connection/control_response_dispatcher.dart`
 * so support copy/paste and cross-client debugging stay consistent.
 */
class ConnectResponseActivityStatusTest {

    @Test
    fun handshakeAndCapabilityResponsesUseConnectedLabelLikeFlutterDefault() {
        assertEquals(
            "Connected",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setHelloAck(
                        Control.HelloAck.newBuilder()
                            .setServerId("srv")
                            .setSessionId("sess")
                            .setHeartbeatIntervalMs(3000),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "Connected",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setCapabilityAck(
                        Control.CapabilityAck.newBuilder()
                            .setDeviceId("d")
                            .setAcceptedGeneration(2)
                            .setSnapshotApplied(true),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "Connected",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setHeartbeat(Control.Heartbeat.newBuilder().setUnixMs(1_700_000_000_000L))
                    .build(),
            ),
        )
        assertEquals("Connected", connectResponseActivityStatus(Control.ConnectResponse.getDefaultInstance()))
    }

    @Test
    fun flutterOrderedStatusLabels() {
        assertEquals(
            "Server error",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setError(Control.ControlError.newBuilder().setMessage("boom"))
                    .build(),
            ),
        )
        assertEquals(
            "UI transition",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setTransitionUi(Ui.TransitionUI.newBuilder().setTransition("fade").setDurationMs(200))
                    .build(),
            ),
        )
        assertEquals(
            "Stream started",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setStartStream(Io.StartStream.newBuilder().setStreamId("s1"))
                    .build(),
            ),
        )
        assertEquals(
            "Stream stopped",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setStopStream(Io.StopStream.newBuilder().setStreamId("s1"))
                    .build(),
            ),
        )
        assertEquals(
            "Route updated",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setRouteStream(Io.RouteStream.newBuilder().setStreamId("r1"))
                    .build(),
            ),
        )
        assertEquals(
            "Notification",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setNotification(
                        Ui.Notification.newBuilder().setTitle("t").setBody("b"),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "WebRTC signal",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setWebrtcSignal(Control.WebRTCSignal.newBuilder().setStreamId("w1"))
                    .build(),
            ),
        )
        assertEquals(
            "Play audio",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setPlayAudio(Io.PlayAudio.newBuilder().setRequestId("a1"))
                    .build(),
            ),
        )
        assertEquals(
            "Show media",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setShowMedia(Io.ShowMedia.newBuilder().setRequestId("m1"))
                    .build(),
            ),
        )
        assertEquals(
            "Bundle install requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setInstallBundle(Io.InstallBundle.newBuilder().setBundleId("b1"))
                    .build(),
            ),
        )
        assertEquals(
            "Bundle removal requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setRemoveBundle(Io.RemoveBundle.newBuilder().setBundleId("b1"))
                    .build(),
            ),
        )
        assertEquals(
            "Flow start requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setStartFlow(Io.StartFlow.newBuilder().setFlowId("f1"))
                    .build(),
            ),
        )
        assertEquals(
            "Flow patch requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setPatchFlow(Io.PatchFlow.newBuilder().setFlowId("f1"))
                    .build(),
            ),
        )
        assertEquals(
            "Flow stop requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setStopFlow(Io.StopFlow.newBuilder().setFlowId("f1"))
                    .build(),
            ),
        )
        assertEquals(
            "Artifact requested",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setRequestArtifact(Io.RequestArtifact.newBuilder().setArtifactId("art1"))
                    .build(),
            ),
        )
        assertEquals(
            "Bug report filed",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setBugReportAck(
                        Diagnostics.BugReportAck.newBuilder().setReportId("br1"),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "UI patched",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setUpdateUi(
                        Ui.UpdateUI.newBuilder().setComponentId("c").setDeviceId("d"),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "Registered",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setRegisterAck(Control.RegisterAck.newBuilder().setServerId("srv"))
                    .build(),
            ),
        )
        assertEquals(
            "Command response",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setCommandResult(
                        Control.CommandResult.newBuilder().setRequestId("r1").setNotification("ok"),
                    )
                    .build(),
            ),
        )
        assertEquals(
            "UI updated",
            connectResponseActivityStatus(
                Control.ConnectResponse.newBuilder()
                    .setSetUi(Ui.SetUI.newBuilder().setRoot(Ui.Node.newBuilder().setId("root")))
                    .build(),
            ),
        )
    }
}
