package com.curtcox.terminals.android.ui

data class RendererPolicy(
    val showUnsupportedFallback: Boolean = true,
    val unsupportedText: String = "Unsupported terminal widget",
) {
    companion object {
        fun default() = RendererPolicy()
    }
}
