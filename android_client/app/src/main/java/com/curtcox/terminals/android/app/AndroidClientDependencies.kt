package com.curtcox.terminals.android.app

import android.content.Context
import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.capabilities.ContextAndroidCapabilityProbe
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.AndroidControlSessionController
import com.curtcox.terminals.android.connection.WebSocketAndroidControlClient
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.util.Clock

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
) {
    companion object {
        fun fromContext(context: Context): AndroidClientDependencies =
            AndroidClientDependencies(
                capabilityProbe = ContextAndroidCapabilityProbe(context),
            )
    }
}
