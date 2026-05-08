package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.platform.AndroidNetworkState
import com.curtcox.terminals.android.platform.FireOsDeviceInfo

class AndroidClientChrome(
    private val buildMetadata: AndroidBuildMetadata,
) {
    fun formatDiagnostics(
        endpoint: EndpointResolution?,
        state: ConnectionState,
        networkState: AndroidNetworkState? = null,
        fireOsDeviceInfo: FireOsDeviceInfo? = null,
    ): String = buildString {
        appendLine("client=android-native")
        appendLine("version=${buildMetadata.versionName}")
        appendLine("build_sha=${buildMetadata.buildSha}")
        appendLine("build_date=${buildMetadata.buildDate}")
        appendLine("state=$state")
        appendLine("endpoint=${endpoint?.displayName ?: "none"}")
        appendLine("network_connected=${networkState?.connected ?: "unknown"}")
        appendLine("network_metered=${networkState?.metered ?: "unknown"}")
        appendLine("device_manufacturer=${fireOsDeviceInfo?.manufacturer ?: "unknown"}")
        appendLine("device_model=${fireOsDeviceInfo?.model ?: "unknown"}")
        appendLine("device_sdk=${fireOsDeviceInfo?.sdkInt ?: "unknown"}")
        appendLine("device_likely_fire_os=${fireOsDeviceInfo?.isLikelyFireOs ?: "unknown"}")
        appendLine("fire_os_target=minSdk25")
        appendLine("google_services=absent")
    }
}
