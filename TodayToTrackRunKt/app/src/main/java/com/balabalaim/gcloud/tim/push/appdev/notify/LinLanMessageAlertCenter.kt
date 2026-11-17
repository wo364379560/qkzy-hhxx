package com.balabalaim.gcloud.tim.push.appdev.notify

import android.app.Application
import android.content.Context
import android.media.AudioAttributes
import android.media.MediaPlayer
import android.net.Uri
import androidx.lifecycle.DefaultLifecycleObserver
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.ProcessLifecycleOwner
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.R
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.tencent.imsdk.v2.V2TIMMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/**
 * 前台播放消息提示音；后台发本地横幅。会话免打扰 / 全局开关生效。
 */
object LinLanMessageAlertCenter {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val soundMutex = Mutex()
    private var mediaPlayer: MediaPlayer? = null
    private var started = false
    @Volatile
    var appInForeground: Boolean = true
        private set
    @Volatile
    var activeConversationId: String? = null

    fun start(app: Application) {
        if (started) return
        started = true
        LinLanLocalNotifyHelper.ensureChannels(app)
        ProcessLifecycleOwner.get().lifecycle.addObserver(object : DefaultLifecycleObserver {
            override fun onStart(owner: LifecycleOwner) {
                appInForeground = true
            }

            override fun onStop(owner: LifecycleOwner) {
                appInForeground = false
            }
        })
        scope.launch {
            LinLanImManager.incomingMessage.collectLatest { msg ->
                handleIncoming(app.applicationContext, msg)
            }
        }
    }

    fun setActiveConversation(conversationId: String?) {
        activeConversationId = conversationId?.trim()?.takeIf { it.isNotEmpty() }
    }

    private suspend fun handleIncoming(context: Context, msg: V2TIMMessage) {
        if (msg.isSelf) return
        if (msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_GROUP_TIPS) return
        if (!LinLanAppContainer.sessionStore().currentSession().isLoggedIn) return

        val muted = runCatching { LinLanImManager.isMessageMuted(msg) }.getOrDefault(false)
        val prefs = LinLanAppContainer.appPrefs()
        val convId = LinLanImManager.conversationIdOf(msg)
        val inActiveChat = !activeConversationId.isNullOrBlank() &&
            activeConversationId == convId

        if (appInForeground) {
            if (!muted && prefs.soundNotifyEnabled && !inActiveChat) {
                playMessageSound(context)
            }
            return
        }

        // 后台：横幅（免打扰不弹）
        if (muted) return
        if (!prefs.messageNotifyEnabled) return
        LinLanLocalNotifyHelper.showMessageNotification(
            context = context,
            message = msg,
            withSound = prefs.soundNotifyEnabled,
        )
    }

    private suspend fun playMessageSound(context: Context) = soundMutex.withLock {
        runCatching {
            mediaPlayer?.release()
            mediaPlayer = MediaPlayer.create(context, R.raw.new_msg)?.apply {
                setAudioAttributes(
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build(),
                )
                setOnCompletionListener {
                    it.release()
                    if (mediaPlayer === it) mediaPlayer = null
                }
                start()
            }
        }
    }

    fun messageSoundUri(context: Context): Uri =
        Uri.parse("android.resource://${context.packageName}/${R.raw.new_msg}")
}
