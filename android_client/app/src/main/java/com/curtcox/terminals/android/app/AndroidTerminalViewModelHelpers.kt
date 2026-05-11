package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import terminals.control.v1.Control

internal fun Control.ConnectResponse.requiresCapabilityRebaseline(): Boolean {
    if (!hasError()) return false
    if (error.code != Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) return false
    return error.message.contains("stale", ignoreCase = true) &&
        error.message.contains("generation", ignoreCase = true)
}

internal fun withoutHandshake(state: AndroidTerminalViewState): AndroidTerminalViewState =
    state.copy(
        controlServerId = null,
        controlSessionId = null,
        serverHeartbeatIntervalMs = null,
        serverBuildSha = null,
        serverBuildDate = null,
        registerAckMessage = null,
        registerAckServerId = null,
        registerAckAssetBaseUrl = null,
        lastCapabilityAckGeneration = 0L,
        lastCapabilityAckSnapshotApplied = false,
        lastCapabilityInvalidationsSummary = null,
        lastServerHeartbeatUnixMs = null,
        lastCommandResultRequestId = null,
        lastCommandResultNotification = null,
        lastOpaqueControlIoSummary = null,
        lastTransition = null,
        lastTransitionDurationMs = null,
        lastBugReportAckDiagnostics = null,
        lastControlErrorCode = null,
        lastControlResponseActivity = null,
        lastLiveMediaLine = null,
        reconnectAttempt = 0,
        outboundHeartbeatCount = 0,
        lastOutboundHeartbeatUnixMs = 0L,
        outboundSensorSendCount = 0,
        lastOutboundSensorUnixMs = 0L,
        streamReadySendCount = 0,
        inboundConnectResponseCount = 0,
        availableApplicationIntents = listOf("terminal"),
        selectedApplicationIntent = "terminal",
        applicationLaunchQueuedIntent = null,
    )

internal fun AudioPlaybackResult.toStatus(requestId: String): Pair<String, String> =
    when (this) {
        is AudioPlaybackResult.Played -> (this.requestId.ifBlank { requestId }) to "played"
        is AudioPlaybackResult.Unsupported -> requestId to "unsupported-audio:$reason"
    }

internal fun MediaDisplayResult.toStatus(requestId: String): Pair<String, String> =
    when (this) {
        is MediaDisplayResult.Shown -> (this.requestId.ifBlank { requestId }) to "shown"
        is MediaDisplayResult.Unsupported -> requestId to "unsupported-media:$reason"
    }
