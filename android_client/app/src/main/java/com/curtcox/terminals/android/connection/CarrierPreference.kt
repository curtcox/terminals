package com.curtcox.terminals.android.connection

import terminals.control.v1.Control

enum class CarrierPreference {
    Grpc,
    WebSocket;

    val protoKind: Control.CarrierKind
        get() = when (this) {
            Grpc -> Control.CarrierKind.CARRIER_KIND_GRPC
            WebSocket -> Control.CarrierKind.CARRIER_KIND_WEBSOCKET
        }

    companion object {
        val fireOsDefaultOrder: List<CarrierPreference> = listOf(Grpc, WebSocket)

        fun fromProto(kind: Control.CarrierKind): CarrierPreference? =
            when (kind) {
                Control.CarrierKind.CARRIER_KIND_GRPC -> Grpc
                Control.CarrierKind.CARRIER_KIND_WEBSOCKET -> WebSocket
                else -> null
            }

        fun preferredOrder(
            localSupported: List<CarrierPreference> = fireOsDefaultOrder,
            serverAdvertised: List<Control.CarrierKind> = emptyList(),
        ): List<CarrierPreference> {
            val local = localSupported.distinct()
            if (serverAdvertised.isEmpty()) return local

            val localSet = local.toSet()
            val serverPreferred = serverAdvertised
                .mapNotNull(::fromProto)
                .filter { it in localSet }
                .distinct()
            val remainingLocal = local.filterNot { it in serverPreferred.toSet() }
            return serverPreferred + remainingLocal
        }
    }
}
