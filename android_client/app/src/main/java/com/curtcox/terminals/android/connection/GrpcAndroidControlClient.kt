package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

class GrpcAndroidControlClient : AndroidControlClient {
    override suspend fun connect(endpoint: EndpointResolution) {
        error("gRPC control stream is planned for the protobuf phase.")
    }

    override suspend fun send(request: Control.ConnectRequest) {
        error("gRPC control stream is planned for the protobuf phase.")
    }

    override suspend fun close() = Unit
}
