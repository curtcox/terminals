package com.curtcox.terminals.android.app

import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.EndpointResolution
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

internal fun AndroidTerminalViewModel.startHeartbeat(connectedSession: AndroidControlSession) {
    stopHeartbeat()
    val intervalMs =
        when {
            effectiveHeartbeatMillis > 0 -> effectiveHeartbeatMillis
            else -> dependencies.heartbeatIntervalMillis
        }
    if (intervalMs <= 0 || !appInForeground) return
    heartbeatJob =
        viewModelScope.launch {
            while (true) {
                delay(intervalMs)
                runCatching {
                    sendHeartbeatTracked(connectedSession)
                }.onFailure { error ->
                    handleControlLoss(connectedSession, error)
                    return@launch
                }
            }
        }
}

internal fun AndroidTerminalViewModel.startSensorTelemetry(connectedSession: AndroidControlSession) {
    stopSensorTelemetry()
    val intervalMs = dependencies.sensorTelemetryIntervalMillis
    if (intervalMs <= 0 || !appInForeground) return
    sensorTelemetryJob =
        viewModelScope.launch {
            while (true) {
                delay(intervalMs)
                runCatching {
                    sendSensorTelemetryTracked(connectedSession)
                }.onFailure { error ->
                    handleControlLoss(connectedSession, error)
                    return@launch
                }
            }
        }
}

internal fun AndroidTerminalViewModel.stopHeartbeat() {
    heartbeatJob?.cancel()
    heartbeatJob = null
}

internal fun AndroidTerminalViewModel.stopSensorTelemetry() {
    sensorTelemetryJob?.cancel()
    sensorTelemetryJob = null
}

internal fun AndroidTerminalViewModel.startCapabilityMonitor(connectedSession: AndroidControlSession) {
    stopCapabilityMonitor()
    val intervalMs = dependencies.capabilityMonitorIntervalMillis
    if (intervalMs <= 0 || !appInForeground) return
    capabilityMonitorJob =
        viewModelScope.launch {
            while (true) {
                delay(intervalMs)
                runCatching {
                    connectedSession.sendCapabilityDeltaIfChanged("runtime_monitor_poll")
                }.onSuccess { sent ->
                    if (sent) {
                        mutableState.update {
                            it.copy(diagnosticsText = "${it.diagnosticsText}\nlast_capability_delta=runtime_monitor_poll")
                        }
                    }
                }.onFailure { error ->
                    handleControlLoss(connectedSession, error)
                    return@launch
                }
            }
        }
}

internal fun AndroidTerminalViewModel.stopCapabilityMonitor() {
    capabilityMonitorJob?.cancel()
    capabilityMonitorJob = null
}

internal fun AndroidTerminalViewModel.stopConnect() {
    connectJob?.cancel()
    connectJob = null
}

internal fun AndroidTerminalViewModel.stopReconnect() {
    reconnectJob?.cancel()
    reconnectJob = null
}

internal fun AndroidTerminalViewModel.handleControlLoss(failedSession: AndroidControlSession, error: Throwable) {
    stopHeartbeat()
    stopSensorTelemetry()
    stopCapabilityMonitor()
    if (session !== failedSession) return
    session = null
    sawRegisterAck = false
    registerAckScenarioQuerySent = false
    val endpoint = parser.parse(mutableState.value.endpointText)
    val message = error.message ?: error::class.java.simpleName
    mutableState.update {
        val next =
            it.copy(
                connectionState = if (endpoint == null) ConnectionState.Disconnected else ConnectionState.Connecting,
                lastError = message,
                applicationLaunchQueuedIntent = null,
            )
        next.copy(
            diagnosticsText =
                formatDiagnostics(endpoint, next.connectionState, next) +
                    "\nreconnect_pending=${endpoint != null}",
        )
    }
    viewModelScope.launch { failedSession.close() }
    if (endpoint != null) {
        startReconnect(endpoint, message)
    }
}

internal fun AndroidTerminalViewModel.startReconnect(
    endpoint: EndpointResolution,
    errorContext: String,
    reconnectCause: String = errorContext,
) {
    stopReconnect()
    reconnectJob =
        viewModelScope.launch {
            var lastError = errorContext
            for (attempt in 1..dependencies.maxReconnectAttempts) {
                delay(dependencies.reconnectPolicy.delayForAttempt(attempt))
                mutableState.update {
                    val basis =
                        it.copy(
                            connectionState = ConnectionState.Connecting,
                            lastError = lastError,
                            reconnectAttempt = attempt,
                        )
                    basis.copy(
                        diagnosticsText =
                            formatDiagnostics(endpoint, ConnectionState.Connecting, basis) +
                                "\nreconnect_attempt=$attempt\nreconnect_cause=$reconnectCause",
                    )
                }
                val nextSession = dependencies.sessionFactory(responseSink)
                nextSession.setPrivacyMode(mutableState.value.privacyModeEnabled)
                val connected =
                    runCatching {
                        nextSession.connect(endpoint)
                        mutableState.update {
                            it.copy(
                                outboundHeartbeatCount = 0,
                                lastOutboundHeartbeatUnixMs = 0L,
                                outboundSensorSendCount = 0,
                                lastOutboundSensorUnixMs = 0L,
                                streamReadySendCount = 0,
                                inboundConnectResponseCount = 0,
                            )
                        }
                        if (appInForeground) {
                            sendHeartbeatTracked(nextSession)
                            sendSensorTelemetryTracked(nextSession)
                        }
                        session = nextSession
                        startHeartbeat(nextSession)
                        startSensorTelemetry(nextSession)
                        startCapabilityMonitor(nextSession)
                    }.onFailure { error ->
                        lastError = error.message ?: error::class.java.simpleName
                        nextSession.close()
                    }.isSuccess
                if (connected) {
                    mutableState.update {
                        it.copy(
                            connectionState = ConnectionState.Connected,
                            lastError = null,
                            reconnectAttempt = 0,
                            diagnosticsText =
                                formatDiagnostics(endpoint, ConnectionState.Connected, it) +
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
                    diagnosticsText =
                        formatDiagnostics(endpoint, ConnectionState.ReadyToConnect, basis) +
                            "\nreconnect_exhausted=${dependencies.maxReconnectAttempts}\nreconnect_cause=$reconnectCause",
                )
            }
            reconnectExhausted = true
            reconnectJob = null
        }
}
