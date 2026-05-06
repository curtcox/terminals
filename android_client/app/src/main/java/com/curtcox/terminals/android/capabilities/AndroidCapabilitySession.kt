package com.curtcox.terminals.android.capabilities

import terminals.capabilities.v1.Capabilities

data class AndroidCapabilityReport(
    val generation: Long,
    val capabilities: Capabilities.DeviceCapabilities,
    val reason: String,
    val fullSnapshot: Boolean,
)

class AndroidCapabilitySession(
    private val deviceId: String,
    private val probe: AndroidCapabilityProbe,
) {
    private var generation: Long = 0
    private var lastCapabilities: Capabilities.DeviceCapabilities? = null

    fun snapshot(reason: String = "initial"): AndroidCapabilityReport {
        generation += 1
        val capabilities = buildCapabilities(deviceId, probe.current())
        lastCapabilities = capabilities
        return AndroidCapabilityReport(generation, capabilities, reason, fullSnapshot = true)
    }

    fun deltaIfChanged(reason: String): AndroidCapabilityReport? {
        val capabilities = buildCapabilities(deviceId, probe.current())
        if (capabilities == lastCapabilities) {
            return null
        }
        generation += 1
        lastCapabilities = capabilities
        return AndroidCapabilityReport(generation, capabilities, reason, fullSnapshot = false)
    }

    fun rebaselineAfterStaleGeneration(): AndroidCapabilityReport = snapshot("stale-generation")

    companion object {
        fun buildCapabilities(
            deviceId: String,
            input: AndroidCapabilitySnapshotInput,
        ): Capabilities.DeviceCapabilities {
            val screen = input.screenMetrics.toProto()
            val builder = Capabilities.DeviceCapabilities.newBuilder()
                .setDeviceId(deviceId)
                .setIdentity(input.identity)
                .setScreen(screen)
                .addDisplays(
                    Capabilities.DisplayCapability.newBuilder()
                        .setDisplayId("primary")
                        .setDisplayName("Primary display")
                        .setPrimary(true)
                        .setScreen(screen),
                )
                .setKeyboard(
                    Capabilities.KeyboardCapability.newBuilder()
                        .setPhysical(input.hardware.physicalKeyboard),
                )
                .setPointer(
                    Capabilities.PointerCapability.newBuilder()
                        .setType(input.hardware.pointerType)
                        .setHover(input.hardware.pointerHover),
                )
                .setTouch(
                    Capabilities.TouchCapability.newBuilder()
                        .setSupported(input.hardware.touchSupported)
                        .setMaxPoints(input.hardware.maxTouchPoints),
                )
                .setSensors(
                    Capabilities.SensorCapability.newBuilder()
                        .setAccelerometer(input.hardware.accelerometer)
                        .setGyroscope(input.hardware.gyroscope)
                        .setCompass(input.hardware.compass)
                        .setAmbientLight(input.hardware.ambientLight)
                        .setProximity(input.hardware.proximity)
                        .setGps(input.hardware.gps),
                )
                .setConnectivity(
                    Capabilities.ConnectivityCapability.newBuilder()
                        .setBluetoothVersion(input.hardware.bluetoothVersion)
                        .setWifiSignalStrength(input.hardware.wifiSignalStrength)
                        .setUsbHost(input.hardware.usbHost)
                        .setUsbPorts(input.hardware.usbPorts)
                        .setNfc(input.hardware.nfc),
                )
                .setBattery(
                    Capabilities.BatteryCapability.newBuilder()
                        .setLevel(input.power.batteryLevel.coerceIn(0f, 1f))
                        .setCharging(input.power.charging),
                )
                .setHaptics(
                    Capabilities.HapticCapability.newBuilder()
                        .setSupported(input.hardware.haptics)
                        .setVibration(input.hardware.haptics),
                )

            if (input.hardware.audioOutput) {
                builder.setSpeakers(
                    Capabilities.AudioOutputCapability.newBuilder()
                        .setChannels(2)
                        .addSampleRates(44100)
                        .addSampleRates(48000),
                )
            }
            if (input.hardware.microphone && input.permissions.microphoneGranted) {
                builder.setMicrophone(
                    Capabilities.AudioInputCapability.newBuilder()
                        .setChannels(1)
                        .addSampleRates(16000)
                        .addSampleRates(48000),
                )
            }
            if (input.permissions.cameraGranted && (input.hardware.frontCamera || input.hardware.backCamera)) {
                val camera = Capabilities.CameraCapability.newBuilder()
                if (input.hardware.frontCamera) {
                    camera.setFront(cameraLens())
                }
                if (input.hardware.backCamera) {
                    camera.setBack(cameraLens())
                }
                builder.setCamera(camera)
            }
            return builder.build()
        }

        private fun AndroidScreenMetrics.toProto(): Capabilities.ScreenCapability =
            Capabilities.ScreenCapability.newBuilder()
                .setWidth(widthPx)
                .setHeight(heightPx)
                .setDensity(density.toDouble())
                .setTouch(true)
                .setOrientation(orientation)
                .setFullscreenSupported(true)
                .setMultiWindowSupported(true)
                .setSafeArea(
                    Capabilities.Insets.newBuilder()
                        .setLeft(safeArea.leftPx)
                        .setTop(safeArea.topPx)
                        .setRight(safeArea.rightPx)
                        .setBottom(safeArea.bottomPx),
                )
                .build()

        private fun cameraLens(): Capabilities.CameraLens =
            Capabilities.CameraLens.newBuilder()
                .setWidth(1280)
                .setHeight(720)
                .setFps(30)
                .build()
    }
}
