package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanUriCache
import com.tencent.imsdk.v2.V2TIMManager
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanGroupPolicy
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.tencent.imsdk.v2.V2TIMFriendInfo
import com.tencent.imsdk.v2.V2TIMGroupInfo
import com.tencent.imsdk.v2.V2TIMGroupMemberFullInfo
import kotlinx.coroutines.launch
import java.net.URLDecoder
import java.nio.charset.StandardCharsets

private val PageBg = Color(0xFFF2F2F7)
private val LinkBlue = Color(0xFF007AFF)
private val DangerRed = Color(0xFFFF3B30)
private val AvatarPurple = Color(0xFF7B61FF)
private val SecondaryGray = Color(0xFF8E8E93)
private val DividerColor = Color(0xFFE5E5EA)
private val QqBlue = Color(0xFF12B7F5)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanGroupNoticeScreen(
    groupId: String,
    onBack: () -> Unit,
) {
    val decodedGroupId = remember(groupId) {
        URLDecoder.decode(groupId, StandardCharsets.UTF_8.name())
    }
    var text by remember { mutableStateOf("") }
    var editing by remember { mutableStateOf(false) }
    var canEdit by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(decodedGroupId) {
        text = LinLanImManager.fetchGroupNoticeContent(decodedGroupId)
        val role = runCatching { LinLanImManager.fetchSelfGroupMemberRole(decodedGroupId) }.getOrDefault(0)
        canEdit = LinLanImManager.canManageGroup(role)
    }

    val bytes = LinLanImManager.utf8ByteCount(if (editing) text else text.trim())
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(PageBg)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        GroupNavBar(
            title = "群公告",
            backLabel = "群聊信息",
            onBack = onBack,
            trailing = if (canEdit) {
                {
                    TextButton(onClick = {
                        if (editing) {
                            scope.launch {
                                runCatching {
                                    text = LinLanImManager.updateGroupNotice(decodedGroupId, text)
                                    editing = false
                                    message = "保存成功"
                                }.onFailure { message = it.message }
                            }
                        } else {
                            editing = true
                            message = null
                        }
                    }) {
                        Text(if (editing) "保存" else "编辑", color = LinkBlue)
                    }
                }
            } else null,
        )
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(Color.White)
                .padding(16.dp)
                .weight(1f),
        ) {
            if (editing) {
                OutlinedTextField(
                    value = text,
                    onValueChange = { text = it },
                    modifier = Modifier.fillMaxSize(),
                    maxLines = Int.MAX_VALUE,
                )
            } else {
                Text(
                    text.ifBlank { "暂无群公告" },
                    modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
                    color = if (text.isBlank()) SecondaryGray else Color.Black,
                    fontSize = 15.sp,
                )
            }
        }
        if (editing) {
            Text(
                "$bytes/${LinLanImManager.GROUP_NOTICE_MAX_BYTES} 字节",
                color = if (bytes > LinLanImManager.GROUP_NOTICE_MAX_BYTES) DangerRed else SecondaryGray,
                modifier = Modifier.padding(horizontal = 16.dp),
                fontSize = 12.sp,
            )
        }
        message?.let {
            Text(it, color = LinkBlue, modifier = Modifier.padding(16.dp), fontSize = 13.sp)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanGroupManageScreen(
    groupId: String,
    onBack: () -> Unit,
    onOpenMemberDetail: ((userId: String, title: String) -> Unit)? = null,
) {
    val decodedGroupId = remember(groupId) {
        URLDecoder.decode(groupId, StandardCharsets.UTF_8.name())
    }
    val members = remember { mutableStateListOf<V2TIMGroupMemberFullInfo>() }
    var selfRole by remember { mutableIntStateOf(0) }
    var allMuted by remember { mutableStateOf(false) }
    var muteAdminAlso by remember { mutableStateOf(false) }
    var profileAccess by remember { mutableStateOf(true) }
    var voiceCall by remember { mutableStateOf(false) }
    var videoCall by remember { mutableStateOf(false) }
    var showAddMute by remember { mutableStateOf(false) }
    var showSetAdmin by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val muteSeconds = 60 * 60 * 24 * 30

    fun reload() {
        scope.launch {
            val role = runCatching { LinLanImManager.fetchSelfGroupMemberRole(decodedGroupId) }.getOrDefault(0)
            selfRole = role
            if (!LinLanImManager.canManageGroup(role)) {
                Toast.makeText(context, "仅群主或管理员可访问", Toast.LENGTH_SHORT).show()
                onBack()
                return@launch
            }
            runCatching { LinLanImManager.fetchGroupMembers(decodedGroupId) }
                .onSuccess { members.clear(); members.addAll(it) }
            runCatching { LinLanImManager.fetchGroupInfo(decodedGroupId) }
                .onSuccess { allMuted = it?.isAllMuted == true }
            runCatching { LinLanGroupPolicy.fetchAll(decodedGroupId) }
                .onSuccess { attrs ->
                    muteAdminAlso = LinLanGroupPolicy.isMuteAdminAlsoEnabled(attrs)
                    profileAccess = LinLanGroupPolicy.isMemberProfileAccessEnabled(attrs)
                    voiceCall = LinLanGroupPolicy.isVoiceCallEnabled(attrs)
                    videoCall = LinLanGroupPolicy.isVideoCallEnabled(attrs)
                }
        }
    }

    LaunchedEffect(decodedGroupId) { reload() }

    val mutedMembers = members.filter { LinLanImManager.isGroupMemberMuted(it) }
    val isOwner = selfRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER

    if (showAddMute) {
        AlertDialog(
            onDismissRequest = { showAddMute = false },
            title = { Text("添加禁言成员") },
            text = {
                LazyColumn {
                    items(
                        members.filter {
                            it.role != V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER &&
                                !LinLanImManager.isGroupMemberMuted(it)
                        },
                        key = { it.userID },
                    ) { member ->
                        val name = member.nameCard?.ifBlank { null }
                            ?: member.nickName?.ifBlank { null }
                            ?: member.userID
                        Text(
                            name,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    scope.launch {
                                        runCatching {
                                            LinLanImManager.muteGroupMember(decodedGroupId, member.userID, muteSeconds)
                                        }.onSuccess {
                                            message = "已禁言 $name"
                                            showAddMute = false
                                            reload()
                                        }.onFailure { message = it.message }
                                    }
                                }
                                .padding(vertical = 12.dp),
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showAddMute = false }) { Text("关闭") }
            },
        )
    }

    if (showSetAdmin) {
        AlertDialog(
            onDismissRequest = { showSetAdmin = false },
            title = { Text("设置管理员") },
            text = {
                LazyColumn {
                    items(
                        members.filter {
                            it.role != V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER
                        },
                        key = { it.userID },
                    ) { member ->
                        val name = member.nameCard?.ifBlank { null }
                            ?: member.nickName?.ifBlank { null }
                            ?: member.userID
                        val isAdmin = member.role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    scope.launch {
                                        val newRole = if (isAdmin) {
                                            V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_MEMBER
                                        } else {
                                            V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN
                                        }
                                        runCatching {
                                            LinLanImManager.setGroupMemberRole(decodedGroupId, member.userID, newRole)
                                        }.onSuccess {
                                            message = if (isAdmin) "已取消管理员" else "已设为管理员"
                                            reload()
                                        }.onFailure { message = it.message }
                                    }
                                }
                                .padding(vertical = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Text(name)
                            Text(
                                if (isAdmin) "取消管理员" else "设为管理员",
                                color = LinkBlue,
                                fontSize = 13.sp,
                            )
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showSetAdmin = false }) { Text("关闭") }
            },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.White)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        GroupNavBar(title = "群管理", backLabel = "群聊信息", onBack = onBack)
        HorizontalDivider(color = DividerColor, thickness = 0.5.dp)
        Column(modifier = Modifier.verticalScroll(rememberScrollState())) {
            ManageToggle("全员禁言", allMuted) { checked ->
                scope.launch {
                    runCatching { LinLanImManager.muteAllGroupMembers(decodedGroupId, checked) }
                        .onSuccess {
                            allMuted = checked
                            if (muteAdminAlso && checked) {
                                members.filter {
                                    it.role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN
                                }.forEach {
                                    runCatching {
                                        LinLanImManager.muteGroupMember(decodedGroupId, it.userID, muteSeconds)
                                    }
                                }
                            }
                            message = if (checked) "已开启全员禁言" else "已关闭全员禁言"
                            reload()
                        }
                        .onFailure { message = it.message }
                }
            }
            ManageToggle("管理员也禁言", muteAdminAlso) { checked ->
                scope.launch {
                    runCatching { LinLanGroupPolicy.updateMuteAdminAlso(decodedGroupId, checked) }
                        .onSuccess {
                            muteAdminAlso = checked
                            message = if (checked) "已开启管理员禁言" else "已关闭管理员禁言"
                        }
                        .onFailure { message = it.message }
                }
            }
            ManageToggle("允许群语音通话", voiceCall, enabled = isOwner) { checked ->
                scope.launch {
                    runCatching { LinLanGroupPolicy.updateVoiceCallAccess(decodedGroupId, checked) }
                        .onSuccess {
                            voiceCall = checked
                            message = if (checked) "已允许群语音通话" else "已关闭群语音通话"
                        }
                        .onFailure { message = it.message }
                }
            }
            ManageToggle("允许群视频通话", videoCall, enabled = isOwner) { checked ->
                scope.launch {
                    runCatching { LinLanGroupPolicy.updateVideoCallAccess(decodedGroupId, checked) }
                        .onSuccess {
                            videoCall = checked
                            message = if (checked) "已允许群视频通话" else "已关闭群视频通话"
                        }
                        .onFailure { message = it.message }
                }
            }
            ManageToggle("允许查看成员资料", profileAccess) { checked ->
                scope.launch {
                    runCatching { LinLanGroupPolicy.updateMemberProfileAccess(decodedGroupId, checked) }
                        .onSuccess {
                            profileAccess = checked
                            message = if (checked) "已允许查看成员资料" else "已禁止查看成员资料"
                        }
                        .onFailure { message = it.message }
                }
            }
            if (isOwner) {
                SettingLinkRow("设置管理员") { showSetAdmin = true }
            }
            Text(
                "禁言成员",
                color = SecondaryGray,
                fontSize = 13.sp,
                modifier = Modifier
                    .fillMaxWidth()
                    .background(PageBg)
                    .padding(horizontal = 16.dp, vertical = 8.dp),
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { showAddMute = true }
                    .padding(horizontal = 16.dp, vertical = 14.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(28.dp)
                        .clip(CircleShape)
                        .border(1.dp, LinkBlue, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Default.Add, null, tint = LinkBlue, modifier = Modifier.size(18.dp))
                }
                Text("添加禁言成员", color = LinkBlue, modifier = Modifier.padding(start = 12.dp))
            }
            mutedMembers.forEach { member ->
                val name = member.nameCard?.ifBlank { null }
                    ?: member.nickName?.ifBlank { null }
                    ?: member.userID
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            if (LinLanImManager.canViewMemberProfile(profileAccess, member.role)) {
                                onOpenMemberDetail?.invoke(member.userID, name)
                            } else {
                                Toast.makeText(context, "群已禁止查看成员资料", Toast.LENGTH_SHORT).show()
                            }
                        }
                        .padding(horizontal = 16.dp, vertical = 12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    MemberAvatar(url = member.faceUrl, name = name, size = 40.dp)
                    Text(name, modifier = Modifier.padding(start = 12.dp).weight(1f))
                    TextButton(onClick = {
                        scope.launch {
                            runCatching { LinLanImManager.muteGroupMember(decodedGroupId, member.userID, 0) }
                                .onSuccess {
                                    message = "已解除禁言"
                                    reload()
                                }
                                .onFailure { message = it.message }
                        }
                    }) { Text("解除", color = LinkBlue) }
                }
                HorizontalDivider(color = DividerColor, thickness = 0.5.dp, modifier = Modifier.padding(start = 68.dp))
            }
            message?.let {
                Text(it, color = LinkBlue, modifier = Modifier.padding(16.dp), fontSize = 13.sp)
            }
            Spacer(Modifier.height(40.dp))
        }
    }
}

