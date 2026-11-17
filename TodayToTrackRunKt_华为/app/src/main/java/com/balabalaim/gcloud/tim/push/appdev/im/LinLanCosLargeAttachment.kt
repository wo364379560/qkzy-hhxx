package com.balabalaim.gcloud.tim.push.appdev.im

import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanApiClient
import com.tencent.imsdk.v2.V2TIMMessage
import org.json.JSONObject
import java.io.File

/**
 * 聊天大文件 COS 上传开关工具。
 *
 * - [ENABLED] = false：走腾讯 IM 原生文件/视频上传
 * - [ENABLED] = true：超过 100MB 时走后端 COS（需配置 COS_CHAT_*）
 *
 * COS 申请好后改成 true 即可启用。
 */
object LinLanCosLargeAttachment {
    const val ENABLED = true

    const val NATIVE_LIMIT_BYTES = 100L * 1024L * 1024L
    private const val MARKER_PREFIX = "CN_ATTACHMENT_V1:"

    fun shouldRouteToCos(fileSize: Long): Boolean {
        if (!ENABLED) return false
        return fileSize > NATIVE_LIMIT_BYTES
    }

    fun fileSize(path: String): Long {
        val file = File(path)
        return if (file.exists()) file.length() else 0L
    }

    fun contentType(fileName: String): String {
        val lower = fileName.lowercase()
        return when {
            lower.endsWith(".mp4") -> "video/mp4"
            lower.endsWith(".mov") -> "video/quicktime"
            lower.endsWith(".m4v") -> "video/x-m4v"
            lower.endsWith(".webm") -> "video/webm"
            lower.endsWith(".3gp") || lower.endsWith(".3gpp") -> "video/3gpp"
            lower.endsWith(".avi") -> "video/x-msvideo"
            lower.endsWith(".mkv") -> "video/x-matroska"
            lower.endsWith(".pdf") -> "application/pdf"
            lower.endsWith(".zip") -> "application/zip"
            lower.endsWith(".png") -> "image/png"
            lower.endsWith(".jpg") || lower.endsWith(".jpeg") -> "image/jpeg"
            else -> "application/octet-stream"
        }
    }

    fun isVideoFileName(fileName: String): Boolean {
        return contentType(fileName).startsWith("video/")
    }

    fun conversationId(receiver: String?, groupId: String?): String {
        return when {
            !groupId.isNullOrBlank() -> "group_$groupId"
            !receiver.isNullOrBlank() -> "c2c_$receiver"
            else -> ""
        }
    }

    fun buildTextMessage(
        fileId: String,
        fileName: String,
        fileSize: Long,
        contentType: String,
        isVideo: Boolean,
    ): String {
        val title = if (isVideo) {
            "视频文件已上传，点击卡片预览或下载"
        } else {
            "文件已上传，点击卡片预览或下载"
        }
        val sizeText = formatSize(fileSize)
        val summary = if (sizeText.isNotEmpty()) "$fileName ($sizeText)" else fileName
        val payload = JSONObject()
            .put("file_id", fileId)
            .put("name", fileName)
            .put("size", fileSize)
            .put("is_video", isVideo)
            .put("content_type", contentType)
        return "$title\n$summary\n$MARKER_PREFIX$payload"
    }

    fun formatSize(size: Long): String {
        if (size <= 0) return ""
        val kb = 1024.0
        val mb = kb * 1024
        val gb = mb * 1024
        return when {
            size >= gb -> String.format("%.1fGB", size / gb)
            size >= mb -> String.format("%.1fMB", size / mb)
            else -> String.format("%.1fKB", size / kb)
        }
    }

    suspend fun uploadAndSend(
        api: LinLanApiClient = LinLanAppContainer.apiClient(),
        path: String,
        fileName: String,
        receiver: String?,
        groupId: String?,
        isVideo: Boolean = false,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
        isActive: () -> Boolean = { true },
    ): V2TIMMessage {
        val file = File(path)
        if (!file.exists()) throw Exception("文件不存在")
        val size = file.length()
        val conversationID = conversationId(receiver, groupId)
        if (conversationID.isBlank()) throw Exception("会话信息不完整")

        val type = contentType(fileName)
        val forceVideo = isVideo || type.startsWith("video/")
        val owner = LinLanAppContainer.sessionStore().currentSession().userId
        val presign = api.uploadAttachmentMultipart(file, fileName, conversationID, type, forceVideo, onProgress = { onProgress((it * 99).toInt()) }, isActive = isActive)
        if (LinLanAppContainer.sessionStore().currentSession().userId != owner) throw kotlinx.coroutines.CancellationException("账号已切换")

        val text = buildTextMessage(
            fileId = presign.fileId,
            fileName = fileName.ifBlank { presign.fileName },
            fileSize = size,
            contentType = presign.contentType.ifBlank { type },
            isVideo = forceVideo || presign.isVideo,
        )
        if (!isActive()) throw LinLanSendFailure("已取消，未发送", notDispatched = true)
        return LinLanImManager.sendTextMessage(text, receiver, groupId, onPrepared = onPrepared, onProgress = onProgress)
    }
}
