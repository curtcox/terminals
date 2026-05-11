@file:Suppress("ImportOrdering")

package com.curtcox.terminals.android.app

import android.Manifest
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.CommandDiagnosticsRequestIds
import com.curtcox.terminals.android.connection.ControlResponseDispatcher
import com.curtcox.terminals.android.connection.EndpointResolution
import com.curtcox.terminals.android.connection.ManualEndpointParser
import com.curtcox.terminals.android.connection.applicationIntentsFromDiagnostics
import com.curtcox.terminals.android.connection.commandResultDataMap
import com.curtcox.terminals.android.connection.diagnosticsTitleForCommandResult
import com.curtcox.terminals.android.connection.firstPlaybackArtifactId
import com.curtcox.terminals.android.diagnostics.AndroidBugReportActions
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.discovery.DiscoveredServer
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.curtcox.terminals.android.util.Clock
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics

class AndroidTerminalViewModel(
    internal val dependencies: AndroidClientDependencies = AndroidClientDependencies(),
) : ViewModel() {
    internal val parser = ManualEndpointParser()
    internal val chrome = AndroidClientChrome(dependencies.buildMetadata)
    internal val dispatcher = ControlResponseDispatcher()
    internal val responseSink = AndroidTerminalInboundSink(this)
    internal var session: AndroidControlSession? = null
    internal var connectJob: Job? = null
    internal var heartbeatJob: Job? = null
    internal var sensorTelemetryJob: Job? = null
    internal var capabilityMonitorJob: Job? = null
    internal var reconnectJob: Job? = null
    private var networkMonitoringActive: Boolean = false
    private var lastDiscoveryRestartAtMillis: Long = -1
    private var lastNetworkCapabilityRefreshAtMillis: Long = -1
    private var lastNetworkReconnectRestoreAtMillis: Long = -1
    internal var reconnectExhausted: Boolean = false
    internal var effectiveHeartbeatMillis: Long = dependencies.heartbeatIntervalMillis
    /** When false, periodic heartbeat and sensor telemetry are paused (Flutter `AppLifecycle` parity). */
    internal var appInForeground: Boolean = true
    internal var debugCommandSeq: Int = 0
    internal var pendingRuntimeStatusRequestId: String? = null
    internal var pendingDeviceStatusRequestId: String? = null
    internal var pendingScenarioRegistryRequestId: String? = null
    internal var pendingPlaybackArtifactsRequestId: String? = null
    internal var pendingPlaybackMetadataRequestId: String? = null

    /** First [Control.RegisterAck] per connection triggers automatic scenario registry query (Flutter shell). */
    internal var registerAckScenarioQuerySent: Boolean = false
    /** True after any inbound [Control.RegisterAck] for the active session (Flutter `_isConnectionRegistered` parity). */
    internal var sawRegisterAck: Boolean = false
    internal val bugReportClock: Clock = Clock(dependencies.nowMillis)
    internal val bugReportQueue: ArrayDeque<Diagnostics.BugReport> = ArrayDeque()
    internal val mutableState = MutableStateFlow(
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
            stopCapabilityMonitor()
            refreshCapabilitiesIfConnected("app_lifecycle_change")
            return
        }
        val connectedSession = session ?: return
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        startHeartbeat(connectedSession)
        startSensorTelemetry(connectedSession)
        startCapabilityMonitor(connectedSession)
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
        resetPerConnectionShellState()

        mutableState.update {
            withoutHandshake(it).copy(
                connectionState = ConnectionState.Connecting,
                lastError = null,
                diagnosticsText = formatDiagnostics(resolved, ConnectionState.Connecting, withoutHandshake(it)),
            )
        }
        stopConnect()
        connectJob = viewModelScope.launch {
            val thisJob = currentCoroutineContext()[Job]
            var nextSession: AndroidControlSession? = null
            try {
                stopReconnect()
                stopHeartbeat()
                stopSensorTelemetry()
                stopCapabilityMonitor()
                session?.close()
                nextSession = dependencies.sessionFactory(responseSink)
                nextSession.setPrivacyMode(mutableState.value.privacyModeEnabled)
                session = nextSession
                nextSession.connect(resolved)
                dependencies.terminalSettings.setLastManualEndpoint(mutableState.value.endpointText)
                if (appInForeground) {
                    runCatching { sendHeartbeatTracked(nextSession) }.onFailure { error ->
                        handleControlLoss(nextSession, error)
                        return@launch
                    }
                    runCatching { sendSensorTelemetryTracked(nextSession) }.onFailure { error ->
                        handleControlLoss(nextSession, error)
                        return@launch
                    }
                }
                startHeartbeat(nextSession)
                startSensorTelemetry(nextSession)
                startCapabilityMonitor(nextSession)
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connected,
                        lastError = null,
                        reconnectAttempt = 0,
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
                stopCapabilityMonitor()
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
                    } + server).sortedWith(
                        compareBy<DiscoveredServer>({ s -> discoveredEndpointText(s) }, { s -> s.name }),
                    )
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
        stopCapabilityMonitor()
        stopReconnect()
        reconnectExhausted = false
        effectiveHeartbeatMillis = dependencies.heartbeatIntervalMillis
        resetPerConnectionShellState()
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
        if (action.action == "privacy.toggle") {
            togglePrivacyMode()
            return
        }
        viewModelScope.launch {
            runCatching {
                session?.sendUiAction(action) ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    val actionLine =
                        "${st.diagnosticsText}\nlast_ui_action=" +
                            "${action.componentId}:${action.action}:${action.value}"
                    st.copy(diagnosticsText = actionLine)
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    /** Flutter shell **Runtime Status** — `COMMAND_KIND_SYSTEM` / `runtime_status`. */
    fun sendRuntimeStatusQuery() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        viewModelScope.launch {
            val id = nextDebugRequestId("debug-runtime-status")
            pendingRuntimeStatusRequestId = id
            runCatching {
                session?.sendSystemCommand(id, "runtime_status")
                    ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    st.copy(diagnosticsText = "${st.diagnosticsText}\nlast_system_command=runtime_status:$id")
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    /** Flutter shell **Device Status** — `COMMAND_KIND_SYSTEM` / `device_status <deviceId>`. */
    fun sendDeviceStatusQuery() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        viewModelScope.launch {
            val id = nextDebugRequestId("debug-device-status")
            pendingDeviceStatusRequestId = id
            val intent = "device_status ${dependencies.deviceId}"
            runCatching {
                session?.sendSystemCommand(id, intent)
                    ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    st.copy(diagnosticsText = "${st.diagnosticsText}\nlast_system_command=device_status:$id")
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    /** Flutter shell **List Playback Artifacts** — `COMMAND_KIND_SYSTEM` / `list_playback_artifacts`. */
    fun sendPlaybackArtifactsQuery() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        viewModelScope.launch {
            val id = nextDebugRequestId("debug-playback-artifacts")
            pendingPlaybackArtifactsRequestId = id
            runCatching {
                session?.sendSystemCommand(id, "list_playback_artifacts")
                    ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    st.copy(diagnosticsText = "${st.diagnosticsText}\nlast_system_command=list_playback_artifacts:$id")
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    fun updatePlaybackArtifactId(text: String) {
        mutableState.update { it.copy(playbackArtifactIdText = text) }
    }

    fun updatePlaybackTargetDeviceId(text: String) {
        mutableState.update { it.copy(playbackTargetDeviceIdText = text) }
    }

    /**
     * Flutter shell **Playback Metadata** — `COMMAND_KIND_MANUAL` / `playback_metadata` with
     * `artifact_id` and `target_device_id` arguments (defaults target to this device when blank).
     */
    fun sendPlaybackMetadataQuery() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        val artifact = mutableState.value.playbackArtifactIdText.trim()
        if (artifact.isEmpty()) {
            mutableState.update {
                it.copy(lastError = "Playback artifact ID required")
            }
            return
        }
        var target = mutableState.value.playbackTargetDeviceIdText.trim()
        if (target.isEmpty()) {
            target = dependencies.deviceId
            mutableState.update { it.copy(playbackTargetDeviceIdText = target) }
        }
        viewModelScope.launch {
            val id = nextDebugRequestId("debug-playback-metadata")
            pendingPlaybackMetadataRequestId = id
            runCatching {
                session?.sendPlaybackMetadataQuery(id, artifact, target)
                    ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    st.copy(
                        lastError = null,
                        diagnosticsText = "${st.diagnosticsText}\nlast_manual_command=playback_metadata:$id",
                    )
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    private fun nextDebugRequestId(prefix: String): String {
        debugCommandSeq += 1
        return "$prefix-$debugCommandSeq"
    }

    fun updateSelectedApplicationIntent(intent: String) {
        mutableState.update { it.copy(selectedApplicationIntent = intent) }
    }

    /** Flutter shell **Refresh Applications** — `COMMAND_KIND_SYSTEM` / `scenario_registry`. */
    fun sendScenarioRegistryQuery() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        viewModelScope.launch {
            val id = nextDebugRequestId("debug-scenario-registry")
            pendingScenarioRegistryRequestId = id
            runCatching {
                session?.sendSystemCommand(id, "scenario_registry")
                    ?: error("Control stream is not connected.")
            }.onSuccess {
                mutableState.update { st ->
                    st.copy(
                        lastError = null,
                        diagnosticsText = "${st.diagnosticsText}\nlast_system_command=scenario_registry:$id",
                    )
                }
            }.onFailure { error ->
                mutableState.update {
                    it.copy(lastError = error.message ?: error::class.java.simpleName)
                }
            }
        }
    }

    /** Flutter shell **Open Application** — manual `COMMAND_ACTION_START` with the selected intent. */
    fun submitApplicationLaunchCommand() {
        if (mutableState.value.connectionState != ConnectionState.Connected) return
        val intent = mutableState.value.selectedApplicationIntent.trim()
        if (intent.isEmpty()) {
            mutableState.update { it.copy(lastError = "Application intent required") }
            return
        }
        if (!sawRegisterAck) {
            mutableState.update { st ->
                val next = st.copy(applicationLaunchQueuedIntent = intent, lastError = null)
                next.copy(
                    diagnosticsText = formatDiagnostics(parser.parse(next.endpointText), next.connectionState, next),
                )
            }
            return
        }
        viewModelScope.launch {
            sendApplicationLaunchNow(intent)
        }
    }

    internal suspend fun sendApplicationLaunchNow(intent: String) {
        val id = nextDebugRequestId("debug-launch-app")
        runCatching {
            session?.sendApplicationLaunchCommand(id, intent)
                ?: error("Control stream is not connected.")
        }.onSuccess {
            mutableState.update { st ->
                st.copy(
                    lastError = null,
                    diagnosticsText = "${st.diagnosticsText}\nlast_manual_command=application_launch:$id:$intent",
                )
            }
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
    }

    /**
     * Flutter shell **Privacy** / `privacy.toggle` UI action: when turning privacy **on**, stops local
     * capture via the live-media seam first; toggles privacy mode; then sends a capability delta with
     * reason `privacy.toggle` when connected (Flutter does not stop capture when turning privacy off).
     */
    fun togglePrivacyMode() {
        val wasOff = !mutableState.value.privacyModeEnabled
        if (wasOff) {
            dependencies.mediaEngine.stopLocalCaptureStreamsForPrivacy()
        }
        val nextPrivacy = !mutableState.value.privacyModeEnabled
        session?.setPrivacyMode(nextPrivacy)
        mutableState.update { st ->
            val updated = st.copy(privacyModeEnabled = nextPrivacy)
            updated.copy(
                diagnosticsText = formatDiagnostics(
                    parser.parse(updated.endpointText),
                    updated.connectionState,
                    updated,
                ),
            )
        }
        refreshCapabilities("privacy.toggle")
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

    /**
     * Refreshes permission/media education and records both network and permission diagnostic timestamps
     * in one state update (so [refreshNetworkDiagnostics] does not get overwritten by
     * [refreshPermissionEducation]), then requests a capability delta when connected.
     */
    fun refreshShellDiagnosticsAndCapabilities(
        networkRefreshReason: String,
        permissionRefreshReason: String,
        capabilityDeltaReason: String,
    ) {
        mutableState.update {
            val endpoint = parser.parse(it.endpointText)
            val permissions = permissionEducation()
            val mediaSupport = mediaSupport()
            it.copy(
                permissionEducation = permissions,
                mediaSupport = mediaSupport,
                diagnosticsText = "${formatDiagnostics(endpoint, it.connectionState, it)}\n" +
                    "last_network_refresh=$networkRefreshReason\n" +
                    "last_permission_refresh=$permissionRefreshReason",
            )
        }
        refreshCapabilities(capabilityDeltaReason)
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
            mutableState.update { st ->
                st.copy(lastDiagnosticsCopyStatus = "copied")
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


    override fun onCleared() {
        stopNetworkMonitoring()
        dependencies.discovery.stop()
        disconnect()
        super.onCleared()
    }



    private fun resetPerConnectionShellState() {
        pendingRuntimeStatusRequestId = null
        pendingDeviceStatusRequestId = null
        pendingScenarioRegistryRequestId = null
        pendingPlaybackArtifactsRequestId = null
        pendingPlaybackMetadataRequestId = null
        registerAckScenarioQuerySent = false
        sawRegisterAck = false
    }

    internal fun snapshotDebugCommandPendingIds(): CommandDiagnosticsRequestIds =
        CommandDiagnosticsRequestIds(
            runtimeStatus = pendingRuntimeStatusRequestId.orEmpty(),
            deviceStatus = pendingDeviceStatusRequestId.orEmpty(),
            scenarioRegistry = pendingScenarioRegistryRequestId.orEmpty(),
            playbackArtifacts = pendingPlaybackArtifactsRequestId.orEmpty(),
            playbackMetadata = pendingPlaybackMetadataRequestId.orEmpty(),
        )

    internal fun applyCommandResultDiagnostics(
        base: AndroidTerminalViewState,
        response: Control.ConnectResponse,
        pending: CommandDiagnosticsRequestIds,
    ): AndroidTerminalViewState =
        if (response.payloadCase == Control.ConnectResponse.PayloadCase.COMMAND_RESULT) {
            mergeShellCommandResultStateIfDataPresent(base, response.commandResult, pending)
        } else {
            base
        }

    internal fun mergeShellCommandResultStateIfDataPresent(
        base: AndroidTerminalViewState,
        result: Control.CommandResult,
        pending: CommandDiagnosticsRequestIds,
    ): AndroidTerminalViewState {
        val data = commandResultDataMap(result)
        return if (data.isEmpty()) {
            base
        } else {
            mergeShellCommandResultStateForTitle(
                base,
                diagnosticsTitleForCommandResult(result, pending),
                data,
            )
        }
    }

    internal fun mergeShellCommandResultStateForTitle(
        base: AndroidTerminalViewState,
        title: String,
        data: Map<String, String>,
    ): AndroidTerminalViewState {
        var out = base
        when (title) {
            "scenario_registry" -> {
                val intents = applicationIntentsFromDiagnostics(data)
                val selected =
                    if (base.selectedApplicationIntent.trim() in intents) {
                        base.selectedApplicationIntent.trim()
                    } else {
                        intents.first()
                    }
                out =
                    out.copy(
                        availableApplicationIntents = intents.toList(),
                        selectedApplicationIntent = selected,
                    )
                pendingScenarioRegistryRequestId = null
            }
            "list_playback_artifacts" -> {
                val first = firstPlaybackArtifactId(data)
                if (first.isNotEmpty()) {
                    out = out.copy(playbackArtifactIdText = first)
                }
                pendingPlaybackArtifactsRequestId = null
            }
            "runtime_status" -> {
                pendingRuntimeStatusRequestId = null
            }
            "device_status" -> {
                pendingDeviceStatusRequestId = null
            }
            "playback_metadata" -> {
                pendingPlaybackMetadataRequestId = null
            }
            else -> {}
        }
        return out
    }

    internal suspend fun sendHeartbeatTracked(session: AndroidControlSession) {
        session.sendHeartbeat()
        recordOutboundHeartbeat()
    }

    internal suspend fun sendSensorTelemetryTracked(session: AndroidControlSession) {
        if (!session.sendSensorTelemetry()) return
        recordOutboundSensor()
    }

    private fun recordOutboundHeartbeat() {
        val now = dependencies.nowMillis()
        mutableState.update {
            val next =
                it.copy(
                    outboundHeartbeatCount = it.outboundHeartbeatCount + 1,
                    lastOutboundHeartbeatUnixMs = now,
                )
            next.copy(diagnosticsText = formatDiagnostics(parser.parse(next.endpointText), next.connectionState, next))
        }
    }

    private fun recordOutboundSensor() {
        val now = dependencies.nowMillis()
        mutableState.update {
            val next =
                it.copy(
                    outboundSensorSendCount = it.outboundSensorSendCount + 1,
                    lastOutboundSensorUnixMs = now,
                )
            next.copy(diagnosticsText = formatDiagnostics(parser.parse(next.endpointText), next.connectionState, next))
        }
    }

    internal fun formatDiagnostics(
        endpoint: EndpointResolution?,
        state: ConnectionState,
        handshakeSource: AndroidTerminalViewState? = null,
    ): String {
        val capabilitySnapshot = runCatching { dependencies.capabilityProbe.current() }.getOrNull()
        return formatTerminalDiagnostics(
            chrome = chrome,
            endpoint = endpoint,
            state = state,
            networkState = runCatching { dependencies.networkStateProvider.current() }.getOrNull(),
            fireOsDeviceInfo = runCatching { dependencies.fireOsDeviceInfoProvider.current() }.getOrNull(),
            capabilitySnapshot = capabilitySnapshot,
            controlStatus = session?.status,
            permissions = permissionEducation(capabilitySnapshot),
            mediaSupport = mediaSupport(),
            handshakeSource = handshakeSource,
        )
    }

    internal fun immersiveStickyForFullscreen(enabled: Boolean): Boolean =
        if (!enabled) {
            false
        } else {
            runCatching { dependencies.terminalSettings.immersiveStickyEnabled() }.getOrDefault(true)
        }

    private fun initialState(): AndroidTerminalViewState {
        val lastEndpoint = runCatching { dependencies.terminalSettings.lastManualEndpoint() }.getOrDefault("")
        val keepAwakeEnabled = runCatching { dependencies.terminalSettings.keepAwakeEnabled() }.getOrDefault(false)
        val fullscreenEnabled = runCatching { dependencies.terminalSettings.fullscreenEnabled() }.getOrDefault(false)
        val immersiveStickyEnabled = runCatching { dependencies.terminalSettings.immersiveStickyEnabled() }.getOrDefault(true)
        val brightDisplayEnabled = runCatching { dependencies.terminalSettings.brightDisplayEnabled() }.getOrDefault(false)
        if (keepAwakeEnabled) {
            runCatching { dependencies.keepAwakeController.setKeepAwake(true) }
        }
        if (fullscreenEnabled) {
            runCatching {
                dependencies.fullscreenController.setFullscreen(true, immersiveStickyEnabled)
            }
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
        val basis = AndroidTerminalViewState(
            endpointText = lastEndpoint,
            connectionState = state,
            lastError = if (state == ConnectionState.InvalidEndpoint) "Enter a host:port or http(s) URL." else null,
            diagnosticsText = "",
            localKeepAwakeEnabled = keepAwakeEnabled,
            localFullscreenEnabled = fullscreenEnabled,
            localImmersiveStickyEnabled = immersiveStickyEnabled,
            localBrightDisplayEnabled = brightDisplayEnabled,
            permissionEducation = permissionEducation(),
            mediaSupport = mediaSupport(),
        )
        return basis.copy(diagnosticsText = formatDiagnostics(resolved, state, basis))
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

}
