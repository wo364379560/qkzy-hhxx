package com.balabalaim.gcloud.tim.push.appdev.rtc

import android.content.Context
import android.media.AudioAttributes
import android.media.AudioManager
import android.media.MediaPlayer
import android.media.ToneGenerator
import android.os.Handler
import android.os.Looper
import com.balabalaim.gcloud.tim.push.appdev.R

/**
 * 拨打 / 来电铃声：res/raw/phone_dialing、phone_ringing。
 */
object LinLanRtcRingtonePlayer {
    enum class Kind { OUTGOING, INCOMING }

    private val mainHandler = Handler(Looper.getMainLooper())
    private var mediaPlayer: MediaPlayer? = null
    private var tone: ToneGenerator? = null
    private var toneRunnable: Runnable? = null

    fun play(context: Context, kind: Kind) {
        stop()
        val resId = when (kind) {
            Kind.INCOMING -> R.raw.phone_ringing
            Kind.OUTGOING -> R.raw.phone_dialing
        }
        val played = runCatching {
            MediaPlayer.create(context, resId)?.apply {
                setAudioAttributes(
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build(),
                )
                isLooping = true
                start()
                mediaPlayer = this
            } != null
        }.getOrDefault(false)

        if (!played) {
            startToneFallback(kind)
        }
    }

    fun stop() {
        runCatching {
            mediaPlayer?.stop()
            mediaPlayer?.release()
        }
        mediaPlayer = null
        toneRunnable?.let { mainHandler.removeCallbacks(it) }
        toneRunnable = null
        runCatching { tone?.release() }
        tone = null
    }

    private fun startToneFallback(kind: Kind) {
        val toneType = if (kind == Kind.INCOMING) {
            ToneGenerator.TONE_CDMA_ALERT_CALL_GUARD
        } else {
            ToneGenerator.TONE_SUP_RINGTONE
        }
        tone = ToneGenerator(AudioManager.STREAM_VOICE_CALL, 80)
        val task = object : Runnable {
            override fun run() {
                runCatching { tone?.startTone(toneType, 800) }
                mainHandler.postDelayed(this, 1600)
            }
        }
        toneRunnable = task
        mainHandler.post(task)
    }
}
