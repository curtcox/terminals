package com.curtcox.terminals.android.discovery

import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.os.Build
import com.curtcox.terminals.android.util.Clock

interface AndroidNsdDiscovery {
    fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit)
    fun stop()
}

class NsdAndroidDiscovery(
    private val nsdManager: NsdManager,
    private val clock: Clock,
    private val serviceType: String = TerminalsServiceType,
) : AndroidNsdDiscovery {
    private var discoveryListener: NsdManager.DiscoveryListener? = null

    override fun start(onServer: (DiscoveredServer) -> Unit, onError: (String) -> Unit) {
        stop()
        val listener = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(regType: String) = Unit

            override fun onServiceFound(serviceInfo: NsdServiceInfo) {
                if (serviceInfo.serviceType != serviceType) return
                nsdManager.resolveService(
                    serviceInfo,
                    object : NsdManager.ResolveListener {
                        override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                            onError("mDNS resolve failed for ${serviceInfo.serviceName}: $errorCode")
                        }

                        override fun onServiceResolved(serviceInfo: NsdServiceInfo) {
                            serviceInfo.toDiscoveredServer(clock.nowMillis())?.let(onServer)
                                ?: onError("mDNS resolved service without usable host/port: ${serviceInfo.serviceName}")
                        }
                    },
                )
            }

            override fun onServiceLost(serviceInfo: NsdServiceInfo) = Unit

            override fun onDiscoveryStopped(serviceType: String) = Unit

            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                onError("mDNS discovery failed to start for $serviceType: $errorCode")
                stop()
            }

            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
                onError("mDNS discovery failed to stop for $serviceType: $errorCode")
                stop()
            }
        }
        discoveryListener = listener
        nsdManager.discoverServices(serviceType, NsdManager.PROTOCOL_DNS_SD, listener)
    }

    override fun stop() {
        val listener = discoveryListener ?: return
        discoveryListener = null
        runCatching { nsdManager.stopServiceDiscovery(listener) }
    }
}

internal const val TerminalsServiceType = "_terminals._tcp."

internal fun NsdServiceInfo.toDiscoveredServer(nowMillis: Long): DiscoveredServer? {
    val resolvedHost = host?.hostAddress ?: return null
    if (port !in 1..65535) return null
    val metadata = terminalTxtMetadata()
    return DiscoveredServer(
        name = metadata["name"].takeUnless { it.isNullOrBlank() } ?: serviceName,
        host = resolvedHost,
        port = port,
        lastSeenMillis = nowMillis,
        grpcEndpoint = metadata["grpc"].orEmpty(),
        webSocketEndpoint = metadata["ws"].orEmpty(),
        httpEndpoint = metadata["http"].orEmpty(),
        carrierPriority = metadata["priority"].orEmpty()
            .split(',')
            .map { it.trim() }
            .filter { it.isNotEmpty() },
        metadata = metadata,
    )
}

internal fun NsdServiceInfo.terminalTxtMetadata(): Map<String, String> {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.LOLLIPOP) return emptyMap()
    return parseTerminalTxtAttributes(attributes)
}

internal fun parseTerminalTxtAttributes(attributes: Map<String, ByteArray>): Map<String, String> =
    attributes.mapNotNull { (key, value) ->
        val text = runCatching { value.toString(Charsets.UTF_8) }.getOrDefault("")
        if (key.isBlank() || text.isBlank()) null else key to text
    }.toMap()
