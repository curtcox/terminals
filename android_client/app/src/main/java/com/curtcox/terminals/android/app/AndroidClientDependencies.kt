package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.AndroidControlSessionController
import com.curtcox.terminals.android.connection.WebSocketAndroidControlClient
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.util.Clock
import terminals.capabilities.v1.Capabilities

data class AndroidClientDependencies(
    val buildMetadata: AndroidBuildMetadata = AndroidBuildMetadata.fromBuildConfig(),
    val deviceId: String = "android-native-terminal",
    val capabilityProbe: AndroidCapabilityProbe = StaticAndroidCapabilityProbe(deviceId),
    val sessionFactory: (AndroidControlResponseSink) -> AndroidControlSession = { sink ->
        AndroidControlSessionController(
            deviceId = deviceId,
            clientVersion = buildMetadata.versionName,
            client = WebSocketAndroidControlClient(deviceId = deviceId, responseSink = sink),
            capabilities = AndroidCapabilitySession(deviceId, capabilityProbe),
            clock = Clock { System.currentTimeMillis() },
        )
    },
)

private class StaticAndroidCapabilityProbe(
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
                density = 1f,
                orientation = "landscape",
            ),
        )
}
