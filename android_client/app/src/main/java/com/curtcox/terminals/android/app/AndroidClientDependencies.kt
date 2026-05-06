package com.curtcox.terminals.android.app

import com.curtcox.terminals.android.diagnostics.AndroidBuildMetadata

data class AndroidClientDependencies(
    val buildMetadata: AndroidBuildMetadata = AndroidBuildMetadata.fromBuildConfig(),
)
