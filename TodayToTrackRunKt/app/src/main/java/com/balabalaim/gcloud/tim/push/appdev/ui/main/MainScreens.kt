package com.balabalaim.gcloud.tim.push.appdev.ui.main

import androidx.compose.material.icons.filled.EventNote
import androidx.compose.material.icons.filled.Assessment
import androidx.compose.material.icons.filled.Info
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanCanvas
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanInk
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanInkSecondary

import androidx.compose.ui.draw.alpha
import androidx.compose.foundation.interaction.collectIsDraggedAsState

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.Manifest
import android.content.ContentValues
import android.graphics.Bitmap
import android.media.MediaMetadataRetriever
import android.net.Uri
import android.os.SystemClock
import android.os.Build
import android.os.Environment
import android.content.pm.PackageManager
import android.provider.MediaStore
import android.widget.Toast
import android.widget.VideoView
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.basicMarquee
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items as gridItems
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.InsertDriveFile
import androidx.compose.material.icons.automirrored.filled.Message
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.AddCircleOutline
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Contacts
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.EmojiEmotions
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.GroupAdd
import androidx.compose.material.icons.filled.Image
import androidx.compose.material.icons.filled.Keyboard
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.NotificationsOff
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.PushPin
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material.icons.outlined.Block
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material.icons.outlined.DirectionsRun
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.roundToInt
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.FileProvider
import androidx.core.content.ContextCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.repeatOnLifecycle
import androidx.lifecycle.LifecycleEventObserver
import coil.compose.AsyncImage
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanAccent
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.data.local.PinnedMessage
import com.balabalaim.gcloud.tim.push.appdev.data.model.ChatProfile
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanEmojiCatalog
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanGroupPolicy
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanQuoteHelper
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanVoiceMessagePlayer
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanVoiceRecorder
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanReliability
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallManager
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallMediaType
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload
import com.balabalaim.gcloud.tim.push.appdev.rtc.RtcCallPeer
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVoiceRecordOverlay
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintAccent
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintCard
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintPageBg
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintSecondary
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintTitle
import com.tencent.imsdk.v2.V2TIMConversation
import com.tencent.imsdk.v2.V2TIMFriendApplication
import com.tencent.imsdk.v2.V2TIMFriendInfo
import com.tencent.imsdk.v2.V2TIMMessage
import com.tencent.imsdk.v2.V2TIMManager
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import java.io.File
import java.io.FileOutputStream
import java.net.URLDecoder
import java.net.HttpURLConnection
import java.net.URL
import java.nio.charset.StandardCharsets
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Date
import java.util.Locale

private val QqBlue = LinLanAccent
private val QqPageBg = MintPageBg
private val QqSearchBg = Color(0xFFF0EBE3)
private val QqSecondary = MintSecondary
private val QqDivider = Color(0xFFEAE4DA)
private val ContactActionBlue = MintAccent
private val MentionBlue = Color(0xFF576B95)
private val MentionSelfHighlight = Color(0xFFD9F0FF)

