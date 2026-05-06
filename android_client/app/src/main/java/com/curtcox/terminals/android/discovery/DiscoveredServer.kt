package com.curtcox.terminals.android.discovery

data class DiscoveredServer(
    val name: String,
    val host: String,
    val port: Int,
    val lastSeenMillis: Long,
    val grpcEndpoint: String = "",
    val webSocketEndpoint: String = "",
    val httpEndpoint: String = "",
    val carrierPriority: List<String> = emptyList(),
    val metadata: Map<String, String> = emptyMap(),
)
