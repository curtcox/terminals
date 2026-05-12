package com.curtcox.terminals.android.app

internal fun formatTerminalDiagnostics(request: TerminalDiagnosticsRequest): String = buildString {
    with(request) {
        append(chrome.formatDiagnostics(endpoint, state, networkState, fireOsDeviceInfo, capabilitySnapshot))
        appendLine()
        appendLine("control_connected=${controlStatus?.connected ?: false}")
        appendLine("control_endpoint=${controlStatus?.endpoint?.displayName ?: "none"}")
        appendLine("control_last_error=${controlStatus?.lastError ?: "none"}")
        appendLine("control_last_capability_generation=${controlStatus?.lastCapabilityGeneration ?: 0}")
        appendLine(permissions.toDiagnostics())
        append(mediaSupport.toDiagnostics())
        handshakeSource?.let { src ->
            appendRegistrationDiagnostics(src)
            appendActivityDiagnostics(src)
        }
    }
}

private fun StringBuilder.appendRegistrationDiagnostics(src: AndroidTerminalViewState) {
    src.controlServerId?.takeIf { it.isNotBlank() }?.let {
        appendLine()
        appendLine("hello_server_id=$it")
    }
    src.controlSessionId?.takeIf { it.isNotBlank() }?.let { appendLine("hello_session_id=$it") }
    src.serverHeartbeatIntervalMs?.takeIf { it > 0 }?.let { appendLine("hello_heartbeat_interval_ms=$it") }
    src.serverBuildSha?.takeIf { it.isNotBlank() }?.let { appendLine("server_build_sha=$it") }
    src.serverBuildDate?.takeIf { it.isNotBlank() }?.let { appendLine("server_build_date=$it") }
    src.registerAckMessage?.takeIf { it.isNotBlank() }?.let { appendLine("register_ack_message=$it") }
    src.registerAckServerId?.takeIf { it.isNotBlank() }?.let { appendLine("register_ack_server_id=$it") }
    src.registerAckAssetBaseUrl?.takeIf { it.isNotBlank() }?.let { appendLine("register_ack_asset_base_url=$it") }
    if (src.lastCapabilityAckGeneration > 0L) {
        appendLine("last_capability_ack_generation=${src.lastCapabilityAckGeneration}")
        appendLine("capability_ack_snapshot_applied=${src.lastCapabilityAckSnapshotApplied}")
        src.lastCapabilityInvalidationsSummary?.takeIf { it.isNotBlank() }?.let { summary ->
            appendLine("last_capability_invalidations=$summary")
        }
    }
    src.lastServerHeartbeatUnixMs?.takeIf { it > 0 }?.let { appendLine("last_server_heartbeat_unix_ms=$it") }
    appendLine("outbound_heartbeat_count=${src.outboundHeartbeatCount}")
    appendLine("last_outbound_heartbeat_unix_ms=${src.lastOutboundHeartbeatUnixMs}")
    appendLine("outbound_sensor_send_count=${src.outboundSensorSendCount}")
    appendLine("last_outbound_sensor_unix_ms=${src.lastOutboundSensorUnixMs}")
    appendLine("stream_ready_send_count=${src.streamReadySendCount}")
    appendLine("inbound_connect_response_count=${src.inboundConnectResponseCount}")
}

private fun StringBuilder.appendActivityDiagnostics(src: AndroidTerminalViewState) {
    src.lastCommandResultRequestId?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_command_result_request_id=$it")
    }
    src.lastCommandResultNotification?.takeIf { it.isNotBlank() }?.let {
        appendLine("last_command_result_notification=$it")
    }
    src.lastOpaqueControlIoSummary?.takeIf { it.isNotBlank() }?.let { appendLine("last_opaque_control_io=$it") }
    src.lastControlResponseActivity?.takeIf { it.isNotBlank() }?.let { appendLine("last_control_activity=$it") }
    src.lastTransition?.takeIf { it.isNotBlank() }?.let { appendLine("last_transition=$it") }
    src.lastTransitionDurationMs?.takeIf { it > 0 }?.let { appendLine("last_transition_duration_ms=$it") }
    src.lastError?.takeIf { it.isNotBlank() }?.let { err -> appendLine("last_error=$err") }
    src.lastControlErrorCode?.takeIf { it.isNotBlank() }?.let { code -> appendLine("last_control_error_code=$code") }
    src.lastBugReportAckDiagnostics?.takeIf { it.isNotBlank() }?.let { bug ->
        appendLine()
        append(bug)
    }
    appendLine()
    appendLine("local_keep_awake=${src.localKeepAwakeEnabled}")
    appendLine("local_fullscreen=${src.localFullscreenEnabled}")
    appendLine("local_immersive_sticky=${src.localImmersiveStickyEnabled}")
    appendLine("local_bright_display=${src.localBrightDisplayEnabled}")
    appendLine("privacy_mode=${src.privacyModeEnabled}")
    src.applicationLaunchQueuedIntent?.takeIf { it.isNotBlank() }?.let { queued ->
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
