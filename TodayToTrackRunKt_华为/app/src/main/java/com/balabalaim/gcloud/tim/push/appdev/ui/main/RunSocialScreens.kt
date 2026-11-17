package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.run.*
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.*
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

private val SportGreen = Color(0xFFC4EF69)

@Composable
fun RunShareCard(card: RunShare, onClick: () -> Unit) {
    Column(Modifier.widthIn(max = 250.dp).fillMaxWidth().background(LinLanInk, RoundedCornerShape(16.dp))
        .clickable(onClick = onClick).padding(16.dp), verticalArrangement = Arrangement.spacedBy(7.dp)) {
        Text("运动打卡", color = SportGreen, fontSize = 12.sp)
        Text(card.distance, color = Color.White, fontSize = 25.sp, fontWeight = FontWeight.Bold)
        Text("${card.duration} · ${card.pace}", color = Color.White.copy(alpha = .8f), fontSize = 12.sp)
        if (card.route.isNotEmpty()) SharedRoute(card.route, Modifier.fillMaxWidth().height(80.dp))
        Text("查看运动详情 →", color = SportGreen, fontSize = 12.sp)
    }
}

@Composable
private fun SharedRoute(route: List<RoutePosition>, modifier: Modifier) {
    Canvas(modifier) {
        val side = minOf(size.width, size.height)
        fun point(p: RoutePosition) = Offset((size.width - side) / 2 + side * p.x, (size.height - side) / 2 + side * p.y)
        route.zipWithNext().forEach { (a, b) -> if (a.segment == b.segment) drawLine(SportGreen, point(a), point(b), 3.dp.toPx(), StrokeCap.Round) }
        route.firstOrNull()?.let { drawCircle(Color.White, 4.dp.toPx(), point(it)) }
        route.lastOrNull()?.let { drawCircle(SportGreen, 5.dp.toPx(), point(it)) }
    }
}

@Composable
fun RunShareDetail(card: RunShare, onDismiss: () -> Unit) {
    val context = androidx.compose.ui.platform.LocalContext.current
    Dialog(onDismissRequest = onDismiss) {
        Surface(shape = RoundedCornerShape(24.dp), color = LinLanInk) {
            Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Text("每一步，都值得记录", color = SportGreen, fontSize = 22.sp, fontWeight = FontWeight.Bold)
                Text(SimpleDateFormat("yyyy年MM月dd日 HH:mm", Locale.CHINA).format(Date(card.startedAt)), color = Color.White.copy(alpha = .7f))
                Text(card.distance, color = Color.White, fontSize = 34.sp, fontWeight = FontWeight.Bold)
                Text("运动时长  ${card.duration}\n平均配速  ${card.pace}", color = Color.White, lineHeight = 28.sp)
                if (card.route.isNotEmpty()) {
                    SharedRoute(card.route, Modifier.fillMaxWidth().height(220.dp))
                    Text("轨迹形状 · 不含具体位置", color = Color.White.copy(alpha = .65f), fontSize = 12.sp)
                }
                if (card.start != null || card.end != null) {
                    Text("点击将起点或终点位置交给百度地图查看。", color = Color.White.copy(alpha = .7f), fontSize = 12.sp)
                    card.start?.let { point -> TextButton(onClick = { openBaiduRunLocation(context, point, "运动起点") }) { Text("百度地图 · 查看起点", color = SportGreen) } }
                    card.end?.let { point -> TextButton(onClick = { openBaiduRunLocation(context, point, "运动终点") }) { Text("百度地图 · 查看终点", color = SportGreen) } }
                }
                Text("打开底部「运动」，记录你的下一次出发。", color = SportGreen, fontSize = 13.sp)
                TextButton(onClick = onDismiss) { Text("返回聊天", color = SportGreen) }
            }
        }
    }
}

private data class RunRecipient(val id: String, val name: String, val group: Boolean)

