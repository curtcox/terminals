package com.curtcox.terminals.android.app

import android.app.Activity
import android.content.Context
import android.net.nsd.NsdManager
import android.os.Build
import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.capabilities.ContextAndroidCapabilityProbe
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.AndroidControlSessionController
import com.curtcox.terminals.android.connection.CarrierSelectingAndroidControlClient
import com.curtcox.terminals.android.connection.ReconnectPolicy
import com.curtcox.terminals.android.connection.TransportResumeTokenStore
import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata
import com.curtcox.terminals.android.diagnostics.ContextDiagnosticClipboard
import com.curtcox.terminals.android.diagnostics.DiagnosticClipboard
import com.curtcox.terminals.android.discovery.AndroidNsdDiscovery
import com.curtcox.terminals.android.discovery.NsdAndroidDiscovery
import com.curtcox.terminals.android.media.AndroidLiveMediaSession
import com.curtcox.terminals.android.media.AndroidMediaEngine
import com.curtcox.terminals.android.media.AndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.AndroidWebRtcAdapter
import com.curtcox.terminals.android.media.ContextAndroidMediaPermissionProbe
import com.curtcox.terminals.android.media.ContextAndroidAudioPlayback
import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidNotificationDelivery
import com.curtcox.terminals.android.platform.AndroidPermissionRequester
import com.curtcox.terminals.android.platform.AndroidTerminalSpeech
import com.curtcox.terminals.android.platform.ContextAndroidTerminalSpeech
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import com.curtcox.terminals.android.platform.ContextAndroidNetworkMonitor
import com.curtcox.terminals.android.platform.ContextAndroidNetworkStateProvider
import com.curtcox.terminals.android.platform.AndroidNetworkMonitor
import com.curtcox.terminals.android.platform.FireOsDeviceInfo
import com.curtcox.terminals.android.platform.FireOsDeviceInfoProvider
import com.curtcox.terminals.android.platform.SharedPreferencesAndroidTerminalSettings
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
    val fullscreenController: AndroidFullscreenController = AndroidFullscreenController { _, _ -> },
    val brightnessController: AndroidBrightnessController = AndroidBrightnessController {},
    val networkStateProvider: AndroidNetworkStateProvider = AndroidNetworkStateProvider.unknown(),
    val networkMonitor: AndroidNetworkMonitor = AndroidNetworkMonitor.none(),
    val notificationDelivery: AndroidNotificationDelivery = AndroidNotificationDelivery.none(),
    val speechDelivery: AndroidTerminalSpeech = AndroidTerminalSpeech.none(),
    val diagnosticClipboard: DiagnosticClipboard = DiagnosticClipboard.none(),
    val discovery: AndroidNsdDiscovery = AndroidNsdDiscovery.unavailable(),
    val mediaEngine: AndroidMediaEngine = AndroidMediaEngine.unsupported(),
    val mediaPermissionProbe: AndroidMediaPermissionProbe = AndroidMediaPermissionProbe.unavailable(),
    val webRtcAdapter: AndroidWebRtcAdapter = AndroidWebRtcAdapter.disabled(),
    val permissionRequester: AndroidPermissionRequester = AndroidPermissionRequester.none(),
    val terminalSettings: AndroidTerminalSettings = AndroidTerminalSettings.inMemory(),
    val fireOsDeviceInfoProvider: FireOsDeviceInfoProvider = FireOsDeviceInfoProvider.unknown(),
    val heartbeatIntervalMillis: Long = 30_000,
    /** Matches Flutter [TerminalClientApp] default `sensorTelemetryInterval` (15 seconds). */
    val sensorTelemetryIntervalMillis: Long = 15_000,
    /**
     * Matches Flutter `terminal_client_shell` `_capabilityMonitorInterval` (2 seconds) in production
     * ([fromContext] supplies this). When > 0 and the app is foregrounded while connected,
     * [AndroidTerminalViewModel] probes capabilities on this interval and sends `runtime_monitor_poll`
     * deltas when changed. Default 0 keeps JVM tests deterministic unless they opt in.
     */
    val capabilityMonitorIntervalMillis: Long = 0,
    val reconnectPolicy: ReconnectPolicy = ReconnectPolicy(),
    val maxReconnectAttempts: Int = 5,
    val discoveryRestartMinIntervalMillis: Long = 1_500,
    val networkCapabilityRefreshMinIntervalMillis: Long = 1_500,
    val networkReconnectRestoreMinIntervalMillis: Long = 5_000,
    /**
     * When false, [AndroidTerminalViewModel.requestNotificationPermission] is a no-op with education refresh only
     * (Android 12 and below). Production uses API 33+; JVM unit tests override when exercising POST_NOTIFICATIONS.
     */
    val runtimeNotificationPermissionPromptSupported: Boolean =
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU,
    val nowMillis: () -> Long = { System.currentTimeMillis() },
    /** WebSocket transport hello resume token; shared across sessions like Flutter [ControlClientTransportHint.resumeToken]. */
    val websocketResumeTokenStore: TransportResumeTokenStore = TransportResumeTokenStore(),
    val sessionFactory: (AndroidControlResponseSink) -> AndroidControlSession = { sink ->
        AndroidControlSessionController(
            deviceId = deviceId,
            clientVersion = buildMetadata.versionName,
            client = CarrierSelectingAndroidControlClient(
                deviceId = deviceId,
                websocketResumeTokenStore = websocketResumeTokenStore,
                responseSink = sink,
            ),
            capabilities = AndroidCapabilitySession(deviceId, capabilityProbe),
            clock = Clock { System.currentTimeMillis() },
        )
    },
) {
    companion object {
        fun fromContext(context: Context): AndroidClientDependencies {
            val webRtcAdapter = AndroidWebRtcAdapter.disabled()
            return AndroidClientDependencies(
                capabilityMonitorIntervalMillis = 2_000,
                capabilityProbe = ContextAndroidCapabilityProbe(context),
                keepAwakeController = if (context is Activity) {
                    WindowAndroidKeepAwakeController(context.window)
                } else {
                    AndroidKeepAwakeController {}
                },
                fullscreenController = if (context is Activity) {
                    WindowAndroidFullscreenController(context.window)
                } else {
                    AndroidFullscreenController { _, _ -> }
                },
                brightnessController = if (context is Activity) {
                    WindowAndroidBrightnessController(context.window)
                } else {
                    AndroidBrightnessController {}
                },
                networkStateProvider = ContextAndroidNetworkStateProvider(context),
                networkMonitor = ContextAndroidNetworkMonitor(context),
                notificationDelivery = StatusBarAndroidNotificationDelivery(context.applicationContext),
                speechDelivery = ContextAndroidTerminalSpeech(context.applicationContext),
                diagnosticClipboard = ContextDiagnosticClipboard(context.applicationContext),
                discovery = (context.applicationContext.getSystemService(Context.NSD_SERVICE) as? NsdManager)?.let {
                    NsdAndroidDiscovery(it, Clock { System.currentTimeMillis() })
                } ?: AndroidNsdDiscovery.unavailable(),
                mediaEngine = AndroidMediaEngine(
                    audioPlayback = ContextAndroidAudioPlayback(context.applicationContext),
                    liveMedia = AndroidLiveMediaSession.fromAdapter(webRtcAdapter),
                ),
                mediaPermissionProbe = ContextAndroidMediaPermissionProbe(context.applicationContext),
                webRtcAdapter = webRtcAdapter,
                terminalSettings = SharedPreferencesAndroidTerminalSettings(context),
                fireOsDeviceInfoProvider = FireOsDeviceInfoProvider {
                    FireOsDeviceInfo(
                        manufacturer = Build.MANUFACTURER,
                        model = Build.MODEL,
                        sdkInt = Build.VERSION.SDK_INT,
                    )
                },
            )
        }
    }
}
