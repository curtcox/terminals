package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Test

class TransportResumeTokenStoreTest {
    @Test
    fun captureFromAckTrimsAndStoresNonBlankTokens() {
        val store = TransportResumeTokenStore()
        store.captureFromAck("  abc  ")
        assertEquals("abc", store.current())
    }

    @Test
    fun captureFromAckIgnoresBlankTokens() {
        val store = TransportResumeTokenStore()
        store.captureFromAck("first")
        store.captureFromAck("   ")
        store.captureFromAck("")
        assertEquals("first", store.current())
    }

    @Test
    fun clearRemovesToken() {
        val store = TransportResumeTokenStore()
        store.captureFromAck("tok")
        store.clear()
        assertEquals("", store.current())
    }
}
