package com.curtcox.terminals.android.connection

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import terminals.control.v1.Control

/**
 * Picks [GrpcAndroidControlClient] vs [WebSocketAndroidControlClient] from [EndpointResolution.carrier]
 * so manual endpoints and discovery metadata can target either transport.
 */
class CarrierSelectingAndroidControlClient(
    private val deviceId: String,
    private val websocketResumeTokenStore: TransportResumeTokenStore,
    private val responseSink: AndroidControlResponseSink?,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.IO),
) : AndroidControlClient {
    private var delegate: AndroidControlClient? = null

    override suspend fun connect(endpoint: EndpointResolution) {
        close()
        val next =
            when (endpoint.carrier) {
                CarrierPreference.Grpc -> GrpcAndroidControlClient(responseSink, scope)
                CarrierPreference.WebSocket ->
                    WebSocketAndroidControlClient(
                        deviceId = deviceId,
                        resumeTokenStore = websocketResumeTokenStore,
                        responseSink = responseSink,
                        scope = scope,
                    )
            }
        delegate = next
        next.connect(endpoint)
    }

    override suspend fun send(request: Control.ConnectRequest) {
        delegate?.send(request) ?: error("control transport is not connected")
    }

    override suspend fun close() {
        delegate?.close()
        delegate = null
    }
}
