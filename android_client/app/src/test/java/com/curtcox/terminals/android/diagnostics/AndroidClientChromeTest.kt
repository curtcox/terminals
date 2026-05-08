package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.platform.FireOsDeviceInfo
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidClientChromeTest {
    @Test
    fun formatsGenericDiagnostics() {
        val chrome = AndroidClientChrome(AndroidBuildMetadata("0.1.0", "abc123", "2026-05-06T00:00:00Z"))
        val text = chrome.formatDiagnostics(
            EndpointResolution("host", 50051),
            ConnectionState.ReadyToConnect,
            fireOsDeviceInfo = FireOsDeviceInfo(
                manufacturer = "Amazon",
                model = "KFTRWI",
                sdkInt = 30,
            ),
        )
        assertTrue(text.contains("client=android-native"))
        assertTrue(text.contains("endpoint=http://host:50051"))
        assertTrue(text.contains("google_services=absent"))
        assertTrue(text.contains("device_manufacturer=Amazon"))
        assertTrue(text.contains("device_model=KFTRWI"))
        assertTrue(text.contains("device_sdk=30"))
        assertTrue(text.contains("device_likely_fire_os=true"))
    }
}
