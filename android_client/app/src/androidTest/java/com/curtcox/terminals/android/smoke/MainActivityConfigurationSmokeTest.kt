package com.curtcox.terminals.android.smoke

import android.content.pm.ActivityInfo
import android.content.res.Configuration
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.curtcox.terminals.android.MainActivity
import org.junit.After
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Exercises [MainActivity.onConfigurationChanged] wiring (orientation retained via manifest
 * `configChanges`). Gradle filter: `--tests '*MainActivityConfiguration*'`.
 */
@RunWith(AndroidJUnit4::class)
class MainActivityConfigurationSmokeTest {
    @get:Rule
    val rule = createAndroidComposeRule<MainActivity>()

    @After
    fun tearDown() {
        rule.activity.requestedOrientation = ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
    }

    @Test
    fun orientationChangeRunsConfigurationRefreshInChrome() {
        val activity = rule.activity
        val goLandscape = activity.resources.configuration.orientation == Configuration.ORIENTATION_PORTRAIT
        activity.requestedOrientation = if (goLandscape) {
            ActivityInfo.SCREEN_ORIENTATION_LANDSCAPE
        } else {
            ActivityInfo.SCREEN_ORIENTATION_PORTRAIT
        }

        rule.waitUntil(timeoutMillis = 10_000) {
            rule.onAllNodesWithText("last_permission_refresh=configuration", substring = true)
                .fetchSemanticsNodes()
                .isNotEmpty()
        }
        rule.onNodeWithText("last_permission_refresh=configuration", substring = true).assertExists()
    }
}
