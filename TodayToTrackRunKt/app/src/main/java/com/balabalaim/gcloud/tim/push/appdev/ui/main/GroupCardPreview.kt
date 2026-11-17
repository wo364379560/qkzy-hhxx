package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.tencent.imsdk.v2.V2TIMGroupInfo

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GroupCardPreview(info: V2TIMGroupInfo?, fallbackName: String, groupID: String, loading: Boolean,
                     error: String?, pending: Boolean, joining: Boolean, onBack: () -> Unit,
                     onRetry: () -> Unit, onJoin: () -> Unit) {
    val inviteOnly = info?.groupType in listOf("Work", "Private") || groupID.contains("@TOPIC#")
    val forbidden = info?.groupAddOpt == V2TIMGroupInfo.V2TIM_GROUP_ADD_FORBID
    Scaffold(topBar = { TopAppBar(title = { Text("群名片") }, navigationIcon = { TextButton(onClick = onBack) { Text("返回") } }) }) { padding ->
        Column(Modifier.padding(padding).padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Text(info?.groupName?.ifBlank { fallbackName } ?: fallbackName, style = MaterialTheme.typography.titleLarge)
            Text("群ID：$groupID")
            info?.let { Text("${it.groupType} · ${it.memberCount}位成员") }
            when {
                loading -> CircularProgressIndicator()
                error != null -> Text(error, color = MaterialTheme.colorScheme.error)
                pending -> Text("申请已提交，等待管理员审批或成员状态确认；可刷新查看")
                inviteOnly -> Text(if (groupID.contains("@TOPIC#")) "请先加入所属社群，再打开对应话题" else "此群需要已有成员邀请，不能通过名片直接加入")
                forbidden -> Text("该群禁止主动加入，请联系群主")
                info != null -> Text("你尚未加入此群")
            }
            if (info != null && !inviteOnly && !forbidden && !pending) {
                Button(onClick = onJoin, enabled = !joining && !loading) {
                    Text(if (joining) "正在提交" else if (info.groupAddOpt == V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH) "申请加入" else "加入群聊")
                }
            }
            TextButton(onClick = onRetry, enabled = !loading && !joining) { Text("刷新群状态") }
        }
    }
}
