package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanForwardJobs

@Composable
fun ForwardResultsHost(owner: String) {
    fun statusName(status: String) = mapOf("pending" to "未发送", "sending" to "发送中", "success" to "已发送", "failed" to "失败", "unknown" to "待确认", "interrupted" to "已中断", "cancelled" to "已停止")[status] ?: "待确认"
    val job by LinLanForwardJobs.current.collectAsState()
    val presentedId by LinLanForwardJobs.presentedJobId.collectAsState()
    val current = job?.takeIf { it.owner == owner } ?: return
    // Dismiss this job's UI completely; a new job ID opens its own result dialog.
    if (presentedId != current.id) return
    AlertDialog(onDismissRequest = { LinLanForwardJobs.dismissResult(current.id) }, title = { Text(if (current.running) "正在转发" else "转发结果") },
        text = {
            Column {
                Text("已发送 ${current.items.count { it.status == "success" }}/${current.items.size}；待确认 ${current.items.count { it.status == "unknown" }}")
                Text("收起后本进程继续；重启只保留结果，待确认消息不会自动重发。")
                LazyColumn(Modifier.heightIn(max = 350.dp)) {
                    items(current.items) { row -> Text("${row.name} · 第${row.index + 1}条：${statusName(row.status)}\n${row.error}", Modifier.padding(vertical = 6.dp)) }
                }
            }
        },
        confirmButton = { TextButton(onClick = { LinLanForwardJobs.dismissResult(current.id) }) { Text(if (current.running) "收起" else "关闭") } },
        dismissButton = {
            if (current.running) TextButton(onClick = { LinLanForwardJobs.cancel() }) { Text("停止未发送部分") }
            else if (current.resumable && current.items.any { it.status in setOf("failed", "pending") }) {
                TextButton(onClick = { LinLanForwardJobs.retry() }) { Text("补发明确失败/未发项") }
            }
        })
}
