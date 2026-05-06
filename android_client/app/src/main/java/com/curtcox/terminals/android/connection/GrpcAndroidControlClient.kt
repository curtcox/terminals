package com.curtcox.terminals.android.connection

class GrpcAndroidControlClient : AndroidControlClient {
    override suspend fun connect(endpoint: EndpointResolution) {
        error("gRPC control stream is planned for the protobuf phase.")
    }

    override suspend fun close() = Unit
}
