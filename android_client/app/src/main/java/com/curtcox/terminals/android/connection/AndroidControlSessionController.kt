package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.curtcox.terminals.android.util.Clock

data class ControlSessionStatus(
    val connected: Boolean = false,
    val endpoint: EndpointResolution? = null,
    val lastError: String? = null,
    val lastCapabilityGeneration: Long = 0,
)

interface AndroidControlSession {
    val status: ControlSessionStatus
    suspend fun connect(endpoint: EndpointResolution)
    suspend fun sendHeartbeat()
    suspend fun sendSensorTelemetry()
    suspend fun sendUiAction(action: ServerDrivenAction)
    suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean
    suspend fun rebaselineCapabilitiesAfterStaleGeneration()
    suspend fun close()
}

class AndroidControlSessionController(
    private val deviceId: String,
    private val clientVersion: String,
    private val client: AndroidControlClient,
    private val capabilities: AndroidCapabilitySession,
    private val builders: ProtocolBuilders = ProtocolBuilders(),
    private val clock: Clock,
) : AndroidControlSession {
    override var status: ControlSessionStatus = ControlSessionStatus()
        private set

    override suspend fun connect(endpoint: EndpointResolution) {
        try {
            client.connect(endpoint)
            sendHelloAndSnapshot()
            status = ControlSessionStatus(
                connected = true,
                endpoint = endpoint,
                lastCapabilityGeneration = status.lastCapabilityGeneration,
            )
        } catch (error: Throwable) {
            status = ControlSessionStatus(
                connected = false,
                endpoint = endpoint,
                lastError = error.message ?: error::class.java.simpleName,
                lastCapabilityGeneration = status.lastCapabilityGeneration,
            )
            client.close()
            throw error
        }
    }

    override suspend fun sendHeartbeat() {
        client.send(builders.heartbeat(deviceId, clock.nowMillis()))
    }

    override suspend fun sendSensorTelemetry() {
        val request =
            builders.sensorTelemetryFromCapabilities(
                deviceId,
                capabilities.lastRegisteredCapabilities,
                clock.nowMillis(),
            ) ?: return
        client.send(request)
    }

    override suspend fun sendUiAction(action: ServerDrivenAction) {
        client.send(builders.uiAction(deviceId, action))
    }

    override suspend fun sendCapabilityDeltaIfChanged(reason: String): Boolean {
        val report = capabilities.deltaIfChanged(reason) ?: return false
        client.send(builders.capabilityDelta(deviceId, report.generation, report.capabilities, report.reason))
        status = status.copy(lastCapabilityGeneration = report.generation)
        return true
    }

    override suspend fun rebaselineCapabilitiesAfterStaleGeneration() {
        val report = capabilities.rebaselineAfterStaleGeneration()
        client.send(builders.capabilitySnapshot(deviceId, report.generation, report.capabilities))
        status = status.copy(lastCapabilityGeneration = report.generation)
    }

    override suspend fun close() {
        client.close()
        status = status.copy(connected = false)
    }

    private suspend fun sendHelloAndSnapshot() {
        val report = capabilities.snapshot()
        client.send(
            builders.hello(
                deviceId = deviceId,
                identity = report.capabilities.identity,
                clientVersion = clientVersion,
            ),
        )
        client.send(builders.capabilitySnapshot(deviceId, report.generation, report.capabilities))
        status = status.copy(lastCapabilityGeneration = report.generation)
    }
}
