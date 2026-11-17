package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.app.Activity
import android.app.KeyguardManager
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.distinctUntilChanged
import org.json.JSONObject

internal object DeviceCredentialFlow { var inProgress = false }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SafetyPage(title: String, onBack: () -> Unit, content: @Composable ColumnScope.() -> Unit) {
    Scaffold(containerColor = LinLanCanvas, topBar = {
        TopAppBar(title = { Text(title, fontWeight = FontWeight.Bold) },
            navigationIcon = { LinLanNavBackButton(onClick = onBack) },
            colors = TopAppBarDefaults.topAppBarColors(containerColor = LinLanCanvas))
    }) { padding ->
        Column(Modifier.padding(padding).fillMaxSize().verticalScroll(rememberScrollState())
            .padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp), content = content)
    }
}

@Composable
fun SafetyCard(title: String, detail: String, content: @Composable ColumnScope.() -> Unit = {}) {
    Surface(color = LinLanSurface, shape = RoundedCornerShape(20.dp)) {
        Column(Modifier.fillMaxWidth().padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text(title, fontWeight = FontWeight.Bold, color = LinLanInk, fontSize = 17.sp)
            if (detail.isNotBlank()) Text(detail, color = LinLanInkSecondary, fontSize = 13.sp)
            content()
        }
    }
}

@Composable
private fun RequestError(message: String?, retry: () -> Unit) {
    if (message != null) {
        Text(message, color = LinLanDanger)
        TextButton(onClick = retry) { Text("重新加载") }
    }
}

@Composable
fun LinLanRealnameScreen(onBack: () -> Unit) {
    val api = remember { LinLanAppContainer.apiClient() }
    val scope = rememberCoroutineScope()
    var status by remember { mutableStateOf<JSONObject?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }
    var id by remember { mutableStateOf("") }
    fun load() { scope.launch {
        busy = true; error = null
        runCatching { api.realnameStatus() }.onSuccess { status = it }.onFailure { error = it.message }
        busy = false
    } }
    LaunchedEffect(Unit) { load() }
    SafetyPage("实名认证", onBack) {
        Icon(Icons.Outlined.VerifiedUser, null, tint = LinLanAccent, modifier = Modifier.size(44.dp))
        Text("让每一次交流，多一份安心", fontSize = 23.sp, fontWeight = FontWeight.Bold)
        RequestError(error, ::load)
        if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
        val current = status
        when {
            current == null -> Unit
            !current.optBoolean("feature_enabled") -> SafetyCard("实名认证暂未开放", "平台尚未开启此功能，请稍后再试。")
            current.optBoolean("verified") -> SafetyCard("已完成实名认证", "认证信息已由平台验证") {
                Text(current.optString("real_name_masked"))
                Text(current.optString("id_number_masked"))
            }
            else -> SafetyCard("填写身份信息", "仅用于身份核验，不会向其他用户展示你的完整姓名或证件号码。") {
                OutlinedTextField(name, { name = it.take(50) }, label = { Text("真实姓名") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
                OutlinedTextField(id, { id = it.trim().uppercase().take(18) }, label = { Text("身份证号码") }, modifier = Modifier.fillMaxWidth(), singleLine = true)
                Button(enabled = !busy, onClick = {
                    if (name.isBlank() || !Regex("([0-9]{15}|[0-9]{17}[0-9X])").matches(id)) {
                        error = "请填写真实姓名和有效的身份证号码"
                    } else scope.launch {
                        busy = true; error = null
                        runCatching { api.verifyRealname(name.trim(), id); api.realnameStatus() }
                            .onSuccess { status = it; name = ""; id = "" }.onFailure { error = it.message }
                        busy = false
                    }
                }, modifier = Modifier.fillMaxWidth()) { Text("提交认证") }
            }
        }
    }
}

@Composable
fun RealnameChatBanner(peerId: String, isGroup: Boolean, onVerify: () -> Unit, onReady: () -> Unit = {}) {
    var status by remember(peerId) { mutableStateOf<JSONObject?>(null) }
    var peer by remember(peerId) { mutableStateOf<Boolean?>(null) }
    val owner = LocalLifecycleOwner.current
    val scope = rememberCoroutineScope()
    val readyCallback by rememberUpdatedState(onReady)
    var loading by remember(peerId) { mutableStateOf(false) }
    fun reload() {
        if (loading) return
        loading = true
        scope.launch {
            try {
                val api = LinLanAppContainer.apiClient()
                val nextStatus = runCatching { api.realnameStatus() }.getOrNull()
                val nextPeer = if (!isGroup) runCatching { api.peerRealname(peerId) }.getOrNull() else null
                // Publish both rows together so the conversation viewport changes only once.
                status = nextStatus
                peer = nextPeer
            } finally {
                loading = false
                readyCallback()
            }
        }
    }
    DisposableEffect(owner, peerId) {
        val observer = LifecycleEventObserver { _, event -> if (event == Lifecycle.Event.ON_RESUME) reload() }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer) }
    }
    if (status?.optBoolean("feature_enabled") == true) {
        if (status?.optBoolean("verified") == false) {
            Row(Modifier.fillMaxWidth().background(Color(0xFFFFEBDD)).clickable(onClick = onVerify).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.VerifiedUser, null, tint = LinLanAccent, modifier = Modifier.size(20.dp))
                Text("请先完成实名认证，保障账号与聊天安全", fontSize = 12.sp, modifier = Modifier.weight(1f).padding(horizontal = 8.dp))
                Text("去认证", color = LinLanAccent, fontWeight = FontWeight.Bold, fontSize = 12.sp)
            }
        }
        if (!isGroup && peer == false) Text("对方尚未实名认证，请谨慎核实身份，勿轻信转账要求。",
            fontSize = 12.sp, color = LinLanInkSecondary, modifier = Modifier.fillMaxWidth().background(LinLanSurface).padding(12.dp))
    }
}

