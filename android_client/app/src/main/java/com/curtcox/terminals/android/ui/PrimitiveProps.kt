package com.curtcox.terminals.android.ui

import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import com.curtcox.terminals.android.util.serverDrivenNodeId
import terminals.ui.v1.Ui

data class PrimitiveProps(
    val componentId: String = "",
    val testTag: String = "",
) {
    fun modifier(): Modifier = if (testTag.isBlank()) Modifier else Modifier.testTag(testTag)

    companion object {
        fun from(node: Ui.Node): PrimitiveProps {
            val tag = node.propsMap["testTag"] ?: node.propsMap["test_tag"] ?: NodeKey.testTag(node)
            return PrimitiveProps(componentId = serverDrivenNodeId(node), testTag = tag)
        }
    }
}
