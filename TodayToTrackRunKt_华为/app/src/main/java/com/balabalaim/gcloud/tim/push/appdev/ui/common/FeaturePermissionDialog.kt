package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp

@Composable
fun FeaturePermissionDialog(onDismiss: () -> Unit) {
    val context = LocalContext.current
    val store = remember(context) { FeaturePermissionStore(context) }
    var revision by remember { mutableIntStateOf(0) }
    AlertDialog(onDismissRequest = onDismiss, title = { Text("功能权限") }, text = {
        Column(Modifier.heightIn(max = 420.dp).verticalScroll(rememberScrollState())) {
            Text("允许或拒绝后会记住您的选择。点击“重新设置”后，下次使用该功能时重新选择；系统授权仍以手机设置为准。")
            key(revision) {
                FeaturePermission.entries.forEach { feature ->
                    Row(Modifier.fillMaxWidth(), verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(feature.title)
                            Text(when (store.choice(feature)) {
                                FeaturePermissionPolicy.Choice.UNKNOWN -> "未选择"
                                FeaturePermissionPolicy.Choice.ALLOWED -> "已允许"
                                FeaturePermissionPolicy.Choice.DENIED -> "已拒绝"
                            }, style = MaterialTheme.typography.bodySmall)
                        }
                        TextButton(onClick = { store.policy.reset(feature); revision++ }) { Text("重新设置") }
                    }
                }
            }
        }
    }, confirmButton = { TextButton(onClick = onDismiss) { Text("完成") } }, dismissButton = {
        TextButton(onClick = {
            context.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.fromParts("package", context.packageName, null)))
        }) { Text("系统权限设置") }
    })
}
