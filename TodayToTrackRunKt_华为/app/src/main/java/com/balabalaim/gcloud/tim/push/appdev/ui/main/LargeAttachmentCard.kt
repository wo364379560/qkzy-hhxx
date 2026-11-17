package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.platform.LocalContext
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import kotlinx.coroutines.launch
import org.json.JSONObject

fun parseLargeAttachment(text: String): JSONObject? = runCatching {
    val line = text.lineSequence().firstOrNull { it.startsWith("CN_ATTACHMENT_V1:") } ?: return@runCatching null
    JSONObject(line.removePrefix("CN_ATTACHMENT_V1:")).takeIf { it.optString("file_id").isNotBlank() && it.optString("name").isNotBlank() }
}.getOrNull()

@Composable
fun LargeAttachmentCard(card: JSONObject) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var opening by remember(card.optString("file_id")) { mutableStateOf(false) }
    TextButton(enabled = !opening, onClick = {
        opening = true
        scope.launch {
            try {
                val url = LinLanAppContainer.apiClient().attachmentAccess(card.getString("file_id"))
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
            } catch (_: Exception) { Toast.makeText(context, "文件暂时无法访问，请稍后重试", Toast.LENGTH_SHORT).show() }
            finally { opening = false }
        }
    }) { Text("${if (card.optBoolean("is_video")) "视频" else "文件"}：${card.optString("name")}\n${if (opening) "正在获取访问地址" else "点击预览或下载"}") }
}
