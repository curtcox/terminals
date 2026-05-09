package com.curtcox.terminals.android.ui

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ColorParsingTest {
    @Test
    fun parseHexColorAcceptsRgbWithoutHash() {
        assertEquals(Color(0xFF112233), parseHexColor("112233"))
    }

    @Test
    fun parseHexColorAcceptsArgbWithHash() {
        assertEquals(Color(0xAA112233), parseHexColor("#AA112233"))
    }

    @Test
    fun parseHexColorRejectsInvalidValues() {
        assertNull(parseHexColor("oops"))
        assertNull(parseHexColor("#12345"))
    }

    @Test
    fun parseColorOrUnspecifiedFallsBackWhenInvalid() {
        assertEquals(Color.Unspecified, parseColorOrUnspecified("invalid"))
    }
}
