package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.ui.ServerDrivenAction
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.io.v1.Io

const val AndroidWireProtocolVersion: Int = 1

class ProtocolBuilders(
    private val protocolVersion: Int = AndroidWireProtocolVersion,
) {
    fun transportHello(
        desiredDeviceId: String,
        resumeToken: String = "",
        supportedCarriers: List<Control.CarrierKind> = listOf(Control.CarrierKind.CARRIER_KIND_GRPC),
    ): Control.WireEnvelope =
        Control.WireEnvelope.newBuilder()
            .setProtocolVersion(protocolVersion)
            .setTransportHello(
                Control.TransportHello.newBuilder()
                    .setProtocolVersion(protocolVersion)
                    .setDesiredDeviceId(desiredDeviceId)
                    .setResumeToken(resumeToken)
                    .addAllSupportedCarriers(supportedCarriers),
            )
            .build()

    fun hello(
        deviceId: String,
        identity: Capabilities.DeviceIdentity,
        clientVersion: String,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setHello(
                Control.Hello.newBuilder()
                    .setDeviceId(deviceId)
                    .setIdentity(identity)
                    .setClientVersion(clientVersion),
            )
            .build()

    fun capabilitySnapshot(
        deviceId: String,
        generation: Long,
        capabilities: Capabilities.DeviceCapabilities,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setCapabilitySnapshot(
                Control.CapabilitySnapshot.newBuilder()
                    .setDeviceId(deviceId)
                    .setGeneration(generation)
                    .setCapabilities(capabilities),
            )
            .build()

    fun capabilityDelta(
        deviceId: String,
        generation: Long,
        capabilities: Capabilities.DeviceCapabilities,
        reason: String,
    ): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setCapabilityDelta(
                Control.CapabilityDelta.newBuilder()
                    .setDeviceId(deviceId)
                    .setGeneration(generation)
                    .setCapabilities(capabilities)
                    .setReason(reason),
            )
            .build()

    fun heartbeat(deviceId: String, unixMs: Long): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setHeartbeat(
                Control.Heartbeat.newBuilder()
                    .setDeviceId(deviceId)
                    .setUnixMs(unixMs),
            )
            .build()

    /**
     * Matches Flutter `buildSensorTelemetryRequest`: only battery fields from the last registered
     * capabilities snapshot; returns null when there is nothing to send.
     */
    fun sensorTelemetryFromCapabilities(
        deviceId: String,
        capabilities: Capabilities.DeviceCapabilities?,
        unixMs: Long,
    ): Control.ConnectRequest? {
        if (deviceId.isEmpty() || capabilities == null || !capabilities.hasBattery()) {
            return null
        }
        val values =
            mapOf(
                "battery.level" to capabilities.battery.level.toDouble(),
                "battery.charging" to if (capabilities.battery.charging) 1.0 else 0.0,
            )
        return Control.ConnectRequest.newBuilder()
            .setSensor(
                Io.SensorData.newBuilder()
                    .setDeviceId(deviceId)
                    .setUnixMs(unixMs)
                    .putAllValues(values),
            )
            .build()
    }

    fun uiAction(deviceId: String, action: ServerDrivenAction): Control.ConnectRequest =
        Control.ConnectRequest.newBuilder()
            .setInput(
                Io.InputEvent.newBuilder()
                    .setDeviceId(deviceId)
                    .setUiAction(
                        Io.UIAction.newBuilder()
                            .setComponentId(action.componentId)
                            .setAction(action.action)
                            .setValue(action.value),
                    ),
            )
            .build()
}