@Composable
fun LinLanGroupMembersScreen(
    groupId: String,
    onBack: () -> Unit,
    onOpenMemberDetail: ((userId: String, title: String) -> Unit)? = null,
) {
    val decodedGroupId = remember(groupId) {
        URLDecoder.decode(groupId, StandardCharsets.UTF_8.name())
    }
    val members = remember { mutableStateListOf<V2TIMGroupMemberFullInfo>() }
    var search by remember { mutableStateOf("") }
    var selfRole by remember { mutableIntStateOf(0) }
    var profileAccess by remember { mutableStateOf(true) }
    var menuUserId by remember { mutableStateOf<String?>(null) }
    var confirmKick by remember { mutableStateOf<V2TIMGroupMemberFullInfo?>(null) }
    var nextMemberSeq by remember { mutableStateOf(0L) }
    var loadingMembers by remember { mutableStateOf(false) }
    var totalMemberCount by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val selfId = remember { V2TIMManager.getInstance().loginUser.orEmpty() }
    val canManage = LinLanImManager.canManageGroup(selfRole)
    val muteSeconds = 60 * 60 * 24 * 30

    fun loadMemberPage(reset: Boolean) {
        if (loadingMembers) return
        scope.launch {
            loadingMembers = true
            val cursor = if (reset) 0L else nextMemberSeq
            runCatching { LinLanImManager.fetchGroupMemberPage(decodedGroupId, cursor) }
                .onSuccess { page ->
                    if (reset) members.clear()
                    val known = members.map { it.userID }.toHashSet()
                    members.addAll(page.members.filter { it.userID !in known })
                    nextMemberSeq = page.nextSeq
                }
            loadingMembers = false
        }
    }

    fun reloadMembers() = loadMemberPage(reset = true)

    LaunchedEffect(decodedGroupId) {
        val first = runCatching { LinLanImManager.fetchGroupMemberPage(decodedGroupId, 0L) }.getOrNull()
        if (first != null) {
            members.clear()
            members.addAll(first.members)
            nextMemberSeq = first.nextSeq
        }
        totalMemberCount = runCatching { LinLanImManager.fetchGroupInfo(decodedGroupId)?.memberCount ?: 0 }
            .getOrDefault(0)
        selfRole = runCatching { LinLanImManager.fetchSelfGroupMemberRole(decodedGroupId) }.getOrDefault(0)
        profileAccess = runCatching { LinLanGroupPolicy.fetchMemberProfileAccessEnabled(decodedGroupId) }.getOrDefault(false)
    }

    val filtered = remember(members.toList(), search) {
        val q = search.trim()
        if (q.isEmpty()) members.toList()
        else members.filter {
            val name = it.nameCard?.ifBlank { null } ?: it.nickName ?: it.userID
            name.contains(q, ignoreCase = true) || it.userID.contains(q, ignoreCase = true)
        }
    }

    if (confirmKick != null) {
        val target = confirmKick!!
        val name = target.nameCard?.ifBlank { null }
            ?: target.nickName?.ifBlank { null }
            ?: target.userID
        AlertDialog(
            onDismissRequest = { confirmKick = null },
            title = { Text("移出群聊") },
            text = { Text("确定将 $name 移出本群？") },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            LinLanImManager.kickGroupMember(decodedGroupId, listOf(target.userID))
                        }.onSuccess {
                            Toast.makeText(context, "已移出", Toast.LENGTH_SHORT).show()
                            confirmKick = null
                            reloadMembers()
                        }.onFailure {
                            Toast.makeText(context, it.message ?: "移出失败", Toast.LENGTH_SHORT).show()
                            confirmKick = null
                        }
                    }
                }) { Text("移出", color = DangerRed) }
            },
            dismissButton = {
                TextButton(onClick = { confirmKick = null }) { Text("取消") }
            },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.White)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        GroupNavBar(
            title = if (totalMemberCount > 0) "群成员 ($totalMemberCount)" else "群成员 (${members.size})",
            backLabel = "群聊信息",
            onBack = onBack,
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp)
                .clip(RoundedCornerShape(10.dp))
                .border(1.dp, DividerColor, RoundedCornerShape(10.dp))
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Default.Search, null, tint = SecondaryGray, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            androidx.compose.foundation.text.BasicTextField(
                value = search,
                onValueChange = { search = it },
                singleLine = true,
                decorationBox = { inner ->
                    if (search.isEmpty()) Text("搜索群成员", color = SecondaryGray, fontSize = 14.sp)
                    inner()
                },
                modifier = Modifier.weight(1f),
            )
        }
        LazyColumn {
            items(filtered, key = { it.userID }) { member ->
                val name = member.nameCard?.ifBlank { null }
                    ?: member.nickName?.ifBlank { null }
                    ?: member.userID
                val showMenu = canManage &&
                    member.userID != selfId &&
                    member.role != V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER &&
                    (
                        selfRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER ||
                            member.role != V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN
                        )
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            if (LinLanImManager.canViewMemberProfile(profileAccess, member.role)) {
                                onOpenMemberDetail?.invoke(member.userID, name)
                            } else {
                                Toast.makeText(context, "群已禁止查看成员资料", Toast.LENGTH_SHORT).show()
                            }
                        }
                        .padding(horizontal = 16.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    MemberAvatar(url = member.faceUrl, name = name, size = 44.dp)
                    Text(
                        name,
                        modifier = Modifier
                            .weight(1f)
                            .padding(start = 12.dp),
                        fontSize = 16.sp,
                    )
                    if (showMenu) {
                        Box {
                            IconButton(onClick = { menuUserId = member.userID }) {
                                Icon(
                                    Icons.Default.MoreVert,
                                    contentDescription = "更多",
                                    tint = SecondaryGray,
                                )
                            }
                            DropdownMenu(
                                expanded = menuUserId == member.userID,
                                onDismissRequest = { menuUserId = null },
                            ) {
                                DropdownMenuItem(
                                    text = {
                                        Text(
                                            if (LinLanImManager.isGroupMemberMuted(member)) "取消禁言" else "禁言",
                                        )
                                    },
                                    onClick = {
                                        menuUserId = null
                                        scope.launch {
                                            val seconds = if (LinLanImManager.isGroupMemberMuted(member)) 0 else muteSeconds
                                            runCatching {
                                                LinLanImManager.muteGroupMember(
                                                    decodedGroupId,
                                                    member.userID,
                                                    seconds,
                                                )
                                            }.onSuccess {
                                                Toast.makeText(
                                                    context,
                                                    if (seconds == 0) "已取消禁言" else "已禁言",
                                                    Toast.LENGTH_SHORT,
                                                ).show()
                                                reloadMembers()
                                            }.onFailure {
                                                Toast.makeText(
                                                    context,
                                                    it.message ?: "操作失败",
                                                    Toast.LENGTH_SHORT,
                                                ).show()
                                            }
                                        }
                                    },
                                )
                                DropdownMenuItem(
                                    text = { Text("移出群聊", color = DangerRed) },
                                    onClick = {
                                        menuUserId = null
                                        confirmKick = member
                                    },
                                )
                            }
                        }
                    }
                }
                HorizontalDivider(color = DividerColor, thickness = 0.5.dp, modifier = Modifier.padding(start = 72.dp))
            }
            if (search.isBlank() && (nextMemberSeq != 0L || loadingMembers)) {
                item(key = "load_more_members") {
                    Box(
                        modifier = Modifier.fillMaxWidth().height(52.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        TextButton(
                            enabled = !loadingMembers,
                            onClick = { loadMemberPage(reset = false) },
                        ) {
                            Text(if (loadingMembers) "加载中…" else "加载更多成员", color = LinkBlue)
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanGroupDetailScreen(
    groupId: String,
    title: String,
    onBack: () -> Unit,
    onOpenChat: (String, String, Boolean) -> Unit,
    onOpenNotice: (String) -> Unit,
    onOpenManage: (String) -> Unit,
    onOpenMembers: (String) -> Unit,
    onOpenReport: (targetId: String, targetType: String, targetName: String) -> Unit,
    onShareGroupCard: (() -> Unit)? = null,
    onLeftGroup: (() -> Unit)? = null,
) {
    val decodedTitle = remember(title) { URLDecoder.decode(title, StandardCharsets.UTF_8.name()) }
    val decodedGroupId = remember(groupId) { URLDecoder.decode(groupId, StandardCharsets.UTF_8.name()) }
    val members = remember { mutableStateListOf<V2TIMGroupMemberFullInfo>() }
    val friends = remember { mutableStateListOf<LinLanImManager.SelectableContact>() }
    val selectedInvite = remember { mutableStateListOf<String>() }
    var groupName by remember { mutableStateOf(decodedTitle) }
    var faceUrl by remember { mutableStateOf<String?>(null) }
    var selfRole by remember { mutableIntStateOf(0) }
    var accessInfo by remember(decodedGroupId) { mutableStateOf<V2TIMGroupInfo?>(null) }
    var accessLoading by remember(decodedGroupId) { mutableStateOf(true) }
    var accessError by remember(decodedGroupId) { mutableStateOf<String?>(null) }
    var joinPending by remember(decodedGroupId) { mutableStateOf(false) }
    var joining by remember(decodedGroupId) { mutableStateOf(false) }
    var groupReadEpoch by remember(decodedGroupId) { mutableIntStateOf(0) }

    var selfNameCard by remember { mutableStateOf("") }
    var pinned by remember { mutableStateOf(false) }
    var muted by remember { mutableStateOf(false) }
    var approveOpt by remember { mutableIntStateOf(V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH) }
    var addOpt by remember { mutableIntStateOf(V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH) }
    var showInvite by remember { mutableStateOf(false) }
    var editName by remember { mutableStateOf(false) }
    var editNameCard by remember { mutableStateOf(false) }
    var draft by remember { mutableStateOf("") }
    var confirmQuit by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    var uploadingAvatar by remember { mutableStateOf(false) }
    var pendingGroupAvatar by remember(decodedGroupId) { mutableStateOf<Pair<String, String>?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }
    val convId = "group_$decodedGroupId"
    val pickGroupAvatar = androidx.activity.compose.rememberLauncherForActivityResult(
        com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVisualMediaPicker(),
    ) { uris ->
        val uri = uris.firstOrNull()
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            uploadingAvatar = true
            runCatching {
                val owner = com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser
                val image = com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanAvatarImages.prepare(context, uri)
                val url = try { api.uploadGroupAvatar(decodedGroupId, image.path) } finally { image.delete() }
                check(owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) { "账号已切换" }
                pendingGroupAvatar = owner.orEmpty() to url
                LinLanImManager.setGroupInfo(
                    V2TIMGroupInfo().apply {
                        groupID = decodedGroupId
                        setFaceUrl(url)
                    },
                )
                check(owner == com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser) { "账号已切换" }
                faceUrl = url
                pendingGroupAvatar = null
            }.onSuccess {
                Toast.makeText(context, "群头像已更新", Toast.LENGTH_SHORT).show()
            }.onFailure {
                message = it.message
                Toast.makeText(context, it.message ?: "上传失败", Toast.LENGTH_SHORT).show()
            }
            uploadingAvatar = false
        }
    }
    val pickGroupAvatarAction = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberAlbumImageAction {
        com.balabalaim.gcloud.tim.push.appdev.ui.common.launchVisualMediaPicker(context, pickGroupAvatar)
    }

    val isOwner = selfRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER
    val isAdmin = LinLanImManager.canManageGroup(selfRole)

    fun reload() {
        val epoch = ++groupReadEpoch
        val owner = V2TIMManager.getInstance().loginUser
        scope.launch {
            accessLoading = true
            accessError = null
            try {
                val info = LinLanImManager.fetchGroupInfo(decodedGroupId) ?: error("群资料未确认")
                if (epoch != groupReadEpoch || owner != V2TIMManager.getInstance().loginUser) return@launch
                accessInfo = info
                groupName = info.groupName?.ifBlank { decodedTitle } ?: decodedTitle
                faceUrl = info.faceUrl
                approveOpt = info.groupApproveOpt
                addOpt = info.groupAddOpt
                selfRole = info.role
                if (selfRole <= 0) return@launch
                joinPending = false
                val page = LinLanImManager.fetchGroupMemberPage(decodedGroupId, 0L)
                if (epoch != groupReadEpoch || owner != V2TIMManager.getInstance().loginUser) return@launch
                members.clear(); members.addAll(page.members)
                selfNameCard = members.firstOrNull { it.userID == owner }?.nameCard.orEmpty()
                pinned = runCatching { LinLanImManager.isConversationPinned(convId) }.getOrDefault(false)
                muted = runCatching { LinLanImManager.isGroupMuted(decodedGroupId) }.getOrDefault(false)
            } catch (error: Exception) { if (epoch == groupReadEpoch) accessError = error.message ?: "群资料加载失败" }
            finally { if (epoch == groupReadEpoch) accessLoading = false }
        }
    }

    LaunchedEffect(decodedGroupId) { reload() }

    if (accessLoading || accessInfo == null || selfRole <= 0) {
        GroupCardPreview(accessInfo, decodedTitle, decodedGroupId, accessLoading, accessError, joinPending, joining,
            onBack, onRetry = { reload() }, onJoin = {
                val owner = V2TIMManager.getInstance().loginUser
                scope.launch {
                    joining = true
                    try {
                        val outcome = LinLanImManager.joinSharedGroup(decodedGroupId)
                        if (owner != V2TIMManager.getInstance().loginUser) return@launch
                        if (outcome == LinLanImManager.GroupJoinOutcome.JOINED) {
                            reload()
                            onOpenChat(convId, groupName, true)
                        } else { joinPending = true; accessError = null }
                    } catch (error: Exception) { accessError = error.message ?: "申请入群失败" }
                    finally { joining = false }
                }
            })
        return
    }

    if (showInvite) {
        AlertDialog(
            onDismissRequest = { showInvite = false },
            title = { Text("邀请好友入群") },
            text = {
                LazyColumn(modifier = Modifier.height(280.dp)) {
                    if (friends.isEmpty()) {
                        item {
                            Text("暂无联系人可选", color = SecondaryGray)
                        }
                    }
                    items(friends, key = { it.userId }) { friend ->
                        val checked = selectedInvite.contains(friend.userId)
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    if (checked) selectedInvite.remove(friend.userId)
                                    else selectedInvite.add(friend.userId)
                                }
                                .padding(vertical = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            androidx.compose.material3.Checkbox(
                                checked = checked,
                                onCheckedChange = {
                                    if (it) selectedInvite.add(friend.userId) else selectedInvite.remove(friend.userId)
                                },
                            )
                            Text(friend.displayName)
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            LinLanImManager.inviteUsersToGroup(decodedGroupId, selectedInvite.toList())
                        }.onSuccess {
                            message = "已发送邀请"
                            showInvite = false
                            selectedInvite.clear()
                            reload()
                        }.onFailure { message = it.message }
                    }
                }) { Text("邀请") }
            },
            dismissButton = { TextButton(onClick = { showInvite = false }) { Text("取消") } },
        )
    }

    if (editName || editNameCard) {
        val titleText = if (editName) "修改群聊名称" else "我的群昵称"
        AlertDialog(
            onDismissRequest = {
                editName = false; editNameCard = false
            },
            title = { Text(titleText) },
            text = {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text("请输入") },
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            when {
                                editName -> {
                                    LinLanImManager.setGroupInfo(
                                        V2TIMGroupInfo().apply {
                                            groupID = decodedGroupId
                                            setGroupName(draft.trim())
                                        },
                                    )
                                    groupName = draft.trim()
                                }
                                else -> {
                                    LinLanImManager.setSelfGroupNameCard(decodedGroupId, draft.trim())
                                    selfNameCard = draft.trim()
                                }
                            }
                        }.onSuccess {
                            message = "已保存"
                            editName = false; editNameCard = false
                        }.onFailure { message = it.message }
                    }
                }) { Text("确定") }
            },
            dismissButton = {
                TextButton(onClick = {
                    editName = false; editNameCard = false
                }) { Text("取消") }
            },
        )
    }

    if (confirmQuit) {
        AlertDialog(
            onDismissRequest = { confirmQuit = false },
            title = { Text(if (isOwner) "确定解散该群？" else "确定退出该群？") },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            if (isOwner) LinLanImManager.dismissGroup(decodedGroupId)
                            else LinLanImManager.quitGroup(decodedGroupId)
                        }.onSuccess {
                            confirmQuit = false
                            Toast.makeText(context, "操作成功", Toast.LENGTH_SHORT).show()
                            onLeftGroup?.invoke()
                        }.onFailure { message = it.message }
                    }
                }) { Text("确定", color = DangerRed) }
            },
            dismissButton = {
                TextButton(onClick = { confirmQuit = false }) { Text("取消") }
            },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(PageBg)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        GroupNavBar(
            title = "群聊信息",
            backLabel = groupName.take(8),
            onBack = onBack,
            trailing = {
                TextButton(onClick = { onOpenReport(decodedGroupId, "group", groupName) }) {
                    Text("举报", color = LinkBlue)
                }
            },
        )

        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(bottom = 24.dp),
        ) {
            // Avatar card
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 10.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color.White)
                    .padding(vertical = 20.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Box(
                    modifier = Modifier
                        .size(72.dp)
                        .clip(RoundedCornerShape(14.dp))
                        .background(AvatarPurple)
                        .clickable(enabled = isAdmin && !uploadingAvatar) {
                            if (isAdmin) pickGroupAvatarAction()
                        },
                    contentAlignment = Alignment.Center,
                ) {
                    if (!faceUrl.isNullOrBlank()) {
                        AsyncImage(faceUrl, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
                    } else {
                        Text(
                            groupName.firstOrNull()?.toString() ?: "群",
                            color = Color.White,
                            fontSize = 28.sp,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
                if (pendingGroupAvatar != null) {
                    TextButton(enabled = !uploadingAvatar, onClick = {
                        val pending = pendingGroupAvatar ?: return@TextButton
                        scope.launch {
                            uploadingAvatar = true
                            try {
                                check(com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser == pending.first) { "账号已切换" }
                                LinLanImManager.setGroupInfo(V2TIMGroupInfo().apply { groupID = decodedGroupId; setFaceUrl(pending.second) })
                                check(com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser == pending.first) { "账号已切换" }
                                faceUrl = pending.second
                                pendingGroupAvatar = null
                                message = "群头像资料已同步"
                            } catch (error: Exception) { message = error.message ?: "资料同步失败" }
                            finally { uploadingAvatar = false }
                        }
                    }) { Text("重试群头像资料同步") }
                }
                Text(
                    "群头像",
                    color = SecondaryGray,
                    fontSize = 13.sp,
                    modifier = Modifier.padding(top = 8.dp),
                )
                if (isAdmin) {
                    Text(
                        if (uploadingAvatar) "上传中…" else "点击从相册选择",
                        color = LinkBlue,
                        fontSize = 11.sp,
                    )
                }
            }

            // Members
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color.White)
                    .padding(vertical = 12.dp),
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 14.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text("群成员 (${members.size})", fontWeight = FontWeight.Medium)
                    Text(
                        "查看全部",
                        color = LinkBlue,
                        modifier = Modifier.clickable { onOpenMembers(decodedGroupId) },
                    )
                }
                Spacer(Modifier.height(12.dp))
                Row(
                    modifier = Modifier
                        .horizontalScroll(rememberScrollState())
                        .padding(horizontal = 10.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    members.take(8).forEach { member ->
                        val name = member.nameCard?.ifBlank { null }
                            ?: member.nickName?.ifBlank { null }
                            ?: member.userID
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            MemberAvatar(url = member.faceUrl, name = name, size = 48.dp)
                            Text(
                                name,
                                fontSize = 11.sp,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                modifier = Modifier.width(52.dp),
                                textAlign = TextAlign.Center,
                            )
                        }
                    }
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        modifier = Modifier.clickable {
                            scope.launch {
                                val memberIds = members.map { it.userID }.toSet()
                                runCatching { LinLanImManager.fetchSelectableContacts() }
                                    .onSuccess { list ->
                                        friends.clear()
                                        friends.addAll(list.filter { it.userId !in memberIds })
                                        selectedInvite.clear()
                                        showInvite = true
                                    }
                                    .onFailure { message = it.message }
                            }
                        },
                    ) {
                        Box(
                            modifier = Modifier
                                .size(48.dp)
                                .clip(CircleShape)
                                .background(Color(0xFFF2F2F7)),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(Icons.Default.Add, null, tint = SecondaryGray)
                        }
                        Text("邀请", fontSize = 11.sp)
                    }
                }
            }

            Spacer(Modifier.height(10.dp))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                QuickAction("群公告") { onOpenNotice(decodedGroupId) }
                QuickAction("群文件") {
                    Toast.makeText(context, "群文件即将上线", Toast.LENGTH_SHORT).show()
                }
                QuickAction("群相册") {
                    Toast.makeText(context, "群相册即将上线", Toast.LENGTH_SHORT).show()
                }
                QuickAction("更多") {
                    if (isAdmin) onOpenManage(decodedGroupId)
                    else Toast.makeText(context, "更多功能开发中", Toast.LENGTH_SHORT).show()
                }
            }

            Spacer(Modifier.height(10.dp))
            SettingsCard {
                SettingValueRow(
                    title = "群聊名称",
                    value = groupName,
                    enabled = isAdmin,
                ) {
                    if (isAdmin) {
                        draft = groupName
                        editName = true
                    }
                }
                SettingValueRow(
                    title = "我的群昵称",
                    value = selfNameCard.ifBlank { "未设置" },
                ) {
                    draft = selfNameCard
                    editNameCard = true
                }
                SettingValueRow(title = "分享群名片", value = "选择好友") {
                    onShareGroupCard?.invoke()
                }
                if (isAdmin) {
                    SettingValueRow(
                        title = "主动加群方式",
                        value = LinLanImManager.groupAddOptLabel(addOpt),
                    ) {
                        // cycle options
                        scope.launch {
                            val next = when (addOpt) {
                                V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY -> V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH
                                V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH -> V2TIMGroupInfo.V2TIM_GROUP_ADD_FORBID
                                else -> V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
                            }
                            runCatching {
                                LinLanImManager.setGroupInfo(
                                    V2TIMGroupInfo().apply {
                                        groupID = decodedGroupId
                                        groupAddOpt = next
                                    },
                                )
                            }.onSuccess {
                                addOpt = next
                                message = "已更新加群方式"
                            }.onFailure { message = it.message }
                        }
                    }
                    SettingValueRow(
                        title = "邀请进群方式",
                        value = LinLanImManager.groupAddOptLabel(approveOpt),
                    ) {
                        scope.launch {
                            val next = when (approveOpt) {
                                V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY -> V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH
                                V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH -> V2TIMGroupInfo.V2TIM_GROUP_ADD_FORBID
                                else -> V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
                            }
                            runCatching {
                                LinLanImManager.setGroupInfo(
                                    V2TIMGroupInfo().apply {
                                        groupID = decodedGroupId
                                        groupApproveOpt = next
                                    },
                                )
                            }.onSuccess {
                                approveOpt = next
                                message = "已更新邀请方式"
                            }.onFailure { message = it.message }
                        }
                    }
                    SettingValueRow(title = "群管理", value = "") {
                        onOpenManage(decodedGroupId)
                    }
                }
            }

            Spacer(Modifier.height(10.dp))
            SettingsCard {
                ManageToggle("置顶聊天", pinned) { checked ->
                    scope.launch {
                        runCatching { LinLanImManager.pinConversation(convId, checked) }
                            .onSuccess {
                                pinned = checked
                                message = if (checked) "已置顶" else "已取消置顶"
                            }
                            .onFailure { message = it.message }
                    }
                }
                ManageToggle("消息免打扰", muted) { checked ->
                    scope.launch {
                        runCatching { LinLanImManager.setGroupMute(decodedGroupId, checked) }
                            .onSuccess {
                                muted = checked
                                message = if (checked) "已开启免打扰" else "已关闭免打扰"
                            }
                            .onFailure { message = it.message }
                    }
                }
            }

            Spacer(Modifier.height(16.dp))
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color.White)
                    .clickable { confirmQuit = true }
                    .padding(vertical = 14.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    if (isOwner) "解散群聊" else "退出群聊",
                    color = DangerRed,
                    fontSize = 16.sp,
                    fontWeight = FontWeight.Medium,
                )
            }

            message?.let {
                Text(
                    it,
                    color = LinkBlue,
                    modifier = Modifier.padding(16.dp),
                    fontSize = 13.sp,
                )
            }
        }
    }
}

