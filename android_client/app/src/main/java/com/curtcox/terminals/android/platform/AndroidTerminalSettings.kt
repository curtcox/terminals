package com.curtcox.terminals.android.platform

import android.content.Context

interface AndroidTerminalSettings {
    fun lastManualEndpoint(): String
    fun setLastManualEndpoint(endpoint: String)

    companion object {
        fun inMemory(initialEndpoint: String = ""): AndroidTerminalSettings =
            object : AndroidTerminalSettings {
                private var endpoint = initialEndpoint

                override fun lastManualEndpoint(): String = endpoint

                override fun setLastManualEndpoint(endpoint: String) {
                    this.endpoint = endpoint
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

    companion object {
        private const val KEY_LAST_MANUAL_ENDPOINT = "last_manual_endpoint"
    }
}
