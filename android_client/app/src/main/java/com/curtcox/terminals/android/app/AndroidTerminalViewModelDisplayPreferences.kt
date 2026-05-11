package com.curtcox.terminals.android.app

fun AndroidTerminalViewModel.setKeepAwake(enabled: Boolean) {
    runCatching {
        dependencies.keepAwakeController.setKeepAwake(enabled)
    }.onFailure { error ->
        mutableState.update {
            it.copy(lastError = error.message ?: error::class.java.simpleName)
        }
    }
}

fun AndroidTerminalViewModel.setLocalKeepAwake(enabled: Boolean) {
    runCatching {
        dependencies.keepAwakeController.setKeepAwake(enabled)
    }.onSuccess {
        dependencies.terminalSettings.setKeepAwakeEnabled(enabled)
        mutableState.update { st ->
            val next = st.copy(localKeepAwakeEnabled = enabled)
            next.copy(
                diagnosticsText =
                    formatDiagnostics(
                        parser.parse(next.endpointText),
                        next.connectionState,
                        next,
                    ),
            )
        }
    }.onFailure { error ->
        mutableState.update {
            it.copy(
                lastError = error.message ?: error::class.java.simpleName,
                localKeepAwakeEnabled = dependencies.terminalSettings.keepAwakeEnabled(),
            )
        }
    }
}

fun AndroidTerminalViewModel.setFullscreen(enabled: Boolean) {
    val sticky = immersiveStickyForFullscreen(enabled)
    runCatching {
        dependencies.fullscreenController.setFullscreen(enabled, sticky)
    }.onFailure { error ->
        mutableState.update {
            it.copy(lastError = error.message ?: error::class.java.simpleName)
        }
    }
}

fun AndroidTerminalViewModel.setLocalFullscreen(enabled: Boolean) {
    val sticky = immersiveStickyForFullscreen(enabled)
    runCatching {
        dependencies.fullscreenController.setFullscreen(enabled, sticky)
    }.onSuccess {
        dependencies.terminalSettings.setFullscreenEnabled(enabled)
        mutableState.update { st ->
            val next = st.copy(localFullscreenEnabled = enabled)
            next.copy(
                diagnosticsText =
                    formatDiagnostics(
                        parser.parse(next.endpointText),
                        next.connectionState,
                        next,
                    ),
            )
        }
    }.onFailure { error ->
        mutableState.update {
            it.copy(
                lastError = error.message ?: error::class.java.simpleName,
                localFullscreenEnabled = dependencies.terminalSettings.fullscreenEnabled(),
            )
        }
    }
}

fun AndroidTerminalViewModel.setLocalImmersiveSticky(enabled: Boolean) {
    dependencies.terminalSettings.setImmersiveStickyEnabled(enabled)
    mutableState.update {
        val next = it.copy(localImmersiveStickyEnabled = enabled)
        var updated =
            next.copy(
                diagnosticsText =
                    formatDiagnostics(
                        parser.parse(next.endpointText),
                        next.connectionState,
                        next,
                    ),
            )
        if (next.localFullscreenEnabled) {
            runCatching {
                dependencies.fullscreenController.setFullscreen(true, enabled)
            }.onFailure { error ->
                updated = updated.copy(lastError = error.message ?: error::class.java.simpleName)
            }
        }
        updated
    }
}

fun AndroidTerminalViewModel.setBrightness(value: Double) {
    runCatching {
        dependencies.brightnessController.setBrightness(value)
    }.onFailure { error ->
        mutableState.update {
            it.copy(lastError = error.message ?: error::class.java.simpleName)
        }
    }
}

fun AndroidTerminalViewModel.setLocalBrightDisplay(enabled: Boolean) {
    runCatching {
        dependencies.brightnessController.setBrightness(if (enabled) 1.0 else 0.5)
    }.onSuccess {
        dependencies.terminalSettings.setBrightDisplayEnabled(enabled)
        mutableState.update { st ->
            val next = st.copy(localBrightDisplayEnabled = enabled)
            next.copy(
                diagnosticsText =
                    formatDiagnostics(
                        parser.parse(next.endpointText),
                        next.connectionState,
                        next,
                    ),
            )
        }
    }.onFailure { error ->
        mutableState.update {
            it.copy(
                lastError = error.message ?: error::class.java.simpleName,
                localBrightDisplayEnabled = dependencies.terminalSettings.brightDisplayEnabled(),
            )
        }
    }
}
