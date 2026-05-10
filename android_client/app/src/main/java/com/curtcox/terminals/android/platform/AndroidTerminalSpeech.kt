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
            val tts = textToSpeech ?: run {
                // OnInit may run before [holder] is assigned; set locale when the instance is already reachable.
                val holder = arrayOfNulls<TextToSpeech>(1)
                val created = TextToSpeech(appContext) { status ->
                    if (status == TextToSpeech.SUCCESS) {
                        holder[0]?.language = Locale.getDefault()
                    }
                }
                holder[0] = created
                created.also { textToSpeech = it }
            }
            tts.speak(trimmed, TextToSpeech.QUEUE_FLUSH, null, "terminal-alert-${System.nanoTime()}")
        } catch (_: RuntimeException) {
            // Match Flutter alert delivery: speech is best-effort if TTS init fails.
        }
    }
}
