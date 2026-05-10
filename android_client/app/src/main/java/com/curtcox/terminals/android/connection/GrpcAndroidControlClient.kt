package com.curtcox.terminals.android.connection

import io.grpc.ManagedChannel
import io.grpc.okhttp.OkHttpChannelBuilder
import io.grpc.stub.StreamObserver
import java.io.IOException
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import terminals.control.v1.Control
import terminals.control.v1.TerminalControlServiceGrpc

class GrpcAndroidControlClient(
    private val responseSink: AndroidControlResponseSink?,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.IO),
) : AndroidControlClient {
    private var channel: ManagedChannel? = null
    private var requestStream: StreamObserver<Control.ConnectRequest>? = null
    private val streamTerminationHandled = AtomicBoolean(false)

    @Volatile
    private var acceptStreamTerminationCallbacks: Boolean = false

    override suspend fun connect(endpoint: EndpointResolution) =
        withContext(Dispatchers.IO) {
            close()
            if (endpoint.carrier != CarrierPreference.Grpc) {
                throw IOException("gRPC client requires carrier GRPC, got ${endpoint.carrier}")
            }
            streamTerminationHandled.set(false)
            val ch =
                OkHttpChannelBuilder.forAddress(endpoint.host, endpoint.port)
                    .apply {
                        if (endpoint.secure) {
                            useTransportSecurity()
                        } else {
                            usePlaintext()
                        }
                    }
                    .build()
            channel = ch
            val stub = TerminalControlServiceGrpc.newStub(ch)
            val incoming =
                object : StreamObserver<Control.ConnectResponse> {
                    override fun onNext(value: Control.ConnectResponse) {
                        scope.launch {
                            responseSink?.onResponse(value)
                        }
                    }

                    override fun onError(t: Throwable) {
                        notifyStreamTerminated(t)
                    }

                    override fun onCompleted() {
                        notifyStreamTerminated(null)
                    }
                }
            requestStream = stub.connect(incoming)
            acceptStreamTerminationCallbacks = true
        }

    override suspend fun send(request: Control.ConnectRequest) =
        withContext(Dispatchers.IO) {
            val stream = requestStream ?: throw IOException("gRPC control stream is not connected")
            stream.onNext(request)
        }

    override suspend fun close() {
        withContext(Dispatchers.IO) {
            acceptStreamTerminationCallbacks = false
            runCatching { requestStream?.onCompleted() }
            requestStream = null
            val ch = channel
            channel = null
            ch?.shutdown()
            runCatching { ch?.awaitTermination(5, TimeUnit.SECONDS) }
            ch?.shutdownNow()
            Unit
        }
    }

    private fun notifyStreamTerminated(error: Throwable?) {
        if (!acceptStreamTerminationCallbacks) return
        if (!streamTerminationHandled.compareAndSet(false, true)) return
        acceptStreamTerminationCallbacks = false
        scope.launch {
            responseSink?.onTransportTerminated(error)
        }
    }
}