private data class MentionMember(
    val userId: String,
    val displayName: String,
    val isAtAll: Boolean,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanMainScreen(
    onOpenRealname: () -> Unit,
    onOpenDevices: () -> Unit,
    onOpenDeviceProtection: () -> Unit,
    onOpenHelp: () -> Unit,
    onOpenAntiFraud: () -> Unit,

    onOpenChat: (conversationId: String, title: String, isGroup: Boolean) -> Unit,
    onAddFriend: () -> Unit,
    onCreatePublicGroup: () -> Unit,
    onOpenFriendDetail: (userId: String, title: String) -> Unit,
    onOpenFriendRequests: () -> Unit,
    onOpenBlacklist: () -> Unit,
    onOpenGroups: () -> Unit,
    onOpenProfile: () -> Unit,
    onOpenSecurity: () -> Unit,
    onOpenNotify: () -> Unit,
    onOpenTeen: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
    onLogout: () -> Unit,
) {
    val activityContext = LocalContext.current
    var antiFraudResolved by rememberSaveable { mutableStateOf(!isHuaweiDevice()) }
    HuaweiHomeAntiFraudNotice(onResolved = { antiFraudResolved = true })
    val notifyPermissionLauncher = androidx.activity.compose.rememberLauncherForActivityResult(
        androidx.activity.result.contract.ActivityResultContracts.RequestPermission(),
    ) { }
    LaunchedEffect(antiFraudResolved) {
        if (
            antiFraudResolved &&
            Build.VERSION.SDK_INT >= 33 &&
            LinLanAppContainer.appPrefs().privacyAccepted &&
            ContextCompat.checkSelfPermission(activityContext, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            notifyPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }
    // 离开主页再返回时保持当前 Tab（通讯录/我的），避免总是回到消息页
    var tab by rememberSaveable { mutableIntStateOf(0) }
    var totalUnread by remember { mutableLongStateOf(0L) }
    val scope = rememberCoroutineScope()

    fun refreshUnread() {
        scope.launch {
            totalUnread = runCatching { LinLanImManager.totalUnreadCount() }.getOrDefault(0L)
        }
    }

    LaunchedEffect(Unit) {
        // 启动页若因弱网提前进入，这里补齐域名探测 / TIM 登录
        runCatching { LinLanAppContainer.authRepository().ensureSessionReady() }
        refreshUnread()
        LinLanImManager.conversationsChanged.collectLatest { refreshUnread() }
    }

    Scaffold(
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        bottomBar = {
            NavigationBar(
                containerColor = Color.White,
                contentColor = Color(0xFF333333),
            ) {
                val entries = listOf(
                    Triple(0, "运动", Icons.Outlined.DirectionsRun),
                    Triple(4, "计划", Icons.Default.EventNote),
                    Triple(5, "记录", Icons.Default.Assessment),
                    Triple(1, "消息", Icons.AutoMirrored.Filled.Message),
                    Triple(2, "通讯录", Icons.Default.Contacts),
                    Triple(3, "我的", Icons.Default.Person),
                )
                entries.forEach { (id, title, icon) ->
                    NavigationBarItem(selected = tab == id, onClick = { tab = id },
                        icon = { BadgedBox(badge = { if (id == 1 && totalUnread > 0) Badge { Text(if (totalUnread > 99) "99+" else totalUnread.toString()) } }) {
                            Icon(icon, title, modifier = Modifier.size(23.dp))
                        } }, label = { Text(title, fontSize = 11.sp, maxLines = 1) },
                        colors = NavigationBarItemDefaults.colors(selectedIconColor = LinLanAccent,
                            selectedTextColor = LinLanAccent, indicatorColor = LinLanAccent.copy(alpha = .13f)))
                }
            }
        }
    ) { padding ->
        val bottomPad = padding.calculateBottomPadding()
        when (tab) {
            1 -> ConversationTab(
                onOpenAntiFraud = onOpenAntiFraud,

                modifier = Modifier.padding(bottom = bottomPad),
                onOpenChat = onOpenChat,
                onAddFriend = onAddFriend,
                onCreatePublicGroup = onCreatePublicGroup,
                onOpenProfile = onOpenProfile,
            )
            2 -> ContactTab(
                modifier = Modifier.padding(bottom = bottomPad),
                onOpenChat = onOpenChat,
                onOpenFriendDetail = onOpenFriendDetail,
                onAddFriend = onAddFriend,
                onCreatePublicGroup = onCreatePublicGroup,
                onOpenFriendRequests = onOpenFriendRequests,
                onOpenBlacklist = onOpenBlacklist,
                onOpenGroups = onOpenGroups,
            )
            0 -> Box(Modifier.padding(bottom = bottomPad)) { LinLanRunScreen(onOpenChat = onOpenChat, onOpenRecords = { tab = 5 }, onOpenPlan = { tab = 4 }) }
            4 -> Box(Modifier.padding(bottom = bottomPad)) { RunPlanScreen { tab = 0 } }
            5 -> Box(Modifier.padding(bottom = bottomPad)) { LinLanRunScreen(onOpenChat = onOpenChat, recordsOnly = true) }
            else -> Box(modifier = Modifier.padding(bottom = bottomPad)) {
                LinLanMineHubScreen(
                    onOpenRealname = onOpenRealname,
                    onOpenDevices = onOpenDevices,
                    onOpenDeviceProtection = onOpenDeviceProtection,
                    onOpenHelp = onOpenHelp,

                    onOpenProfile = onOpenProfile,
                    onOpenSecurity = onOpenSecurity,
                    onOpenNotify = onOpenNotify,
                    onOpenTeen = onOpenTeen,
                    onOpenLegal = onOpenLegal,
                    onLogout = onLogout,
                )
            }
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ConversationTab(
    onOpenAntiFraud: () -> Unit,

    modifier: Modifier = Modifier,
    onOpenChat: (String, String, Boolean) -> Unit,
    onAddFriend: () -> Unit,
    onCreatePublicGroup: () -> Unit,
    onOpenProfile: () -> Unit,
) {
    val items = remember { mutableStateListOf<V2TIMConversation>() }
    var menuExpanded by remember { mutableStateOf(false) }
    var search by remember { mutableStateOf("") }
    var profile by remember { mutableStateOf<ChatProfile?>(null) }
    var deleteTarget by remember { mutableStateOf<V2TIMConversation?>(null) }
    val scope = rememberCoroutineScope()
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }

    fun reload() {
        scope.launch {
            runCatching { LinLanImManager.fetchConversations() }
                .onSuccess { items.clear(); items.addAll(it) }
            runCatching { api.fetchProfile() }.onSuccess { profile = it }
        }
    }

    val imState by LinLanImManager.connectionState.collectAsState()
    LaunchedEffect(imState) {
        if (imState == LinLanImManager.ConnectionState.Connected) reload()
        LinLanImManager.conversationsChanged.collectLatest { reload() }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) reload()
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    val filtered = remember(items.toList(), search) {
        val q = search.trim()
        if (q.isEmpty()) items.toList()
        else items.filter { conv ->
            (conv.showName ?: conv.conversationID).contains(q, ignoreCase = true) ||
                conv.lastMessage?.let { LinLanImManager.messagePreview(it).contains(q, ignoreCase = true) } == true
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MintPageBg)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                "睿之禾 · 消息",
                color = MintTitle,
                fontWeight = FontWeight.Bold,
                fontSize = 28.sp,
                modifier = Modifier.weight(1f),
            )
            Box {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(CircleShape)
                        .background(MintAccent)
                        .clickable { menuExpanded = true },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Default.Add, contentDescription = "添加", tint = Color.White, modifier = Modifier.size(22.dp))
                }
                DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
                    DropdownMenuItem(
                        text = { Text("添加好友") },
                        onClick = { menuExpanded = false; onAddFriend() },
                    )
                    DropdownMenuItem(
                        text = { Text("新建群") },
                        onClick = { menuExpanded = false; onCreatePublicGroup() },
                    )
                    DropdownMenuItem(
                        text = { Text("个人资料") },
                        onClick = { menuExpanded = false; onOpenProfile() },
                    )
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp)
                .clip(RoundedCornerShape(14.dp))
                .background(MintCard)
                .padding(horizontal = 14.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Default.Search, null, tint = QqSecondary, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            BasicTextField(
                value = search,
                onValueChange = { search = it },
                singleLine = true,
                cursorBrush = SolidColor(MintAccent),
                decorationBox = { inner ->
                    Box {
                        if (search.isEmpty()) {
                            Text("搜索会话", color = QqSecondary, fontSize = 15.sp)
                        }
                        inner()
                    }
                },
                modifier = Modifier.weight(1f),
            )
        }

        HuaweiAntiFraudBanner(onOpenAntiFraud)
        Spacer(Modifier.height(10.dp))

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (filtered.isEmpty()) item {
                SafetyCard(if (search.isBlank()) "与朋友分享每一次进步" else "没有找到会话", if (search.isBlank()) "完成运动后，分享打卡到这里，一起聊聊今天的收获。" else "试试其他关键词")
            }
            items(filtered, key = { it.conversationID }) { conv ->
                ConversationRow(
                    conv = conv,
                    onClick = {
                        val title = conv.showName ?: conv.conversationID
                        val isGroup = conv.type == V2TIMConversation.V2TIM_GROUP
                        onOpenChat(conv.conversationID, title, isGroup)
                    },
                    onTogglePin = {
                        val willPin = !conv.isPinned
                        scope.launch {
                            runCatching { LinLanImManager.pinConversation(conv.conversationID, willPin) }
                                .onSuccess {
                                    Toast.makeText(
                                        context,
                                        if (willPin) "已置顶" else "已取消置顶",
                                        Toast.LENGTH_SHORT,
                                    ).show()
                                }
                                .onFailure {
                                    Toast.makeText(context, it.message ?: "置顶失败", Toast.LENGTH_SHORT).show()
                                }
                        }
                    },
                    onDelete = { deleteTarget = conv },
                )
            }
        }
    }

    deleteTarget?.let { target ->
        val name = target.showName ?: target.conversationID
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text("删除会话") },
            text = { Text("确定删除「$name」？删除后该会话会从消息列表消失。") },
            confirmButton = {
                TextButton(
                    onClick = {
                        val id = target.conversationID
                        deleteTarget = null
                        scope.launch {
                            runCatching { LinLanImManager.deleteConversation(id) }
                                .onSuccess {
                                    Toast.makeText(context, "已删除", Toast.LENGTH_SHORT).show()
                                }
                                .onFailure {
                                    Toast.makeText(context, it.message ?: "删除失败", Toast.LENGTH_SHORT).show()
                                }
                        }
                    },
                ) { Text("删除", color = Color(0xFFFA5151)) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text("取消") }
            },
        )
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ConversationRow(
    conv: V2TIMConversation,
    onClick: () -> Unit,
    onTogglePin: () -> Unit,
    onDelete: () -> Unit,
) {
    val title = conv.showName ?: conv.conversationID
    val preview = conv.lastMessage?.let { LinLanImManager.messagePreview(it) }.orEmpty()
    val unread = conv.unreadCount
    val timeText = formatConversationTime(conv.lastMessage?.timestamp ?: 0)
    val muted = LinLanImManager.isRecvOptMuted(conv.recvOpt)
    val pinned = conv.isPinned
    val atFlags = remember(conv.conversationID, unread, conv.groupAtInfoList) {
        LinLanImManager.groupAtMentionFlags(conv)
    }
    val previewAnnotated = remember(preview, unread, atFlags) {
        buildAnnotatedString {
            if (unread > 0 && atFlags.showAtAll) {
                withStyle(SpanStyle(color = Color(0xFFF25151))) { append("[@所有人]") }
            }
            if (unread > 0 && atFlags.showAtMe) {
                withStyle(SpanStyle(color = Color(0xFFF25151))) { append("[有人@我]") }
            }
            withStyle(SpanStyle(color = QqSecondary)) { append(preview) }
        }
    }
    val rowBg = if (pinned) Color(0xFFFFF8E8) else MintCard
    val density = LocalDensity.current
    val revealPx = with(density) { 160.dp.toPx() }
    var offsetX by remember(conv.conversationID) { mutableFloatStateOf(0f) }
    var showMenu by remember(conv.conversationID) { mutableStateOf(false) }

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp)),
    ) {
        Row(
            modifier = Modifier.matchParentSize(),
            horizontalArrangement = Arrangement.End,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .width(80.dp)
                    .fillMaxHeight()
                    .background(MintAccent)
                    .clickable {
                        offsetX = 0f
                        onTogglePin()
                    },
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    if (pinned) "取消置顶" else "置顶",
                    color = Color.White,
                    fontSize = 13.sp,
                    textAlign = TextAlign.Center,
                )
            }
            Box(
                modifier = Modifier
                    .width(80.dp)
                    .fillMaxHeight()
                    .background(Color(0xFFFA5151))
                    .clickable {
                        offsetX = 0f
                        onDelete()
                    },
                contentAlignment = Alignment.Center,
            ) {
                Text("删除", color = Color.White, fontSize = 13.sp)
            }
        }
    Row(
        modifier = Modifier
            .offset { IntOffset(offsetX.roundToInt(), 0) }
            .fillMaxWidth()
            .background(rowBg)
            .combinedClickable(
                onClick = {
                    if (offsetX < -8f) {
                        offsetX = 0f
                    } else {
                        onClick()
                    }
                },
                onLongClick = { showMenu = true },
            )
            .pointerInput(revealPx) {
                detectHorizontalDragGestures(
                    onDragEnd = {
                        offsetX = if (offsetX < -revealPx / 2f) -revealPx else 0f
                    },
                    onHorizontalDrag = { change, dragAmount ->
                        change.consume()
                        offsetX = (offsetX + dragAmount).coerceIn(-revealPx, 0f)
                    },
                )
            }
            .padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RoundAvatar(url = conv.faceUrl, name = title, size = 48.dp)
        Column(
            modifier = Modifier
                .weight(1f)
                .padding(start = 12.dp, end = 8.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    title,
                    color = MintTitle,
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 16.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (pinned) {
                    Icon(
                        Icons.Default.PushPin,
                        contentDescription = "置顶",
                        tint = MintAccent,
                        modifier = Modifier
                            .padding(end = 4.dp)
                            .size(14.dp),
                    )
                }
                if (muted) {
                    Icon(
                        Icons.Default.NotificationsOff,
                        contentDescription = "免打扰",
                        tint = QqSecondary,
                        modifier = Modifier
                            .padding(end = 4.dp)
                            .size(16.dp),
                    )
                }
                Text(timeText, color = QqSecondary, fontSize = 12.sp)
            }
            Spacer(Modifier.height(4.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    previewAnnotated,
                    fontSize = 13.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                if (unread > 0) {
                    Spacer(Modifier.width(8.dp))
                    Box(
                        modifier = Modifier
                            .clip(CircleShape)
                            .background(if (muted) Color(0xFFBDBDBD) else Color(0xFFFA5151))
                            .padding(horizontal = 6.dp, vertical = 2.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = if (unread > 99) "99+" else unread.toString(),
                            color = Color.White,
                            fontSize = 11.sp,
                        )
                    }
                }
            }
        }
    }
        DropdownMenu(expanded = showMenu, onDismissRequest = { showMenu = false }) {
            DropdownMenuItem(
                text = { Text(if (pinned) "取消置顶" else "置顶") },
                leadingIcon = { Icon(Icons.Default.PushPin, contentDescription = null) },
                onClick = {
                    showMenu = false
                    offsetX = 0f
                    onTogglePin()
                },
            )
            DropdownMenuItem(
                text = { Text("删除") },
                leadingIcon = { Icon(Icons.Default.Delete, contentDescription = null) },
                onClick = {
                    showMenu = false
                    offsetX = 0f
                    onDelete()
                },
            )
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ContactTab(
    modifier: Modifier = Modifier,
    onOpenChat: (String, String, Boolean) -> Unit,
    onOpenFriendDetail: (String, String) -> Unit,
    onAddFriend: () -> Unit,
    onCreatePublicGroup: () -> Unit,
    onOpenFriendRequests: () -> Unit,
    onOpenBlacklist: () -> Unit,
    onOpenGroups: () -> Unit,
) {
    val friends = remember { mutableStateListOf<V2TIMFriendInfo>() }
    var applyCount by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val lifecycleOwner = LocalLifecycleOwner.current

    fun reload() {
        scope.launch {
            runCatching { LinLanImManager.fetchFriendList() }
                .onSuccess { friends.clear(); friends.addAll(it) }
            runCatching { LinLanImManager.fetchIncomingFriendApplications() }
                .onSuccess { applyCount = it.size }
        }
    }

    val imState by LinLanImManager.connectionState.collectAsState()
    LaunchedEffect(imState) {
        if (imState == LinLanImManager.ConnectionState.Connected) reload()
        LinLanImManager.friendsChanged.collectLatest { reload() }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) reload()
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    val sorted = remember(friends.toList()) {
        friends.sortedBy {
            (it.friendRemark?.ifBlank { null } ?: it.userProfile?.nickName ?: it.userID)
                .lowercase(Locale.ROOT)
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MintPageBg)
            .windowInsetsPadding(WindowInsets.statusBars),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                "通讯录",
                color = MintTitle,
                fontWeight = FontWeight.Bold,
                fontSize = 28.sp,
                modifier = Modifier.weight(1f),
            )
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(MintAccent)
                    .clickable(onClick = onAddFriend),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Default.PersonAdd, contentDescription = "添加好友", tint = Color.White, modifier = Modifier.size(20.dp))
            }
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                Text("把熟悉的人，放在身边", fontSize = 14.sp, color = MintSecondary, modifier = Modifier.padding(bottom = 16.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    ContactQuickTile("新的朋友", Icons.Default.PersonAdd, applyCount, Modifier.weight(1f), onOpenFriendRequests)
                    ContactQuickTile("发起群聊", Icons.Default.GroupAdd, 0, Modifier.weight(1f), onCreatePublicGroup)
                }
                Spacer(Modifier.height(12.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    ContactQuickTile("我的群聊", Icons.Default.Folder, 0, Modifier.weight(1f), onOpenGroups)
                    ContactQuickTile("黑名单", Icons.Outlined.Block, 0, Modifier.weight(1f), onOpenBlacklist)
                }
            }
            item {
                Text("全部联系人 · ${sorted.size}", color = MintTitle, fontSize = 16.sp,
                    fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 18.dp, bottom = 8.dp))
                if (sorted.isEmpty()) Text("还没有联系人，点击右上角添加朋友。", color = MintSecondary, fontSize = 14.sp)
            }
            items(sorted, key = { it.userID }) { friend ->
                val name = friend.friendRemark?.ifBlank { null }
                    ?: friend.userProfile?.nickName
                    ?: friend.userID
                val avatar = friend.userProfile?.faceUrl
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(16.dp))
                        .background(MintCard)
                        .combinedClickable(
                            onClick = { onOpenChat("c2c_${friend.userID}", name, false) },
                            onLongClick = { onOpenFriendDetail(friend.userID, name) },
                        )
                        .padding(horizontal = 14.dp, vertical = 12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    RoundAvatar(url = avatar, name = name, size = 44.dp)
                    Text(
                        name,
                        color = MintTitle,
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Medium,
                        modifier = Modifier.padding(start = 12.dp),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun ContactActionRow(
    icon: ImageVector,
    title: String,
    badge: Int = 0,
    onClick: () -> Unit,
    showDivider: Boolean = true,
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(MintAccent.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(icon, null, tint = MintAccent, modifier = Modifier.size(20.dp))
            }
            Text(
                title,
                color = MintTitle,
                fontSize = 16.sp,
                modifier = Modifier
                    .padding(start = 12.dp)
                    .weight(1f),
            )
            if (badge > 0) {
                Box(
                    modifier = Modifier
                        .clip(CircleShape)
                        .background(Color(0xFFFA5151))
                        .padding(horizontal = 6.dp, vertical = 2.dp),
                ) {
                    Text(
                        if (badge > 99) "99+" else badge.toString(),
                        color = Color.White,
                        fontSize = 11.sp,
                    )
                }
            }
        }
        if (showDivider) {
            HorizontalDivider(color = QqDivider, thickness = 0.5.dp, modifier = Modifier.padding(start = 62.dp))
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun RoundAvatar(
    url: String?,
    name: String,
    size: androidx.compose.ui.unit.Dp,
    onClick: (() -> Unit)? = null,
    onLongClick: (() -> Unit)? = null,
) {
    val mod = Modifier
        .size(size)
        .clip(CircleShape)
        .background(QqBlue)
        .then(if (onClick != null || onLongClick != null) Modifier.combinedClickable(onClick = { onClick?.invoke() }, onLongClick = onLongClick) else Modifier)
    Box(mod, contentAlignment = Alignment.Center) {
        if (!url.isNullOrBlank()) {
            AsyncImage(url, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
        } else {
            Text(
                name.firstOrNull()?.toString() ?: "?",
                color = Color.White,
                fontWeight = FontWeight.Bold,
                fontSize = (size.value * 0.38f).sp,
            )
        }
    }
}

private fun formatConversationTime(seconds: Long): String {
    if (seconds <= 0L) return ""
    val date = Date(seconds * 1000)
    val now = Calendar.getInstance()
    val target = Calendar.getInstance().apply { time = date }
    return when {
        now.get(Calendar.YEAR) == target.get(Calendar.YEAR) &&
            now.get(Calendar.DAY_OF_YEAR) == target.get(Calendar.DAY_OF_YEAR) ->
            SimpleDateFormat("HH:mm", Locale.getDefault()).format(date)
        now.get(Calendar.YEAR) == target.get(Calendar.YEAR) ->
            SimpleDateFormat("MM/dd", Locale.getDefault()).format(date)
        else -> SimpleDateFormat("yyyy/MM/dd", Locale.getDefault()).format(date)
    }
}

internal sealed class ChatMediaPreview {
    data class Image(val model: String) : ChatMediaPreview()
    data class Video(val pathOrUrl: String) : ChatMediaPreview()
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanChatScreen(
    onOpenRealname: () -> Unit,
    conversationId: String,
    title: String,
    isGroup: Boolean,
    onBack: () -> Unit,
    onOpenGroupDetail: ((groupId: String, title: String) -> Unit)? = null,
    onOpenGroupCard: ((groupId: String, title: String) -> Unit)? = null,
    onOpenFriendDetail: ((userId: String, title: String) -> Unit)? = null,
    onForward: ((messages: List<V2TIMMessage>, merge: Boolean, mergeTitle: String) -> Unit)? = null,
    onOpenMergerDetail: ((V2TIMMessage) -> Unit)? = null,
    onPickContactCard: (() -> Unit)? = null,
    onOpenSearch: (() -> Unit)? = null,
) {
    val decodedTitle = remember(title) { URLDecoder.decode(title, StandardCharsets.UTF_8.name()) }
    val decodedConversationId = remember(conversationId) {
        URLDecoder.decode(conversationId, StandardCharsets.UTF_8.name())
    }
    DisposableEffect(decodedConversationId) {
        com.balabalaim.gcloud.tim.push.appdev.notify.LinLanMessageAlertCenter.setActiveConversation(decodedConversationId)
        onDispose {
            com.balabalaim.gcloud.tim.push.appdev.notify.LinLanMessageAlertCenter.setActiveConversation(null)
        }
    }
    val sessionStore = remember { LinLanAppContainer.sessionStore() }
    val chatSession by sessionStore.sessionFlow.collectAsState(initial = com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession())
    val chatAppId = chatSession.sdkAppId.takeIf { it > 0 } ?: com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants.SDK_APP_ID
    val historyScope = "$chatAppId:${chatSession.userId}:$decodedConversationId:${sessionStore.sessionGeneration()}"
    val latestHistoryScope by rememberUpdatedState(historyScope)
    val historyEpoch = remember(historyScope) { java.util.concurrent.atomic.AtomicLong() }
    val removedHistoryIDs = remember(historyScope) { mutableSetOf<String>() }
    val messages = remember(historyScope) { mutableStateListOf<V2TIMMessage>() }
    DisposableEffect(historyScope) { onDispose { historyEpoch.incrementAndGet() } }

    val mediaSendProgress = remember { mutableStateMapOf<Int, Int>() }
    val imConnectionState by LinLanImManager.connectionState.collectAsState()
    var input by remember { mutableStateOf("") }
    var lastSendAtMs by remember { mutableLongStateOf(0L) }
    val inputFocus = remember { FocusRequester() }
    val keyboardController = LocalSoftwareKeyboardController.current
    var peerReadThrough by remember(decodedConversationId) { mutableLongStateOf(0L) }
    val peerReadIDs = remember(decodedConversationId) { mutableStateListOf<String>() }
    var quoteTarget by remember { mutableStateOf<V2TIMMessage?>(null) }
    var menuMsg by remember { mutableStateOf<V2TIMMessage?>(null) }
    var showEmoji by remember { mutableStateOf(false) }
    var showMorePanel by remember { mutableStateOf(false) }
    var showRunPicker by remember { mutableStateOf(false) }
    var voiceInputMode by remember { mutableStateOf(false) }
    var composerIntent by remember { mutableLongStateOf(0L) }
    var voicePermissionIntent by remember { mutableLongStateOf(-1L) }
    var voiceRecording by remember { mutableStateOf(false) }
    var voiceCancelMode by remember { mutableStateOf(false) }
    var voiceLevel by remember { mutableFloatStateOf(0.2f) }
    var mediaPreview by remember { mutableStateOf<ChatMediaPreview?>(null) }
    var highlightedMsgId by remember { mutableStateOf<String?>(null) }
    var suppressBottomScroll by remember { mutableStateOf(false) }
    var historyLoading by remember(historyScope) { mutableStateOf(false) }
    var historyError by remember(historyScope) { mutableStateOf<String?>(null) }
    var historyEndReached by remember(historyScope) { mutableStateOf(false) }
    var groupProfileAccess by remember { mutableStateOf(true) }
    var groupMemberRoleMap by remember { mutableStateOf<Map<String, Int>>(emptyMap()) }
    var selfGroupRole by remember { mutableIntStateOf(0) }
    /** 个人禁言 / 全员禁言（普通成员）时隐藏转发、多选。 */
    var groupSendBlocked by remember(historyScope) { mutableStateOf(isGroup) }
    var groupSendBlockedHint by remember(historyScope) { mutableStateOf("正在确认群权限") }
    var groupSendCanRetry by remember(historyScope) { mutableStateOf(false) }
    val permissionRequest = remember(historyScope) { java.util.concurrent.atomic.AtomicLong() }
    DisposableEffect(historyScope) { onDispose { permissionRequest.incrementAndGet() } }
    var mentionMembers by remember { mutableStateOf<List<MentionMember>>(emptyList()) }
    var mentionNameToUserId by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var mentionLoading by remember { mutableStateOf(false) }
    var mentionPickerDismissed by remember { mutableStateOf(false) }
    var multiSelectMode by remember { mutableStateOf(false) }
    val selectedMsgIds = remember { mutableStateListOf<String>() }
    val scope = rememberCoroutineScope()
    fun requestTextInput() {
        composerIntent++
        val expected = composerIntent
        showEmoji = false
        showMorePanel = false
        voiceInputMode = false
        scope.launch {
            androidx.compose.runtime.withFrameNanos { }
            if (composerIntent == expected) {
                inputFocus.requestFocus()
                keyboardController?.show()
            }
        }
    }

    val context = LocalContext.current
    val voiceRecorder = remember(context) { LinLanVoiceRecorder(context.applicationContext) }
    val listState = rememberLazyListState()
    var followLatest by remember(decodedConversationId) { mutableStateOf(true) }
    var initialBottomReady by remember(historyScope) { mutableStateOf(false) }
    var safetyLayoutReady by remember(historyScope) { mutableStateOf(false) }
    var groupLayoutReady by remember(historyScope) { mutableStateOf(!isGroup) }
    var pinsLayoutReady by remember(historyScope) { mutableStateOf(!isGroup) }
    var pendingNewMessageCount by remember(decodedConversationId) { mutableStateOf(0) }
    var jumpToLatestRequested by remember(decodedConversationId) { mutableStateOf(false) }
    fun requestLatestMessages() {
        pendingNewMessageCount = 0
        followLatest = true
        jumpToLatestRequested = true
    }
    val userDragging by listState.interactionSource.collectIsDraggedAsState()
    LaunchedEffect(userDragging, decodedConversationId) {
        if (userDragging) followLatest = false
        else if (initialBottomReady && !listState.canScrollForward) followLatest = true
    }
    LaunchedEffect(listState, decodedConversationId) {
        snapshotFlow { !listState.isScrollInProgress && !listState.canScrollForward }
            .collect { atBottom ->
                if (initialBottomReady && atBottom) {
                    followLatest = true
                    pendingNewMessageCount = 0
                }
            }
    }
    val pinStore = remember { LinLanAppContainer.pinnedMessageStore() }
    val pinTick by pinStore.changed.collectAsState()
    var cloudPins by remember(decodedConversationId) { mutableStateOf<List<PinnedMessage>>(emptyList()) }
    val localPins = remember(decodedConversationId, pinTick) { pinStore.list(decodedConversationId) }
    val pins = if (isGroup) cloudPins else localPins
    val historyPageSize = 30

    fun addOptimisticMedia(message: V2TIMMessage) {
        requestLatestMessages()
        if (messages.none { it === message }) messages.add(message)
        mediaSendProgress[System.identityHashCode(message)] = 0
    }

    fun updateOptimisticMedia(message: V2TIMMessage?, progress: Int) {
        if (message != null) mediaSendProgress[System.identityHashCode(message)] = progress
    }

    fun finishOptimisticMedia(draft: V2TIMMessage?, sent: V2TIMMessage) {
        if (draft == null) {
            requestLatestMessages()
            messages.add(sent)
            return
        }
        mediaSendProgress.remove(System.identityHashCode(draft))
        val index = messages.indexOfFirst { it === draft }
        if (index >= 0) messages[index] = sent else messages.add(sent)
    }

    fun failOptimisticMedia(draft: V2TIMMessage?) {
        if (draft != null) mediaSendProgress[System.identityHashCode(draft)] = -1
    }

    fun isHistoryVisible(msg: V2TIMMessage) = LinLanImManager.belongsToConversation(msg, decodedConversationId, isGroup) &&
        !LinLanImManager.shouldSilentlyHideRevokedMessage(msg) && !LinLanImManager.shouldHideRtcSignalingInChat(msg) &&
        (!LinLanImManager.isGroupTipsMessage(msg) || LinLanImManager.shouldShowGroupTipsInChat(msg))

    fun historyKey(msg: V2TIMMessage): String = msg.msgID?.takeIf { it.isNotBlank() } ?: "local-${System.identityHashCode(msg)}"
    val historyPager = remember(historyScope) {
        com.balabalaim.gcloud.tim.push.appdev.im.RawHistoryPager(historyPageSize, ::historyKey, ::isHistoryVisible) { cursor: V2TIMMessage? ->
            LinLanImManager.fetchHistory(decodedConversationId, historyPageSize, cursor)
        }
    }
    fun mergeMessages(incoming: List<V2TIMMessage>, history: Boolean) {
        val next = com.balabalaim.gcloud.tim.push.appdev.im.mergeHistoryTimeline(messages.filterNot { LinLanImManager.shouldSilentlyHideRevokedMessage(it) }, incoming.filterNot { LinLanImManager.shouldSilentlyHideRevokedMessage(it) }, ::historyKey, { it.timestamp.takeIf { value -> value > 0 } ?: Long.MAX_VALUE }, history, removedHistoryIDs, { it.seq.toLong() })
        next.forEachIndexed { index, message -> if (index < messages.size) messages[index] = message else messages.add(message) }
        while (messages.size > next.size) messages.removeAt(messages.lastIndex)
    }

    suspend fun loadHistory(latest: Boolean) {
        if (!latest && (historyLoading || historyEndReached)) return
        if (!LinLanImManager.isReadyFor(chatSession.userId, chatAppId)) { historyError = "聊天尚未就绪，连接后自动重试"; return }
        val epoch = historyEpoch.incrementAndGet()
        val active = { epoch == historyEpoch.get() && latestHistoryScope == historyScope && LinLanImManager.isReadyFor(chatSession.userId, chatAppId) }
        historyLoading = true
        historyError = null
        val anchorID = messages.getOrNull((listState.firstVisibleItemIndex - 1).coerceAtLeast(0))?.let(::historyKey)
        val anchorOffset = listState.firstVisibleItemScrollOffset
        suppressBottomScroll = !latest
        try {
            val page = historyPager.load(latest, active)
            if (!active()) return
            mergeMessages(page, true)
            historyEndReached = !historyPager.hasMore
            if (!latest && anchorID != null) {
                val index = messages.indexOfFirst { historyKey(it) == anchorID }
                if (index >= 0) listState.scrollToItem(index + 1, anchorOffset)
            }
            LinLanImManager.markConversationRead(decodedConversationId, page)
        } catch (error: kotlinx.coroutines.CancellationException) { throw error }
          catch (error: Exception) { if (epoch == historyEpoch.get() && latestHistoryScope == historyScope) historyError = error.message ?: "历史消息加载失败，请重试" }
        finally {
            if (epoch == historyEpoch.get() && latestHistoryScope == historyScope) { historyLoading = false; suppressBottomScroll = false }
        }
    }

    suspend fun loadOlderHistory() = loadHistory(latest = false)

    fun selectedMessages(): List<V2TIMMessage> =
        messages.filter { it.msgID != null && it.msgID in selectedMsgIds && LinLanImManager.canForwardMessage(it) }

    fun startForward(merge: Boolean) {
        if (isGroup && groupSendBlocked) {
            Toast.makeText(context, groupSendBlockedHint, Toast.LENGTH_SHORT).show()
            return
        }
        val picked = selectedMessages()
        if (picked.isEmpty()) {
            Toast.makeText(context, "请选择可转发的消息", Toast.LENGTH_SHORT).show()
            return
        }
        if (picked.size > LinLanReliability.MAX_FORWARD_MESSAGES) {
            Toast.makeText(context, "最多选择 ${LinLanReliability.MAX_FORWARD_MESSAGES} 条消息", Toast.LENGTH_SHORT).show()
            return
        }
        val mergeTitle = LinLanImManager.mergeForwardTitle(picked, isGroup, decodedTitle)
        onForward?.invoke(picked, merge, mergeTitle)
        multiSelectMode = false
        selectedMsgIds.clear()
    }

    val canShowForwardActions = !isGroup || !groupSendBlocked

    fun locatePinnedMessage(msgId: String) {
        if (msgId.isBlank()) return
        scope.launch {
            suppressBottomScroll = true
            var index = messages.indexOfFirst { it.msgID == msgId }
            if (index < 0) {
                var found = runCatching { LinLanImManager.findMessages(listOf(msgId)) }.getOrNull()?.firstOrNull()
                var cursor: V2TIMMessage? = null
                val seen = mutableSetOf<String>()
                while (found == null) {
                    kotlinx.coroutines.currentCoroutineContext().ensureActive()
                    val page = runCatching { LinLanImManager.fetchHistory(decodedConversationId, historyPageSize, cursor) }.getOrNull() ?: break
                    found = page.firstOrNull { it.msgID == msgId }
                    val next = page.lastOrNull() ?: break
                    if (found != null || page.size < historyPageSize || !seen.add(next.msgID.orEmpty())) break
                    cursor = next
                }
                if (found == null || !LinLanImManager.belongsToConversation(found, decodedConversationId, isGroup)) {
                    Toast.makeText(context, "无法定位到该消息", Toast.LENGTH_SHORT).show()
                    suppressBottomScroll = false
                    return@launch
                }
                val targetMessage = found
                val insertAt = messages.indexOfFirst { it.timestamp > targetMessage.timestamp }
                    .let { if (it < 0) messages.size else it }
                messages.add(insertAt, targetMessage)
                index = insertAt
            }
            listState.animateScrollToItem(index + 1)
            highlightedMsgId = msgId
            kotlinx.coroutines.delay(300)
            suppressBottomScroll = false
        }
    }

    LaunchedEffect(highlightedMsgId) {
        if (highlightedMsgId != null) {
            kotlinx.coroutines.delay(1600)
            if (highlightedMsgId != null) highlightedMsgId = null
        }
    }
    val chatLifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(chatLifecycleOwner, decodedConversationId) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                ChatSearchBus.consume(decodedConversationId)?.let { msgId ->
                    scope.launch { locatePinnedMessage(msgId) }
                }
            }
        }
        chatLifecycleOwner.lifecycle.addObserver(observer)
        onDispose { chatLifecycleOwner.lifecycle.removeObserver(observer) }
    }
    val peerId = remember(decodedConversationId) {
        decodedConversationId.removePrefix("c2c_").removePrefix("group_")
    }
    val isSecurityAssistant = !isGroup && peerId == "security_assistant"
    val receiver = if (isGroup) null else peerId
    val groupId = if (isGroup) peerId else null

    LaunchedEffect(groupId, historyScope) {
        if (groupId != null) LinLanImManager.groupMembershipChanged.collect {
            if (LinLanImManager.wasRemoved(groupId)) onBack()
        }
    }
    suspend fun reloadGroupPins() {
        if (!isGroup || groupId.isNullOrBlank()) return
        runCatching { LinLanImManager.fetchGroupPinnedMessages(groupId) }.onSuccess { cloudPins = it }
        pinsLayoutReady = true
    }

    val pinLifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(historyScope, groupId, imConnectionState, pinLifecycle) {
        if (isGroup) pinLifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) { reloadGroupPins(); kotlinx.coroutines.awaitCancellation() }
    }

    LaunchedEffect(groupId, isGroup) {
        if (!isGroup || groupId.isNullOrBlank()) return@LaunchedEffect
        LinLanImManager.groupPinnedChanged.collectLatest { changedGroupId ->
            if (changedGroupId == groupId) reloadGroupPins()
        }
    }

    suspend fun applyGroupSendPermission() {
        if (!isGroup || peerId.isBlank()) {
            groupSendBlocked = false
            return
        }
        val request = permissionRequest.incrementAndGet()
        val permission = try {
            LinLanImManager.resolveGroupSendPermission(peerId)
        } catch (error: kotlinx.coroutines.CancellationException) {
            throw error
        } catch (_: Exception) {
            LinLanImManager.GroupSendPermission(false, "群权限暂未确认，点此重试", canRetry = true)
        }
        kotlinx.coroutines.currentCoroutineContext().ensureActive()
        if (request != permissionRequest.get() || latestHistoryScope != historyScope) return
        groupLayoutReady = true
        groupSendCanRetry = permission.canRetry
        groupSendBlocked = !permission.allowsSend
        groupSendBlockedHint = permission.blockedHint.ifBlank { "全体禁言" }
        if (groupSendBlocked) {
            multiSelectMode = false
            selectedMsgIds.clear()
            showEmoji = false
            showMorePanel = false
            voiceInputMode = false
            quoteTarget = null
        }
    }

    LaunchedEffect(decodedConversationId, isGroup) {
        if (isGroup) {
            // 重登后属性拉取可能短暂失败；失败时 fail-closed，避免「禁止私聊」失效
            var access: Boolean? = null
            repeat(3) { attempt ->
                val result = runCatching { LinLanGroupPolicy.fetchMemberProfileAccessEnabled(peerId) }
                if (result.isSuccess) {
                    access = result.getOrThrow()
                    return@repeat
                }
                if (attempt < 2) kotlinx.coroutines.delay(400L * (attempt + 1))
            }
            groupProfileAccess = access ?: false
            groupMemberRoleMap = runCatching {
                LinLanImManager.fetchGroupMemberRoleMap(peerId)
            }.getOrDefault(emptyMap())
            val infoRole = runCatching { LinLanImManager.fetchGroupInfo(peerId)?.role }
                .getOrNull() ?: 0
            val loginId = V2TIMManager.getInstance().loginUser.orEmpty()
            selfGroupRole = when {
                infoRole > 0 -> infoRole
                loginId.isNotBlank() -> groupMemberRoleMap[loginId] ?: 0
                else -> 0
            }
            applyGroupSendPermission()
        } else {
            groupProfileAccess = true
            groupMemberRoleMap = emptyMap()
            selfGroupRole = 0
            groupSendBlocked = false
            mentionMembers = emptyList()
            mentionNameToUserId = emptyMap()
        }
    }

    LaunchedEffect(historyScope, imConnectionState) {
        if (isGroup && imConnectionState == LinLanImManager.ConnectionState.Connected) applyGroupSendPermission()
    }

    LaunchedEffect(groupId, isGroup) {
        if (!isGroup || groupId.isNullOrBlank()) return@LaunchedEffect
        LinLanImManager.groupSendPolicyChanged.collectLatest { changedGroupId ->
            if (changedGroupId == groupId) applyGroupSendPermission()
        }
    }

    suspend fun reloadMentionMembers() {
        if (!isGroup || peerId.isBlank()) {
            mentionMembers = emptyList()
            mentionNameToUserId = emptyMap()
            return
        }
        mentionLoading = true
        try {
            val selfId = com.tencent.imsdk.v2.V2TIMManager.getInstance().loginUser.orEmpty()
            val members = runCatching { LinLanImManager.fetchAllGroupMembers(peerId) }.getOrDefault(emptyList())
            val role = members.firstOrNull { it.userID == selfId }?.role
                ?: groupMemberRoleMap[selfId]
                ?: runCatching {
                    LinLanImManager.fetchGroupMemberRoleMap(peerId)[selfId]
                }.getOrNull()
                ?: 0
            val items = buildList {
                // 群主/管理员：顶部「全体成员」
                if (LinLanImManager.canManageGroup(role)) {
                    add(
                        MentionMember(
                            userId = LinLanImManager.AT_ALL_TAG,
                            displayName = LinLanImManager.AT_ALL_DISPLAY,
                            isAtAll = true,
                        ),
                    )
                }
                members.filter { it.userID != selfId && !it.userID.isNullOrBlank() }.forEach { m ->
                    add(
                        MentionMember(
                            userId = m.userID.orEmpty(),
                            displayName = LinLanImManager.groupMemberDisplayName(m),
                            isAtAll = false,
                        ),
                    )
                }
            }
            mentionMembers = items
            mentionNameToUserId = items.associate { it.displayName to it.userId }
        } finally {
            mentionLoading = false
        }
    }

    // @ 成员列表单独拉取，避免被群策略网络请求拖住
    LaunchedEffect(decodedConversationId, isGroup) {
        if (isGroup) reloadMentionMembers()
        else {
            mentionMembers = emptyList()
            mentionNameToUserId = emptyMap()
        }
    }

    fun mentionQueryAtEnd(text: String): String? {
        if (!isGroup) return null
        val at = text.lastIndexOf('@')
        if (at < 0) return null
        val query = text.substring(at + 1)
        if (query.contains(' ') || query.contains('\n')) return null
        return query
    }

    fun resolveAtUserIds(text: String): List<String> {
        if (!isGroup) return emptyList()
        val ids = linkedSetOf<String>()
        val allToken = "@${LinLanImManager.AT_ALL_DISPLAY}"
        if (text.contains(allToken)) {
            ids += mentionNameToUserId[LinLanImManager.AT_ALL_DISPLAY] ?: LinLanImManager.AT_ALL_TAG
        }
        mentionNameToUserId.forEach { (name, uid) ->
            if (name == LinLanImManager.AT_ALL_DISPLAY) return@forEach
            if (text.contains("@$name")) ids += uid
        }
        return ids.toList()
    }

    fun applyMentionSelection(member: MentionMember) {
        val at = input.lastIndexOf('@')
        if (at < 0) return
        val mentionText = if (member.isAtAll) {
            "@${LinLanImManager.AT_ALL_DISPLAY} "
        } else {
            "@${member.displayName} "
        }
        input = input.substring(0, at) + mentionText
        mentionNameToUserId = mentionNameToUserId + (member.displayName to member.userId)
        mentionPickerDismissed = true
    }

    val mentionFilter = mentionQueryAtEnd(input)
    LaunchedEffect(mentionFilter) {
        if (mentionFilter == null) {
            mentionPickerDismissed = false
        } else {
            // 输入 @ 时若列表还空，立刻再拉一次
            if (mentionMembers.isEmpty()) {
                reloadMentionMembers()
            }
        }
    }
    val showMentionPicker = mentionFilter != null && !mentionPickerDismissed
    val filteredMentions = remember(mentionMembers, mentionFilter) {
        val q = mentionFilter?.lowercase().orEmpty()
        if (mentionFilter == null) emptyList()
        else if (q.isEmpty()) mentionMembers
        else mentionMembers.filter {
            it.isAtAll ||
                it.displayName.lowercase().contains(q) ||
                it.userId.lowercase().contains(q)
        }
    }

    val pickImage = androidx.activity.compose.rememberLauncherForActivityResult(
        com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVisualMediaPicker(multiple = true)
    ) { uris ->
        if (uris.isEmpty()) return@rememberLauncherForActivityResult
        scope.launch {
          var failures = 0
          for (uri in uris.distinct()) {
            kotlinx.coroutines.currentCoroutineContext().ensureActive()
            val path = copyUriToCache(context, uri, "img_")
            if (path == null) { failures++; continue }
            var draft: V2TIMMessage? = null
            runCatching {
                LinLanImManager.sendImageMessage(
                    path,
                    receiver,
                    groupId,
                    onPrepared = { draft = it; addOptimisticMedia(it) },
                    onProgress = { updateOptimisticMedia(draft, it) },
                )
            }
                .onSuccess { finishOptimisticMedia(draft, it) }
                .onFailure {
                    failOptimisticMedia(draft)
                    failures++
                }
          }
          if (failures > 0) Toast.makeText(context, "$failures 张图片发送失败，可点击失败消息重试", Toast.LENGTH_LONG).show()
        }
    }
    val pickVideo = androidx.activity.compose.rememberLauncherForActivityResult(
        com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanVisualMediaPicker(video = true, multiple = true)
    ) { uris ->
        if (uris.isNotEmpty() && !com.balabalaim.gcloud.tim.push.appdev.im.LinLanVideoJobs.select(context, uris, receiver, groupId)) {
            Toast.makeText(context, "最多选择9个视频；请等待当前批次结束后再选择", Toast.LENGTH_LONG).show()
        }
    }
    val pickFile = androidx.activity.compose.rememberLauncherForActivityResult(
        androidx.activity.result.contract.ActivityResultContracts.GetContent()
    ) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            val name = uri.lastPathSegment?.substringAfterLast('/') ?: "file"
            val path = copyUriToCache(context, uri, "file_") ?: return@launch
            var draft: V2TIMMessage? = null
            runCatching {
                LinLanImManager.sendFileMessage(
                    path,
                    name,
                    receiver,
                    groupId,
                    onPrepared = { draft = it; addOptimisticMedia(it) },
                    onProgress = { updateOptimisticMedia(draft, it) },
                )
            }
                .onSuccess { finishOptimisticMedia(draft, it) }
                .onFailure {
                    failOptimisticMedia(draft)
                    Toast.makeText(context, it.message ?: "发送文件失败", Toast.LENGTH_SHORT).show()
                }
        }
    }
    val requestPickImage = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberAlbumImageAction {
        com.balabalaim.gcloud.tim.push.appdev.ui.common.launchVisualMediaPicker(context, pickImage)
    }
    val requestPickVideo = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberAlbumVideoAction {
        com.balabalaim.gcloud.tim.push.appdev.ui.common.launchVisualMediaPicker(context, pickVideo)
    }
    val requestPickFile = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberFilePickAction {
        pickFile.launch("*/*")
    }
    val requestVoiceInput = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberCallPermissionAction(
        needCamera = false,
    ) {
        if (voicePermissionIntent == composerIntent) {
            showEmoji = false
            showMorePanel = false
            voiceInputMode = true
            keyboardController?.hide()
            Toast.makeText(context, "按住说话，上滑取消", Toast.LENGTH_SHORT).show()
        }
    }
    val requestAudioCall = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberCallPermissionAction(
        needCamera = false,
    ) {
        LinLanRtcCallManager.startCall(
            activityContext = context,
            peer = RtcCallPeer(peerId, decodedTitle),
            mediaType = LinLanRtcCallMediaType.AUDIO,
        )
    }
    val requestVideoCall = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberCallPermissionAction(
        needCamera = true,
    ) {
        LinLanRtcCallManager.startCall(
            activityContext = context,
            peer = RtcCallPeer(peerId, decodedTitle),
            mediaType = LinLanRtcCallMediaType.VIDEO,
        )
    }

    fun finishVoice(send: Boolean) {
        if (!voiceRecording) return
        voiceRecording = false
        voiceCancelMode = false
        voiceLevel = 0.2f
        if (!send) {
            voiceRecorder.stopAndDiscard()
            Toast.makeText(context, "已取消", Toast.LENGTH_SHORT).show()
            return
        }
        scope.launch {
            runCatching { voiceRecorder.stop() }
                .onSuccess { result ->
                    var draft: V2TIMMessage? = null
                    runCatching {
                        LinLanImManager.sendSoundMessage(
                            result.path,
                            result.durationSec,
                            receiver,
                            groupId,
                            onPrepared = { draft = it; addOptimisticMedia(it) },
                            onProgress = { updateOptimisticMedia(draft, it) },
                        )
                    }.onSuccess { finishOptimisticMedia(draft, it) }
                        .onFailure {
                            failOptimisticMedia(draft)
                            // 发送失败才删本地文件；成功后保留，便于立刻点击播放
                            Toast.makeText(context, it.message ?: "发送语音失败", Toast.LENGTH_SHORT).show()
                        }
                }
                .onFailure {
                    voiceRecorder.stopAndDiscard()
                    Toast.makeText(
                        context,
                        it.message ?: "说话时间太短",
                        Toast.LENGTH_SHORT,
                    ).show()
                }
        }
    }

    DisposableEffect(Unit) {
        onDispose {
            voiceRecorder.stopAndDiscard()
            LinLanVoiceMessagePlayer.stop()
        }
    }

    LaunchedEffect(voiceRecording) {
        if (!voiceRecording) return@LaunchedEffect
        while (isActive && voiceRecording) {
            voiceLevel = voiceRecorder.currentLevel()
            if (voiceRecorder.elapsedSec() >= LinLanVoiceRecorder.MAX_DURATION_SEC) {
                finishVoice(send = !voiceCancelMode)
                break
            }
            delay(80)
        }
    }

    suspend fun applyRevokedMessage(msgId: String) {
        if (msgId.isBlank()) return
        if (quoteTarget?.msgID == msgId) quoteTarget = null
        val expectedScope = historyScope
        val found = try { LinLanImManager.findMessages(listOf(msgId)).firstOrNull() }
            catch (error: kotlinx.coroutines.CancellationException) { throw error }
            catch (_: Exception) { null }
        kotlinx.coroutines.currentCoroutineContext().ensureActive()
        if (latestHistoryScope != expectedScope) return
        val idx = messages.indexOfFirst { it.msgID == msgId }
        val updated = found ?: messages.getOrNull(idx)
        if (updated == null || LinLanImManager.shouldSilentlyHideRevokedMessage(updated)) {
            removedHistoryIDs.add(msgId)
            if (idx >= 0) messages.removeAt(idx)
            return
        }
        removedHistoryIDs.remove(msgId)
        if (idx >= 0) messages[idx] = updated
        else if (LinLanImManager.belongsToConversation(updated, decodedConversationId, isGroup)) messages.add(updated)
    }

    LaunchedEffect(historyScope, imConnectionState) {
        if (chatSession.userId.isBlank()) return@LaunchedEffect
        if (imConnectionState == LinLanImManager.ConnectionState.Connected && LinLanImManager.isReadyFor(chatSession.userId, chatAppId)) loadHistory(latest = true)
        else { historyError = "聊天连接中，连接成功后自动补拉历史"; LinLanAppContainer.authRepository().ensureSessionReady() }
    }

    LaunchedEffect(decodedConversationId, listState) {
        snapshotFlow {
            userDragging && initialBottomReady &&
                listState.firstVisibleItemIndex == 0 &&
                listState.firstVisibleItemScrollOffset == 0
        }
            .distinctUntilChanged()
            .collect { atPulledTop ->
                if (atPulledTop) loadOlderHistory()
            }
    }

    LaunchedEffect(historyScope) {
        LinLanImManager.incomingMessage.collectLatest { msg ->
            if (LinLanImManager.belongsToConversation(msg, decodedConversationId, isGroup)) {
                if (isGroup && LinLanImManager.isGroupTipsMessage(msg)) {
                    applyGroupSendPermission()
                }
                if (LinLanImManager.shouldSilentlyHideRevokedMessage(msg)) return@collectLatest
                if (LinLanImManager.shouldHideRtcSignalingInChat(msg)) return@collectLatest
                val isNew = historyKey(msg) !in removedHistoryIDs && messages.none { historyKey(it) == historyKey(msg) }
                if (isNew) {
                    if (msg.isSelf) requestLatestMessages()
                    else if (!followLatest) pendingNewMessageCount++
                }
                mergeMessages(listOf(msg), false)
                if (!msg.isSelf) LinLanImManager.markConversationRead(decodedConversationId, listOf(msg))
            }
        }
    }

    LaunchedEffect(historyScope) {
        LinLanImManager.messageRevoked.collectLatest { msgId ->
            applyRevokedMessage(msgId)
        }
    }

    fun applyReadReceipts(receipts: List<com.tencent.imsdk.v2.V2TIMMessageReceipt>, legacy: Boolean = false) {
        if (isGroup) return
        receipts.filter { it.userID == peerId }.forEach { receipt ->
            if (legacy) peerReadThrough = maxOf(peerReadThrough, receipt.timestamp)
            else if (receipt.isPeerRead && !receipt.msgID.isNullOrBlank() && receipt.msgID !in peerReadIDs) peerReadIDs.add(receipt.msgID)
        }
    }
    LaunchedEffect(decodedConversationId) {
        LinLanImManager.readReceipts.collect { (receipts, legacy) -> applyReadReceipts(receipts, legacy) }
    }
    LaunchedEffect(decodedConversationId, messages.size) {
        if (!isGroup) applyReadReceipts(LinLanImManager.fetchReadReceipts(messages.toList()))
    }

    LaunchedEffect(historyScope, messages.size, historyLoading, suppressBottomScroll, jumpToLatestRequested, safetyLayoutReady, groupLayoutReady, pinsLayoutReady) {
        if (messages.isNotEmpty() && !suppressBottomScroll && !historyLoading &&
            (initialBottomReady || (safetyLayoutReady && groupLayoutReady && pinsLayoutReady)) &&
            (jumpToLatestRequested || (followLatest && !userDragging))) {
            // Target a trailing anchor, not the top of a potentially tall last bubble.
            androidx.compose.runtime.withFrameNanos { }
            if (jumpToLatestRequested || (followLatest && !userDragging)) {
                listState.scrollToItem(messages.size + 1)
                if (!initialBottomReady) {
                    // Keep the first positioning pass hidden until lazy-item measurement has settled.
                    androidx.compose.runtime.withFrameNanos { }
                    listState.scrollToItem(messages.size + 1)
                    androidx.compose.runtime.withFrameNanos { }
                }
                followLatest = true
                pendingNewMessageCount = 0
                jumpToLatestRequested = false
                initialBottomReady = true
            }
        }
    }
    // Keep the tail visible after image measurement, keyboard or banner height changes.
    LaunchedEffect(decodedConversationId, initialBottomReady) {
        snapshotFlow {
            val layout = listState.layoutInfo
            Triple(layout.viewportEndOffset, layout.visibleItemsInfo.lastOrNull()?.let { it.offset + it.size }, layout.totalItemsCount)
        }.distinctUntilChanged().collect {
            if (initialBottomReady && followLatest && !userDragging && !listState.isScrollInProgress && !historyLoading && !suppressBottomScroll && messages.isNotEmpty() && listState.canScrollForward) {
                listState.scrollToItem(messages.size + 1)
            }
        }
    }

    mediaPreview?.let { preview ->
        MediaPreviewDialog(preview = preview, onDismiss = { mediaPreview = null })
    }

    if (showMentionPicker) {
        Dialog(
            onDismissRequest = { mentionPickerDismissed = true },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            Surface(
                modifier = Modifier
                    .fillMaxWidth(0.92f)
                    .heightIn(max = 420.dp),
                shape = RoundedCornerShape(16.dp),
                color = Color.White,
                shadowElevation = 8.dp,
            ) {
                Column(modifier = Modifier.fillMaxWidth()) {
                    Text(
                        text = "选择要@的人",
                        fontWeight = FontWeight.SemiBold,
                        fontSize = 17.sp,
                        color = MintTitle,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 14.dp),
                    )
                    HorizontalDivider(color = QqDivider, thickness = 0.5.dp)
                    when {
                        mentionLoading && filteredMentions.isEmpty() -> {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(120.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                CircularProgressIndicator(color = QqBlue, modifier = Modifier.size(28.dp))
                            }
                        }
                        filteredMentions.isEmpty() -> {
                            Text(
                                text = "暂无可@的成员",
                                color = QqSecondary,
                                modifier = Modifier.padding(24.dp),
                            )
                        }
                        else -> {
                            LazyColumn(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .heightIn(max = 340.dp),
                            ) {
                                items(
                                    filteredMentions,
                                    key = { if (it.isAtAll) "__at_all__" else it.userId },
                                ) { member ->
                                    Column(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clickable { applyMentionSelection(member) }
                                            .padding(horizontal = 16.dp, vertical = 14.dp),
                                    ) {
                                        Text(
                                            text = if (member.isAtAll) "@全体成员" else "@${member.displayName}",
                                            color = if (member.isAtAll) QqBlue else MintTitle,
                                            fontWeight = if (member.isAtAll) FontWeight.SemiBold else FontWeight.Normal,
                                            fontSize = 16.sp,
                                        )
                                        if (!member.isAtAll) {
                                            Text(
                                                text = member.userId,
                                                color = QqSecondary,
                                                fontSize = 12.sp,
                                                modifier = Modifier.padding(top = 2.dp),
                                            )
                                        } else {
                                            Text(
                                                text = "提醒所有成员",
                                                color = QqSecondary,
                                                fontSize = 12.sp,
                                                modifier = Modifier.padding(top = 2.dp),
                                            )
                                        }
                                    }
                                    HorizontalDivider(
                                        color = QqDivider,
                                        thickness = 0.5.dp,
                                        modifier = Modifier.padding(start = 16.dp),
                                    )
                                }
                            }
                        }
                    }
                    TextButton(
                        onClick = { mentionPickerDismissed = true },
                        modifier = Modifier
                            .align(Alignment.End)
                            .padding(8.dp),
                    ) { Text("取消", color = QqBlue) }
                }
            }
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(QqPageBg)
            .imePadding(),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.White)
                .windowInsetsPadding(WindowInsets.statusBars)
                .height(48.dp)
                .padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = {
                if (multiSelectMode) {
                    multiSelectMode = false
                    selectedMsgIds.clear()
                } else {
                    onBack()
                }
            }) {
                Icon(
                    if (multiSelectMode) Icons.Default.Close else Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = if (multiSelectMode) "取消" else "返回",
                )
            }
            Text(
                if (multiSelectMode) "已选 ${selectedMsgIds.size} 条" else decodedTitle,
                fontWeight = FontWeight.SemiBold,
                fontSize = 17.sp,
                maxLines = 1,
                overflow = TextOverflow.Clip,
                modifier = Modifier
                    .weight(1f)
                    .basicMarquee(iterations = Int.MAX_VALUE)
                    .clickable(enabled = !multiSelectMode) {
                        if (isGroup) onOpenGroupDetail?.invoke(peerId, decodedTitle)
                        else onOpenFriendDetail?.invoke(peerId, decodedTitle)
                    },
            )
            if (!multiSelectMode) {
                IconButton(onClick = { onOpenSearch?.invoke() }) {
                    Icon(Icons.Default.Search, contentDescription = "搜索聊天记录")
                }
                if (!isGroup && !isSecurityAssistant) {
                    IconButton(onClick = { requestAudioCall() }) {
                        Icon(Icons.Default.Call, contentDescription = "语音通话")
                    }
                    IconButton(onClick = { requestVideoCall() }) {
                        Icon(Icons.Default.Videocam, contentDescription = "视频通话")
                    }
                }
                IconButton(onClick = {
                    if (isGroup) onOpenGroupDetail?.invoke(peerId, decodedTitle)
                    else onOpenFriendDetail?.invoke(peerId, decodedTitle)
                }) {
                    Icon(Icons.Default.MoreHoriz, contentDescription = "详情")
                }
            }
        }
        HorizontalDivider(color = QqDivider, thickness = 0.5.dp)

        if (!multiSelectMode) androidx.compose.runtime.key(historyScope) {
            RealnameChatBanner(peerId, isGroup, onOpenRealname, onReady = { safetyLayoutReady = true })
        }

        if (imConnectionState != LinLanImManager.ConnectionState.Connected) {
            val connectionText = when (imConnectionState) {
                LinLanImManager.ConnectionState.Connecting -> "网络较弱，正在重新连接…"
                LinLanImManager.ConnectionState.Disconnected -> "网络连接已断开，消息将在恢复后同步"
                LinLanImManager.ConnectionState.KickedOffline -> "账号已在其他设备登录"
                LinLanImManager.ConnectionState.Connected -> ""
            }
            Text(
                connectionText,
                modifier = Modifier.fillMaxWidth().background(Color(0xFFFFF3CD)).padding(7.dp),
                color = Color(0xFF8A6100),
                fontSize = 12.sp,
                textAlign = TextAlign.Center,
            )
        }

        if (pins.isNotEmpty()) {
            PinnedBar(
                pins = pins,
                canManage = !isGroup || LinLanImManager.canManageGroup(selfGroupRole),
                onLocate = { locatePinnedMessage(it.msgId) },
                onRemove = { pin ->
                    if (isGroup && !groupId.isNullOrBlank()) {
                        scope.launch {
                            runCatching { LinLanImManager.removeGroupPinnedMessage(groupId, pin.msgId) }
                                .onSuccess {
                                    reloadGroupPins()
                                    Toast.makeText(context, "已取消置顶", Toast.LENGTH_SHORT).show()
                                }
                                .onFailure {
                                    Toast.makeText(context, it.message ?: "取消置顶失败", Toast.LENGTH_SHORT).show()
                                }
                        }
                    } else {
                        pinStore.remove(decodedConversationId, pin.msgId)
                        Toast.makeText(context, "已取消置顶", Toast.LENGTH_SHORT).show()
                    }
                },
            )
        }

        Box(Modifier.weight(1f).fillMaxWidth()) {
        LazyColumn(
            state = listState,
            modifier = Modifier
                .alpha(if (messages.isEmpty() || initialBottomReady || historyError != null) 1f else 0f)
                .fillMaxSize()
                .padding(horizontal = 10.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item(key = "history-loader") {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(28.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    when {
                        historyError != null -> TextButton(onClick = { scope.launch { loadHistory(latest = historyPager.cursor == null) } }) { Text(historyError + " · 重试", fontSize = 12.sp) }
                        historyLoading -> CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                            color = QqBlue,
                        )
                        historyEndReached -> Text("没有更多消息", fontSize = 12.sp, color = QqSecondary)
                        else -> TextButton(onClick = { scope.launch { loadOlderHistory() } }) { Text("加载更早消息", fontSize = 12.sp, color = QqSecondary) }
                    }
                }
            }
            items(messages, key = { it.msgID ?: it.hashCode().toString() }) { msg ->
                if (LinLanImManager.shouldSilentlyHideRevokedMessage(msg)) return@items
                if (LinLanImManager.isGroupTipsMessage(msg) && !LinLanImManager.shouldShowGroupTipsInChat(msg)) return@items
                val msgId = msg.msgID.orEmpty()
                val selected = msgId.isNotEmpty() && msgId in selectedMsgIds
                val selfId = V2TIMManager.getInstance().loginUser.orEmpty()
                val selfRole = if (selfGroupRole > 0) {
                    selfGroupRole
                } else {
                    groupMemberRoleMap[selfId] ?: 0
                }
                val senderRole = groupMemberRoleMap[msg.sender.orEmpty()] ?: 0
                val canPinMessage = !isGroup || LinLanImManager.canManageGroup(selfRole)
                val canRevoke = LinLanImManager.canRevokeMessage(msg, selfRole, senderRole)
                if (LinLanImManager.isRevokedMessage(msg)) {
                    RevokedMessageTip(text = LinLanImManager.revokedTipText(msg))
                    return@items
                }
                if (LinLanImManager.isGroupCreateTipMessage(msg)) {
                    RevokedMessageTip(text = LinLanImManager.groupCreateTipText(msg))
                    return@items
                }
                if (LinLanImManager.isFriendGreetingTipMessage(msg)) {
                    RevokedMessageTip(text = LinLanImManager.friendGreetingTipText(msg))
                    return@items
                }
                if (LinLanImManager.isGroupTipsMessage(msg)) {
                    val tip = LinLanImManager.groupTipsTipText(msg)
                    if (tip.isNotBlank()) {
                        RevokedMessageTip(text = tip)
                    }
                    return@items
                }
                Box(modifier = Modifier.fillMaxWidth()) {
                    MessageBubble(
                        msg = msg,
                        transferProgress = mediaSendProgress[System.identityHashCode(msg)],
                        peerTitle = decodedTitle,
                        isGroup = isGroup,
                        highlighted = msg.msgID != null && msg.msgID == highlightedMsgId,
                        peerRead = msg.isPeerRead || msg.msgID in peerReadIDs || (peerReadThrough > 0 && msg.timestamp <= peerReadThrough),
                        onQuoteClick = { locatePinnedMessage(it) },
                        onAvatarLongClick = {
                            if (isGroup && !multiSelectMode && !msg.isSelf && !groupSendBlocked) {
                                val uid = msg.sender.orEmpty()
                                if (uid.isNotBlank()) {
                                    val name = LinLanImManager.displayNameOfMessage(msg).ifBlank { uid }
                                    input += "@$name "
                                    mentionNameToUserId = mentionNameToUserId + (name to uid)
                                    mentionPickerDismissed = true
                                    voiceInputMode = false
                                    showEmoji = false
                                    scope.launch {
                                        kotlinx.coroutines.yield()
                                        inputFocus.requestFocus()
                                        keyboardController?.show()
                                    }
                                }
                            }
                        },
                        multiSelectMode = multiSelectMode,
                        selected = selected,
                        onToggleSelect = {
                            if (msgId.isBlank() || !LinLanImManager.canForwardMessage(msg)) {
                                Toast.makeText(context, "该消息无法转发", Toast.LENGTH_SHORT).show()
                                return@MessageBubble
                            }
                            if (selected) {
                                selectedMsgIds.remove(msgId)
                            } else if (selectedMsgIds.size >= LinLanReliability.MAX_FORWARD_MESSAGES) {
                                Toast.makeText(
                                    context,
                                    "最多选择 ${LinLanReliability.MAX_FORWARD_MESSAGES} 条消息",
                                    Toast.LENGTH_SHORT,
                                ).show()
                            } else {
                                selectedMsgIds.add(msgId)
                            }
                        },
                        onLongPress = {
                            if (!multiSelectMode) menuMsg = msg
                        },
                        onRetrySend = {
                            val key = System.identityHashCode(msg)
                            mediaSendProgress[key] = 0
                            scope.launch {
                                runCatching {
                                    LinLanImManager.retrySendMessage(msg) { mediaSendProgress[key] = it }
                                }.onSuccess { sent ->
                                    mediaSendProgress.remove(key)
                                    val index = messages.indexOfFirst { it === msg || it.msgID == msg.msgID }
                                    if (index >= 0) messages[index] = sent
                                }.onFailure {
                                    mediaSendProgress[key] = -1
                                    Toast.makeText(context, it.message ?: "重试失败", Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        onOpenImage = { mediaPreview = ChatMediaPreview.Image(it) },
                        onOpenVideo = { path -> mediaPreview = ChatMediaPreview.Video(path) },
                        onOpenFile = { path, name -> openPathWithChooser(context, path, name) },
                        onOpenContact = { userId, name -> onOpenFriendDetail?.invoke(userId, name) },
                        onOpenGroup = { gid, gTitle -> (onOpenGroupCard ?: onOpenGroupDetail)?.invoke(gid, gTitle) },
                        onOpenMerger = { onOpenMergerDetail?.invoke(it) },
                        onAvatarClick = {
                            if (multiSelectMode || msg.isSelf) return@MessageBubble
                            val uid = msg.sender?.takeIf { it.isNotBlank() } ?: peerId
                            if (!isGroup) {
                                onOpenFriendDetail?.invoke(uid, decodedTitle)
                                return@MessageBubble
                            }
                            scope.launch {
                                var role = groupMemberRoleMap[uid]
                                if (role == null) {
                                    val map = runCatching {
                                        LinLanImManager.fetchGroupMemberRoleMap(peerId)
                                    }.getOrDefault(emptyMap())
                                    groupMemberRoleMap = map
                                    role = map[uid]
                                }
                                if (LinLanImManager.canViewMemberProfile(groupProfileAccess, role ?: 0)) {
                                    onOpenFriendDetail?.invoke(
                                        uid,
                                        LinLanImManager.displayNameOfMessage(msg).ifBlank { uid },
                                    )
                                } else {
                                    Toast.makeText(
                                        context,
                                        "群已禁止查看成员资料",
                                        Toast.LENGTH_SHORT,
                                    ).show()
                                }
                            }
                        },
                    )
                    // 锚定在气泡侧，避免菜单贴屏幕最左侧
                    Box(
                        modifier = Modifier
                            .align(if (msg.isSelf) Alignment.TopEnd else Alignment.TopStart)
                            .padding(
                                start = if (msg.isSelf) 0.dp else 48.dp,
                                end = if (msg.isSelf) 48.dp else 0.dp,
                                top = if (!msg.isSelf && isGroup) 22.dp else 6.dp,
                            ),
                    ) {
                    DropdownMenu(
                        expanded = !multiSelectMode && menuMsg?.msgID == msg.msgID && menuMsg != null,
                        onDismissRequest = { menuMsg = null },
                    ) {
                        DropdownMenuItem(
                            text = { Text("复制") },
                            onClick = {
                                copyText(context, LinLanImManager.copyableText(msg))
                                Toast.makeText(context, "已复制", Toast.LENGTH_SHORT).show()
                                menuMsg = null
                            },
                        )
                        if (!isSecurityAssistant) DropdownMenuItem(
                            text = { Text("引用") },
                            onClick = {
                                quoteTarget = msg
                                menuMsg = null
                            },
                        )
                        if (canRevoke) {
                            DropdownMenuItem(
                                text = { Text("撤回") },
                                onClick = {
                                    menuMsg = null
                                    scope.launch {
                                        runCatching { LinLanImManager.revokeMessage(msg) }
                                            .onSuccess {
                                                val id = msg.msgID.orEmpty()
                                                if (msg.isSelf) {
                                                    applyRevokedMessage(id)
                                                } else {
                                                    val idx = messages.indexOfFirst { it.msgID == id }
                                                    if (idx >= 0) messages.removeAt(idx)
                                                }
                                            }
                                            .onFailure {
                                                Toast.makeText(
                                                    context,
                                                    it.message ?: "撤回失败",
                                                    Toast.LENGTH_SHORT,
                                                ).show()
                                            }
                                    }
                                },
                            )
                        }
                        if (canPinMessage) DropdownMenuItem(
                            text = {
                                Text(
                                    if (pins.any { it.msgId == msg.msgID.orEmpty() }) {
                                        "取消置顶"
                                    } else {
                                        "置顶"
                                    }
                                )
                            },
                            onClick = {
                                val id = msg.msgID.orEmpty()
                                if (id.isBlank()) {
                                    Toast.makeText(context, "消息暂不可置顶", Toast.LENGTH_SHORT).show()
                                } else {
                                    val pinnedMessage = PinnedMessage(
                                        msgId = id,
                                        summary = LinLanImManager.messagePreview(msg),
                                        sender = LinLanQuoteHelper.senderName(msg),
                                    )
                                    if (isGroup && !groupId.isNullOrBlank()) {
                                        scope.launch {
                                            val isPinned = pins.any { it.msgId == id }
                                            runCatching {
                                                if (isPinned) LinLanImManager.removeGroupPinnedMessage(groupId, id)
                                                else LinLanImManager.addGroupPinnedMessage(groupId, msg)
                                            }.onSuccess { reloadGroupPins() }
                                                .onFailure {
                                                    Toast.makeText(
                                                        context,
                                                        it.message ?: "设置置顶失败",
                                                        Toast.LENGTH_SHORT,
                                                    ).show()
                                                }
                                        }
                                    } else if (pins.any { it.msgId == id }) {
                                        pinStore.remove(decodedConversationId, id)
                                    } else {
                                        pinStore.add(decodedConversationId, pinnedMessage)
                                    }
                                }
                                menuMsg = null
                            },
                        )
                        if (canShowForwardActions) {
                            DropdownMenuItem(
                                text = { Text("转发") },
                                onClick = {
                                    menuMsg = null
                                    if (!LinLanImManager.canForwardMessage(msg)) {
                                        Toast.makeText(context, "该消息无法转发", Toast.LENGTH_SHORT).show()
                                    } else {
                                        onForward?.invoke(listOf(msg), false, "")
                                    }
                                },
                            )
                        }
                        if (canShowForwardActions) {
                            DropdownMenuItem(
                                text = { Text("多选") },
                                onClick = {
                                    menuMsg = null
                                    multiSelectMode = true
                                    showEmoji = false
                                    showMorePanel = false
                                    if (msgId.isNotEmpty() && LinLanImManager.canForwardMessage(msg)) {
                                        selectedMsgIds.clear()
                                        selectedMsgIds.add(msgId)
                                    }
                                },
                            )
                        }
                    }
                    }
                }
            }
            item(key = "chat-bottom-anchor") { Spacer(Modifier.height(1.dp)) }
        }
        if (!initialBottomReady && historyError == null && (historyLoading || messages.isNotEmpty())) {
            CircularProgressIndicator(Modifier.align(Alignment.Center).size(24.dp), color = QqBlue)
        }
        if (pendingNewMessageCount > 0) {
            Surface(
                modifier = Modifier.align(Alignment.BottomEnd).padding(end = 16.dp, bottom = 12.dp),
                shape = RoundedCornerShape(24.dp),
                color = Color.White,
                shadowElevation = 6.dp,
            ) {
                TextButton(onClick = { requestLatestMessages() }) {
                    Text("$pendingNewMessageCount 条新消息  ↓", color = QqBlue,
                        modifier = Modifier.padding(horizontal = 8.dp), fontWeight = FontWeight.Medium)
                }
            }
        }
        }

        if (groupSendBlocked) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(enabled = groupSendCanRetry) { scope.launch { applyGroupSendPermission() } }
                    .background(Color.White)
                    .navigationBarsPadding()
                    .padding(horizontal = 12.dp, vertical = 16.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    groupSendBlockedHint.ifBlank { "已禁言" },
                    color = QqSecondary,
                    textAlign = TextAlign.Center,
                    fontWeight = FontWeight.Medium,
                )
            }
        } else if (multiSelectMode) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color.White)
                    .navigationBarsPadding()
                    .padding(horizontal = 12.dp, vertical = 10.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                TextButton(
                    onClick = {
                        val selectable = messages
                            .filter { LinLanImManager.canForwardMessage(it) }
                            .mapNotNull { it.msgID }
                            .take(LinLanReliability.MAX_FORWARD_MESSAGES)
                        if (selectedMsgIds.size == selectable.size && selectedMsgIds.containsAll(selectable)) {
                            selectedMsgIds.clear()
                        } else {
                            selectedMsgIds.clear()
                            selectedMsgIds.addAll(selectable)
                            if (messages.count { LinLanImManager.canForwardMessage(it) } > LinLanReliability.MAX_FORWARD_MESSAGES) {
                                Toast.makeText(
                                    context,
                                    "已选择最近 ${LinLanReliability.MAX_FORWARD_MESSAGES} 条可转发消息",
                                    Toast.LENGTH_SHORT,
                                ).show()
                            }
                        }
                    },
                ) { Text(if (selectedMsgIds.isEmpty()) "全选" else "重选") }
                TextButton(
                    onClick = { startForward(merge = false) },
                    modifier = Modifier.weight(1f),
                    enabled = selectedMsgIds.isNotEmpty(),
                ) { Text("逐条转发") }
                Button(
                    onClick = { startForward(merge = true) },
                    modifier = Modifier.weight(1f),
                    enabled = selectedMsgIds.isNotEmpty(),
                ) { Text("合并转发") }
            }
        } else if (!isSecurityAssistant) {
        quoteTarget?.let { q ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color(0xFFE8F7FC))
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "引用 ${LinLanQuoteHelper.senderName(q)}: ${LinLanImManager.messagePreview(q)}",
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    color = Color(0xFF666666),
                )
                TextButton(onClick = { quoteTarget = null }) { Text("取消") }
            }
        }

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.White)
                .navigationBarsPadding()
                .padding(horizontal = 10.dp, vertical = 8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (voiceInputMode) {
                    val density = LocalDensity.current
                    val cancelThresholdPx = with(density) { 60.dp.toPx() }
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .heightIn(min = 40.dp, max = 140.dp)
                            .focusRequester(inputFocus)
                            .clip(RoundedCornerShape(8.dp))
                            .background(
                                if (voiceCancelMode) Color(0x33FF3B30) else QqBlue.copy(alpha = 0.16f),
                            )
                            .pointerInput(Unit) {
                                awaitEachGesture {
                                    awaitFirstDown(requireUnconsumed = false)
                                    if (voiceRecording) return@awaitEachGesture
                                    showEmoji = false
                                    showMorePanel = false
                                    voiceCancelMode = false
                                    voiceLevel = 0.2f
                                    val started = runCatching { voiceRecorder.start() }
                                    if (started.isFailure) {
                                        Toast.makeText(
                                            context,
                                            started.exceptionOrNull()?.message ?: "录音启动失败",
                                            Toast.LENGTH_SHORT,
                                        ).show()
                                        return@awaitEachGesture
                                    }
                                    voiceRecording = true
                                    var cancel = false
                                    var startY = 0f
                                    var gotStartY = false
                                    try {
                                        while (true) {
                                            val event = awaitPointerEvent()
                                            val change = event.changes.firstOrNull() ?: break
                                            if (!gotStartY) {
                                                startY = change.position.y
                                                gotStartY = true
                                            }
                                            cancel = (startY - change.position.y) > cancelThresholdPx
                                            voiceCancelMode = cancel
                                            if (!change.pressed) break
                                        }
                                    } finally {
                                        finishVoice(send = !cancel)
                                    }
                                }
                            }
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = when {
                                voiceCancelMode -> "松开 取消"
                                voiceRecording -> "松开 发送"
                                else -> "按住 说话"
                            },
                            color = if (voiceCancelMode) Color(0xFFFF3B30) else QqBlue,
                            textAlign = TextAlign.Center,
                            fontWeight = FontWeight.Medium,
                        )
                    }
                } else {
                    BasicTextField(
                        value = input,
                        onValueChange = { next ->
                            composerIntent++
                            input = LinLanEmojiCatalog.coerceAfterEdit(input, next)
                        },
                        modifier = Modifier
                            .weight(1f)
                            .focusRequester(inputFocus)
                            .heightIn(min = 40.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .background(QqSearchBg)
                            .padding(horizontal = 12.dp, vertical = 10.dp),
                        singleLine = false,
                        minLines = 1,
                        maxLines = 5,
                        keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Default),
                        visualTransformation = LinLanEmojiCatalog.inputVisualTransformation,
                        decorationBox = { inner ->
                            if (input.isEmpty()) {
                                Text(
                                    if (isGroup) "输入消息，@可提人" else "输入消息",
                                    color = QqSecondary,
                                )
                            }
                            inner()
                        },
                    )
                    Spacer(Modifier.width(8.dp))
                    Button(
                        onClick = {
                            val text = input.trim()
                            val now = SystemClock.elapsedRealtime()
                            if (text.isEmpty() || now - lastSendAtMs < 400L) return@Button
                            lastSendAtMs = now
                            val atIds = resolveAtUserIds(text)
                            // 点击即清空，避免连点重复发出同一条
                            input = ""
                            val cloud = quoteTarget?.let {
                                LinLanQuoteHelper.embedQuoteCloudCustomData(it, LinLanQuoteHelper.senderName(it))
                            }
                            quoteTarget = null
                            showEmoji = false
                            scope.launch {
                                try {
                                    val msg = LinLanImManager.sendTextMessage(
                                        text = text,
                                        receiver = receiver,
                                        groupId = groupId,
                                        cloudCustomData = cloud,
                                        atUserIds = atIds,
                                    )
                                    requestLatestMessages()
                                    mergeMessages(listOf(msg), false)
                                } catch (err: Exception) {
                                    Toast.makeText(
                                        context,
                                        err.message ?: "发送失败",
                                        Toast.LENGTH_SHORT,
                                    ).show()
                                }
                            }
                        },
                        enabled = input.isNotBlank(),
                        colors = ButtonDefaults.buttonColors(containerColor = QqBlue),
                        shape = RoundedCornerShape(8.dp),
                    ) { Text("发送") }
                }
            }
            Spacer(Modifier.height(8.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly,
            ) {
                if (voiceInputMode) {
                    ChatToolIcon(Icons.Default.Keyboard, "键盘") { requestTextInput() }
                } else {
                    ChatToolIcon(Icons.Default.Mic, "语音") {
                        composerIntent++
                        voicePermissionIntent = composerIntent
                        requestVoiceInput()
                    }
                }
                ChatToolIcon(Icons.Default.Image, "图片") {
                    showEmoji = false; showMorePanel = false; requestPickImage()
                }
                ChatToolIcon(Icons.Default.Videocam, "视频") {
                    showEmoji = false; showMorePanel = false; requestPickVideo()
                }
                ChatToolIcon(Icons.AutoMirrored.Filled.InsertDriveFile, "文件") {
                    showEmoji = false; showMorePanel = false; requestPickFile()
                }
                ChatToolIcon(Icons.Default.EmojiEmotions, "表情") {
                    if (showEmoji) requestTextInput()
                    else {
                        composerIntent++
                        showMorePanel = false
                        voiceInputMode = false
                        showEmoji = true
                        keyboardController?.hide()
                    }
                }
                ChatToolIcon(Icons.Default.AddCircleOutline, "更多") {
                    showEmoji = false; showMorePanel = !showMorePanel
                }
            }
            if (showRunPicker && !groupSendBlocked) ChatRunPicker(peerId, decodedTitle, isGroup) { showRunPicker = false }
            if (showMorePanel) {
                Row(Modifier.fillMaxWidth().padding(top = 14.dp, bottom = 8.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    ChatActionTile("运动路线", "选记录分享", Icons.Outlined.DirectionsRun, Modifier.weight(1f)) {
                        if (!groupSendBlocked) { showMorePanel = false; showRunPicker = true }
                    }
                    ChatActionTile("个人名片", "推荐给朋友", Icons.Default.Person, Modifier.weight(1f)) {
                        showMorePanel = false; onPickContactCard?.invoke()
                    }
                    ChatActionTile(if (isGroup) "群资料" else "好友资料", "查看详情", Icons.Default.Info, Modifier.weight(1f)) {
                        showMorePanel = false
                        if (isGroup) onOpenGroupDetail?.invoke(peerId, decodedTitle) else onOpenFriendDetail?.invoke(peerId, decodedTitle)
                    }
                }
            }
            if (showEmoji) {
                LazyVerticalGrid(
                    columns = GridCells.Fixed(8),
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(180.dp)
                        .padding(top = 8.dp)
                        .background(QqSearchBg, RoundedCornerShape(12.dp))
                        .padding(6.dp),
                ) {
                    gridItems(LinLanEmojiCatalog.items) { item ->
                        AsyncImage(
                            model = item.assetPath,
                            contentDescription = item.tag,
                            modifier = Modifier
                                .padding(2.dp)
                                .size(32.dp)
                                .clickable { input += item.tag },
                        )
                    }
                }
            }
        }
        } // end composer
        if (isSecurityAssistant && !multiSelectMode) {
            Spacer(Modifier.navigationBarsPadding())
        }
    }
    LinLanVoiceRecordOverlay(
        visible = voiceRecording,
        cancelMode = voiceCancelMode,
        level = voiceLevel,
    )
    }
}

