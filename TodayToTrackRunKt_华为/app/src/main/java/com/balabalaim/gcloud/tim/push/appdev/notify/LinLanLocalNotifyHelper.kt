package com.balabalaim.gcloud.tim.push.appdev.notify

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.media.AudioAttributes
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.balabalaim.gcloud.tim.push.appdev.MainActivity
import com.balabalaim.gcloud.tim.push.appdev.R
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.tencent.imsdk.v2.V2TIMMessage

object LinLanLocalNotifyHelper {
    const val CHANNEL_MESSAGE = "msg_banner"
    const val CHANNEL_MESSAGE_SILENT = "msg_banner_silent"
    const val CHANNEL_CALL = "call_banner"

    const val EXTRA_PEER_ID = "notify_peer_id"
    const val EXTRA_IS_GROUP = "notify_is_group"
    const val EXTRA_TITLE = "notify_title"
    const val EXTRA_IS_CALL = "notify_is_call"

    fun ensureChannels(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = context.getSystemService(NotificationManager::class.java) ?: return
        val soundUri = LinLanMessageAlertCenter.messageSoundUri(context)
        val attrs = AudioAttributes.Builder()
            .setUsage(AudioAttributes.USAGE_NOTIFICATION)
            .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
            .build()

        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_MESSAGE, "消息通知", NotificationManager.IMPORTANCE_HIGH).apply {
                description = "后台收到新消息时的横幅提醒"
                setSound(soundUri, attrs)
                enableVibration(true)
            },
        )
        nm.createNotificationChannel(
            NotificationChannel(
                CHANNEL_MESSAGE_SILENT,
                "消息通知（静音）",
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                description = "后台消息横幅（无声音）"
                setSound(null, null)
                enableVibration(true)
            },
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_CALL, "来电通知", NotificationManager.IMPORTANCE_HIGH).apply {
                description = "后台来电提醒"
                setSound(
                    android.net.Uri.parse(
                        "android.resource://${context.packageName}/${R.raw.phone_ringing}",
                    ),
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build(),
                )
                enableVibration(true)
            },
        )
    }

    fun canPostNotifications(context: Context): Boolean {
        if (Build.VERSION.SDK_INT < 33) return true
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
    }

    fun showMessageNotification(
        context: Context,
        message: V2TIMMessage,
        withSound: Boolean,
    ) {
        if (!canPostNotifications(context)) return
        val preview = LinLanImManager.messagePreview(message).ifBlank { "新消息" }
        val title = message.nickName?.takeIf { it.isNotBlank() }
            ?: message.friendRemark?.takeIf { it.isNotBlank() }
            ?: message.nameCard?.takeIf { it.isNotBlank() }
            ?: message.sender
            ?: LinLanAppConstants.APP_NAME
        val isGroup = !message.groupID.isNullOrBlank()
        val peerId = if (isGroup) {
            message.groupID.orEmpty()
        } else {
            message.sender.orEmpty()
        }
        if (peerId.isBlank()) return

        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP or
                Intent.FLAG_ACTIVITY_SINGLE_TOP
            putExtra(EXTRA_PEER_ID, peerId)
            putExtra(EXTRA_IS_GROUP, isGroup)
            putExtra(EXTRA_TITLE, title)
            putExtra(EXTRA_IS_CALL, false)
        }
        val requestCode = (message.msgID ?: peerId).hashCode()
        val pending = PendingIntent.getActivity(
            context,
            requestCode,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val channel = if (withSound) CHANNEL_MESSAGE else CHANNEL_MESSAGE_SILENT
        val notification = NotificationCompat.Builder(context, channel)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(title)
            .setContentText(preview)
            .setStyle(NotificationCompat.BigTextStyle().bigText(preview))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setAutoCancel(true)
            .setContentIntent(pending)
            .setVisibility(NotificationCompat.VISIBILITY_PRIVATE)
            .build()

        if (Build.VERSION.SDK_INT >= 33 && ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS,
            ) != PackageManager.PERMISSION_GRANTED
        ) return
        runCatching {
            NotificationManagerCompat.from(context)
                .notify(requestCode, notification)
        }
    }

    fun showIncomingCallNotification(
        context: Context,
        callerName: String,
        isVideo: Boolean,
    ) {
        if (!canPostNotifications(context)) return
        val body = if (isVideo) "邀请你视频通话" else "邀请你语音通话"
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP or
                Intent.FLAG_ACTIVITY_SINGLE_TOP
            putExtra(EXTRA_IS_CALL, true)
        }
        val pending = PendingIntent.getActivity(
            context,
            "call".hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(context, CHANNEL_CALL)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(callerName.ifBlank { LinLanAppConstants.APP_NAME })
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setAutoCancel(true)
            .setContentIntent(pending)
            .build()
        if (Build.VERSION.SDK_INT >= 33 && ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS,
            ) != PackageManager.PERMISSION_GRANTED
        ) return
        runCatching {
            NotificationManagerCompat.from(context).notify("call".hashCode(), notification)
        }
    }
}
