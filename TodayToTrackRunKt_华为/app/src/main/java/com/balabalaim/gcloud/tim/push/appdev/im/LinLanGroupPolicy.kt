package com.balabalaim.gcloud.tim.push.appdev.im

/**
 * Aligns with iOS chenDao_GroupPolicy.
 */
object LinLanGroupPolicy {
    const val MEMBER_PROFILE_ACCESS_KEY = "inner_attr_member_profile_access"
    const val MUTE_ADMIN_ALSO_KEY = "inner_attr_mute_admin_also"
    const val VOICE_CALL_KEY = "inner_attr_group_voice_call"
    const val VIDEO_CALL_KEY = "inner_attr_group_video_call"

    private const val ENABLED = "1"
    private const val DISABLED = "0"

    fun isMemberProfileAccessEnabled(attributes: Map<String, String>): Boolean =
        isEnabled(attributes[MEMBER_PROFILE_ACCESS_KEY], defaultValue = true)

    fun isMuteAdminAlsoEnabled(attributes: Map<String, String>): Boolean =
        isEnabled(attributes[MUTE_ADMIN_ALSO_KEY], defaultValue = false)

    fun isVoiceCallEnabled(attributes: Map<String, String>): Boolean =
        isEnabled(attributes[VOICE_CALL_KEY], defaultValue = false)

    fun isVideoCallEnabled(attributes: Map<String, String>): Boolean =
        isEnabled(attributes[VIDEO_CALL_KEY], defaultValue = false)

    suspend fun fetchAll(groupId: String): Map<String, String> =
        LinLanImManager.getGroupAttributes(
            groupId,
            listOf(MEMBER_PROFILE_ACCESS_KEY, MUTE_ADMIN_ALSO_KEY, VOICE_CALL_KEY, VIDEO_CALL_KEY),
        )

    suspend fun updateMemberProfileAccess(groupId: String, enabled: Boolean) {
        LinLanImManager.setGroupAttributes(
            groupId,
            mapOf(MEMBER_PROFILE_ACCESS_KEY to if (enabled) ENABLED else DISABLED),
        )
    }

    suspend fun updateMuteAdminAlso(groupId: String, enabled: Boolean) {
        LinLanImManager.setGroupAttributes(
            groupId,
            mapOf(MUTE_ADMIN_ALSO_KEY to if (enabled) ENABLED else DISABLED),
        )
    }

    suspend fun updateVoiceCallAccess(groupId: String, enabled: Boolean) {
        LinLanImManager.setGroupAttributes(
            groupId,
            mapOf(VOICE_CALL_KEY to if (enabled) ENABLED else DISABLED),
        )
    }

    suspend fun updateVideoCallAccess(groupId: String, enabled: Boolean) {
        LinLanImManager.setGroupAttributes(
            groupId,
            mapOf(VIDEO_CALL_KEY to if (enabled) ENABLED else DISABLED),
        )
    }

    suspend fun fetchMemberProfileAccessEnabled(groupId: String): Boolean {
        val attrs = LinLanImManager.getGroupAttributes(groupId, listOf(MEMBER_PROFILE_ACCESS_KEY))
        return isMemberProfileAccessEnabled(attrs)
    }

    private fun isEnabled(raw: String?, defaultValue: Boolean): Boolean {
        val normalized = raw?.trim()?.lowercase().orEmpty()
        if (normalized.isEmpty()) return defaultValue
        return normalized == ENABLED || normalized == "true" || normalized == "yes" || normalized == "on"
    }
}
