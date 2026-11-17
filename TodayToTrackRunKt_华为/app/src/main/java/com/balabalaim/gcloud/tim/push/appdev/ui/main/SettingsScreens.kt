package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.outlined.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanSurface
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanOnAccent
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanInkSecondary
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanInk
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanHairline
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanDanger
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanCanvas
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanAccent
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.material.icons.outlined.Shield
import androidx.compose.material.icons.outlined.PrivacyTip
import androidx.compose.material.icons.outlined.Description
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.data.model.ChatProfile
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintAccent
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintCard
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintHeaderBottom
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintHeaderTop
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintPageBg
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintSecondary
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintTitle
import kotlinx.coroutines.launch

private val Seafoam = MintAccent
private val Clay = Color(0xFFC45C26)

@Composable
fun LinLanMineHubScreen(
    onOpenRealname: () -> Unit,
    onOpenDevices: () -> Unit,
    onOpenDeviceProtection: () -> Unit,
    onOpenHelp: () -> Unit,

    onOpenProfile: () -> Unit,
    onOpenSecurity: () -> Unit,
    onOpenNotify: () -> Unit,
    onOpenTeen: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
    onLogout: () -> Unit,
) {
    var showFeaturePermissions by remember { mutableStateOf(false) }
    if (showFeaturePermissions) com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermissionDialog { showFeaturePermissions = false }
    var profile by remember { mutableStateOf<ChatProfile?>(null) }
    val ownerProfile by com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.currentProfile.collectAsState()
    LaunchedEffect(ownerProfile) {
        val update=ownerProfile
        if(update!=null && update.generation==LinLanAppContainer.sessionStore().sessionGeneration() && update.owner==com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) profile=update.profile
    }

    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }

    LaunchedEffect(Unit) {
        runCatching { api.fetchProfile() }.onSuccess { profile = it }
    }

    val nickname = profile?.nickname.orEmpty().ifBlank { "未设置昵称" }
    val phone = LinLanAppContainer.appPrefs().accountPhone.ifBlank { "未绑定手机号" }
    val signal = profile?.chuanluId?.takeIf { it.isNotBlank() } ?: "未设置"

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(LinLanCanvas)
            .windowInsetsPadding(WindowInsets.statusBars)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 16.dp),
    ) {
        Text(
            "我的睿之禾",
            color = LinLanInk,
            fontWeight = FontWeight.Bold,
            fontSize = 28.sp,
            modifier = Modifier.padding(bottom = 16.dp),
        )

        // 睿之禾个人名片
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(Brush.linearGradient(listOf(Color(0xFF26364E), Color(0xFF40536D))))
                .clickable(onClick = onOpenProfile)
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(64.dp)
                    .clip(CircleShape)
                    .background(LinLanAccent),
                contentAlignment = Alignment.Center,
            ) {
                val url = profile?.avatarUrl
                if (!url.isNullOrBlank()) {
                    AsyncImage(url, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
                } else {
                    Text(
                        nickname.firstOrNull()?.toString() ?: "?",
                        color = LinLanOnAccent,
                        fontWeight = FontWeight.Bold,
                        fontSize = 26.sp,
                    )
                }
            }
            Column(modifier = Modifier.padding(start = 14.dp).weight(1f)) {
                Text(nickname, color = Color.White, fontWeight = FontWeight.Bold, fontSize = 22.sp)
                Text(
                    phone,
                    color = Color(0xFFCBD5E1),
                    fontSize = 13.sp,
                    modifier = Modifier.padding(top = 4.dp),
                )
                Text(
                    "睿之禾号 $signal",
                    color = Color(0xFFCBD5E1),
                    fontSize = 13.sp,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
            Text(
                "编辑资料",
                color = Color(0xFFFFBE91),
                fontWeight = FontWeight.SemiBold,
                fontSize = 14.sp,
            )
        }

        Spacer(Modifier.height(20.dp))
        LinLanSectionLabel("账号与设置")
        LinLanSettingsCard {
            LinLanSettingsRow(
                title = "个人资料",
                subtitle = "昵称、头像、手机号、睿之禾号",
                icon = Icons.Default.Person,
                onClick = onOpenProfile,
            )
            LinLanSettingsRow(
                title = "消息通知",
                subtitle = "提醒与提示音",
                icon = Icons.Default.Notifications,
                onClick = onOpenNotify,
            )
            LinLanSettingsRow(
                title = "功能权限",
                subtitle = "查看选择，重新设置已拒绝的功能",
                icon = Icons.Default.Security,
                onClick = { showFeaturePermissions = true },
            )
            LinLanSettingsRow(
                title = "隐私与安全",
                subtitle = "加好友验证、改密、注销",
                icon = Icons.Default.Security,
                onClick = onOpenSecurity,
                showDivider = false,
            )
        }

        Spacer(Modifier.height(16.dp))
        LinLanSectionLabel("安心使用")
        LinLanSettingsCard {
            LinLanSettingsRow("实名认证", "核验身份，安心交流", Icons.Outlined.VerifiedUser, onOpenRealname)
            LinLanSettingsRow("登录设备管理", "查看设备、验证新登录", Icons.Outlined.Devices, onOpenDevices)
            LinLanSettingsRow("设备保护", "登录验证、提醒与设备锁", Icons.Outlined.Lock, onOpenDeviceProtection)
            LinLanSettingsRow("青少年模式", icon = Icons.Outlined.Shield, onClick = onOpenTeen)
            LinLanSettingsRow("帮助与反馈", "常见问题与意见反馈", Icons.Outlined.HelpOutline, onOpenHelp, showDivider = false)
        }

        message?.let {
            Text(it, color = LinLanDanger, modifier = Modifier.padding(top = 12.dp))
        }

        Spacer(Modifier.height(20.dp))
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(LinLanSurface)
                .clickable {
                    scope.launch {
                        LinLanAppContainer.authRepository().logout()
                        onLogout()
                    }
                }
                .padding(vertical = 16.dp),
            contentAlignment = Alignment.Center,
        ) {
            Text("退出账号", color = LinLanDanger, fontWeight = FontWeight.SemiBold, fontSize = 16.sp)
        }
        Spacer(Modifier.height(28.dp))
    }
}

