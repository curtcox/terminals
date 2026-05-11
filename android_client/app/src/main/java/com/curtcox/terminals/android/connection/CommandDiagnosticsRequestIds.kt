package com.curtcox.terminals.android.connection

/** Pending debug request ids for [diagnosticsTitleForCommandResult] (Flutter `CommandDiagnosticsRequestIDs`). */
data class CommandDiagnosticsRequestIds(
    val runtimeStatus: String = "",
    val deviceStatus: String = "",
    val scenarioRegistry: String = "",
    val playbackArtifacts: String = "",
    val playbackMetadata: String = "",
)
