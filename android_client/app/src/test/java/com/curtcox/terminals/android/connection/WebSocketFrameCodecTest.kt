package com.curtcox.terminals.android.connection

import java.io.BufferedInputStream
import java.io.BufferedOutputStream
import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Test

class WebSocketFrameCodecTest {
    @Test
    fun writesAndReadsMaskedBinaryFrame() {
        val payload = "hello terminal".toByteArray(Charsets.UTF_8)
        val raw = ByteArrayOutputStream()

        writeFrame(BufferedOutputStream(raw), payload, masked = true)

        val frame = readFrame(BufferedInputStream(ByteArrayInputStream(raw.toByteArray())))

        assertEquals(2, frame.opcode)
        assertArrayEquals(payload, frame.payload)
    }

    @Test
    fun writesAndReadsExtendedLengthFrame() {
        val payload = ByteArray(300) { index -> index.toByte() }
        val raw = ByteArrayOutputStream()

        writeFrame(BufferedOutputStream(raw), payload, masked = false)

        val frame = readFrame(BufferedInputStream(ByteArrayInputStream(raw.toByteArray())))

        assertEquals(2, frame.opcode)
        assertArrayEquals(payload, frame.payload)
    }
}
