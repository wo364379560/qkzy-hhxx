package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.content.Context
import android.graphics.Bitmap
import android.media.MediaMetadataRetriever
import android.net.Uri
import android.provider.OpenableColumns
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import java.io.File
import kotlin.coroutines.coroutineContext

data class PickedVideo(val uri: Uri, val name: String, val size: Long)
data class PreparedVideo(val file: File, val snapshot: File, val seconds: Int, val type: String) {
    fun cleanup() { file.delete(); snapshot.delete() }
}

object LinLanVideoFiles {
    suspend fun describe(context: Context, uri: Uri, index: Int): PickedVideo = withContext(Dispatchers.IO) {
        var name = "视频${index + 1}.mp4"; var size = -1L
        context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE), null, null, null)?.use { cursor ->
            if (cursor.moveToFirst()) {
                val n = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME); val s = cursor.getColumnIndex(OpenableColumns.SIZE)
                if (n >= 0) name = cursor.getString(n)?.take(180)?.ifBlank { name } ?: name
                if (s >= 0 && !cursor.isNull(s)) size = cursor.getLong(s)
            }
        }
        PickedVideo(uri, name, size)
    }

    suspend fun prepare(context: Context, picked: PickedVideo, maxBytes: Long, active: () -> Boolean): PreparedVideo = withContext(Dispatchers.IO) {
        val file = File.createTempFile("rui_video_", ".video", context.cacheDir)
        val snapshot = File.createTempFile("rui_video_cover_", ".jpg", context.cacheDir)
        val retriever = MediaMetadataRetriever()
        try {
            context.contentResolver.openInputStream(picked.uri)?.use { input ->
                file.outputStream().use { output ->
                    val buffer = ByteArray(256 * 1024); var total = 0L
                    while (true) {
                        coroutineContext.ensureActive(); if (!active()) throw kotlinx.coroutines.CancellationException("已取消")
                        val count = input.read(buffer); if (count < 0) break
                        total += count; require(total <= maxBytes) { "视频超过本批次剩余容量或单文件上限" }
                        output.write(buffer, 0, count)
                    }
                }
            } ?: error("视频读取失败，请重新选择")
            require(file.length() > 0) { "视频文件为空" }
            retriever.setDataSource(file.path)
            val seconds = ((retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)?.toLongOrNull() ?: error("无法读取视频时长")) / 1000).toInt().coerceAtLeast(1)
            val frame = if (android.os.Build.VERSION.SDK_INT >= 27) retriever.getScaledFrameAtTime(0, MediaMetadataRetriever.OPTION_CLOSEST_SYNC, 640, 640)
                else retriever.getFrameAtTime(0, MediaMetadataRetriever.OPTION_CLOSEST_SYNC)
            requireNotNull(frame) { "视频封面生成失败" }
            try { snapshot.outputStream().use { check(frame.compress(Bitmap.CompressFormat.JPEG, 85, it)) { "视频封面编码失败" } } }
            finally { frame.recycle() }
            val type = when (retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_MIMETYPE)) {
                "video/quicktime" -> "mov"; "video/webm" -> "webm"; "video/3gpp" -> "3gp"; else -> "mp4"
            }
            PreparedVideo(file, snapshot, seconds, type)
        } catch (error: Throwable) { file.delete(); snapshot.delete(); throw error }
        finally { retriever.release() }
    }
}
