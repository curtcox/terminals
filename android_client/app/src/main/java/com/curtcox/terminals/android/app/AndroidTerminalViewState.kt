package com.curtcox.terminals.android.app

import terminals.ui.v1.Ui

data class AndroidTerminalViewState(
    val endpointText: String = "",
    val connectionState: ConnectionState = ConnectionState.Disconnected,
    val lastError: String? = null,
    val diagnosticsText: String = "",
    val serverRoot: Ui.Node? = null,
    val lastNotificationTitle: String? = null,
    val lastNotificationBody: String? = null,
    val lastTransition: String? = null,
)

enum class ConnectionState {
    Disconnected,
    InvalidEndpoint,
    ReadyToConnect,
    Connecting,
    Connected,
}
