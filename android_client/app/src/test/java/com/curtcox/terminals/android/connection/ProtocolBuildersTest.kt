package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.ui.ServerDrivenAction
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control

class ProtocolBuildersTest {
    private val builders = ProtocolBuilders()

    @Test
    fun transportHelloUsesGeneratedEnvelopeAndCarrierList() {
        val envelope = builders.transportHello(
            desiredDeviceId = "fire-tablet",
            resumeToken = "resume-1",
            supportedCarriers = listOf(
                Control.CarrierKind.CARRIER_KIND_GRPC,
                Control.CarrierKind.CARRIER_KIND_WEBSOCKET,
            ),
        )

        assertEquals(AndroidWireProtocolVersion, envelope.protocolVersion)
        assertTrue(envelope.hasTransportHello())
        assertEquals("fire-tablet", envelope.transportHello.desiredDeviceId)
        assertEquals("resume-1", envelope.transportHello.resumeToken)
        assertEquals(AndroidWireProtocolVersion, envelope.transportHello.protocolVersion)
        assertEquals(
            listOf(Control.CarrierKind.CARRIER_KIND_GRPC, Control.CarrierKind.CARRIER_KIND_WEBSOCKET),
            envelope.transportHello.supportedCarriersList,
        )
    }

    @Test
    fun helloIncludesDeviceIdentityAndClientVersion() {
        val identity = Capabilities.DeviceIdentity.newBuilder()
            .setDeviceName("Kitchen Fire")
            .setDeviceType("tablet")
            .setPlatform("android")
            .build()

        val request = builders.hello("device-1", identity, "0.1.0")

        assertTrue(request.hasHello())
        assertEquals("device-1", request.hello.deviceId)
        assertEquals(identity, request.hello.identity)
        assertEquals("0.1.0", request.hello.clientVersion)
    }

    @Test
    fun capabilitySnapshotCarriesGenerationAndCapabilities() {
        val capabilities = Capabilities.DeviceCapabilities.newBuilder()
            .setDeviceId("device-1")
            .setIdentity(Capabilities.DeviceIdentity.newBuilder().setPlatform("android"))
            .build()

        val request = builders.capabilitySnapshot("device-1", 7, capabilities)

        assertTrue(request.hasCapabilitySnapshot())
        assertEquals("device-1", request.capabilitySnapshot.deviceId)
        assertEquals(7, request.capabilitySnapshot.generation)
        assertEquals(capabilities, request.capabilitySnapshot.capabilities)
    }

    @Test
    fun capabilityDeltaCarriesReason() {
        val capabilities = Capabilities.DeviceCapabilities.newBuilder()
            .setDeviceId("device-1")
            .build()

        val request = builders.capabilityDelta("device-1", 8, capabilities, "orientation")

        assertTrue(request.hasCapabilityDelta())
        assertEquals(8, request.capabilityDelta.generation)
        assertEquals("orientation", request.capabilityDelta.reason)
    }

    @Test
    fun heartbeatUsesControlHeartbeatPayload() {
        val request = builders.heartbeat("device-1", 1234)

        assertTrue(request.hasHeartbeat())
        assertEquals("device-1", request.heartbeat.deviceId)
        assertEquals(1234, request.heartbeat.unixMs)
    }

    @Test
    fun uiActionMapsRendererActionToInputEvent() {
        val request = builders.uiAction(
            deviceId = "device-1",
            action = ServerDrivenAction(
                componentId = "start",
                action = "tap",
            ),
        )

        assertTrue(request.hasInput())
        assertEquals("device-1", request.input.deviceId)
        assertTrue(request.input.hasUiAction())
        assertEquals("start", request.input.uiAction.componentId)
        assertEquals("tap", request.input.uiAction.action)
        assertEquals("", request.input.uiAction.value)
    }

    @Test
    fun sensorTelemetryFromCapabilitiesMatchesFlutterBatteryKeys() {
        val caps =
            Capabilities.DeviceCapabilities.newBuilder()
                .setDeviceId("device-1")
                .setBattery(
                    Capabilities.BatteryCapability.newBuilder()
                        .setLevel(0.77f)
                        .setCharging(true),
                )
                .build()

        val request = builders.sensorTelemetryFromCapabilities("device-1", caps, 9_001L)

        assertTrue(request!!.hasSensor())
        assertEquals("device-1", request.sensor.deviceId)
        assertEquals(9_001L, request.sensor.unixMs)
        assertEquals(0.77, request.sensor.valuesMap["battery.level"]!!, 0.0001)
        assertEquals(1.0, request.sensor.valuesMap["battery.charging"]!!, 0.0001)
    }

    @Test
    fun sensorTelemetryFromCapabilitiesReturnsNullWithoutBattery() {
        val caps =
            Capabilities.DeviceCapabilities.newBuilder()
                .setDeviceId("device-1")
                .build()

        assertNull(builders.sensorTelemetryFromCapabilities("device-1", caps, 1L))
    }

    @Test
    fun sensorTelemetryFromCapabilitiesReturnsNullForEmptyDeviceId() {
        val caps =
            Capabilities.DeviceCapabilities.newBuilder()
                .setBattery(Capabilities.BatteryCapability.newBuilder().setLevel(0.5f))
                .build()

        assertNull(builders.sensorTelemetryFromCapabilities("", caps, 1L))
    }
}
