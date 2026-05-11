package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Test
import terminals.control.v1.Control

class CommandResultDiagnosticsTest {
    @Test
    fun applicationIntentsFromDiagnosticsKeepsDefaultFirstAndSortsData() {
        val intents =
            applicationIntentsFromDiagnostics(
                mapOf(
                    "zoo" to "x",
                    "alpha" to "y",
                    "terminal" to "ignored",
                ),
            )
        assertEquals(listOf("terminal", "alpha", "zoo"), intents)
    }

    @Test
    fun applicationIntentsFromDiagnosticsTrimsKeys() {
        val intents =
            applicationIntentsFromDiagnostics(
                mapOf("  beta  " to "v"),
            )
        assertEquals(listOf("terminal", "beta"), intents)
    }

    @Test
    fun firstPlaybackArtifactIdUsesSortedKeysAndPipeSplit() {
        val id =
            firstPlaybackArtifactId(
                mapOf(
                    "b" to "ignored|tail",
                    "a" to "first-id|rest",
                ),
            )
        assertEquals("first-id", id)
    }

    @Test
    fun diagnosticsTitleUsesPendingScenarioRegistryId() {
        val title =
            diagnosticsTitleForCommandResult(
                Control.CommandResult.newBuilder().setRequestId("rid-1").build(),
                CommandDiagnosticsRequestIds(scenarioRegistry = "rid-1"),
            )
        assertEquals("scenario_registry", title)
    }

    @Test
    fun diagnosticsTitleFallsBackToNotification() {
        val title =
            diagnosticsTitleForCommandResult(
                Control.CommandResult.newBuilder()
                    .setRequestId("unknown")
                    .setNotification("System query: list_playback_artifacts")
                    .build(),
                CommandDiagnosticsRequestIds(),
            )
        assertEquals("list_playback_artifacts", title)
    }

    @Test
    fun commandResultDataMapPrefersTypedData() {
        val result =
            Control.CommandResult.newBuilder()
                .putData("legacy", "1")
                .addTypedData(
                    Control.CommandResultDataEntry.newBuilder()
                        .setKey("k")
                        .setValue(Control.CommandTypedValue.newBuilder().setStringValue("v").build()),
                )
                .build()
        assertEquals(mapOf("k" to "v"), commandResultDataMap(result))
    }
}
