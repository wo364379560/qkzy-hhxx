package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.Manifest
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants

object PermissionCopy {
    const val CAMERA =
        "睿之禾需要使用相机，在您进行视频通话时采集并向通话对方传输画面。拒绝授权后，将无法开启视频画面，但仍可使用文字聊天和已授权的语音功能。您可随时在系统设置中管理相机权限。"
    const val MICROPHONE =
        "睿之禾需要使用麦克风，在您主动录制聊天语音时采集声音并发送语音消息，以及在语音或视频通话时向通话对方传输声音。拒绝授权后，将无法录制语音消息或使用需要麦克风的通话功能，但不影响文字聊天和浏览消息。您可随时在系统设置中管理麦克风权限。"
    const val CAMERA_AND_MIC =
        "睿之禾需要使用相机和麦克风，在视频通话中采集并向通话对方传输您的画面和声音。相机用于视频画面，麦克风用于通话声音。拒绝相应权限后，将无法正常进行视频通话，但不影响文字聊天和浏览消息。您可随时在系统设置中管理这些权限。"
    const val ALBUM_IMAGE =
        "睿之禾将打开系统图片选择器，用于选择聊天中发送的图片或设置个人、群聊头像。应用只读取您主动选择的图片；选择完成后，该图片将用于发送或上传头像。无需授权访问全部照片，取消选择不会影响文字聊天等其他功能。"
    const val ALBUM_VIDEO =
        "睿之禾将打开系统视频选择器，用于选择并发送聊天视频。应用只读取您主动选择的视频及生成发送所需的封面，选择完成后将发送给当前聊天对象。无需授权访问全部照片和视频，取消选择不会影响其他聊天功能。"
    const val ALBUM_AUDIO =
        "睿之禾将打开系统文件选择器，用于选择并发送音频文件。应用只读取您主动选择的文件，取消选择不会影响文字聊天等其他功能。"
    const val STORAGE_FILE =
        "睿之禾将打开系统文件选择器，用于选择并发送聊天文件。应用只读取您主动选择的文件，选择完成后将发送给当前聊天对象。取消选择不会影响其他聊天功能。"
    const val SAVE_TO_ALBUM =
        "睿之禾需要存储写入权限，将您在聊天中主动点击保存的图片或视频写入手机相册，便于您在系统相册中查看和使用。仅在您执行保存操作时写入。拒绝授权后，该内容将无法保存到相册，但仍可在聊天中查看。"
    const val LOCATION =
        "睿之禾需要使用位置信息，开始运动后通过手机 GPS 获取精确位置，记录并显示运动轨迹、计算里程和配速。轨迹保存在本机。为了保持运动记录连续，运动期间即使切换到后台或锁屏，也会通过持续显示通知的定位服务继续记录。暂停或结束运动后，将停止该次运动的持续定位。拒绝授权后，无法记录跑步轨迹，但不影响聊天等其他功能。"
}

/** The explanation is asked once; denial blocks the action until an explicit settings reset. */
@Composable
fun rememberExplainedPermissionAction(
    permissions: List<String>,
    rationale: String,
    features: List<FeaturePermission>,
    store: FeaturePermissionStore? = null,
    onGranted: () -> Unit,
): () -> Unit {
    val grantedAction by androidx.compose.runtime.rememberUpdatedState(onGranted)
    val context = LocalContext.current
    val savedStore = remember(context, store) { store ?: FeaturePermissionStore(context) }
    var showRationale by remember { mutableStateOf(false) }
    var requesting by remember { mutableStateOf(false) }
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) {
        requesting = false
        if (savedStore.action(features, permissions) == FeaturePermissionPolicy.Action.RUN) grantedAction()
        else Toast.makeText(context, "权限未开启，可在系统设置中开启；也可到“我的 → 功能权限”重新设置", Toast.LENGTH_LONG).show()
    }
    fun proceed() {
        if (requesting || showRationale) return
        when (savedStore.action(features, permissions)) {
            FeaturePermissionPolicy.Action.EXPLAIN -> showRationale = true
            FeaturePermissionPolicy.Action.RUN -> grantedAction()
            FeaturePermissionPolicy.Action.REQUEST_SYSTEM -> {
                val missing = permissions.filter { it !in savedStore.granted(permissions) }
                savedStore.policy.requested(missing)
                requesting = true
                launcher.launch(missing.toTypedArray())
            }
            FeaturePermissionPolicy.Action.BLOCKED_FEATURE -> Toast.makeText(context,
                "已拒绝此功能，可到“我的 → 功能权限”重新设置", Toast.LENGTH_LONG).show()
            FeaturePermissionPolicy.Action.BLOCKED_SYSTEM -> Toast.makeText(context,
                "权限未开启，可在系统设置中开启；也可到“我的 → 功能权限”重新设置", Toast.LENGTH_LONG).show()
        }
    }
    if (showRationale) AlertDialog(
        onDismissRequest = { showRationale = false },
        title = { Text("${LinLanAppConstants.APP_NAME}权限说明") },
        text = { Text(rationale) },
        confirmButton = { TextButton(onClick = {
            savedStore.policy.choose(features, FeaturePermissionPolicy.Choice.ALLOWED)
            showRationale = false
            proceed()
        }) { Text("允许") } },
        dismissButton = { TextButton(onClick = {
            savedStore.policy.choose(features, FeaturePermissionPolicy.Choice.DENIED)
            showRationale = false
        }) { Text("拒绝") } },
    )
    return { proceed() }
}

@Composable
fun rememberAlbumImageAction(onGranted: () -> Unit): () -> Unit {
    // Only the URI selected in the gallery is read; broad library access is unnecessary.
    return rememberExplainedPermissionAction(emptyList(), PermissionCopy.ALBUM_IMAGE, listOf(FeaturePermission.IMAGE_PICK), onGranted = onGranted)
}

@Composable
fun rememberAlbumVideoAction(onGranted: () -> Unit): () -> Unit {
    return rememberExplainedPermissionAction(emptyList(), PermissionCopy.ALBUM_VIDEO, listOf(FeaturePermission.VIDEO_PICK), onGranted = onGranted)
}

@Composable
fun rememberAlbumAudioAction(onGranted: () -> Unit): () -> Unit {
    return rememberExplainedPermissionAction(emptyList(), PermissionCopy.ALBUM_AUDIO, listOf(FeaturePermission.AUDIO_PICK), onGranted = onGranted)
}

@Composable
fun rememberFilePickAction(onGranted: () -> Unit): () -> Unit =
    rememberExplainedPermissionAction(emptyList(), PermissionCopy.STORAGE_FILE, listOf(FeaturePermission.FILE_PICK), onGranted = onGranted)

@Composable
fun rememberCallPermissionAction(needCamera: Boolean, onGranted: () -> Unit): () -> Unit {
    val perms = buildList {
        add(Manifest.permission.RECORD_AUDIO)
        if (needCamera) add(Manifest.permission.CAMERA)
    }
    return rememberExplainedPermissionAction(
        permissions = perms,
        rationale = if (needCamera) PermissionCopy.CAMERA_AND_MIC else PermissionCopy.MICROPHONE,
        features = if (needCamera) listOf(FeaturePermission.MICROPHONE, FeaturePermission.CAMERA) else listOf(FeaturePermission.MICROPHONE),
        onGranted = onGranted,
    )
}
