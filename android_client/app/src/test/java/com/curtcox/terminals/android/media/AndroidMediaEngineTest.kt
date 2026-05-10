package com.curtcox.terminals.android.media

import com.google.protobuf.ByteString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test
import terminals.control.v1.Control
import terminals.io.v1.Io

class AndroidMediaEngineTest {
    @Test
    fun unsupportedAudioPlaybackReportsSourceReason() {
        val result = AndroidAudioPlayback.unsupported().play(
            Io.PlayAudio.newBuilder()
                .setRequestId("audio-1")
                .setDeviceId("device-1")
                .setPcmData(ByteString.copyFrom(byteArrayOf(1, 2, 3)))
                .build(),
        )

        assertEquals(AudioPlaybackResult.Unsupported("pcm_data"), result)
    }

    @Test
    fun unsupportedMediaDisplayReportsMediaType() {
        val result = AndroidMediaDisplay.unsupported().show(
            Io.ShowMedia.newBuilder()
                .setRequestId("media-1")
                .setDeviceId("device-1")
                .setMediaUrl("https://example.test/photo.png")
                .setMediaType("image/png")
                .build(),
        )

        assertEquals(MediaDisplayResult.Unsupported("image/png"), result)
    }

    @Test
    fun unsupportedMediaDisplayUsesDeterministicMissingTypeReason() {
        val result = AndroidMediaDisplay.unsupported().show(
            Io.ShowMedia.newBuilder()
                .setRequestId("media-2")
                .setDeviceId("device-1")
                .setMediaUrl("https://example.test/asset")
                .build(),
        )

        assertEquals(MediaDisplayResult.Unsupported("unspecified-media"), result)
    }

    @Test
    fun mediaEngineDelegatesAudioAndDisplayAdapters() {
        val engine = AndroidMediaEngine(
            audioPlayback = AndroidAudioPlayback { AudioPlaybackResult.Played(it.requestId) },
            mediaDisplay = AndroidMediaDisplay { MediaDisplayResult.Shown(it.requestId) },
        )

        val audio = engine.playAudio(
            Io.PlayAudio.newBuilder()
                .setRequestId("audio-2")
                .setDeviceId("device-1")
                .setTtsText("hello")
                .build(),
        )
        val media = engine.showMedia(
            Io.ShowMedia.newBuilder()
                .setRequestId("media-3")
                .setDeviceId("device-1")
                .setMediaUrl("https://example.test/clip.mp4")
                .setMediaType("video/mp4")
                .build(),
        )

        assertEquals(AudioPlaybackResult.Played("audio-2"), audio)
        assertEquals(MediaDisplayResult.Shown("media-3"), media)
    }

    @Test
    fun disabledWebRtcAdapterReportsCompatibilityDecision() {
        val support = AndroidWebRtcAdapter.disabled("fire-os-webrtc-not-enabled").currentSupport()

        assertFalse(support.supported)
        assertEquals("fire-os-webrtc-not-enabled", support.reason)
    }

    @Test
    fun liveMediaDelegatesStopRouteAndSignalToSession() {
        val live = RecordingLiveMediaSession()
        val engine = AndroidMediaEngine(liveMedia = live)

        val start = Io.StartStream.newBuilder().setStreamId("a1").build()
        assertEquals(LiveMediaSessionResult.Applied, engine.applyStartStream(start))
        engine.applyStopStream("a1")
        val route = Io.RouteStream.newBuilder().setStreamId("a1").setSourceDeviceId("s").setTargetDeviceId("t").build()
        engine.applyRouteStream(route)
        val signal = Control.WebRTCSignal.newBuilder().setStreamId("a1").build()
        engine.applyWebRtcSignal(signal)

        assertEquals(listOf(start), live.starts)
        assertEquals(listOf("a1"), live.stops)
        assertEquals(listOf(route), live.routes)
        assertEquals(listOf(signal), live.signals)
    }

    private class RecordingLiveMediaSession : AndroidLiveMediaSession {
        val starts = mutableListOf<Io.StartStream>()
        val stops = mutableListOf<String>()
        val routes = mutableListOf<Io.RouteStream>()
        val signals = mutableListOf<Control.WebRTCSignal>()

        override fun applyStartStream(start: Io.StartStream): LiveMediaSessionResult {
            starts += start
            return LiveMediaSessionResult.Applied
        }

        override fun applyStopStream(streamId: String): LiveMediaSessionResult {
            stops += streamId
            return LiveMediaSessionResult.Applied
        }

        override fun applyRouteStream(route: Io.RouteStream): LiveMediaSessionResult {
            routes += route
            return LiveMediaSessionResult.Applied
        }

        override fun applyWebRtcSignal(signal: Control.WebRTCSignal): LiveMediaSessionResult {
            signals += signal
            return LiveMediaSessionResult.Applied
        }
    }
}