@Composable
fun RunShareDialog(run: RunSession, onDismiss: () -> Unit, onSent: (String, String, Boolean) -> Unit, fixedRecipient: Triple<String, String, Boolean>? = null) {
    val scope = rememberCoroutineScope()
    var targets by remember { mutableStateOf<List<RunRecipient>>(emptyList()) }
    var target by remember { mutableStateOf<RunRecipient?>(fixedRecipient?.let { RunRecipient(it.first, it.second, it.third) }) }
    var includeShape by remember { mutableStateOf(true) }
    var includeLocations by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var search by remember { mutableStateOf("") }
    var reload by remember { mutableIntStateOf(0) }
    LaunchedEffect(reload) {
        if (fixedRecipient != null) { loading = false; return@LaunchedEffect }
        loading = true; error = null
        runCatching {
            val contacts = LinLanImManager.fetchSelectableContacts().filter { it.userId != "security_assistant" }
                .map { RunRecipient(it.userId, it.displayName, false) }
            val groups = LinLanImManager.fetchJoinedGroups().map { RunRecipient(it.groupID, it.groupName?.ifBlank { it.groupID } ?: it.groupID, true) }
            contacts + groups
        }.onSuccess { targets = it }.onFailure { error = "加载分享对象失败，请重试" }
        loading = false
    }
    AlertDialog(onDismissRequest = { if (!busy) onDismiss() }, title = { Text("分享运动打卡") }, text = {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(RunShare.from(run, false).summary, fontSize = 13.sp)
            Row {
                Checkbox(checked = includeShape, enabled = !busy, onCheckedChange = { includeShape = it })
                Text("附上轨迹形状（不含经纬度）", modifier = Modifier.padding(top = 12.dp), fontSize = 12.sp)
            }
            Row {
                Checkbox(checked = includeLocations, enabled = !busy, onCheckedChange = { includeLocations = it })
                Text("附上起终点位置，允许用百度定位", modifier = Modifier.padding(top = 12.dp), fontSize = 12.sp)
            }
            if (includeLocations) Text("接收者可查看、转发起终点的具体位置。", color = LinLanDanger, fontSize = 12.sp)
            if (fixedRecipient == null) {
            OutlinedTextField(search, { search = it }, enabled = !busy, singleLine = true, label = { Text("搜索好友或群聊") })
            if (loading) LinearProgressIndicator(Modifier.fillMaxWidth())
            error?.let { Text(it, color = LinLanDanger); if (!busy) TextButton(onClick = { reload++ }) { Text("重新加载") } }
            LazyColumn(Modifier.heightIn(max = 220.dp)) {
                val visible = targets.filter { it.name.contains(search, true) }
                if (!loading && visible.isEmpty()) item { Text("暂无可分享的好友或群聊", modifier = Modifier.padding(12.dp)) }
                items(visible, key = { "${it.group}:${it.id}" }) { item ->
                    Row(Modifier.fillMaxWidth().clickable(enabled = !busy) { target = item }.padding(vertical = 5.dp)) {
                        RadioButton(selected = target == item, enabled = !busy, onClick = { target = item })
                        Text((if (item.group) "群聊 · " else "好友 · ") + item.name, modifier = Modifier.padding(top = 12.dp), maxLines = 2)
                    }
                }
            }
            }
            error?.takeIf { fixedRecipient != null }?.let { Text(it, color = LinLanDanger) }
            target?.let { Text("将发送给：${it.name}", color = LinLanAccent, fontSize = 12.sp) }
        }
    }, confirmButton = {
        TextButton(enabled = target != null && !busy && !loading, onClick = {
            val recipient = target ?: return@TextButton
            busy = true; error = null
            scope.launch {
                runCatching {
                    val card = RunShare.from(run, includeShape, includeLocations)
                    LinLanImManager.sendRunCard(card,
                        if (recipient.group) null else recipient.id, if (recipient.group) recipient.id else null)
                }.onSuccess {
                    onSent(if (recipient.group) "group_${recipient.id}" else "c2c_${recipient.id}", recipient.name, recipient.group)
                }.onFailure { error = it.message ?: "发送失败，请重试" }
                busy = false
            }
        }) { Text(if (busy) "发送中…" else "发送") }
    }, dismissButton = { TextButton(enabled = !busy, onClick = onDismiss) { Text("取消") } })
}

