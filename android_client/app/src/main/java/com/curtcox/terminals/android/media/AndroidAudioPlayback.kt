package com.curtcox.terminals.android.media

import terminals.io.v1.Io

interface AndroidAudioPlayback {
    fun play(command: Io.PlayAudio): AudioPlaybackResult

    companion object {
        fun unsupported(): AndroidAudioPlayback = AndroidAudioPlayback { command ->
            AudioPlaybackResult.Unsupported(command.sourceCase.name.lowercase())
        }
    }
}

sealed class AudioPlaybackResult {
    data class Played(val requestId: String) : AudioPlaybackResult()
    data class Unsupported(val reason: String) : AudioPlaybackResult()
}
