package com.curtcox.terminals.android.platform

import android.os.Build
import android.view.View
import android.view.Window
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat

class WindowAndroidFullscreenController(
    private val window: Window,
) : AndroidFullscreenController {
    override fun setFullscreen(enabled: Boolean) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            WindowCompat.setDecorFitsSystemWindows(window, !enabled)
            val insetsController = WindowInsetsControllerCompat(window, window.decorView)
            insetsController.systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
            if (enabled) {
                insetsController.hide(WindowInsetsCompat.Type.systemBars())
            } else {
                insetsController.show(WindowInsetsCompat.Type.systemBars())
            }
        } else {
            @Suppress("DEPRECATION")
            run {
                window.decorView.systemUiVisibility = legacySystemUiVisibility(enabled)
            }
        }
    }
}

@Suppress("DEPRECATION")
internal fun legacySystemUiVisibility(enabled: Boolean): Int =
    if (enabled) {
        View.SYSTEM_UI_FLAG_FULLSCREEN or
            View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY or
            View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_LAYOUT_STABLE
    } else {
        View.SYSTEM_UI_FLAG_LAYOUT_STABLE
    }
