package com.curtcox.terminals.android.capabilities

import terminals.capabilities.v1.Capabilities

interface AndroidCapabilityProbe {
    fun current(): AndroidCapabilitySnapshotInput
}

data class AndroidCapabilitySnapshotInput(
    val identity: Capabilities.DeviceIdentity,
    val screenMetrics: AndroidScreenMetrics,
    val permissions: PermissionCapabilityState = PermissionCapabilityState(),
    val hardware: AndroidHardwareCapabilities = AndroidHardwareCapabilities(),
    val power: PowerCapabilityState = PowerCapabilityState(),
)

data class AndroidHardwareCapabilities(
    val touchSupported: Boolean = true,
    val maxTouchPoints: Int = 1,
    val physicalKeyboard: Boolean = false,
    val pointerType: String = "touch",
    val pointerHover: Boolean = false,
    val audioOutput: Boolean = true,
    val microphone: Boolean = false,
    val frontCamera: Boolean = false,
    val backCamera: Boolean = false,
    val accelerometer: Boolean = false,
    val gyroscope: Boolean = false,
    val compass: Boolean = false,
    val ambientLight: Boolean = false,
    val proximity: Boolean = false,
    val gps: Boolean = false,
    val bluetoothVersion: String = "",
    val wifiSignalStrength: Boolean = true,
    val usbHost: Boolean = false,
    val usbPorts: Int = 0,
    val nfc: Boolean = false,
    val haptics: Boolean = false,
)
