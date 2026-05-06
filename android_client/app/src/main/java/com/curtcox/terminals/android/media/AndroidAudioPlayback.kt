package com.curtcox.terminals.android.media

interface AndroidAudioPlayback {
    fun play(bytes: ByteArray, contentType: String)
}
