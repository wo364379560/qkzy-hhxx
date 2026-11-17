package com.balabalaim.gcloud.tim.push.appdev.rtc

import android.app.*
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat

class CallForegroundService : Service() {
    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val channel = "ongoing_rtc_call"
        if (Build.VERSION.SDK_INT >= 26) {
            getSystemService(NotificationManager::class.java).createNotificationChannel(
                NotificationChannel(channel, "进行中的通话", NotificationManager.IMPORTANCE_LOW)
            )
        }
        val open = Intent(this, LinLanRtcCallActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_REORDER_TO_FRONT or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        val notification = NotificationCompat.Builder(this, channel)
            .setSmallIcon(android.R.drawable.sym_call_outgoing)
            .setContentTitle("通话进行中").setContentText("点此返回通话")
            .setContentIntent(PendingIntent.getActivity(this, 0, open, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT))
            .setOngoing(true).setOnlyAlertOnce(true).build()
        var types = ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
        if (LinLanRtcCallManager.callContext.value?.mediaType == LinLanRtcCallMediaType.VIDEO) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
        }
        if (Build.VERSION.SDK_INT >= 30) startForeground(41, notification, types)
        else startForeground(41, notification)
        if (LinLanRtcCallManager.callContext.value == null) stopSelf()
        return START_NOT_STICKY
    }
}
