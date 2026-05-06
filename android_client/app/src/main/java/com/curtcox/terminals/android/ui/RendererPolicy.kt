package com.curtcox.terminals.android.ui

data class RendererPolicy(
    val showUnsupportedFallback: Boolean = true,
) {
    companion object {
        fun default() = RendererPolicy()
    }
}
