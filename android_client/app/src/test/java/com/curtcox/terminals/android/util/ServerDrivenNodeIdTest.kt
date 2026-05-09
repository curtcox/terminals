package com.curtcox.terminals.android.util

import org.junit.Assert.assertEquals
import org.junit.Test
import terminals.ui.v1.Ui

class ServerDrivenNodeIdTest {
    @Test
    fun prefersProtobufIdOverProps() {
        val node = Ui.Node.newBuilder()
            .setId("a")
            .putProps("id", "b")
            .build()
        assertEquals("a", serverDrivenNodeId(node))
    }

    @Test
    fun fallsBackToPropsIdWhenProtobufIdBlank() {
        val node = Ui.Node.newBuilder()
            .putProps("id", "from-props")
            .setText(Ui.TextWidget.newBuilder().setValue("x"))
            .build()
        assertEquals("from-props", serverDrivenNodeId(node))
    }
}
