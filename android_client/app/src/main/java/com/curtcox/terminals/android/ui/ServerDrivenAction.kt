package com.curtcox.terminals.android.ui

data class ServerDrivenAction(
    val componentId: String,
    val action: String,
    val value: String = "",
)