@Composable
private fun LinLanSectionLabel(text: String) {
    Text(
        text,
        color = LinLanInkSecondary,
        fontSize = 13.sp,
        fontWeight = FontWeight.Medium,
        modifier = Modifier.padding(start = 4.dp, bottom = 8.dp),
    )
}

@Composable
private fun LinLanSettingsCard(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(LinLanSurface),
    ) { content() }
}

@Composable
private fun LinLanSettingsRow(
    title: String,
    subtitle: String? = null,
    icon: ImageVector,
    onClick: () -> Unit,
    showDivider: Boolean = true,
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .padding(horizontal = 14.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(LinLanAccent.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(icon, contentDescription = null, tint = LinLanAccent, modifier = Modifier.size(20.dp))
            }
            Column(modifier = Modifier.padding(start = 12.dp).weight(1f)) {
                Text(title, color = LinLanInk, fontSize = 16.sp, fontWeight = FontWeight.Medium)
                if (!subtitle.isNullOrBlank()) {
                    Text(
                        subtitle,
                        color = LinLanInkSecondary,
                        fontSize = 12.sp,
                        modifier = Modifier.padding(top = 2.dp),
                    )
                }
            }
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = LinLanInkSecondary,
            )
        }
        if (showDivider) {
            HorizontalDivider(
                color = LinLanHairline,
                thickness = 0.5.dp,
                modifier = Modifier.padding(start = 62.dp),
            )
        }
    }
}

// 旧 SettingsCard / SettingsRow 保留给子页复用（视觉对齐 LinLan）
@Composable
private fun SettingsCard(content: @Composable () -> Unit) {
    LinLanSettingsCard(content)
}

