package com.curtcox.terminals.android.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlResponseDispatcher
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ManualEndpointParser
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import terminals.control.v1.Control

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
                runCatching {
                    dependencies.notificationDelivery.deliver(response.notification.title, response.notification.body)
                }.isSuccess
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
            mutableState.update {
                val next = dispatcher.dispatch(it, response)
                var diagnostics = formatDiagnostics(parser.parse(next.endpointText), next.connectionState)
                if (notificationDelivered) {
                    diagnostics += "\nlast_notification=${response.notification.title}"
                }
                val mediaStatus = audioResult?.toStatus(response.playAudio.requestId)
                    ?: mediaResult?.toStatus(response.showMedia.requestId)
                if (mediaStatus != null) {
                    diagnostics += "\nlast_media=${mediaStatus.first}:${mediaStatus.second}"
                }
                next.copy(
                    diagnosticsText = if (rebaselineSent) {
                        "$diagnostics\nlast_capability_rebaseline=stale-generation"
                    } else {
                        diagnostics
                    },
                    lastMediaRequestId = mediaStatus?.first ?: next.lastMediaRequestId,
                    lastMediaStatus = mediaStatus?.second ?: next.lastMediaStatus,
                )
            }
        }
    }
    private var session: AndroidControlSession? = null
    private var heartbeatJob: Job? = null
    private var reconnectJob: Job? = null
    private val mutableState = MutableStateFlow(
        initialState(),
    )

    val state: StateFlow<AndroidTerminalViewState> = mutableState

    fun updateEndpoint(text: String) {
        val resolved = parser.parse(text)
        if (resolved != null) {
            dependencies.terminalSettings.setLastManualEndpoint(text)
        }
        mutableState.update {
            it.copy(
                endpointText = text,
                connectionState = if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect,
                lastError = if (resolved == null && text.isNotBlank()) "Enter a host:port or http(s) URL." else null,
                diagnosticsText = formatDiagnostics(resolved, if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect),
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

        mutableState.update {
            it.copy(
                connectionState = ConnectionState.Connecting,
                lastError = null,
                diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connecting),
            )
        }
        viewModelScope.launch {
            runCatching {
                stopReconnect()
                stopHeartbeat()
                session?.close()
                val nextSession = dependencies.sessionFactory(responseSink)
                session = nextSession
                nextSession.connect(resolved)
                dependencies.terminalSettings.setLastManualEndpoint(mutableState.value.endpointText)
                startHeartbeat(nextSession)
            }.onSuccess {
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connected,
                        lastError = null,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connected),
                    )
                }
            }.onFailure { error ->
                stopHeartbeat()
                session = null
                mutableState.update {
                    val message = error.message ?: error::class.java.simpleName
                    it.copy(
                        connectionState = ConnectionState.ReadyToConnect,
                        lastError = message,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.ReadyToConnect) +
                            "\nlast_error=$message",
                    )
                }
            }
        }
    }

    fun disconnect() {
        val closingSession = session
        session = null
        stopHeartbeat()
        stopReconnect()
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            it.copy(
                connectionState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.ReadyToConnect,
                diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Disconnected),
            )
        }
        viewModelScope.launch { closingSession?.close() }
    }

    fun sendUiAction(action: ServerDrivenAction) {
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
                diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState)}\n" +
                    "last_permission_refresh=$reason\n" +
                    permissions.toDiagnostics() + "\n" +
                    mediaSupport.toDiagnostics(),
            )
        }
    }

    fun refreshNetworkDiagnostics(reason: String) {
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            it.copy(diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState)}\nlast_network_refresh=$reason")
        }
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

    fun setFullscreen(enabled: Boolean) {
        runCatching {
            dependencies.fullscreenController.setFullscreen(enabled)
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
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

    override fun onCleared() {
        disconnect()
        super.onCleared()
    }

    private fun startHeartbeat(connectedSession: AndroidControlSession) {
        if (dependencies.heartbeatIntervalMillis <= 0) return
        heartbeatJob = viewModelScope.launch {
            while (true) {
                delay(dependencies.heartbeatIntervalMillis)
                runCatching {
                    connectedSession.sendHeartbeat()
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

    private fun stopReconnect() {
        reconnectJob?.cancel()
        reconnectJob = null
    }

    private fun handleControlLoss(failedSession: AndroidControlSession, error: Throwable) {
        heartbeatJob = null
        if (session !== failedSession) return
        session = null
        val endpoint = parser.parse(mutableState.value.endpointText)
        val message = error.message ?: error::class.java.simpleName
        mutableState.update {
            it.copy(
                connectionState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.Connecting,
                lastError = message,
                diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connecting) +
                    "\nlast_error=$message\nreconnect_pending=${endpoint != null}",
            )
        }
        viewModelScope.launch { failedSession.close() }
        if (endpoint != null) {
            startReconnect(endpoint, message)
        }
    }

    private fun startReconnect(endpoint: EndpointResolution, cause: String) {
        stopReconnect()
        reconnectJob = viewModelScope.launch {
            var lastError = cause
            for (attempt in 1..dependencies.maxReconnectAttempts) {
                delay(dependencies.reconnectPolicy.delayForAttempt(attempt))
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connecting,
                        diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connecting) +
                            "\nlast_error=$lastError\nreconnect_attempt=$attempt",
                    )
                }
                val nextSession = dependencies.sessionFactory(responseSink)
                val connected = runCatching {
                    nextSession.connect(endpoint)
                    session = nextSession
                    startHeartbeat(nextSession)
                }.onFailure { error ->
                    lastError = error.message ?: error::class.java.simpleName
                    nextSession.close()
                }.isSuccess
                if (connected) {
                    mutableState.update {
                        it.copy(
                            connectionState = ConnectionState.Connected,
                            lastError = null,
                            diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connected) +
                                "\nreconnect_success_attempt=$attempt",
                        )
                    }
                    reconnectJob = null
                    return@launch
                }
            }
            mutableState.update {
                it.copy(
                    connectionState = ConnectionState.ReadyToConnect,
                    lastError = lastError,
                    diagnosticsText = formatDiagnostics(endpoint, ConnectionState.ReadyToConnect) +
                        "\nlast_error=$lastError\nreconnect_exhausted=${dependencies.maxReconnectAttempts}",
                )
            }
            reconnectJob = null
        }
    }

    private fun Control.ConnectResponse.requiresCapabilityRebaseline(): Boolean {
        if (!hasError()) return false
        if (error.code != Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) return false
        return error.message.contains("stale", ignoreCase = true) &&
            error.message.contains("generation", ignoreCase = true)
    }

    private fun formatDiagnostics(endpoint: EndpointResolution?, state: ConnectionState): String =
        chrome.formatDiagnostics(
            endpoint = endpoint,
            state = state,
            networkState = runCatching { dependencies.networkStateProvider.current() }.getOrNull(),
        )

    private fun initialState(): AndroidTerminalViewState {
        val lastEndpoint = runCatching { dependencies.terminalSettings.lastManualEndpoint() }.getOrDefault("")
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
            diagnosticsText = formatDiagnostics(resolved, state),
            permissionEducation = permissionEducation(),
            mediaSupport = mediaSupport(),
        )
    }

    private fun permissionEducation(): PermissionEducationState {
        val snapshot = runCatching { dependencies.capabilityProbe.current() }.getOrNull()
            ?: return PermissionEducationState()
        return PermissionEducationState(
            notificationsGranted = snapshot.permissions.notificationsGranted,
            microphonePresent = snapshot.hardware.microphone,
            microphoneAvailable = snapshot.hardware.microphone && snapshot.permissions.microphoneGranted,
            cameraPresent = snapshot.hardware.frontCamera || snapshot.hardware.backCamera,
            cameraAvailable = (snapshot.hardware.frontCamera || snapshot.hardware.backCamera) &&
                snapshot.permissions.cameraGranted,
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
