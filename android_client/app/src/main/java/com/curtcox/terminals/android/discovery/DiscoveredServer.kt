package com.curtcox.terminals.android.discovery

data class DiscoveredServer(
    val name: String,
    val host: String,
    val port: Int,
    val lastSeenMillis: Long,
)
