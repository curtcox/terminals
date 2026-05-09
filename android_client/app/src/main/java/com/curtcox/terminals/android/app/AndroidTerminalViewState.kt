package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.discovery.DiscoveredServer
import terminals.ui.v1.Ui

data class AndroidTerminalViewState(
    val endpointText: String = "",
    val connectionState: ConnectionState = ConnectionState.Disconnected,
    val lastError: String? = null,
    val diagnosticsText: String = "",
    val serverRoot: Ui.Node? = null,
    val lastNotificationTitle: String? = null,
    val lastNotificationBody: String? = null,
    val lastMediaRequestId: String? = null,
    val lastMediaStatus: String? = null,
    val lastDiagnosticsCopyStatus: String? = null,
    val lastTransition: String? = null,
    val lastBugReportAckDiagnostics: String? = null,
    val controlServerId: String? = null,
    val controlSessionId: String? = null,
    val serverHeartbeatIntervalMs: Long? = null,
    val serverBuildSha: String? = null,
    val serverBuildDate: String? = null,
    val registerAckAssetBaseUrl: String? = null,
    val lastCapabilityAckGeneration: Long = 0L,
    val lastServerHeartbeatUnixMs: Long? = null,
    val lastCommandResultRequestId: String? = null,
    val lastCommandResultNotification: String? = null,
    val localKeepAwakeEnabled: Boolean = false,
    val localFullscreenEnabled: Boolean = false,
    val localBrightDisplayEnabled: Boolean = false,
    val permissionEducation: PermissionEducationState = PermissionEducationState(),
    val mediaSupport: MediaSupportState = MediaSupportState(),
    val discoveryState: DiscoveryState = DiscoveryState(),
)

enum class ConnectionState {
    Disconnected,
    InvalidEndpoint,
    ReadyToConnect,
    Connecting,
    Connected,
}

data class PermissionEducationState(
    val notificationsGranted: Boolean = false,
    val microphonePresent: Boolean = false,
    val microphoneAvailable: Boolean = false,
    val cameraPresent: Boolean = false,
    val cameraAvailable: Boolean = false,
) {
    val messages: List<String>
        get() = buildList {
            if (!notificationsGranted) {
                add("Notifications are disabled; server notifications will stay in terminal diagnostics.")
            }
            if (microphonePresent && !microphoneAvailable) {
                add("Microphone capture is unavailable until hardware and permission are both present.")
            }
            if (cameraPresent && !cameraAvailable) {
                add("Camera capture is unavailable until hardware and permission are both present.")
            }
        }
}

data class MediaSupportState(
    val microphonePermissionGranted: Boolean = false,
    val cameraPermissionGranted: Boolean = false,
    val webRtcSupported: Boolean = false,
    val webRtcReason: String = "unknown",
) {
    fun toDiagnostics(): String = buildString {
        appendLine("media_microphone_permission=$microphonePermissionGranted")
        appendLine("media_camera_permission=$cameraPermissionGranted")
        appendLine("media_webrtc_supported=$webRtcSupported")
        append("media_webrtc_reason=$webRtcReason")
    }
}

data class DiscoveryState(
    val scanning: Boolean = false,
    val servers: List<DiscoveredServer> = emptyList(),
    val lastError: String? = null,
) {
    val statusText: String
        get() = when {
            scanning -> "Scanning for servers"
            lastError != null -> "Discovery unavailable: $lastError"
            servers.isEmpty() -> "No discovered servers"
            else -> "${servers.size} discovered server${if (servers.size == 1) "" else "s"}"
        }
}