@Composable
fun LinLanDevicesScreen(onBack: () -> Unit) {
    val api = remember { LinLanAppContainer.apiClient() }
    val scope = rememberCoroutineScope()
    var devices by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var challenges by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var confirm by remember { mutableStateOf<JSONObject?>(null) }
    suspend fun refresh() {
        devices = api.loginDevices()
        challenges = api.loginChallenges()
    }
    fun load() { scope.launch {
        busy = true; error = null
        runCatching { refresh() }.onFailure { error = it.message }
        busy = false
    } }
    LaunchedEffect(Unit) { load() }
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(lifecycle) { while (true) {
        delay(10000)
        if (lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED) && !busy) load()
    } }
    SafetyPage("登录设备管理", onBack) {
        SafetyCard("你的账号，由你掌握", "删除其他设备会撤销其登录与信任状态。开启设备保护后，该设备再次登录需要重新验证。")
        RequestError(error, ::load)
        if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
        challenges.forEach { item ->
            SafetyCard("待验证 · ${item.optString("device_name")}", "${item.optString("platform")} · ${item.optString("ip")}\n验证码 ${item.optString("short_code")}\n有效期至 ${item.optString("expires_at")}") {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf("仅允许本次登录" to (true to false), "信任此设备并登录" to (true to true), "拒绝登录" to (false to false)).forEach { (label, choice) ->
                        OutlinedButton(enabled = !busy, onClick = {
                            scope.launch {
                                busy = true; error = null
                                runCatching { api.decideChallenge(item.optLong("id"), choice.first, choice.second); refresh() }.onFailure { error = it.message }
                                busy = false
                            }
                        }, modifier = Modifier.fillMaxWidth()) { Text(label) }
                    }
                }
            }
        }
        Text("已登录设备 · ${devices.size}", fontWeight = FontWeight.Bold)
        if (!busy && devices.isEmpty() && error == null) Text("暂无登录设备记录", color = LinLanInkSecondary)
        devices.forEach { item ->
            SafetyCard(item.optString("device_name", "未知设备") + if (item.optBoolean("is_current")) " · 本机" else "",
                "${item.optString("platform")} · ${if (item.optBoolean("is_online")) "在线" else "离线"}\n${item.optString("last_login_at", item.optString("last_used_at"))}\n${item.optString("ip")}") {
                if (item.optString("device_id").isNotBlank()) OutlinedButton(enabled = !busy, onClick = { confirm = item }) {
                    Icon(if (item.optBoolean("is_current")) Icons.Outlined.Logout else Icons.Outlined.DeleteOutline,
                        null, tint = LinLanDanger, modifier = Modifier.size(18.dp))
                    Spacer(Modifier.width(6.dp))
                    Text(if (item.optBoolean("is_current")) "退出本机" else "删除设备", color = LinLanDanger)
                }
            }
        }
        TextButton(onClick = ::load, enabled = !busy) { Text("刷新设备列表") }
    }
    confirm?.let { item -> AlertDialog(onDismissRequest = { confirm = null }, title = { Text(if (item.optBoolean("is_current")) "退出本机登录？" else "删除此设备？") },
        text = { Text(if (item.optBoolean("is_current")) "确认后将退出当前账号，需要重新登录。"
            else "删除「${item.optString("device_name")}」并撤销该设备的登录与信任状态。开启设备保护后，再次登录需要其他手机批准或短信验证。") },
        confirmButton = { TextButton(onClick = {
            confirm = null
            scope.launch { busy = true; error = null; runCatching {
                api.revokeDevice(item.optString("device_id"))
                if (item.optBoolean("is_current")) LinLanAppContainer.authRepository().logout() else refresh()
            }
                .onFailure { error = it.message }; busy = false }
        }) { Text(if (item.optBoolean("is_current")) "确认退出" else "确认删除") } }, dismissButton = { TextButton(onClick = { confirm = null }) { Text("取消") } }) }
}

