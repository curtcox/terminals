package com.curtcox.terminals.android.connection

data class TransportDiagnostics(
    val endpoint: EndpointResolution?,
    val carrier: CarrierPreference?,
    val lastError: String?,
    val reconnectAttempt: Int = 0,
)
