package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidHardwareCapabilities
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.capabilities.PermissionCapabilityState
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.platform.FireOsDeviceInfo
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.capabilities.v1.Capabilities

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
            capabilitySnapshot = AndroidCapabilitySnapshotInput(
                identity = Capabilities.DeviceIdentity.newBuilder()
                    .setDeviceName("Hallway")
                    .setDeviceType("tablet")
                    .setPlatform("android")
                    .build(),
                screenMetrics = AndroidScreenMetrics(
                    widthPx = 1920,
                    heightPx = 1200,
                    density = 2.0f,
                    orientation = "landscape",
                ),
                permissions = PermissionCapabilityState(
                    microphoneGranted = true,
                    cameraGranted = false,
                    notificationsGranted = true,
                ),
                hardware = AndroidHardwareCapabilities(
                    touchSupported = true,
                    microphone = true,
                    frontCamera = true,
                ),
            ),
        )
        assertTrue(text.contains("client=android-native"))
        assertTrue(text.contains("endpoint=http://host:50051"))
        assertTrue(text.contains("google_services=absent"))
        assertTrue(text.contains("device_manufacturer=Amazon"))
        assertTrue(text.contains("device_model=KFTRWI"))
        assertTrue(text.contains("device_sdk=30"))
        assertTrue(text.contains("device_likely_fire_os=true"))
        assertTrue(text.contains("cap_orientation=landscape"))
        assertTrue(text.contains("cap_display_px=1920x1200"))
        assertTrue(text.contains("cap_density=2.0"))
        assertTrue(text.contains("cap_touch_supported=true"))
        assertTrue(text.contains("cap_microphone_present=true"))
        assertTrue(text.contains("cap_microphone_granted=true"))
        assertTrue(text.contains("cap_camera_present=true"))
        assertTrue(text.contains("cap_camera_granted=false"))
        assertTrue(text.contains("cap_notifications_granted=true"))
    }
}
