package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.app.AndroidTerminalViewState
import com.curtcox.terminals.android.diagnostics.AndroidBugReportChrome
import terminals.control.v1.Control
import terminals.io.v1.Io
import terminals.ui.v1.Ui

/** Human-readable last-message label; order matches Flutter `statusFromConnectResponse` where applicable. */
fun connectResponseActivityStatus(response: Control.ConnectResponse): String =
    when (response.payloadCase) {
        Control.ConnectResponse.PayloadCase.ERROR -> "Server error"
        Control.ConnectResponse.PayloadCase.TRANSITION_UI -> "UI transition"
        Control.ConnectResponse.PayloadCase.START_STREAM -> "Stream started"
        Control.ConnectResponse.PayloadCase.STOP_STREAM -> "Stream stopped"
        Control.ConnectResponse.PayloadCase.ROUTE_STREAM -> "Route updated"
        Control.ConnectResponse.PayloadCase.WEBRTC_SIGNAL -> "WebRTC signal"
        Control.ConnectResponse.PayloadCase.PLAY_AUDIO -> "Play audio"
        Control.ConnectResponse.PayloadCase.SHOW_MEDIA -> "Show media"
        Control.ConnectResponse.PayloadCase.INSTALL_BUNDLE -> "Bundle install requested"
        Control.ConnectResponse.PayloadCase.REMOVE_BUNDLE -> "Bundle removal requested"
        Control.ConnectResponse.PayloadCase.START_FLOW -> "Flow start requested"
        Control.ConnectResponse.PayloadCase.PATCH_FLOW -> "Flow patch requested"
        Control.ConnectResponse.PayloadCase.STOP_FLOW -> "Flow stop requested"
        Control.ConnectResponse.PayloadCase.REQUEST_ARTIFACT -> "Artifact requested"
        Control.ConnectResponse.PayloadCase.BUG_REPORT_ACK -> "Bug report filed"
        Control.ConnectResponse.PayloadCase.UPDATE_UI -> "UI patched"
        Control.ConnectResponse.PayloadCase.REGISTER_ACK -> "Registered"
        Control.ConnectResponse.PayloadCase.COMMAND_RESULT -> "Command response"
        Control.ConnectResponse.PayloadCase.SET_UI -> "UI updated"
        Control.ConnectResponse.PayloadCase.NOTIFICATION -> "Notification"
        Control.ConnectResponse.PayloadCase.HELLO_ACK,
        Control.ConnectResponse.PayloadCase.CAPABILITY_ACK,
        Control.ConnectResponse.PayloadCase.HEARTBEAT,
        Control.ConnectResponse.PayloadCase.PAYLOAD_NOT_SET,
        -> "Connected"
        else -> "Control payload (${response.payloadCase.name})"
    }

