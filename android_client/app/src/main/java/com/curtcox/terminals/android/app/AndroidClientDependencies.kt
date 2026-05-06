package com.curtcox.terminals.android.app

import android.app.Activity
import android.content.Context
import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.capabilities.ContextAndroidCapabilityProbe
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.AndroidControlSessionController
import com.curtcox.terminals.android.connection.WebSocketAndroidControlClient
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.StatusBarAndroidNotificationDelivery
import com.curtcox.terminals.android.platform.WindowAndroidBrightnessController
import com.curtcox.terminals.android.platform.WindowAndroidFullscreenController
import com.curtcox.terminals.android.platform.WindowAndroidKeepAwakeController
import com.curtcox.terminals.android.util.Clock

data class AndroidClientDependencies(
    val buildMetadata: AndroidBuildMetadata = AndroidBuildMetadata.fromBuildConfig(),
    val deviceId: String = "android-native-terminal",
    val capabilityProbe: AndroidCapabilityProbe = StaticAndroidCapabilityProbe(deviceId),
    val keepAwakeController: AndroidKeepAwakeController = AndroidKeepAwakeController {},
    val fullscreenController: AndroidFullscreenController = AndroidFullscreenController {},
    val brightnessController: AndroidBrightnessController = AndroidBrightnessController {},
    val notificationDelivery: AndroidNotificationDelivery = AndroidNotificationDelivery.none(),
    val mediaEngine: AndroidMediaEngine = AndroidMediaEngine.unsupported(),
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
                keepAwakeController = if (context is Activity) {
                    WindowAndroidKeepAwakeController(context.window)
                } else {
                    AndroidKeepAwakeController {}
                },
                fullscreenController = if (context is Activity) {
                    WindowAndroidFullscreenController(context.window)
                } else {
                    AndroidFullscreenController {}
                },
                brightnessController = if (context is Activity) {
                    WindowAndroidBrightnessController(context.window)
                } else {
                    AndroidBrightnessController {}
                },
                notificationDelivery = StatusBarAndroidNotificationDelivery(context.applicationContext),
            )
    }
}
