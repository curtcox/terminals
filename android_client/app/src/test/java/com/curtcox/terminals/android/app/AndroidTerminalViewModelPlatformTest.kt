package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.platform.AndroidBrightnessController
import com.curtcox.terminals.android.platform.AndroidFullscreenController
import com.curtcox.terminals.android.platform.AndroidKeepAwakeController
import com.curtcox.terminals.android.platform.AndroidTerminalSettings
import kotlinx.coroutines.ExperimentalCoroutinesApi
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class AndroidTerminalViewModelPlatformTest : AndroidTerminalViewModelTestBase() {

    @Test
    fun keepAwakeDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(keepAwakeController = AndroidKeepAwakeController { calls.add(it) }),
        )

        viewModel.setKeepAwake(true)
        viewModel.setKeepAwake(false)

        assertEquals(listOf(true, false), calls)
    }

    @Test
    fun localKeepAwakeSettingIsRestoredAndApplied() {
        val calls = mutableListOf<Boolean>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialKeepAwakeEnabled = true),
                keepAwakeController = AndroidKeepAwakeController { calls.add(it) },
            ),
        )

        assertEquals(true, viewModel.state.value.localKeepAwakeEnabled)
        assertEquals(listOf(true), calls)
    }

    @Test
    fun localKeepAwakeTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Boolean>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                keepAwakeController = AndroidKeepAwakeController { calls.add(it) },
            ),
        )

        viewModel.setLocalKeepAwake(true)

        assertEquals(true, settings.keepAwakeEnabled())
        assertEquals(true, viewModel.state.value.localKeepAwakeEnabled)
        assertEquals(listOf(true), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_keep_awake=true"))
    }

    @Test
    fun localFullscreenSettingIsRestoredAndApplied() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialFullscreenEnabled = true),
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        assertEquals(true, viewModel.state.value.localFullscreenEnabled)
        assertEquals(listOf(true to true), calls)
    }

    @Test
    fun localFullscreenRestoresWithImmersiveStickyDisabled() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(
                    initialFullscreenEnabled = true,
                    initialImmersiveStickyEnabled = false,
                ),
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        assertEquals(listOf(true to false), calls)
        assertEquals(false, viewModel.state.value.localImmersiveStickyEnabled)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_immersive_sticky=false"))
    }

    @Test
    fun localFullscreenTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        viewModel.setLocalFullscreen(true)

        assertEquals(true, settings.fullscreenEnabled())
        assertEquals(true, viewModel.state.value.localFullscreenEnabled)
        assertEquals(listOf(true to true), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_fullscreen=true"))
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_immersive_sticky=true"))
    }

    @Test
    fun localImmersiveStickyToggleReappliesWhenFullscreenOn() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        viewModel.setLocalFullscreen(true)
        calls.clear()
        viewModel.setLocalImmersiveSticky(false)

        assertEquals(false, settings.immersiveStickyEnabled())
        assertEquals(listOf(true to false), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_immersive_sticky=false"))
    }

    @Test
    fun fullscreenDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        viewModel.setFullscreen(true)
        viewModel.setFullscreen(false)

        assertEquals(listOf(true to true, false to false), calls)
    }

    @Test
    fun serverFullscreenUsesImmersiveStickyTerminalSetting() {
        val calls = mutableListOf<Pair<Boolean, Boolean>>()
        val settings = AndroidTerminalSettings.inMemory(initialImmersiveStickyEnabled = false)
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                fullscreenController = AndroidFullscreenController { enabled, sticky ->
                    calls.add(enabled to sticky)
                },
            ),
        )

        viewModel.setFullscreen(true)

        assertEquals(listOf(true to false), calls)
    }

    @Test
    fun brightnessDelegatesToPlatformAdapter() {
        val calls = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(brightnessController = AndroidBrightnessController { calls.add(it) }),
        )

        viewModel.setBrightness(0.25)
        viewModel.setBrightness(1.0)

        assertEquals(listOf(0.25, 1.0), calls)
    }

    @Test
    fun localBrightDisplaySettingIsRestoredAndApplied() {
        val calls = mutableListOf<Double>()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = AndroidTerminalSettings.inMemory(initialBrightDisplayEnabled = true),
                brightnessController = AndroidBrightnessController { calls.add(it) },
            ),
        )

        assertEquals(true, viewModel.state.value.localBrightDisplayEnabled)
        assertEquals(listOf(1.0), calls)
    }

    @Test
    fun localBrightDisplayTogglePersistsAndUpdatesDiagnostics() {
        val calls = mutableListOf<Double>()
        val settings = AndroidTerminalSettings.inMemory()
        val viewModel = AndroidTerminalViewModel(
            AndroidClientDependencies(
                terminalSettings = settings,
                brightnessController = AndroidBrightnessController { calls.add(it) },
            ),
        )

        viewModel.setLocalBrightDisplay(true)

        assertEquals(true, settings.brightDisplayEnabled())
        assertEquals(true, viewModel.state.value.localBrightDisplayEnabled)
        assertEquals(listOf(1.0), calls)
        assertTrue(viewModel.state.value.diagnosticsText.contains("local_bright_display=true"))
    }
}
