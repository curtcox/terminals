package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

/**
 * Merges typed_data entries into a string map, else falls back to [Control.CommandResult.getDataMap].
 * Matches Flutter `commandResultDataMap`.
 */
fun commandResultDataMap(result: Control.CommandResult): Map<String, String> {
    val typed = LinkedHashMap<String, String>()
    for (entry in result.typedDataList) {
        val key = entry.key.trim()
        if (key.isNotEmpty() && entry.hasValue()) {
            commandTypedValueToString(entry.value)?.let { typed[key] = it }
        }
    }
    if (typed.isNotEmpty()) return typed
    return result.dataMap
}

private fun commandTypedValueToString(value: Control.CommandTypedValue): String? =
    when (value.kindCase) {
        Control.CommandTypedValue.KindCase.STRING_VALUE -> value.stringValue
        Control.CommandTypedValue.KindCase.INT64_VALUE -> value.int64Value.toString()
        Control.CommandTypedValue.KindCase.BOOL_VALUE -> if (value.boolValue) "true" else "false"
        Control.CommandTypedValue.KindCase.DOUBLE_VALUE -> value.doubleValue.toString()
        Control.CommandTypedValue.KindCase.STRING_LIST_VALUE ->
            value.stringListValue.valuesList.joinToString(",")
        Control.CommandTypedValue.KindCase.KIND_NOT_SET -> null
        else -> null
    }

private fun notificationDiagnosticsTitle(notification: String): String =
    when (notification) {
        "System query: runtime_status" -> "runtime_status"
        "System query: device_status" -> "device_status"
        "System query: scenario_registry" -> "scenario_registry"
        "System query: list_playback_artifacts" -> "list_playback_artifacts"
        "Playback metadata ready" -> "playback_metadata"
        else -> ""
    }

/** Matches Flutter `diagnosticsTitleForCommandResult`. */
fun diagnosticsTitleForCommandResult(
    result: Control.CommandResult,
    pending: CommandDiagnosticsRequestIds,
): String {
    val id = result.requestId
    val byId =
        if (id.isEmpty()) {
            null
        } else {
            when (id) {
                pending.runtimeStatus -> "runtime_status"
                pending.deviceStatus -> "device_status"
                pending.scenarioRegistry -> "scenario_registry"
                pending.playbackArtifacts -> "list_playback_artifacts"
                pending.playbackMetadata -> "playback_metadata"
                else -> null
            }
        }
    return byId ?: notificationDiagnosticsTitle(result.notification)
}

/** Matches Flutter `applicationIntentsFromDiagnostics`. */
fun applicationIntentsFromDiagnostics(
    data: Map<String, String>,
    defaultIntent: String = "terminal",
): List<String> {
    val fallback = defaultIntent.trim()
    val discovered =
        data.keys
            .map { it.trim() }
            .filter { it.isNotEmpty() && it != fallback }
            .toSortedSet()
    return buildList {
        if (fallback.isNotEmpty()) add(fallback)
        addAll(discovered)
    }
}

/** Matches Flutter `firstPlaybackArtifactID`. */
fun firstPlaybackArtifactId(data: Map<String, String>): String {
    for (key in data.keys.sorted()) {
        val first = data[key]?.split('|')?.firstOrNull()?.trim().orEmpty()
        if (first.isNotEmpty()) return first
    }
    return ""
}
