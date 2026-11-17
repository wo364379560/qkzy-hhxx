package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.run.*
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.Calendar
import java.util.Locale

@Composable
fun RunPlanScreen(onStart: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val today = PlanRules.day(System.currentTimeMillis())
    var selectedDay by remember { mutableStateOf(today) }
    var month by remember { mutableStateOf(today.substring(0, 7)) }
    var plans by remember { mutableStateOf<List<RunPlan>>(emptyList()) }
    var records by remember { mutableStateOf<List<RunSession>>(emptyList()) }
    var store by remember { mutableStateOf<RunPlanStore?>(null) }
    var busy by remember { mutableStateOf(false) }
    var loaded by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var reload by remember { mutableIntStateOf(0) }
    var editing by remember { mutableStateOf<RunPlan?>(null) }
    var linking by remember { mutableStateOf<RunPlan?>(null) }
    var deleting by remember { mutableStateOf<RunPlan?>(null) }
    var detail by remember { mutableStateOf<RunSession?>(null) }
    var goal by remember { mutableIntStateOf(10) }
    var goalKey by remember { mutableStateOf("") }
    val goalPrefs = remember { context.getSharedPreferences("run_goals", android.content.Context.MODE_PRIVATE) }
    LaunchedEffect(reload) {
        loaded = false; error = null
        runCatching {
            val owner = LinLanAppContainer.sessionStore().currentSession().userId
            val result = withContext(Dispatchers.IO) {
                val storage = RunPlanStore(context, owner)
                Triple(storage, storage.load(), RunRecords.list(context, owner, includeDraft = false).filter { it.points.isNotEmpty() && it.durationSeconds > 0 })
            }
            store = result.first; plans = result.second; records = result.third
            goalKey = java.security.MessageDigest.getInstance("SHA-256").digest(owner.toByteArray()).joinToString("") { "%02x".format(it) }
            goal = goalPrefs.getInt(goalKey, 10).coerceIn(1, 100)
            loaded = true
        }.onFailure { error = "读取计划失败：${it.message}" }
    }
    fun save(next: List<RunPlan>, onSuccess: () -> Unit = {}) {
        if (!loaded || busy) return
        busy = true; error = null
        scope.launch {
            runCatching { withContext(Dispatchers.IO) { requireNotNull(store).save(next) } }
                .onSuccess { plans = next; onSuccess() }.onFailure { error = it.message ?: "保存失败，请重试" }
            busy = false
        }
    }
    fun shiftMonth(amount: Int) {
        val calendar = PlanRules.calendar("$month-01"); calendar.add(Calendar.MONTH, amount)
        month = PlanRules.day(calendar.timeInMillis).substring(0, 7)
    }
    Column(Modifier.fillMaxSize().background(LinLanCanvas).windowInsetsPadding(WindowInsets.statusBars)
        .verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text("运动日历", fontSize = 30.sp, fontWeight = FontWeight.Bold, color = LinLanInk)
        Text("把想出发的日子，提前记下来", color = LinLanInkSecondary)
        if (!loaded || busy) LinearProgressIndicator(Modifier.fillMaxWidth())
        error?.let { Text(it, color = LinLanDanger); if (!loaded) TextButton(onClick = { reload++ }) { Text("重试") } }
        Surface(color = LinLanSurface, shape = RoundedCornerShape(22.dp)) {
            Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.SpaceBetween) {
                    TextButton(onClick = { shiftMonth(-1) }) { Text("‹", fontSize = 25.sp) }
                    Text(month.replace("-", "年") + "月", fontWeight = FontWeight.Bold, fontSize = 19.sp)
                    TextButton(onClick = { shiftMonth(1) }) { Text("›", fontSize = 25.sp) }
                    TextButton(onClick = { month = today.substring(0, 7); selectedDay = today }) { Text("今天") }
                }
                Row { listOf("一", "二", "三", "四", "五", "六", "日").forEach { Text(it, modifier = Modifier.weight(1f), textAlign = androidx.compose.ui.text.style.TextAlign.Center, color = LinLanInkSecondary) } }
                val cal = PlanRules.calendar("$month-01")
                val offset = (cal.get(Calendar.DAY_OF_WEEK) + 5) % 7
                val days = cal.getActualMaximum(Calendar.DAY_OF_MONTH)
                for (week in 0 until (offset + days + 6) / 7) {
                    Row {
                        for (column in 0..6) {
                            val day = week * 7 + column - offset + 1
                            if (day !in 1..days) Spacer(Modifier.weight(1f).height(49.dp))
                            else {
                                val date = "$month-${day.toString().padStart(2, '0')}"
                                val count = plans.count { it.day == date }
                                Surface(onClick = { selectedDay = date }, modifier = Modifier.weight(1f).height(49.dp), shape = RoundedCornerShape(12.dp),
                                    color = if (date == selectedDay) LinLanInk else Color.Transparent) {
                                    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                                        Text(day.toString(), color = if (date == selectedDay) Color.White else if (date == today) LinLanAccent else LinLanInk, fontWeight = FontWeight.Bold)
                                        Text(if (count > 0) "$count 项" else "", fontSize = 9.sp, color = if (date == selectedDay) Color(0xFFC4EF69) else LinLanAccent)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(selectedDay, fontSize = 19.sp, fontWeight = FontWeight.Bold)
            Button(enabled = loaded && !busy, onClick = { editing = RunPlan(day = selectedDay, target = 3.0) }) { Text("＋ 新增计划") }
        }
        val daily = plans.filter { it.day == selectedDay }
        if (loaded && daily.isEmpty()) SafetyCard("这一天还没有计划", "点击新增，安排一次跑步或步行。")
        daily.forEach { plan ->
            val record = records.firstOrNull { it.startedAt == plan.recordId }
            val progress = PlanRules.progress(plan, record)
            val state = if (record != null) { if (progress >= 1) "已完成" else "已关联 · 未达目标" }
                else if (plan.recordId != null) "关联记录不可用" else if (plan.day < today) "已过期 · 未完成" else "待完成"
            Surface(color = LinLanSurface, shape = RoundedCornerShape(20.dp)) {
                Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text("${plan.activity} · ${String.format(Locale.US, "%g", plan.target).trimEnd('0').trimEnd('.')} ${plan.unit}", fontWeight = FontWeight.Bold, fontSize = 18.sp)
                        Text(state, fontSize = 12.sp, color = LinLanAccent)
                    }
                    if (plan.note.isNotBlank()) Text(plan.note, color = LinLanInkSecondary, fontSize = 13.sp)
                    if (record != null) {
                        LinearProgressIndicator(progress = { progress.toFloat().coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth())
                        Text("实际 ${String.format(Locale.US, "%.2f", record.distanceMeters / 1000)} 公里 · ${record.durationSeconds / 60} 分钟", fontSize = 12.sp)
                        TextButton(onClick = { detail = record }) { Text("查看关联轨迹 ›") }
                    }
                    Row {
                        TextButton(enabled = !busy, onClick = { editing = plan }) { Text("编辑") }
                        TextButton(enabled = !busy, onClick = { linking = plan }) { Text("关联运动") }
                        if (plan.recordId != null) TextButton(enabled = !busy, onClick = { save(plans.map { if (it.id == plan.id) it.copy(recordId = null) else it }) }) { Text("解除") }
                        TextButton(enabled = !busy, onClick = { deleting = plan }) { Text("删除", color = LinLanDanger) }
                    }
                }
            }
        }
        Button(onClick = onStart, modifier = Modifier.fillMaxWidth()) { Text("去运动  →") }
        Text("完成并保存运动后，回到当天计划点「关联运动」。达到目标才显示已完成，同一记录只能关联一项计划。", color = LinLanInkSecondary, fontSize = 12.sp)
        Text("本周里程目标", fontWeight = FontWeight.Bold, fontSize = 19.sp)
        val weekStart = PlanRules.calendar(today).apply { add(Calendar.DAY_OF_YEAR, -((get(Calendar.DAY_OF_WEEK) + 5) % 7)) }.timeInMillis
        val total = records.filter { it.startedAt >= weekStart }.sumOf { it.distanceMeters } / 1000
        Text(String.format(Locale.US, "%.2f / %d 公里", total, goal), color = LinLanAccent)
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) { listOf(5, 10, 20).forEach { value ->
            FilterChip(selected = value == goal, enabled = loaded && !busy, onClick = { goal = value; goalPrefs.edit().putInt(goalKey, value).apply() }, label = { Text("$value 公里") })
        } }
        Text("计划按账号保存在本机，退出再登录仍可查看。", fontSize = 12.sp, color = LinLanInkSecondary)
    }
    editing?.let { plan -> PlanEditor(plan, busy, error, onDismiss = { editing = null }, onSave = { updated ->
        save(plans.filterNot { it.id == updated.id } + updated) { editing = null }
    }) }
    deleting?.let { plan -> AlertDialog(onDismissRequest = { if (!busy) deleting = null }, title = { Text("删除这项计划？") },
        text = { Text("只删除计划，已保存的运动记录仍会保留。") }, confirmButton = { TextButton(enabled = !busy, onClick = { save(plans.filterNot { it.id == plan.id }) { deleting = null } }) { Text("删除", color = LinLanDanger) } },
        dismissButton = { TextButton(enabled = !busy, onClick = { deleting = null }) { Text("取消") } }) }
    linking?.let { plan -> AlertDialog(onDismissRequest = { if (!busy) linking = null }, title = { Text("关联 ${plan.day} 的运动") }, text = {
        Column {
            error?.let { Text(it, color = LinLanDanger) }
            val choices = records.filter { PlanRules.day(it.startedAt) == plan.day && plans.none { p -> p.id != plan.id && p.recordId == it.startedAt } }
            if (choices.isEmpty()) Text("当天暂无可关联记录，请先完成并保存运动。")
            LazyColumn(Modifier.heightIn(max = 280.dp)) { items(choices) { record ->
                Text("${java.text.SimpleDateFormat("HH:mm", Locale.CHINA).format(java.util.Date(record.startedAt))} · ${String.format(Locale.US, "%.2f", record.distanceMeters / 1000)} 公里  ›",
                    modifier = Modifier.fillMaxWidth().clickable(enabled = !busy) {
                        runCatching { PlanRules.link(plans, plan, record) }.onSuccess { linked ->
                            save(plans.map { if (it.id == plan.id) linked else it }) { linking = null }
                        }.onFailure { error = it.message }
                    }.padding(16.dp))
            } }
        }
    }, confirmButton = { TextButton(enabled = !busy, onClick = { linking = null }) { Text("取消") } }) }
    detail?.let { RunShareDetail(RunShare.from(it, true)) { detail = null } }
}

@Composable
private fun PlanEditor(plan: RunPlan, busy: Boolean, saveError: String?, onDismiss: () -> Unit, onSave: (RunPlan) -> Unit) {
    var date by remember(plan) { mutableStateOf(plan.day) }
    var activity by remember(plan) { mutableStateOf(plan.activity) }
    var unit by remember(plan) { mutableStateOf(plan.unit) }
    var target by remember(plan) { mutableStateOf(plan.target.toString()) }
    var note by remember(plan) { mutableStateOf(plan.note) }
    var error by remember { mutableStateOf<String?>(null) }
    AlertDialog(onDismissRequest = { if (!busy) onDismiss() }, title = { Text("保存运动计划") }, text = {
        Column(Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(date, { date = it }, enabled = !busy, label = { Text("日期 yyyy-MM-dd") }, singleLine = true)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { listOf("跑步", "步行").forEach { value -> FilterChip(selected = activity == value, enabled = !busy, onClick = { activity = value }, label = { Text(value) }) } }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) { listOf("公里", "分钟").forEach { value -> FilterChip(selected = unit == value, enabled = !busy, onClick = { unit = value; target = if (value == "公里") "3" else "30" }, label = { Text(value) }) } }
            OutlinedTextField(target, { target = it }, enabled = !busy, label = { Text("目标（$unit）") }, singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal))
            OutlinedTextField(note, { note = it.take(200) }, enabled = !busy, label = { Text("备注（选填）") }, maxLines = 3)
            (error ?: saveError)?.let { Text(it, color = LinLanDanger) }
            if (plan.recordId != null) Text("修改日期或运动类型后，需要重新关联记录。", fontSize = 12.sp)
        }
    }, confirmButton = { TextButton(enabled = !busy, onClick = {
        runCatching {
            val updated = plan.copy(day = date.trim(), activity = activity, unit = unit, target = target.toDoubleOrNull() ?: Double.NaN, note = note.trim(),
                recordId = if (date.trim() == plan.day && activity == plan.activity) plan.recordId else null)
            PlanRules.validate(updated); updated
        }.onSuccess(onSave).onFailure { error = it.message ?: "请检查填写内容" }
    }) { Text(if (busy) "保存中…" else "保存") } }, dismissButton = { TextButton(enabled = !busy, onClick = onDismiss) { Text("取消") } })
}
