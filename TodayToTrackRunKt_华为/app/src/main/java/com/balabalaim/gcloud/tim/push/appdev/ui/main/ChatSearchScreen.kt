package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanAccent
import com.tencent.imsdk.v2.V2TIMMessage
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.net.URLDecoder
import java.nio.charset.StandardCharsets
import java.text.SimpleDateFormat
import java.util.Date

object ChatSearchBus {
    @Volatile
    private var selected: Pair<String, String>? = null

    fun select(conversationId: String, messageId: String) {
        selected = conversationId to messageId
    }

    fun consume(conversationId: String): String? {
        val value = selected ?: return null
        if (value.first != conversationId) return null
        selected = null
        return value.second
    }
}

@Composable
fun LinLanChatSearchScreen(
    conversationId: String,
    onBack: () -> Unit,
    onResult: (V2TIMMessage) -> Unit,
) {
    val decodedConversationId = remember(conversationId) {
        URLDecoder.decode(conversationId, StandardCharsets.UTF_8.name())
    }
    var keyword by remember { mutableStateOf("") }
    val results = remember { mutableStateListOf<V2TIMMessage>() }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var nextCursor by remember { mutableStateOf("") }
    var pageIndex by remember { mutableStateOf(0) }
    var fromCloud by remember { mutableStateOf(true) }
    var hasMore by remember { mutableStateOf(false) }
    var searchJob by remember { mutableStateOf<Job?>(null) }
    val scope = rememberCoroutineScope()
    val locale = LocalConfiguration.current.locales[0]

    fun search(reset: Boolean) {
        val term = keyword.trim()
        searchJob?.cancel()
        if (term.isEmpty()) {
            results.clear()
            error = null
            loading = false
            hasMore = false
            return
        }
        searchJob = scope.launch {
            if (reset) delay(280)
            loading = true
            error = null
            val targetCursor = if (reset) "" else nextCursor
            val targetPage = if (reset) 0 else pageIndex + 1
            runCatching {
                LinLanImManager.searchMessages(
                    conversationId = decodedConversationId,
                    keyword = term,
                    cursor = targetCursor,
                    pageIndex = targetPage,
                    preferCloud = if (reset) true else fromCloud,
                )
            }.onSuccess { page ->
                if (reset) results.clear()
                val known = results.mapNotNull { it.msgID }.toHashSet()
                results.addAll(page.messages.filter { it.msgID.isNullOrBlank() || it.msgID !in known })
                nextCursor = page.nextCursor
                pageIndex = targetPage
                fromCloud = page.fromCloud
                hasMore = page.hasMore
            }.onFailure {
                if (reset) results.clear()
                error = it.message ?: "搜索失败"
            }
            loading = false
        }
    }

    LaunchedEffect(keyword) { search(reset = true) }

    Column(modifier = Modifier.fillMaxSize().background(Color(0xFFF5F7F6))) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.White)
                .windowInsetsPadding(WindowInsets.statusBars)
                .height(52.dp)
                .padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
            }
            Row(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(10.dp))
                    .background(Color(0xFFF0F3F2))
                    .padding(horizontal = 12.dp, vertical = 9.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Default.Search, null, tint = Color(0xFF8A918F), modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                BasicTextField(
                    value = keyword,
                    onValueChange = { keyword = it },
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                    decorationBox = { inner ->
                        if (keyword.isEmpty()) Text("搜索聊天记录", color = Color(0xFF8A918F))
                        inner()
                    },
                )
            }
            Spacer(Modifier.width(8.dp))
        }
        when {
            keyword.isBlank() -> SearchEmpty("输入关键词搜索文本、文件名等消息")
            loading && results.isEmpty() -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LinLanAccent)
            }
            error != null && results.isEmpty() -> SearchEmpty(error.orEmpty())
            results.isEmpty() -> SearchEmpty("没有找到相关聊天记录")
            else -> LazyColumn(modifier = Modifier.fillMaxSize()) {
                items(results, key = { it.msgID ?: "${it.seq}_${it.timestamp}" }) { msg ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(Color.White)
                            .clickable { onResult(msg) }
                            .padding(horizontal = 16.dp, vertical = 12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier.size(42.dp).clip(CircleShape).background(Color(0xFFE0F0EC)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                LinLanImManager.displayNameOfMessage(msg).ifBlank { "消息" }.take(1),
                                color = LinLanAccent,
                                fontWeight = FontWeight.Bold,
                            )
                        }
                        Spacer(Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    LinLanImManager.displayNameOfMessage(msg).ifBlank { "用户" },
                                    fontWeight = FontWeight.Medium,
                                    modifier = Modifier.weight(1f),
                                )
                                Text(
                                    SimpleDateFormat("yyyy-MM-dd HH:mm", locale)
                                        .format(Date(msg.timestamp * 1000)),
                                    color = Color(0xFF8A918F),
                                    fontSize = 11.sp,
                                )
                            }
                            Text(
                                LinLanImManager.messagePreview(msg),
                                color = Color(0xFF626967),
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                                fontSize = 14.sp,
                            )
                        }
                    }
                    HorizontalDivider(color = Color(0xFFE6ECE9), thickness = 0.5.dp)
                }
                if (hasMore || loading) {
                    item(key = "more") {
                        Box(Modifier.fillMaxWidth().height(52.dp), contentAlignment = Alignment.Center) {
                            if (loading) CircularProgressIndicator(
                                modifier = Modifier.size(22.dp),
                                strokeWidth = 2.dp,
                                color = LinLanAccent,
                            ) else TextButton(onClick = { search(reset = false) }) { Text("加载更多") }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SearchEmpty(text: String) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Text(text, color = Color(0xFF8A918F), fontSize = 14.sp)
    }
}
