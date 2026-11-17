package com.balabalaim.gcloud.tim.push.appdev.rtc

import com.tencent.imsdk.v2.V2TIMManager
import com.tencent.imsdk.v2.V2TIMMessage
import com.tencent.imsdk.v2.V2TIMSignalingInfo
import org.json.JSONObject
import java.nio.charset.StandardCharsets

enum class LinLanRtcSignalingAction(val raw: String) {
    INVITE("invite"),
    ACCEPT("accept"),
    REJECT("reject"),
    BUSY("busy"),
    CANCEL("cancel"),
    HANGUP("hangup"),
    NO_RESPONSE("no_response");

    companion object {
        fun from(raw: String?): LinLanRtcSignalingAction =
            entries.firstOrNull { it.raw == raw } ?: INVITE
    }
}

data class LinLanRtcSignalingPayload(
    val action: LinLanRtcSignalingAction,
    val mediaType: LinLanRtcCallMediaType,
    val roomId: String,
    val peerName: String? = null,
    val peerAvatar: String? = null,
    val durationSec: Int? = null,
    val businessId: String = BUSINESS,
    val version: Int = CURRENT_VERSION,
    val roomType: String = "string",
    val inviteId: String? = null,
) {
    fun toJson(): String = JSONObject().apply {
        put("businessID", businessId)
        put("action", action.raw)
        put("version", version)
        put("mediaType", mediaType.value)
        put("roomId", roomId)
        put("roomType", roomType)
        if (action == LinLanRtcSignalingAction.INVITE) put("expiresAtMs", System.currentTimeMillis() + 45000)
        if (!inviteId.isNullOrBlank()) put("inviteId", inviteId)
        if (!peerName.isNullOrBlank()) put("peerName", peerName)
        if (!peerAvatar.isNullOrBlank()) put("peerAvatar", peerAvatar)
        if (durationSec != null) put("durationSec", durationSec)
    }.toString()

    /** 会话列表 / 气泡展示文案（对齐微信：已接听后展示通话时长） */
    fun displayText(isSelf: Boolean): String {
        val media = if (mediaType == LinLanRtcCallMediaType.VIDEO) "视频" else "语音"
        return when (action) {
            LinLanRtcSignalingAction.INVITE -> if (isSelf) "[发起${media}通话]" else "[${media}通话]"
            LinLanRtcSignalingAction.ACCEPT -> "[已接听]"
            LinLanRtcSignalingAction.BUSY -> "[对方忙线]"
            LinLanRtcSignalingAction.REJECT -> if (isSelf) "[已拒绝]" else "[对方已拒绝]"
            LinLanRtcSignalingAction.CANCEL -> if (isSelf) "[已取消]" else "[对方已取消]"
            LinLanRtcSignalingAction.HANGUP -> {
                val sec = durationSec ?: 0
                if (sec > 0) "通话时长 ${formatClock(sec)}" else "[${media}通话结束]"
            }
            LinLanRtcSignalingAction.NO_RESPONSE -> if (isSelf) "[对方无应答]" else "[未接听]"
        }
    }

    companion object {
        const val BUSINESS = "rtc_call_v1"
        const val CURRENT_VERSION = 2

        fun formatClock(sec: Int): String {
            val safe = sec.coerceAtLeast(0)
            val m = safe / 60
            val s = safe % 60
            return String.format("%02d:%02d", m, s)
        }

        /** 邀请 / 接听是中间态，聊天里不单独占一条；拒绝、取消、未接、挂断（含时长）保留。 */
        fun shouldHideInChat(payload: LinLanRtcSignalingPayload): Boolean =
            payload.action == LinLanRtcSignalingAction.INVITE ||
                payload.action == LinLanRtcSignalingAction.ACCEPT

        fun parse(json: String?): LinLanRtcSignalingPayload? {
            if (json.isNullOrBlank()) return null
            return runCatching {
                val obj = unwrapJsonObject(json) ?: return null
                val business = obj.optString("businessID")
                val longdo = obj.optString("longdo_type")
                if (longdo == "longdo_rtc_call_record_v1") return null
                val legacyWeb = longdo.startsWith("longdo_rtc_call_")
                if (!legacyWeb && business != BUSINESS && !(business.isBlank() && obj.has("roomId") && obj.has("mediaType"))) return null
                val action = if (legacyWeb) when (longdo) {
                    "longdo_rtc_call_invite_v1" -> "invite"
                    "longdo_rtc_call_accept_v1" -> "accept"
                    "longdo_rtc_call_reject_v1" -> "reject"
                    "longdo_rtc_call_busy_v1" -> "busy"
                    "longdo_rtc_call_cancel_v1" -> "cancel"
                    else -> return null
                } else obj.optString("action", "invite")
                if (LinLanRtcSignalingAction.entries.none { it.raw == action }) return null
                val version = obj.optInt(if (legacyWeb) "longdo_version" else "version", 1)
                if (version !in 1..CURRENT_VERSION) return null
                val room = obj.optString(if (legacyWeb) "longdo_room_id" else "roomId")
                val type = if (legacyWeb) "number" else obj.optString("roomType", "string")
                if (type !in listOf("string", "number")) return null
                if (action == "invite" && (room.isBlank() || room.length > 128 || (obj.optLong("expiresAtMs") > 0 && obj.optLong("expiresAtMs") + 5000 < System.currentTimeMillis()))) return null
                if (type == "number" && room.isNotBlank() && (room.toLongOrNull() ?: 0) !in 1..2147483647L) return null
                LinLanRtcSignalingPayload(
                    action = LinLanRtcSignalingAction.from(action),
                    mediaType = LinLanRtcCallMediaType.from(if (legacyWeb) { if (obj.optString("longdo_media_type") == "longdo_video") 2 else 1 } else obj.optInt("mediaType", 1)),
                    roomId = room,
                    peerName = obj.optString(if (legacyWeb) "longdo_caller_name" else "peerName").ifBlank { null },
                    peerAvatar = obj.optString(if (legacyWeb) "longdo_caller_avatar_url" else "peerAvatar").ifBlank { null },
                    durationSec = if (obj.has("durationSec")) obj.optInt("durationSec") else null,
                    version = version, roomType = type, inviteId = obj.optString("inviteId").ifBlank { null },
                )
            }.getOrNull()
        }

        /**
         * TIM 信令会把业务 data 包一层；需用 getSignalingInfo 取内层 JSON。
         */
        fun parseFromMessage(msg: V2TIMMessage): LinLanRtcSignalingPayload? {
            val custom = msg.customElem
            if (custom != null) {
                val raw = custom.data?.toString(StandardCharsets.UTF_8)
                parse(raw)?.let { return it }
                parse(custom.description)?.let { return it }
                val ext = custom.extension
                val extStr = when (ext) {
                    is String -> ext
                    is ByteArray -> ext.toString(StandardCharsets.UTF_8)
                    else -> ext?.toString()
                }
                parse(extStr)?.let { return it }
            }
            val info = runCatching {
                V2TIMManager.getSignalingManager().getSignalingInfo(msg)
            }.getOrNull()
            if (info != null) {
                parse(info.data)?.let { payload ->
                    return payload.copy(action = actionFromSignalingInfo(info, payload.action))
                }
                // 有信令壳但 data 解析失败时，仍按 actionType 给默认语音通话文案
                if (!info.inviteID.isNullOrBlank() || info.actionType > 0) {
                    return LinLanRtcSignalingPayload(
                        action = actionFromSignalingInfo(info, LinLanRtcSignalingAction.INVITE),
                        mediaType = LinLanRtcCallMediaType.AUDIO,
                        roomId = "",
                    )
                }
            }
            return null
        }

        private fun unwrapJsonObject(json: String): JSONObject? {
            var current = JSONObject(json)
            repeat(4) {
                if (current.optString("businessID") == BUSINESS || current.has("longdo_type") || current.has("roomId")) return current
                val nested = current.opt("data")
                current = when (nested) {
                    is JSONObject -> nested
                    is String -> runCatching { JSONObject(nested) }.getOrNull() ?: return null
                    else -> return null
                }
            }
            return null
        }

        private fun actionFromSignalingInfo(
            info: V2TIMSignalingInfo,
            fallback: LinLanRtcSignalingAction,
        ): LinLanRtcSignalingAction = when (info.actionType) {
            V2TIMSignalingInfo.SIGNALING_ACTION_TYPE_INVITE -> LinLanRtcSignalingAction.INVITE
            V2TIMSignalingInfo.SIGNALING_ACTION_TYPE_CANCEL_INVITE -> LinLanRtcSignalingAction.CANCEL
            V2TIMSignalingInfo.SIGNALING_ACTION_TYPE_ACCEPT_INVITE -> LinLanRtcSignalingAction.ACCEPT
            V2TIMSignalingInfo.SIGNALING_ACTION_TYPE_REJECT_INVITE -> LinLanRtcSignalingAction.REJECT
            V2TIMSignalingInfo.SIGNALING_ACTION_TYPE_INVITE_TIMEOUT -> LinLanRtcSignalingAction.NO_RESPONSE
            else -> fallback
        }
    }
}
