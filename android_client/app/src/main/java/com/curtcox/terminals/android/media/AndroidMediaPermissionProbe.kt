package com.curtcox.terminals.android.media

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import androidx.core.content.ContextCompat

fun interface AndroidMediaPermissionProbe {
    fun current(): AndroidMediaPermissionState

    companion object {
        fun unavailable(): AndroidMediaPermissionProbe = AndroidMediaPermissionProbe {
            AndroidMediaPermissionState()
        }
    }
}

data class AndroidMediaPermissionState(
    val microphoneGranted: Boolean = false,
    val cameraGranted: Boolean = false,
)

class ContextAndroidMediaPermissionProbe(context: Context) : AndroidMediaPermissionProbe {
    private val appContext = context.applicationContext

    override fun current(): AndroidMediaPermissionState =
        AndroidMediaPermissionState(
            microphoneGranted = isGranted(Manifest.permission.RECORD_AUDIO),
            cameraGranted = isGranted(Manifest.permission.CAMERA),
        )

    private fun isGranted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(appContext, permission) == PackageManager.PERMISSION_GRANTED
}
