package com.curtcox.terminals.android.connection

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import terminals.control.v1.Control

class CarrierSelectingAndroidControlClientTest {
    @Test
    fun sendWithoutConnectFails() = runTest {
        val client =
            CarrierSelectingAndroidControlClient(
                deviceId = "unit-test-device",
                websocketResumeTokenStore = TransportResumeTokenStore(),
                responseSink = null,
            )
        try {
            client.send(Control.ConnectRequest.getDefaultInstance())
            fail("expected error when transport was never connected")
        } catch (e: IllegalStateException) {
            assertTrue(e.message!!.contains("not connected"))
        }
    }
}
