package com.curtcox.terminals.android.connection

import kotlin.math.min

class ReconnectPolicy(
    private val initialDelayMillis: Long = 500,
    private val maxDelayMillis: Long = 15_000,
) {
    fun delayForAttempt(attempt: Int): Long {
        if (attempt <= 0) return 0
        var delay = initialDelayMillis
        repeat(attempt - 1) {
            delay = min(delay * 2, maxDelayMillis)
        }
        return delay
    }
}
