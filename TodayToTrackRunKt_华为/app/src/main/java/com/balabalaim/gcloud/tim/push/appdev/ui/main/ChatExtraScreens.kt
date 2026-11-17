package com.balabalaim.gcloud.tim.push.appdev.ui.main

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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.tencent.imsdk.v2.V2TIMConversation
import com.tencent.imsdk.v2.V2TIMFriendInfo
import com.tencent.imsdk.v2.V2TIMMessage
import kotlinx.coroutines.launch
import java.net.URLDecoder
import java.nio.charset.StandardCharsets

data class ForwardTarget(
    val key: String,
    val title: String,
    val receiver: String?,
    val groupId: String?,
    val isGroup: Boolean,
    val faceUrl: String? = null,
)

data class ForwardPayload(
    val messages: List<V2TIMMessage>,
    val merge: Boolean,
    val mergeTitle: String = "",
)

private enum class ForwardFilter { All, Group, Friend }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ForwardTargetScreen(
    onBack: () -> Unit,
    onPicked: (receiver: String?, groupId: String?, title: String) -> Unit = { _, _, _ -> },
    onPickedMultiple: ((List<ForwardTarget>) -> Unit)? = null,
    title: String = "选择会话",
    friendsOnly: Boolean = false,
    multiSelect: Boolean = false,
    excludeConversationId: String? = null,
) {
    val friends = remember { mutableStateListOf<V2TIMFriendInfo>() }
    val conversations = remember { mutableStateListOf<V2TIMConversation>() }
    var message by remember { mutableStateOf<String?>(null) }
    var filter by remember { mutableStateOf(if (friendsOnly) ForwardFilter.Friend else ForwardFilter.All) }
    var search by remember { mutableStateOf("") }
    val selectedKeys = remember { mutableStateListOf<String>() }
    var sending by remember { mutableStateOf(false) }
    val linkBlue = Color(0xFF007AFF)
    val pageBg = Color(0xFFF2F2F7)

    LaunchedEffect(Unit) {
        runCatching { LinLanImManager.fetchFriendList() }.onSuccess { friends.clear(); friends.addAll(it) }
        if (!friendsOnly) {
            runCatching { LinLanImManager.fetchConversations() }
                .onSuccess { conversations.clear(); conversations.addAll(it) }
                .onFailure { message = it.message }
        }
    }

    val targets = remember(friends.toList(), conversations.toList(), friendsOnly, excludeConversationId) {
        buildList {
            friends.forEach { friend ->
                val name = friend.friendRemark?.ifBlank { null }
                    ?: friend.userProfile?.nickName
                    ?: friend.userID
                val key = "c2c_${friend.userID}"
                if (key != excludeConversationId) {
                    add(
                        ForwardTarget(
                            key = key,
                            title = name,
                            receiver = friend.userID,
                            groupId = null,
                            isGroup = false,
                            faceUrl = friend.userProfile?.faceUrl,
                        ),
                    )
                }
            }
            if (!friendsOnly) {
                conversations.filter { it.type == V2TIMConversation.V2TIM_GROUP }.forEach { conv ->
                    val gid = conv.groupID ?: conv.conversationID.removePrefix("group_")
                    val gTitle = conv.showName ?: gid
                    val key = conv.conversationID ?: "group_$gid"
                    if (key != excludeConversationId && "group_$gid" != excludeConversationId) {
                        add(
                            ForwardTarget(
                                key = key,
                                title = gTitle,
                                receiver = null,
                                groupId = gid,
                                isGroup = true,
                                faceUrl = conv.faceUrl,
                            ),
                        )
                    }
                }
            }
        }
    }

    val filtered = remember(targets, filter, search, friendsOnly) {
        val q = search.trim()
        targets.filter { item ->
            val tabOk = when (filter) {
                ForwardFilter.All -> true
                ForwardFilter.Group -> item.isGroup
                ForwardFilter.Friend -> !item.isGroup
            }
            val searchOk = q.isEmpty() ||
                item.title.contains(q, ignoreCase = true) ||
                item.key.contains(q, ignoreCase = true)
            tabOk && searchOk
        }
    }

    fun toggle(key: String) {
        if (sending) return
        if (selectedKeys.contains(key)) selectedKeys.remove(key) else selectedKeys.add(key)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title) },
                navigationIcon = {
                    IconButton(
                        onClick = onBack,
                        enabled = !sending,
                    ) {
                        Icon(Icons.Default.Close, contentDescription = "关闭")
                    }
                },
                actions = {
                    if (multiSelect) {
                        TextButton(
                            enabled = selectedKeys.isNotEmpty() && !sending,
                            onClick = {
                                if (sending) return@TextButton
                                val picked = targets.filter { it.key in selectedKeys }
                                if (picked.isEmpty()) return@TextButton
                                sending = true
                                onPickedMultiple?.invoke(picked)
                            },
                        ) {
                            Text(
                                when {
                                    sending -> "发送中…"
                                    selectedKeys.isEmpty() -> "发送"
                                    else -> "发送(${selectedKeys.size})"
                                },
                                color = if (selectedKeys.isEmpty() || sending) Color.Gray else linkBlue,
                            )
                        }
                    }
                },
            )
        },
        containerColor = pageBg,
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            if (!friendsOnly) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    listOf(
                        ForwardFilter.All to "全部",
                        ForwardFilter.Group to "群聊",
                        ForwardFilter.Friend to "好友",
                    ).forEach { (tab, label) ->
                        val active = filter == tab
                        Text(
                            label,
                            color = if (active) Color.White else Color(0xFF111111),
                            modifier = Modifier
                                .clip(RoundedCornerShape(16.dp))
                                .background(if (active) Color(0xFF1A3A6B) else Color(0xFFE5E5EA))
                                .clickable { filter = tab }
                                .padding(horizontal = 14.dp, vertical = 8.dp),
                            fontSize = 14.sp,
                        )
                    }
                }
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(Color(0xFFE5E5EA))
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Default.Search, null, tint = Color(0xFF8E8E93), modifier = Modifier.size(18.dp))
                Spacer(modifier = Modifier.width(8.dp))
                BasicTextField(
                    value = search,
                    onValueChange = { search = it },
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                    decorationBox = { inner ->
                        if (search.isEmpty()) Text("搜索会话", color = Color(0xFF8E8E93), fontSize = 14.sp)
                        inner()
                    },
                )
            }
            message?.let {
                Text(it, color = Color.Red, modifier = Modifier.padding(16.dp))
            }
            LazyColumn(modifier = Modifier.fillMaxSize()) {
                items(filtered, key = { it.key }) { item ->
                    val checked = selectedKeys.contains(item.key)
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(Color.White)
                            .clickable {
                                if (multiSelect) {
                                    toggle(item.key)
                                } else {
                                    onPicked(item.receiver, item.groupId, item.title)
                                }
                            }
                            .padding(horizontal = 16.dp, vertical = 10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        if (multiSelect) {
                            Box(
                                modifier = Modifier
                                    .size(22.dp)
                                    .clip(CircleShape)
                                    .background(if (checked) linkBlue else Color.Transparent)
                                    .border(
                                        1.5.dp,
                                        if (checked) linkBlue else Color(0xFFC7C7CC),
                                        CircleShape,
                                    ),
                                contentAlignment = Alignment.Center,
                            ) {
                                if (checked) {
                                    Icon(
                                        Icons.Default.Check,
                                        contentDescription = null,
                                        tint = Color.White,
                                        modifier = Modifier.size(14.dp),
                                    )
                                }
                            }
                            Spacer(modifier = Modifier.width(12.dp))
                        }
                        Box(
                            modifier = Modifier
                                .size(44.dp)
                                .clip(CircleShape)
                                .background(Color(0xFFD0E8FF)),
                            contentAlignment = Alignment.Center,
                        ) {
                            if (!item.faceUrl.isNullOrBlank()) {
                                AsyncImage(
                                    model = item.faceUrl,
                                    contentDescription = null,
                                    modifier = Modifier.fillMaxSize(),
                                    contentScale = ContentScale.Crop,
                                )
                            } else {
                                Text(
                                    item.title.take(1),
                                    color = linkBlue,
                                    fontWeight = FontWeight.SemiBold,
                                )
                            }
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                        Text(item.title, fontSize = 16.sp, modifier = Modifier.weight(1f))
                    }
                    HorizontalDivider(color = Color(0xFFE5E5EA), thickness = 0.5.dp)
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanFriendDetailScreen(
    userId: String,
    title: String,
    onBack: () -> Unit,
    onOpenChat: (conversationId: String, title: String, isGroup: Boolean) -> Unit,
    onOpenReport: (targetId: String, targetType: String, targetName: String) -> Unit,
    onDeleted: () -> Unit,
) {
    val decodedId = remember(userId) { URLDecoder.decode(userId, StandardCharsets.UTF_8.name()) }
    val decodedTitle = remember(title) { URLDecoder.decode(title, StandardCharsets.UTF_8.name()) }
    val convId = "c2c_$decodedId"
    var nickName by remember { mutableStateOf(decodedTitle) }
    var faceUrl by remember { mutableStateOf<String?>(null) }
    var remark by remember { mutableStateOf("") }
    var isFriend by remember { mutableStateOf(false) }
    var pinned by remember { mutableStateOf(false) }
    var muted by remember { mutableStateOf(false) }
    var editingRemark by remember { mutableStateOf(false) }
    var showDelete by remember { mutableStateOf(false) }
    var showBlack by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val pageBg = Color(0xFFF2F2F7)
    val linkBlue = Color(0xFF007AFF)
    val displayName = remark.ifBlank { nickName.ifBlank { decodedTitle } }

    LaunchedEffect(decodedId) {
        isFriend = LinLanImManager.isFriend(decodedId)
        LinLanImManager.fetchUserBrief(decodedId)?.let {
            nickName = it.nickName.ifBlank { decodedTitle }
            faceUrl = it.faceUrl
        }
        remark = LinLanImManager.fetchFriendRemark(decodedId)
        if (isFriend) {
            pinned = LinLanImManager.isConversationPinned(convId)
            muted = LinLanImManager.isC2CMuted(decodedId)
        }
    }

    if (showDelete) {
        AlertDialog(
            onDismissRequest = { showDelete = false },
            title = { Text("删除好友") },
            text = { Text("确定删除 $displayName？将同时删除聊天入口。") },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching {
                            LinLanImManager.deleteFriend(decodedId)
                            LinLanImManager.deleteConversation(convId)
                        }.onSuccess { onDeleted() }
                            .onFailure { message = it.message }
                        showDelete = false
                    }
                }) { Text("删除") }
            },
            dismissButton = { TextButton(onClick = { showDelete = false }) { Text("取消") } },
        )
    }
    if (showBlack) {
        AlertDialog(
            onDismissRequest = { showBlack = false },
            title = { Text("拉黑") },
            text = { Text("拉黑后将无法收到对方消息，是否继续？") },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching { LinLanImManager.addToBlackList(decodedId) }
                            .onSuccess {
                                Toast.makeText(context, "已拉黑", Toast.LENGTH_SHORT).show()
                                showBlack = false
                            }
                            .onFailure { message = it.message; showBlack = false }
                    }
                }) { Text("拉黑") }
            },
            dismissButton = { TextButton(onClick = { showBlack = false }) { Text("取消") } },
        )
    }
    if (editingRemark) {
        AlertDialog(
            onDismissRequest = { editingRemark = false },
            title = { Text("设置备注") },
            text = {
                OutlinedTextField(
                    value = remark,
                    onValueChange = { remark = it },
                    label = { Text("备注名") },
                    placeholder = { Text("填写备注，方便识别") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        runCatching { LinLanImManager.setFriendRemark(decodedId, remark.trim()) }
                            .onSuccess {
                                editingRemark = false
                                message = "备注已更新"
                            }
                            .onFailure { message = it.message }
                    }
                }) { Text("保存") }
            },
            dismissButton = { TextButton(onClick = { editingRemark = false }) { Text("取消") } },
        )
    }

    Scaffold(
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            TopAppBar(
                title = { Text("个人详情") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .background(pageBg),
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 12.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color.White)
                    .padding(16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(64.dp)
                        .clip(CircleShape)
                        .background(Color(0xFF5AC8FA)),
                    contentAlignment = Alignment.Center,
                ) {
                    if (!faceUrl.isNullOrBlank()) {
                        AsyncImage(
                            faceUrl,
                            contentDescription = null,
                            modifier = Modifier.fillMaxSize(),
                            contentScale = ContentScale.Crop,
                        )
                    } else {
                        Text(
                            nickName.firstOrNull()?.toString() ?: "?",
                            color = Color.White,
                            fontSize = 26.sp,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
                Spacer(modifier = Modifier.width(14.dp))
                Text(
                    nickName.ifBlank { "未设置昵称" },
                    fontSize = 20.sp,
                    fontWeight = FontWeight.SemiBold,
                )
            }

            if (isFriend) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(Color.White)
                        .clickable { editingRemark = true }
                        .padding(horizontal = 16.dp, vertical = 16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        "备注",
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Medium,
                        color = Color(0xFF111111),
                    )
                    Spacer(modifier = Modifier.weight(1f))
                    Text(
                        remark.ifBlank { "点击设置备注" },
                        fontSize = 15.sp,
                        color = if (remark.isBlank()) linkBlue else Color(0xFF8E8E93),
                    )
                    Icon(
                        Icons.AutoMirrored.Filled.KeyboardArrowRight,
                        contentDescription = null,
                        tint = Color(0xFFC7C7CC),
                    )
                }
            }

            message?.let {
                Text(
                    it,
                    color = Color(0xFFFF3B30),
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                )
            }

            Spacer(modifier = Modifier.height(20.dp))
            if (!isFriend) {
                Button(
                    onClick = {
                        scope.launch {
                            runCatching { LinLanImManager.addFriend(decodedId) }
                                .onSuccess {
                                    Toast.makeText(context, "已发送好友申请", Toast.LENGTH_SHORT).show()
                                    message = "已发送好友申请"
                                }
                                .onFailure { message = it.message }
                        }
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp),
                ) { Text("添加好友") }
            } else {
                Button(
                    onClick = { onOpenChat(convId, displayName, false) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp),
                ) { Text("发消息") }
                Spacer(Modifier.height(12.dp))
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(Color.White)
                        .padding(horizontal = 16.dp),
                ) {
                    DetailSwitch("置顶聊天", pinned) { checked ->
                        scope.launch {
                            runCatching { LinLanImManager.pinConversation(convId, checked) }
                                .onSuccess {
                                    pinned = checked
                                    message = if (checked) "已置顶" else "已取消置顶"
                                }
                                .onFailure { message = it.message }
                        }
                    }
                    HorizontalDivider(color = Color(0xFFE5E5EA), thickness = 0.5.dp)
                    DetailSwitch("消息免打扰", muted) { checked ->
                        scope.launch {
                            runCatching { LinLanImManager.setC2CMute(decodedId, checked) }
                                .onSuccess {
                                    muted = checked
                                    message = if (checked) "已开启免打扰" else "已关闭免打扰"
                                }
                                .onFailure { message = it.message }
                        }
                    }
                }
                Spacer(Modifier.height(12.dp))
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(Color.White),
                ) {
                    TextButton(
                        onClick = { onOpenReport(decodedId, "user", displayName) },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("举报") }
                    HorizontalDivider(color = Color(0xFFE5E5EA), thickness = 0.5.dp)
                    TextButton(
                        onClick = { showBlack = true },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("拉黑") }
                    HorizontalDivider(color = Color(0xFFE5E5EA), thickness = 0.5.dp)
                    TextButton(
                        onClick = { showDelete = true },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("删除好友", color = Color(0xFFFF3B30)) }
                }
            }
        }
    }
}

@Composable
private fun DetailSwitch(title: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, modifier = Modifier.weight(1f))
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

object ForwardBus {
    @Volatile
    var pending: ForwardPayload? = null
}

object MergerDetailBus {
    @Volatile
    var pending: V2TIMMessage? = null
}
