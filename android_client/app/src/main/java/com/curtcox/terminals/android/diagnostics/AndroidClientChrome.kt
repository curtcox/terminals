package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.app.ConnectionState
import com.curtcox.terminals.android.connection.EndpointResolution

class AndroidClientChrome(
    private val buildMetadata: AndroidBuildMetadata,
) {
    fun formatDiagnostics(endpoint: EndpointResolution?, state: ConnectionState): String = buildString {
        appendLine("client=android-native")
        appendLine("version=${buildMetadata.versionName}")
        appendLine("build_sha=${buildMetadata.buildSha}")
        appendLine("build_date=${buildMetadata.buildDate}")
        appendLine("state=$state")
        appendLine("endpoint=${endpoint?.displayName ?: "none"}")
        appendLine("fire_os_target=minSdk25")
        appendLine("google_services=absent")
    }
}
