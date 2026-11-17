package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.ContextCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.run.*
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.*
import com.balabalaim.gcloud.tim.push.appdev.ui.common.PermissionCopy
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun LinLanRunScreen(onOpenChat: (String, String, Boolean) -> Unit = { _, _, _ -> }, recordsOnly: Boolean = false, onOpenRecords: () -> Unit = {}, onOpenPlan: () -> Unit = {}) {
    val context = LocalContext.current
    val session by RunState.session.collectAsState()
    var owner by remember { mutableStateOf("") }
    var records by remember { mutableStateOf<List<RunSession>>(emptyList()) }
    var selected by remember { mutableStateOf<RunSession?>(null) }
    var finish by remember { mutableStateOf(false) }
    var share by remember { mutableStateOf<RunSession?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var runLocationRationale by remember { mutableStateOf(false) }
    val locations = remember { context.getSystemService(android.location.LocationManager::class.java) }
    fun gpsEnabled() = locations.isProviderEnabled(android.location.LocationManager.GPS_PROVIDER)
    var locationEnabled by remember { mutableStateOf(gpsEnabled()) }
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    DisposableEffect(context, lifecycle) {
        val receiver = object : android.content.BroadcastReceiver() {
            override fun onReceive(context: android.content.Context?, intent: Intent?) { locationEnabled = gpsEnabled() }
        }
        ContextCompat.registerReceiver(context, receiver,
            android.content.IntentFilter(android.location.LocationManager.PROVIDERS_CHANGED_ACTION), ContextCompat.RECEIVER_EXPORTED)
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) locationEnabled = gpsEnabled()
        }
        lifecycle.addObserver(observer)
        onDispose { context.unregisterReceiver(receiver); lifecycle.removeObserver(observer) }
    }
    fun command(action: String) {
        val intent = Intent(context, RunTrackingService::class.java).setAction(action).putExtra("owner", owner)
        runCatching { if (action == "start") ContextCompat.startForegroundService(context, intent) else context.startService(intent) }
            .onFailure { error = "无法启动运动记录：${it.message}" }
    }
    val permission = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { grants ->
        if (grants[Manifest.permission.ACCESS_FINE_LOCATION] == true) { selected = null; command("start") }
        else error = "请在系统设置中允许精确定位后开始运动。"
    }
    fun start() {
        if (owner.isBlank()) { error = "请登录后开始运动"; return }
        locationEnabled = gpsEnabled()
        if (!locationEnabled) { error = "请开启系统定位后开始运动"; return }
        error = null
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED) {
            selected = null; command("start")
        } else runLocationRationale = true
    }
    LaunchedEffect(Unit) { owner = LinLanAppContainer.sessionStore().currentSession().userId }
    LaunchedEffect(owner, session.active, session.message) {
        if (owner.isNotBlank() && !session.active) records = withContext(Dispatchers.IO) { RunRecords.list(context, owner) }
    }
    val current = if (session.owner == owner) session else RunSession()
    val display = selected ?: current
    Column(Modifier.fillMaxSize().background(LinLanCanvas).windowInsetsPadding(WindowInsets.statusBars)
        .verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Column { Text(if (recordsOnly) "运动记录" else "运动", fontSize = 30.sp, fontWeight = FontWeight.Bold); Text("记录运动，与朋友一起进步", color = LinLanInkSecondary, fontSize = 13.sp) }
            Icon(Icons.Outlined.DirectionsRun, null, tint = LinLanAccent, modifier = Modifier.size(34.dp))
        }
        if (selected == null && !current.active) RunProgressSummary(records)
        if (!recordsOnly && selected == null) {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Surface(onClick = onOpenPlan, modifier = Modifier.weight(1f), color = Color(0xFFE7F1D4), shape = RoundedCornerShape(20.dp)) {
                    Column(Modifier.padding(18.dp)) { Text("我的运动计划", fontWeight = FontWeight.Bold); Text("设定目标  →", fontSize = 12.sp, modifier = Modifier.padding(top = 8.dp)) }
                }
                Surface(onClick = onOpenRecords, modifier = Modifier.weight(1f), color = Color(0xFFFFE9D7), shape = RoundedCornerShape(20.dp)) {
                    Column(Modifier.padding(18.dp)) { Text("分享我的路线", fontWeight = FontWeight.Bold); Text("回顾与分享  →", fontSize = 12.sp, modifier = Modifier.padding(top = 8.dp)) }
                }
            }
        }
        if (!recordsOnly && selected == null) {
            Text(current.message, color = LinLanInkSecondary, fontSize = 13.sp)
            error?.let { Text(it, color = LinLanDanger) }
            when {
                !current.active -> Button(enabled = current.message != "正在保存运动记录", onClick = { if (session.active && session.owner != owner) error = "另一账号的运动仍在记录，请先结束" else start() }, modifier = Modifier.fillMaxWidth().height(54.dp)) { Text("开始运动", fontSize = 17.sp) }
                else -> Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Button(onClick = { command(if (current.paused) "resume" else "pause") }, modifier = Modifier.weight(1f).height(52.dp)) { Text(if (current.paused) "继续运动" else "暂停") }
                    OutlinedButton(onClick = { finish = true }, modifier = Modifier.weight(1f).height(52.dp)) { Text("结束并保存") }
                }
            }
        } else if (selected != null) TextButton(onClick = { selected = null }) { Text(if (recordsOnly) "返回记录列表" else "返回当前运动") }
        if (!recordsOnly || selected != null) {
        Surface(color = LinLanInk, shape = RoundedCornerShape(24.dp)) {
            Column(Modifier.fillMaxWidth().padding(22.dp)) {
                Text(if (selected == null) "户外跑步 / 步行" else "运动轨迹回顾", color = Color(0xFFCBD5E1), fontSize = 13.sp)
                Row(verticalAlignment = Alignment.Bottom) {
                    Text(String.format(Locale.US, "%.2f", display.distanceMeters / 1000), color = Color.White, fontSize = 50.sp, fontWeight = FontWeight.Bold)
                    Text(" 公里", color = Color(0xFFC4EF69), modifier = Modifier.padding(bottom = 10.dp))
                }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    RunMetric("运动时长", runTime(display.durationSeconds))
                    RunMetric("平均配速", if (display.distanceMeters >= 10) runPace(display) else "—")
                    RunMetric("轨迹点", "${display.points.size}")
                }
            }
        }
        Box(Modifier.fillMaxWidth().height(300.dp).clip(RoundedCornerShape(24.dp))) {
            RoutePreview(display.points, if (selected != null) "已保存的运动轨迹" else if (current.paused) "运动已暂停" else "运动轨迹")
        }
        }
        if (!recordsOnly && selected == null && !locationEnabled) {
            TextButton(onClick = { context.startActivity(Intent(android.provider.Settings.ACTION_LOCATION_SOURCE_SETTINGS)) }) {
                Icon(Icons.Outlined.LocationOn, null, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(6.dp))
                Text("系统定位已关闭 · 去开启")
            }
        }
        if (selected != null && display.points.isNotEmpty() && display.durationSeconds > 0) {
            Button(onClick = { share = display }, modifier = Modifier.fillMaxWidth()) { Text("分享运动打卡给好友 / 群聊") }
            Text("分享成绩、交流进步，让运动成为共同话题。", color = LinLanInkSecondary, fontSize = 12.sp)
        }
        Text("运动记录", fontSize = 21.sp, fontWeight = FontWeight.Bold)
        if (records.isEmpty()) SafetyCard("第一段轨迹，从今天开始", "完成运动后，可在这里查看路线、里程和用时。记录保存在本机。")
        records.forEach { record ->
            Surface(Modifier.fillMaxWidth().clickable { selected = record }, shape = RoundedCornerShape(18.dp), color = LinLanSurface) {
                Row(Modifier.padding(18.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Column { Text(SimpleDateFormat("MM月dd日 HH:mm", Locale.CHINA).format(Date(record.startedAt)), fontWeight = FontWeight.Bold)
                        Text("${runTime(record.durationSeconds)} · 查看轨迹", fontSize = 12.sp, color = LinLanInkSecondary) }
                    Text(String.format(Locale.US, "%.2f km", record.distanceMeters / 1000), color = LinLanAccent, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
    share?.let { record ->
        RunShareDialog(record, onDismiss = { share = null }, onSent = { id, title, group ->
            share = null
            onOpenChat(id, title, group)
        })
    }
    if (runLocationRationale) LocationPermissionExplanation(
        onContinue = { runLocationRationale = false; permission.launch(arrayOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION)) },
        onCancel = { runLocationRationale = false },
    )
    if (finish) AlertDialog(onDismissRequest = { finish = false }, title = { Text("结束本次运动？") },
        text = { Text("将保存已经记录的轨迹、距离和时长。") }, confirmButton = { TextButton(onClick = { finish = false; command("stop") }) { Text("结束并保存") } },
        dismissButton = { TextButton(onClick = { finish = false }) { Text("继续运动") } })

}

private fun runTime(seconds: Long) = String.format(Locale.US, "%02d:%02d:%02d", seconds / 3600, seconds / 60 % 60, seconds % 60)
private fun runPace(run: RunSession): String { val seconds = (run.durationSeconds * 1000 / run.distanceMeters).toLong(); return String.format(Locale.US, "%d′%02d″", seconds / 60, seconds % 60) }
@Composable private fun RunMetric(label: String, value: String) {
    Column { Text(value, color = Color.White, fontWeight = FontWeight.SemiBold, fontSize = 18.sp); Text(label, color = Color(0xFFCBD5E1), fontSize = 11.sp) }
}

@Composable
private fun LocationPermissionExplanation(onContinue: () -> Unit, onCancel: () -> Unit) {
    AlertDialog(onDismissRequest = onCancel, title = { Text("睿之禾定位权限说明") },
        text = { Text(PermissionCopy.LOCATION, modifier = Modifier.verticalScroll(rememberScrollState())) },
        confirmButton = { TextButton(onClick = onContinue) { Text("继续授权") } },
        dismissButton = { TextButton(onClick = onCancel) { Text("暂不开启") } })
}

@Composable
private fun RoutePreview(points: List<TrackPoint>, title: String) {
    val projected = remember(points) { RouteProjection.project(points) }
    Box(Modifier.fillMaxSize().background(LinLanSurface)) {
        Canvas(Modifier.fillMaxSize().padding(start = 26.dp, end = 26.dp, top = 48.dp, bottom = 64.dp)) {
            val grid = LinLanInkSecondary.copy(alpha = 0.10f)
            for (step in 0..4) {
                val fraction = step / 4f
                drawLine(grid, Offset(size.width * fraction, 0f), Offset(size.width * fraction, size.height), 1.dp.toPx())
                drawLine(grid, Offset(0f, size.height * fraction), Offset(size.width, size.height * fraction), 1.dp.toPx())
            }
            // Use one scale for both axes so the route keeps its real proportions.
            val side = minOf(size.width, size.height)
            fun position(p: RoutePosition) = Offset(
                (size.width - side) / 2 + p.x * side,
                (size.height - side) / 2 + p.y * side,
            )
            projected.zipWithNext().forEach { (a, b) ->
                if (a.segment == b.segment) drawLine(LinLanAccent, position(a), position(b),
                    strokeWidth = 3.dp.toPx(), cap = androidx.compose.ui.graphics.StrokeCap.Round)
            }
            if (projected.isNotEmpty()) {
                drawCircle(LinLanInk, 6.dp.toPx(), position(projected.first()))
                drawCircle(Color.White, 7.dp.toPx(), position(projected.last()))
                drawCircle(LinLanAccent, 4.dp.toPx(), position(projected.last()))
            }
        }
        Row(Modifier.align(Alignment.TopStart).fillMaxWidth().padding(16.dp), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(title, fontWeight = FontWeight.Bold)
            Text("北 ↑", color = LinLanInkSecondary, fontSize = 12.sp)
        }
        if (points.isEmpty()) Text("开始运动后，在室外等待 GPS 定位",
            modifier = Modifier.align(Alignment.Center).background(LinLanSurface.copy(alpha = 0.9f)).padding(12.dp),
            color = LinLanInkSecondary, fontSize = 13.sp)
        Column(Modifier.align(Alignment.BottomCenter).padding(12.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            if (points.isNotEmpty()) Text(if (points.size == 1) "已记录起点，继续移动即可形成轨迹" else "深色为起点 · 橙色为当前位置或终点",
                color = LinLanInkSecondary, fontSize = 11.sp)
            Text("GPS 轨迹 · 不显示道路底图", color = LinLanInkSecondary, fontSize = 11.sp)
        }
    }
}

@Composable
private fun RunProgressSummary(records: List<RunSession>) {
    val now = java.util.Calendar.getInstance()
    val weekStart = (now.clone() as java.util.Calendar).apply {
        set(java.util.Calendar.HOUR_OF_DAY, 0); set(java.util.Calendar.MINUTE, 0)
        set(java.util.Calendar.SECOND, 0); set(java.util.Calendar.MILLISECOND, 0)
        add(java.util.Calendar.DAY_OF_YEAR, -((get(java.util.Calendar.DAY_OF_WEEK) + 5) % 7))
    }.timeInMillis
    val week = records.filter { it.startedAt >= weekStart && it.points.isNotEmpty() }
    val days = week.map { SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date(it.startedAt)) }.distinct().size
    Surface(color = LinLanSurface, shape = RoundedCornerShape(20.dp)) {
        Column(Modifier.fillMaxWidth().padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("本周的每一步", fontWeight = FontWeight.Bold)
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text(String.format(Locale.US, "%.2f 公里", week.sumOf { it.distanceMeters } / 1000), color = LinLanAccent, fontWeight = FontWeight.Bold)
                Text("${week.size} 次运动 · $days 天打卡", color = LinLanInkSecondary, fontSize = 13.sp)
            }
            Text("完成运动 → 查看记录 → 分享给朋友", color = LinLanInkSecondary, fontSize = 12.sp)
        }
    }
}
