package com.curtcox.terminals.android.smoke

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.curtcox.terminals.android.MainActivity
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Exercises the real launcher activity (production ViewModel factory + context-backed deps).
 * Compose-only smoke tests inject a fake session factory and do not cover app startup.
 */
@RunWith(AndroidJUnit4::class)
class MainActivityLaunchSmokeTest {
    @get:Rule
    val rule = createAndroidComposeRule<MainActivity>()

    @Test
    fun mainActivityLaunchesAndShowsManualConnectChrome() {
        rule.onNodeWithTag("terminal-endpoint-field").assertIsDisplayed()
        rule.onNodeWithTag("terminal-connect-button").assertIsDisplayed()
        rule.onNodeWithTag("terminal-discovery-start-button").assertIsDisplayed()
        rule.onNodeWithTag("terminal-live-media-status").assertIsDisplayed()
        rule.onNodeWithTag("terminal-last-server-activity").assertIsDisplayed()
    }
}