@Composable
fun LinLanDeviceProtectionScreen(onBack: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }
    val prefs = remember { context.getSharedPreferences("rzh_device_lock", Context.MODE_PRIVATE) }
    var userId by remember { mutableStateOf("") }
    var settings by remember { mutableStateOf<JSONObject?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var locked by remember { mutableStateOf(false) }
    var desiredLock by remember { mutableStateOf(false) }
    val credential = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        DeviceCredentialFlow.inProgress = false
        if (result.resultCode == Activity.RESULT_OK && userId.isNotBlank()) {
            locked = desiredLock; prefs.edit().putBoolean(userId, locked).apply()
        }
    }
    fun load() { scope.launch {
        busy = true; error = null
        runCatching { api.securitySettings() }.onSuccess { settings = it }.onFailure { error = it.message }
        busy = false
    } }
    LaunchedEffect(Unit) {
        userId = LinLanAppContainer.sessionStore().currentSession().userId
        locked = prefs.getBoolean(userId, false); load()
    }
    SafetyPage("设备保护", onBack) {
        SafetyCard("账号保护", "开启新设备验证后，陌生设备需要经过你的确认或短信验证才能登录。")
        RequestError(error, ::load)
        if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
        listOf("login_verification" to "新设备登录验证", "remote_login_alert" to "异地登录提醒").forEach { (key, title) ->
            SafetyCard(title, if (key == "login_verification") "新手机登录需由已登录手机批准，或使用账号绑定手机号的短信验证码验证。已信任设备无需重复验证。" else "发现异常登录及时提醒") {
                Switch(checked = settings?.optBoolean(key) == true, enabled = settings != null && !busy, onCheckedChange = { enabled ->
                    scope.launch { busy = true; error = null
                        runCatching { api.updateSecuritySetting(key, enabled) }.onSuccess { settings = it }.onFailure { error = it.message }
                        busy = false
                    }
                })
            }
        }
        SafetyCard("设备锁", "返回睿之禾时，使用系统锁屏凭据验证身份。仅作用于本机当前账号。") {
            Switch(checked = locked, enabled = userId.isNotBlank(), onCheckedChange = { value ->
                val manager = context.getSystemService(Context.KEYGUARD_SERVICE) as KeyguardManager
                val intent = manager.createConfirmDeviceCredentialIntent("验证身份", "确认后${if (value) "开启" else "关闭"}睿之禾设备锁")
                if (intent == null) error = "请先在系统设置中设置锁屏密码"
                else { desiredLock = value; DeviceCredentialFlow.inProgress = true; credential.launch(intent) }
            })
        }
    }
}

