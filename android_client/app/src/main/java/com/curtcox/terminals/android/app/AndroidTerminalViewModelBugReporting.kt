package com.curtcox.terminals.android.app

import android.os.Build
import androidx.lifecycle.viewModelScope
import com.curtcox.terminals.android.connection.AndroidControlSession
import com.curtcox.terminals.android.diagnostics.AndroidBugReportActions
import com.curtcox.terminals.android.diagnostics.AndroidBugReportBuilder
import com.curtcox.terminals.android.ui.ServerDrivenAction
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import terminals.diagnostics.v1.Diagnostics
import java.util.Locale
import java.util.TimeZone

internal fun AndroidTerminalViewModel.submitBugReportFromServerDrivenAction(action: ServerDrivenAction) {
    val subject = resolveBugReportSubject(action)
    val report =
        buildShellBugReport(
            description = "Filed from on-device bug report button",
            source = Diagnostics.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
            subjectDeviceId = subject,
            extraHints =
            mapOf(
                "component_id" to action.componentId,
                "action" to action.action,
            ),
        )
    queueOrSendBugReport(report)
}

internal fun AndroidTerminalViewModel.resolveBugReportSubject(action: ServerDrivenAction): String {
    val fromAction = action.action
        .takeIf { it.startsWith(AndroidBugReportActions.PREFIX) }
        ?.removePrefix(AndroidBugReportActions.PREFIX)
        ?.removePrefix(":")
        ?.trim()
        ?.takeIf { it.isNotEmpty() }
    return fromAction
        ?: action.value.trim().takeIf { it.isNotEmpty() }
        ?: dependencies.deviceId
}

internal fun AndroidTerminalViewModel.buildShellBugReport(
    description: String,
    source: Diagnostics.BugReportSource,
    subjectDeviceId: String,
    extraHints: Map<String, String>,
): Diagnostics.BugReport {
    val s = mutableState.value
    val screenshotPng =
        runCatching { dependencies.bugReportScreenshotCapture() }.getOrNull()?.takeIf {
            it.isNotEmpty()
        }
    return AndroidBugReportBuilder.build(
        description = description,
        source = source,
        reporterDeviceId = dependencies.deviceId,
        subjectDeviceId = subjectDeviceId,
        extraSourceHints = extraHints,
        clock = bugReportClock,
        buildMetadata = dependencies.buildMetadata,
        serverRoot = s.serverRoot,
        connectionState = s.connectionState,
        lastServerHeartbeatUnixMs = s.lastServerHeartbeatUnixMs,
        registeredCapabilities = session?.lastRegisteredCapabilities,
        localeTag = Locale.getDefault().toLanguageTag(),
        timezoneId = TimeZone.getDefault().id,
        osVersion = "${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})",
        reconnectAttempt = s.reconnectAttempt,
        lastStatus = s.lastControlResponseActivity,
        screenshotPng = screenshotPng,
    )
}

internal fun AndroidTerminalViewModel.queueOrSendBugReport(report: Diagnostics.BugReport) {
    viewModelScope.launch {
        val currentSession = session
        val connected = mutableState.value.connectionState == ConnectionState.Connected
        if (currentSession != null && connected) {
            runCatching { currentSession.sendBugReport(report) }
                .onSuccess {
                    val word = report.sourceHintsMap["bug_token_word"].orEmpty()
                    mutableState.update { st ->
                        st.copy(
                            lastBugReportSubmitStatus = "Sent bug report ${report.reportId} (word=$word).",
                            lastError = null,
                        )
                    }
                }
                .onFailure { e ->
                    mutableState.update {
                        it.copy(
                            lastBugReportSubmitStatus =
                            "Bug report send failed: ${e.message ?: e.javaClass.simpleName}",
                        )
                    }
                }
        } else {
            bugReportQueue.addLast(report)
            val word = report.sourceHintsMap["bug_token_word"].orEmpty()
            mutableState.update {
                it.copy(
                    lastBugReportSubmitStatus = "Queued bug report (word=$word) until connected.",
                )
            }
        }
    }
}

internal suspend fun AndroidTerminalViewModel.flushQueuedBugReports(target: AndroidControlSession) {
    val pending = bugReportQueue.size
    if (pending == 0) return
    var sent = 0
    var lastWord = ""
    var lastFailure: String? = null
    while (bugReportQueue.isNotEmpty()) {
        val report = bugReportQueue.removeFirst()
        runCatching { target.sendBugReport(report) }
            .onSuccess {
                sent++
                lastWord = report.sourceHintsMap["bug_token_word"].orEmpty()
            }
            .onFailure { e ->
                lastFailure = e.message ?: e::class.java.simpleName
            }
    }
    val status =
        when {
            sent == pending && sent == 1 -> "Sent queued bug report (word=$lastWord)."
            sent == pending && sent > 1 -> "Sent $sent queued bug reports (last word=$lastWord)."
            sent > 0 ->
                "Sent $sent of $pending queued bug reports (last word=$lastWord). Remainder failed: $lastFailure"
            else -> "Queued bug reports failed to send: $lastFailure"
        }
    mutableState.update { it.copy(lastBugReportSubmitStatus = status) }
}
