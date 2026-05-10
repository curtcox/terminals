package com.curtcox.terminals.android.connection

import java.net.URI

class ManualEndpointParser {
    fun parse(raw: String): EndpointResolution? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null

        val schemeLower = trimmed.substringBefore("://", missingDelimiterValue = "").lowercase()
        if (schemeLower == "grpc" || schemeLower == "grpcs") {
            val uri = runCatching { URI(trimmed) }.getOrNull() ?: return null
            val scheme = uri.scheme?.lowercase() ?: return null
            if (scheme != "grpc" && scheme != "grpcs") return null
            val host = uri.host ?: return null
            val port =
                when {
                    uri.port in 1..65535 -> uri.port
                    scheme == "grpcs" -> 443
                    else -> 50051
                }
            return EndpointResolution(
                host = host,
                port = port,
                secure = scheme == "grpcs",
                path = "",
                carrier = CarrierPreference.Grpc,
            )
        }

        val uriText = if (trimmed.contains("://")) trimmed else "http://$trimmed"
        val uri = runCatching { URI(uriText) }.getOrNull() ?: return null
        val scheme = uri.scheme?.lowercase() ?: return null
        if (scheme != "http" && scheme != "https" && scheme != "ws" && scheme != "wss") return null

        val host = uri.host ?: return null
        val port =
            when {
                uri.port in 1..65535 -> uri.port
                scheme == "https" || scheme == "wss" -> 443
                else -> 80
            }
        val path = uri.rawPath.takeUnless { p -> p.isNullOrBlank() || p == "/" }.orEmpty()
        return EndpointResolution(
            host = host,
            port = port,
            secure = scheme == "https" || scheme == "wss",
            path = path,
            carrier = CarrierPreference.WebSocket,
        )
    }
}
