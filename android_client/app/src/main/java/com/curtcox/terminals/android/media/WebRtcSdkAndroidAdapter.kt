package com.curtcox.terminals.android.media

import android.content.Context
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import org.webrtc.PeerConnection
import org.webrtc.PeerConnectionFactory
import terminals.control.v1.Control

/**
 * [AndroidWebRtcAdapter] backed by `io.github.webrtc-sdk:android`. Reports [AndroidWebRtcSupport.supported]
 * = true when [PeerConnectionFactory] initializes successfully. Falls back to [AndroidWebRtcAdapter.disabled]
 * via [fromContext] when initialization fails (e.g. on devices where the native libs cannot load).
 */
class WebRtcSdkAndroidAdapter private constructor(
    internal val factory: PeerConnectionFactory,
) : AndroidWebRtcAdapter {

    override fun currentSupport(): AndroidWebRtcSupport =
        AndroidWebRtcSupport(supported = true, reason = "webrtc-sdk-android")

    /**
     * Creates a [WebRtcSdkLiveMediaSession] backed by this adapter's [PeerConnectionFactory].
     *
     * @param signalSender Coroutine that forwards outbound WebRTC signals (ICE candidates, SDP answer)
     *   to the server. Defaults to a no-op if the send channel is not yet wired.
     * @param scope Coroutine scope used to launch [signalSender] from WebRTC observer callbacks.
     * @param iceServers ICE server list passed to each [PeerConnection]. Empty by default (LAN
     *   connectivity does not require STUN/TURN).
     */
    fun createSession(
        signalSender: suspend (Control.WebRTCSignal) -> Unit = {},
        scope: CoroutineScope = CoroutineScope(Dispatchers.Main),
        iceServers: List<PeerConnection.IceServer> = emptyList(),
    ): WebRtcSdkLiveMediaSession = WebRtcSdkLiveMediaSession(factory, signalSender, scope, iceServers)

    companion object {
        /**
         * Attempts to initialize [PeerConnectionFactory] using [context]. Returns a
         * [WebRtcSdkAndroidAdapter] on success, or [AndroidWebRtcAdapter.disabled] if
         * initialization throws (native library load failure, missing permissions, etc.).
         */
        fun fromContext(context: Context): AndroidWebRtcAdapter =
            runCatching {
                PeerConnectionFactory.initialize(
                    PeerConnectionFactory.InitializationOptions
                        .builder(context.applicationContext)
                        .createInitializationOptions(),
                )
                WebRtcSdkAndroidAdapter(PeerConnectionFactory.builder().createPeerConnectionFactory())
            }.getOrElse { e ->
                AndroidWebRtcAdapter.disabled("webrtc-init-failed:${e.javaClass.simpleName}")
            }
    }
}
