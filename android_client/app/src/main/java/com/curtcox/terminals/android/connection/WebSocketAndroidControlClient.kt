package com.curtcox.terminals.android.connection

class WebSocketAndroidControlClient : AndroidControlClient {
    override suspend fun connect(endpoint: EndpointResolution) {
        error("WebSocket fallback is planned for the connection phase.")
    }

    override suspend fun close() = Unit
}
