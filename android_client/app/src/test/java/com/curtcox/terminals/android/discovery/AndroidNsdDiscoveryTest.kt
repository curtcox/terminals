package com.curtcox.terminals.android.discovery

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class AndroidNsdDiscoveryTest {
    @Test
    fun parseTerminalTxtAttributesKeepsGenericCarrierMetadata() {
        val metadata = parseTerminalTxtAttributes(
            mapOf(
                "name" to "HomeServer".encodeToByteArray(),
                "grpc" to "10.0.0.4:50051".encodeToByteArray(),
                "ws" to "ws://10.0.0.4:50054/control".encodeToByteArray(),
                "priority" to "ws,grpc".encodeToByteArray(),
                "" to "ignored".encodeToByteArray(),
                "empty" to ByteArray(0),
            ),
        )

        assertEquals("HomeServer", metadata["name"])
        assertEquals("10.0.0.4:50051", metadata["grpc"])
        assertEquals("ws://10.0.0.4:50054/control", metadata["ws"])
        assertEquals("ws,grpc", metadata["priority"])
        assertFalse(metadata.containsKey(""))
        assertFalse(metadata.containsKey("empty"))
    }
}
