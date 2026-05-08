package com.curtcox.terminals.android.platform

interface AndroidPermissionRequester {
    fun hasPermission(permission: String): Boolean
    fun requestPermission(permission: String, onResult: (Boolean) -> Unit)

    companion object {
        fun none(): AndroidPermissionRequester =
            object : AndroidPermissionRequester {
                override fun hasPermission(permission: String): Boolean = false

                override fun requestPermission(permission: String, onResult: (Boolean) -> Unit) {
                    onResult(false)
                }
            }
    }
}
