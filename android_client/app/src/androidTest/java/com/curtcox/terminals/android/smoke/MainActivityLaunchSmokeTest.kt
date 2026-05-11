package com.curtcox.terminals.android.smoke

import android.Manifest
import androidx.compose.ui.test.assert
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.rule.GrantPermissionRule
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
    @get:Rule(order = 0)
    val grantPermissions: GrantPermissionRule = GrantPermissionRule.grant(
        Manifest.permission.RECORD_AUDIO,
        Manifest.permission.CAMERA,
    )

    @get:Rule(order = 1)
    val rule = createAndroidComposeRule<MainActivity>()

    @Test
    fun mainActivityLaunchesAndShowsManualConnectChrome() {
        rule.onNodeWithTag("terminal-endpoint-field").assertIsDisplayed()
        rule.onNodeWithTag("terminal-connect-button").assertIsDisplayed()
        rule.onNodeWithTag("terminal-discovery-start-button").assertIsDisplayed()
        rule.onNodeWithTag("terminal-live-media-status").assertIsDisplayed()
        rule.onNodeWithTag("terminal-last-server-activity").assertIsDisplayed()
        rule.onNodeWithTag("terminal-privacy-toggle-button").assertIsDisplayed()
        rule.onNodeWithTag("terminal-report-bug-button").assertIsDisplayed()
    }

    @Test
    fun reportBugWhileOfflineShowsQueuedStatus() {
        rule.onNodeWithTag("terminal-report-bug-button").performClick()
        rule.onNodeWithTag("terminal-bug-report-status").assertIsDisplayed()
        rule.onNodeWithTag("terminal-bug-report-status").assert(hasText("Queued", substring = true))
    }

    @Test
    fun copyDiagnosticsFromMainActivityShowsCopiedStatus() {
        rule.onNodeWithTag("terminal-copy-diagnostics-button").performClick()
        rule.onNodeWithTag("terminal-diagnostics-copy-status").assertIsDisplayed()
        rule.onNodeWithTag("terminal-diagnostics-copy-status").assert(hasText("copied", substring = true))
    }
}
