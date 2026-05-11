package com.curtcox.terminals.android.app

import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.LiveMediaSessionResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import kotlinx.coroutines.launch
import terminals.control.v1.Control
import java.io.EOFException

/** Inbound control stream handler (parity with Flutter shell response dispatch). */
internal class AndroidTerminalInboundSink(
    private val viewModel: AndroidTerminalViewModel,
) : AndroidControlResponseSink {
    override suspend fun onResponse(response: Control.ConnectResponse) {
        val pendingSnapshot = viewModel.snapshotDebugCommandPendingIds()
        val rebaselineSent =
            if (response.requiresCapabilityRebaseline()) {
                runCatching {
                    viewModel.session?.rebaselineCapabilitiesAfterStaleGeneration()
                }.isSuccess
            } else {
                false
            }
        val notificationDelivered =
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.NOTIFICATION) {
                val title = response.notification.title.trim()
                val body = response.notification.body.trim()
                if (title.isEmpty() && body.isEmpty()) {
                    false
                } else {
                    val notificationOk =
                        runCatching {
                            viewModel.dependencies.notificationDelivery.deliver(title, body)
                        }.isSuccess
                    val spoken = if (body.isNotEmpty()) body else title
                    runCatching { viewModel.dependencies.speechDelivery.speak(spoken) }
                    notificationOk
                }
            } else {
                false
            }
        val audioResult =
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.PLAY_AUDIO) {
                runCatching { viewModel.dependencies.mediaEngine.playAudio(response.playAudio) }
                    .getOrElse {
                        AudioPlaybackResult.Unsupported(
                            it.message ?: it::class.java.simpleName,
                        )
                    }
            } else {
                null
            }
        val mediaResult =
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.SHOW_MEDIA) {
                runCatching { viewModel.dependencies.mediaEngine.showMedia(response.showMedia) }
                    .getOrElse {
                        MediaDisplayResult.Unsupported(
                            it.message ?: it::class.java.simpleName,
                        )
                    }
            } else {
                null
            }
        var liveMediaLine: String? = null
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.START_STREAM) {
            val streamId = response.startStream.streamId.trim()
            if (streamId.isNotEmpty()) {
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
                when (val lr = viewModel.dependencies.mediaEngine.applyStartStream(response.startStream)) {
                    is LiveMediaSessionResult.Unsupported ->
                        liveMediaLine = "start_stream:$streamId:${lr.reason}"
                    else -> {}
                }
            }
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.STOP_STREAM) {
            val streamId = response.stopStream.streamId.trim()
            if (streamId.isNotEmpty()) {
                viewModel.dependencies.mediaEngine.applyStopStream(streamId)
            }
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.ROUTE_STREAM) {
            viewModel.dependencies.mediaEngine.applyRouteStream(response.routeStream)
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.WEBRTC_SIGNAL) {
            when (val sr = viewModel.dependencies.mediaEngine.applyWebRtcSignal(response.webrtcSignal)) {
                is LiveMediaSessionResult.Unsupported ->
                    liveMediaLine = liveMediaLine ?: "webrtc_signal:${sr.reason}"
                else -> {}
            }
        }
        viewModel.mutableState.update {
            val dispatched = viewModel.dispatcher.dispatch(it, response)
            val enriched =
                viewModel.applyCommandResultDiagnostics(dispatched, response, pendingSnapshot).copy(
                    inboundConnectResponseCount = it.inboundConnectResponseCount + 1,
                )
            val endpoint = viewModel.parser.parse(enriched.endpointText)
            var diagnostics =
                viewModel.formatDiagnostics(
                    endpoint,
                    enriched.connectionState,
                    enriched,
                )
            if (notificationDelivered) {
                val head =
                    response.notification.title.trim()
                        .ifEmpty { response.notification.body.trim() }
                diagnostics += "\nlast_notification=$head"
            }
            val mediaStatus =
                audioResult?.toStatus(response.playAudio.requestId)
                    ?: mediaResult?.toStatus(response.showMedia.requestId)
            if (mediaStatus != null) {
                diagnostics += "\nlast_media=${mediaStatus.first}:${mediaStatus.second}"
            }
            val resolvedLiveMediaLine = liveMediaLine ?: enriched.lastLiveMediaLine
            resolvedLiveMediaLine?.takeIf(String::isNotBlank)?.let { line ->
                diagnostics += "\nlast_live_media=$line"
            }
            enriched.copy(
                diagnosticsText =
                    if (rebaselineSent) {
                        "$diagnostics\nlast_capability_rebaseline=stale-generation"
                    } else {
                        diagnostics
                    },
                lastMediaRequestId = mediaStatus?.first ?: enriched.lastMediaRequestId,
                lastMediaStatus = mediaStatus?.second ?: enriched.lastMediaStatus,
                lastLiveMediaLine = resolvedLiveMediaLine,
            )
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.REGISTER_ACK) {
            viewModel.sawRegisterAck = true
            if (!viewModel.registerAckScenarioQuerySent) {
                viewModel.registerAckScenarioQuerySent = true
                viewModel.viewModelScope.launch {
                    viewModel.sendScenarioRegistryQuery()
                }
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
                viewModel.viewModelScope.launch {
                    viewModel.sendApplicationLaunchNow(flushIntent)
                }
            }
        }
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.HELLO_ACK) {
            val ms = response.helloAck.heartbeatIntervalMs
            viewModel.effectiveHeartbeatMillis =
                if (ms > 0) {
                    ms
                } else {
                    viewModel.dependencies.heartbeatIntervalMillis
                }
            val connectedSession = viewModel.session
            if (connectedSession != null) {
                viewModel.startHeartbeat(connectedSession)
                viewModel.startSensorTelemetry(connectedSession)
                viewModel.startCapabilityMonitor(connectedSession)
            }
        }
    }

    override suspend fun onTransportTerminated(error: Throwable?) {
        val connectedSession = viewModel.session ?: return
        viewModel.handleControlLoss(connectedSession, error ?: EOFException("control transport closed"))
    }
}
