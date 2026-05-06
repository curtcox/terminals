package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ManualEndpointParserTest {
    private val parser = ManualEndpointParser()

    @Test
    fun parsesHostPort() {
        val endpoint = parser.parse("192.168.1.20:50051")
        assertEquals("192.168.1.20", endpoint?.host)
        assertEquals(50051, endpoint?.port)
        assertEquals(false, endpoint?.secure)
    }

    @Test
    fun parsesUrlDefaults() {
        val endpoint = parser.parse("https://terminal.local/control")
        assertEquals("terminal.local", endpoint?.host)
        assertEquals(443, endpoint?.port)
        assertEquals(true, endpoint?.secure)
        assertEquals("/control", endpoint?.path)
    }

    @Test
    fun rejectsInvalidInput() {
        assertNull(parser.parse("not a host"))
        assertNull(parser.parse("ftp://terminal.local:50051"))
    }
}