@Composable
private fun ChatToolIcon(icon: ImageVector, label: String, onClick: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        IconButton(onClick = onClick, modifier = Modifier.size(40.dp).background(LinLanCanvas, RoundedCornerShape(13.dp))) {
            Icon(icon, contentDescription = label, tint = LinLanInk, modifier = Modifier.size(22.dp))
        }
        Text(label, fontSize = 10.sp, color = LinLanInkSecondary, modifier = Modifier.padding(top = 4.dp))
    }
}

@Composable
private fun ChatActionTile(title: String, subtitle: String, icon: ImageVector, modifier: Modifier, onClick: () -> Unit) {
    Column(modifier.clip(RoundedCornerShape(16.dp)).background(LinLanCanvas).clickable(onClick = onClick).padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Icon(icon, null, tint = LinLanAccent, modifier = Modifier.size(25.dp))
        Text(title, fontWeight = FontWeight.SemiBold, fontSize = 13.sp, color = LinLanInk)
        Text(subtitle + " ›", fontSize = 10.sp, color = LinLanInkSecondary)
    }
}

@Composable
private fun PinnedBar(
    pins: List<PinnedMessage>,
    canManage: Boolean,
    onLocate: (PinnedMessage) -> Unit,
    onRemove: (PinnedMessage) -> Unit,
) {
    var index by remember { mutableIntStateOf(0) }
    var menuExpanded by remember { mutableStateOf(false) }
    LaunchedEffect(pins.size) {
        if (index >= pins.size) index = 0
    }
    if (pins.isEmpty()) return
    val current = pins[index.coerceIn(0, pins.lastIndex)]
    val countLabel = if (pins.size > 1) "${index + 1}/${pins.size}" else "置顶"
    val digest = current.sender.trim().let { name ->
        if (name.isEmpty()) current.summary else "$name: ${current.summary}"
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(Color(0xFFF7FBFF))
            .clickable {
                onLocate(current)
                if (pins.size > 1) {
                    index = (index + 1) % pins.size
                }
            }
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Default.PushPin,
            contentDescription = null,
            tint = QqBlue,
            modifier = Modifier.size(16.dp),
        )
        Spacer(modifier = Modifier.width(8.dp))
        Text(
            countLabel,
            color = QqBlue,
            fontSize = 12.sp,
            fontWeight = FontWeight.SemiBold,
        )
        Spacer(modifier = Modifier.width(8.dp))
        Text(
            digest,
            modifier = Modifier.weight(1f),
            color = Color(0xFF111111),
            fontSize = 13.sp,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        if (canManage) Box {
            IconButton(
                onClick = { menuExpanded = true },
                modifier = Modifier.size(32.dp),
            ) {
                Icon(
                    Icons.Default.KeyboardArrowDown,
                    contentDescription = "管理置顶",
                    tint = Color(0xFF8E8E93),
                )
            }
            DropdownMenu(
                expanded = menuExpanded,
                onDismissRequest = { menuExpanded = false },
            ) {
                pins.forEach { pin ->
                    val label = pin.sender.trim().let { name ->
                        if (name.isEmpty()) pin.summary else "$name: ${pin.summary}"
                    }
                    DropdownMenuItem(
                        text = {
                            Text(
                                label,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        },
                        trailingIcon = {
                            Text("删除", color = Color(0xFFFF3B30), fontSize = 13.sp)
                        },
                        onClick = {
                            menuExpanded = false
                            onRemove(pin)
                            if (index >= pins.size - 1) index = 0
                        },
                    )
                }
            }
        }
    }
    HorizontalDivider(color = QqDivider, thickness = 0.5.dp)
}

@Composable
private fun RevokedMessageTip(text: String) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text,
            color = Color(0xFF8E8E93),
            fontSize = 12.sp,
            modifier = Modifier
                .background(Color(0xFFEDEDED), RoundedCornerShape(4.dp))
                .padding(horizontal = 10.dp, vertical = 4.dp),
        )
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun MessageBubble(
    msg: V2TIMMessage,
    transferProgress: Int? = null,
    peerTitle: String,
    isGroup: Boolean,
    highlighted: Boolean = false,
    multiSelectMode: Boolean = false,
    selected: Boolean = false,
    onToggleSelect: () -> Unit = {},
    onLongPress: () -> Unit,
    onRetrySend: () -> Unit = {},
    onOpenImage: (String) -> Unit,
    onOpenVideo: (String) -> Unit,
    onOpenFile: (path: String, name: String) -> Unit,
    onOpenContact: (userId: String, name: String) -> Unit,
    onOpenGroup: (groupId: String, title: String) -> Unit,
    onOpenMerger: (V2TIMMessage) -> Unit = {},
    onAvatarClick: () -> Unit,
    onAvatarLongClick: () -> Unit = {},
    onQuoteClick: (String) -> Unit = {},
    peerRead: Boolean = false,
) {
    val isSelf = msg.isSelf
    val bubbleColor = if (isSelf) QqBlue else Color.White
    val textColor = if (isSelf) Color.White else Color(0xFF111111)
    val quote = remember(msg.msgID, msg.cloudCustomData) { LinLanQuoteHelper.resolveQuote(msg) }
    val sportCard = remember(msg.msgID, msg.cloudCustomData, msg.customElem?.data?.contentHashCode()) {
        com.balabalaim.gcloud.tim.push.appdev.run.RunShare.decode(
            if (msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_CUSTOM) msg.customElem?.data?.toString(Charsets.UTF_8) else msg.cloudCustomData)
    }
    var showSportDetail by remember(msg.msgID) { mutableStateOf(false) }
    if (showSportDetail && sportCard != null) RunShareDetail(sportCard) { showSportDetail = false }

    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val peerNick = remember(msg.msgID, msg.nickName, msg.nameCard, msg.friendRemark, peerTitle) {
        LinLanImManager.displayNameOfMessage(msg).ifBlank { peerTitle }
    }
    val avatarUrl = if (isSelf) null else msg.faceUrl
    val avatarName = if (isSelf) "我" else peerNick
    val timeText = remember(msg.timestamp) {
        SimpleDateFormat("HH:mm", Locale.getDefault()).format(Date(msg.timestamp * 1000))
    }
    var videoCover by remember(msg.msgID) { mutableStateOf<String?>(null) }
    var loadingMedia by remember { mutableStateOf(false) }
    var downloadProgress by remember(msg.msgID) { mutableStateOf<Int?>(null) }
    var downloaded by remember(msg.msgID) { mutableStateOf(LinLanImManager.isMediaCached(context, msg)) }
    val mergerCard = remember(msg.msgID, msg.elemType) { LinLanImManager.parseMergerCard(msg) }

    LaunchedEffect(msg.msgID, msg.elemType) {
        if (msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO) {
            videoCover = LinLanImManager.resolveVideoSnapshot(msg)
        }
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(if (highlighted || selected) Color(0xFFFFF3BF) else Color.Transparent)
            .clickable(enabled = multiSelectMode, onClick = onToggleSelect)
            .padding(vertical = 2.dp, horizontal = 2.dp),
        horizontalArrangement = if (isSelf) Arrangement.End else Arrangement.Start,
        verticalAlignment = Alignment.Top,
    ) {
        if (multiSelectMode) {
            Box(
                modifier = Modifier
                    .padding(top = 10.dp, end = 8.dp)
                    .size(22.dp)
                    .clip(CircleShape)
                    .background(if (selected) QqBlue else Color.Transparent)
                    .border(
                        1.5.dp,
                        if (selected) QqBlue else Color(0xFFC7C7CC),
                        CircleShape,
                    ),
                contentAlignment = Alignment.Center,
            ) {
                if (selected) {
                    Icon(
                        Icons.Default.Check,
                        contentDescription = null,
                        tint = Color.White,
                        modifier = Modifier.size(14.dp),
                    )
                }
            }
        }
        if (!isSelf) {
            RoundAvatar(url = avatarUrl, name = avatarName, size = 40.dp, onClick = onAvatarClick, onLongClick = onAvatarLongClick)
            Spacer(Modifier.width(8.dp))
        }
        Column(
            horizontalAlignment = if (isSelf) Alignment.End else Alignment.Start,
            modifier = Modifier.widthIn(max = 260.dp),
        ) {
            if (!isSelf) {
                Text(
                    peerNick,
                    color = Color(0xFF8E8E93),
                    fontSize = 12.sp,
                    modifier = Modifier.padding(bottom = 3.dp, start = 2.dp),
                )
            }
            val mediaType = msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE ||
                msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO ||
                msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_FILE ||
                msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_MERGER
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(10.dp))
                    .background(
                        if (mediaType || mergerCard != null) Color.Transparent else bubbleColor,
                    )
                    .combinedClickable(
                        onClick = {
                            if (multiSelectMode) {
                                onToggleSelect()
                                return@combinedClickable
                            }
                            if (transferProgress != null && transferProgress < 0) {
                                onRetrySend()
                                return@combinedClickable
                            }
                            when (msg.elemType) {
                                V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> {
                                    if (loadingMedia) return@combinedClickable
                                    scope.launch {
                                        loadingMedia = true
                                        downloadProgress = 0
                                        val path = LinLanImManager.resolveImageLocal(context, msg) {
                                            downloadProgress = it
                                        }
                                        loadingMedia = false
                                        if (path.isNullOrBlank()) {
                                            downloadProgress = null
                                            Toast.makeText(context, "图片暂不可查看", Toast.LENGTH_SHORT).show()
                                        } else {
                                            downloaded = !path.startsWith("http")
                                            downloadProgress = if (downloaded) 100 else null
                                            onOpenImage(path)
                                        }
                                    }
                                }
                                V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> {
                                    if (loadingMedia) return@combinedClickable
                                    scope.launch {
                                        loadingMedia = true
                                        downloadProgress = 0
                                        val path = LinLanImManager.resolveVideoPlayable(context, msg) {
                                            downloadProgress = it
                                        }
                                        loadingMedia = false
                                        if (path.isNullOrBlank()) {
                                            Toast.makeText(context, "视频暂不可播放", Toast.LENGTH_SHORT).show()
                                        } else {
                                            downloaded = !path.startsWith("http")
                                            downloadProgress = if (downloaded) 100 else null
                                            onOpenVideo(path)
                                        }
                                    }
                                }
                                V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                                    if (loadingMedia) return@combinedClickable
                                    scope.launch {
                                        loadingMedia = true
                                        downloadProgress = 0
                                        val path = LinLanImManager.resolveFileLocal(context, msg) {
                                            downloadProgress = it
                                        }
                                        loadingMedia = false
                                        val fileName = msg.fileElem?.fileName ?: "文件"
                                        if (path.isNullOrBlank()) {
                                            Toast.makeText(context, "文件暂不可打开", Toast.LENGTH_SHORT).show()
                                        } else if (LinLanImManager.isImageFileName(fileName)) {
                                            downloaded = !path.startsWith("http")
                                            downloadProgress = if (downloaded) 100 else null
                                            onOpenImage(path)
                                        } else {
                                            downloaded = !path.startsWith("http")
                                            downloadProgress = if (downloaded) 100 else null
                                            onOpenFile(path, fileName)
                                        }
                                    }
                                }
                                V2TIMMessage.V2TIM_ELEM_TYPE_SOUND -> {
                                    if (loadingMedia) return@combinedClickable
                                    val msgId = msg.msgID.orEmpty()
                                    if (msgId.isBlank()) {
                                        Toast.makeText(context, "语音暂不可播放", Toast.LENGTH_SHORT).show()
                                        return@combinedClickable
                                    }
                                    if (LinLanVoiceMessagePlayer.isPlaying(msgId)) {
                                        LinLanVoiceMessagePlayer.stop()
                                        return@combinedClickable
                                    }
                                    scope.launch {
                                        loadingMedia = true
                                        val path = runCatching {
                                            LinLanImManager.resolveSoundLocal(context, msg) {
                                                downloadProgress = it
                                            }
                                        }.getOrNull()
                                        loadingMedia = false
                                        if (path.isNullOrBlank()) {
                                            Toast.makeText(context, "语音暂不可播放", Toast.LENGTH_SHORT).show()
                                        } else {
                                            downloaded = !path.startsWith("http")
                                            downloadProgress = if (downloaded) 100 else null
                                            runCatching {
                                                LinLanVoiceMessagePlayer.toggle(context, msgId, path)
                                            }.onFailure {
                                                Toast.makeText(
                                                    context,
                                                    it.message ?: "语音播放失败",
                                                    Toast.LENGTH_SHORT,
                                                ).show()
                                            }
                                        }
                                    }
                                }
                                V2TIMMessage.V2TIM_ELEM_TYPE_MERGER -> onOpenMerger(msg)
                                else -> Unit
                            }
                        },
                        onLongClick = onLongPress,
                    )
                    .then(
                        if (mediaType || mergerCard != null) {
                            Modifier
                        } else {
                            Modifier.padding(horizontal = 12.dp, vertical = 8.dp)
                        },
                    ),
            ) {
                Column {
                    quote?.let {
                        Text(
                            it.displayLine,
                            color = textColor.copy(alpha = 0.75f),
                            style = MaterialTheme.typography.bodySmall,
                            modifier = Modifier
                                .padding(bottom = 4.dp)
                                .clickable(enabled = !multiSelectMode) { onQuoteClick(it.quotedMsgId) }
                                .background(Color.Black.copy(alpha = 0.08f), RoundedCornerShape(6.dp))
                                .padding(6.dp),
                        )
                    }
                    when (msg.elemType) {
                        V2TIMMessage.V2TIM_ELEM_TYPE_MERGER -> {
                            val card = mergerCard
                            if (card != null) {
                                MergerMessageCard(
                                    title = card.title,
                                    abstracts = card.abstractList,
                                    onClick = null,
                                )
                            } else {
                                Text("[聊天记录]", color = textColor)
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> {
                            val url = LinLanImManager.imageUrlOf(msg) ?: LinLanImManager.imageUrlOf(msg, preferLarge = true)
                            if (!url.isNullOrBlank()) {
                                Box {
                                    AsyncImage(
                                        model = url,
                                        contentDescription = "图片",
                                        contentScale = ContentScale.Crop,
                                        modifier = Modifier
                                            .widthIn(min = 120.dp, max = 220.dp)
                                            .heightIn(min = 120.dp, max = 240.dp)
                                            .clip(RoundedCornerShape(8.dp)),
                                    )
                                    Text(
                                        timeText,
                                        color = Color.White,
                                        fontSize = 11.sp,
                                        modifier = Modifier
                                            .align(Alignment.BottomEnd)
                                            .padding(6.dp)
                                            .background(Color.Black.copy(0.45f), RoundedCornerShape(4.dp))
                                            .padding(horizontal = 5.dp, vertical = 2.dp),
                                    )
                                }
                            } else {
                                Text("[图片]", color = textColor, modifier = Modifier.padding(8.dp))
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> {
                            Box(
                                modifier = Modifier
                                    .width(180.dp)
                                    .height(240.dp)
                                    .clip(RoundedCornerShape(8.dp))
                                    .background(Color(0xFF333333)),
                                contentAlignment = Alignment.Center,
                            ) {
                                if (!videoCover.isNullOrBlank()) {
                                    AsyncImage(
                                        model = videoCover,
                                        contentDescription = "视频封面",
                                        contentScale = ContentScale.Crop,
                                        modifier = Modifier.fillMaxSize(),
                                    )
                                }
                                Box(
                                    modifier = Modifier
                                        .size(48.dp)
                                        .clip(CircleShape)
                                        .background(Color.Black.copy(alpha = 0.45f)),
                                    contentAlignment = Alignment.Center,
                                ) {
                                    Icon(
                                        Icons.Default.PlayArrow,
                                        contentDescription = "播放",
                                        tint = Color.White,
                                        modifier = Modifier.size(32.dp),
                                    )
                                }
                                Text(
                                    "${msg.videoElem?.duration ?: 0}\"",
                                    color = Color.White,
                                    fontSize = 12.sp,
                                    modifier = Modifier
                                        .align(Alignment.BottomStart)
                                        .padding(8.dp),
                                )
                                Text(
                                    timeText,
                                    color = Color.White,
                                    fontSize = 11.sp,
                                    modifier = Modifier
                                        .align(Alignment.BottomEnd)
                                        .padding(6.dp)
                                        .background(Color.Black.copy(0.45f), RoundedCornerShape(4.dp))
                                        .padding(horizontal = 5.dp, vertical = 2.dp),
                                )
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_SOUND -> {
                            val duration = msg.soundElem?.duration ?: 1
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Icon(
                                    if (loadingMedia) Icons.Default.PlayArrow else Icons.Default.Mic,
                                    null,
                                    tint = textColor,
                                    modifier = Modifier.size(18.dp),
                                )
                                Spacer(Modifier.width(6.dp))
                                Text(
                                    if (loadingMedia) "加载中…" else "${duration}\"",
                                    color = textColor,
                                )
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                            val fileName = msg.fileElem?.fileName ?: "文件"
                            Row(
                                modifier = Modifier
                                    .widthIn(min = 160.dp)
                                    .background(bubbleColor, RoundedCornerShape(10.dp))
                                    .padding(horizontal = 12.dp, vertical = 8.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Icon(
                                    Icons.AutoMirrored.Filled.InsertDriveFile,
                                    null,
                                    tint = if (isSelf) Color.White else QqBlue,
                                    modifier = Modifier.size(36.dp),
                                )
                                Spacer(Modifier.width(8.dp))
                                Column {
                                    Text(fileName, color = textColor, maxLines = 2, overflow = TextOverflow.Ellipsis)
                                    Text(
                                        when {
                                            loadingMedia && downloadProgress != null -> "下载中 ${downloadProgress}%"
                                            downloaded -> "已下载"
                                            else -> "点击下载"
                                        },
                                        color = textColor.copy(0.7f),
                                        fontSize = 12.sp,
                                    )
                                }
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_CUSTOM -> {
                            val card = LinLanImManager.parseCustomCard(msg)
                            if (sportCard != null) RunShareCard(sportCard) { if (!multiSelectMode) showSportDetail = true }
                            else if (card != null) {
                                Column(
                                    modifier = Modifier
                                        .width(200.dp)
                                        .clickable {
                                            if (card.isGroup) {
                                                onOpenGroup(card.groupId, card.groupName)
                                            } else if (card.userId.isNotBlank()) {
                                                onOpenContact(card.userId, card.nickname)
                                            }
                                        },
                                ) {
                                    Text(
                                        if (card.isGroup) "群名片" else "个人名片",
                                        color = textColor.copy(0.7f),
                                        fontSize = 12.sp,
                                    )
                                    Spacer(Modifier.height(4.dp))
                                    Text(
                                        if (card.isGroup) card.groupName else card.nickname,
                                        color = textColor,
                                        fontWeight = FontWeight.Medium,
                                        fontSize = 16.sp,
                                    )
                                    HorizontalDivider(
                                        color = textColor.copy(0.2f),
                                        thickness = 0.5.dp,
                                        modifier = Modifier.padding(vertical = 6.dp),
                                    )
                                    Text("点击查看", color = textColor.copy(0.65f), fontSize = 12.sp)
                                }
                            } else {
                                val rtc = LinLanRtcSignalingPayload.parseFromMessage(msg)
                                if (rtc != null) {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        Icon(
                                            if (rtc.mediaType == LinLanRtcCallMediaType.VIDEO) {
                                                Icons.Default.Videocam
                                            } else {
                                                Icons.Default.Call
                                            },
                                            contentDescription = null,
                                            tint = textColor,
                                            modifier = Modifier.size(18.dp),
                                        )
                                        Spacer(Modifier.width(8.dp))
                                        Text(rtc.displayText(msg.isSelf), color = textColor)
                                    }
                                } else {
                                    Text(LinLanImManager.messagePreview(msg), color = textColor)
                                }
                            }
                        }
                        V2TIMMessage.V2TIM_ELEM_TYPE_TEXT -> {
                            if (sportCard != null) RunShareCard(sportCard) { if (!multiSelectMode) showSportDetail = true }
                            else if (parseLargeAttachment(msg.textElem?.text.orEmpty()) != null) LargeAttachmentCard(parseLargeAttachment(msg.textElem?.text.orEmpty())!!)
                            else EmojiText(
                                text = msg.textElem?.text.orEmpty(),
                                color = textColor,
                                mentionColor = if (isSelf) MentionSelfHighlight else MentionBlue,
                            )
                        }
                        else -> Text(LinLanImManager.messagePreview(msg), color = textColor)
                    }
                }
                val visibleProgress = transferProgress?.takeIf { it < 100 } ?: downloadProgress?.takeIf { it in 0..99 }
                if (visibleProgress != null) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.Center)
                            .size(54.dp)
                            .clip(CircleShape)
                            .background(Color.Black.copy(alpha = 0.58f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (visibleProgress < 0) {
                            Text("!", color = Color.White, fontSize = 24.sp, fontWeight = FontWeight.Bold)
                        } else {
                            CircularProgressIndicator(
                                progress = { visibleProgress / 100f },
                                modifier = Modifier.size(42.dp),
                                color = Color.White,
                                trackColor = Color.White.copy(alpha = 0.25f),
                                strokeWidth = 3.dp,
                            )
                            Text("$visibleProgress%", color = Color.White, fontSize = 10.sp)
                        }
                    }
                }
            }
            if (!mediaType || (isSelf && !isGroup)) {
                Text(
                    if (isSelf && !isGroup && msg.status == V2TIMMessage.V2TIM_MSG_STATUS_SEND_SUCC) {
                        "$timeText · ${if (peerRead) "已读" else "未读"}"
                    } else timeText,
                    color = Color(0xFF8E8E93),
                    fontSize = 11.sp,
                    modifier = Modifier.padding(top = 3.dp, start = 2.dp, end = 2.dp),
                )
            }
        }
        if (isSelf) {
            Spacer(Modifier.width(8.dp))
            RoundAvatar(url = null, name = "我", size = 40.dp)
        }
    }
}

@Composable
private fun MediaPreviewDialog(preview: ChatMediaPreview, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var saving by remember { mutableStateOf(false) }

    fun saveNow() {
        if (saving) return
        saving = true
        scope.launch {
            runCatching { savePreviewToGallery(context, preview) }
                .onSuccess { Toast.makeText(context, "已保存到手机相册", Toast.LENGTH_SHORT).show() }
                .onFailure { Toast.makeText(context, it.message ?: "保存失败", Toast.LENGTH_SHORT).show() }
            saving = false
        }
    }

    val explainLegacyStorage = com.balabalaim.gcloud.tim.push.appdev.ui.common.rememberExplainedPermissionAction(
        listOf(Manifest.permission.WRITE_EXTERNAL_STORAGE),
        com.balabalaim.gcloud.tim.push.appdev.ui.common.PermissionCopy.SAVE_TO_ALBUM,
        features = listOf(com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermission.SAVE_MEDIA),
        onGranted = ::saveNow,
    )

    fun requestSave() {
        if (
            Build.VERSION.SDK_INT <= 28 &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.WRITE_EXTERNAL_STORAGE) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            explainLegacyStorage()
        } else {
            saveNow()
        }
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black)
                .clickable(onClick = onDismiss),
        ) {
            when (preview) {
                is ChatMediaPreview.Image -> {
                    AsyncImage(
                        model = preview.model,
                        contentDescription = "图片预览",
                        contentScale = ContentScale.Fit,
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(8.dp),
                    )
                }
                is ChatMediaPreview.Video -> {
                    AndroidView(
                        factory = { ctx ->
                            VideoView(ctx).apply {
                                if (preview.pathOrUrl.startsWith("http")) {
                                    setVideoURI(Uri.parse(preview.pathOrUrl))
                                } else {
                                    setVideoPath(preview.pathOrUrl)
                                }
                                setOnPreparedListener { it.isLooping = false; start() }
                            }
                        },
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(8.dp)
                            .align(Alignment.Center),
                    )
                }
            }
            Row(
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .windowInsetsPadding(WindowInsets.statusBars)
                    .padding(8.dp),
            ) {
                TextButton(onClick = { requestSave() }, enabled = !saving,
                    modifier = Modifier.background(Color.Black.copy(alpha = .65f), RoundedCornerShape(20.dp))) {
                    if (saving) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(24.dp),
                            strokeWidth = 2.dp,
                            color = Color.White,
                        )
                    } else {
                        Icon(Icons.Default.Download, contentDescription = null, tint = Color.White)
                        Spacer(Modifier.width(6.dp))
                        Text("保存到相册", color = Color.White)
                    }
                }
                IconButton(onClick = onDismiss) {
                    Icon(Icons.Default.Close, contentDescription = "关闭", tint = Color.White)
                }
            }
        }
    }
}