private fun openBaiduRunLocation(context: android.content.Context, point: RunLocation, title: String) {
    fun uri(base: String) = android.net.Uri.parse(base).buildUpon()
        .appendQueryParameter("location", "${point.latitude},${point.longitude}")
        .appendQueryParameter("coord_type", "wgs84")
        .appendQueryParameter("title", title)
        .appendQueryParameter("content", "睿之禾运动记录")
        .appendQueryParameter("src", "andr.${context.packageName}")
        .appendQueryParameter("output", "html").build()
    try {
        context.startActivity(android.content.Intent(android.content.Intent.ACTION_VIEW, uri("baidumap://map/marker"))
            .setPackage("com.baidu.BaiduMap"))
    } catch (_: android.content.ActivityNotFoundException) {
        runCatching { context.startActivity(android.content.Intent(android.content.Intent.ACTION_VIEW, uri("https://api.map.baidu.com/marker"))) }
            .onFailure { android.widget.Toast.makeText(context, "未找到可打开地图的应用", android.widget.Toast.LENGTH_LONG).show() }
    } catch (_: Exception) {
        android.widget.Toast.makeText(context, "暂时无法打开百度地图", android.widget.Toast.LENGTH_LONG).show()
    }
}

@Composable
fun ChatRunPicker(peerId: String, title: String, group: Boolean, onDismiss: () -> Unit) {
    val context = androidx.compose.ui.platform.LocalContext.current
    var records by remember { mutableStateOf<List<RunSession>>(emptyList()) }
    var selected by remember { mutableStateOf<RunSession?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(Unit) {
        runCatching {
            val owner = com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer.sessionStore().currentSession().userId
            require(owner.isNotBlank())
            kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) { RunRecords.list(context, owner) }
                .filter { it.points.isNotEmpty() && it.durationSeconds > 0 }
        }.onSuccess { records = it }.onFailure { error = "加载运动记录失败，请关闭后重试" }
        loading = false
    }
    val run = selected
    if (run != null) RunShareDialog(run, onDismiss = { selected = null }, onSent = { _, _, _ -> onDismiss() },
        fixedRecipient = Triple(peerId, title, group))
    else AlertDialog(onDismissRequest = onDismiss, title = { Text("分享运动路线") }, text = {
        LazyColumn(Modifier.heightIn(max = 350.dp)) {
            if (loading) item { Text("正在读取运动记录…") }
            error?.let { item { Text(it, color = LinLanDanger) } }
            if (!loading && error == null && records.isEmpty()) item { Text("还没有可分享的运动。请先在「运动」中完成并保存记录。") }
            items(records) { record ->
                Surface(onClick = { selected = record }, shape = RoundedCornerShape(16.dp), color = LinLanSurface,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 5.dp)) {
                    Row(Modifier.padding(16.dp), verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
                            Text(RunShare.from(record, false).distance, fontWeight = FontWeight.Bold, fontSize = 20.sp, color = LinLanInk)
                            Text(SimpleDateFormat("MM月dd日 HH:mm", Locale.CHINA).format(Date(record.startedAt)), color = LinLanInkSecondary, fontSize = 12.sp)
                            Text("点击选择这条路线", color = LinLanAccent, fontSize = 12.sp)
                        }
                        Text("›", fontSize = 32.sp, color = LinLanAccent)
                    }
                }
            }
        }
    }, confirmButton = { TextButton(onClick = onDismiss) { Text("取消") } })
}
