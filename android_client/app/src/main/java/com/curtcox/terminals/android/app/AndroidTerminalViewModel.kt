package com.curtcox.terminals.android.app

import android.Manifest
import android.os.Build
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlResponseDispatcher
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ManualEndpointParser
import com.curtcox.terminals.android.diagnostics.AndroidBugReportActions
import com.curtcox.terminals.android.diagnostics.AndroidBugReportBuilder
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.LiveMediaSessionResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.curtcox.terminals.android.util.Clock
import java.io.EOFException
import java.util.Locale
import java.util.TimeZone
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlin.coroutines.coroutineContext
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics

class AndroidTerminalViewModel(
    private val dependencies: AndroidClientDependencies = AndroidClientDependencies(),
) : ViewModel() {
    private val parser = ManualEndpointParser()
    private val chrome = AndroidClientChrome(dependencies.buildMetadata)
    private val dispatcher = ControlResponseDispatcher()
    private val responseSink = object : AndroidControlResponseSink {
        override suspend fun onResponse(response: Control.ConnectResponse) {
            val rebaselineSent = if (response.requiresCapabilityRebaseline()) {
                runCatching {
                    session?.rebaselineCapabilitiesAfterStaleGeneration()
                }.isSuccess
            } else {
                false
            }
            val notificationDelivered = if (response.payloadCase == Control.ConnectResponse.PayloadCase.NOTIFICATION) {
                val title = response.notification.title.trim()
                val body = response.notification.body.trim()
                if (title.isEmpty() && body.isEmpty()) {
                    false
                } else {
                    val notificationOk = runCatching {
                        dependencies.notificationDelivery.deliver(title, body)
                    }.isSuccess
                    val spoken = if (body.isNotEmpty()) body else title
                    runCatching { dependencies.speechDelivery.speak(spoken) }
                    notificationOk
                }
            } else {
                false
            }
            val audioResult = if (response.payloadCase == Control.ConnectResponse.PayloadCase.PLAY_AUDIO) {
                runCatching { dependencies.mediaEngine.playAudio(response.playAudio) }
                    .getOrElse { AudioPlaybackResult.Unsupported(it.message ?: it::class.java.simpleName) }
            } else {
                null
            }
            val mediaResult = if (response.payloadCase == Control.ConnectResponse.PayloadCase.SHOW_MEDIA) {
                runCatching { dependencies.mediaEngine.showMedia(response.showMedia) }
                    .getOrElse { MediaDisplayResult.Unsupported(it.message ?: it::class.java.simpleName) }
            } else {
                null
            }
            var liveMediaLine: String? = null
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.START_STREAM) {
                val streamId = response.startStream.streamId.trim()
                if (streamId.isNotEmpty()) {
                    val connectedSession = session
                    if (connectedSession != null) {
                        runCatching { connectedSession.sendStreamReady(streamId) }
                            .onFailure { handleControlLoss(connectedSession, it) }
                    }
                    when (val lr = dependencies.mediaEngine.applyStartStream(response.startStream)) {
                        is LiveMediaSessionResult.Unsupported ->
                            liveMediaLine = "start_stream:$streamId:${lr.reason}"
                        else -> {}
                    }
                }
            }
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.STOP_STREAM) {
                val streamId = response.stopStream.streamId.trim()
                if (streamId.isNotEmpty()) {
                    dependencies.mediaEngine.applyStopStream(streamId)
                }
            }
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.ROUTE_STREAM) {
                dependencies.mediaEngine.applyRouteStream(response.routeStream)
            }
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.WEBRTC_SIGNAL) {
                when (val sr = dependencies.mediaEngine.applyWebRtcSignal(response.webrtcSignal)) {
                    is LiveMediaSessionResult.Unsupported ->
                        liveMediaLine = liveMediaLine ?: "webrtc_signal:${sr.reason}"
                    else -> {}
                }
            }
            mutableState.update {
                val next = dispatcher.dispatch(it, response)
                var diagnostics = formatDiagnostics(parser.parse(next.endpointText), next.connectionState, next)
                if (notificationDelivered) {
                    val head = response.notification.title.trim()
                        .ifEmpty { response.notification.body.trim() }
                    diagnostics += "\nlast_notification=$head"
                }
                val mediaStatus = audioResult?.toStatus(response.playAudio.requestId)
                    ?: mediaResult?.toStatus(response.showMedia.requestId)
                if (mediaStatus != null) {
                    diagnostics += "\nlast_media=${mediaStatus.first}:${mediaStatus.second}"
                }
                val resolvedLiveMediaLine = liveMediaLine ?: next.lastLiveMediaLine
                resolvedLiveMediaLine?.takeIf { it.isNotBlank() }?.let { line ->
                    diagnostics += "\nlast_live_media=$line"
                }
                next.copy(
                    diagnosticsText = if (rebaselineSent) {
                        "$diagnostics\nlast_capability_rebaseline=stale-generation"
                    } else {
                        diagnostics
                    },
                    lastMediaRequestId = mediaStatus?.first ?: next.lastMediaRequestId,
                    lastMediaStatus = mediaStatus?.second ?: next.lastMediaStatus,
                    lastLiveMediaLine = resolvedLiveMediaLine,
                )
            }
            if (response.payloadCase == Control.ConnectResponse.PayloadCase.HELLO_ACK) {
                val ms = response.helloAck.heartbeatIntervalMs
                effectiveHeartbeatMillis = if (ms > 0) ms else dependencies.heartbeatIntervalMillis
                val connectedSession = session
                if (connectedSession != null) {
                    startHeartbeat(connectedSession)
                    startSensorTelemetry(connectedSession)
                }
            }
        }

        override suspend fun onTransportTerminated(error: Throwable?) {
            val connectedSession = session ?: return
            handleControlLoss(connectedSession, error ?: EOFException("control transport closed"))
        }
    }
    private var session: AndroidControlSession? = null
    private var connectJob: Job? = null
    private var heartbeatJob: Job? = null
    private var sensorTelemetryJob: Job? = null
    private var reconnectJob: Job? = null
    private var networkMonitoringActive: Boolean = false
    private var lastDiscoveryRestartAtMillis: Long = -1
    private var lastNetworkCapabilityRefreshAtMillis: Long = -1
    private var lastNetworkReconnectRestoreAtMillis: Long = -1
    private var reconnectExhausted: Boolean = false
    private var effectiveHeartbeatMillis: Long = dependencies.heartbeatIntervalMillis
    /** When false, periodic heartbeat and sensor telemetry are paused (Flutter `AppLifecycle` parity). */
    private var appInForeground: Boolean = true
    private val bugReportClock: Clock = Clock(dependencies.nowMillis)
    private val bugReportQueue: ArrayDeque<Diagnostics.BugReport> = ArrayDeque()
    private val mutableState = MutableStateFlow(
        initialState(),
    )

    val state: StateFlow<AndroidTerminalViewState> = mutableState

    /**
     * Mirrors Flutter terminal shell behavior: outbound heartbeat and sensor telemetry loops run only
     * while the app is foregrounded (`Activity.onStart` / `Activity.onStop`).
     */
    fun setAppForegrounded(foregrounded: Boolean) {
        if (appInForeground == foregrounded) return
        appInForeground = foregrounded
        if (!foregrounded) {
            stopHeartbeat()
            stopSensorTelemetry()
            refreshCapabilitiesIfConnected("app_lifecycle_change")
            return
        }
        val connectedSession = session ?: return
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        startHeartbeat(connectedSession)
        startSensorTelemetry(connectedSession)
        refreshCapabilitiesIfConnected("app_lifecycle_change")
    }

    fun updateEndpoint(text: String) {
        val resolved = parser.parse(text)
        if (resolved != null) {
            dependencies.terminalSettings.setLastManualEndpoint(text)
        }
        reconnectExhausted = false
        mutableState.update {
            it.copy(
                endpointText = text,
                connectionState = if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect,
                lastError = if (resolved == null && text.isNotBlank()) "Enter a host:port or http(s) URL." else null,
                diagnosticsText = formatDiagnostics(
                    resolved,
                    if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect,
                    it,
                ),
            )
        }
    }

    fun connect() {
        val resolved = parser.parse(mutableState.value.endpointText)
        if (resolved == null) {
            mutableState.update {
                it.copy(connectionState = ConnectionState.InvalidEndpoint, lastError = "Endpoint is not valid.")
            }
            return
        }
        reconnectExhausted = false
        effectiveHeartbeatMillis = dependencies.heartbeatIntervalMillis

        mutableState.update {
            withoutHandshake(it).copy(
                connectionState = ConnectionState.Connecting,
                lastError = null,
                diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connecting, withoutHandshake(it)),
            )
        }
        stopConnect()
        connectJob = viewModelScope.launch {
            val thisJob = coroutineContext[Job]
            var nextSession: AndroidControlSession? = null
            try {
                stopReconnect()
                stopHeartbeat()
                stopSensorTelemetry()
                session?.close()
                nextSession = dependencies.sessionFactory(responseSink)
                session = nextSession
                nextSession.connect(resolved)
                dependencies.terminalSettings.setLastManualEndpoint(mutableState.value.endpointText)
                startHeartbeat(nextSession)
                startSensorTelemetry(nextSession)
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connected,
                        lastError = null,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connected, it),
                    )
                }
                flushQueuedBugReports(nextSession)
            } catch (error: CancellationException) {
                if (session === nextSession) {
                    session = null
                }
                runCatching { nextSession?.close() }
                throw error
            } catch (error: Throwable) {
                stopHeartbeat()
                stopSensorTelemetry()
                if (session === nextSession) {
                    session = null
                }
                runCatching { nextSession?.close() }
                mutableState.update {
                    val message = error.message ?: error::class.java.simpleName
                    val cleared = withoutHandshake(it).copy(lastError = message)
                    cleared.copy(
                        connectionState = ConnectionState.ReadyToConnect,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.ReadyToConnect, cleared),
                    )
                }
            } finally {
                if (connectJob === thisJob) {
                    connectJob = null
                }
            }
        }
    }

    fun startDiscovery() {
        mutableState.update {
            it.copy(
                discoveryState = it.discoveryState.copy(scanning = true, lastError = null),
                diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\ndiscovery=scanning",
            )
        }
        dependencies.discovery.start(
            onServer = { server ->
                mutableState.update {
                    val current = it.discoveryState.servers
                    val nextServers = (current.filterNot { existing ->
                        existing.host == server.host && existing.port == server.port
                    } + server).sortedWith(compareBy<DiscoveredServer> { discoveredEndpointText(it) }.thenBy { it.name })
                    it.copy(
                        discoveryState = it.discoveryState.copy(
                            scanning = true,
                            servers = nextServers,
                            lastError = null,
                        ),
                        diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\n" +
                            "discovery=scanning\n" +
                            "discovered_servers=${nextServers.size}\n" +
                            "last_discovered=${server.name}@${discoveredEndpointText(server)}",
                    )
                }
            },
            onError = { message ->
                mutableState.update {
                    it.copy(
                        discoveryState = it.discoveryState.copy(scanning = false, lastError = message),
                        diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\n" +
                            "discovery=error\n" +
                            "discovery_error=$message",
                    )
                }
            },
        )
    }

    fun stopDiscovery() {
        dependencies.discovery.stop()
        mutableState.update {
            it.copy(
                discoveryState = it.discoveryState.copy(scanning = false),
                diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\ndiscovery=stopped",
            )
        }
    }

    fun selectDiscoveredServer(server: DiscoveredServer) {
        updateEndpoint(discoveredEndpointText(server))
        stopDiscovery()
    }

    fun disconnect() {
        val closingSession = session
        session = null
        stopConnect()
        stopHeartbeat()
        stopSensorTelemetry()
        stopReconnect()
        reconnectExhausted = false
        effectiveHeartbeatMillis = dependencies.heartbeatIntervalMillis
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            val nextState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.ReadyToConnect
            val clearedHandshake = withoutHandshake(it)
            val diagnosticsSource = clearedHandshake.copy(
                lastBugReportAckDiagnostics = it.lastBugReportAckDiagnostics,
                lastControlErrorCode = it.lastControlErrorCode,
                registerAckMessage = it.registerAckMessage,
                registerAckServerId = it.registerAckServerId,
                registerAckAssetBaseUrl = it.registerAckAssetBaseUrl,
                lastControlResponseActivity = it.lastControlResponseActivity,
            )
            diagnosticsSource.copy(
                connectionState = nextState,
                diagnosticsText = formatDiagnostics(endpoint, nextState, diagnosticsSource),
            )
        }
        viewModelScope.launch { closingSession?.close() }
    }

    fun sendUiAction(action: ServerDrivenAction) {
        if (action.action.startsWith(AndroidBugReportActions.PREFIX)) {
            submitBugReportFromServerDrivenAction(action)
            return
        }
        viewModelScope.launch {
            runCatching {
                session?.sendUiAction(action) ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update {
                    it.copy(diagnosticsText = "${it.diagnosticsText}\nlast_ui_action=${action.componentId}:${action.action}:${action.value}")
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    /** Flutter `BugReportButton` / shell filing parity — sends [Diagnostics.BugReport] on the control stream. */
    fun submitChromeBugReport() {
        val report =
            buildShellBugReport(
                description = "Filed from native Android terminal shell",
                source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
                subjectDeviceId = dependencies.deviceId,
                extraHints = mapOf("entry_point" to "native_android_shell"),
            )
        queueOrSendBugReport(report)
    }

    private fun submitBugReportFromServerDrivenAction(action: ServerDrivenAction) {
        val subject = resolveBugReportSubject(action)
        val report =
            buildShellBugReport(
                description = "Filed from on-device bug report button",
                source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
                subjectDeviceId = subject,
                extraHints =
                    mapOf(
                        "component_id" to action.componentId,
                        "action" to action.action,
                    ),
            )
        queueOrSendBugReport(report)
    }

    private fun resolveBugReportSubject(action: ServerDrivenAction): String {
        if (action.action.startsWith(AndroidBugReportActions.PREFIX)) {
            val explicit =
                action.action
                    .removePrefix(AndroidBugReportActions.PREFIX)
                    .removePrefix(":")
                    .trim()
            if (explicit.isNotEmpty()) {
                return explicit
            }
        }
        val v = action.value.trim()
        if (v.isNotEmpty()) {
            return v
        }
        return dependencies.deviceId
    }

    private fun buildShellBugReport(
        description: String,
        source: Diagnostics.BugReportSource,
        subjectDeviceId: String,
        extraHints: Map<String, String>,
    ): Diagnostics.BugReport {
        val s = mutableState.value
        return AndroidBugReportBuilder.build(
            description = description,
            source = source,
            reporterDeviceId = dependencies.deviceId,
            subjectDeviceId = subjectDeviceId,
            extraSourceHints = extraHints,
            clock = bugReportClock,
            buildMetadata = dependencies.buildMetadata,
            serverRoot = s.serverRoot,
            connectionState = s.connectionState,
            lastServerHeartbeatUnixMs = s.lastServerHeartbeatUnixMs,
            registeredCapabilities = session?.lastRegisteredCapabilities,
            localeTag = Locale.getDefault().toLanguageTag(),
            timezoneId = TimeZone.getDefault().id,
            osVersion = "${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})",
        )
    }

    private fun queueOrSendBugReport(report: Diagnostics.BugReport) {
        viewModelScope.launch {
            val currentSession = session
            val connected = mutableState.value.connectionState == ConnectionState.Connected
            if (currentSession != null && connected) {
                runCatching { currentSession.sendBugReport(report) }
                    .onSuccess {
                        val word = report.sourceHintsMap["bug_token_word"] ?: ""
                        mutableState.update {
                            it.copy(
                                lastBugReportSubmitStatus = "Sent bug report ${report.reportId} (word=$word).",
                                lastError = null,
                            )
                        }
                    }
                    .onFailure { e ->
                        mutableState.update {
                            it.copy(
                                lastBugReportSubmitStatus =
                                    "Bug report send failed: ${e.message ?: e.javaClass.simpleName}",
                            )
                        }
                    }
            } else {
                bugReportQueue.addLast(report)
                val word = report.sourceHintsMap["bug_token_word"] ?: ""
                mutableState.update {
                    it.copy(
                        lastBugReportSubmitStatus = "Queued bug report (word=$word) until connected.",
                    )
                }
            }
        }
    }

    private suspend fun flushQueuedBugReports(target: AndroidControlSession) {
        val pending = bugReportQueue.size
        if (pending == 0) return
        var sent = 0
        var lastWord = ""
        var lastFailure: String? = null
        while (bugReportQueue.isNotEmpty()) {
            val report = bugReportQueue.removeFirst()
            runCatching { target.sendBugReport(report) }
                .onSuccess {
                    sent++
                    lastWord = report.sourceHintsMap["bug_token_word"] ?: ""
                }
                .onFailure { e ->
                    lastFailure = e.message ?: e.javaClass.simpleName
                }
        }
        val status =
            when {
                sent == pending && sent == 1 -> "Sent queued bug report (word=$lastWord)."
                sent == pending && sent > 1 -> "Sent $sent queued bug reports (last word=$lastWord)."
                sent > 0 ->
                    "Sent $sent of $pending queued bug reports (last word=$lastWord). Remainder failed: $lastFailure"
                else -> "Queued bug reports failed to send: $lastFailure"
            }
        mutableState.update { it.copy(lastBugReportSubmitStatus = status) }
    }

    /**
     * Shell `terminal_input` parity with Flutter `_sendKeyText`: streams UTF-16 text chunks (including
     * `"\b"` backspace repeats and `"\n"` on IME done) as protobuf `InputEvent.key.text`.
     */
    fun sendTerminalKeyText(text: String) {
        if (text.isEmpty()) return
        viewModelScope.launch {
            runCatching {
                session?.sendKeyText(text) ?: return@launch
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    fun refreshCapabilities(reason: String) {
        val connectedSession = session ?: return
        viewModelScope.launch {
            runCatching {
                connectedSession.sendCapabilityDeltaIfChanged(reason)
            }.onSuccess { sent ->
                if (sent) {
                    mutableState.update {
                        it.copy(diagnosticsText = "${it.diagnosticsText}\nlast_capability_delta=$reason")
                    }
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    fun refreshPermissionEducation(reason: String) {
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            val permissions = permissionEducation()
            val mediaSupport = mediaSupport()
            it.copy(
                permissionEducation = permissions,
                mediaSupport = mediaSupport,
                diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState, it)}\n" +
                    "last_permission_refresh=$reason",
            )
        }
    }

    fun requestNotificationPermission() {
        if (!dependencies.runtimeNotificationPermissionPromptSupported) {
            refreshPermissionEducation("notification-permission-not-required")
            return
        }
        requestPermission(Manifest.permission.POST_NOTIFICATIONS, "notification-permission")
    }

    fun requestMicrophonePermission() {
        requestPermission(Manifest.permission.RECORD_AUDIO, "microphone-permission")
    }

    fun requestCameraPermission() {
        requestPermission(Manifest.permission.CAMERA, "camera-permission")
    }

    fun requestMissingPermissions() {
        val permissions = mutableState.value.permissionEducation
        if (!permissions.notificationsGranted) {
            requestNotificationPermission()
        }
        if (permissions.microphonePresent && !permissions.microphoneAvailable) {
            requestMicrophonePermission()
        }
        if (permissions.cameraPresent && !permissions.cameraAvailable) {
            requestCameraPermission()
        }
    }

    fun refreshNetworkDiagnostics(reason: String) {
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            it.copy(diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState, it)}\nlast_network_refresh=$reason")
        }
    }

    fun startNetworkMonitoring() {
        if (networkMonitoringActive) return
        networkMonitoringActive = true
        dependencies.networkMonitor.start {
            refreshNetworkDiagnostics("network-callback")
            refreshCapabilitiesFromNetworkCallback("network-callback")
            restartDiscoveryIfScanning("network-callback")
            retryConnectIfReconnectExhausted("network-callback")
        }
    }

    fun stopNetworkMonitoring() {
        if (!networkMonitoringActive) return
        networkMonitoringActive = false
        dependencies.networkMonitor.stop()
    }

    fun copyDiagnostics() {
        val diagnostics = mutableState.value.diagnosticsText
        runCatching {
            dependencies.diagnosticClipboard.copy(diagnostics)
        }.onSuccess {
            mutableState.update {
                it.copy(lastDiagnosticsCopyStatus = "copied")
            }
        }.onFailure { error ->
            mutableState.update {
                it.copy(
                    lastDiagnosticsCopyStatus = "failed",
                    lastError = error.message ?: error::class.java.simpleName,
                )
            }
        }
    }

    fun setKeepAwake(enabled: Boolean) {
        runCatching {
            dependencies.keepAwakeController.setKeepAwake(enabled)
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
    }

    fun setLocalKeepAwake(enabled: Boolean) {
        runCatching {
            dependencies.keepAwakeController.setKeepAwake(enabled)
        }.onSuccess {
            dependencies.terminalSettings.setKeepAwakeEnabled(enabled)
            mutableState.update {
                it.copy(
                    localKeepAwakeEnabled = enabled,
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\n" +
                        "local_keep_awake=$enabled",
                )
            }
        }.onFailure { error ->
            mutableState.update {
                it.copy(
                    lastError = error.message ?: error::class.java.simpleName,
                    localKeepAwakeEnabled = dependencies.terminalSettings.keepAwakeEnabled(),
                )
            }
        }
    }

    fun setFullscreen(enabled: Boolean) {
        runCatching {
            dependencies.fullscreenController.setFullscreen(enabled)
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
    }

    fun setLocalFullscreen(enabled: Boolean) {
        runCatching {
            dependencies.fullscreenController.setFullscreen(enabled)
        }.onSuccess {
            dependencies.terminalSettings.setFullscreenEnabled(enabled)
            mutableState.update {
                it.copy(
                    localFullscreenEnabled = enabled,
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\n" +
                        "local_fullscreen=$enabled",
                )
            }
        }.onFailure { error ->
            mutableState.update {
                it.copy(
                    lastError = error.message ?: error::class.java.simpleName,
                    localFullscreenEnabled = dependencies.terminalSettings.fullscreenEnabled(),
                )
            }
        }
    }

    fun setBrightness(value: Double) {
        runCatching {
            dependencies.brightnessController.setBrightness(value)
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
    }

    fun setLocalBrightDisplay(enabled: Boolean) {
        runCatching {
            dependencies.brightnessController.setBrightness(if (enabled) 1.0 else 0.5)
        }.onSuccess {
            dependencies.terminalSettings.setBrightDisplayEnabled(enabled)
            mutableState.update {
                it.copy(
                    localBrightDisplayEnabled = enabled,
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState, it)}\n" +
                        "local_bright_display=$enabled",
                )
            }
        }.onFailure { error ->
            mutableState.update {
                it.copy(
                    lastError = error.message ?: error::class.java.simpleName,
                    localBrightDisplayEnabled = dependencies.terminalSettings.brightDisplayEnabled(),
                )
            }
        }
    }

    override fun onCleared() {
        stopNetworkMonitoring()
        dependencies.discovery.stop()
        disconnect()
        super.onCleared()
    }

    private fun discoveredEndpointText(server: DiscoveredServer): String {
        val ws = server.webSocketEndpoint.trim()
        if (ws.isNotEmpty()) return ws
        val grpc = server.grpcEndpoint.trim()
        if (grpc.isNotEmpty()) {
            return if (grpc.contains("://")) grpc else "grpc://$grpc"
        }
        val http = server.httpEndpoint.trim()
        if (http.isNotEmpty()) return http
        return "${server.host}:${server.port}"
    }

    private fun startHeartbeat(connectedSession: AndroidControlSession) {
        stopHeartbeat()
        val intervalMs = when {
            effectiveHeartbeatMillis > 0 -> effectiveHeartbeatMillis
            else -> dependencies.heartbeatIntervalMillis
        }
        if (intervalMs <= 0 || !appInForeground) return
        heartbeatJob = viewModelScope.launch {
            while (true) {
                delay(intervalMs)
                runCatching {
                    connectedSession.sendHeartbeat()
                }.onFailure { error ->
                    handleControlLoss(connectedSession, error)
                    return@launch
                }
            }
        }
    }

    private fun startSensorTelemetry(connectedSession: AndroidControlSession) {
        stopSensorTelemetry()
        val intervalMs = dependencies.sensorTelemetryIntervalMillis
        if (intervalMs <= 0 || !appInForeground) return
        sensorTelemetryJob = viewModelScope.launch {
            while (true) {
                delay(intervalMs)
                runCatching {
                    connectedSession.sendSensorTelemetry()
                }.onFailure { error ->
                    handleControlLoss(connectedSession, error)
                    return@launch
                }
            }
        }
    }

    private fun stopHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = null
    }

    private fun stopSensorTelemetry() {
        sensorTelemetryJob?.cancel()
        sensorTelemetryJob = null
    }

    private fun stopConnect() {
        connectJob?.cancel()
        connectJob = null
    }

    private fun stopReconnect() {
        reconnectJob?.cancel()
        reconnectJob = null
    }

    private fun handleControlLoss(failedSession: AndroidControlSession, error: Throwable) {
        stopHeartbeat()
        stopSensorTelemetry()
        if (session !== failedSession) return
        session = null
        val endpoint = parser.parse(mutableState.value.endpointText)
        val message = error.message ?: error::class.java.simpleName
        mutableState.update {
            val next = it.copy(
                connectionState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.Connecting,
                lastError = message,
            )
            next.copy(
                diagnosticsText = formatDiagnostics(endpoint, next.connectionState, next) +
                    "\nreconnect_pending=${endpoint != null}",
            )
        }
        viewModelScope.launch { failedSession.close() }
        if (endpoint != null) {
            startReconnect(endpoint, message)
        }
    }

    private fun startReconnect(endpoint: EndpointResolution, errorContext: String, reconnectCause: String = errorContext) {
        stopReconnect()
        reconnectJob = viewModelScope.launch {
            var lastError = errorContext
            for (attempt in 1..dependencies.maxReconnectAttempts) {
                delay(dependencies.reconnectPolicy.delayForAttempt(attempt))
                mutableState.update {
                    val basis = it.copy(connectionState = ConnectionState.Connecting, lastError = lastError)
                    basis.copy(
                        diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connecting, basis) +
                            "\nreconnect_attempt=$attempt\nreconnect_cause=$reconnectCause",
                    )
                }
                val nextSession = dependencies.sessionFactory(responseSink)
                val connected = runCatching {
                    nextSession.connect(endpoint)
                    session = nextSession
                    startHeartbeat(nextSession)
                    startSensorTelemetry(nextSession)
                }.onFailure { error ->
                    lastError = error.message ?: error::class.java.simpleName
                    nextSession.close()
                }.isSuccess
                if (connected) {
                    mutableState.update {
                        it.copy(
                            connectionState = ConnectionState.Connected,
                            lastError = null,
                            diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connected, it) +
                                "\nreconnect_success_attempt=$attempt\nreconnect_cause=$reconnectCause",
                        )
                    }
                    flushQueuedBugReports(nextSession)
                    reconnectExhausted = false
                    reconnectJob = null
                    return@launch
                }
            }
            mutableState.update {
                val basis = it.copy(connectionState = ConnectionState.ReadyToConnect, lastError = lastError)
                basis.copy(
                    diagnosticsText = formatDiagnostics(endpoint, ConnectionState.ReadyToConnect, basis) +
                        "\nreconnect_exhausted=${dependencies.maxReconnectAttempts}\nreconnect_cause=$reconnectCause",
                )
            }
            reconnectExhausted = true
            reconnectJob = null
        }
    }

    private fun Control.ConnectResponse.requiresCapabilityRebaseline(): Boolean {
        if (!hasError()) return false
        if (error.code != Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) return false
        return error.message.contains("stale", ignoreCase = true) &&
            error.message.contains("generation", ignoreCase = true)
    }

    private fun withoutHandshake(state: AndroidTerminalViewState): AndroidTerminalViewState =
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
        )

    private fun formatDiagnostics(
        endpoint: EndpointResolution?,
        state: ConnectionState,
        handshakeSource: AndroidTerminalViewState? = null,
    ): String {
        val capabilitySnapshot = runCatching { dependencies.capabilityProbe.current() }.getOrNull()
        val permissions = permissionEducation(capabilitySnapshot)
        val mediaSupport = mediaSupport()
        val controlStatus = session?.status
        return buildString {
            append(
                chrome.formatDiagnostics(
                    endpoint = endpoint,
                    state = state,
                    networkState = runCatching { dependencies.networkStateProvider.current() }.getOrNull(),
                    fireOsDeviceInfo = runCatching { dependencies.fireOsDeviceInfoProvider.current() }.getOrNull(),
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
        }
    }

    private fun initialState(): AndroidTerminalViewState {
        val lastEndpoint = runCatching { dependencies.terminalSettings.lastManualEndpoint() }.getOrDefault("")
        val keepAwakeEnabled = runCatching { dependencies.terminalSettings.keepAwakeEnabled() }.getOrDefault(false)
        val fullscreenEnabled = runCatching { dependencies.terminalSettings.fullscreenEnabled() }.getOrDefault(false)
        val brightDisplayEnabled = runCatching { dependencies.terminalSettings.brightDisplayEnabled() }.getOrDefault(false)
        if (keepAwakeEnabled) {
            runCatching { dependencies.keepAwakeController.setKeepAwake(true) }
        }
        if (fullscreenEnabled) {
            runCatching { dependencies.fullscreenController.setFullscreen(true) }
        }
        if (brightDisplayEnabled) {
            runCatching { dependencies.brightnessController.setBrightness(1.0) }
        }
        val resolved = parser.parse(lastEndpoint)
        val state = when {
            lastEndpoint.isBlank() -> ConnectionState.Disconnected
            resolved != null -> ConnectionState.ReadyToConnect
            else -> ConnectionState.InvalidEndpoint
        }
        return AndroidTerminalViewState(
            endpointText = lastEndpoint,
            connectionState = state,
            lastError = if (state == ConnectionState.InvalidEndpoint) "Enter a host:port or http(s) URL." else null,
            diagnosticsText = formatDiagnostics(resolved, state, null),
            localKeepAwakeEnabled = keepAwakeEnabled,
            localFullscreenEnabled = fullscreenEnabled,
            localBrightDisplayEnabled = brightDisplayEnabled,
            permissionEducation = permissionEducation(),
            mediaSupport = mediaSupport(),
        )
    }

    private fun permissionEducation(snapshot: AndroidCapabilitySnapshotInput? = null): PermissionEducationState {
        val capabilitySnapshot = snapshot ?: runCatching { dependencies.capabilityProbe.current() }.getOrNull()
            ?: return PermissionEducationState()
        return PermissionEducationState(
            notificationsGranted = capabilitySnapshot.permissions.notificationsGranted,
            microphonePresent = capabilitySnapshot.hardware.microphone,
            microphoneAvailable = capabilitySnapshot.hardware.microphone && capabilitySnapshot.permissions.microphoneGranted,
            cameraPresent = capabilitySnapshot.hardware.frontCamera || capabilitySnapshot.hardware.backCamera,
            cameraAvailable = (capabilitySnapshot.hardware.frontCamera || capabilitySnapshot.hardware.backCamera) &&
                capabilitySnapshot.permissions.cameraGranted,
        )
    }

    private fun PermissionEducationState.toDiagnostics(): String = buildString {
        appendLine("permission_notifications=$notificationsGranted")
        appendLine("permission_microphone_present=$microphonePresent")
        appendLine("permission_microphone_available=$microphoneAvailable")
        appendLine("permission_camera_present=$cameraPresent")
        append("permission_camera_available=$cameraAvailable")
    }

    private fun mediaSupport(): MediaSupportState {
        val permissions = runCatching { dependencies.mediaPermissionProbe.current() }.getOrNull()
        val webRtc = runCatching { dependencies.webRtcAdapter.currentSupport() }.getOrNull()
        return MediaSupportState(
            microphonePermissionGranted = permissions?.microphoneGranted == true,
            cameraPermissionGranted = permissions?.cameraGranted == true,
            webRtcSupported = webRtc?.supported == true,
            webRtcReason = webRtc?.reason?.ifBlank { "available" } ?: "unavailable",
        )
    }

    private fun requestPermission(permission: String, reason: String) {
        if (dependencies.permissionRequester.hasPermission(permission)) {
            refreshPermissionEducation("$reason-already-granted")
            refreshCapabilities(reason)
            return
        }
        dependencies.permissionRequester.requestPermission(permission) { granted ->
            viewModelScope.launch {
                mutableState.update {
                    it.copy(diagnosticsText = "${it.diagnosticsText}\n$reason-granted=$granted")
                }
                refreshPermissionEducation("$reason-result")
                refreshCapabilities(reason)
            }
        }
    }

    private fun restartDiscoveryIfScanning(reason: String) {
        if (!mutableState.value.discoveryState.scanning) return
        if (!appInForeground) {
            mutableState.update {
                it.copy(
                    diagnosticsText = "${it.diagnosticsText}\ndiscovery_restart_suppressed=app-background",
                )
            }
            return
        }
        val now = dependencies.nowMillis()
        if (lastDiscoveryRestartAtMillis >= 0 &&
            now - lastDiscoveryRestartAtMillis < dependencies.discoveryRestartMinIntervalMillis
        ) {
            mutableState.update {
                it.copy(
                    diagnosticsText = "${it.diagnosticsText}\ndiscovery_restart_suppressed=$reason",
                )
            }
            return
        }
        lastDiscoveryRestartAtMillis = now
        dependencies.discovery.stop()
        startDiscovery()
        mutableState.update {
            it.copy(
                diagnosticsText = "${it.diagnosticsText}\ndiscovery_restart_reason=$reason",
            )
        }
    }

    private fun retryConnectIfReconnectExhausted(reason: String) {
        if (!reconnectExhausted) return
        if (mutableState.value.connectionState != ConnectionState.ReadyToConnect) return
        val resolved = parser.parse(mutableState.value.endpointText) ?: return
        if (connectJob != null || reconnectJob != null) return
        val now = dependencies.nowMillis()
        if (lastNetworkReconnectRestoreAtMillis >= 0 &&
            now - lastNetworkReconnectRestoreAtMillis < dependencies.networkReconnectRestoreMinIntervalMillis
        ) {
            mutableState.update {
                it.copy(
                    diagnosticsText = "${it.diagnosticsText}\nnetwork_reconnect_restore_suppressed=$reason",
                )
            }
            return
        }
        lastNetworkReconnectRestoreAtMillis = now
        val lastError = mutableState.value.lastError ?: "reconnect exhausted"
        startReconnect(
            resolved,
            errorContext = lastError,
            reconnectCause = "network-restore:$reason",
        )
    }

    private fun refreshCapabilitiesFromNetworkCallback(reason: String) {
        if (!appInForeground) {
            mutableState.update {
                it.copy(
                    diagnosticsText = "${it.diagnosticsText}\ncapability_refresh_suppressed=app-background",
                )
            }
            return
        }
        val now = dependencies.nowMillis()
        if (lastNetworkCapabilityRefreshAtMillis >= 0 &&
            now - lastNetworkCapabilityRefreshAtMillis < dependencies.networkCapabilityRefreshMinIntervalMillis
        ) {
            mutableState.update {
                it.copy(
                    diagnosticsText = "${it.diagnosticsText}\ncapability_refresh_suppressed=$reason",
                )
            }
            return
        }
        lastNetworkCapabilityRefreshAtMillis = now
        refreshCapabilities(reason)
    }

    /** Matches Flutter shell lifecycle: capability delta on foreground/background transitions (`app_lifecycle_change`). */
    private fun refreshCapabilitiesIfConnected(reason: String) {
        if (session == null) return
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        refreshCapabilities(reason)
    }

    private fun AudioPlaybackResult.toStatus(requestId: String): Pair<String, String> =
        when (this) {
            is AudioPlaybackResult.Played -> (this.requestId.ifBlank { requestId }) to "played"
            is AudioPlaybackResult.Unsupported -> requestId to "unsupported-audio:$reason"
        }

    private fun MediaDisplayResult.toStatus(requestId: String): Pair<String, String> =
        when (this) {
            is MediaDisplayResult.Shown -> (this.requestId.ifBlank { requestId }) to "shown"
            is MediaDisplayResult.Unsupported -> requestId to "unsupported-media:$reason"
        }
}