@Composable
private fun GroupNavBar(
    title: String,
    @Suppress("UNUSED_PARAMETER") backLabel: String,
    onBack: () -> Unit,
    trailing: (@Composable () -> Unit)? = null,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(48.dp)
            .background(Color.White)
            .padding(horizontal = 4.dp),
    ) {
        IconButton(
            onClick = onBack,
            modifier = Modifier.align(Alignment.CenterStart),
        ) {
            Icon(
                Icons.AutoMirrored.Filled.ArrowBack,
                contentDescription = "返回",
                tint = LinkBlue,
            )
        }
        Text(
            title,
            fontWeight = FontWeight.SemiBold,
            fontSize = 17.sp,
            modifier = Modifier.align(Alignment.Center),
        )
        Box(modifier = Modifier.align(Alignment.CenterEnd)) {
            trailing?.invoke()
        }
    }
}

@Composable
private fun QuickAction(title: String, onClick: () -> Unit) {
    Box(
        modifier = Modifier
            .height(40.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(Color.White)
            .border(0.5.dp, Color(0xFFD0E8FF), RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(title, color = LinkBlue, fontSize = 13.sp)
    }
}

@Composable
private fun SettingsCard(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(Color.White),
    ) { content() }
}

@Composable
private fun SettingValueRow(
    title: String,
    value: String,
    enabled: Boolean = true,
    showChevron: Boolean = true,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, modifier = Modifier.weight(1f), fontSize = 16.sp)
        Text(
            value,
            color = SecondaryGray,
            fontSize = 14.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.width(140.dp),
            textAlign = TextAlign.End,
        )
        if (showChevron && enabled) {
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                null,
                tint = SecondaryGray,
                modifier = Modifier.size(18.dp),
            )
        }
    }
    HorizontalDivider(color = DividerColor, thickness = 0.5.dp, modifier = Modifier.padding(start = 16.dp))
}

