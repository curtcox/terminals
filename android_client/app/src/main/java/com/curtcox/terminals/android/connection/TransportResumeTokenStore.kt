package com.curtcox.terminals.android.connection

/**
 * Holds the last non-empty resume token from `TransportHelloAck` after a successful WebSocket
 * transport handshake, mirroring the Flutter shell’s static transport resume hint so reconnects
 * can offer session resumption to the server.
 */
class TransportResumeTokenStore {
    @Volatile
    private var token: String = ""

    fun current(): String = token

    fun captureFromAck(resumeToken: String) {
        val trimmed = resumeToken.trim()
        if (trimmed.isNotEmpty()) {
            token = trimmed
        }
    }

    fun clear() {
        token = ""
    }
}
