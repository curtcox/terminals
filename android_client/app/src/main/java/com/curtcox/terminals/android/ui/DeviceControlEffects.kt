package com.curtcox.terminals.android.ui

data class DeviceControlEffects(
    val setKeepAwake: (Boolean) -> Unit = {},
    val setFullscreen: (Boolean) -> Unit = {},
    val setBrightness: (Double) -> Unit = {},
) {
    companion object {
        fun none(): DeviceControlEffects = DeviceControlEffects()
    }
}