@Composable
fun DeviceLockGate(content: @Composable () -> Unit) {
    val context = LocalContext.current
    val prefs = remember { context.getSharedPreferences("rzh_device_lock", Context.MODE_PRIVATE) }
    val owner = LocalLifecycleOwner.current
    val scope = rememberCoroutineScope()
    var locked by remember { mutableStateOf(false) }
    var authenticating by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var ready by remember { mutableStateOf(false) }
    var accountId by remember { mutableStateOf("") }
    val savedContent = androidx.compose.runtime.saveable.rememberSaveableStateHolder()
    DisposableEffect(prefs, accountId) {
        val activity = context as? Activity
        fun updatePrivacy() {
            val protectionEnabled = prefs.getBoolean(accountId, false)
            if (protectionEnabled) activity?.window?.addFlags(android.view.WindowManager.LayoutParams.FLAG_SECURE)
            else activity?.window?.clearFlags(android.view.WindowManager.LayoutParams.FLAG_SECURE)
        }
        val listener = android.content.SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> updatePrivacy() }
        updatePrivacy(); prefs.registerOnSharedPreferenceChangeListener(listener)
        onDispose { prefs.unregisterOnSharedPreferenceChangeListener(listener) }
    }
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        authenticating = false
        if (result.resultCode == Activity.RESULT_OK) locked = false
    }
    suspend fun refreshLock() {
        val session = LinLanAppContainer.sessionStore().currentSession()
        accountId = session.userId
        locked = session.isLoggedIn && prefs.getBoolean(session.userId, false)
        ready = true
    }
    LaunchedEffect(Unit) {
        LinLanAppContainer.sessionStore().sessionFlow.map { it.userId }.distinctUntilChanged().collect { refreshLock() }
    }
    DisposableEffect(owner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_STOP && !authenticating && !DeviceCredentialFlow.inProgress) { ready = false; scope.launch { refreshLock() } }
        }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer) }
    }
    Box(Modifier.fillMaxSize()) {
    if (ready && !locked) savedContent.SaveableStateProvider("unlocked_app") { content() }
    androidx.activity.compose.BackHandler(locked || !ready) {}
    if (!ready || locked) Box(Modifier.fillMaxSize().background(LinLanCanvas).clickable(interactionSource = remember { androidx.compose.foundation.interaction.MutableInteractionSource() }, indication = null) {}, contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(20.dp)) {
            Icon(Icons.Outlined.Lock, null, tint = LinLanAccent, modifier = Modifier.size(48.dp))
            Text("睿之禾 · 设备已锁定", fontWeight = FontWeight.Bold, fontSize = 24.sp)
            error?.let { Text(it, color = LinLanDanger) }
            Button(enabled = ready && !authenticating, onClick = {
                val manager = context.getSystemService(Context.KEYGUARD_SERVICE) as KeyguardManager
                val intent = manager.createConfirmDeviceCredentialIntent("解锁睿之禾", "请验证你的身份")
                if (intent == null) error = "系统验证不可用，请检查锁屏设置" else { authenticating = true; launcher.launch(intent) }
            }) { Text("验证并解锁") }
        }
    }
    }
}

@Composable
fun LinLanHelpScreen(onBack: () -> Unit) {
    var text by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("问题反馈") }
    var busy by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    SafetyPage("帮助与反馈", onBack) {
        SafetyCard("常见问题", "添加好友：通讯录右上角 +，通过手机号或用户 ID 搜索。\n\n消息发送失败：检查网络；群聊禁言请联系群主。\n\n注销账号：我的 → 隐私与安全 → 注销账号。\n\n运动定位：在系统中允许精确定位，并在室外开始运动。")
        SafetyCard("告诉我们你的想法", "我们会将反馈发送给平台处理。") {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                listOf("问题反馈", "功能建议", "其他").forEach { type -> FilterChip(selected = category == type, onClick = { category = type }, label = { Text(type) }) }
            }
            OutlinedTextField(text, { text = it.take(500) }, label = { Text("描述问题或建议") }, minLines = 5, modifier = Modifier.fillMaxWidth(), supportingText = { Text("${text.length}/500") })
            message?.let { Text(it) }
            Button(enabled = !busy && text.isNotBlank(), onClick = { scope.launch {
                busy = true; message = null
                runCatching { LinLanAppContainer.apiClient().submitReport("app_feedback", "user", "睿之禾意见反馈", "other", "[$category] ${text.trim()}") }
                    .onSuccess { text = ""; message = "反馈已提交，感谢你的意见" }.onFailure { message = it.message }
                busy = false
            } }, modifier = Modifier.fillMaxWidth()) { Text(if (busy) "正在提交…" else "提交反馈") }
        }
    }
}

