package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.app.AndroidTerminalViewState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics.BugReportAck
import terminals.diagnostics.v1.Diagnostics.BugReportStatus
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
