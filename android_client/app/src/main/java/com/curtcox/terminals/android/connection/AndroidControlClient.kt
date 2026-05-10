package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

interface AndroidControlClient {
    suspend fun connect(endpoint: EndpointResolution)
    suspend fun send(request: Control.ConnectRequest)
    suspend fun close()
}

interface AndroidControlResponseSink {
    suspend fun onResponse(response: Control.ConnectResponse)

    /**
     * Invoked when the inbound control stream ends unexpectedly or the server half-closes gRPC.
     * Implementations typically mirror heartbeat/send failures by tearing down the session and reconnecting.
     */
    suspend fun onTransportTerminated(error: Throwable?) {
        // Default: older tests and no-op sinks ignore transport failure.
    }
}
