package com.balabalaim.gcloud.tim.push.appdev.data.model

import org.json.JSONObject

data class FriendOwnershipScope(val sdk: Int, val user: String, val generation: Long) {
    fun matches(sdk: Int, user: String, generation: Long) =
        this.user.isNotBlank() && this.sdk == sdk && this.user == user && this.generation == generation
}

object FriendOwnerProtocol {
    fun profile(data: JSONObject) = ChatProfile(
        imUserId = data.optString("im_user_id"),
        chuanluId = data.optString("chuanlu_id").ifBlank { data.optString("wanan_id") },
        nickname = data.optString("nickname"), avatarUrl = data.optString("avatar_url"),
        inviteCode = data.optString("invite_code"), canBindInviteCode = data.optBoolean("can_bind_invite_code", true),
        allowType = data.optInt("allow_type", 1), ownerId = data.optLong("owner_id"), ownerName = data.optString("owner_name"),
        ownerRevision = data.optLong("owner_revision"),
        isAssigned = if (data.has("is_assigned") && !data.isNull("is_assigned")) data.optBoolean("is_assigned") else null,
        ownerState = data.optString("owner_state", "unknown"), ownerReasonCode = data.optString("owner_reason_code"),
    )
    fun assignment(data: JSONObject, requestId: String): FriendOwnerAssignment {
        val result = data.optJSONObject("assignment")
        return FriendOwnerAssignment(result?.optString("state", "pending") ?: "pending",
            result?.optString("reason_code") ?: "legacy_server_response", result?.optString("request_id", requestId) ?: requestId)
    }
}
