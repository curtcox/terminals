package com.curtcox.terminals.android.connection

data class EndpointResolution(
    val host: String,
    val port: Int,
    val secure: Boolean = false,
    val path: String = "",
    val carrier: CarrierPreference = CarrierPreference.WebSocket,
) {
    val displayName: String
        get() =
            when (carrier) {
                CarrierPreference.Grpc ->
                    "${if (secure) "grpcs" else "grpc"}://$host:$port"
                CarrierPreference.WebSocket ->
                    "${if (secure) "https" else "http"}://$host:$port$path"
            }
}
