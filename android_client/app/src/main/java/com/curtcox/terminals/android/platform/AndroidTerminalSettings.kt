package com.curtcox.terminals.android.platform

import android.content.Context

interface AndroidTerminalSettings {
    fun lastManualEndpoint(): String
    fun setLastManualEndpoint(endpoint: String)
    fun keepAwakeEnabled(): Boolean
    fun setKeepAwakeEnabled(enabled: Boolean)
    fun fullscreenEnabled(): Boolean
    fun setFullscreenEnabled(enabled: Boolean)

    companion object {
        fun inMemory(
            initialEndpoint: String = "",
            initialKeepAwakeEnabled: Boolean = false,
            initialFullscreenEnabled: Boolean = false,
        ): AndroidTerminalSettings =
            object : AndroidTerminalSettings {
                private var endpoint = initialEndpoint
                private var keepAwake = initialKeepAwakeEnabled
                private var fullscreen = initialFullscreenEnabled

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

    companion object {
        private const val KEY_LAST_MANUAL_ENDPOINT = "last_manual_endpoint"
        private const val KEY_KEEP_AWAKE_ENABLED = "keep_awake_enabled"
        private const val KEY_FULLSCREEN_ENABLED = "fullscreen_enabled"
    }
}
