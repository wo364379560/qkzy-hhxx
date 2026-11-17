package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import kotlinx.coroutines.flow.collectLatest
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
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.data.model.SearchUserResult
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.tencent.imsdk.v2.V2TIMFriendApplication
import com.tencent.imsdk.v2.V2TIMFriendInfo
import com.tencent.imsdk.v2.V2TIMGroupInfo
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanAddFriendScreen(
    onBack: () -> Unit,
    onOpenChat: (conversationId: String, title: String, isGroup: Boolean) -> Unit,
    onOpenFriendDetail: ((userId: String, title: String) -> Unit)? = null,
) {
    var keyword by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var searching by remember { mutableStateOf(false) }
    val results = remember { mutableStateListOf<SearchUserResult>() }
    val friendIds = remember { mutableStateListOf<String>() }
    val addingFriendIds = remember { mutableStateListOf<String>() }
    val waitingOwnerRequests by com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.waitingRequests.collectAsState()
    val ownerResults by com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.results.collectAsState()

    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }

    LaunchedEffect(Unit) {
        runCatching { LinLanImManager.fetchFriendList() }
            .onSuccess {
                friendIds.clear()
                friendIds.addAll(it.map { f -> f.userID })
            }
    }

    fun doSearch() {
        val q = keyword.trim()
        if (q.isEmpty()) return
        scope.launch {
            searching = true
            message = null
            runCatching { api.searchUsersPreferExact(q) }
                .onSuccess {
                    results.clear()
                    results.addAll(it)
                    if (it.isEmpty()) message = "未找到用户"
                }
                .onFailure { message = it.message }
            searching = false
        }
    }

    Scaffold(
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            TopAppBar(
                title = { Text("添加好友") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
        ) {
            OutlinedTextField(
                value = keyword,
                onValueChange = { keyword = it },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                label = { Text("搜索睿之禾号和手机号") },
                placeholder = { Text("搜索睿之禾号和手机号") },
            )
            Spacer(Modifier.height(8.dp))
            Button(
                onClick = { doSearch() },
                enabled = !searching,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(if (searching) "搜索中…" else "搜索") }
            message?.let {
                Text(it, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(top = 8.dp))
            }
            Spacer(Modifier.height(8.dp))
            LazyColumn(modifier = Modifier.weight(1f)) {
                items(results, key = { it.imUserId }) { user ->
                    val alreadyFriend = friendIds.contains(user.imUserId)
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp)
                            .clip(RoundedCornerShape(10.dp))
                            .background(Color.White)
                            .then(
                                if (alreadyFriend) {
                                    Modifier.clickable {
                                        onOpenFriendDetail?.invoke(user.imUserId, user.displayName)
                                    }
                                } else {
                                    Modifier
                                },
                            )
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(44.dp)
                                .clip(CircleShape)
                                .background(Color(0xFF5AC8FA)),
                            contentAlignment = Alignment.Center,
                        ) {
                            if (!user.avatarUrl.isNullOrBlank()) {
                                AsyncImage(
                                    user.avatarUrl,
                                    contentDescription = null,
                                    modifier = Modifier.fillMaxSize(),
                                    contentScale = ContentScale.Crop,
                                )
                            } else {
                                Text(
                                    user.displayName.firstOrNull()?.toString() ?: "?",
                                    color = Color.White,
                                    fontWeight = FontWeight.Bold,
                                )
                            }
                        }
                        Spacer(Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                user.displayName,
                                fontSize = 16.sp,
                                fontWeight = FontWeight.Medium,
                            )
                            val ownershipState=ownerResults[com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()+":"+user.imUserId]?.state
                            if (ownershipState in setOf("pending","blocked","conflict")) Text(if(ownershipState=="pending") "好友已建立，归属同步中" else "好友已建立，归属待管理员确认",fontSize=12.sp,color=Color.Gray)
                            if (!alreadyFriend && !user.canAddFriend) {
                                Text(
                                    user.addBlockedReason.ifBlank { "暂不可添加" },
                                    color = Color.Gray,
                                    fontSize = 12.sp,
                                    modifier = Modifier.padding(top = 2.dp),
                                )
                            }
                        }
                        if (alreadyFriend) {
                            TextButton(onClick = {
                                onOpenChat("c2c_${user.imUserId}", user.displayName, false)
                            }) { Text("发消息") }
                        } else {
                            TextButton(
                                enabled = user.canAddFriend && !addingFriendIds.contains(user.imUserId) && !waitingOwnerRequests.contains(com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()+":"+user.imUserId),
                                onClick = {
                                scope.launch {
                                    val actionScope = com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()
                                    addingFriendIds.add(user.imUserId)
                                    runCatching { LinLanImManager.addFriend(user.imUserId) }
                                        .onSuccess { outcome ->
                                            if (actionScope != com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()) return@onSuccess
                                            message = outcome.toast
                                            Toast.makeText(context, outcome.toast, Toast.LENGTH_SHORT).show()
                                            if (outcome == LinLanImManager.AddFriendOutcome.AddedDirectly ||
                                                outcome == LinLanImManager.AddFriendOutcome.AlreadyFriends
                                            ) {
                                                if (!friendIds.contains(user.imUserId)) {
                                                    friendIds.add(user.imUserId)
                                                }
                                            }
                                        }
                                        .onFailure { if (actionScope == com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()) message = it.message }
                                    addingFriendIds.remove(user.imUserId)
                                }
                                },
                            ) { Text(when {
                                addingFriendIds.contains(user.imUserId) -> "处理中"
                                waitingOwnerRequests.contains(com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.sessionKey()+":"+user.imUserId) -> "待验证"
                                !user.canAddFriend -> "不可添加"
                                else -> "添加"
                            }) }
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanCreateGroupScreen(
    onBack: () -> Unit,
    onCreated: (groupId: String, title: String) -> Unit,
) {
    var groupName by remember { mutableStateOf("") }
    var createdGroupId by remember { mutableStateOf<String?>(null) }
    var message by remember { mutableStateOf<String?>(null) }
    var creating by remember { mutableStateOf(false) }
    val friends = remember { mutableStateListOf<LinLanImManager.SelectableContact>() }
    val selected = remember { mutableStateListOf<String>() }
    val scope = rememberCoroutineScope()

    suspend fun reloadContacts() {
        runCatching { LinLanImManager.fetchSelectableContacts() }
            .onSuccess {
                friends.clear()
                friends.addAll(it)
            }
            .onFailure { message = it.message }
    }

    LaunchedEffect(Unit) { reloadContacts() }
    LaunchedEffect(Unit) {
        LinLanImManager.friendsChanged.collectLatest { reloadContacts() }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("新建群聊") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
                actions = {
                    TextButton(
                        enabled = !creating,
                        onClick = {
                            val name = groupName.trim()
                            createdGroupId?.let { onCreated(it, name); return@TextButton }
                            if (name.isEmpty()) {
                                message = "请输入群名称"
                                return@TextButton
                            }
                            scope.launch {
                                creating = true
                                message = null
                                runCatching {
                                    // 默认建社群（Community）；Public 仍保留 createPublicGroup
                                    LinLanImManager.createCommunityGroup(name, selected.toList())
                                }.onSuccess { gid ->
                                    onCreated(gid, name)
                                }.onFailure {
                                    if (it is LinLanImManager.PartialGroupCreation) createdGroupId = it.groupId
                                    message = it.message
                                }
                                creating = false
                            }
                        },
                    ) { Text(if (creating) "创建中" else if (createdGroupId != null) "进入群聊" else "创建") }
                },
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
        ) {
            OutlinedTextField(
                value = groupName,
                onValueChange = { groupName = it },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                label = { Text("群名称") },
            )
            Spacer(Modifier.height(8.dp))
            Text("群类型：Community（社群）", style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(8.dp))
            Text("已选 ${selected.size}/100 位好友", style = MaterialTheme.typography.titleMedium)
            TextButton(enabled = !creating && createdGroupId == null, onClick = {
                if (selected.size >= minOf(100, friends.size)) selected.clear()
                else {
                    val next = (selected + friends.map { it.userId }).distinct().take(100)
                    selected.clear()
                    selected.addAll(next)
                }
            }) { Text(if (selected.isNotEmpty() && selected.size >= minOf(100, friends.size)) "取消全选" else "全选（最多100人）") }
            message?.let {
                Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 4.dp))
            }
            LazyColumn(modifier = Modifier.weight(1f)) {
                if (friends.isEmpty()) {
                    item {
                        Text(
                            "暂无联系人可选，请先添加好友或与对方发起过单聊",
                            color = Color.Gray,
                            modifier = Modifier.padding(vertical = 16.dp),
                        )
                    }
                }
                items(friends, key = { it.userId }) { friend ->
                    val checked = selected.contains(friend.userId)
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable(enabled = !creating && createdGroupId == null) {
                                if (checked) selected.remove(friend.userId)
                                else if (selected.size < 100) selected.add(friend.userId)
                                else message = "最多选择 100 位好友"
                            }
                            .padding(vertical = 4.dp),
                    ) {
                        Checkbox(
                            checked = checked,
                            enabled = !creating && createdGroupId == null,
                            onCheckedChange = {
                                if (!it) selected.remove(friend.userId)
                                else if (!checked && selected.size < 100) selected.add(friend.userId)
                                else message = "最多选择 100 位好友"
                            },
                        )
                        Text(friend.displayName, modifier = Modifier.padding(top = 12.dp))
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanFriendRequestsScreen(onBack: () -> Unit) {
    val pending = remember { mutableStateListOf<String>() }
    val applications = remember { mutableStateListOf<V2TIMFriendApplication>() }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun reload() {
        scope.launch {
            runCatching { LinLanImManager.fetchIncomingFriendApplications() }
                .onSuccess { applications.clear(); applications.addAll(it) }
                .onFailure { message = it.message }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("新的朋友") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        LazyColumn(modifier = Modifier.padding(padding)) {
            message?.let { msg ->
                item { Text(msg, modifier = Modifier.padding(16.dp), color = MaterialTheme.colorScheme.primary) }
            }
            if (applications.isEmpty()) {
                item {
                    Text("暂无好友申请", modifier = Modifier.padding(16.dp), color = Color.Gray)
                }
            }
            items(applications, key = { it.userID + it.addTime }) { app ->
                val name = app.nickname?.ifBlank { null } ?: app.userID
                ListItem(
                    headlineContent = { Text(name) },
                    supportingContent = {
                        Text(app.addWording?.ifBlank { "请求添加你为好友" } ?: "请求添加你为好友")
                    },
                    trailingContent = {
                        Row {
                            TextButton(enabled = app.userID !in pending, onClick = {
                                pending.add(app.userID)
                                scope.launch {
                                    runCatching { LinLanImManager.acceptFriendApplication(app) }
                                        .onSuccess { message = "已接受"; reload() }
                                        .onFailure { message = it.message }
                                    pending.remove(app.userID)
                                    reload()
                                }
                            }) { Text("接受") }
                            TextButton(enabled = app.userID !in pending, onClick = {
                                pending.add(app.userID)
                                scope.launch {
                                    runCatching { LinLanImManager.rejectFriendApplication(app) }
                                        .onSuccess { message = "已拒绝"; reload() }
                                        .onFailure { message = it.message }
                                    pending.remove(app.userID)
                                    reload()
                                }
                            }) { Text("拒绝") }
                        }
                    },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanBlacklistScreen(onBack: () -> Unit) {
    val users = remember { mutableStateListOf<V2TIMFriendInfo>() }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun reload() {
        scope.launch {
            runCatching { LinLanImManager.fetchBlackList() }
                .onSuccess { users.clear(); users.addAll(it) }
                .onFailure { message = it.message }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("黑名单") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        LazyColumn(modifier = Modifier.padding(padding)) {
            message?.let { msg ->
                item { Text(msg, modifier = Modifier.padding(16.dp)) }
            }
            if (users.isEmpty()) {
                item { Text("黑名单为空", modifier = Modifier.padding(16.dp), color = Color.Gray) }
            }
            items(users, key = { it.userID }) { user ->
                val name = user.friendRemark?.ifBlank { null }
                    ?: user.userProfile?.nickName
                    ?: user.userID
                ListItem(
                    leadingContent = {
                        Box(
                            Modifier
                                .size(40.dp)
                                .clip(CircleShape)
                                .background(Color(0xFF12B7F5)),
                            contentAlignment = Alignment.Center,
                        ) {
                            val url = user.userProfile?.faceUrl
                            if (!url.isNullOrBlank()) {
                                AsyncImage(url, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
                            } else {
                                Text(name.firstOrNull()?.toString() ?: "?", color = Color.White, fontWeight = FontWeight.Bold)
                            }
                        }
                    },
                    headlineContent = { Text(name) },
                    trailingContent = {
                        TextButton(onClick = {
                            scope.launch {
                                runCatching { LinLanImManager.removeFromBlackList(user.userID) }
                                    .onSuccess { message = "已移出黑名单"; reload() }
                                    .onFailure { message = it.message }
                            }
                        }) { Text("移出") }
                    },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanJoinedGroupsScreen(
    onBack: () -> Unit,
    onOpenChat: (conversationId: String, title: String, isGroup: Boolean) -> Unit,
) {
    val groups = remember { mutableStateListOf<V2TIMGroupInfo>() }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    val membershipRevision by LinLanImManager.groupMembershipChanged.collectAsState()
    LaunchedEffect(membershipRevision) {
        runCatching { LinLanImManager.fetchJoinedGroups() }
            .onSuccess { groups.clear(); groups.addAll(it) }
            .onFailure { message = it.message }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("群聊") },
                navigationIcon = { com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton(onClick = onBack) },
            )
        }
    ) { padding ->
        LazyColumn(modifier = Modifier.padding(padding)) {
            message?.let { msg ->
                item { Text(msg, modifier = Modifier.padding(16.dp)) }
            }
            if (groups.isEmpty()) {
                item { Text("暂无群聊", modifier = Modifier.padding(16.dp), color = Color.Gray) }
            }
            items(groups, key = { it.groupID }) { group ->
                val title = group.groupName?.ifBlank { null } ?: group.groupID
                ListItem(
                    leadingContent = {
                        Box(
                            Modifier
                                .size(40.dp)
                                .clip(CircleShape)
                                .background(Color(0xFF12B7F5)),
                            contentAlignment = Alignment.Center,
                        ) {
                            val url = group.faceUrl
                            if (!url.isNullOrBlank()) {
                                AsyncImage(url, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
                            } else {
                                Text(title.firstOrNull()?.toString() ?: "群", color = Color.White, fontWeight = FontWeight.Bold)
                            }
                        }
                    },
                    headlineContent = { Text(title) },
                    supportingContent = { Text("成员 ${group.memberCount}") },
                    modifier = Modifier.clickable {
                        onOpenChat("group_${group.groupID}", title, true)
                    },
                )
            }
        }
    }
}
