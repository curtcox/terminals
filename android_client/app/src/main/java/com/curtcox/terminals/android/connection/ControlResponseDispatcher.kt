package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.app.AndroidTerminalViewState
import com.curtcox.terminals.android.diagnostics.AndroidBugReportChrome
import terminals.control.v1.Control
import terminals.ui.v1.Ui

class ControlResponseDispatcher {
    fun dispatch(
        state: AndroidTerminalViewState,
        response: Control.ConnectResponse,
    ): AndroidTerminalViewState {
        return when (response.payloadCase) {
            Control.ConnectResponse.PayloadCase.HELLO_ACK -> {
                val ack = response.helloAck
                state.copy(
                    controlServerId = ack.serverId.takeIf { it.isNotBlank() },
                    controlSessionId = ack.sessionId.takeIf { it.isNotBlank() },
                    serverHeartbeatIntervalMs = ack.heartbeatIntervalMs.takeIf { it > 0 },
                )
            }
            Control.ConnectResponse.PayloadCase.CAPABILITY_ACK ->
                state.copy(lastCapabilityAckGeneration = response.capabilityAck.acceptedGeneration)
            Control.ConnectResponse.PayloadCase.REGISTER_ACK -> mergeRegisterAck(state, response)
            Control.ConnectResponse.PayloadCase.SET_UI -> state.copy(serverRoot = response.setUi.root)
            Control.ConnectResponse.PayloadCase.UPDATE_UI -> {
                val root = state.serverRoot ?: return state
                state.copy(serverRoot = replaceNode(root, response.updateUi.componentId, response.updateUi.node))
            }
            Control.ConnectResponse.PayloadCase.TRANSITION_UI -> state.copy(lastTransition = response.transitionUi.transition)
            Control.ConnectResponse.PayloadCase.NOTIFICATION -> state.copy(
                lastNotificationTitle = response.notification.title,
                lastNotificationBody = response.notification.body,
            )
            Control.ConnectResponse.PayloadCase.ERROR -> state.copy(lastError = response.error.message)
            Control.ConnectResponse.PayloadCase.BUG_REPORT_ACK -> state.copy(
                lastBugReportAckDiagnostics = AndroidBugReportChrome.formatDiagnosticsLines(response.bugReportAck),
            )
            else -> state
        }
    }

    private fun mergeRegisterAck(state: AndroidTerminalViewState, response: Control.ConnectResponse): AndroidTerminalViewState {
        val ack = response.registerAck
        val metaMap = ack.metadataMap
        val serverMeta = if (ack.hasServerMetadata()) ack.serverMetadata else null
        val build = serverMeta?.takeIf { it.hasBuild() }?.build
        val sha = build?.sha?.takeIf { it.isNotBlank() }
            ?: metaMap["server_build_sha"]?.takeIf { it.isNotBlank() }
        val date = build?.dateRfc3339?.takeIf { it.isNotBlank() }
            ?: metaMap["server_build_date"]?.takeIf { it.isNotBlank() }
        val assetBase = serverMeta?.photoFrameAssetBaseUrl?.takeIf { it.isNotBlank() }
        return state.copy(
            serverBuildSha = sha ?: state.serverBuildSha,
            serverBuildDate = date ?: state.serverBuildDate,
            registerAckAssetBaseUrl = assetBase ?: state.registerAckAssetBaseUrl,
        )
    }

    private fun replaceNode(root: Ui.Node, componentId: String, replacement: Ui.Node): Ui.Node {
        if (root.id == componentId) return replacement
        if (root.childrenCount == 0) return root

        var changed = false
        val children = root.childrenList.map { child ->
            val next = replaceNode(child, componentId, replacement)
            if (next != child) changed = true
            next
        }
        if (!changed) return root

        return root.toBuilder()
            .clearChildren()
            .addAllChildren(children)
            .build()
    }
}