@Composable
private fun SettingsRow(title: String, onClick: () -> Unit) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .padding(horizontal = 16.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(title, color = LinLanInk, modifier = Modifier.weight(1f), fontSize = 16.sp)
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = LinLanInkSecondary,
            )
        }
        HorizontalDivider(color = LinLanHairline, thickness = 0.5.dp)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanEditProfileScreen(onBack: () -> Unit) {
    var nickname by remember { mutableStateOf("") }
    var avatarUrl by remember { mutableStateOf("") }
    var imUserId by remember { mutableStateOf("") }
    /** Server wanan_id / chuanlu_id; blank means unset. */
    var signalId by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var uploading by remember { mutableStateOf(false) }
    var pendingAvatar by remember { mutableStateOf<Pair<String, String>?>(null) }
    var showSignalDialog by remember { mutableStateOf(false) }
    var signalDraft by remember { mutableStateOf("") }
    var signalSaving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }
    val pickAvatar = androidx.activity.compose.rememberLauncherForActivityResult(
        com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVisualMediaPicker(),
    ) { uris ->
        val uri = uris.firstOrNull()
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            uploading = true
            runCatching {
                val owner = com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser.orEmpty()
                val file = com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanAvatarImages.prepare(context, uri)
                try {
                    val uid = imUserId.ifBlank { api.fetchProfile().imUserId }.ifBlank { error("用户未登录") }
                    check(owner == uid && owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) { "账号已切换" }
                    val url = api.uploadUserAvatar(uid, file.path)
                    pendingAvatar = uid to url
                    check(owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) { "账号已切换" }
                    api.updateProfile(avatarUrl = url)
                    LinLanImManager.syncSelfProfile(null, url)
                    check(owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) { "账号已切换" }
                    avatarUrl = url
                    pendingAvatar = null
                } finally { file.delete() }
            }.onSuccess {
                message = "头像已更新"
                Toast.makeText(context, "头像已更新", Toast.LENGTH_SHORT).show()
            }.onFailure {
                message = it.message
                Toast.makeText(context, it.message ?: "上传失败", Toast.LENGTH_SHORT).show()
            }
            uploading = false
        }
    }
    val requestPickAvatar = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberAlbumImageAction {
        com.balabalaim.gcloud.tim.push.appdev.ui.common.launchVisualMediaPicker(context, pickAvatar)
    }

    fun reloadProfile() {
        scope.launch {
            runCatching { api.fetchProfile() }.onSuccess {
                nickname = it.nickname
                avatarUrl = it.avatarUrl
                imUserId = it.imUserId
                signalId = it.chuanluId
            }
        }
    }

    fun isValidSignalId(raw: String): Boolean =
        Regex("^[a-z][a-z0-9_-]{5,19}$").matches(raw)

    LaunchedEffect(Unit) { reloadProfile() }

    if (showSignalDialog) {
        AlertDialog(
            onDismissRequest = { if (!signalSaving) showSignalDialog = false },
            title = {
                Text(if (signalId.isBlank()) "设置睿之禾号" else "修改睿之禾号")
            },
            text = {
                Column {
                    OutlinedTextField(
                        value = signalDraft,
                        onValueChange = { raw ->
                            signalDraft = raw.lowercase()
                                .filter { it in 'a'..'z' || it in '0'..'9' || it == '_' || it == '-' }
                                .take(20)
                        },
                        singleLine = true,
                        label = { Text("输入睿之禾号") },
                        placeholder = { Text("例如：ruizhihe2026") },
                        shape = RoundedCornerShape(12.dp),
                        colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                            unfocusedBorderColor = MintAccent.copy(alpha = 0.65f),
                            focusedBorderColor = MintAccent,
                            focusedContainerColor = Color.White,
                            unfocusedContainerColor = Color.White,
                        ),
                        supportingText = {
                            Text("6-20 位，以字母开头，可包含数字、下划线或短横线")
                        },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    enabled = !signalSaving,
                    onClick = {
                        val value = signalDraft.trim().lowercase()
                        if (value.isEmpty()) {
                            message = "请输入睿之禾号"
                            return@TextButton
                        }
                        if (!isValidSignalId(value)) {
                            message = "睿之禾号格式不正确"
                            Toast.makeText(context, "格式：6-20位，字母开头", Toast.LENGTH_SHORT).show()
                            return@TextButton
                        }
                        if (value == signalId) {
                            showSignalDialog = false
                            return@TextButton
                        }
                        scope.launch {
                            signalSaving = true
                            runCatching { api.updateProfile(wananId = value) }
                                .onSuccess {
                                    signalId = value
                                    showSignalDialog = false
                                    message = "睿之禾号已更新"
                                    Toast.makeText(context, "睿之禾号已更新", Toast.LENGTH_SHORT).show()
                                }
                                .onFailure {
                                    message = it.message
                                    Toast.makeText(context, it.message ?: "更新失败", Toast.LENGTH_SHORT).show()
                                }
                            signalSaving = false
                        }
                    },
                ) { Text(if (signalSaving) "保存中…" else "确定") }
            },
            dismissButton = {
                TextButton(
                    enabled = !signalSaving,
                    onClick = { showSignalDialog = false },
                ) { Text("取消") }
            },
        )
    }

    Scaffold(
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        containerColor = MintPageBg,
        topBar = {
            TopAppBar(
                title = { Text("编辑资料") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
                actions = {
                    TextButton(onClick = {
                        scope.launch {
                            runCatching {
                                api.updateProfile(nickname = nickname.trim())
                                LinLanImManager.syncSelfProfile(nickname.trim(), null)
                            }.onSuccess {
                                message = "已保存"
                                Toast.makeText(context, "已保存", Toast.LENGTH_SHORT).show()
                            }.onFailure {
                                message = it.message
                                Toast.makeText(context, it.message ?: "保存失败", Toast.LENGTH_SHORT).show()
                            }
                        }
                    }) { Text("保存", color = MintAccent) }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(16.dp))
                    .background(MintCard)
                    .clickable(enabled = !uploading) { requestPickAvatar() }
                    .padding(20.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Box(
                    modifier = Modifier
                        .size(88.dp)
                        .clip(CircleShape)
                        .background(Seafoam),
                    contentAlignment = Alignment.Center,
                ) {
                    if (avatarUrl.isNotBlank()) {
                        AsyncImage(
                            avatarUrl,
                            contentDescription = "头像",
                            modifier = Modifier.fillMaxSize(),
                            contentScale = ContentScale.Crop,
                        )
                    } else {
                        Text(
                            nickname.firstOrNull()?.toString() ?: "我",
                            color = Color.White,
                            fontSize = 32.sp,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
                Text(
                    if (uploading) "上传中…" else "点击更换头像",
                    color = Seafoam,
                    modifier = Modifier.padding(top = 8.dp),
                )
            }
            if (pendingAvatar != null) {
                TextButton(enabled = !uploading, onClick = {
                    val pending = pendingAvatar ?: return@TextButton
                    scope.launch {
                        uploading = true
                        try {
                            check(com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser == pending.first) { "账号已切换" }
                            api.updateProfile(avatarUrl = pending.second)
                            LinLanImManager.syncSelfProfile(null, pending.second)
                            check(com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser == pending.first) { "账号已切换" }
                            avatarUrl = pending.second
                            pendingAvatar = null
                            message = "头像资料已同步"
                        } catch (error: Exception) { message = error.message ?: "资料同步失败，请重试" }
                        finally { uploading = false }
                    }
                }) { Text("重试头像资料同步（无需重传）") }
            }
            Spacer(Modifier.height(12.dp))
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(16.dp))
                    .background(MintCard)
                    .padding(16.dp),
            ) {
                OutlinedTextField(
                    value = nickname,
                    onValueChange = { nickname = it },
                    label = { Text("昵称") },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(14.dp))
                Text("手机号", color = MintSecondary, fontSize = 13.sp)
                Text(
                    LinLanAppContainer.appPrefs().accountPhone.ifBlank { "未绑定" },
                    color = MintTitle,
                    fontSize = 17.sp,
                    fontWeight = FontWeight.Medium,
                    modifier = Modifier.padding(top = 4.dp),
                )
                Spacer(Modifier.height(14.dp))
                Text("睿之禾号", color = MintTitle, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                Spacer(Modifier.height(8.dp))
                Row(
                    modifier = Modifier.fillMaxWidth().border(1.dp, MintAccent.copy(alpha = 0.55f), RoundedCornerShape(12.dp))
                        .background(MintPageBg, RoundedCornerShape(12.dp))
                        .clickable { signalDraft = signalId; showSignalDialog = true }
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(signalId.ifBlank { "尚未设置睿之禾号" }, color = if (signalId.isBlank()) MintSecondary else MintTitle,
                            fontSize = 16.sp, fontWeight = FontWeight.Medium)
                        Text("点击右侧按钮填写", color = MintSecondary, fontSize = 11.sp)
                    }
                    Button(onClick = { signalDraft = signalId; showSignalDialog = true },
                        shape = RoundedCornerShape(10.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 16.dp, vertical = 8.dp)) {
                        Text(if (signalId.isBlank()) "设置" else "修改")
                    }
                    if (signalId.isNotBlank()) IconButton(onClick = {
                        val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                        cm.setPrimaryClip(ClipData.newPlainText("ruizhihe_id", signalId))
                        Toast.makeText(context, "已复制睿之禾号", Toast.LENGTH_SHORT).show()
                    }) { Icon(Icons.Default.ContentCopy, "复制", tint = MintAccent) }
                }
                Spacer(Modifier.height(8.dp))
                Text("6-20 位，字母开头；添加好友时可搜索", color = MintSecondary, fontSize = 12.sp)
            }
            message?.let { Text(it, modifier = Modifier.padding(top = 8.dp), color = Seafoam) }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanSecurityPrivacyScreen(
    onBack: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
    onOpenChangePassword: () -> Unit,
    onLogout: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }
    var showDelete by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    var needFriendVerify by remember { mutableStateOf(true) }
    var updatingVerify by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        runCatching { api.fetchProfile() }
            .onSuccess { needFriendVerify = it.allowType != 0 }
    }

    if (showDelete) {
        AlertDialog(
            onDismissRequest = { showDelete = false },
            title = { Text("注销账号" , color = LinLanDanger) },
            text = { Text("注销后账号数据将无法恢复，确定继续？") },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            api.deleteAccount()
                            LinLanAppContainer.authRepository().logout()
                        }.onSuccess {
                            showDelete = false
                            Toast.makeText(context, "账号已注销", Toast.LENGTH_SHORT).show()
                            onLogout()
                        }.onFailure {
                            message = it.message
                            showDelete = false
                        }
                    }
                }) { Text("注销") }
            },
            dismissButton = { TextButton(onClick = { showDelete = false }) { Text("取消") } },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("隐私与安全") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("加我为好友需要验证")
                Switch(
                    checked = needFriendVerify,
                    enabled = !updatingVerify,
                    onCheckedChange = { checked ->
                        if (updatingVerify) return@Switch
                        val previous = needFriendVerify
                        needFriendVerify = checked
                        updatingVerify = true
                        scope.launch {
                            val allowType = if (checked) 1 else 0
                            runCatching {
                                api.updateProfile(allowType = allowType)
                                LinLanImManager.syncSelfProfile(
                                    nickname = null,
                                    faceUrl = null,
                                    allowType = allowType,
                                )
                            }.onSuccess {
                                Toast.makeText(
                                    context,
                                    if (checked) "已开启好友验证" else "已关闭好友验证",
                                    Toast.LENGTH_SHORT,
                                ).show()
                            }.onFailure {
                                needFriendVerify = previous
                                message = it.message
                            }
                            updatingVerify = false
                        }
                    },
                )
            }
            ListButton("修改密码") { onOpenChangePassword() }
            ListButton("睿之禾用户服务协议") { onOpenLegal("睿之禾用户服务协议", LinLanAppConstants.USER_AGREEMENT_URL) }
            ListButton("睿之禾隐私政策") { onOpenLegal("睿之禾隐私政策", LinLanAppConstants.PRIVACY_POLICY_URL) }
            ListButton("睿之禾个人信息收集清单") {
                onOpenLegal("睿之禾个人信息收集清单", LinLanAppConstants.PERSONAL_INFO_PROTOCOL_URL)
            }
            ListButton("睿之禾免责声明") {
                onOpenLegal("睿之禾免责声明", LinLanAppConstants.STATEMENT_PROTOCOL_URL)
            }
            ListButton("未成年人保护") {
                onOpenLegal("未成年人保护", LinLanAppConstants.MINOR_PROTECTION_URL)
            }
            ListButton("睿之禾个人信息共享清单") {
                onOpenLegal("睿之禾个人信息共享清单", LinLanAppConstants.THIRD_INFO_SHARE_URL)
            }
            ListButton("帮助与客服") { onOpenLegal("帮助与客服", LinLanAppConstants.SUPPORT_URL) }
            Text(
                "我们仅在提供聊天、通话和运动记录服务所必需时处理你的信息，详情见隐私政策。",
                style = MaterialTheme.typography.bodySmall,
                color = Color.Gray,
                modifier = Modifier.padding(top = 12.dp),
            )
            message?.let {
                Text(it, color = Color(0xFFD32F2F), modifier = Modifier.padding(top = 8.dp))
            }
            Spacer(modifier = Modifier.height(24.dp))
            TextButton(
                onClick = { showDelete = true },
                modifier = Modifier.fillMaxWidth(),
            ) { Text("注销账号", color = LinLanDanger) }
            Text(
                LinLanAppConstants.ICP_FILING_NUMBER,
                style = MaterialTheme.typography.bodySmall.copy(
                    textDecoration = androidx.compose.ui.text.style.TextDecoration.Underline,
                ),
                color = Color(0xFF1677FF),
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable {
                        val intent = Intent(
                            Intent.ACTION_VIEW,
                            Uri.parse(LinLanAppConstants.ICP_FILING_URL),
                        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                        runCatching { context.startActivity(intent) }
                            .onFailure {
                                Toast.makeText(context, "无法打开备案查询网站", Toast.LENGTH_SHORT).show()
                            }
                    }
                    .padding(top = 16.dp, bottom = 8.dp),
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanChangePasswordScreen(onBack: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }
    var current by remember { mutableStateOf("") }
    var next by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var submitting by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("修改密码") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            OutlinedTextField(
                value = current,
                onValueChange = { current = it },
                label = { Text("当前密码") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
            )
            OutlinedTextField(
                value = next,
                onValueChange = { next = it },
                label = { Text("新密码（至少 8 位）") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
            )
            OutlinedTextField(
                value = confirm,
                onValueChange = { confirm = it },
                label = { Text("确认新密码") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
            )
            message?.let { Text(it, color = Color(0xFFD32F2F)) }
            Button(
                onClick = {
                    when {
                        current.isBlank() -> message = "请输入当前密码"
                        next.length < 8 -> message = "新密码至少 8 位"
                        next != confirm -> message = "两次密码不一致"
                        else -> {
                            submitting = true
                            message = null
                            scope.launch {
                                runCatching { api.updatePassword(current, next) }
                                    .onSuccess {
                                        Toast.makeText(context, "密码已修改", Toast.LENGTH_SHORT).show()
                                        onBack()
                                    }
                                    .onFailure { message = it.message ?: "修改失败" }
                                submitting = false
                            }
                        }
                    }
                },
                enabled = !submitting,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.buttonColors(containerColor = MintAccent),
            ) { Text(if (submitting) "提交中..." else "确认修改") }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanMessageNotifyScreen(onBack: () -> Unit) {
    val prefs = remember { LinLanAppContainer.appPrefs() }
    var notify by remember { mutableStateOf(prefs.messageNotifyEnabled) }
    var sound by remember { mutableStateOf(prefs.soundNotifyEnabled) }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("消息通知") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        Column(modifier = Modifier.padding(padding).padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("接收新消息通知")
                Switch(checked = notify, onCheckedChange = {
                    notify = it
                    prefs.messageNotifyEnabled = it
                })
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("通知提示音")
                Switch(checked = sound, onCheckedChange = {
                    sound = it
                    prefs.soundNotifyEnabled = it
                })
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanTeenModeScreen(onBack: () -> Unit) {
    val prefs = remember { LinLanAppContainer.appPrefs() }
    var enabled by remember { mutableStateOf(prefs.teenModeEnabled) }
    var pin by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("青少年模式") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        Column(modifier = Modifier.padding(padding).padding(16.dp)) {
            Text("开启后，将限制部分社交能力，关闭需验证 4 位密码。")
            Spacer(Modifier.height(12.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("青少年模式")
                Switch(
                    checked = enabled,
                    onCheckedChange = { turnOn ->
                        if (turnOn) {
                            if (pin.length != 4 || !pin.all { it.isDigit() }) {
                                message = "请先设置 4 位数字密码"
                            } else {
                                prefs.teenModePin = pin
                                prefs.teenModeEnabled = true
                                enabled = true
                                message = "青少年模式已开启"
                            }
                        } else {
                            if (pin != prefs.teenModePin) {
                                message = "密码错误"
                            } else {
                                prefs.teenModeEnabled = false
                                enabled = false
                                message = "青少年模式已关闭"
                            }
                        }
                    },
                )
            }
            OutlinedTextField(
                value = pin,
                onValueChange = { if (it.length <= 4 && it.all { c -> c.isDigit() }) pin = it },
                label = { Text(if (enabled) "输入密码以关闭" else "设置 4 位密码") },
                modifier = Modifier.fillMaxWidth(),
            )
            message?.let { Text(it, modifier = Modifier.padding(top = 8.dp)) }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanLegalWebScreen(title: String, url: String, onBack: () -> Unit) {
    val context = LocalContext.current
    var loading by remember(url) { mutableStateOf(true) }
    Scaffold(
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            TopAppBar(
                title = { Text(title, maxLines = 1) },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
                actions = {
                    TextButton(onClick = {
                        runCatching {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                        }
                    }) { Text("浏览器") }
                },
            )
        },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            key(url) {
                androidx.compose.ui.viewinterop.AndroidView(
                    factory = { ctx ->
                        android.webkit.WebView(ctx).apply {
                            settings.javaScriptEnabled = true
                            settings.domStorageEnabled = true
                            settings.loadsImagesAutomatically = true
                            webViewClient = object : android.webkit.WebViewClient() {
                                override fun onPageFinished(view: android.webkit.WebView?, finishedUrl: String?) {
                                    loading = false
                                }
                            }
                            loadUrl(url)
                        }
                    },
                    modifier = Modifier.fillMaxSize(),
                )
            }
            if (loading) {
                androidx.compose.material3.CircularProgressIndicator(
                    modifier = Modifier.align(Alignment.Center),
                    color = Seafoam,
                )
            }
        }
    }
}

@Composable
private fun ListButton(title: String, onClick: () -> Unit) {
    val icon = when {
        title.contains("密码") -> Icons.Outlined.Lock
        title.contains("隐私") -> Icons.Outlined.PrivacyTip
        title.contains("收集") -> Icons.Outlined.FactCheck
        title.contains("共享") -> Icons.Outlined.Share
        title.contains("未成年") -> Icons.Outlined.Shield
        title.contains("帮助") -> Icons.Outlined.HelpOutline
        title.contains("声明") -> Icons.Outlined.Info
        else -> Icons.Outlined.Description
    }
    LinLanSettingsRow(title = title, icon = icon, onClick = onClick)
}
