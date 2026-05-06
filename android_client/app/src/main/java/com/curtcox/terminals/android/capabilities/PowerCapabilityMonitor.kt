package com.curtcox.terminals.android.capabilities

data class PowerCapabilityState(
    val batteryLevel: Float = 0f,
    val charging: Boolean = false,
    val keepAwakeSupported: Boolean = true,
)

class PowerCapabilityMonitor(
    private var state: PowerCapabilityState = PowerCapabilityState(),
) {
    fun current(): PowerCapabilityState = state

    fun update(next: PowerCapabilityState): PowerCapabilityState {
        state = next
        return state
    }
}