private suspend fun savePreviewToGallery(context: Context, preview: ChatMediaPreview): String =
    withContext(Dispatchers.IO) {
        val isVideo = preview is ChatMediaPreview.Video
        val source = when (preview) {
            is ChatMediaPreview.Image -> preview.model
            is ChatMediaPreview.Video -> preview.pathOrUrl
        }
        val extension = source.substringBefore('?').substringAfterLast('.', "")
            .takeIf { it.length in 2..5 }
            ?: if (isVideo) "mp4" else "jpg"
        val displayName = "ruizhihe_${System.currentTimeMillis()}.$extension"
        val mime = if (isVideo) "video/$extension" else "image/$extension"

        fun openSource(): java.io.InputStream {
            if (!source.startsWith("http://") && !source.startsWith("https://")) {
                return File(source).inputStream()
            }
            val connection = URL(source).openConnection() as HttpURLConnection
            connection.connectTimeout = 15_000
            connection.readTimeout = 60_000
            connection.instanceFollowRedirects = true
            connection.connect()
            if (connection.responseCode !in 200..299) {
                connection.disconnect()
                throw IllegalStateException("下载失败(${connection.responseCode})")
            }
            return connection.inputStream
        }

        if (Build.VERSION.SDK_INT >= 29) {
            val values = ContentValues().apply {
                put(MediaStore.MediaColumns.DISPLAY_NAME, displayName)
                put(MediaStore.MediaColumns.MIME_TYPE, mime)
                put(
                    MediaStore.MediaColumns.RELATIVE_PATH,
                    (if (isVideo) Environment.DIRECTORY_MOVIES else Environment.DIRECTORY_PICTURES) + "/睿之禾",
                )
                put(MediaStore.MediaColumns.IS_PENDING, 1)
            }
            val collection = if (isVideo) {
                MediaStore.Video.Media.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
            } else {
                MediaStore.Images.Media.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
            }
            val uri = context.contentResolver.insert(collection, values)
                ?: throw IllegalStateException("无法创建相册文件")
            try {
                openSource().use { input ->
                    context.contentResolver.openOutputStream(uri)?.use { output -> input.copyTo(output) }
                        ?: throw IllegalStateException("无法写入相册")
                }
                context.contentResolver.update(
                    uri,
                    ContentValues().apply { put(MediaStore.MediaColumns.IS_PENDING, 0) },
                    null,
                    null,
                )
            } catch (error: Throwable) {
                context.contentResolver.delete(uri, null, null)
                throw error
            }
            uri.toString()
        } else {
            @Suppress("DEPRECATION")
            val parent = Environment.getExternalStoragePublicDirectory(
                if (isVideo) Environment.DIRECTORY_MOVIES else Environment.DIRECTORY_PICTURES,
            )
            val folder = File(parent, "睿之禾").also { it.mkdirs() }
            val target = File(folder, displayName)
            openSource().use { input -> target.outputStream().use { output -> input.copyTo(output) } }
            android.media.MediaScannerConnection.scanFile(context, arrayOf(target.absolutePath), arrayOf(mime), null)
            target.absolutePath
        }
    }

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun EmojiText(
    text: String,
    color: Color,
    mentionColor: Color = MentionBlue,
) {
    val parts = remember(text) { LinLanEmojiCatalog.splitWithEmojis(text) }
    FlowRow {
        parts.forEach { (isEmoji, part) ->
            if (isEmoji) {
                val asset = LinLanEmojiCatalog.assetForTag(part)
                if (asset != null) {
                    AsyncImage(
                        model = asset,
                        contentDescription = part,
                        modifier = Modifier.size(22.dp),
                    )
                } else {
                    Text(part, color = color)
                }
            } else {
                Text(
                    text = remember(part, color, mentionColor) {
                        buildMentionAnnotated(part, color, mentionColor)
                    },
                )
            }
        }
    }
}

