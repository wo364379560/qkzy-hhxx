package com.balabalaim.gcloud.tim.push.appdev.im

import kotlinx.coroutines.CancellationException

/** History never replaces a newer live/outbox/revoked representation of an ID. */
fun <T> mergeHistoryTimeline(current: List<T>, incoming: List<T>, key: (T) -> String, timestamp: (T) -> Long,
                             preferCurrent: Boolean = true, removedIDs: Set<String> = emptySet(), sequence: (T) -> Long = { 0L }): List<T> {
    val rows = linkedMapOf<String, T>()
    current.forEach { rows[key(it)] = it }
    incoming.forEach { value ->
        val id = key(value)
        if (id !in removedIDs && (!preferCurrent || id !in rows)) rows[id] = value
    }
    return rows.values.sortedWith(compareBy<T> { timestamp(it) }.thenBy { sequence(it).takeIf { value -> value > 0 } ?: Long.MAX_VALUE })
}

/** Raw pagination survives pages consisting only of hidden control messages. */
class RawHistoryPager<T>(private val pageSize: Int, private val key: (T) -> String,
                         private val visible: (T) -> Boolean, private val fetch: suspend (T?) -> List<T>) {
    var cursor: T? = null; private set
    var hasMore = true; private set

    suspend fun load(latest: Boolean, active: () -> Boolean): List<T> {
        if (!latest && !hasMore) return emptyList()
        var rawCursor = if (latest) null else cursor
        var exhausted: Boolean
        val result = mutableListOf<T>()
        var pages = 0
        do {
            if (!active()) throw CancellationException("历史请求作用域已变化")
            val raw = fetch(rawCursor)
            if (!active()) throw CancellationException("历史请求作用域已变化")
            val next = raw.lastOrNull()
            exhausted = raw.size < pageSize || next == null
            if (!exhausted && rawCursor != null && key(next!!) == key(rawCursor)) error("历史游标未推进，请重试")
            rawCursor = next ?: rawCursor
            result += raw.filter(visible).asReversed()
            pages++
        } while (!exhausted && result.isEmpty() && pages < 4)
        if (!latest || cursor == null) { cursor = rawCursor; hasMore = !exhausted }
        return result
    }
}

fun matchesChatConversation(conversationID: String, groupID: String?, userID: String?, sender: String?): Boolean {
    if (conversationID.startsWith("group_")) return conversationID.removePrefix("group_") == groupID
    if (!groupID.isNullOrBlank()) return false
    val peer = conversationID.removePrefix("c2c_")
    return peer.isNotBlank() && (userID == peer || sender == peer)
}
