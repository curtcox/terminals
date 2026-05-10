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
    /** Last live-stream / WebRTC seam diagnostic line from the shell (e.g. start_stream id + reason). */
    val lastLiveMediaLine: String? = null,
    val lastDiagnosticsCopyStatus: String? = null,
    /** User-visible status for the last shell bug-report submit attempt (queued/sent/failed). */
    val lastBugReportSubmitStatus: String? = null,
    val lastTransition: String? = null,
    /** Non-zero duration from the last server TransitionUI, for copyable diagnostics. */
    val lastTransitionDurationMs: Long? = null,
    val lastBugReportAckDiagnostics: String? = null,
    val controlServerId: String? = null,
    val controlSessionId: String? = null,
    val serverHeartbeatIntervalMs: Long? = null,
    val serverBuildSha: String? = null,
    val serverBuildDate: String? = null,
    /** Non-empty `RegisterAck.message` from the server, for generic copyable diagnostics. */
    val registerAckMessage: String? = null,
    /** Non-empty `RegisterAck.server_id` from the server (may differ from HelloAck server id). */
    val registerAckServerId: String? = null,
    val registerAckAssetBaseUrl: String? = null,
    val lastCapabilityAckGeneration: Long = 0L,
    /** From the last server `CapabilityAck.snapshot_applied`. */
    val lastCapabilityAckSnapshotApplied: Boolean = false,
    /** Compact summary of `CapabilityAck.invalidations` for copyable diagnostics. */
    val lastCapabilityInvalidationsSummary: String? = null,
    val lastServerHeartbeatUnixMs: Long? = null,
    val lastCommandResultRequestId: String? = null,
    val lastCommandResultNotification: String? = null,
    /** From the last server `ControlError.code` (generic protocol debugging). */
    val lastControlErrorCode: String? = null,
    /** Opaque summary for server IO/control messages not yet executed natively (streams, flows, WebRTC, bundles). */
    val lastOpaqueControlIoSummary: String? = null,
    /** Short label for the last inbound control message (aligned with Flutter `statusFromConnectResponse`). */
    val lastControlResponseActivity: String? = null,
    /**
     * Current reconnect attempt counter. Incremented in [AndroidTerminalViewModel.startReconnect]
     * for every reconnect iteration; reset to `0` on user disconnect and on successful (re)connect.
     * Surfaced in bug-report `ConnectionHealth.reconnect_attempt` for Flutter parity.
     */
    val reconnectAttempt: Int = 0,
    /** Outbound control-stream telemetry (Flutter debug panel parity for smoke tests / copyable diagnostics). */
    val outboundHeartbeatCount: Int = 0,
    val lastOutboundHeartbeatUnixMs: Long = 0L,
    val outboundSensorSendCount: Int = 0,
    val lastOutboundSensorUnixMs: Long = 0L,
    val streamReadySendCount: Int = 0,
    val localKeepAwakeEnabled: Boolean = false,
    val localFullscreenEnabled: Boolean = false,
    /** Local kiosk preference: transient vs sticky immersive behavior when fullscreen is applied. */
    val localImmersiveStickyEnabled: Boolean = true,
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
