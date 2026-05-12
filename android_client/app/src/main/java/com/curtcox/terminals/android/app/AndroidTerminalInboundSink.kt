package com.curtcox.terminals.android.app

import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.CommandDiagnosticsRequestIds
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.LiveMediaSessionResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import terminals.control.v1.Control
import java.io.EOFException

private data class ResponseContext(
    val pendingSnapshot: CommandDiagnosticsRequestIds,
    val rebaselineSent: Boolean,
    val notificationDelivered: Boolean,
    val audioResult: AudioPlaybackResult?,
    val mediaResult: MediaDisplayResult?,
    val liveMediaLine: String?,
)

/** Inbound control stream handler (parity with Flutter shell response dispatch). */
internal class AndroidTerminalInboundSink(
    private val viewModel: AndroidTerminalViewModel,
) : AndroidControlResponseSink {
    override suspend fun onResponse(response: Control.ConnectResponse) {
        val pendingSnapshot = viewModel.snapshotDebugCommandPendingIds()
        val rebaselineSent = if (response.requiresCapabilityRebaseline()) {
            runCatching { viewModel.session?.rebaselineCapabilitiesAfterStaleGeneration() }.isSuccess
        } else {
            false
        }
        val context = ResponseContext(
            pendingSnapshot = pendingSnapshot,
            rebaselineSent = rebaselineSent,
            notificationDelivered = handleNotification(response),
            audioResult = handleAudio(response),
            mediaResult = handleMedia(response),
            liveMediaLine = handleStreamAndMedia(response),
        )
        applyResponseState(response, context)
        handleRegisterAck(response)
        handleHelloAck(response)
    }

    private fun handleNotification(response: Control.ConnectResponse): Boolean {
        if (response.payloadCase != Control.ConnectResponse.PayloadCase.NOTIFICATION) return false
        val title = response.notification.title.trim()
        val body = response.notification.body.trim()
        val hasContent = title.isNotEmpty() || body.isNotEmpty()
        val notificationOk = hasContent &&
            runCatching { viewModel.dependencies.notificationDelivery.deliver(title, body) }.isSuccess
        if (hasContent) {
            val spoken = if (body.isNotEmpty()) body else title
            runCatching { viewModel.dependencies.speechDelivery.speak(spoken) }
        }
        return notificationOk
    }

    private fun handleAudio(response: Control.ConnectResponse): AudioPlaybackResult? {
        if (response.payloadCase != Control.ConnectResponse.PayloadCase.PLAY_AUDIO) return null
        return runCatching { viewModel.dependencies.mediaEngine.playAudio(response.playAudio) }
            .getOrElse { AudioPlaybackResult.Unsupported(it.message ?: it::class.java.simpleName) }
    }

    private fun handleMedia(response: Control.ConnectResponse): MediaDisplayResult? {
        if (response.payloadCase != Control.ConnectResponse.PayloadCase.SHOW_MEDIA) return null
        return runCatching { viewModel.dependencies.mediaEngine.showMedia(response.showMedia) }
            .getOrElse { MediaDisplayResult.Unsupported(it.message ?: it::class.java.simpleName) }
    }

    private suspend fun handleStreamAndMedia(response: Control.ConnectResponse): String? {
        var liveMediaLine: String? = null
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.START_STREAM) {
            liveMediaLine = handleStartStream(viewModel, response)
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.STOP_STREAM) {
            val streamId = response.stopStream.streamId.trim()
            if (streamId.isNotEmpty()) viewModel.dependencies.mediaEngine.applyStopStream(streamId)
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.ROUTE_STREAM) {
            viewModel.dependencies.mediaEngine.applyRouteStream(response.routeStream)
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.WEBRTC_SIGNAL) {
            liveMediaLine = liveMediaLine ?: handleWebRtcSignal(viewModel, response)
        }
        return liveMediaLine
    }

    private fun applyResponseState(response: Control.ConnectResponse, ctx: ResponseContext) {
        viewModel.mutableState.update {
            val dispatched = viewModel.dispatcher.dispatch(it, response)
            val enriched =
                viewModel.applyCommandResultDiagnostics(dispatched, response, ctx.pendingSnapshot).copy(
                    inboundConnectResponseCount = it.inboundConnectResponseCount + 1,
                )
            val endpoint = viewModel.parser.parse(enriched.endpointText)
            var diagnostics = viewModel.formatDiagnostics(endpoint, enriched.connectionState, enriched)
            if (ctx.notificationDelivered) {
                val head = response.notification.title.trim().ifEmpty { response.notification.body.trim() }
                diagnostics += "\nlast_notification=$head"
            }
            val mediaStatus =
                ctx.audioResult?.toStatus(response.playAudio.requestId)
                    ?: ctx.mediaResult?.toStatus(response.showMedia.requestId)
            if (mediaStatus != null) {
                diagnostics += "\nlast_media=${mediaStatus.first}:${mediaStatus.second}"
            }
            val resolvedLiveMediaLine = ctx.liveMediaLine ?: enriched.lastLiveMediaLine
            resolvedLiveMediaLine?.takeIf(String::isNotBlank)?.let { line ->
                diagnostics += "\nlast_live_media=$line"
            }
            enriched.copy(
                diagnosticsText =
                if (ctx.rebaselineSent) {
                    "$diagnostics\nlast_capability_rebaseline=stale-generation"
                } else {
                    diagnostics
                },
                lastMediaRequestId = mediaStatus?.first ?: enriched.lastMediaRequestId,
                lastMediaStatus = mediaStatus?.second ?: enriched.lastMediaStatus,
                lastLiveMediaLine = resolvedLiveMediaLine,
            )
        }
    }

    private suspend fun handleRegisterAck(response: Control.ConnectResponse) {
        if (response.payloadCase != Control.ConnectResponse.PayloadCase.REGISTER_ACK) return
        viewModel.sawRegisterAck = true
        if (!viewModel.registerAckScenarioQuerySent) {
            viewModel.registerAckScenarioQuerySent = true
            viewModel.viewModelScope.launch { viewModel.sendScenarioRegistryQuery() }
        }
        val flushIntent = viewModel.mutableState.value.applicationLaunchQueuedIntent?.trim().orEmpty()
        if (flushIntent.isNotEmpty()) {
            viewModel.mutableState.update { st ->
                val cleared = st.copy(applicationLaunchQueuedIntent = null, lastError = null)
                cleared.copy(
                    diagnosticsText =
                    viewModel.formatDiagnostics(
                        viewModel.parser.parse(cleared.endpointText),
                        cleared.connectionState,
                        cleared,
                    ),
                )
            }
            viewModel.viewModelScope.launch { viewModel.sendApplicationLaunchNow(flushIntent) }
        }
    }

    private fun handleHelloAck(response: Control.ConnectResponse) {
        if (response.payloadCase != Control.ConnectResponse.PayloadCase.HELLO_ACK) return
        val ms = response.helloAck.heartbeatIntervalMs
        viewModel.effectiveHeartbeatMillis =
            if (ms > 0) ms else viewModel.dependencies.heartbeatIntervalMillis
        val connectedSession = viewModel.session
        if (connectedSession != null) {
            viewModel.startHeartbeat(connectedSession)
            viewModel.startSensorTelemetry(connectedSession)
            viewModel.startCapabilityMonitor(connectedSession)
        }
    }

    override suspend fun onTransportTerminated(error: Throwable?) {
        val connectedSession = viewModel.session ?: return
        viewModel.handleControlLoss(connectedSession, error ?: EOFException("control transport closed"))
    }
}

