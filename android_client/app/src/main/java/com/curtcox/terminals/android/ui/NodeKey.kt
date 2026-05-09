package com.curtcox.terminals.android.ui

import com.curtcox.terminals.android.util.serverDrivenNodeId
import terminals.ui.v1.Ui

object NodeKey {
    fun testTag(node: Ui.Node): String {
        val id = serverDrivenNodeId(node)
        val suffix = id.ifBlank { node.widgetCase.name.lowercase() }
        return "terminal-node-$suffix"
    }
}
