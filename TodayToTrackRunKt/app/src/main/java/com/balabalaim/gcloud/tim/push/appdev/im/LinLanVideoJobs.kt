package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.Context
import android.net.Uri
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVideoFiles
import com.balabalaim.gcloud.tim.push.appdev.ui.common.PickedVideo
import com.tencent.imsdk.v2.V2TIMManager
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

data class VideoItemResult(val index: Int, val name: String, val size: Long = -1, val status: String = "checking", val progress: Int = 0, val error: String = "", val sdkID: String = "")
data class VideoBatchResult(val id: String, val owner: String, val conversationID: String, val items: List<VideoItemResult>,
                            val running: Boolean = false, val confirmed: Boolean = false, val fileLimit: Long = 100L * 1024 * 1024)

object LinLanVideoJobs {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val mutable = MutableStateFlow<VideoBatchResult?>(null)
    val current = mutable.asStateFlow()
    private data class Plan(val context: Context, val id: String, val owner: String, val appID: Int, val sessionGeneration: Long, val receiver: String?, val group: String?, val videos: MutableList<PickedVideo>, val allocated: MutableMap<Int, Long> = mutableMapOf(), @Volatile var stopped: Boolean = false)
    private var plan: Plan? = null

    fun select(context: Context, uris: List<Uri>, receiver: String?, group: String?): Boolean {
        if (mutable.value?.running == true && plan?.let(::active) == true || uris.isEmpty() || uris.size > VideoBatchPolicy.MAX_COUNT) return false
        val owner = V2TIMManager.getInstance().loginUser.orEmpty(); if (owner.isBlank()) return false
        plan?.stopped = true
        val id = UUID.randomUUID().toString()
        val selected = Plan(context.applicationContext, id, owner, LinLanRuntimeConfig.sdkAppId, LinLanAppContainer.sessionStore().sessionGeneration(), receiver, group, mutableListOf())
        plan = selected
        mutable.value = VideoBatchResult(id, owner, LinLanCosLargeAttachment.conversationId(receiver, group), uris.mapIndexed { i, _ -> VideoItemResult(i, "视频${i + 1}") })
        scope.launch {
            var limit = 100L * 1024 * 1024
            runCatching { LinLanAppContainer.apiClient().getChatAttachmentUploadSettings() }.onSuccess { settings ->
                if (settings.optBoolean("enabled", true)) limit = settings.optLong("max_size_mb", 300).coerceIn(1, 500) * 1024 * 1024
            }
            if (!active(selected)) return@launch
            mutable.value = mutable.value?.copy(fileLimit = limit)
            uris.forEachIndexed { i, uri ->
                try {
                    val video = LinLanVideoFiles.describe(selected.context, uri, i)
                    if (!active(selected)) return@launch
                    selected.videos += video
                    update(selected, i) { it.copy(name = video.name, size = video.size, status = "ready") }
                } catch (error: Exception) {
                    selected.videos += PickedVideo(uri, "视频${i + 1}", -1)
                    update(selected, i) { it.copy(status = "failed", error = "读取视频资料失败，请重新选择") }
                }
            }
        }
        return true
    }

    private fun ownsSession(value: Plan) = V2TIMManager.getInstance().loginUser == value.owner && LinLanRuntimeConfig.sdkAppId == value.appID && LinLanAppContainer.sessionStore().sessionGeneration() == value.sessionGeneration
    private fun active(value: Plan) = plan === value && !value.stopped && ownsSession(value)
    private fun update(value: Plan, index: Int, change: (VideoItemResult) -> VideoItemResult) {
        val current = mutable.value ?: return
        if (current.id != value.id || !ownsSession(value)) return
        mutable.value = current.copy(items = current.items.map { if (it.index == index) change(it) else it })
    }

    fun cancel() {
        val value = plan ?: return; value.stopped = true
        mutable.value = mutable.value?.copy(items = mutable.value!!.items.map { if (it.status in setOf("checking", "ready", "pending")) it.copy(status = "cancelled", error = "已取消，未发送") else it })
    }

    fun send(retry: Boolean = false) {
        val value = plan ?: return; val job = mutable.value ?: return
        if (job.running || job.items.any { it.status == "checking" } || !ownsSession(value)) return
        value.stopped = false
        val selected = job.items.filter { if (retry) VideoBatchPolicy.canRetry(it.status) else it.status == "ready" }.map { it.index }
        if (selected.isEmpty()) return
        mutable.value = job.copy(running = true, confirmed = true)
        scope.launch {
            try {
                runIndependentVideoItems(selected, { active(value) }, send = { index ->
                    var dispatched = false
                    var prepared: com.balabalaim.gcloud.tim.push.appdev.ui.common.PreparedVideo? = null
                    try {
                        val otherBytes = value.allocated.filterKeys { it != index }.values.sum()
                        val limit = minOf(job.fileLimit, VideoBatchPolicy.MAX_TOTAL_BYTES - otherBytes)
                        update(value, index) { it.copy(status = "preparing", progress = 0, error = "") }
                        val ready = LinLanVideoFiles.prepare(value.context, value.videos[index], limit) { active(value) }
                        prepared = ready
                        VideoBatchPolicy.validateSize(ready.file.length(), otherBytes, job.fileLimit)
                        value.allocated[index] = ready.file.length()
                        update(value, index) { it.copy(size = ready.file.length(), status = "uploading") }
                        val sent = LinLanImManager.sendVideoMessage(ready.file.path, ready.type, ready.seconds, ready.snapshot.path, value.receiver, value.group,
                            onPrepared = { if (!active(value)) throw LinLanSendFailure("已取消，未发送", notDispatched = true); dispatched = true; update(value, index) { it.copy(status = "sending") } },
                            onProgress = { progress -> scope.launch { update(value, index) { if (it.status in setOf("uploading", "sending")) it.copy(progress = progress) else it } } },
                            fileName = value.videos[index].name.substringBeforeLast('.', value.videos[index].name) + "." + ready.type,
                            isActive = { active(value) })
                        update(value, index) { it.copy(status = "success", progress = 100, sdkID = sent.msgID.orEmpty()) }
                    } catch (error: Throwable) {
                        val unknown = (error as? LinLanSendFailure)?.unknown == true || dispatched && error !is LinLanSendFailure
                        update(value, index) { it.copy(status = if (unknown) "unknown" else if (!active(value)) "cancelled" else "failed", error = if (unknown) "发送结果待确认，不会自动重试" else error.message ?: "视频发送失败") }
                    } finally { prepared?.cleanup() }
                }, failed = { index, _ -> update(value, index) { it.copy(status = "failed", error = "视频处理失败") } })
            } finally {
                if (mutable.value?.id == value.id) mutable.value = mutable.value!!.copy(running = false, items = mutable.value!!.items.map {
                    if (!active(value) && it.status in setOf("ready", "pending")) it.copy(status = "cancelled", error = "已停止，未发送") else it
                })
            }
        }
    }
}
