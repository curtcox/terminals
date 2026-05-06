package com.curtcox.terminals.android.connection

data class EndpointResolution(
    val host: String,
    val port: Int,
    val secure: Boolean = false,
    val path: String = "",
) {
    val displayName: String
        get() = "${if (secure) "https" else "http"}://$host:$port$path"
}
