package com.curtcox.terminals.android.ui

import terminals.ui.v1.Ui

object NodeKey {
    fun testTag(node: Ui.Node): String = "terminal-node-${node.id.ifBlank { node.widgetCase.name.lowercase() }}"
}
