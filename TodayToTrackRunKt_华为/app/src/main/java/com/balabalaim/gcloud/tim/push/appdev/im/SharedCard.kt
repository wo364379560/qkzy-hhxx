package com.balabalaim.gcloud.tim.push.appdev.im

import org.json.JSONObject

data class RuiSharedCard(val id: String, val name: String, val isGroup: Boolean, val avatar: String = "") {
    fun encode(): String = JSONObject().put("businessID", if (isGroup) "group_card_v1" else "contact_card_v1")
        .put("version", 2).put(if (isGroup) "groupId" else "userId", id)
        .put(if (isGroup) "groupName" else "nickname", name).put("avatar", avatar).toString()

    companion object {
        fun parse(raw: String): RuiSharedCard? = runCatching {
            if (raw.length > 16384) return null
            val data = JSONObject(raw)
            val business = data.optString("businessID")
            if (business !in listOf("contact_card_v1", "group_card_v1", "zhihuan_contact_card")) return null
            if (data.optInt("version", 1) !in 1..2) return null
            val group = business == "group_card_v1"
            val id = data.optString(if (group) "groupId" else "userId").ifBlank { data.optString(if (group) "groupID" else "userID") }.trim()
            if (id.isBlank() || id.length > 128 || id.any { it.isWhitespace() || it.code < 32 }) return null
            val name = data.optString(if (group) "groupName" else "nickname").trim().take(128).ifBlank { id }
            val avatar = data.optString("avatar").takeIf { value -> runCatching { java.net.URI(value).let { it.scheme in listOf("https", "http") && !it.host.isNullOrBlank() && it.userInfo == null } }.getOrDefault(false) }.orEmpty()
            RuiSharedCard(id, name, group, avatar)
        }.getOrNull()
    }
}

class LinLanSdkOperationError(val stage: String, val sdkCode: Int, detail: String? = null) : Exception(
    "$stage 失败（$sdkCode）：" + when (sdkCode) {
        10007 -> "操作权限不足，请核对成员身份、角色和群类型"
        10010 -> "群不存在或已解散"
        10015 -> "群ID或参数无效"
        6004, 6014, 6206 -> "连接或登录状态异常，请重连后再试"
        else -> detail.orEmpty().replace(Regex("https?://[^\\s]+"), "[地址已隐藏]").take(120).ifBlank { "请稍后重试" }
    }
)
