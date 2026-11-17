package com.balabalaim.gcloud.tim.push.appdev.data.local

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.json.JSONArray
import org.json.JSONObject

data class PinnedMessage(
    val msgId: String,
    val summary: String,
    val sender: String,
)

class LinLanPinnedMessageStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences("pinned_messages", Context.MODE_PRIVATE)
    private val _changed = MutableStateFlow(0)
    val changed: StateFlow<Int> = _changed.asStateFlow()

    fun list(conversationId: String): List<PinnedMessage> {
        val json = prefs.getString(conversationId, null) ?: return emptyList()
        return runCatching {
            val arr = JSONArray(json)
            buildList {
                for (i in 0 until arr.length()) {
                    val o = arr.optJSONObject(i) ?: continue
                    val id = o.optString("msgId")
                    if (id.isBlank()) continue
                    add(
                        PinnedMessage(
                            msgId = id,
                            summary = o.optString("summary"),
                            sender = o.optString("sender"),
                        )
                    )
                }
            }
        }.getOrDefault(emptyList())
    }

    fun add(conversationId: String, pin: PinnedMessage) {
        val current = list(conversationId).toMutableList()
        if (current.any { it.msgId == pin.msgId }) return
        current.add(0, pin)
        persist(conversationId, current)
    }

    fun remove(conversationId: String, msgId: String) {
        persist(conversationId, list(conversationId).filter { it.msgId != msgId })
    }

    fun isPinned(conversationId: String, msgId: String): Boolean =
        list(conversationId).any { it.msgId == msgId }

    private fun persist(conversationId: String, pins: List<PinnedMessage>) {
        val arr = JSONArray()
        pins.forEach { pin ->
            arr.put(
                JSONObject()
                    .put("msgId", pin.msgId)
                    .put("summary", pin.summary)
                    .put("sender", pin.sender)
            )
        }
        prefs.edit().putString(conversationId, arr.toString()).apply()
        _changed.value++
    }
}
