package com.balabalaim.gcloud.tim.push.appdev.data.model

data class AuthSession(
    val sdkAppId: Int = 0,
    val userId: String = "",
    val userSig: String = "",
    val userSigExpiresAt: String = "",
    val token: String = "",
    val tokenExpiresAt: String = "",
    val refreshToken: String = "",
    val mustChangePassword: Boolean = false,
) {
    val isLoggedIn: Boolean
        get() = userId.isNotBlank() && userSig.isNotBlank() && token.isNotBlank()
}

data class ChatProfile(
    val imUserId: String = "",
    /** 睿之禾号 / 业务号，优先服务端 wanan_id / chuanlu_id */
    val chuanluId: String = "",
    val nickname: String = "",
    val avatarUrl: String = "",
    val inviteCode: String = "",
    val canBindInviteCode: Boolean = false,
    /** 0=直接通过 1=需验证 2=拒绝任何人；默认 1 */
    val allowType: Int = 1,
    val ownerId: Long = 0,
    val ownerName: String = "",
    val ownerRevision: Long = 0,
    val isAssigned: Boolean? = null,
    val ownerState: String = "unknown",
    val ownerReasonCode: String = "",
) {
    val displaySignalId: String
        get() = chuanluId.ifBlank { imUserId }
}

data class FriendOwnerAssignment(val state: String = "pending", val reasonCode: String = "", val requestId: String = "")

data class GroupNoticeResult(
    val groupId: Int = 0,
    val content: String = "",
    val summary: String = "",
    val source: String = "",
    val hasValue: Boolean = false,
)

data class SearchUserResult(
    val imUserId: String = "",
    val wananId: String? = null,
    val nickname: String = "",
    val userTag: String? = null,
    val avatarUrl: String? = null,
    val canAddFriend: Boolean = true,
    val addBlockedReason: String = "",
) {
    val displayName: String
        get() = nickname.ifBlank { wananId.orEmpty().ifBlank { imUserId } }
}
