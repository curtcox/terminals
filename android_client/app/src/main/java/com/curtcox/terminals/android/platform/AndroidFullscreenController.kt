package com.curtcox.terminals.android.platform

fun interface AndroidFullscreenController {
    /**
     * @param immersiveStickyWhenEnabled When [enabled] is false, callers may pass `false`; the implementation ignores it.
     */
    fun setFullscreen(enabled: Boolean, immersiveStickyWhenEnabled: Boolean)
}
