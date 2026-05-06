package com.curtcox.terminals.android.diagnostics

import com.curtcox.terminals.android.BuildConfig

data class AndroidBuildMetadata(
    val versionName: String,
    val buildSha: String,
    val buildDate: String,
) {
    companion object {
        fun fromBuildConfig() = AndroidBuildMetadata(
            versionName = BuildConfig.VERSION_NAME,
            buildSha = BuildConfig.TERMINALS_BUILD_SHA,
            buildDate = BuildConfig.TERMINALS_BUILD_DATE,
        )
    }
}
