package com.curtcox.terminals.android.media

import android.content.Context
import android.media.AudioAttributes
import android.media.MediaPlayer
import android.speech.tts.TextToSpeech
import java.util.Locale
import terminals.io.v1.Io

fun interface AndroidAudioPlayback {
    fun play(command: Io.PlayAudio): AudioPlaybackResult

    companion object {
        fun unsupported(): AndroidAudioPlayback = AndroidAudioPlayback { command: Io.PlayAudio ->
            AudioPlaybackResult.Unsupported(command.getSourceCase().name.lowercase())
        }
    }
}

class ContextAndroidAudioPlayback(
    context: Context,
) : AndroidAudioPlayback {
    private val appContext = context.applicationContext
    private var textToSpeech: TextToSpeech? = null

    override fun play(command: Io.PlayAudio): AudioPlaybackResult =
        when (command.sourceCase) {
            Io.PlayAudio.SourceCase.URL -> playUrl(command)
            Io.PlayAudio.SourceCase.TTS_TEXT -> speak(command)
            Io.PlayAudio.SourceCase.PCM_DATA -> {
                AudioPlaybackResult.Unsupported("pcm-data:${command.format.ifBlank { "unspecified-format" }}")
            }
            Io.PlayAudio.SourceCase.SOURCE_NOT_SET -> AudioPlaybackResult.Unsupported("missing-source")
            null -> AudioPlaybackResult.Unsupported("unknown-source")
        }

    private fun playUrl(command: Io.PlayAudio): AudioPlaybackResult {
        val url = command.url.trim()
        if (url.isEmpty()) {
            return AudioPlaybackResult.Unsupported("empty-url")
        }
        return try {
            val player = MediaPlayer()
            player.setAudioAttributes(
                AudioAttributes.Builder()
                    .setUsage(AudioAttributes.USAGE_MEDIA)
                    .setContentType(AudioAttributes.CONTENT_TYPE_MUSIC)
                    .build(),
            )
            player.setDataSource(url)
            player.setOnPreparedListener { prepared -> prepared.start() }
            player.setOnCompletionListener { completed -> completed.release() }
            player.setOnErrorListener { failed, _, _ ->
                failed.release()
                true
            }
            player.prepareAsync()
            AudioPlaybackResult.Played(command.requestId)
        } catch (_: RuntimeException) {
            AudioPlaybackResult.Unsupported("url-playback-failed")
        } catch (_: java.io.IOException) {
            AudioPlaybackResult.Unsupported("url-playback-failed")
        }
    }

    private fun speak(command: Io.PlayAudio): AudioPlaybackResult {
        val text = command.ttsText.trim()
        if (text.isEmpty()) {
            return AudioPlaybackResult.Unsupported("empty-tts-text")
        }
        return try {
            val tts = textToSpeech ?: TextToSpeech(appContext) { status ->
                if (status == TextToSpeech.SUCCESS) {
                    textToSpeech?.language = Locale.getDefault()
                }
            }.also { textToSpeech = it }
            val result = tts.speak(text, TextToSpeech.QUEUE_FLUSH, null, command.requestId.ifBlank { null })
            if (result == TextToSpeech.ERROR) {
                AudioPlaybackResult.Unsupported("tts-playback-failed")
            } else {
                AudioPlaybackResult.Played(command.requestId)
            }
        } catch (_: RuntimeException) {
            AudioPlaybackResult.Unsupported("tts-playback-failed")
        }
    }
}

sealed class AudioPlaybackResult {
    data class Played(val requestId: String) : AudioPlaybackResult()
    data class Unsupported(val reason: String) : AudioPlaybackResult()
}
