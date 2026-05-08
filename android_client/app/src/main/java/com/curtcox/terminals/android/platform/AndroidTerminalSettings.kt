package com.curtcox.terminals.android.platform

import android.content.Context

interface AndroidTerminalSettings {
    fun lastManualEndpoint(): String
    fun setLastManualEndpoint(endpoint: String)
    fun keepAwakeEnabled(): Boolean
    fun setKeepAwakeEnabled(enabled: Boolean)
    fun fullscreenEnabled(): Boolean
    fun setFullscreenEnabled(enabled: Boolean)
    fun brightDisplayEnabled(): Boolean
    fun setBrightDisplayEnabled(enabled: Boolean)

    companion object {
        fun inMemory(
            initialEndpoint: String = "",
            initialKeepAwakeEnabled: Boolean = false,
            initialFullscreenEnabled: Boolean = false,
            initialBrightDisplayEnabled: Boolean = false,
        ): AndroidTerminalSettings =
            object : AndroidTerminalSettings {
                private var endpoint = initialEndpoint
                private var keepAwake = initialKeepAwakeEnabled
                private var fullscreen = initialFullscreenEnabled
                private var brightDisplay = initialBrightDisplayEnabled

                override fun lastManualEndpoint(): String = endpoint

                override fun setLastManualEndpoint(endpoint: String) {
                    this.endpoint = endpoint
                }

                override fun keepAwakeEnabled(): Boolean = keepAwake

                override fun setKeepAwakeEnabled(enabled: Boolean) {
                    keepAwake = enabled
                }

                override fun fullscreenEnabled(): Boolean = fullscreen

                override fun setFullscreenEnabled(enabled: Boolean) {
                    fullscreen = enabled
                }

                override fun brightDisplayEnabled(): Boolean = brightDisplay

                override fun setBrightDisplayEnabled(enabled: Boolean) {
                    brightDisplay = enabled
                }
            }
    }
}

class SharedPreferencesAndroidTerminalSettings(
    context: Context,
) : AndroidTerminalSettings {
    private val preferences = context.applicationContext.getSharedPreferences(
        "terminal_settings",
        Context.MODE_PRIVATE,
    )

    override fun lastManualEndpoint(): String =
        preferences.getString(KEY_LAST_MANUAL_ENDPOINT, "").orEmpty()

    override fun setLastManualEndpoint(endpoint: String) {
        preferences.edit().putString(KEY_LAST_MANUAL_ENDPOINT, endpoint).apply()
    }

    override fun keepAwakeEnabled(): Boolean =
        preferences.getBoolean(KEY_KEEP_AWAKE_ENABLED, false)

    override fun setKeepAwakeEnabled(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_KEEP_AWAKE_ENABLED, enabled).apply()
    }

    override fun fullscreenEnabled(): Boolean =
        preferences.getBoolean(KEY_FULLSCREEN_ENABLED, false)

    override fun setFullscreenEnabled(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_FULLSCREEN_ENABLED, enabled).apply()
    }

    override fun brightDisplayEnabled(): Boolean =
        preferences.getBoolean(KEY_BRIGHT_DISPLAY_ENABLED, false)

    override fun setBrightDisplayEnabled(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_BRIGHT_DISPLAY_ENABLED, enabled).apply()
    }

    companion object {
        private const val KEY_LAST_MANUAL_ENDPOINT = "last_manual_endpoint"
        private const val KEY_KEEP_AWAKE_ENABLED = "keep_awake_enabled"
        private const val KEY_FULLSCREEN_ENABLED = "fullscreen_enabled"
        private const val KEY_BRIGHT_DISPLAY_ENABLED = "bright_display_enabled"
    }
}
