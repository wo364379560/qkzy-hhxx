package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.Context
import android.media.MediaRecorder
import android.os.Build
import java.io.File
import kotlin.math.ln
import kotlin.math.max
import kotlin.math.min

class LinLanVoiceRecorder(private val context: Context) {
    data class Result(val path: String, val durationSec: Int)

    private var recorder: MediaRecorder? = null
    private var outputPath: String? = null
    private var startedAtMs: Long = 0L

    fun start(): String {
        stopAndDiscard()
        val file = File(context.cacheDir, "voice_${System.currentTimeMillis()}.m4a")
        val path = file.absolutePath
        val mediaRecorder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            MediaRecorder(context)
        } else {
            @Suppress("DEPRECATION")
            MediaRecorder()
        }
        mediaRecorder.setAudioSource(MediaRecorder.AudioSource.MIC)
        mediaRecorder.setOutputFormat(MediaRecorder.OutputFormat.MPEG_4)
        mediaRecorder.setAudioEncoder(MediaRecorder.AudioEncoder.AAC)
        mediaRecorder.setAudioSamplingRate(16_000)
        mediaRecorder.setAudioEncodingBitRate(32_000)
        mediaRecorder.setAudioChannels(1)
        mediaRecorder.setOutputFile(path)
        mediaRecorder.prepare()
        mediaRecorder.start()
        recorder = mediaRecorder
        outputPath = path
        startedAtMs = System.currentTimeMillis()
        return path
    }

    /** 约 0...1，供录音动画使用。 */
    fun currentLevel(): Float {
        val r = recorder ?: return 0.08f
        val amp = runCatching { r.maxAmplitude }.getOrDefault(0)
        if (amp <= 0) return 0.12f
        val normalized = (ln(amp.toDouble() + 1.0) / ln(32768.0)).toFloat()
        return max(0.08f, min(1f, normalized))
    }

    fun elapsedSec(): Int {
        if (startedAtMs <= 0L) return 0
        return ((System.currentTimeMillis() - startedAtMs) / 1000L).toInt()
    }

    fun stop(): Result {
        val path = outputPath ?: error("未在录音")
        val duration = max(1, elapsedSec())
        runCatching {
            recorder?.stop()
        }
        recorder?.release()
        recorder = null
        outputPath = null
        startedAtMs = 0L
        if (duration < MIN_DURATION_SEC) {
            File(path).delete()
            error("说话时间太短")
        }
        return Result(path, min(duration, MAX_DURATION_SEC))
    }

    fun stopAndDiscard() {
        runCatching { recorder?.stop() }
        recorder?.release()
        recorder = null
        outputPath?.let { File(it).delete() }
        outputPath = null
        startedAtMs = 0L
    }

    companion object {
        const val MIN_DURATION_SEC = 1
        const val MAX_DURATION_SEC = 60
    }
}
