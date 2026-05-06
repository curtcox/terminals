package com.curtcox.terminals.android.connection

import org.junit.Assert.assertEquals
import org.junit.Test
import terminals.control.v1.Control

class CarrierPreferenceTest {
    @Test
    fun usesLocalDefaultWhenServerDoesNotAdvertiseCarriers() {
        assertEquals(
            listOf(CarrierPreference.Grpc, CarrierPreference.WebSocket),
            CarrierPreference.preferredOrder(),
        )
    }

    @Test
    fun honorsServerPriorityForLocallySupportedCarriers() {
        val order = CarrierPreference.preferredOrder(
            serverAdvertised = listOf(
                Control.CarrierKind.CARRIER_KIND_WEBSOCKET,
                Control.CarrierKind.CARRIER_KIND_GRPC,
            ),
        )

        assertEquals(listOf(CarrierPreference.WebSocket, CarrierPreference.Grpc), order)
    }

    @Test
    fun ignoresUnsupportedAndUnknownServerCarriers() {
        val order = CarrierPreference.preferredOrder(
            localSupported = listOf(CarrierPreference.WebSocket),
            serverAdvertised = listOf(
                Control.CarrierKind.CARRIER_KIND_TCP,
                Control.CarrierKind.CARRIER_KIND_GRPC,
                Control.CarrierKind.CARRIER_KIND_WEBSOCKET,
            ),
        )

        assertEquals(listOf(CarrierPreference.WebSocket), order)
    }

    @Test
    fun appendsLocalFallbacksAfterServerPreferredIntersection() {
        val order = CarrierPreference.preferredOrder(
            localSupported = listOf(CarrierPreference.Grpc, CarrierPreference.WebSocket),
            serverAdvertised = listOf(Control.CarrierKind.CARRIER_KIND_WEBSOCKET),
        )

        assertEquals(listOf(CarrierPreference.WebSocket, CarrierPreference.Grpc), order)
    }
}
