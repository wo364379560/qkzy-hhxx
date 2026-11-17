package com.balabalaim.gcloud.tim.push.appdev.core

import java.io.IOException

/** Pure reliability helpers shared by paging, cache and network code. */
object LinLanReliability {
    const val MAX_FORWARD_MESSAGES = 50

    fun retryDelayMs(attempt: Int): Long = when (attempt.coerceAtLeast(0)) {
        0 -> 250L
        1 -> 500L
        2 -> 1_000L
        else -> 2_000L
    }

    fun shouldRetryRequest(method: String, attempt: Int, error: Throwable): Boolean =
        attempt == 0 && method.equals("GET", ignoreCase = true) && error is IOException

    fun safeCacheName(raw: String, fallback: String = "file"): String {
        val leaf = raw.replace('\\', '/').substringAfterLast('/').trim()
        val clean = leaf.replace(Regex("[^A-Za-z0-9._()\\-\\u4e00-\\u9fa5]"), "_")
            .trim('.', ' ')
            .take(120)
        return clean.ifBlank { fallback }
    }

    fun <T, K> appendDistinct(
        current: List<T>,
        incoming: List<T>,
        keyOf: (T) -> K,
        limit: Int = Int.MAX_VALUE,
    ): List<T> {
        if (limit <= 0) return emptyList()
        val result = ArrayList<T>(minOf(limit, current.size + incoming.size))
        val keys = HashSet<K>(minOf(limit, current.size + incoming.size))
        for (item in current) {
            if (result.size >= limit) break
            if (keys.add(keyOf(item))) result += item
        }
        for (item in incoming) {
            if (result.size >= limit) break
            if (keys.add(keyOf(item))) result += item
        }
        return result
    }
}
