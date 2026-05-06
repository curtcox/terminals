package com.curtcox.terminals.android.media

import terminals.io.v1.Io

fun interface AndroidAudioPlayback {
    fun play(command: Io.PlayAudio): AudioPlaybackResult

    companion object {
        fun unsupported(): AndroidAudioPlayback = AndroidAudioPlayback { command: Io.PlayAudio ->
            AudioPlaybackResult.Unsupported(command.getSourceCase().name.lowercase())
        }
    }
}

sealed class AudioPlaybackResult {
    data class Played(val requestId: String) : AudioPlaybackResult()
    data class Unsupported(val reason: String) : AudioPlaybackResult()
}
