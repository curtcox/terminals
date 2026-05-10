package com.curtcox.terminals.android.media

import terminals.control.v1.Control
import terminals.io.v1.Io

/**
 * Live streaming / WebRTC control hooks invoked from the terminal shell for inbound
 * [terminals.control.v1.ConnectResponse] payloads, parallel to Flutter [ClientMediaEngine].
 */
interface AndroidLiveMediaSession {
    fun applyStartStream(start: Io.StartStream): LiveMediaSessionResult

    fun applyStopStream(streamId: String): LiveMediaSessionResult

    fun applyRouteStream(route: Io.RouteStream): LiveMediaSessionResult

    fun applyWebRtcSignal(signal: Control.WebRTCSignal): LiveMediaSessionResult

    companion object {
        fun disabled(reason: String = "webrtc-dependency-not-enabled"): AndroidLiveMediaSession =
            object : AndroidLiveMediaSession {
                override fun applyStartStream(start: Io.StartStream) =
                    LiveMediaSessionResult.Unsupported(reason)

                override fun applyStopStream(streamId: String) = LiveMediaSessionResult.Applied

                override fun applyRouteStream(route: Io.RouteStream) = LiveMediaSessionResult.Applied

                override fun applyWebRtcSignal(signal: Control.WebRTCSignal) =
                    LiveMediaSessionResult.Applied
            }

        fun fromAdapter(adapter: AndroidWebRtcAdapter): AndroidLiveMediaSession =
            WebRtcGatedLiveMediaSession(adapter)
    }
}

sealed class LiveMediaSessionResult {
    data object Applied : LiveMediaSessionResult()

    data class Unsupported(val reason: String) : LiveMediaSessionResult()
}

private class WebRtcGatedLiveMediaSession(
    private val adapter: AndroidWebRtcAdapter,
) : AndroidLiveMediaSession {
    override fun applyStartStream(start: Io.StartStream): LiveMediaSessionResult {
        val s = adapter.currentSupport()
        if (!s.supported) return LiveMediaSessionResult.Unsupported(s.reason)
        return LiveMediaSessionResult.Unsupported("live-media-session-not-implemented")
    }

    override fun applyStopStream(streamId: String): LiveMediaSessionResult = LiveMediaSessionResult.Applied

    override fun applyRouteStream(route: Io.RouteStream): LiveMediaSessionResult =
        LiveMediaSessionResult.Applied

    override fun applyWebRtcSignal(signal: Control.WebRTCSignal): LiveMediaSessionResult {
        val s = adapter.currentSupport()
        if (!s.supported) return LiveMediaSessionResult.Applied
        return LiveMediaSessionResult.Unsupported("live-media-session-not-implemented")
    }
}
