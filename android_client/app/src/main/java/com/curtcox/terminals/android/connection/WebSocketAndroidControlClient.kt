package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

class WebSocketAndroidControlClient : AndroidControlClient {
    override suspend fun connect(endpoint: EndpointResolution) {
        error("WebSocket fallback is planned for the connection phase.")
    }

    override suspend fun send(request: Control.ConnectRequest) {
        error("WebSocket fallback is planned for the connection phase.")
    }

    override suspend fun close() = Unit
}
