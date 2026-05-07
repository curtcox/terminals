package com.curtcox.terminals.android.diagnostics

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context

fun interface DiagnosticClipboard {
    fun copy(text: String)

    companion object {
        fun none(): DiagnosticClipboard = DiagnosticClipboard {}
    }
}

class ContextDiagnosticClipboard(context: Context) : DiagnosticClipboard {
    private val appContext = context.applicationContext

    override fun copy(text: String) {
        val clipboard = appContext.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("Terminals diagnostics", text))
    }
}
