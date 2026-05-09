package com.curtcox.terminals.android.discovery

import android.net.nsd.NsdManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidNsdDiscoveryTest {
    @Test
    fun formatNsdFailureDetailMapsKnownCodesWithHints() {
        val internal = formatNsdFailureDetail("start_discovery", NsdManager.FAILURE_INTERNAL_ERROR)
        assertTrue(internal.contains("internal_error"))
        assertTrue(internal.contains("code=0"))
        assertTrue(internal.contains("manual endpoint"))

        val active = formatNsdFailureDetail("resolve", NsdManager.FAILURE_ALREADY_ACTIVE)
        assertTrue(active.contains("already_active"))
        assertTrue(active.contains("code=3"))

        val limit = formatNsdFailureDetail("stop_discovery", NsdManager.FAILURE_MAX_LIMIT)
        assertTrue(limit.contains("max_limit"))
        assertTrue(limit.contains("code=4"))
    }

    @Test
    fun formatNsdFailureDetailUnknownCodeStillSuggestsManualEndpoint() {
        val unknown = formatNsdFailureDetail("start_discovery", 99999)
        assertTrue(unknown.contains("unknown"))
        assertTrue(unknown.contains("code=99999"))
        assertTrue(unknown.contains("manual endpoint"))
    }

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
