package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.content.Context
import android.net.Uri
import java.io.File

object LinLanUriCache {
    fun copyToCache(context: Context, uri: Uri, prefix: String = "media_"): String? = runCatching {
        val dir = File(context.cacheDir, "chat_media").also { it.mkdirs() }
        val out = File(dir, "$prefix${System.currentTimeMillis()}")
        context.contentResolver.openInputStream(uri)?.use { input ->
            out.outputStream().use { output -> input.copyTo(output) }
        } ?: return null
        out.absolutePath
    }.getOrNull()
}
