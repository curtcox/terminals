package com.curtcox.terminals.android.platform

import android.view.Window
import android.view.WindowManager

class WindowAndroidBrightnessController(
    private val window: Window,
) : AndroidBrightnessController {
    override fun setBrightness(value: Double) {
        val attributes = WindowManager.LayoutParams().also {
            it.copyFrom(window.attributes)
            it.screenBrightness = value.coerceIn(0.0, 1.0).toFloat()
        }
        window.attributes = attributes
    }
}
