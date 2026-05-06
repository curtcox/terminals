package com.curtcox.terminals.android.capabilities

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.content.res.Configuration
import android.os.BatteryManager
import android.os.Build
import android.os.Vibrator
import android.os.VibratorManager
import android.provider.Settings
import android.view.WindowInsets
import android.view.WindowManager
import androidx.core.content.ContextCompat
import terminals.capabilities.v1.Capabilities

class ContextAndroidCapabilityProbe(
    context: Context,
    private val deviceName: String = defaultDeviceName(context),
) : AndroidCapabilityProbe {
    private val appContext = context.applicationContext
    private val packageManager = appContext.packageManager

    override fun current(): AndroidCapabilitySnapshotInput =
        AndroidCapabilitySnapshotInput(
            identity = Capabilities.DeviceIdentity.newBuilder()
                .setDeviceName(deviceName)
                .setDeviceType("tablet")
                .setPlatform("android")
                .build(),
            screenMetrics = currentScreenMetrics(),
            permissions = currentPermissions(),
            hardware = currentHardware(),
            power = currentPower(),
        )

    private fun currentScreenMetrics(): AndroidScreenMetrics {
        val resources = appContext.resources
        val density = resources.displayMetrics.density
        val orientation = when (resources.configuration.orientation) {
            Configuration.ORIENTATION_PORTRAIT -> "portrait"
            Configuration.ORIENTATION_LANDSCAPE -> "landscape"
            else -> "unknown"
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            val windowManager = appContext.getSystemService(WindowManager::class.java)
            val metrics = windowManager.currentWindowMetrics
            val bounds = metrics.bounds
            val insets = metrics.windowInsets.getInsetsIgnoringVisibility(
                WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout(),
            )
            return AndroidScreenMetrics(
                widthPx = bounds.width(),
                heightPx = bounds.height(),
                density = density,
                orientation = orientation,
                safeArea = AndroidInsets(
                    leftPx = insets.left,
                    topPx = insets.top,
                    rightPx = insets.right,
                    bottomPx = insets.bottom,
                ),
            )
        }

        @Suppress("DEPRECATION")
        val displayMetrics = resources.displayMetrics
        return AndroidScreenMetrics(
            widthPx = displayMetrics.widthPixels,
            heightPx = displayMetrics.heightPixels,
            density = density,
            orientation = orientation,
        )
    }

    private fun currentPermissions(): PermissionCapabilityState =
        PermissionCapabilityState(
            microphoneGranted = isGranted(Manifest.permission.RECORD_AUDIO),
            cameraGranted = isGranted(Manifest.permission.CAMERA),
            notificationsGranted = Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
                isGranted(Manifest.permission.POST_NOTIFICATIONS),
        )

    private fun currentHardware(): AndroidHardwareCapabilities =
        AndroidHardwareCapabilities(
            touchSupported = hasFeature(PackageManager.FEATURE_TOUCHSCREEN),
            maxTouchPoints = maxTouchPoints(),
            physicalKeyboard = appContext.resources.configuration.keyboard == Configuration.KEYBOARD_QWERTY,
            pointerType = if (hasFeature(PackageManager.FEATURE_TOUCHSCREEN)) "touch" else "none",
            pointerHover = false,
            audioOutput = hasFeature(PackageManager.FEATURE_AUDIO_OUTPUT),
            microphone = hasFeature(PackageManager.FEATURE_MICROPHONE),
            frontCamera = hasFeature(PackageManager.FEATURE_CAMERA_FRONT),
            backCamera = hasFeature(PackageManager.FEATURE_CAMERA),
            accelerometer = hasFeature(PackageManager.FEATURE_SENSOR_ACCELEROMETER),
            gyroscope = hasFeature(PackageManager.FEATURE_SENSOR_GYROSCOPE),
            compass = hasFeature(PackageManager.FEATURE_SENSOR_COMPASS),
            ambientLight = hasFeature(PackageManager.FEATURE_SENSOR_LIGHT),
            proximity = hasFeature(PackageManager.FEATURE_SENSOR_PROXIMITY),
            gps = hasFeature(PackageManager.FEATURE_LOCATION_GPS),
            wifiSignalStrength = hasFeature(PackageManager.FEATURE_WIFI),
            usbHost = hasFeature(PackageManager.FEATURE_USB_HOST),
            nfc = hasFeature(PackageManager.FEATURE_NFC),
            haptics = hasVibrator(),
        )

    private fun currentPower(): PowerCapabilityState {
        val battery = appContext.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val level = battery?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
        val scale = battery?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
        val status = battery?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
        val charging = status == BatteryManager.BATTERY_STATUS_CHARGING ||
            status == BatteryManager.BATTERY_STATUS_FULL
        val normalizedLevel = if (level >= 0 && scale > 0) level.toFloat() / scale.toFloat() else 0f
        return PowerCapabilityState(
            batteryLevel = normalizedLevel,
            charging = charging,
            keepAwakeSupported = true,
        )
    }

    private fun maxTouchPoints(): Int =
        when {
            hasFeature(PackageManager.FEATURE_TOUCHSCREEN_MULTITOUCH_JAZZHAND) -> 5
            hasFeature(PackageManager.FEATURE_TOUCHSCREEN_MULTITOUCH_DISTINCT) -> 2
            hasFeature(PackageManager.FEATURE_TOUCHSCREEN_MULTITOUCH) -> 2
            hasFeature(PackageManager.FEATURE_TOUCHSCREEN) -> 1
            else -> 0
        }

    private fun hasFeature(feature: String): Boolean = packageManager.hasSystemFeature(feature)

    private fun hasVibrator(): Boolean =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            appContext.getSystemService(VibratorManager::class.java).defaultVibrator.hasVibrator()
        } else {
            @Suppress("DEPRECATION")
            (appContext.getSystemService(Context.VIBRATOR_SERVICE) as? Vibrator)?.hasVibrator() == true
        }

    private fun isGranted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(appContext, permission) == PackageManager.PERMISSION_GRANTED

    private companion object {
        fun defaultDeviceName(context: Context): String {
            val userVisibleName = runCatching {
                Settings.Global.getString(context.contentResolver, Settings.Global.DEVICE_NAME)
            }.getOrNull()
            return userVisibleName?.takeIf { it.isNotBlank() }
                ?: listOf(Build.MANUFACTURER, Build.MODEL)
                    .filter { it.isNotBlank() }
                    .joinToString(" ")
                    .ifBlank { "Android terminal" }
        }
    }
}
