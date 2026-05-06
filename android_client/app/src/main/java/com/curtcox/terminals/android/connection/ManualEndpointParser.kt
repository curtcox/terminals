package com.curtcox.terminals.android.connection

import java.net.URI

class ManualEndpointParser {
    fun parse(raw: String): EndpointResolution? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null

        val uriText = if (trimmed.contains("://")) trimmed else "http://$trimmed"
        val uri = runCatching { URI(uriText) }.getOrNull() ?: return null
        val scheme = uri.scheme?.lowercase() ?: return null
        if (scheme != "http" && scheme != "https" && scheme != "ws" && scheme != "wss") return null

        val host = uri.host ?: return null
        val port = when {
            uri.port in 1..65535 -> uri.port
            scheme == "https" || scheme == "wss" -> 443
            else -> 80
        }
        val path = uri.rawPath.takeUnless { it.isNullOrBlank() || it == "/" } ?: ""
        return EndpointResolution(host = host, port = port, secure = scheme == "https" || scheme == "wss", path = path)
    }
}