class ControlResponseDispatcher {
    fun dispatch(
        state: AndroidTerminalViewState,
        response: Control.ConnectResponse,
    ): AndroidTerminalViewState {
        val next = when (response.payloadCase) {
            Control.ConnectResponse.PayloadCase.HELLO_ACK -> {
                val ack = response.helloAck
                state.copy(
                    controlServerId = ack.serverId.takeIf { it.isNotBlank() },
                    controlSessionId = ack.sessionId.takeIf { it.isNotBlank() },
                    serverHeartbeatIntervalMs = ack.heartbeatIntervalMs.takeIf { it > 0 },
                )
            }
            Control.ConnectResponse.PayloadCase.CAPABILITY_ACK -> {
                val ack = response.capabilityAck
                state.copy(
                    lastCapabilityAckGeneration = ack.acceptedGeneration,
                    lastCapabilityAckSnapshotApplied = ack.snapshotApplied,
                    lastCapabilityInvalidationsSummary = summarizeCapabilityInvalidations(ack),
                )
            }
            Control.ConnectResponse.PayloadCase.REGISTER_ACK -> mergeRegisterAck(state, response)
            Control.ConnectResponse.PayloadCase.SET_UI -> state.copy(serverRoot = response.setUi.root)
            Control.ConnectResponse.PayloadCase.UPDATE_UI -> {
                val root = state.serverRoot
                if (root == null) {
                    state
                } else {
                    state.copy(serverRoot = replaceNode(root, response.updateUi.componentId, response.updateUi.node))
                }
            }
            Control.ConnectResponse.PayloadCase.TRANSITION_UI -> {
                val tu = response.transitionUi
                state.copy(
                    lastTransition = tu.transition.takeIf { it.isNotBlank() },
                    lastTransitionDurationMs = tu.durationMs.takeIf { it > 0 },
                )
            }
            Control.ConnectResponse.PayloadCase.NOTIFICATION -> state.copy(
                lastNotificationTitle = response.notification.title,
                lastNotificationBody = response.notification.body,
            )
            Control.ConnectResponse.PayloadCase.ERROR -> state.copy(
                lastError = response.error.message.takeIf { it.isNotBlank() },
                lastControlErrorCode = response.error.code.name,
            )
            Control.ConnectResponse.PayloadCase.BUG_REPORT_ACK -> state.copy(
                lastBugReportAckDiagnostics = AndroidBugReportChrome.formatDiagnosticsLines(response.bugReportAck),
            )
            Control.ConnectResponse.PayloadCase.HEARTBEAT -> state.copy(
                lastServerHeartbeatUnixMs = response.heartbeat.unixMs.takeIf { it > 0 },
            )
            Control.ConnectResponse.PayloadCase.COMMAND_RESULT -> {
                val result = response.commandResult
                state.copy(
                    lastCommandResultRequestId = result.requestId.takeIf { it.isNotBlank() },
                    lastCommandResultNotification = result.notification.takeIf { it.isNotBlank() },
                )
            }
            Control.ConnectResponse.PayloadCase.START_STREAM ->
                state.copy(lastOpaqueControlIoSummary = summarizeStartStream(response.startStream))
            Control.ConnectResponse.PayloadCase.STOP_STREAM ->
                state.copy(lastOpaqueControlIoSummary = summarizeStopStream(response.stopStream))
            Control.ConnectResponse.PayloadCase.ROUTE_STREAM ->
                state.copy(lastOpaqueControlIoSummary = summarizeRouteStream(response.routeStream))
            Control.ConnectResponse.PayloadCase.WEBRTC_SIGNAL ->
                state.copy(lastOpaqueControlIoSummary = summarizeWebRtcSignal(response.webrtcSignal))
            Control.ConnectResponse.PayloadCase.INSTALL_BUNDLE ->
                state.copy(lastOpaqueControlIoSummary = summarizeInstallBundle(response.installBundle))
            Control.ConnectResponse.PayloadCase.REMOVE_BUNDLE ->
                state.copy(lastOpaqueControlIoSummary = summarizeRemoveBundle(response.removeBundle))
            Control.ConnectResponse.PayloadCase.START_FLOW ->
                state.copy(lastOpaqueControlIoSummary = summarizeStartFlow(response.startFlow))
            Control.ConnectResponse.PayloadCase.PATCH_FLOW ->
                state.copy(lastOpaqueControlIoSummary = summarizePatchFlow(response.patchFlow))
            Control.ConnectResponse.PayloadCase.STOP_FLOW ->
                state.copy(lastOpaqueControlIoSummary = summarizeStopFlow(response.stopFlow))
            Control.ConnectResponse.PayloadCase.REQUEST_ARTIFACT ->
                state.copy(lastOpaqueControlIoSummary = summarizeRequestArtifact(response.requestArtifact))
            Control.ConnectResponse.PayloadCase.PLAY_AUDIO,
            Control.ConnectResponse.PayloadCase.SHOW_MEDIA,
            -> state.copy(lastOpaqueControlIoSummary = null)
            Control.ConnectResponse.PayloadCase.PAYLOAD_NOT_SET -> state
            else -> state.copy(
                lastOpaqueControlIoSummary = "type=unhandled_payload payload_case=${response.payloadCase.name}",
            )
        }
        return if (response.payloadCase == Control.ConnectResponse.PayloadCase.PAYLOAD_NOT_SET) {
            next
        } else {
            next.copy(lastControlResponseActivity = connectResponseActivityStatus(response))
        }
    }

    private fun summarizeCapabilityInvalidations(ack: Control.CapabilityAck): String? {
        if (ack.invalidationsCount == 0) return null
        val list = ack.invalidationsList
        val maxShown = 4
        return buildString {
            list.take(maxShown).forEachIndexed { index, inv ->
                if (index > 0) append("; ")
                val resource = inv.resource.takeIf { it.isNotBlank() } ?: "?"
                val reason = inv.reason.takeIf { it.isNotBlank() } ?: "-"
                append(resource).append(':').append(reason)
            }
            if (list.size > maxShown) {
                append("; +").append(list.size - maxShown).append(" more")
            }
        }
    }

