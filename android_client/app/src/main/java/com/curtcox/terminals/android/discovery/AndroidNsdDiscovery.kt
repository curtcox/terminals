package com.curtcox.terminals.android.discovery

interface AndroidNsdDiscovery {
    fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit)
    fun stop()
}