internal fun isHuaweiDevice(): Boolean = Build.MANUFACTURER.equals("HUAWEI", true) || Build.BRAND.equals("HUAWEI", true)

private val antiFraudTitles = listOf(
    "安全提醒：警惕冒充好友与陌生链接",
    "防骗指南：远离虚假投资与交友骗局",
    "三条底线：不下载、不转账、不裸聊",
    "特别警示：警惕外部平台引流诈骗",
)

@Composable
fun HuaweiAntiFraudBanner(onOpen: () -> Unit) {
    if (!isHuaweiDevice()) return
    Row(Modifier.padding(horizontal = 16.dp, vertical = 8.dp).fillMaxWidth()
        .background(LinLanSurface, RoundedCornerShape(16.dp)).clickable(onClick = onOpen).padding(14.dp),
        verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(46.dp).background(Color(0xFFFFEBDD), RoundedCornerShape(13.dp)), contentAlignment = Alignment.Center) {
            Icon(Icons.Outlined.GppGood, null, tint = LinLanAccent)
        }
        Column(Modifier.weight(1f).padding(horizontal = 12.dp)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text("反诈公告", fontWeight = FontWeight.Bold, fontSize = 16.sp)
                Text("置顶", color = LinLanAccent, fontSize = 11.sp)
            }
            Text("4 篇安全提醒 · 查看防骗指南", fontSize = 12.sp, color = LinLanInkSecondary)
        }
        Icon(Icons.Outlined.ChevronRight, null, tint = LinLanInkSecondary)
    }
}

@Composable
fun LinLanAntiFraudScreen(onBack: () -> Unit) {
    val context = LocalContext.current
    var selected by androidx.compose.runtime.saveable.rememberSaveable { mutableStateOf<Int?>(null) }
    androidx.activity.compose.BackHandler(selected != null) { selected = null }
    val article = selected
    if (article == null) {
        SafetyPage("反诈公告", onBack) {
            Text("多一分警惕，少一分风险", fontSize = 23.sp, fontWeight = FontWeight.Bold)
            Text("睿之禾安全中心 · 4 篇公告", color = LinLanInkSecondary, fontSize = 13.sp)
            antiFraudTitles.forEachIndexed { index, title ->
                Surface(Modifier.fillMaxWidth().clickable { selected = index }, color = LinLanSurface, shape = RoundedCornerShape(18.dp)) {
                    Row(Modifier.padding(18.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("0${index + 1}", color = LinLanAccent, fontSize = 22.sp, fontWeight = FontWeight.Bold)
                        Column(Modifier.weight(1f).padding(horizontal = 14.dp)) {
                            Text(title, fontWeight = FontWeight.SemiBold, fontSize = 16.sp)
                            Text("官方公告 · 点击阅读全文", fontSize = 12.sp, color = LinLanInkSecondary)
                        }
                        Icon(Icons.Outlined.ChevronRight, null, tint = LinLanInkSecondary)
                    }
                }
            }
        }
    } else {
        val text = remember(article) { context.assets.open("fanzha${article + 1}.txt").bufferedReader().use { it.readText() } }
        SafetyPage("公告详情", { selected = null }) {
            Text(antiFraudTitles[article], fontSize = 24.sp, fontWeight = FontWeight.Bold)
            Text(text, lineHeight = 27.sp, fontSize = 15.sp)
        }
    }
}
