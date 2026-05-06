package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

interface AndroidControlClient {
    suspend fun connect(endpoint: EndpointResolution)
    suspend fun send(request: Control.ConnectRequest)
    suspend fun close()
}