    private fun mergeRegisterAck(state: AndroidTerminalViewState, response: Control.ConnectResponse): AndroidTerminalViewState {
        val ack = response.registerAck
        val metaMap = ack.metadataMap
        val serverMeta = if (ack.hasServerMetadata()) ack.serverMetadata else null
        val build = serverMeta?.takeIf { it.hasBuild() }?.build
        val sha = build?.sha?.takeIf { it.isNotBlank() }
            ?: metaMap["server_build_sha"]?.takeIf { it.isNotBlank() }
        val date = build?.dateRfc3339?.takeIf { it.isNotBlank() }
            ?: metaMap["server_build_date"]?.takeIf { it.isNotBlank() }
        val assetBase = serverMeta?.photoFrameAssetBaseUrl?.takeIf { it.isNotBlank() }
        val message = ack.message.takeIf { it.isNotBlank() }
        val serverId = ack.serverId.takeIf { it.isNotBlank() }
        return state.copy(
            serverBuildSha = sha ?: state.serverBuildSha,
            serverBuildDate = date ?: state.serverBuildDate,
            registerAckMessage = message ?: state.registerAckMessage,
            registerAckServerId = serverId ?: state.registerAckServerId,
            registerAckAssetBaseUrl = assetBase ?: state.registerAckAssetBaseUrl,
        )
    }

    private fun replaceNode(root: Ui.Node, componentId: String, replacement: Ui.Node): Ui.Node {
        if (root.id == componentId) return replacement
        if (root.childrenCount == 0) return root

        var changed = false
        val children = root.childrenList.map { child ->
            val next = replaceNode(child, componentId, replacement)
            if (next != child) changed = true
            next
        }
        if (!changed) return root

        return root.toBuilder()
            .clearChildren()
            .addAllChildren(children)
            .build()
    }

    private fun summarizeStartStream(stream: Io.StartStream): String = buildString {
        append("type=start_stream")
        append(" stream_id=").append(stream.streamId.takeIf { it.isNotBlank() } ?: "none")
        if (stream.streamKind != Io.StreamKind.STREAM_KIND_UNSPECIFIED) {
            append(" stream_kind=").append(stream.streamKind.name)
        }
        if (stream.kind.isNotBlank()) {
            append(" kind=").append(stream.kind)
        }
    }

    private fun summarizeStopStream(stream: Io.StopStream): String =
        "type=stop_stream stream_id=${stream.streamId.takeIf { it.isNotBlank() } ?: "none"}"

    private fun summarizeRouteStream(route: Io.RouteStream): String = buildString {
        append("type=route_stream")
        append(" stream_id=").append(route.streamId.takeIf { it.isNotBlank() } ?: "none")
        if (route.streamKind != Io.StreamKind.STREAM_KIND_UNSPECIFIED) {
            append(" stream_kind=").append(route.streamKind.name)
        }
        if (route.kind.isNotBlank()) {
            append(" kind=").append(route.kind)
        }
    }

    private fun summarizeWebRtcSignal(signal: Control.WebRTCSignal): String = buildString {
        append("type=webrtc_signal")
        append(" stream_id=").append(signal.streamId.takeIf { it.isNotBlank() } ?: "none")
        when {
            signal.signalTypeEnum != Control.WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_UNSPECIFIED ->
                append(" signal_type=").append(signal.signalTypeEnum.name)
            signal.signalType.isNotBlank() -> append(" signal_type=").append(signal.signalType)
            else -> Unit
        }
    }

    private fun summarizeInstallBundle(bundle: Io.InstallBundle): String = buildString {
        append("type=install_bundle bundle_id=").append(bundle.bundleId.takeIf { it.isNotBlank() } ?: "none")
        if (bundle.version.isNotBlank()) {
            append(" version=").append(bundle.version)
        }
        if (bundle.sha256.isNotBlank()) {
            val sha = bundle.sha256
            val short = if (sha.length > 12) sha.take(12) + "..." else sha
            append(" sha256_prefix=").append(short)
        }
        val tarLen = bundle.tarGz.size()
        if (tarLen > 0) {
            append(" tar_gz_bytes=").append(tarLen)
        }
    }

    private fun summarizeRemoveBundle(bundle: Io.RemoveBundle): String =
        "type=remove_bundle bundle_id=${bundle.bundleId.takeIf { it.isNotBlank() } ?: "none"}"

    private fun summarizeStartFlow(flow: Io.StartFlow): String =
        "type=start_flow flow_id=${flow.flowId.takeIf { it.isNotBlank() } ?: "none"} " +
            "nodes=${flow.plan.nodesCount} edges=${flow.plan.edgesCount}"

    private fun summarizePatchFlow(flow: Io.PatchFlow): String =
        "type=patch_flow flow_id=${flow.flowId.takeIf { it.isNotBlank() } ?: "none"} " +
            "nodes=${flow.plan.nodesCount} edges=${flow.plan.edgesCount}"

    private fun summarizeStopFlow(flow: Io.StopFlow): String =
        "type=stop_flow flow_id=${flow.flowId.takeIf { it.isNotBlank() } ?: "none"}"

    private fun summarizeRequestArtifact(request: Io.RequestArtifact): String =
        "type=request_artifact artifact_id=${request.artifactId.takeIf { it.isNotBlank() } ?: "none"}"
}
