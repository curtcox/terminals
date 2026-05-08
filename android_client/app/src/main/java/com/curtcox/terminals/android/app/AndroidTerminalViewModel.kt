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
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.media.AudioPlaybackResult
import com.curtcox.terminals.android.media.MediaDisplayResult
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlin.coroutines.coroutineContext
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
    private var connectJob: Job? = null
    private var heartbeatJob: Job? = null
    private var reconnectJob: Job? = null
    private var networkMonitoringActive: Boolean = false
    private var lastDiscoveryRestartAtMillis: Long = -1
    private var lastNetworkCapabilityRefreshAtMillis: Long = -1
    private var lastNetworkReconnectRestoreAtMillis: Long = -1
    private var reconnectExhausted: Boolean = false
    private val mutableState = MutableStateFlow(
        initialState(),
    )

    val state: StateFlow<AndroidTerminalViewState> = mutableState

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
        reconnectExhausted = false

        mutableState.update {
            it.copy(
                connectionState = ConnectionState.Connecting,
                lastError = null,
                diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connecting),
            )
        }
        stopConnect()
        connectJob = viewModelScope.launch {
            val thisJob = coroutineContext[Job]
            var nextSession: AndroidControlSession? = null
            try {
                stopReconnect()
                stopHeartbeat()
                session?.close()
                nextSession = dependencies.sessionFactory(responseSink)
                session = nextSession
                nextSession.connect(resolved)
                dependencies.terminalSettings.setLastManualEndpoint(mutableState.value.endpointText)
                startHeartbeat(nextSession)
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connected,
                        lastError = null,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connected),
                    )
                }
            } catch (error: CancellationException) {
                if (session === nextSession) {
                    session = null
                }
                runCatching { nextSession?.close() }
                throw error
            } catch (error: Throwable) {
                stopHeartbeat()
                if (session === nextSession) {
                    session = null
                }
                runCatching { nextSession?.close() }
                mutableState.update {
                    val message = error.message ?: error::class.java.simpleName
                    it.copy(
                        connectionState = ConnectionState.ReadyToConnect,
                        lastError = message,
                        diagnosticsText = formatDiagnostics(resolved, ConnectionState.ReadyToConnect) +
                            "\nlast_error=$message",
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
                diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\ndiscovery=scanning",
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
                        diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\n" +
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
                        diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\n" +
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
                diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\ndiscovery=stopped",
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
        stopReconnect()
        reconnectExhausted = false
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            val nextState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.ReadyToConnect
            it.copy(
                connectionState = nextState,
                diagnosticsText = formatDiagnostics(endpoint, nextState),
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
                    "last_permission_refresh=$reason",
            )
        }
    }

    fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
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
            it.copy(diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState)}\nlast_network_refresh=$reason")
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
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\n" +
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
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\n" +
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
                    diagnosticsText = "${formatDiagnostics(parser.parse(it.endpointText), it.connectionState)}\n" +
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

    private fun discoveredEndpointText(server: DiscoveredServer): String =
        server.webSocketEndpoint.ifBlank {
            server.grpcEndpoint.ifBlank {
                server.httpEndpoint.ifBlank {
                    "${server.host}:${server.port}"
                }
            }
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

    private fun stopConnect() {
        connectJob?.cancel()
        connectJob = null
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

    private fun startReconnect(endpoint: EndpointResolution, errorContext: String, reconnectCause: String = errorContext) {
        stopReconnect()
        reconnectJob = viewModelScope.launch {
            var lastError = errorContext
            for (attempt in 1..dependencies.maxReconnectAttempts) {
                delay(dependencies.reconnectPolicy.delayForAttempt(attempt))
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connecting,
                        diagnosticsText = formatDiagnostics(endpoint, ConnectionState.Connecting) +
                            "\nlast_error=$lastError\nreconnect_attempt=$attempt\nreconnect_cause=$reconnectCause",
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
                                "\nreconnect_success_attempt=$attempt\nreconnect_cause=$reconnectCause",
                        )
                    }
                    reconnectExhausted = false
                    reconnectJob = null
                    return@launch
                }
            }
            mutableState.update {
                it.copy(
                    connectionState = ConnectionState.ReadyToConnect,
                    lastError = lastError,
                    diagnosticsText = formatDiagnostics(endpoint, ConnectionState.ReadyToConnect) +
                        "\nlast_error=$lastError\nreconnect_exhausted=${dependencies.maxReconnectAttempts}\nreconnect_cause=$reconnectCause",
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

    private fun formatDiagnostics(endpoint: EndpointResolution?, state: ConnectionState): String {
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
            diagnosticsText = formatDiagnostics(resolved, state),
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