/** 把纯文本段里的 @全体成员 / @某人 标成区别色（微信风格）。 */
private fun buildMentionAnnotated(
    text: String,
    baseColor: Color,
    mentionColor: Color,
): androidx.compose.ui.text.AnnotatedString = buildAnnotatedString {
    var i = 0
    while (i < text.length) {
        val at = text.indexOf('@', i)
        if (at < 0) {
            withStyle(SpanStyle(color = baseColor)) { append(text.substring(i)) }
            break
        }
        if (at > i) {
            withStyle(SpanStyle(color = baseColor)) { append(text.substring(i, at)) }
        }
        val allToken = "@${LinLanImManager.AT_ALL_DISPLAY}"
        val end = if (text.startsWith(allToken, at)) {
            at + allToken.length
        } else {
            var e = at + 1
            while (e < text.length) {
                val ch = text[e]
                if (ch == ' ' || ch == '\n' || ch == '@' || ch == '[') break
                e++
            }
            e
        }
        if (end > at + 1) {
            withStyle(SpanStyle(color = mentionColor, fontWeight = FontWeight.Medium)) {
                append(text.substring(at, end))
            }
            i = end
        } else {
            withStyle(SpanStyle(color = baseColor)) { append("@") }
            i = at + 1
        }
    }
}

