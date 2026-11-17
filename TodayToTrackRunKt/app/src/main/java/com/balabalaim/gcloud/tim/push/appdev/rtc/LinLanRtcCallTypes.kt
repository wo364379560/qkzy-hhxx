package com.balabalaim.gcloud.tim.push.appdev.rtc

enum class LinLanRtcCallStatus {
    NONE,
    WAITING,
    ACCEPT,
}

enum class LinLanRtcCallMediaType(val value: Int) {
    AUDIO(1),
    VIDEO(2);

    companion object {
        fun from(value: Int): LinLanRtcCallMediaType =
            entries.firstOrNull { it.value == value } ?: AUDIO
    }
}

enum class LinLanRtcCallRole {
    CALLER,
    CALLED,
}

enum class LinLanRtcCallEndReason {
    HANGUP,
    REJECT,
    NO_RESPONSE,
    BUSY,
    CANCEL,
}

data class RtcCallPeer(
    val userId: String,
    val displayName: String,
    val avatarUrl: String? = null,
)

data class LinLanRtcCallContext(
    val inviteId: String,
    val roomId: String,
    val peer: RtcCallPeer,
    val mediaType: LinLanRtcCallMediaType,
    val role: LinLanRtcCallRole,
    val roomType: String = "string",
)
