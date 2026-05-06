package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import terminals.capabilities.v1.Capabilities

class StaticAndroidCapabilityProbe(
    private val deviceId: String,
) : AndroidCapabilityProbe {
    override fun current(): AndroidCapabilitySnapshotInput =
        AndroidCapabilitySnapshotInput(
            identity = Capabilities.DeviceIdentity.newBuilder()
                .setDeviceName(deviceId)
                .setDeviceType("tablet")
                .setPlatform("android")
                .build(),
            screenMetrics = AndroidScreenMetrics(
                widthPx = 1280,
                heightPx = 800,
                density = 1.0f,
                orientation = "landscape",
            ),
        )
}
