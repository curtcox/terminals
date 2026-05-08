package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
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
        capabilitySnapshot: AndroidCapabilitySnapshotInput? = null,
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
        appendLine("cap_orientation=${capabilitySnapshot?.screenMetrics?.orientation ?: "unknown"}")
        appendLine(
            "cap_display_px=" +
                "${capabilitySnapshot?.screenMetrics?.widthPx ?: "unknown"}x" +
                "${capabilitySnapshot?.screenMetrics?.heightPx ?: "unknown"}",
        )
        appendLine("cap_density=${capabilitySnapshot?.screenMetrics?.density ?: "unknown"}")
        appendLine("cap_touch_supported=${capabilitySnapshot?.hardware?.touchSupported ?: "unknown"}")
        appendLine("cap_microphone_present=${capabilitySnapshot?.hardware?.microphone ?: "unknown"}")
        appendLine("cap_microphone_granted=${capabilitySnapshot?.permissions?.microphoneGranted ?: "unknown"}")
        val cameraPresent = capabilitySnapshot?.let {
            it.hardware.frontCamera || it.hardware.backCamera
        }
        appendLine("cap_camera_present=${cameraPresent ?: "unknown"}")
        appendLine("cap_camera_granted=${capabilitySnapshot?.permissions?.cameraGranted ?: "unknown"}")
        appendLine("cap_notifications_granted=${capabilitySnapshot?.permissions?.notificationsGranted ?: "unknown"}")
        appendLine("fire_os_target=minSdk25")
        appendLine("google_services=absent")
    }
}
