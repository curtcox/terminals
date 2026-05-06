package com.curtcox.terminals.android.capabilities

data class PermissionCapabilityState(
    val microphoneGranted: Boolean = false,
    val cameraGranted: Boolean = false,
    val notificationsGranted: Boolean = false,
)

class PermissionCapabilityMonitor(
    private var state: PermissionCapabilityState = PermissionCapabilityState(),
) {
    fun current(): PermissionCapabilityState = state

    fun update(next: PermissionCapabilityState): PermissionCapabilityState {
        state = next
        return state
    }
}
