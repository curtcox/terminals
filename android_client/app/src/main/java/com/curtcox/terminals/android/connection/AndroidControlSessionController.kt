package com.curtcox.terminals.android.connection

import com.curtcox.terminals.android.capabilities.AndroidCapabilitySession
import com.curtcox.terminals.android.ui.ServerDrivenAction
import com.curtcox.terminals.android.util.Clock
import terminals.capabilities.v1.Capabilities
import terminals.diagnostics.v1.Diagnostics

data class ControlSessionStatus(
    val connected: Boolean = false,
    val endpoint: EndpointResolution? = null,
    val lastError: String? = null,
    val lastCapabilityGeneration: Long = 0,
)

interface AndroidControlSession {
    val status: ControlSessionStatus
    val lastRegisteredCapabilities: Capabilities.DeviceCapabilities?
    fun setPrivacyMode(enabled: Boolean)
    suspend fun connect(endpoint: EndpointResolution)
    suspend fun sendHeartbeat()
    /** @return true when a sensor payload was sent (Flutter `buildSensorTelemetryRequest` non-null). */
    suspend fun sendSensorTelemetry(): Boolean
    suspend fun sendUiAction(action: ServerDrivenAction)
    suspend fun sendStreamReady(streamId: String)
    suspend fun sendKeyText(text: String)
    suspend fun sendBugReport(report: Diagnostics.BugReport)
    suspend fun sendSystemCommand(requestId: String, intent: String)
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

    override val lastRegisteredCapabilities: Capabilities.DeviceCapabilities?
        get() = capabilities.lastRegisteredCapabilities

    override fun setPrivacyMode(enabled: Boolean) {
        capabilities.setPrivacyMode(enabled)
    }

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

    override suspend fun sendSensorTelemetry(): Boolean {
        val request =
            builders.sensorTelemetryFromCapabilities(
                deviceId,
                capabilities.lastRegisteredCapabilities,
                clock.nowMillis(),
            ) ?: return false
        client.send(request)
        return true
    }

    override suspend fun sendUiAction(action: ServerDrivenAction) {
        client.send(builders.uiAction(deviceId, action))
    }

    override suspend fun sendStreamReady(streamId: String) {
        val trimmed = streamId.trim()
        if (trimmed.isEmpty()) return
        client.send(builders.streamReady(trimmed))
    }

    override suspend fun sendKeyText(text: String) {
        if (deviceId.isEmpty() || text.isEmpty()) return
        client.send(builders.keyInput(deviceId, text))
    }

    override suspend fun sendBugReport(report: Diagnostics.BugReport) {
        client.send(builders.bugReport(report))
    }

    override suspend fun sendSystemCommand(requestId: String, intent: String) {
        if (requestId.isEmpty() || intent.isEmpty()) return
        client.send(builders.systemCommand(requestId, intent))
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
