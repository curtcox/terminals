package com.curtcox.terminals.android.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlResponseSink
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.connection.ControlResponseDispatcher
import com.curtcox.terminals.android.connection.ManualEndpointParser
import com.curtcox.terminals.android.diagnostics.AndroidClientChrome
import com.curtcox.terminals.android.ui.ServerDrivenAction
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
            mutableState.update {
                val next = dispatcher.dispatch(it, response)
                val diagnostics = chrome.formatDiagnostics(parser.parse(next.endpointText), next.connectionState)
                next.copy(
                    diagnosticsText = if (rebaselineSent) {
                        "$diagnostics\nlast_capability_rebaseline=stale-generation"
                    } else {
                        diagnostics
                    },
                )
            }
        }
    }
    private var session: AndroidControlSession? = null
    private val mutableState = MutableStateFlow(
        AndroidTerminalViewState(diagnosticsText = chrome.formatDiagnostics(null, ConnectionState.Disconnected)),
    )

    val state: StateFlow<AndroidTerminalViewState> = mutableState

    fun updateEndpoint(text: String) {
        val resolved = parser.parse(text)
        mutableState.update {
            it.copy(
                endpointText = text,
                connectionState = if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect,
                lastError = if (resolved == null && text.isNotBlank()) "Enter a host:port or http(s) URL." else null,
                diagnosticsText = chrome.formatDiagnostics(resolved, if (resolved == null) ConnectionState.InvalidEndpoint else ConnectionState.ReadyToConnect),
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
                diagnosticsText = chrome.formatDiagnostics(resolved, ConnectionState.Connecting),
            )
        }
        viewModelScope.launch {
            runCatching {
                val nextSession = dependencies.sessionFactory(responseSink)
                session = nextSession
                nextSession.connect(resolved)
            }.onSuccess {
                mutableState.update {
                    it.copy(
                        connectionState = ConnectionState.Connected,
                        lastError = null,
                        diagnosticsText = chrome.formatDiagnostics(resolved, ConnectionState.Connected),
                    )
                }
            }.onFailure { error ->
                session = null
                mutableState.update {
                    val message = error.message ?: error::class.java.simpleName
                    it.copy(
                        connectionState = ConnectionState.ReadyToConnect,
                        lastError = message,
                        diagnosticsText = chrome.formatDiagnostics(resolved, ConnectionState.ReadyToConnect) +
                            "\nlast_error=$message",
                    )
                }
            }
        }
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

    fun setKeepAwake(enabled: Boolean) {
        runCatching {
            dependencies.keepAwakeController.setKeepAwake(enabled)
        }.onFailure { error ->
            mutableState.update {
                it.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
    }

    override fun onCleared() {
        val closingSession = session
        session = null
        viewModelScope.launch { closingSession?.close() }
        super.onCleared()
    }

    private fun Control.ConnectResponse.requiresCapabilityRebaseline(): Boolean {
        if (!hasError()) return false
        if (error.code != Control.ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) return false
        return error.message.contains("stale", ignoreCase = true) &&
            error.message.contains("generation", ignoreCase = true)
    }
}
