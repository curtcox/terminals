package com.curtcox.terminals.android.platform

import android.graphics.Bitmap
import android.graphics.Canvas
import android.view.Window
import java.io.ByteArrayOutputStream

/**
 * Best-effort full-window PNG for [Diagnostics.BugReport.screenshot_png], mirroring the Flutter
 * shell’s RepaintBoundary capture (generic terminal diagnostics only).
 */
object WindowBugReportScreenshotCapture {
    fun capturePngOrNull(window: Window): ByteArray? {
        val view = window.decorView
        val w = view.width
        val h = view.height
        if (w <= 0 || h <= 0) {
            return null
        }
        return try {
            val bitmap = Bitmap.createBitmap(w, h, Bitmap.Config.ARGB_8888)
            val canvas = Canvas(bitmap)
            view.draw(canvas)
            val stream = ByteArrayOutputStream(bitmap.byteCount.coerceAtLeast(16_384))
            val ok = bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream)
            bitmap.recycle()
            if (!ok) {
                return null
            }
            stream.toByteArray().takeIf { it.isNotEmpty() }
        } catch (_: Exception) {
            null
        }
    }
}