@Composable
private fun SettingLinkRow(title: String, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, modifier = Modifier.weight(1f), fontSize = 16.sp)
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            null,
            tint = SecondaryGray,
            modifier = Modifier.size(18.dp),
        )
    }
    HorizontalDivider(color = DividerColor, thickness = 0.5.dp, modifier = Modifier.padding(start = 16.dp))
}

@Composable
private fun ManageToggle(
    title: String,
    checked: Boolean,
    enabled: Boolean = true,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            title,
            modifier = Modifier.weight(1f),
            fontSize = 16.sp,
            color = if (enabled) Color.Black else SecondaryGray,
        )
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            enabled = enabled,
            colors = SwitchDefaults.colors(checkedTrackColor = LinkBlue),
        )
    }
    HorizontalDivider(color = DividerColor, thickness = 0.5.dp, modifier = Modifier.padding(start = 16.dp))
}

@Composable
private fun MemberAvatar(url: String?, name: String, size: androidx.compose.ui.unit.Dp) {
    Box(
        modifier = Modifier
            .size(size)
            .clip(CircleShape)
            .background(QqBlue),
        contentAlignment = Alignment.Center,
    ) {
        if (!url.isNullOrBlank()) {
            AsyncImage(url, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
        } else {
            Text(
                name.firstOrNull()?.toString() ?: "?",
                color = Color.White,
                fontWeight = FontWeight.Bold,
            )
        }
    }
}
