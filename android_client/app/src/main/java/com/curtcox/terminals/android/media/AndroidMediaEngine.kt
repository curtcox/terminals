package com.curtcox.terminals.android.media

import terminals.io.v1.Io

class AndroidMediaEngine(
    private val audioPlayback: AndroidAudioPlayback = AndroidAudioPlayback.unsupported(),
    private val mediaDisplay: AndroidMediaDisplay = AndroidMediaDisplay.unsupported(),
) {
    fun playAudio(command: Io.PlayAudio): AudioPlaybackResult = audioPlayback.play(command)

    fun showMedia(command: Io.ShowMedia): MediaDisplayResult = mediaDisplay.show(command)

    companion object {
        fun unsupported(): AndroidMediaEngine = AndroidMediaEngine()
    }
}

fun interface AndroidMediaDisplay {
    fun show(command: Io.ShowMedia): MediaDisplayResult

    companion object {
        fun unsupported(): AndroidMediaDisplay = AndroidMediaDisplay { command ->
            MediaDisplayResult.Unsupported(command.mediaType.ifBlank { "unspecified-media" })
        }
    }
}

sealed class MediaDisplayResult {
    data class Shown(val requestId: String) : MediaDisplayResult()
    data class Unsupported(val reason: String) : MediaDisplayResult()
}
