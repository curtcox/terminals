package com.curtcox.terminals.android.ui

import androidx.compose.ui.graphics.Color

internal fun parseHexColor(raw: String?): Color? {
    if (raw.isNullOrBlank()) return null
    var value = raw.trim()
    if (value.startsWith("#")) value = value.drop(1)
    if (value.length == 6) value = "FF$value"
    if (value.length != 8) return null
    val parsed = value.toLongOrNull(16) ?: return null
    return Color(parsed)
}

internal fun parseColorOrUnspecified(raw: String?): Color = parseHexColor(raw) ?: Color.Unspecified
