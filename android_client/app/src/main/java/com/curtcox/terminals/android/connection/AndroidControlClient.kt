package com.curtcox.terminals.android.connection

interface AndroidControlClient {
    suspend fun connect(endpoint: EndpointResolution)
    suspend fun close()
}
