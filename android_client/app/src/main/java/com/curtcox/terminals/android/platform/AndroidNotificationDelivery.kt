package com.curtcox.terminals.android.platform

interface AndroidNotificationDelivery {
    fun deliver(title: String, body: String)
}
