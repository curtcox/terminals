package com.curtcox.terminals.android.ui

data class RendererPolicy(
    val showUnsupportedFallback: Boolean = true,
    val unsupportedText: String = "Unsupported UI node",
) {
    companion object {
        fun default() = RendererPolicy()
    }
}
