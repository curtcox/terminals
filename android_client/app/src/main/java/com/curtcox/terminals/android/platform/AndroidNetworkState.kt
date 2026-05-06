package com.curtcox.terminals.android.platform

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities

data class AndroidNetworkState(
    val connected: Boolean,
    val metered: Boolean,
)

fun interface AndroidNetworkStateProvider {
    fun current(): AndroidNetworkState

    companion object {
        fun unknown(): AndroidNetworkStateProvider = AndroidNetworkStateProvider {
            AndroidNetworkState(connected = false, metered = false)
        }
    }
}

class ContextAndroidNetworkStateProvider(
    context: Context,
) : AndroidNetworkStateProvider {
    private val connectivityManager =
        context.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager

    override fun current(): AndroidNetworkState {
        val network = connectivityManager.activeNetwork ?: return AndroidNetworkState(
            connected = false,
            metered = connectivityManager.isActiveNetworkMetered,
        )
        val capabilities = connectivityManager.getNetworkCapabilities(network)
        val connected = capabilities?.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) == true &&
            capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
        return AndroidNetworkState(
            connected = connected,
            metered = connectivityManager.isActiveNetworkMetered,
        )
    }
}
