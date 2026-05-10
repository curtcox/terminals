package com.curtcox.terminals.android.platform

import android.view.View
import org.junit.Assert.assertEquals
import org.junit.Test

class WindowAndroidFullscreenControllerTest {
    @Test
    fun legacySystemUiVisibilityEnabledUsesImmersiveStickySystemBarsFlags() {
        val expected = View.SYSTEM_UI_FLAG_FULLSCREEN or
            View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY or
            View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_LAYOUT_STABLE

        assertEquals(expected, legacySystemUiVisibility(enabled = true, immersiveStickyWhenEnabled = true))
    }

    @Test
    fun legacySystemUiVisibilityEnabledNonStickyUsesImmersiveFlag() {
        val expected = View.SYSTEM_UI_FLAG_FULLSCREEN or
            View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_IMMERSIVE or
            View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_LAYOUT_STABLE

        assertEquals(expected, legacySystemUiVisibility(enabled = true, immersiveStickyWhenEnabled = false))
    }

    @Test
    fun legacySystemUiVisibilityDisabledUsesStableLayoutFlagOnly() {
        assertEquals(
            View.SYSTEM_UI_FLAG_LAYOUT_STABLE,
            legacySystemUiVisibility(enabled = false, immersiveStickyWhenEnabled = false),
        )
    }
}
