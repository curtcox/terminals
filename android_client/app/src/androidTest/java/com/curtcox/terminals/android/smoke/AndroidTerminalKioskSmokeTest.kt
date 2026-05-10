package com.curtcox.terminals.android.smoke

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.curtcox.terminals.android.MainActivity
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Phase 7 kiosk chrome on a real [MainActivity] (production deps). Gradle filter: `--tests '*Kiosk*'`.
 */
@RunWith(AndroidJUnit4::class)
class AndroidTerminalKioskSmokeTest {
    @get:Rule
    val rule = createAndroidComposeRule<MainActivity>()

    @Test
    fun localKeepAwakeToggleUpdatesChromeLabel() {
        rule.onNodeWithText("Keep awake off").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-keep-awake-button").performClick()
        rule.onNodeWithText("Keep awake on").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-keep-awake-button").performClick()
        rule.onNodeWithText("Keep awake off").assertIsDisplayed()
    }

    @Test
    fun localFullscreenToggleUpdatesChromeLabel() {
        rule.onNodeWithText("Fullscreen off").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-fullscreen-button").performClick()
        rule.onNodeWithText("Fullscreen on").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-fullscreen-button").performClick()
        rule.onNodeWithText("Fullscreen off").assertIsDisplayed()
    }

    @Test
    fun localBrightDisplayToggleUpdatesChromeLabel() {
        rule.onNodeWithText("Bright display off").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-bright-display-button").performClick()
        rule.onNodeWithText("Bright display on").assertIsDisplayed()
        rule.onNodeWithTag("terminal-local-bright-display-button").performClick()
        rule.onNodeWithText("Bright display off").assertIsDisplayed()
    }
}
