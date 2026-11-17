package com.balabalaim.gcloud.tim.push.appdev.ui.common

enum class FeaturePermission(val key: String, val title: String, val systemPermissions: List<String> = emptyList()) {
    MICROPHONE("microphone", "麦克风", listOf("android.permission.RECORD_AUDIO")),
    CAMERA("camera", "摄像头", listOf("android.permission.CAMERA")),
    IMAGE_PICK("image_pick", "选择图片与头像"),
    VIDEO_PICK("video_pick", "选择视频"),
    AUDIO_PICK("audio_pick", "选择音频文件"),
    FILE_PICK("file_pick", "选择文件"),
    SAVE_MEDIA("save_media", "保存到相册", listOf("android.permission.WRITE_EXTERNAL_STORAGE")),
}

/** App disclosure choices are separate from Android's actual permission grants. */
class FeaturePermissionPolicy(
    private val readChoice: (FeaturePermission) -> Choice,
    private val writeChoice: (FeaturePermission, Choice) -> Unit,
    private val wasSystemRequested: (String) -> Boolean,
    private val writeSystemRequested: (String, Boolean) -> Unit,
) {
    enum class Choice { UNKNOWN, ALLOWED, DENIED }
    enum class Action { EXPLAIN, RUN, REQUEST_SYSTEM, BLOCKED_FEATURE, BLOCKED_SYSTEM }

    fun action(features: List<FeaturePermission>, permissions: List<String>, granted: Set<String>): Action = when {
        features.any { readChoice(it) == Choice.DENIED } -> Action.BLOCKED_FEATURE
        features.any { readChoice(it) == Choice.UNKNOWN } -> Action.EXPLAIN
        permissions.all { it in granted } -> Action.RUN
        permissions.any { it !in granted && wasSystemRequested(it) } -> Action.BLOCKED_SYSTEM
        else -> Action.REQUEST_SYSTEM
    }

    fun choose(features: List<FeaturePermission>, choice: Choice) {
        require(choice != Choice.UNKNOWN)
        // Refusing the newly requested camera must not revoke an already accepted microphone.
        features.distinct().filter { readChoice(it) == Choice.UNKNOWN }.forEach { writeChoice(it, choice) }
    }

    fun requested(permissions: List<String>) = permissions.distinct().forEach { writeSystemRequested(it, true) }
    fun reset(feature: FeaturePermission) {
        writeChoice(feature, Choice.UNKNOWN)
        feature.systemPermissions.forEach { writeSystemRequested(it, false) }
    }
}
