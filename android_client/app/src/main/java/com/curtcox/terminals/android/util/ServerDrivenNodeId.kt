package com.curtcox.terminals.android.util

import terminals.ui.v1.Ui

/** Matches Flutter `serverDrivenNodeId` (protobuf `id`, else `props["id"]`). */
fun serverDrivenNodeId(node: Ui.Node): String {
    val id = node.id.trim()
    if (id.isNotEmpty()) return id
    return node.propsMap["id"]?.trim().orEmpty()
}
