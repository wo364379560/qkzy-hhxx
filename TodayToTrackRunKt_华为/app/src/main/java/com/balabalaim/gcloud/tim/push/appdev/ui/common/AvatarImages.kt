package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import java.io.File
import kotlin.coroutines.coroutineContext

/** Bounded decode and actual JPEG encoding; no image work on the UI thread. */
object LinLanAvatarImages {
    suspend fun prepare(context: Context, uri: Uri): File = withContext(Dispatchers.IO) {
        val source = File.createTempFile("avatar_source_", ".bin", context.cacheDir)
        val output = File.createTempFile("avatar_ready_", ".jpg", context.cacheDir)
        val bitmaps = mutableSetOf<Bitmap>()
        try {
            context.contentResolver.openInputStream(uri)?.use { input ->
                source.outputStream().use { target ->
                    val buffer = ByteArray(32 * 1024)
                    var total = 0L
                    while (true) {
                        coroutineContext.ensureActive()
                        val size = input.read(buffer)
                        if (size < 0) break
                        total += size
                        require(total <= 20L * 1024 * 1024) { "头像原图不能超过20MB" }
                        target.write(buffer, 0, size)
                    }
                }
            } ?: error("无法读取相册图片")
            val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
            BitmapFactory.decodeFile(source.path, bounds)
            require(bounds.outWidth > 0 && bounds.outHeight > 0) { "图片格式无法识别" }
            var sample = 1
            while (maxOf(bounds.outWidth, bounds.outHeight) / sample > 2048) sample *= 2
            val decoded = BitmapFactory.decodeFile(source.path, BitmapFactory.Options().apply { inSampleSize = sample })
                ?: error("图片解码失败")
            bitmaps += decoded
            val orientation = runCatching { ExifInterface(source.path).getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL) }.getOrDefault(ExifInterface.ORIENTATION_NORMAL)
            val matrix = Matrix().apply {
                when (orientation) {
                    ExifInterface.ORIENTATION_FLIP_HORIZONTAL -> setScale(-1f, 1f)
                    ExifInterface.ORIENTATION_ROTATE_180 -> setRotate(180f)
                    ExifInterface.ORIENTATION_FLIP_VERTICAL -> setScale(1f, -1f)
                    ExifInterface.ORIENTATION_TRANSPOSE -> { setRotate(90f); postScale(-1f, 1f) }
                    ExifInterface.ORIENTATION_ROTATE_90 -> setRotate(90f)
                    ExifInterface.ORIENTATION_TRANSVERSE -> { setRotate(270f); postScale(-1f, 1f) }
                    ExifInterface.ORIENTATION_ROTATE_270 -> setRotate(270f)
                }
            }
            val oriented = Bitmap.createBitmap(decoded, 0, 0, decoded.width, decoded.height, matrix, true)
            bitmaps += oriented
            val scale = minOf(1f, 1024f / maxOf(oriented.width, oriented.height))
            val scaled = Bitmap.createScaledBitmap(oriented, maxOf(1, (oriented.width * scale).toInt()), maxOf(1, (oriented.height * scale).toInt()), true)
            bitmaps += scaled
            coroutineContext.ensureActive()
            output.outputStream().use { require(scaled.compress(Bitmap.CompressFormat.JPEG, 85, it)) { "头像编码失败" } }
            output
        } catch (error: Throwable) { output.delete(); throw error }
        finally { source.delete(); bitmaps.forEach { it.recycle() } }
    }
}
