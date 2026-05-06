package com.curtcox.terminals.android.platform

import android.view.Window
import android.view.WindowManager

class WindowAndroidKeepAwakeController(
    private val window: Window,
) : AndroidKeepAwakeController {
    override fun setKeepAwake(enabled: Boolean) {
        if (enabled) {
            window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        } else {
            window.clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        }
    }
}
