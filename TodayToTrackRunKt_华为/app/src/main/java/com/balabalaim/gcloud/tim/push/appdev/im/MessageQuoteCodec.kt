package com.balabalaim.gcloud.tim.push.appdev.im

import org.json.JSONObject

/** Wire-format compatibility; no dependence on the IM SDK or Android runtime. */
object MessageQuoteCodec {
    fun decode(raw: String?): MessageQuote? {
        if (raw.isNullOrBlank()) return null
        val root = runCatching { JSONObject(raw) }.getOrNull() ?: return null
        return parse(root.optJSONObject("messageReply"), "messageID", "messageSender", "messageAbstract")
            ?: parse(root.optJSONObject("atomicx_quote_v1"), "msgID", "msgSender", "msgAbstract")
    }

    private fun text(json: JSONObject, key: String) = (json.opt(key) as? String)?.trim().orEmpty()

    private fun parse(payload: JSONObject?, idKey: String, senderKey: String, abstractKey: String): MessageQuote? {
        payload ?: return null
        val id = text(payload, idKey).takeIf { it.isNotBlank() } ?: return null
        val sender = text(payload, senderKey)
        val summary = text(payload, abstractKey).ifBlank {
            when (payload.optInt("messageType")) {
                3 -> "[图片]"
                4 -> "[语音]"
                5 -> "[视频]"
                6 -> "[文件]"
                7 -> "[位置]"
                8 -> "[表情]"
                9 -> "[群提示]"
                10 -> "[聊天记录]"
                else -> "[消息]"
            }
        }
        return MessageQuote(id, if (sender.isBlank()) summary else "$sender: $summary")
    }

    fun encode(id: String, sender: String, summary: String, type: Int, sequence: Long = 0, time: Long = 0): String {
        if (id.isBlank()) return "{}"
        return JSONObject()
            .put("atomicx_quote_v1", JSONObject().put("msgID", id).put("msgSender", sender)
                .put("msgAbstract", summary).put("messageType", type))
            .put("messageReply", JSONObject().put("messageID", id).put("messageSender", sender)
                .put("messageAbstract", summary).put("messageType", type).put("messageSequence", sequence)
                .put("messageTime", time).put("version", 1))
            .toString()
    }
}
