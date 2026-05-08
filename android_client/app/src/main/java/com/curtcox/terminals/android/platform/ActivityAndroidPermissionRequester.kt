package com.curtcox.terminals.android.platform

import android.content.pm.PackageManager
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat

class ActivityAndroidPermissionRequester(
    private val activity: ComponentActivity,
) : AndroidPermissionRequester {
    private val pendingCallbacks = mutableMapOf<String, MutableList<(Boolean) -> Unit>>()
    private val requestQueue = ArrayDeque<String>()
    private var activePermission: String? = null
    private val requestLauncher = activity.registerForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { granted ->
        val permission = activePermission
        val callbacks = pendingCallbacks.remove(permission.orEmpty()).orEmpty()
        callbacks.forEach { callback -> callback(granted) }
        activePermission = null
        launchNext()
    }

    override fun hasPermission(permission: String): Boolean =
        ContextCompat.checkSelfPermission(activity, permission) == PackageManager.PERMISSION_GRANTED

    override fun requestPermission(permission: String, onResult: (Boolean) -> Unit) {
        if (hasPermission(permission)) {
            onResult(true)
            return
        }
        pendingCallbacks.getOrPut(permission) { mutableListOf() }.add(onResult)
        if (activePermission != permission && !requestQueue.contains(permission)) {
            requestQueue.addLast(permission)
        }
        launchNext()
    }

    private fun launchNext() {
        if (activePermission != null) return
        val nextPermission = requestQueue.removeFirstOrNull() ?: return
        activePermission = nextPermission
        requestLauncher.launch(nextPermission)
    }
}
