package com.curtcox.terminals.android.platform

interface AndroidPermissionRequester {
    fun hasPermission(permission: String): Boolean
}
