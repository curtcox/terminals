package com.curtcox.terminals.android.platform

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
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

interface AndroidNetworkMonitor {
    fun start(onChanged: () -> Unit)
    fun stop()

    companion object {
        fun none(): AndroidNetworkMonitor = object : AndroidNetworkMonitor {
            override fun start(onChanged: () -> Unit) = Unit
            override fun stop() = Unit
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

class ContextAndroidNetworkMonitor(
    context: Context,
) : AndroidNetworkMonitor {
    private val connectivityManager =
        context.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    private var callback: ConnectivityManager.NetworkCallback? = null

    override fun start(onChanged: () -> Unit) {
        stop()
        val nextCallback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                onChanged()
            }

            override fun onLost(network: Network) {
                onChanged()
            }

            override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
                onChanged()
            }
        }
        connectivityManager.registerDefaultNetworkCallback(nextCallback)
        callback = nextCallback
    }

    override fun stop() {
        val previous = callback ?: return
        callback = null
        runCatching { connectivityManager.unregisterNetworkCallback(previous) }
    }
}
