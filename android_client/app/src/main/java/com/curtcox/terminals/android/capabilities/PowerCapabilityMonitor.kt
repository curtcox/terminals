package com.curtcox.terminals.android.capabilities

/** Battery snapshot for capability probes (no scenario logic). */
data class PowerCapabilityState(
    val batteryLevel: Float = 0f,
    val charging: Boolean = false,
    val keepAwakeSupported: Boolean = true,
)
