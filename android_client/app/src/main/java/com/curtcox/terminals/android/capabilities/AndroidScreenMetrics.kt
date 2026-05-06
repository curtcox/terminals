package com.curtcox.terminals.android.capabilities

data class AndroidScreenMetrics(
    val widthPx: Int,
    val heightPx: Int,
    val density: Float,
    val orientation: String,
    val safeArea: AndroidInsets = AndroidInsets(),
)

data class AndroidInsets(
    val leftPx: Int = 0,
    val topPx: Int = 0,
    val rightPx: Int = 0,
    val bottomPx: Int = 0,
)
