package com.curtcox.terminals.android.util

import terminals.ui.v1.Ui

/** Matches Flutter `server_driven_node_key.dart` (non-empty protobuf `id`, else `props["id"]`). */
fun serverDrivenNodeId(node: Ui.Node): String {
    val id = node.id
    if (id.isNotEmpty()) return id
    return node.propsMap["id"].orEmpty()
}
