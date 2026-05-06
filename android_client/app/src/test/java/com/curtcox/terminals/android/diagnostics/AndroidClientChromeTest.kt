package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.connection.EndpointResolution
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidClientChromeTest {
    @Test
    fun formatsGenericDiagnostics() {
        val chrome = AndroidClientChrome(AndroidBuildMetadata("0.1.0", "abc123", "2026-05-06T00:00:00Z"))
        val text = chrome.formatDiagnostics(EndpointResolution("host", 50051), ConnectionState.ReadyToConnect)
        assertTrue(text.contains("client=android-native"))
        assertTrue(text.contains("endpoint=http://host:50051"))
        assertTrue(text.contains("google_services=absent"))
    }
}
