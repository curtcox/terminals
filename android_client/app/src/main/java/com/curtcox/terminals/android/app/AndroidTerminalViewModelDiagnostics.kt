package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.connection.ControlSessionStatus
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.platform.AndroidNetworkState
import com.curtcox.terminals.android.platform.FireOsDeviceInfo

internal fun formatTerminalDiagnostics(
    chrome: AndroidClientChrome,
    endpoint: EndpointResolution?,
    state: ConnectionState,
    networkState: AndroidNetworkState?,
    fireOsDeviceInfo: FireOsDeviceInfo?,
    capabilitySnapshot: AndroidCapabilitySnapshotInput?,
    controlStatus: ControlSessionStatus?,
    permissions: PermissionEducationState,
    mediaSupport: MediaSupportState,
    handshakeSource: AndroidTerminalViewState? = null,
): String = buildString {
    append(
        chrome.formatDiagnostics(
            endpoint = endpoint,
            state = state,
            networkState = networkState,
            fireOsDeviceInfo = fireOsDeviceInfo,
            capabilitySnapshot = capabilitySnapshot,
        ),
    )
    appendLine()
    appendLine("control_connected=${controlStatus?.connected ?: false}")
    appendLine("control_endpoint=${controlStatus?.endpoint?.displayName ?: "none"}")
    appendLine("control_last_error=${controlStatus?.lastError ?: "none"}")
    appendLine("control_last_capability_generation=${controlStatus?.lastCapabilityGeneration ?: 0}")
    appendLine(permissions.toDiagnostics())
    append(mediaSupport.toDiagnostics())
    handshakeSource?.controlServerId?.takeIf { it.isNotBlank() }?.let {
        appendLine()
        appendLine("hello_server_id=$it")
    }
    handshakeSource?.controlSessionId?.takeIf { it.isNotBlank() }?.let {
        appendLine("hello_session_id=$it")
    }
    handshakeSource?.serverHeartbeatIntervalMs?.takeIf { it > 0 }?.let {
        appendLine("hello_heartbeat_interval_ms=$it")
    }
    handshakeSource?.serverBuildSha?.takeIf { it.isNotBlank() }?.let {
        appendLine("server_build_sha=$it")
    }
    handshakeSource?.serverBuildDate?.takeIf { it.isNotBlank() }?.let {
        appendLine("server_build_date=$it")
    }
    handshakeSource?.registerAckMessage?.takeIf { it.isNotBlank() }?.let {
        appendLine("register_ack_message=$it")
    }
    handshakeSource?.registerAckServerId?.takeIf { it.isNotBlank() }?.let {
        appendLine("register_ack_server_id=$it")
    }
    handshakeSource?.registerAckAssetBaseUrl?.takeIf { it.isNotBlank() }?.let {
        appendLine("register_ack_asset_base_url=$it")
    }
    handshakeSource?.takeIf { it.lastCapabilityAckGeneration > 0L }?.let {
        appendLine("last_capability_ack_generation=${it.lastCapabilityAckGeneration}")
        appendLine("capability_ack_snapshot_applied=${it.lastCapabilityAckSnapshotApplied}")
        it.lastCapabilityInvalidationsSummary?.takeIf { summary -> summary.isNotBlank() }?.let { summary ->
            appendLine("last_capability_invalidations=$summary")
        }
    }
    handshakeSource?.lastServerHeartbeatUnixMs?.takeIf { it > 0 }?.let {
        appendLine("last_server_heartbeat_unix_ms=$it")
    }
    handshakeSource?.let { src ->
        appendLine("outbound_heartbeat_count=${src.outboundHeartbeatCount}")
        appendLine("last_outbound_heartbeat_unix_ms=${src.lastOutboundHeartbeatUnixMs}")
        appendLine("outbound_sensor_send_count=${src.outboundSensorSendCount}")
        appendLine("last_outbound_sensor_unix_ms=${src.lastOutboundSensorUnixMs}")
        appendLine("stream_ready_send_count=${src.streamReadySendCount}")
        appendLine("inbound_connect_response_count=${src.inboundConnectResponseCount}")
    }
    handshakeSource?.lastCommandResultRequestId?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_command_result_request_id=$it")
    }
    handshakeSource?.lastCommandResultNotification?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_command_result_notification=$it")
    }
    handshakeSource?.lastOpaqueControlIoSummary?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_opaque_control_io=$it")
    }
    handshakeSource?.lastControlResponseActivity?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_control_activity=$it")
    }
    handshakeSource?.lastTransition?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_transition=$it")
    }
    handshakeSource?.lastTransitionDurationMs?.takeIf { it > 0 }?.let {
        appendLine("last_transition_duration_ms=$it")
    }
    handshakeSource?.lastError?.takeIf { it.isNotBlank() }?.let { err ->
        appendLine("last_error=$err")
    }
    handshakeSource?.lastControlErrorCode?.takeIf { it.isNotBlank() }?.let { code ->
        appendLine("last_control_error_code=$code")
    }
    handshakeSource?.lastBugReportAckDiagnostics?.takeIf { it.isNotBlank() }?.let { bug ->
        appendLine()
        append(bug)
    }
    handshakeSource?.let { src ->
        appendLine()
        appendLine("local_keep_awake=${src.localKeepAwakeEnabled}")
        appendLine("local_fullscreen=${src.localFullscreenEnabled}")
        appendLine("local_immersive_sticky=${src.localImmersiveStickyEnabled}")
        appendLine("local_bright_display=${src.localBrightDisplayEnabled}")
        appendLine("privacy_mode=${src.privacyModeEnabled}")
    }
    handshakeSource?.applicationLaunchQueuedIntent?.takeIf { it.isNotBlank() }?.let { queued ->
        appendLine("application_launch_queued_until_register_ack=$queued")
    }
}

internal fun PermissionEducationState.toDiagnostics(): String = buildString {
    appendLine("permission_notifications=$notificationsGranted")
    appendLine("permission_microphone_present=$microphonePresent")
    appendLine("permission_microphone_available=$microphoneAvailable")
    appendLine("permission_camera_present=$cameraPresent")
    append("permission_camera_available=$cameraAvailable")
}
