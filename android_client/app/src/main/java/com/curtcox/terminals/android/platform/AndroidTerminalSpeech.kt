package com.curtcox.terminals.android.platform

import android.content.Context
import android.speech.tts.TextToSpeech
import java.util.Locale

/** Best-effort TTS for generic terminal alerts (Flutter `AlertDeliveryService` / `speakText` parity). */
fun interface AndroidTerminalSpeech {
    fun speak(text: String)

    companion object {
        fun none(): AndroidTerminalSpeech = AndroidTerminalSpeech { }
    }
}

class ContextAndroidTerminalSpeech(
    context: Context,
) : AndroidTerminalSpeech {
    private val appContext = context.applicationContext
    private var textToSpeech: TextToSpeech? = null

    override fun speak(text: String) {
        val trimmed = text.trim()
        if (trimmed.isEmpty()) return
        try {
            val tts = textToSpeech ?: TextToSpeech(appContext) { status ->
                if (status == TextToSpeech.SUCCESS) {
                    textToSpeech?.language = Locale.getDefault()
                }
            }.also { textToSpeech = it }
            tts.speak(trimmed, TextToSpeech.QUEUE_FLUSH, null, "terminal-alert-${System.nanoTime()}")
        } catch (_: RuntimeException) {
            // Match Flutter alert delivery: speech is best-effort if TTS init fails.
        }
    }
}
