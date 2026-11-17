package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.Context
import android.media.AudioAttributes
import android.media.MediaPlayer
import java.util.concurrent.atomic.AtomicReference

/** 聊天语音消息播放器：再次点击同一条则停止。 */
object LinLanVoiceMessagePlayer {
    private val playerRef = AtomicReference<MediaPlayer?>(null)
    @Volatile
    var currentMsgId: String? = null
        private set

    fun isPlaying(msgId: String?): Boolean =
        !msgId.isNullOrBlank() && currentMsgId == msgId && playerRef.get() != null

    fun toggle(context: Context, msgId: String, pathOrUrl: String) {
        if (msgId.isBlank() || pathOrUrl.isBlank()) return
        if (currentMsgId == msgId) {
            stop()
            return
        }
        stop()
        val appCtx = context.applicationContext
        val player = MediaPlayer()
        try {
            player.setAudioAttributes(
                AudioAttributes.Builder()
                    .setUsage(AudioAttributes.USAGE_MEDIA)
                    .setContentType(AudioAttributes.CONTENT_TYPE_SPEECH)
                    .build(),
            )
            player.setDataSource(pathOrUrl)
            player.setOnCompletionListener { stop() }
            player.setOnErrorListener { _, _, _ ->
                stop()
                true
            }
            player.setOnPreparedListener {
                currentMsgId = msgId
                playerRef.set(player)
                it.start()
            }
            player.prepareAsync()
            // 先占住引用，避免 prepare 期间被 GC；真正 currentMsgId 在 onPrepared
            playerRef.set(player)
            currentMsgId = msgId
        } catch (_: Exception) {
            runCatching { player.release() }
            if (playerRef.get() === player) playerRef.set(null)
            currentMsgId = null
            throw IllegalStateException("语音播放失败")
        }
    }

    fun stop() {
        val player = playerRef.getAndSet(null)
        currentMsgId = null
        if (player == null) return
        runCatching {
            if (player.isPlaying) player.stop()
        }
        runCatching { player.release() }
    }
}
