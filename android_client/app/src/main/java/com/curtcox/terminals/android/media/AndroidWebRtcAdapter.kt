package com.curtcox.terminals.android.media

fun interface AndroidWebRtcAdapter {
    fun currentSupport(): AndroidWebRtcSupport

    companion object {
        fun disabled(reason: String = "webrtc-dependency-not-enabled"): AndroidWebRtcAdapter =
            AndroidWebRtcAdapter { AndroidWebRtcSupport(supported = false, reason = reason) }
    }
}

data class AndroidWebRtcSupport(
    val supported: Boolean,
    val reason: String = "",
)
