package com.curtcox.terminals.android.platform

import android.annotation.SuppressLint
import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.curtcox.terminals.android.R

fun interface AndroidNotificationDelivery {
    fun deliver(title: String, body: String)

    companion object {
        fun none(): AndroidNotificationDelivery = AndroidNotificationDelivery { _, _ -> }
    }
}

class StatusBarAndroidNotificationDelivery(
    private val context: Context,
) : AndroidNotificationDelivery {
    @SuppressLint("MissingPermission")
    override fun deliver(title: String, body: String) {
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            return
        }

        ensureChannel()
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentTitle(title.ifBlank { context.getString(R.string.app_name) })
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setAutoCancel(true)
            .build()

        NotificationManagerCompat.from(context).notify(nextNotificationId(), notification)
    }

    private fun ensureChannel() {
        if (Build.VERSION.SDK_INT < 26) return
        val manager = context.getSystemService(NotificationManager::class.java)
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return
        manager.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                context.getString(R.string.notification_channel_terminal_events),
                NotificationManager.IMPORTANCE_DEFAULT,
            ),
        )
    }

    private fun nextNotificationId(): Int = (System.currentTimeMillis() and 0x7FFFFFFF).toInt()

    private companion object {
        const val CHANNEL_ID = "terminal_events"
    }
}