private fun copyText(context: Context, text: String) {
    val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    cm.setPrimaryClip(ClipData.newPlainText("message", text))
}

private fun copyUriToCache(
    context: Context,
    uri: android.net.Uri,
    prefix: String,
): String? = runCatching {
    val dir = File(context.cacheDir, "chat_media").also { it.mkdirs() }
    val out = File(dir, "$prefix${System.currentTimeMillis()}")
    context.contentResolver.openInputStream(uri)?.use { input ->
        out.outputStream().use { output -> input.copyTo(output) }
    }
    out.absolutePath
}.getOrNull()

/** @return Pair(snapshotPath, durationSeconds) */
private fun createVideoSnapshot(context: Context, videoPath: String): Pair<String, Int>? = runCatching {
    val retriever = MediaMetadataRetriever()
    retriever.setDataSource(videoPath)
    val durationMs = retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)
        ?.toLongOrNull() ?: 1000L
    val bitmap = retriever.getFrameAtTime(0L, MediaMetadataRetriever.OPTION_CLOSEST_SYNC)
        ?: retriever.getFrameAtTime()
        ?: return@runCatching null
    val dir = File(context.cacheDir, "chat_media").also { it.mkdirs() }
    val out = File(dir, "snap_${System.currentTimeMillis()}.jpg")
    FileOutputStream(out).use { bitmap.compress(Bitmap.CompressFormat.JPEG, 85, it) }
    retriever.release()
    out.absolutePath to (durationMs / 1000L).toInt().coerceAtLeast(1)
}.getOrNull()

