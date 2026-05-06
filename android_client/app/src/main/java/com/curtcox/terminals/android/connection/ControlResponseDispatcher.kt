package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.app.AndroidTerminalViewState
import terminals.control.v1.Control
import terminals.ui.v1.Ui

class ControlResponseDispatcher {
    fun dispatch(
        state: AndroidTerminalViewState,
        response: Control.ConnectResponse,
    ): AndroidTerminalViewState {
        return when (response.payloadCase) {
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
            else -> state
        }
    }

    private fun replaceNode(root: Ui.Node, componentId: String, replacement: Ui.Node): Ui.Node {
        if (root.id == componentId) return replacement
        if (root.childrenCount == 0) return root

        var changed = false
        val children = root.childrenList.map { child ->
            val next = replaceNode(child, componentId, replacement)
            if (next !== child && next != child) changed = true
            next
        }
        if (!changed) return root

        return root.toBuilder()
            .clearChildren()
            .addAllChildren(children)
            .build()
    }
}
