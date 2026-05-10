package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.capabilities.AndroidCapabilityProbe
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.capabilities.AndroidCapabilitySnapshotInput
import com.curtcox.terminals.android.capabilities.AndroidScreenMetrics
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.curtcox.terminals.android.util.Clock
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import terminals.capabilities.v1.Capabilities
import terminals.control.v1.Control
import terminals.diagnostics.v1.Diagnostics

class AndroidControlSessionControllerTest {
    @Test
    fun connectSendsHelloAndCapabilitySnapshot() = runTest {
        val client = FakeControlClient()
        val controller = controller(client = client)
        val endpoint = EndpointResolution("10.0.0.8", 8080)

        controller.connect(endpoint)

        assertEquals(endpoint, client.connectedEndpoint)
        assertTrue(controller.status.connected)
        assertEquals(1, controller.status.lastCapabilityGeneration)
        assertEquals(2, client.sent.size)
        assertTrue(client.sent[0].hasHello())
        assertEquals("device-1", client.sent[0].hello.deviceId)
        assertEquals("0.1.0-test", client.sent[0].hello.clientVersion)
        assertTrue(client.sent[1].hasCapabilitySnapshot())
        assertEquals(1, client.sent[1].capabilitySnapshot.generation)
    }

    @Test
    fun heartbeatAndUiActionUseProtocolBuilders() = runTest {
        val client = FakeControlClient()
        val controller = controller(client = client)
        controller.connect(EndpointResolution("10.0.0.8", 8080))
        client.sent.clear()

        controller.sendHeartbeat()
        controller.sendSensorTelemetry()
        controller.sendUiAction(ServerDrivenAction(componentId = "start", action = "tap", value = "go"))
        controller.sendKeyText("x")

        assertTrue(client.sent[0].hasHeartbeat())
        assertEquals(4242, client.sent[0].heartbeat.unixMs)
        assertTrue(client.sent[1].hasSensor())
        assertEquals(4242, client.sent[1].sensor.unixMs)
        assertEquals(0.0, client.sent[1].sensor.valuesMap["battery.level"]!!, 0.001)
        assertEquals(0.0, client.sent[1].sensor.valuesMap["battery.charging"]!!, 0.001)
        assertTrue(client.sent[2].hasInput())
        assertEquals("start", client.sent[2].input.uiAction.componentId)
        assertEquals("tap", client.sent[2].input.uiAction.action)
        assertEquals("go", client.sent[2].input.uiAction.value)
        assertTrue(client.sent[3].hasInput())
        assertTrue(client.sent[3].input.hasKey())
        assertEquals("x", client.sent[3].input.key.text)
    }

    @Test
    fun capabilityDeltaIsOnlySentWhenProbeChanges() = runTest {
        val client = FakeControlClient()
        val probe = MutableProbe(baseInput())
        val controller = controller(client = client, probe = probe)

        controller.connect(EndpointResolution("terminal.local", 8080))
        assertFalse(controller.sendCapabilityDeltaIfChanged("unchanged"))

        probe.input = baseInput().copy(
            screenMetrics = AndroidScreenMetrics(widthPx = 800, heightPx = 1280, density = 2f, orientation = "portrait"),
        )

        assertTrue(controller.sendCapabilityDeltaIfChanged("orientation"))
        assertTrue(client.sent.last().hasCapabilityDelta())
        assertEquals(2, client.sent.last().capabilityDelta.generation)
        assertEquals("orientation", client.sent.last().capabilityDelta.reason)
    }

    @Test
    fun staleGenerationRebaselineSendsFreshSnapshot() = runTest {
        val client = FakeControlClient()
        val probe = MutableProbe(baseInput())
        val controller = controller(client = client, probe = probe)

        controller.connect(EndpointResolution("terminal.local", 8080))
        probe.input = baseInput().copy(
            screenMetrics = AndroidScreenMetrics(widthPx = 800, heightPx = 1280, density = 2f, orientation = "portrait"),
        )
        controller.rebaselineCapabilitiesAfterStaleGeneration()

        assertTrue(client.sent.last().hasCapabilitySnapshot())
        assertEquals(2, client.sent.last().capabilitySnapshot.generation)
        assertEquals("portrait", client.sent.last().capabilitySnapshot.capabilities.screen.orientation)
        assertEquals(2, controller.status.lastCapabilityGeneration)
    }

