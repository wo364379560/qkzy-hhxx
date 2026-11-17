package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanVideoJobs

@Composable
fun VideoResultsHost(owner: String) {
    val job by LinLanVideoJobs.current.collectAsState()
    val current = job?.takeIf { it.owner == owner } ?: return
    var visible by remember(current.id) { mutableStateOf(true) }
    if (!visible) {
        Box(Modifier.fillMaxSize()) { TextButton(onClick = { visible = true }, Modifier.align(Alignment.BottomEnd)) { Text("查看视频发送结果") } }
        return
    }
    val names = mapOf("checking" to "检查资料", "ready" to "待确认发送", "preparing" to "准备视频和封面", "uploading" to "上传中", "sending" to "发送中", "success" to "已发送", "failed" to "失败", "cancelled" to "已取消", "unknown" to "待确认")
    AlertDialog(onDismissRequest = { visible = false }, title = { Text(if (!current.confirmed) "确认发送${current.items.size}个视频" else "视频发送结果") },
        text = { Column {
            Text("最多9个；单个不超过${current.fileLimit / 1024 / 1024}MB，本批总计不超过1GB。逐个处理，失败不会阻断其他视频。")
            LazyColumn(Modifier.heightIn(max = 340.dp)) { items(current.items, key = { it.index }) { row ->
                Text("${row.index + 1}. ${row.name}\n${names[row.status] ?: row.status}${if (row.status in setOf("uploading", "sending")) " ${row.progress}%" else ""}${if (row.error.isBlank()) "" else "\n${row.error}"}", Modifier.padding(vertical = 8.dp))
            } }
            if (current.items.any { it.status == "unknown" }) Text("待确认项请先核对接收端，不会自动重发。")
        } }, confirmButton = {
            if (!current.confirmed) TextButton(onClick = { LinLanVideoJobs.send() }, enabled = current.items.any { it.status == "ready" } && current.items.none { it.status == "checking" }) { Text("确认发送") }
            else TextButton(onClick = { visible = false }) { Text("收起") }
        }, dismissButton = {
            if (current.running || !current.confirmed) TextButton(onClick = { LinLanVideoJobs.cancel(); if (!current.confirmed) visible = false }) { Text("取消未发送部分") }
            else if (current.items.any { it.status == "failed" }) TextButton(onClick = { LinLanVideoJobs.send(retry = true) }) { Text("仅重试失败项") }
        })
}