private suspend fun handleStartStream(viewModel: AndroidTerminalViewModel, response: Control.ConnectResponse): String? {
    val streamId = response.startStream.streamId.trim()
    if (streamId.isEmpty()) return null
    val connectedSession = viewModel.session
    if (connectedSession != null) {
        runCatching { connectedSession.sendStreamReady(streamId) }
            .onSuccess {
                viewModel.mutableState.update { state ->
                    val next = state.copy(streamReadySendCount = state.streamReadySendCount + 1)
                    next.copy(
                        diagnosticsText =
                        viewModel.formatDiagnostics(
                            viewModel.parser.parse(next.endpointText),
                            next.connectionState,
                            next,
                        ),
                    )
                }
            }
            .onFailure { viewModel.handleControlLoss(connectedSession, it) }
    }
    return when (val lr = viewModel.dependencies.mediaEngine.applyStartStream(response.startStream)) {
        is LiveMediaSessionResult.Unsupported -> "start_stream:$streamId:${lr.reason}"
        is LiveMediaSessionResult.Applied -> "start_stream:$streamId:applied"
    }
}

private fun handleWebRtcSignal(viewModel: AndroidTerminalViewModel, response: Control.ConnectResponse): String? {
    val signalStreamId = response.webrtcSignal.streamId.trim()
    return when (val sr = viewModel.dependencies.mediaEngine.applyWebRtcSignal(response.webrtcSignal)) {
        is LiveMediaSessionResult.Unsupported -> "webrtc_signal:${sr.reason}"
        is LiveMediaSessionResult.Applied -> "webrtc_signal:$signalStreamId:applied"
    }
}