private fun openPathWithChooser(context: Context, pathOrUrl: String, fileName: String) {
    runCatching {
        if (pathOrUrl.startsWith("http://") || pathOrUrl.startsWith("https://")) {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(pathOrUrl)))
            return
        }
        val file = File(pathOrUrl)
        val uri = FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            file,
        )
        val mime = when {
            LinLanImManager.isImageFileName(fileName) -> "image/*"
            fileName.lowercase().endsWith(".mp4") || fileName.lowercase().endsWith(".mov") -> "video/*"
            fileName.lowercase().endsWith(".pdf") -> "application/pdf"
            else -> "*/*"
        }
        val intent = Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, mime)
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        context.startActivity(Intent.createChooser(intent, "打开文件"))
    }.onFailure {
        Toast.makeText(context, it.message ?: "无法打开文件", Toast.LENGTH_SHORT).show()
    }
}

@Composable
private fun ContactQuickTile(title: String, icon: ImageVector, badge: Int, modifier: Modifier, onClick: () -> Unit) {
    Column(modifier.clip(RoundedCornerShape(20.dp)).background(MintCard).clickable(onClick = onClick).padding(18.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Icon(icon, null, tint = MintAccent, modifier = Modifier.size(28.dp))
            if (badge > 0) Badge { Text(if (badge > 99) "99+" else badge.toString()) }
        }
        Spacer(Modifier.height(18.dp))
        Text(title, fontWeight = FontWeight.SemiBold, color = MintTitle)
    }
}