    @Test
    fun failedRebaselineLeavesPreviousGenerationInStatus() = runTest {
        val client = FakeControlClient(sendErrorOnCapabilitySnapshotAfterFirst = IllegalStateException("stream closed"))
        val controller = controller(client = client)

        controller.connect(EndpointResolution("terminal.local", 8080))
        val result = runCatching { controller.rebaselineCapabilitiesAfterStaleGeneration() }

        assertTrue(result.isFailure)
        assertEquals(1, controller.status.lastCapabilityGeneration)
    }

    @Test
    fun sendBugReportUsesProtocolBuilder() = runTest {
        val client = FakeControlClient()
        val controller = controller(client = client)
        val report = Diagnostics.BugReport.newBuilder().setReportId("br-9").setDescription("d").build()

        controller.connect(EndpointResolution("10.0.0.8", 8080))
        controller.sendBugReport(report)

        assertTrue(client.sent.last().hasBugReport())
        assertEquals("br-9", client.sent.last().bugReport.reportId)
    }

    @Test
    fun sendStreamReadyUsesProtocolBuilder() = runTest {
        val client = FakeControlClient()
        val controller = controller(client = client)
        controller.connect(EndpointResolution("10.0.0.8", 8080))
        client.sent.clear()

        controller.sendStreamReady("  stream-a  ")

        assertTrue(client.sent.single().hasStreamReady())
        assertEquals("stream-a", client.sent.single().streamReady.streamId)
    }

    @Test
    fun sendStreamReadyNoOpsWhenBlank() = runTest {
        val client = FakeControlClient()
        val controller = controller(client = client)
        controller.connect(EndpointResolution("10.0.0.8", 8080))
        client.sent.clear()

        controller.sendStreamReady("   ")

        assertTrue(client.sent.isEmpty())
    }

    @Test
    fun failedConnectClosesClientAndRecordsError() = runTest {
        val client = FakeControlClient(connectError = IllegalStateException("no route"))
        val controller = controller(client = client)

        runCatching {
            controller.connect(EndpointResolution("10.0.0.8", 8080))
        }

        assertFalse(controller.status.connected)
        assertEquals("no route", controller.status.lastError)
        assertTrue(client.closed)
    }

    private fun controller(
        client: FakeControlClient,
        probe: MutableProbe = MutableProbe(baseInput()),
    ): AndroidControlSessionController =
        AndroidControlSessionController(
            deviceId = "device-1",
            clientVersion = "0.1.0-test",
            client = client,
            capabilities = AndroidCapabilitySession("device-1", probe),
            clock = Clock { 4242 },
        )

    private class FakeControlClient(
        private val connectError: Throwable? = null,
        private val sendErrorOnCapabilitySnapshotAfterFirst: Throwable? = null,
    ) : AndroidControlClient {
        var connectedEndpoint: EndpointResolution? = null
        val sent = mutableListOf<Control.ConnectRequest>()
        var closed = false

        override suspend fun connect(endpoint: EndpointResolution) {
            connectError?.let { throw it }
            connectedEndpoint = endpoint
        }

        override suspend fun send(request: Control.ConnectRequest) {
            if (
                sendErrorOnCapabilitySnapshotAfterFirst != null &&
                request.hasCapabilitySnapshot() &&
                sent.any { it.hasCapabilitySnapshot() }
            ) {
                throw sendErrorOnCapabilitySnapshotAfterFirst
            }
            sent += request
        }

        override suspend fun close() {
            closed = true
        }
    }

    private class MutableProbe(
        var input: AndroidCapabilitySnapshotInput,
    ) : AndroidCapabilityProbe {
        override fun current(): AndroidCapabilitySnapshotInput = input
    }

    private companion object {
        val identity: Capabilities.DeviceIdentity = Capabilities.DeviceIdentity.newBuilder()
            .setDeviceName("Kitchen Fire")
            .setDeviceType("tablet")
            .setPlatform("android")
            .build()

        fun baseInput(): AndroidCapabilitySnapshotInput =
            AndroidCapabilitySnapshotInput(
                identity = identity,
                screenMetrics = AndroidScreenMetrics(
                    widthPx = 1280,
                    heightPx = 800,
                    density = 2f,
                    orientation = "landscape",
                ),
            )
    }
}
