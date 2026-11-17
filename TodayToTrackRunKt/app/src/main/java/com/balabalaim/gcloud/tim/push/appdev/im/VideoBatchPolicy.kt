package com.balabalaim.gcloud.tim.push.appdev.im

import kotlinx.coroutines.ensureActive

object VideoBatchPolicy {
    const val MAX_COUNT = 9
    const val MAX_FILE_BYTES = 500L * 1024 * 1024
    const val MAX_TOTAL_BYTES = 1024L * 1024 * 1024
    fun canRetry(status: String) = status == "failed"
    fun validateSize(size: Long, otherBytes: Long, serverLimit: Long) {
        require(size > 0) { "视频文件为空" }
        require(size <= minOf(MAX_FILE_BYTES, serverLimit)) { "视频超过当前单文件上限" }
        require(size <= MAX_TOTAL_BYTES - otherBytes) { "本批视频总大小不能超过1GB" }
    }
}

/** A failed item never prevents the other selected videos from being attempted. */
suspend fun <T> runIndependentVideoItems(items: List<T>, active: () -> Boolean,
                                         send: suspend (T) -> Unit, failed: suspend (T, Throwable) -> Unit) {
    for (item in items) {
        if (!active()) break
        try { send(item) } catch (error: Throwable) { kotlinx.coroutines.currentCoroutineContext().ensureActive(); failed(item, error) }
    }
}
