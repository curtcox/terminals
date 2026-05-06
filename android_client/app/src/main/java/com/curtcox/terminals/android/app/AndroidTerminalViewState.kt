package com.curtcox.terminals.android.app

data class AndroidTerminalViewState(
    val endpointText: String = "",
    val connectionState: ConnectionState = ConnectionState.Disconnected,
    val lastError: String? = null,
    val diagnosticsText: String = "",
)

enum class ConnectionState {
    Disconnected,
    InvalidEndpoint,
    ReadyToConnect,
    Connecting,
}
