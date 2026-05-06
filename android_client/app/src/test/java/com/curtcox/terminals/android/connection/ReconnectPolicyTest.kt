package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Test

class ReconnectPolicyTest {
    @Test
    fun backsOffWithCap() {
        val policy = ReconnectPolicy(initialDelayMillis = 100, maxDelayMillis = 250)
        assertEquals(0, policy.delayForAttempt(0))
        assertEquals(100, policy.delayForAttempt(1))
        assertEquals(200, policy.delayForAttempt(2))
        assertEquals(250, policy.delayForAttempt(3))
        assertEquals(250, policy.delayForAttempt(6))
    }
}
