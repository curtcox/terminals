package com.curtcox.terminals.android.platform

data class FireOsDeviceInfo(
    val manufacturer: String,
    val model: String,
    val sdkInt: Int,
) {
    val isLikelyFireOs: Boolean
        get() = manufacturer.equals("Amazon", ignoreCase = true)
}

fun interface FireOsDeviceInfoProvider {
    fun current(): FireOsDeviceInfo

    companion object {
        fun unknown(): FireOsDeviceInfoProvider = FireOsDeviceInfoProvider {
            FireOsDeviceInfo(
                manufacturer = "unknown",
                model = "unknown",
                sdkInt = 0,
            )
        }
    }
}
