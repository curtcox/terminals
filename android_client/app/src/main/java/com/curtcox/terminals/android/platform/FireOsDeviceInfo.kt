package com.curtcox.terminals.android.platform

data class FireOsDeviceInfo(
    val manufacturer: String,
    val model: String,
    val sdkInt: Int,
) {
    val isLikelyFireOs: Boolean
        get() = manufacturer.equals("Amazon", ignoreCase = true)
}
