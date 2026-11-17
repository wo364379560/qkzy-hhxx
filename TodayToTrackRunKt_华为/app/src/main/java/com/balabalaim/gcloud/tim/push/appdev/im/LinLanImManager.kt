package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.Context
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.data.local.PinnedMessage
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanReliability
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallManager
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload
import com.tencent.imsdk.v2.V2TIMAdvancedMsgListener
import com.tencent.imsdk.v2.V2TIMCallback
import com.tencent.imsdk.v2.V2TIMConversation
import com.tencent.imsdk.v2.V2TIMConversationListener
import com.tencent.imsdk.v2.V2TIMConversationResult
import com.tencent.imsdk.v2.V2TIMCreateGroupMemberInfo
import com.tencent.imsdk.v2.V2TIMFriendAddApplication
import com.tencent.imsdk.v2.V2TIMFriendApplication
import com.tencent.imsdk.v2.V2TIMFriendApplicationResult
import com.tencent.imsdk.v2.V2TIMFriendInfo
import com.tencent.imsdk.v2.V2TIMFriendshipListener
import com.tencent.imsdk.v2.V2TIMFriendOperationResult
import com.tencent.imsdk.v2.V2TIMGroupInfo
import com.tencent.imsdk.v2.V2TIMGroupChangeInfo
import com.tencent.imsdk.v2.V2TIMGroupListener
import com.tencent.imsdk.v2.V2TIMGroupInfoResult
import com.tencent.imsdk.v2.V2TIMGroupMemberChangeInfo
import com.tencent.imsdk.v2.V2TIMGroupMemberFullInfo
import com.tencent.imsdk.v2.V2TIMGroupMemberInfo
import com.tencent.imsdk.v2.V2TIMGroupMemberInfoResult
import com.tencent.imsdk.v2.V2TIMGroupMemberOperationResult
import com.tencent.imsdk.v2.V2TIMGroupTipsElem
import com.tencent.imsdk.v2.V2TIMManager
import com.tencent.imsdk.v2.V2TIMMessage
import com.tencent.imsdk.v2.V2TIMMessageReceipt
import com.tencent.imsdk.v2.V2TIMMessageListGetOption
import com.tencent.imsdk.v2.V2TIMSDKConfig
import com.tencent.imsdk.v2.V2TIMSDKListener
import com.tencent.imsdk.v2.V2TIMSendCallback
import com.tencent.imsdk.v2.V2TIMUserFullInfo
import com.tencent.imsdk.v2.V2TIMValueCallback
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import org.json.JSONObject
import org.json.JSONArray
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

object LinLanImManager {
    enum class ConnectionState { Connecting, Connected, Disconnected, KickedOffline }
    const val GROUP_NOTICE_MAX_BYTES = 5000
    const val GROUP_NOTICE_IM_MAX_BYTES = 400
    const val CARD_BUSINESS_ID = "contact_card_v1"
    const val GROUP_CARD_BUSINESS_ID = "group_card_v1"
    const val GROUP_CREATE_BUSINESS_ID = "group_create"
    private const val GROUP_PINNED_MESSAGES_KEY = "inner_attr_pinned_messages_v1"
    const val AT_ALL_DISPLAY = "全体成员"
    val AT_ALL_TAG: String = com.tencent.imsdk.v2.V2TIMGroupAtInfo.AT_ALL_TAG

    /** 同意好友后自动下发的问候语（各端文案），聊天里按 tip 展示。 */
    const val FRIEND_GREETING_TEXT = "你我已成为好友，开始聊天吧"
    private val FRIEND_GREETING_TEXTS = setOf(
        "我已通过你的好友验证请求，现在我们可以开始聊天了",
        "I have accepted your friend request, now we can start chatting",
        FRIEND_GREETING_TEXT,
    )

    enum class AddFriendOutcome {
        AddedDirectly,
        PendingConfirm,
        AlreadyFriends,
        ;

        val toast: String
            get() = when (this) {
                AddedDirectly -> "已添加为好友"
                PendingConfirm -> "已发送好友申请"
                AlreadyFriends -> "对方已经是你的好友"
            }
    }

    private var initialized = false
    private var initializedSdkAppId = 0
    private var listenersRegistered = false
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private var recallPrefs: android.content.SharedPreferences? = null
    private val recallSession = java.util.concurrent.atomic.AtomicLong()
    private val recalled = RecalledMessageState(
        read = { recallPrefs?.getString(it, null) },
        write = { key, data -> recallPrefs?.edit()?.putString(key, data)?.apply(); Unit },
    )
    private fun recallScope(): RecalledMessageState.Scope? = recalled.current()?.takeIf {
        it.key == "${currentSdkAppId()}:${V2TIMManager.getInstance().loginUser.orEmpty()}"
    }
    private fun observeRecalls(messages: List<V2TIMMessage>): List<V2TIMMessage> {
        val owner = recallScope() ?: return messages
        messages.filter { it.status == V2TIMMessage.V2TIM_MSG_STATUS_LOCAL_REVOKED }.forEach {
            recalled.receive(owner, it.msgID.orEmpty(), it.revokerInfo?.userID.orEmpty(), it.sender.orEmpty(), it.groupID.orEmpty())
        }
        return messages
    }
    private fun receivedRecall(id: String, operator: String = "") {
        val owner = recallScope() ?: return
        if (!recalled.receive(owner, id, operator)) return
        _messageRevoked.tryEmit(id)
        scope.launch {
            if (!recalled.isCurrent(owner)) return@launch
            val found = runCatching { findMessages(listOf(id)) }.getOrDefault(emptyList())
            if (!recalled.isCurrent(owner)) return@launch
            found.firstOrNull()?.let {
                recalled.receive(owner, id, operator.ifBlank { it.revokerInfo?.userID.orEmpty() }, it.sender.orEmpty(), it.groupID.orEmpty())
                it.groupID?.takeIf { group -> group.isNotBlank() }?.let { group -> _groupPinnedChanged.tryEmit(group) }
            }
            _messageRevoked.tryEmit(id)
            notifyConversationsChanged()
        }
        notifyConversationsChanged()
    }

    private val _conversationsChanged = MutableSharedFlow<Unit>(extraBufferCapacity = 16)
    val conversationsChanged: SharedFlow<Unit> = _conversationsChanged.asSharedFlow()

    private val _incomingMessage = MutableSharedFlow<V2TIMMessage>(extraBufferCapacity = 64)
    val incomingMessage: SharedFlow<V2TIMMessage> = _incomingMessage.asSharedFlow()

    private val _readReceipts = MutableSharedFlow<Pair<List<V2TIMMessageReceipt>, Boolean>>(extraBufferCapacity = 64)
    val readReceipts = _readReceipts.asSharedFlow()

    private val _messageRevoked = MutableSharedFlow<String>(extraBufferCapacity = 64)
    val messageRevoked: SharedFlow<String> = _messageRevoked.asSharedFlow()

    /** 消息列表页收到、尚未进聊天查看的对方消息（实时对象，用于已读回执）。 */
    private val pendingReceiptMessages = mutableMapOf<String, MutableList<V2TIMMessage>>()

    /** groupId → 是否 Community；避免每次发送都拉取群资料。 */
    private val communityTypeCache = mutableMapOf<String, Boolean>()

    private val _friendsChanged = MutableSharedFlow<Unit>(extraBufferCapacity = 16)
    val friendsChanged: SharedFlow<Unit> = _friendsChanged.asSharedFlow()

    private val _groupPinnedChanged = MutableSharedFlow<String>(extraBufferCapacity = 16)
    val groupPinnedChanged: SharedFlow<String> = _groupPinnedChanged.asSharedFlow()

    private val _groupSendPolicyChanged = MutableSharedFlow<String>(extraBufferCapacity = 16)
    val groupSendPolicyChanged: SharedFlow<String> = _groupSendPolicyChanged.asSharedFlow()

    private val _connectionState = MutableStateFlow(ConnectionState.Disconnected)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val conversationListener = object : V2TIMConversationListener() {
        override fun onConversationChanged(conversationList: MutableList<V2TIMConversation>?) {
            notifyConversationsChanged()
        }

        override fun onNewConversation(conversationList: MutableList<V2TIMConversation>?) {
            notifyConversationsChanged()
        }

        override fun onTotalUnreadMessageCountChanged(totalUnreadCount: Long) {
            notifyConversationsChanged()
        }
    }

    private val messageListener = object : V2TIMAdvancedMsgListener() {
        override fun onRecvC2CReadReceipt(receiptList: MutableList<V2TIMMessageReceipt>?) {
            _readReceipts.tryEmit(receiptList.orEmpty() to true)
        }
        override fun onRecvMessageReadReceipts(receiptList: MutableList<V2TIMMessageReceipt>?) {
            _readReceipts.tryEmit(receiptList.orEmpty() to false)
        }

        override fun onRecvNewMessage(msg: V2TIMMessage?) {
            if (msg != null) {
                LinLanRtcCallManager.onIncomingMessage(msg)
                bufferReceiptMessageIfNeeded(msg)
                seedGroupAtMentionCache(msg)
                _incomingMessage.tryEmit(msg)
                val groupId = msg.groupID?.trim().orEmpty()
                if (groupId.isNotEmpty()) {
                    // Community 常不在 getConversationList，收到群消息时主动拉会话。
                    scope.launch {
                        runCatching { fetchConversationOrNull("group_$groupId") }
                        notifyConversationsChanged()
                    }
                    return
                }
            }
            notifyConversationsChanged()
        }

        override fun onRecvMessageRevoked(msgID: String?) {
            val id = msgID?.trim().orEmpty()
            if (id.isNotEmpty()) receivedRecall(id)
        }

        override fun onRecvMessageRevoked(
            msgID: String?,
            operateUser: V2TIMUserFullInfo?,
            reason: String?,
        ) {
            val id = msgID?.trim().orEmpty()
            if (id.isNotEmpty()) receivedRecall(id, operateUser?.userID.orEmpty())
        }
    }

    private val friendshipListener = object : V2TIMFriendshipListener() {
        override fun onFriendListAdded(infoList: MutableList<V2TIMFriendInfo>?) {
            infoList.orEmpty().take(16).forEach { LinLanFriendOwnership.established(it.userID) }
            notifyFriendsChanged()
        }

        override fun onFriendListDeleted(userIDList: MutableList<String>?) {
            notifyFriendsChanged()
        }

        override fun onFriendApplicationListAdded(applicationList: MutableList<V2TIMFriendApplication>?) {
            notifyFriendsChanged()
        }

        override fun onFriendApplicationListDeleted(userIDList: MutableList<String>?) {
            notifyFriendsChanged()
        }
    }

    private val groupListener = object : V2TIMGroupListener() {
        override fun onGroupAttributeChanged(
            groupID: String?,
            groupAttributeMap: MutableMap<String, String>?,
        ) {
            val groupId = groupID?.trim().orEmpty()
            if (groupId.isNotEmpty() && groupAttributeMap?.containsKey(GROUP_PINNED_MESSAGES_KEY) == true) {
                _groupPinnedChanged.tryEmit(groupId)
            }
        }

        override fun onAllGroupMembersMuted(groupID: String?, isMute: Boolean) {
            groupID?.takeIf { it.isNotBlank() }?.let(::announceGroupSendPermission)
        }

        override fun onGrantAdministrator(groupID: String?, opUser: V2TIMGroupMemberInfo?, memberList: MutableList<V2TIMGroupMemberInfo>?) {
            refreshSelfPermission(groupID, memberList)
        }

        override fun onRevokeAdministrator(groupID: String?, opUser: V2TIMGroupMemberInfo?, memberList: MutableList<V2TIMGroupMemberInfo>?) {
            refreshSelfPermission(groupID, memberList)
        }

        override fun onMemberEnter(groupID: String?, memberList: MutableList<V2TIMGroupMemberInfo>?) {
            refreshSelfPermission(groupID, memberList)
        }

        override fun onMemberInvited(groupID: String?, opUser: V2TIMGroupMemberInfo?, memberList: MutableList<V2TIMGroupMemberInfo>?) {
            refreshSelfPermission(groupID, memberList)
        }

        override fun onMemberKicked(groupID: String?, opUser: V2TIMGroupMemberInfo?, memberList: MutableList<V2TIMGroupMemberInfo>?) {
            refreshSelfPermission(groupID, memberList)
        }

        override fun onMemberLeave(groupID: String?, member: V2TIMGroupMemberInfo?) {
            refreshSelfPermission(groupID, listOfNotNull(member))
        }

        private fun refreshSelfPermission(groupID: String?, members: List<V2TIMGroupMemberInfo>?) {
            val owner = V2TIMManager.getInstance().loginUser.orEmpty()
            if (!groupID.isNullOrBlank() && owner.isNotBlank() && members.orEmpty().any { it.userID == owner }) {
                announceGroupSendPermission(groupID)
            }
        }

        override fun onGroupInfoChanged(
            groupID: String?,
            changeInfos: MutableList<V2TIMGroupChangeInfo>?,
        ) {
            val groupId = groupID?.trim().orEmpty()
            if (groupId.isNotEmpty()) announceGroupSendPermission(groupId)
        }

        override fun onMemberInfoChanged(
            groupID: String?,
            v2TIMGroupMemberChangeInfoList: MutableList<V2TIMGroupMemberChangeInfo>?,
        ) {
            val groupId = groupID?.trim().orEmpty()
            if (groupId.isNotEmpty()) announceGroupSendPermission(groupId)
        }
    }

    private val sdkListener = object : V2TIMSDKListener() {
        override fun onConnecting() {
            _connectionState.value = ConnectionState.Connecting
        }

        override fun onConnectSuccess() {
            _connectionState.value = ConnectionState.Connected
        }

        override fun onConnectFailed(code: Int, error: String?) {
            _connectionState.value = ConnectionState.Disconnected
        }

        override fun onKickedOffline() {
            _connectionState.value = ConnectionState.KickedOffline
        }

        override fun onUserSigExpired() {
            _connectionState.value = ConnectionState.Connecting
            scope.launch {
                runCatching {
                    val refreshed = LinLanAppContainer.apiClient().refreshSession()
                    LinLanRuntimeConfig.applySdkAppId(refreshed.sdkAppId)
                    login(refreshed.userId, refreshed.userSig)
                    LinLanRtcCallManager.refreshLoginState(refreshed.userId, refreshed.userSig)
                }.onFailure {
                    _connectionState.value = ConnectionState.Disconnected
                }
            }
        }
    }

    fun init(context: Context, sdkAppId: Int = LinLanAppConstants.SDK_APP_ID) {
        recallPrefs = context.applicationContext.getSharedPreferences("rui_recalled_messages_v1", Context.MODE_PRIVATE)
        val effectiveSdkAppId = sdkAppId.takeIf { it > 0 } ?: LinLanAppConstants.SDK_APP_ID
        if (initialized && initializedSdkAppId == effectiveSdkAppId) return
        if (initialized) {
            recallSession.incrementAndGet(); recalled.deactivate()
            permissionMemo.clear()
            LinLanRtcCallManager.unbindSignaling()
            unregisterListenersIfNeeded()
            V2TIMManager.getInstance().unInitSDK()
            initialized = false
        }
        val config = V2TIMSDKConfig().apply {
            logLevel = V2TIMSDKConfig.V2TIM_LOG_INFO
        }
        val success = V2TIMManager.getInstance().initSDK(context.applicationContext, effectiveSdkAppId, config)
        if (!success) throw IllegalStateException("腾讯云 IM SDK 初始化失败")
        initialized = true
        initializedSdkAppId = effectiveSdkAppId
        registerListenersIfNeeded()
        LinLanRtcCallManager.setup(context)
    }

    fun currentSdkAppId(): Int = if (initialized) initializedSdkAppId else 0
    fun isReadyFor(userID: String, sdkAppID: Int): Boolean = initialized && initializedSdkAppId == sdkAppID &&
        userID.isNotBlank() && V2TIMManager.getInstance().loginUser == userID &&
        V2TIMManager.getInstance().loginStatus == V2TIMManager.V2TIM_STATUS_LOGINED


    private fun unregisterListenersIfNeeded() {
        if (!listenersRegistered) return
        V2TIMManager.getConversationManager().removeConversationListener(conversationListener)
        V2TIMManager.getMessageManager().removeAdvancedMsgListener(messageListener)
        V2TIMManager.getFriendshipManager().removeFriendListener(friendshipListener)
        V2TIMManager.getInstance().removeGroupListener(groupListener)
        V2TIMManager.getInstance().removeIMSDKListener(sdkListener)
        listenersRegistered = false
    }

    private fun registerListenersIfNeeded() {
        if (listenersRegistered) return
        V2TIMManager.getConversationManager().addConversationListener(conversationListener)
        V2TIMManager.getMessageManager().addAdvancedMsgListener(messageListener)
        V2TIMManager.getFriendshipManager().addFriendListener(friendshipListener)
        V2TIMManager.getInstance().addGroupListener(groupListener)
        V2TIMManager.getInstance().addIMSDKListener(sdkListener)
        listenersRegistered = true
    }

    fun notifyConversationsChanged() {
        _conversationsChanged.tryEmit(Unit)
    }

    fun notifyFriendsChanged() {
        _friendsChanged.tryEmit(Unit)
    }

    suspend fun login(userId: String, userSig: String) = suspendCancellableCoroutine { cont ->
        val generation = recallSession.incrementAndGet()
        recalled.deactivate()
        _connectionState.value = ConnectionState.Connecting
        registerListenersIfNeeded()
        V2TIMManager.getInstance().login(userId, userSig, object : V2TIMCallback {
            override fun onSuccess() {
                if (recallSession.get() != generation || V2TIMManager.getInstance().loginUser != userId) {
                    if (cont.isActive) cont.resumeWithException(CancellationException("账号已切换"))
                    return
                }
                recalled.activate("${currentSdkAppId()}:$userId")
                _connectionState.value = ConnectionState.Connected
                notifyConversationsChanged()
                if (cont.isActive) cont.resume(Unit)
            }

            override fun onError(code: Int, desc: String?) {
                if (cont.isActive) cont.resumeWithException(Exception("IM 登录失败($code): $desc"))
            }
        })
    }

    suspend fun logout() = suspendCancellableCoroutine { cont ->
        recallSession.incrementAndGet(); recalled.deactivate()
        permissionMemo.clear()
        V2TIMManager.getInstance().logout(object : V2TIMCallback {
            override fun onSuccess() {
                _connectionState.value = ConnectionState.Disconnected
                if (cont.isActive) cont.resume(Unit)
            }

            override fun onError(code: Int, desc: String?) {
                if (cont.isActive) cont.resumeWithException(Exception("IM 登出失败($code): $desc"))
            }
        })
    }

    suspend fun fetchConversations(): List<V2TIMConversation> {
        val all = fetchAllConversationPages().toMutableList()
        val knownIds = all.mapNotNull { it.conversationID?.trim()?.takeIf { id -> id.isNotEmpty() } }.toMutableSet()
        val knownGroupIds = all.mapNotNull { it.groupID?.trim()?.takeIf { id -> id.isNotEmpty() } }.toMutableSet()

        // 社群（Community / @TGS#_）经常不在 getConversationList，按已加入群补拉（对齐 Web/iOS）。
        val joinedGroups = runCatching { fetchJoinedGroups() }.getOrDefault(emptyList())
        for (group in joinedGroups) {
            val groupId = group.groupID?.trim().orEmpty()
            if (groupId.isEmpty()) continue
            if (knownGroupIds.any { communityPeerRelated(it, groupId) }) continue
            val convId = "group_$groupId"
            if (knownIds.contains(convId)) continue
            val conv = fetchConversationOrNull(convId) ?: continue
            all += conv
            knownIds += conv.conversationID?.trim().orEmpty()
            knownGroupIds += conv.groupID?.trim().orEmpty()
        }

        return all.sortedWith(
            compareByDescending<V2TIMConversation> { it.isPinned }
                .thenByDescending { it.orderKey }
        )
    }

    private suspend fun fetchAllConversationPages(): List<V2TIMConversation> {
        val all = mutableListOf<V2TIMConversation>()
        var nextSeq = 0L
        var finished = false
        while (!finished) {
            val page = fetchConversationPage(nextSeq)
            all += page.list
            nextSeq = page.nextSeq
            finished = page.isFinished
        }
        return all
    }

    private suspend fun fetchConversationPage(
        nextSeq: Long,
    ): ConversationPage = suspendCancellableCoroutine { cont ->
        V2TIMManager.getConversationManager().getConversationList(
            nextSeq,
            100,
            object : V2TIMValueCallback<V2TIMConversationResult> {
                override fun onSuccess(result: V2TIMConversationResult?) {
                    if (!cont.isActive) return
                    cont.resume(
                        ConversationPage(
                            list = result?.conversationList.orEmpty(),
                            nextSeq = result?.nextSeq ?: 0L,
                            isFinished = result?.isFinished ?: true,
                        )
                    )
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resumeWithException(Exception("获取会话失败($code): $desc"))
                }
            }
        )
    }

    private data class ConversationPage(
        val list: List<V2TIMConversation>,
        val nextSeq: Long,
        val isFinished: Boolean,
    )

    suspend fun fetchConversationOrNull(conversationId: String): V2TIMConversation? =
        suspendCancellableCoroutine { cont ->
            val id = conversationId.trim()
            if (id.isEmpty()) {
                cont.resume(null)
                return@suspendCancellableCoroutine
            }
            V2TIMManager.getConversationManager().getConversation(
                id,
                object : V2TIMValueCallback<V2TIMConversation> {
                    override fun onSuccess(t: V2TIMConversation?) {
                        if (cont.isActive) cont.resume(t)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(null)
                    }
                },
            )
        }

    /** 社群根 ID（@TGS#_xxx）与默认群 ID（@TGS#_@TGS#xxx）视为同一社群。 */
    private fun communityPeerRelated(lhs: String, rhs: String): Boolean = lhs.trim() == rhs.trim()

    suspend fun fetchHistory(
        conversationId: String,
        count: Int = 30,
        lastMessage: V2TIMMessage? = null,
    ): List<V2TIMMessage> = suspendCancellableCoroutine { cont ->
        val requestedScope = recallScope()

        val option = V2TIMMessageListGetOption().apply {
            setGetType(V2TIMMessageListGetOption.V2TIM_GET_CLOUD_OLDER_MSG)
            setCount(count)
            if (lastMessage != null) setLastMsg(lastMessage)
            if (conversationId.startsWith("group_")) {
                setGroupID(conversationId.removePrefix("group_"))
            } else {
                setUserID(conversationId.removePrefix("c2c_"))
            }
        }
        V2TIMManager.getMessageManager().getHistoryMessageList(
            option,
            object : V2TIMValueCallback<List<V2TIMMessage>> {
                override fun onSuccess(result: List<V2TIMMessage>?) {
                    if (cont.isActive) {
                        if (!recalled.isCurrent(requestedScope)) cont.resumeWithException(CancellationException("账号已切换"))
                        else cont.resume(observeRecalls(result.orEmpty()))
                    }
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resumeWithException(Exception("获取消息失败($code): $desc"))
                }
            }
        )
    }

    suspend fun findMessages(ids: List<String>): List<V2TIMMessage> =
        suspendCancellableCoroutine { cont ->
        val requestedScope = recallScope()

            if (ids.isEmpty()) {
                cont.resume(emptyList())
                return@suspendCancellableCoroutine
            }
            V2TIMManager.getMessageManager().findMessages(
                ids,
                object : V2TIMValueCallback<List<V2TIMMessage>> {
                    override fun onSuccess(t: List<V2TIMMessage>?) {
                        if (cont.isActive) {
                        if (!recalled.isCurrent(requestedScope)) cont.resumeWithException(CancellationException("账号已切换"))
                        else cont.resume(observeRecalls(t.orEmpty()))
                    }
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("查找消息失败($code): $desc"))
                        }
                    }
                },
            )
        }

    /** Group pins are stored in TIM group attributes so every member/device sees the same list. */
    suspend fun fetchGroupPinnedMessages(groupId: String): List<PinnedMessage> {
        val raw = getGroupAttributes(groupId, listOf(GROUP_PINNED_MESSAGES_KEY))[GROUP_PINNED_MESSAGES_KEY]
            .orEmpty()
        if (raw.isBlank()) return emptyList()
        return runCatching {
            val array = JSONArray(raw)
            buildList {
                for (index in 0 until array.length()) {
                    val item = array.optJSONObject(index) ?: continue
                    val msgId = item.optString("msg_id").trim()
                    if (msgId.isEmpty()) continue
                    add(
                        PinnedMessage(
                            msgId = msgId,
                            summary = item.optString("summary"),
                            sender = item.optString("sender"),
                        ),
                    )
                }
            }
        }.getOrDefault(emptyList()).filterNot { isKnownRecalledMessage(it.msgId) }
    }

    suspend fun addGroupPinnedMessage(groupId: String, message: PinnedMessage) {
        val current = fetchGroupPinnedMessages(groupId).toMutableList()
        current.removeAll { it.msgId == message.msgId }
        current.add(0, message)
        persistGroupPinnedMessages(groupId, current.take(20))
    }

    suspend fun removeGroupPinnedMessage(groupId: String, messageId: String) {
        persistGroupPinnedMessages(
            groupId,
            fetchGroupPinnedMessages(groupId).filterNot { it.msgId == messageId },
        )
    }

    private suspend fun persistGroupPinnedMessages(groupId: String, messages: List<PinnedMessage>) {
        val array = JSONArray()
        messages.forEach { message ->
            array.put(
                JSONObject()
                    .put("msg_id", message.msgId)
                    .put("summary", truncateUtf8(message.summary, 240))
                    .put("sender", truncateUtf8(message.sender, 80)),
            )
        }
        setGroupAttributes(groupId, mapOf(GROUP_PINNED_MESSAGES_KEY to array.toString()))
        _groupPinnedChanged.tryEmit(groupId)
    }

    data class MessageSearchPage(
        val messages: List<V2TIMMessage>,
        val totalCount: Int,
        val nextCursor: String,
        val pageIndex: Int,
        val pageSize: Int,
        val fromCloud: Boolean,
    ) {
        val hasMore: Boolean
            get() = if (fromCloud) {
                nextCursor.isNotBlank()
            } else {
                messages.isNotEmpty() && (pageIndex + 1) * pageSize < totalCount
            }
    }

    /**
     * Search the current conversation. Cloud search keeps results complete across devices;
     * unsupported editions automatically fall back to the local TIM database.
     */
    suspend fun searchMessages(
        conversationId: String,
        keyword: String,
        pageSize: Int = 30,
        cursor: String = "",
        pageIndex: Int = 0,
        preferCloud: Boolean = true,
    ): MessageSearchPage {
        val term = keyword.trim()
        val normalizedPageSize = pageSize.coerceIn(1, 100)
        if (term.isEmpty()) {
            return MessageSearchPage(emptyList(), 0, "", pageIndex, normalizedPageSize, preferCloud)
        }
        val param = com.tencent.imsdk.v2.V2TIMMessageSearchParam().apply {
            setConversationID(conversationId)
            setKeywordList(listOf(term))
            setKeywordListMatchType(
                com.tencent.imsdk.v2.V2TIMMessageSearchParam.V2TIM_KEYWORD_LIST_MATCH_TYPE_OR,
            )
            setPageSize(normalizedPageSize)
            if (cursor.isNotBlank()) setSearchCursor(cursor) else setPageIndex(pageIndex.coerceAtLeast(0))
        }
        if (preferCloud) {
            runCatching { executeMessageSearch(param, cloud = true) }
                .getOrNull()
                ?.let { result ->
                    return result.toSearchPage(
                        pageIndex = pageIndex,
                        pageSize = normalizedPageSize,
                        fromCloud = true,
                    )
                }
        }
        return executeMessageSearch(param, cloud = false)
            .toSearchPage(
                pageIndex = pageIndex,
                pageSize = normalizedPageSize,
                fromCloud = false,
            )
    }

    private suspend fun executeMessageSearch(
        param: com.tencent.imsdk.v2.V2TIMMessageSearchParam,
        cloud: Boolean,
    ): com.tencent.imsdk.v2.V2TIMMessageSearchResult = suspendCancellableCoroutine { cont ->
        val callback = object : V2TIMValueCallback<com.tencent.imsdk.v2.V2TIMMessageSearchResult> {
            override fun onSuccess(result: com.tencent.imsdk.v2.V2TIMMessageSearchResult?) {
                if (cont.isActive) {
                    if (result == null) cont.resumeWithException(Exception("搜索结果为空"))
                    else cont.resume(result)
                }
            }

            override fun onError(code: Int, desc: String?) {
                if (cont.isActive) cont.resumeWithException(Exception("搜索消息失败($code): $desc"))
            }
        }
        if (cloud) {
            V2TIMManager.getMessageManager().searchCloudMessages(param, callback)
        } else {
            V2TIMManager.getMessageManager().searchLocalMessages(param, callback)
        }
    }

    private fun com.tencent.imsdk.v2.V2TIMMessageSearchResult.toSearchPage(
        pageIndex: Int,
        pageSize: Int,
        fromCloud: Boolean,
    ): MessageSearchPage {
        val flattened = messageSearchResultItems.orEmpty()
            .flatMap { it.messageList.orEmpty() }
            .distinctBy { it.msgID ?: "${it.seq}_${it.timestamp}" }
            .sortedByDescending { it.timestamp }
        return MessageSearchPage(
            messages = flattened,
            totalCount = totalCount,
            nextCursor = searchCursor.orEmpty(),
            pageIndex = pageIndex,
            pageSize = pageSize,
            fromCloud = fromCloud,
        )
    }

    suspend fun sendTextMessage(
        text: String,
        receiver: String?,
        groupId: String?,
        cloudCustomData: String? = null,
        atUserIds: List<String> = emptyList(),
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage {
        val mgr = V2TIMManager.getMessageManager()
        val msg = if (!groupId.isNullOrBlank() && atUserIds.isNotEmpty()) {
            @Suppress("DEPRECATION")
            mgr.createTextAtMessage(text, ArrayList(atUserIds))
                ?: mgr.createTextMessage(text)
        } else {
            mgr.createTextMessage(text)
        }
        if (!cloudCustomData.isNullOrBlank()) {
            msg.setCloudCustomData(cloudCustomData)
        }
        return sendCreated(msg, receiver, groupId, onPrepared, onProgress)
    }

    /** 微信风格：会话列表 [@所有人] / [有人@我] */
    data class GroupAtFlags(val showAtAll: Boolean, val showAtMe: Boolean)

    private val groupAtMentionCache = mutableMapOf<String, GroupAtFlags>()

    fun groupAtMentionFlags(conv: V2TIMConversation): GroupAtFlags {
        if (conv.type != V2TIMConversation.V2TIM_GROUP) return GroupAtFlags(false, false)
        val convId = conv.conversationID.orEmpty()
        if (conv.unreadCount <= 0) {
            groupAtMentionCache.remove(convId)
            return GroupAtFlags(false, false)
        }
        val list = conv.groupAtInfoList
        if (!list.isNullOrEmpty()) {
            val flags = parseGroupAtList(list)
            groupAtMentionCache[convId] = flags
            return flags
        }
        return groupAtMentionCache[convId] ?: GroupAtFlags(false, false)
    }

    private fun parseGroupAtList(list: List<com.tencent.imsdk.v2.V2TIMGroupAtInfo>): GroupAtFlags {
        var showAtAll = false
        var showAtMe = false
        for (info in list) {
            when (info.atType) {
                com.tencent.imsdk.v2.V2TIMGroupAtInfo.TIM_AT_ME -> showAtMe = true
                com.tencent.imsdk.v2.V2TIMGroupAtInfo.TIM_AT_ALL -> showAtAll = true
                com.tencent.imsdk.v2.V2TIMGroupAtInfo.TIM_AT_ALL_AT_ME -> {
                    showAtAll = true
                    showAtMe = true
                }
            }
        }
        return GroupAtFlags(showAtAll, showAtMe)
    }

    private fun seedGroupAtMentionCache(message: V2TIMMessage) {
        val groupId = message.groupID?.trim().orEmpty()
        if (groupId.isEmpty()) return
        val atList = message.groupAtUserList?.mapNotNull { it?.toString()?.trim() }?.filter { it.isNotEmpty() }
            .orEmpty()
        if (atList.isEmpty()) return
        val convId = "group_$groupId"
        val selfId = V2TIMManager.getInstance().loginUser?.trim().orEmpty()
        var showAtAll = false
        var showAtMe = false
        for (uid in atList) {
            when {
                uid == AT_ALL_TAG -> showAtAll = true
                selfId.isNotEmpty() && uid == selfId -> showAtMe = true
            }
        }
        if (!showAtAll && !showAtMe) return
        val existing = groupAtMentionCache[convId] ?: GroupAtFlags(false, false)
        groupAtMentionCache[convId] = GroupAtFlags(
            showAtAll = existing.showAtAll || showAtAll,
            showAtMe = existing.showAtMe || showAtMe,
        )
    }

    fun groupMemberDisplayName(member: V2TIMGroupMemberFullInfo): String {
        val card = member.nameCard?.trim().orEmpty()
        if (card.isNotEmpty()) return card
        val nick = member.nickName?.trim().orEmpty()
        if (nick.isNotEmpty()) return nick
        return member.userID?.trim().orEmpty().ifBlank { "成员" }
    }

    suspend fun sendFileMessage(
        path: String,
        fileName: String,
        receiver: String?,
        groupId: String?,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage {
        val size = LinLanCosLargeAttachment.fileSize(path)
        if (LinLanCosLargeAttachment.shouldRouteToCos(size)) {
            return LinLanCosLargeAttachment.uploadAndSend(
                path = path,
                fileName = fileName,
                receiver = receiver,
                groupId = groupId,
                isVideo = false,
            )
        }
        return sendCreated(
            V2TIMManager.getMessageManager().createFileMessage(path, fileName),
            receiver,
            groupId,
            onPrepared,
            onProgress,
        )
    }

    suspend fun retrySendMessage(
        message: V2TIMMessage,
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage {
        val groupId = message.groupID?.takeIf { it.isNotBlank() }
        val receiver = if (groupId == null) message.userID?.takeIf { it.isNotBlank() } else null
        return sendCreated(
            message = message,
            receiver = receiver,
            groupId = groupId,
            onProgress = onProgress,
        )
    }

    suspend fun forwardMessage(
        source: V2TIMMessage,
        receiver: String?,
        groupId: String?,
    ): V2TIMMessage = sendCreated(
        V2TIMManager.getMessageManager().createForwardMessage(source)
            ?: throw LinLanSendFailure("创建转发消息失败", notDispatched = true),
        receiver,
        groupId,
    )

    fun canForwardMessage(msg: V2TIMMessage): Boolean {
        return msg.status == V2TIMMessage.V2TIM_MSG_STATUS_SEND_SUCC &&
            !isRevokedMessage(msg) &&
            msg.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_GROUP_TIPS &&
            !isGroupCreateTipMessage(msg) &&
            !isFriendGreetingTipMessage(msg)
    }

    /**
     * 群主/管理员撤回他人消息时不展示「撤回了一条消息」，消息直接消失。
     * 自己撤回自己的消息仍展示提示。
     */
    fun shouldSilentlyHideRevokedMessage(msg: V2TIMMessage): Boolean = recalled.shouldHide(
        recallScope(), msg.msgID.orEmpty(), msg.groupID.orEmpty(), msg.sender.orEmpty(),
        msg.revokerInfo?.userID.orEmpty(), msg.status == V2TIMMessage.V2TIM_MSG_STATUS_LOCAL_REVOKED,
    )

    fun isKnownRecalledMessage(messageId: String): Boolean = recalled.isRevoked(recallScope(), messageId, false)

    fun isRevokedMessage(msg: V2TIMMessage): Boolean = recalled.isRevoked(
        recallScope(), msg.msgID.orEmpty(), msg.status == V2TIMMessage.V2TIM_MSG_STATUS_LOCAL_REVOKED,
    )

    fun revokedTipText(msg: V2TIMMessage): String =
        if (msg.isSelf) "你撤回了一条消息" else "对方撤回了一条消息"

    fun canRevokeMessage(
        msg: V2TIMMessage,
        selfGroupRole: Int = 0,
        senderMemberRole: Int = 0,
    ): Boolean {
        if (isRevokedMessage(msg)) return false
        if (msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_GROUP_TIPS) return false
        if (isGroupCreateTipMessage(msg)) return false
        if (isFriendGreetingTipMessage(msg)) return false
        if (msg.status != V2TIMMessage.V2TIM_MSG_STATUS_SEND_SUCC) return false

        if (msg.isSelf) {
            val ageMs = System.currentTimeMillis() - msg.timestamp * 1000L
            return ageMs in 0..120_000L
        }

        val groupId = msg.groupID?.trim().orEmpty()
        if (groupId.isEmpty() || !canManageGroup(selfGroupRole)) return false
        val selfId = V2TIMManager.getInstance().loginUser?.trim().orEmpty()
        if (selfId.isEmpty()) return false
        return canRevokeGroupMemberMessage(
            senderUserId = msg.sender?.trim().orEmpty(),
            senderRole = senderMemberRole,
            selfUserId = selfId,
            selfRole = selfGroupRole,
        )
    }

    fun canRevokeGroupMemberMessage(
        senderUserId: String,
        senderRole: Int,
        selfUserId: String,
        selfRole: Int,
    ): Boolean {
        if (senderUserId.isEmpty() || senderUserId == selfUserId) return false
        if (selfRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER || selfRole == 400) {
            return true
        }
        if (selfRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN || selfRole == 300) {
            return senderRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_MEMBER ||
                senderRole == 0 ||
                senderRole == 200
        }
        return false
    }

    /**
     * 自己撤回走 SDK（仍受 2 分钟限制）；群主/管理员撤回他人走后端 REST（无时限）。
     */
    suspend fun revokeMessage(msg: V2TIMMessage) {
        val owner = recallScope() ?: throw IllegalStateException("聊天服务尚未登录")
        val groupId = msg.groupID?.trim().orEmpty()
        val kind = if (!msg.isSelf && groupId.isNotEmpty()) RecalledMessageState.Kind.MODERATOR else RecalledMessageState.Kind.OWN
        val intent = recalled.begin(owner, msg.msgID.orEmpty(), kind, msg.sender.orEmpty(), groupId, V2TIMManager.getInstance().loginUser.orEmpty())
        try {
            if (kind == RecalledMessageState.Kind.MODERATOR) {
                val seq = msg.seq
                if (seq <= 0L) throw Exception("撤回失败：消息序号无效")
                LinLanAppContainer.apiClient().revokeGroupMessage(groupId, seq)
            } else {
                revokeOwnMessageBySdk(msg)
            }
            if (!recalled.confirm(intent)) throw CancellationException("账号已切换")
            _messageRevoked.tryEmit(msg.msgID.orEmpty())
            if (groupId.isNotEmpty()) _groupPinnedChanged.tryEmit(groupId)
            notifyConversationsChanged()
        } catch (error: Throwable) {
            recalled.cancel(intent)
            throw error
        }
    }

    private suspend fun revokeOwnMessageBySdk(msg: V2TIMMessage) {
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getMessageManager().revokeMessage(
                msg,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("撤回失败($code): $desc"))
                        }
                    }
                },
            )
        }
    }

    fun mergeForwardTitle(
        messages: List<V2TIMMessage>,
        isGroup: Boolean,
        conversationTitle: String,
    ): String {
        if (isGroup) {
            val name = conversationTitle.trim().ifBlank { "群聊" }
            return "${name}的聊天记录"
        }
        val names = messages.map { displayNameOfMessage(it).ifBlank { "用户" } }
            .distinct()
            .take(2)
        return when (names.size) {
            0 -> "聊天记录"
            1 -> "${names[0]}的聊天记录"
            else -> "${names[0]}和${names[1]}的聊天记录"
        }
    }

    fun mergeForwardAbstractList(messages: List<V2TIMMessage>): List<String> {
        return messages
            .sortedBy { it.timestamp }
            .take(5)
            .map { msg ->
                val sender = displayNameOfMessage(msg).ifBlank { "用户" }
                "$sender：${messagePreview(msg)}"
            }
    }

    suspend fun forwardMessagesOneByOne(
        sources: List<V2TIMMessage>,
        receiver: String?,
        groupId: String?,
    ) {
        val sorted = sources
            .filter { canForwardMessage(it) }
            .sortedWith(compareBy({ it.timestamp }, { it.seq }))
        if (sorted.isEmpty()) throw Exception("没有可转发的消息")
        for (source in sorted) {
            forwardMessage(source, receiver, groupId)
        }
    }

    suspend fun forwardMessagesMerged(
        sources: List<V2TIMMessage>,
        title: String,
        receiver: String?,
        groupId: String?,
    ): V2TIMMessage {
        val sorted = sources
            .filter { canForwardMessage(it) }
            .sortedWith(compareBy({ it.timestamp }, { it.seq }))
            .take(100)
        if (sorted.isEmpty()) throw Exception("没有可转发的消息")
        val abstracts = mergeForwardAbstractList(sorted)
        val merger = V2TIMManager.getMessageManager().createMergerMessage(
            sorted,
            title.ifBlank { "聊天记录" },
            abstracts,
            "请升级到最新版本查看聊天记录",
        ) ?: throw LinLanSendFailure("创建合并消息失败", notDispatched = true)
        return sendCreated(merger, receiver, groupId)
    }

    data class MergerCard(
        val title: String,
        val abstractList: List<String>,
        val layersOverLimit: Boolean,
    )

    fun parseMergerCard(msg: V2TIMMessage): MergerCard? {
        if (msg.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_MERGER) return null
        val elem = msg.mergerElem ?: return null
        return MergerCard(
            title = elem.title.orEmpty().ifBlank { "聊天记录" },
            abstractList = elem.abstractList.orEmpty(),
            layersOverLimit = elem.isLayersOverLimit,
        )
    }

    suspend fun downloadMergerMessage(msg: V2TIMMessage): List<V2TIMMessage> =
        suspendCancellableCoroutine { cont ->
            val elem = msg.mergerElem
            if (elem == null) {
                cont.resume(emptyList())
                return@suspendCancellableCoroutine
            }
            elem.downloadMergerMessage(object : V2TIMValueCallback<List<V2TIMMessage>> {
                override fun onSuccess(t: List<V2TIMMessage>?) {
                    if (cont.isActive) cont.resume(t.orEmpty())
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("加载聊天记录失败($code): $desc"))
                    }
                }
            })
        }

    /** Contact card custom message (businessID = contact_card_v1). */
    suspend fun sendContactCard(
        userId: String,
        nickname: String,
        receiver: String?,
        groupId: String?,
    ): V2TIMMessage {
        val payload = RuiSharedCard(userId, nickname, false).encode().toByteArray(Charsets.UTF_8)
        val desc = "[个人名片] $nickname".toByteArray(Charsets.UTF_8)
        val msg = V2TIMManager.getMessageManager().createCustomMessage(payload, "[个人名片] $nickname", desc)
        return sendCreated(msg, receiver, groupId)
    }

    suspend fun sendRunCard(card: com.balabalaim.gcloud.tim.push.appdev.run.RunShare, receiver: String?, groupId: String?): V2TIMMessage {
        val message = V2TIMManager.getMessageManager().createCustomMessage(
            card.encode().toByteArray(Charsets.UTF_8), card.summary, "rzh_run_v2".toByteArray(Charsets.UTF_8))
        val sent = sendCreated(message, receiver, groupId)
        _incomingMessage.tryEmit(sent)
        return sent
    }

    suspend fun sendGroupCard(
        groupId: String,
        groupName: String,
        receiver: String?,
        toGroupId: String?,
    ): V2TIMMessage {
        val payload = RuiSharedCard(groupId, groupName, true).encode().toByteArray(Charsets.UTF_8)
        val label = "[群名片] $groupName"
        val msg = V2TIMManager.getMessageManager().createCustomMessage(
            payload,
            label,
            label.toByteArray(Charsets.UTF_8),
        )
        return sendCreated(msg, receiver, toGroupId)
    }

    suspend fun sendRtcCallRecord(
        peerUserId: String,
        payload: com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload,
    ): V2TIMMessage {
        val text = payload.displayText(isSelf = true)
        val bytes = payload.toJson().toByteArray(Charsets.UTF_8)
        val msg = V2TIMManager.getMessageManager().createCustomMessage(
            bytes,
            text,
            text.toByteArray(Charsets.UTF_8),
        )
        val sent = sendCreated(msg, peerUserId, null)
        _incomingMessage.tryEmit(sent)
        return sent
    }

    fun shouldHideRtcSignalingInChat(msg: V2TIMMessage): Boolean {
        val payload = com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload.parseFromMessage(msg)
            ?: return false
        return com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload.shouldHideInChat(payload)
    }

    fun conversationIdOf(msg: V2TIMMessage): String {
        val gid = msg.groupID
        if (!gid.isNullOrBlank()) return "group_$gid"
        // 收消息用 sender 作为对方；自己发的用 userID
        val peer = if (msg.isSelf) {
            msg.userID?.takeIf { it.isNotBlank() } ?: msg.sender
        } else {
            msg.sender?.takeIf { it.isNotBlank() } ?: msg.userID
        }
        return "c2c_${peer.orEmpty()}"
    }

    fun isRecvOptMuted(recvOpt: Int): Boolean {
        return recvOpt == V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE ||
            recvOpt == V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE_EXCEPT_AT ||
            recvOpt == V2TIMMessage.V2TIM_NOT_RECEIVE_MESSAGE ||
            recvOpt == V2TIMMessage.V2TIM_NOT_RECEIVE_MESSAGE_EXCEPT_AT
    }

    suspend fun isMessageMuted(msg: V2TIMMessage): Boolean {
        val convId = conversationIdOf(msg)
        if (convId.isBlank() || convId == "c2c_" || convId == "group_") return false
        return runCatching {
            isRecvOptMuted(fetchConversationRecvOpt(convId))
        }.getOrDefault(false)
    }

    suspend fun fetchConversationRecvOpt(conversationId: String): Int =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().getConversation(
                conversationId,
                object : V2TIMValueCallback<V2TIMConversation> {
                    override fun onSuccess(t: V2TIMConversation?) {
                        if (cont.isActive) cont.resume(t?.recvOpt ?: V2TIMMessage.V2TIM_RECEIVE_MESSAGE)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(V2TIMMessage.V2TIM_RECEIVE_MESSAGE)
                    }
                },
            )
        }

    fun belongsToConversation(msg: V2TIMMessage, conversationId: String, isGroup: Boolean): Boolean =
        matchesChatConversation(conversationId, msg.groupID, msg.userID, msg.sender)

    private fun bufferReceiptMessageIfNeeded(msg: V2TIMMessage) {
        if (msg.isSelf) return
        val convId = conversationIdOf(msg)
        if (convId.isBlank() || convId == "c2c_" || convId == "group_") return
        val msgId = msg.msgID.orEmpty()
        if (msgId.isEmpty()) return
        val list = pendingReceiptMessages.getOrPut(convId) { mutableListOf() }
        synchronized(list) {
            if (list.any { it.msgID == msgId }) return
            list.add(msg)
            if (list.size > 100) list.removeAt(0)
        }
    }

    private fun drainPendingReceiptMessages(conversationId: String): List<V2TIMMessage> {
        val list = pendingReceiptMessages.remove(conversationId) ?: return emptyList()
        synchronized(list) { return list.toList() }
    }

    private fun pendingReadReceiptMessages(messages: List<V2TIMMessage>): List<V2TIMMessage> {
        val peerMessages = messages.filter {
            !it.isSelf &&
                it.status == V2TIMMessage.V2TIM_MSG_STATUS_SEND_SUCC &&
                it.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_GROUP_TIPS
        }
        val pending = peerMessages.filter { it.isNeedReadReceipt }
        // 本地缓存有时丢失 needReadReceipt，C2C 兜底尝试全部对方消息（与 iOS ShangLan 一致）。
        return if (pending.isNotEmpty()) pending else peerMessages.filter { it.groupID.isNullOrBlank() }
    }

    /** PC/Web 端「已读」依赖 sendMessageReadReceipts，仅 cleanConversationUnread 不够。 */
    fun sendReadReceipts(messages: List<V2TIMMessage>) {
        val targets = pendingReadReceiptMessages(messages)
            .distinctBy { it.msgID.orEmpty() }
            .filter { it.msgID.orEmpty().isNotEmpty() }
        if (targets.isEmpty()) return
        targets.groupBy { conversationIdOf(it) }.values.forEach { group ->
            group.chunked(30).forEach { batch ->
                V2TIMManager.getMessageManager().sendMessageReadReceipts(batch, object : V2TIMCallback {
                    override fun onSuccess() = Unit
                    override fun onError(code: Int, desc: String?) = Unit
                })
            }
        }
    }

    suspend fun fetchReadReceipts(messages: List<V2TIMMessage>): List<V2TIMMessageReceipt> {
        val result = mutableListOf<V2TIMMessageReceipt>()
        messages.filter { it.isSelf && it.groupID.isNullOrBlank() && it.isNeedReadReceipt }
            .groupBy { conversationIdOf(it) }.values.forEach { group ->
                group.chunked(30).forEach { batch ->
                    val receipts = suspendCancellableCoroutine<List<V2TIMMessageReceipt>> { cont ->
                        V2TIMManager.getMessageManager().getMessageReadReceipts(batch, object : V2TIMValueCallback<List<V2TIMMessageReceipt>> {
                            override fun onSuccess(value: List<V2TIMMessageReceipt>?) { if (cont.isActive) cont.resume(value.orEmpty()) }
                            override fun onError(code: Int, desc: String?) { if (cont.isActive) cont.resume(emptyList()) }
                        })
                    }
                    result.addAll(receipts)
                }
            }
        return result
    }

    suspend fun markConversationRead(
        conversationId: String,
        receiptMessages: List<V2TIMMessage> = emptyList(),
    ) {
        val trimmed = conversationId.trim()
        if (trimmed.isEmpty()) return
        if (trimmed.startsWith("c2c_")) {
            V2TIMManager.getMessageManager().markC2CMessageAsRead(trimmed.removePrefix("c2c_"), null)
        }
        val buffered = drainPendingReceiptMessages(trimmed)
        sendReadReceipts(buffered + receiptMessages)
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().cleanConversationUnreadMessageCount(
                trimmed,
                0,
                0,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        notifyConversationsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(Unit)
                    }
                },
            )
        }
    }

    suspend fun cleanConversationUnread(conversationId: String) {
        markConversationRead(conversationId, emptyList())
    }

    suspend fun totalUnreadCount(): Long = suspendCancellableCoroutine { cont ->
        V2TIMManager.getConversationManager().getTotalUnreadMessageCount(
            object : V2TIMValueCallback<Long> {
                override fun onSuccess(t: Long?) {
                    if (cont.isActive) cont.resume(t ?: 0L)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resume(0L)
                }
            }
        )
    }

    suspend fun deleteConversation(conversationId: String) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().deleteConversation(
                conversationId,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        notifyConversationsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("删除会话失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun deleteFriend(userId: String) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getFriendshipManager().deleteFromFriendList(
                listOf(userId),
                V2TIMFriendInfo.V2TIM_FRIEND_TYPE_BOTH,
                object : V2TIMValueCallback<List<V2TIMFriendOperationResult>> {
                    override fun onSuccess(t: List<V2TIMFriendOperationResult>?) {
                        notifyFriendsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("删除好友失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun pinConversation(conversationId: String, pinned: Boolean) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().pinConversation(
                conversationId,
                pinned,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        notifyConversationsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("置顶失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun isConversationPinned(conversationId: String): Boolean =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().getConversation(
                conversationId,
                object : V2TIMValueCallback<V2TIMConversation> {
                    override fun onSuccess(t: V2TIMConversation?) {
                        if (cont.isActive) cont.resume(t?.isPinned == true)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(false)
                    }
                }
            )
        }

    suspend fun setC2CMute(userId: String, muted: Boolean) =
        suspendCancellableCoroutine { cont ->
            val opt = if (muted) {
                V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
            } else {
                V2TIMMessage.V2TIM_RECEIVE_MESSAGE
            }
            V2TIMManager.getMessageManager().setC2CReceiveMessageOpt(
                listOf(userId),
                opt,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("设置免打扰失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun isC2CMuted(userId: String): Boolean =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getMessageManager().getC2CReceiveMessageOpt(
                listOf(userId),
                object : V2TIMValueCallback<List<com.tencent.imsdk.v2.V2TIMReceiveMessageOptInfo>> {
                    override fun onSuccess(t: List<com.tencent.imsdk.v2.V2TIMReceiveMessageOptInfo>?) {
                        val opt = t?.firstOrNull()?.c2CReceiveMessageOpt
                        if (cont.isActive) {
                            cont.resume(opt == V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE)
                        }
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(false)
                    }
                }
            )
        }

    suspend fun setFriendRemark(userId: String, remark: String) =
        suspendCancellableCoroutine { cont ->
            val info = V2TIMFriendInfo().apply {
                setUserID(userId)
                setFriendRemark(remark)
            }
            V2TIMManager.getFriendshipManager().setFriendInfo(
                info,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        notifyFriendsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("设置备注失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun addToBlackList(userId: String) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getFriendshipManager().addToBlackList(
                listOf(userId),
                object : V2TIMValueCallback<List<V2TIMFriendOperationResult>> {
                    override fun onSuccess(t: List<V2TIMFriendOperationResult>?) {
                        notifyFriendsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("拉黑失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun fetchBlackList(): List<V2TIMFriendInfo> = suspendCancellableCoroutine { cont ->
        V2TIMManager.getFriendshipManager().getBlackList(
            object : V2TIMValueCallback<List<V2TIMFriendInfo>> {
                override fun onSuccess(t: List<V2TIMFriendInfo>?) {
                    if (cont.isActive) cont.resume(t.orEmpty())
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("获取黑名单失败($code): $desc"))
                    }
                }
            }
        )
    }

    suspend fun removeFromBlackList(userId: String) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getFriendshipManager().deleteFromBlackList(
                listOf(userId),
                object : V2TIMValueCallback<List<V2TIMFriendOperationResult>> {
                    override fun onSuccess(t: List<V2TIMFriendOperationResult>?) {
                        notifyFriendsChanged()
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("移出黑名单失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun fetchJoinedGroups(): List<V2TIMGroupInfo> {
        val normal = fetchJoinedGroupListRaw()
        val communities = runCatching { fetchJoinedCommunityListRaw() }.getOrDefault(emptyList())
        if (communities.isEmpty()) return normal
        val merged = LinkedHashMap<String, V2TIMGroupInfo>()
        for (group in normal + communities) {
            val id = group.groupID?.trim().orEmpty()
            if (id.isEmpty()) continue
            merged[id] = group
            communityTypeCache[id] =
                group.groupType.equals("Community", ignoreCase = true) || looksLikeCommunityGroupId(id)
        }
        return merged.values.toList()
    }

    private suspend fun fetchJoinedGroupListRaw(): List<V2TIMGroupInfo> =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().getJoinedGroupList(
                object : V2TIMValueCallback<List<V2TIMGroupInfo>> {
                    override fun onSuccess(t: List<V2TIMGroupInfo>?) {
                        if (cont.isActive) cont.resume(t.orEmpty())
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("获取群列表失败($code): $desc"))
                        }
                    }
                }
            )
        }

    /** 支持话题的社群不在 getJoinedGroupList；与 Web/ShangLan 一样补拉。 */
    private suspend fun fetchJoinedCommunityListRaw(): List<V2TIMGroupInfo> =
        suspendCancellableCoroutine { cont ->
            runCatching {
                V2TIMManager.getCommunityManager().getJoinedCommunityList(
                    object : V2TIMValueCallback<List<V2TIMGroupInfo>> {
                        override fun onSuccess(t: List<V2TIMGroupInfo>?) {
                            if (cont.isActive) cont.resume(t.orEmpty())
                        }

                        override fun onError(code: Int, desc: String?) {
                            // 未开通话题社群/SDK 不支持时忽略，不影响普通群
                            if (cont.isActive) cont.resume(emptyList())
                        }
                    }
                )
            }.onFailure {
                if (cont.isActive) cont.resume(emptyList())
            }
        }

    suspend fun syncSelfProfile(
        nickname: String?,
        faceUrl: String?,
        allowType: Int? = null,
    ) = suspendCancellableCoroutine { cont ->
        val info = com.tencent.imsdk.v2.V2TIMUserFullInfo().apply {
            if (!nickname.isNullOrBlank()) setNickname(nickname)
            if (!faceUrl.isNullOrBlank()) setFaceUrl(faceUrl)
            if (allowType != null) {
                // 0=AllowAny 1=NeedConfirm 2=DenyAny，与业务库 / 腾讯 TIM 一致
                setAllowType(allowType)
            }
        }
        V2TIMManager.getInstance().setSelfInfo(
            info,
            object : V2TIMCallback {
                override fun onSuccess() {
                    if (cont.isActive) cont.resume(Unit)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resumeWithException(IllegalStateException("聊天资料同步失败（$code），请重试资料同步"))
                }
            }
        )
    }

    suspend fun sendFriendGreetingMessage(userId: String) {
        val uid = userId.trim()
        if (uid.isEmpty()) return
        runCatching {
            sendTextMessage(
                text = FRIEND_GREETING_TEXT,
                receiver = uid,
                groupId = null,
            )
        }
    }

    suspend fun fetchFriendRemark(userId: String): String {
        val friends = runCatching { fetchFriendList() }.getOrDefault(emptyList())
        return friends.firstOrNull { it.userID == userId }?.friendRemark.orEmpty()
    }

    suspend fun getGroupAttributes(groupId: String, keys: List<String>? = null): Map<String, String> =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().getGroupAttributes(
                groupId,
                keys,
                object : V2TIMValueCallback<Map<String, String>> {
                    override fun onSuccess(t: Map<String, String>?) {
                        if (cont.isActive) cont.resume(t.orEmpty())
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("获取群属性失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun setGroupAttributes(groupId: String, attributes: Map<String, String>) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().setGroupAttributes(
                groupId,
                HashMap(attributes),
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        // First set may need init
                        V2TIMManager.getGroupManager().initGroupAttributes(
                            groupId,
                            HashMap(attributes),
                            object : V2TIMCallback {
                                override fun onSuccess() {
                                    if (cont.isActive) cont.resume(Unit)
                                }

                                override fun onError(c2: Int, d2: String?) {
                                    if (cont.isActive) {
                                        cont.resumeWithException(Exception("设置群属性失败($code/$c2): $desc / $d2"))
                                    }
                                }
                            }
                        )
                    }
                }
            )
        }

    suspend fun fetchSelfGroupMemberRole(groupId: String): Int {
        return fetchGroupInfo(groupId)?.role ?: 0
    }

    /** 关闭「允许查看成员资料」后：只能查看群主/管理员，普通成员不可点开。 */
    fun canViewMemberProfile(profileAccessEnabled: Boolean, targetRole: Int): Boolean {
        if (profileAccessEnabled) return true
        return targetRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER ||
            targetRole == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN
    }

    fun isPrivilegedGroupRole(role: Int): Boolean =
        role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER ||
            role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN

    suspend fun fetchGroupMemberRoleMap(groupId: String): Map<String, Int> {
        return fetchGroupMembers(groupId).associate { it.userID to it.role }
    }

    suspend fun sendImageMessage(
        path: String,
        receiver: String?,
        groupId: String?,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage = sendCreated(
        V2TIMManager.getMessageManager().createImageMessage(path),
        receiver,
        groupId,
        onPrepared,
        onProgress,
    )

    suspend fun sendSoundMessage(
        path: String,
        durationSec: Int,
        receiver: String?,
        groupId: String?,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage = sendCreated(
        V2TIMManager.getMessageManager().createSoundMessage(path, durationSec),
        receiver,
        groupId,
        onPrepared,
        onProgress,
    )

    suspend fun sendVideoMessage(
        videoPath: String,
        type: String,
        durationSec: Int,
        snapshotPath: String,
        receiver: String?,
        groupId: String?,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
        fileName: String? = null,
        isActive: () -> Boolean = { true },
    ): V2TIMMessage {
        if (!isActive()) throw LinLanSendFailure("已取消，未发送", notDispatched = true)
        val size = LinLanCosLargeAttachment.fileSize(videoPath)
        if (LinLanCosLargeAttachment.shouldRouteToCos(size)) {
            val name = fileName ?: videoPath.substringAfterLast('/').ifBlank { "video.$type" }
            return LinLanCosLargeAttachment.uploadAndSend(
                path = videoPath,
                fileName = name,
                receiver = receiver,
                groupId = groupId,
                isVideo = true,
                onPrepared = onPrepared, onProgress = onProgress, isActive = isActive,
            )
        }
        return sendCreated(
            V2TIMManager.getMessageManager().createVideoMessage(videoPath, type, durationSec, snapshotPath),
            receiver,
            groupId,
            onPrepared,
            onProgress,
        )
    }

    /**
     * 社群（Community）不支持消息已读回执；强制开启会报
     * `The community group does not support message read receipt`。
     * 与 Web（GRP_COMMUNITY 不开回执）一致。
     */
    fun looksLikeCommunityGroupId(groupId: String?): Boolean {
        val id = groupId?.trim().orEmpty()
        if (id.isEmpty()) return false
        // 社群根 ID：@TGS#_xxx；话题会话：xxx@TOPIC#_yyy
        return id.startsWith("@TGS#_") || id.contains("@TOPIC#_")
    }

    suspend fun isCommunityGroup(groupId: String?): Boolean {
        val id = groupId?.trim().orEmpty()
        if (id.isEmpty()) return false
        if (looksLikeCommunityGroupId(id)) return true
        communityTypeCache[id]?.let { return it }
        val type = runCatching { fetchGroupInfo(id)?.groupType }.getOrNull().orEmpty()
        val isCommunity = type.equals("Community", ignoreCase = true)
        communityTypeCache[id] = isCommunity
        return isCommunity
    }

    private suspend fun sendCreated(
        message: V2TIMMessage,
        receiver: String?,
        groupId: String?,
        onPrepared: (V2TIMMessage) -> Unit = {},
        onProgress: (Int) -> Unit = {},
    ): V2TIMMessage {
        val owner = V2TIMManager.getInstance().loginUser.orEmpty()
        if (owner.isBlank()) throw LinLanSendFailure("聊天服务尚未登录", notDispatched = true)
        if (!groupId.isNullOrBlank()) {
            val permission = resolveGroupSendPermission(groupId)
            if (!permission.allowsSend) throw LinLanSendFailure(permission.blockedHint, notDispatched = true)
        }
        if (V2TIMManager.getInstance().loginUser != owner) throw LinLanSendFailure("账号已切换", notDispatched = true)
        val enableReadReceipt = groupId.isNullOrBlank() || !isCommunityGroup(groupId)
        return suspendCancellableCoroutine { cont ->
            message.isNeedReadReceipt = enableReadReceipt
            onPrepared(message)
            V2TIMManager.getMessageManager().sendMessage(
                message,
                receiver,
                groupId,
                V2TIMMessage.V2TIM_PRIORITY_NORMAL,
                false,
                com.balabalaim.gcloud.tim.push.appdev.push.LinLanOfflinePushInfo.message(owner, groupId, messagePreview(message)),
                object : V2TIMSendCallback<V2TIMMessage> {
                    override fun onProgress(progress: Int) = onProgress(progress.coerceIn(0, 100))

                    override fun onSuccess(msg: V2TIMMessage?) {
                        if (cont.isActive) cont.resume(msg ?: message)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(LinLanSendFailure(formatImSendError(code, desc), code = code))
                        }
                    }
                }
            )
        }
    }

    fun utf8ByteCount(text: String): Int = text.toByteArray(Charsets.UTF_8).size

    fun truncateUtf8(text: String, maxBytes: Int): String {
        if (utf8ByteCount(text) <= maxBytes) return text
        val sb = StringBuilder()
        var used = 0
        for (ch in text) {
            val piece = ch.toString().toByteArray(Charsets.UTF_8).size
            if (used + piece > maxBytes) break
            sb.append(ch)
            used += piece
        }
        return sb.toString()
    }

    /** Backend full notice (5000) + IM notification summary (≤400). */
    suspend fun updateGroupNotice(groupId: String, notice: String): String {
        val trimmed = notice.trim()
        if (utf8ByteCount(trimmed) > GROUP_NOTICE_MAX_BYTES) {
            throw Exception("群公告不能超过 $GROUP_NOTICE_MAX_BYTES 字节")
        }
        val saved = LinLanAppContainer.apiClient().saveGroupNotice(groupId, trimmed)
        val summary = saved.summary.trim()
        val imNotice = when {
            summary.isNotEmpty() -> truncateUtf8(summary, GROUP_NOTICE_IM_MAX_BYTES)
            trimmed.isEmpty() -> ""
            else -> truncateUtf8(trimmed, GROUP_NOTICE_IM_MAX_BYTES)
        }
        val info = V2TIMGroupInfo().apply {
            this.groupID = groupId
            notification = imNotice
        }
        setGroupInfo(info)
        return saved.content.ifBlank { trimmed }
    }

    suspend fun fetchGroupNoticeContent(groupId: String): String {
        runCatching {
            val result = LinLanAppContainer.apiClient().getGroupNotice(groupId)
            if (result.hasValue) return result.content.trim()
        }
        return fetchGroupInfo(groupId)?.notification?.trim().orEmpty()
    }

    suspend fun fetchGroupInfo(groupId: String): V2TIMGroupInfo? = suspendCancellableCoroutine { cont ->
        V2TIMManager.getGroupManager().getGroupsInfo(
            listOf(groupId),
            object : V2TIMValueCallback<List<V2TIMGroupInfoResult>> {
                override fun onSuccess(t: List<V2TIMGroupInfoResult>?) {
                    if (!cont.isActive) return
                    val result = t?.firstOrNull()
                    if (result == null || result.resultCode != 0 || result.groupInfo == null) {
                        cont.resumeWithException(LinLanSdkOperationError("群资料查询", result?.resultCode ?: -1, result?.resultMessage))
                    } else cont.resume(result.groupInfo)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resumeWithException(LinLanSdkOperationError("群资料查询", code, desc))
                }
            }
        )
    }

    suspend fun fetchGroupMembers(groupId: String): List<V2TIMGroupMemberFullInfo> =
        fetchAllGroupMembers(groupId)

    /** 分页拉取全部群成员（@ 选人、成员列表等用）。 */
    suspend fun fetchAllGroupMembers(groupId: String): List<V2TIMGroupMemberFullInfo> {
        val all = mutableListOf<V2TIMGroupMemberFullInfo>()
        var nextSeq = 0L
        var guard = 0
        do {
            val page = fetchGroupMemberPage(groupId, nextSeq)
            all += page.members
            nextSeq = page.nextSeq
            guard++
        } while (nextSeq != 0L && guard < 50)
        return all
    }

    data class GroupMemberPage(
        val members: List<V2TIMGroupMemberFullInfo>,
        val nextSeq: Long,
    )

    suspend fun fetchGroupMemberPage(
        groupId: String,
        nextSeq: Long,
    ): GroupMemberPage = suspendCancellableCoroutine { cont ->
        V2TIMManager.getGroupManager().getGroupMemberList(
            groupId,
            V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_FILTER_ALL,
            nextSeq,
            object : V2TIMValueCallback<V2TIMGroupMemberInfoResult> {
                override fun onSuccess(result: V2TIMGroupMemberInfoResult?) {
                    if (!cont.isActive) return
                    cont.resume(
                        GroupMemberPage(
                            members = result?.memberInfoList.orEmpty(),
                            nextSeq = result?.nextSeq ?: 0L,
                        ),
                    )
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("获取成员失败($code): $desc"))
                    }
                }
            },
        )
    }

    suspend fun muteGroupMember(groupId: String, userId: String, seconds: Int) =
        suspendCancellableCoroutine { cont ->
        permissionMemo.invalidateGroup(groupId)
            V2TIMManager.getGroupManager().muteGroupMember(
                groupId,
                userId,
                seconds,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("禁言失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun transferGroupOwner(groupId: String, newOwnerId: String) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().transferGroupOwner(
                groupId,
                newOwnerId,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("转让失败($code): $desc"))
                        }
                    }
                }
            )
        }

    suspend fun kickGroupMember(groupId: String, userIds: List<String>, reason: String = "") =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().kickGroupMember(
                groupId,
                userIds,
                reason,
                object : V2TIMValueCallback<List<V2TIMGroupMemberOperationResult>> {
                    override fun onSuccess(t: List<V2TIMGroupMemberOperationResult>?) {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("移出失败($code): $desc"))
                        }
                    }
                }
            )
        }

    enum class GroupJoinOutcome { JOINED, PENDING }
    suspend fun joinSharedGroup(groupId: String): GroupJoinOutcome {
        require(!groupId.contains("@TOPIC#")) { "请先加入话题所属的社群，再进入对应话题" }
        val owner = V2TIMManager.getInstance().loginUser
        val info = fetchGroupInfo(groupId) ?: throw LinLanSdkOperationError("群资料查询", -1)
        if (info.role > 0) return GroupJoinOutcome.JOINED
        require(info.groupType !in listOf("Work", "Private")) { "此类群需要已有群成员邀请，不能通过名片直接加入" }
        require(info.groupAddOpt != V2TIMGroupInfo.V2TIM_GROUP_ADD_FORBID) { "该群禁止主动加入，请联系群主" }
        suspendCancellableCoroutine<Unit> { cont ->
            V2TIMManager.getInstance().joinGroup(groupId, "通过群名片申请加入", object : V2TIMCallback {
                override fun onSuccess() { if (cont.isActive) cont.resume(Unit) }
                override fun onError(code: Int, desc: String?) { if (cont.isActive) cont.resumeWithException(LinLanSdkOperationError("申请入群", code, desc)) }
            })
        }
        if (owner != V2TIMManager.getInstance().loginUser) throw CancellationException("账号已切换")
        val joined = fetchGroupInfo(groupId)?.role?.let { it > 0 } == true
        notifyConversationsChanged()
        return if (joined) GroupJoinOutcome.JOINED else GroupJoinOutcome.PENDING
    }

    suspend fun inviteUsersToGroup(groupId: String, userIds: List<String>) {
        require((fetchGroupInfo(groupId)?.role ?: 0) > 0) { "尚未加入此群，不能邀请其他成员" }
        for (batch in userIds.filter { it.isNotBlank() }.distinct().chunked(20)) {
            inviteUsersToGroupBatch(groupId, batch)
        }
    }

    private suspend fun inviteUsersToGroupBatch(groupId: String, userIds: List<String>) =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getGroupManager().inviteUserToGroup(
                groupId,
                userIds,
                object : V2TIMValueCallback<List<com.tencent.imsdk.v2.V2TIMGroupMemberOperationResult>> {
                    override fun onSuccess(t: List<com.tencent.imsdk.v2.V2TIMGroupMemberOperationResult>?) {
                        if (!cont.isActive) return
                        val incomplete = t.orEmpty().filter { it.result != 1 && it.result != 2 }
                        if (incomplete.isNotEmpty()) {
                            cont.resumeWithException(Exception("部分好友未入群或正在等待确认，请在群设置中查看"))
                        } else cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception(formatImInviteError(code, desc)))
                        }
                    }
                }
            )
        }

    suspend fun fetchFriendList(): List<com.tencent.imsdk.v2.V2TIMFriendInfo> =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getFriendshipManager().getFriendList(
                object : V2TIMValueCallback<List<com.tencent.imsdk.v2.V2TIMFriendInfo>> {
                    override fun onSuccess(t: List<com.tencent.imsdk.v2.V2TIMFriendInfo>?) {
                        if (cont.isActive) cont.resume(t.orEmpty())
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("获取好友失败($code): $desc"))
                        }
                    }
                }
            )
        }

    data class SelectableContact(
        val userId: String,
        val displayName: String,
    )

    fun friendUserId(friend: V2TIMFriendInfo): String =
        friend.userID?.takeIf { it.isNotBlank() }
            ?: friend.userProfile?.userID?.takeIf { it.isNotBlank() }
            ?: ""

    fun friendDisplayName(friend: V2TIMFriendInfo): String {
        val uid = friendUserId(friend)
        return friend.friendRemark?.takeIf { it.isNotBlank() }
            ?: friend.userProfile?.nickName?.takeIf { it.isNotBlank() }
            ?: uid
    }

    /** 建群 / 邀请：好友 + 单聊会话，过滤空 userID，避免 LazyColumn key 崩溃导致整页空白。 */
    suspend fun fetchSelectableContacts(): List<SelectableContact> {
        var friends: List<V2TIMFriendInfo> = emptyList()
        repeat(3) { attempt ->
            friends = runCatching { fetchFriendList() }.getOrDefault(emptyList())
            if (friends.isNotEmpty() || !V2TIMManager.getInstance().loginUser.isNullOrBlank()) {
                if (friends.isNotEmpty() || attempt >= 2) return@repeat
            }
            delay(400L * (attempt + 1))
        }
        val map = linkedMapOf<String, SelectableContact>()
        friends.forEach { friend ->
            val id = friendUserId(friend)
            if (id.isNotBlank()) {
                map[id] = SelectableContact(id, friendDisplayName(friend))
            }
        }
        runCatching { fetchConversations() }.getOrDefault(emptyList())
            .filter { it.type == V2TIMConversation.V2TIM_C2C }
            .forEach { conv ->
                val id = conv.userID?.takeIf { it.isNotBlank() }
                    ?: conv.conversationID?.removePrefix("c2c_")?.takeIf { it.isNotBlank() }
                    ?: ""
                if (id.isNotBlank() && !map.containsKey(id)) {
                    map[id] = SelectableContact(
                        userId = id,
                        displayName = conv.showName?.takeIf { it.isNotBlank() } ?: id,
                    )
                }
            }
        return map.values.sortedBy { it.displayName.lowercase() }
    }

    suspend fun isFriend(userId: String): Boolean {
        if (userId.isBlank()) return false
        return runCatching { fetchFriendList().any { it.userID == userId } }.getOrDefault(false)
    }

    data class UserBrief(
        val userId: String,
        val nickName: String,
        val faceUrl: String?,
    )

    suspend fun fetchUserBrief(userId: String): UserBrief? =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getInstance().getUsersInfo(
                listOf(userId),
                object : V2TIMValueCallback<List<V2TIMUserFullInfo>> {
                    override fun onSuccess(t: List<V2TIMUserFullInfo>?) {
                        val u = t?.firstOrNull()
                        if (cont.isActive) {
                            cont.resume(
                                u?.let {
                                    UserBrief(
                                        userId = it.userID.orEmpty(),
                                        nickName = it.nickName.orEmpty(),
                                        faceUrl = it.faceUrl,
                                    )
                                },
                            )
                        }
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(null)
                    }
                },
            )
        }

    fun displayNameOfMessage(msg: V2TIMMessage): String {
        return msg.friendRemark?.takeIf { it.isNotBlank() }
            ?: msg.nameCard?.takeIf { it.isNotBlank() }
            ?: msg.nickName?.takeIf { it.isNotBlank() }
            ?: msg.sender.orEmpty()
    }

    fun imageUrlOf(msg: V2TIMMessage, preferLarge: Boolean = false): String? {
        val images = msg.imageElem?.imageList ?: return null
        if (preferLarge) {
            images.firstOrNull { it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_LARGE }?.url
                ?.let { return it }
            images.firstOrNull { it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_ORIGIN }?.url
                ?.let { return it }
        }
        return images.firstOrNull { it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_THUMB }?.url
            ?: images.firstOrNull()?.url
    }

    data class ContactCardPayload(
        val userId: String,
        val nickname: String,
        val isGroup: Boolean = false,
        val groupId: String = "",
        val groupName: String = "",
    )

    fun parseCustomCard(msg: V2TIMMessage): ContactCardPayload? {
        if (msg.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_CUSTOM) return null
        val raw = msg.customElem?.data?.toString(Charsets.UTF_8).orEmpty()
        val card = RuiSharedCard.parse(raw) ?: return null
        return ContactCardPayload(if (card.isGroup) "" else card.id, if (card.isGroup) "" else card.name,
            card.isGroup, if (card.isGroup) card.id else "", if (card.isGroup) card.name else "")
    }

    suspend fun resolveVideoSnapshot(msg: V2TIMMessage): String? {
        val elem = msg.videoElem ?: return null
        elem.snapshotPath?.takeIf { java.io.File(it).exists() }?.let { return it }
        return suspendCancellableCoroutine { cont ->
            elem.getSnapshotUrl(object : V2TIMValueCallback<String> {
                override fun onSuccess(t: String?) {
                    if (cont.isActive) cont.resume(t)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) cont.resume(null)
                }
            })
        }
    }

    suspend fun resolveVideoPlayable(context: android.content.Context, msg: V2TIMMessage): String? {
        return resolveVideoPlayable(context, msg) { }
    }

    suspend fun resolveVideoPlayable(
        context: android.content.Context,
        msg: V2TIMMessage,
        onProgress: (Int) -> Unit,
    ): String? {
        val elem = msg.videoElem ?: return null
        elem.videoPath?.takeIf { java.io.File(it).exists() }?.let {
            onProgress(100)
            return it
        }
        val cache = java.io.File(context.cacheDir, "chat_media")
        cache.mkdirs()
        val uuid = elem.videoUUID?.ifBlank { null } ?: msg.msgID ?: System.currentTimeMillis().toString()
        val out = java.io.File(cache, "video_$uuid.mp4")
        if (out.exists() && out.length() > 0 &&
            (elem.videoSize <= 0 || out.length() == elem.videoSize.toLong())
        ) {
            onProgress(100)
            return out.absolutePath
        }
        return suspendCancellableCoroutine { cont ->
            elem.downloadVideo(
                out.absolutePath,
                object : com.tencent.imsdk.v2.V2TIMDownloadCallback {
                    override fun onProgress(progressInfo: com.tencent.imsdk.v2.V2TIMElem.V2ProgressInfo?) {
                        onProgress(downloadPercent(progressInfo))
                    }
                    override fun onSuccess() {
                        onProgress(100)
                        if (cont.isActive) cont.resume(out.absolutePath)
                    }

                    override fun onError(code: Int, desc: String?) {
                        elem.getVideoUrl(object : V2TIMValueCallback<String> {
                            override fun onSuccess(t: String?) {
                                if (cont.isActive) cont.resume(t)
                            }

                            override fun onError(c2: Int, d2: String?) {
                                if (cont.isActive) cont.resume(null)
                            }
                        })
                    }
                },
            )
        }
    }

    suspend fun resolveFileLocal(context: android.content.Context, msg: V2TIMMessage): String? {
        return resolveFileLocal(context, msg) { }
    }

    suspend fun resolveFileLocal(
        context: android.content.Context,
        msg: V2TIMMessage,
        onProgress: (Int) -> Unit,
    ): String? {
        val elem = msg.fileElem ?: return null
        elem.path?.takeIf { java.io.File(it).exists() }?.let {
            onProgress(100)
            return it
        }
        val cache = java.io.File(context.cacheDir, "chat_media")
        cache.mkdirs()
        val name = LinLanReliability.safeCacheName(
            elem.fileName?.ifBlank { null }
                ?: (elem.uuid?.ifBlank { null } ?: msg.msgID ?: "file"),
        )
        val out = java.io.File(cache, "file_$name")
        if (out.exists() && out.length() > 0 &&
            (elem.fileSize <= 0 || out.length() == elem.fileSize.toLong())
        ) {
            onProgress(100)
            return out.absolutePath
        }
        return suspendCancellableCoroutine { cont ->
            elem.downloadFile(
                out.absolutePath,
                object : com.tencent.imsdk.v2.V2TIMDownloadCallback {
                    override fun onProgress(progressInfo: com.tencent.imsdk.v2.V2TIMElem.V2ProgressInfo?) {
                        onProgress(downloadPercent(progressInfo))
                    }
                    override fun onSuccess() {
                        onProgress(100)
                        if (cont.isActive) cont.resume(out.absolutePath)
                    }

                    override fun onError(code: Int, desc: String?) {
                        elem.getUrl(object : V2TIMValueCallback<String> {
                            override fun onSuccess(t: String?) {
                                if (cont.isActive) cont.resume(t)
                            }

                            override fun onError(c2: Int, d2: String?) {
                                if (cont.isActive) cont.resume(null)
                            }
                        })
                    }
                },
            )
        }
    }

    suspend fun resolveSoundLocal(context: android.content.Context, msg: V2TIMMessage): String? {
        return resolveSoundLocal(context, msg) { }
    }

    suspend fun resolveSoundLocal(
        context: android.content.Context,
        msg: V2TIMMessage,
        onProgress: (Int) -> Unit,
    ): String? {
        val elem = msg.soundElem ?: return null
        elem.path?.takeIf { java.io.File(it).exists() }?.let {
            onProgress(100)
            return it
        }
        val cache = java.io.File(context.cacheDir, "chat_media")
        cache.mkdirs()
        val uuid = elem.uuid?.ifBlank { null } ?: msg.msgID ?: System.currentTimeMillis().toString()
        val out = java.io.File(cache, "sound_$uuid.m4a")
        if (out.exists() && out.length() > 0 &&
            (elem.dataSize <= 0 || out.length() == elem.dataSize.toLong())
        ) {
            onProgress(100)
            return out.absolutePath
        }
        return suspendCancellableCoroutine { cont ->
            elem.downloadSound(
                out.absolutePath,
                object : com.tencent.imsdk.v2.V2TIMDownloadCallback {
                    override fun onProgress(progressInfo: com.tencent.imsdk.v2.V2TIMElem.V2ProgressInfo?) {
                        onProgress(downloadPercent(progressInfo))
                    }
                    override fun onSuccess() {
                        onProgress(100)
                        if (cont.isActive) cont.resume(out.absolutePath)
                    }

                    override fun onError(code: Int, desc: String?) {
                        elem.getUrl(object : V2TIMValueCallback<String> {
                            override fun onSuccess(t: String?) {
                                if (cont.isActive) cont.resume(t)
                            }

                            override fun onError(c2: Int, d2: String?) {
                                if (cont.isActive) cont.resume(null)
                            }
                        })
                    }
                },
            )
        }
    }

    suspend fun resolveImageLocal(
        context: android.content.Context,
        msg: V2TIMMessage,
        onProgress: (Int) -> Unit = {},
    ): String? {
        val elem = msg.imageElem ?: return null
        elem.path?.takeIf { java.io.File(it).exists() }?.let {
            onProgress(100)
            return it
        }
        val image = elem.imageList.orEmpty().firstOrNull {
            it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_ORIGIN
        } ?: elem.imageList.orEmpty().firstOrNull {
            it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_LARGE
        } ?: elem.imageList.orEmpty().firstOrNull() ?: return null
        val cache = java.io.File(context.cacheDir, "chat_media").also { it.mkdirs() }
        val uuid = image.uuid?.ifBlank { null } ?: msg.msgID ?: System.currentTimeMillis().toString()
        val out = java.io.File(cache, "image_$uuid.jpg")
        if (out.exists() && out.length() > 0 &&
            (image.size <= 0 || out.length() == image.size.toLong())
        ) {
            onProgress(100)
            return out.absolutePath
        }
        return suspendCancellableCoroutine { cont ->
            image.downloadImage(
                out.absolutePath,
                object : com.tencent.imsdk.v2.V2TIMDownloadCallback {
                    override fun onProgress(progressInfo: com.tencent.imsdk.v2.V2TIMElem.V2ProgressInfo?) {
                        onProgress(downloadPercent(progressInfo))
                    }

                    override fun onSuccess() {
                        onProgress(100)
                        if (cont.isActive) cont.resume(out.absolutePath)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(image.url)
                    }
                },
            )
        }
    }

    fun isMediaCached(context: android.content.Context, msg: V2TIMMessage): Boolean {
        when (msg.elemType) {
            V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> {
                msg.imageElem?.path?.takeIf { java.io.File(it).exists() }?.let { return true }
                val image = msg.imageElem?.imageList.orEmpty().firstOrNull {
                    it.type == com.tencent.imsdk.v2.V2TIMImageElem.V2TIM_IMAGE_TYPE_ORIGIN
                } ?: msg.imageElem?.imageList.orEmpty().firstOrNull() ?: return false
                val uuid = image.uuid?.ifBlank { null } ?: msg.msgID ?: return false
                return java.io.File(context.cacheDir, "chat_media/image_$uuid.jpg").let {
                    it.exists() && it.length() > 0 && (image.size <= 0 || it.length() == image.size.toLong())
                }
            }
            V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> {
                msg.videoElem?.videoPath?.takeIf { java.io.File(it).exists() }?.let { return true }
                val elem = msg.videoElem ?: return false
                val uuid = elem.videoUUID?.ifBlank { null } ?: msg.msgID ?: return false
                return java.io.File(context.cacheDir, "chat_media/video_$uuid.mp4").let {
                    it.exists() && it.length() > 0 && (elem.videoSize <= 0 || it.length() == elem.videoSize.toLong())
                }
            }
            V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                msg.fileElem?.path?.takeIf { java.io.File(it).exists() }?.let { return true }
                val elem = msg.fileElem ?: return false
                val name = LinLanReliability.safeCacheName(
                    elem.fileName?.ifBlank { null }
                        ?: (elem.uuid?.ifBlank { null } ?: msg.msgID ?: return false),
                )
                return java.io.File(context.cacheDir, "chat_media/file_$name").let {
                    it.exists() && it.length() > 0 && (elem.fileSize <= 0 || it.length() == elem.fileSize.toLong())
                }
            }
        }
        return false
    }

    private fun downloadPercent(info: com.tencent.imsdk.v2.V2TIMElem.V2ProgressInfo?): Int {
        val total = info?.totalSize ?: 0L
        val current = info?.currentSize ?: 0L
        if (total <= 0L) return 0
        return ((current * 100L) / total).toInt().coerceIn(0, 99)
    }

    fun isImageFileName(name: String?): Boolean {
        val n = name.orEmpty().lowercase()
        return n.endsWith(".png") || n.endsWith(".jpg") || n.endsWith(".jpeg") ||
            n.endsWith(".gif") || n.endsWith(".webp") || n.startsWith("image")
    }

    private val outgoingFriendActions = java.util.concurrent.ConcurrentHashMap.newKeySet<String>()
    suspend fun areBothWayFriends(userId: String): Boolean = suspendCancellableCoroutine { cont ->
        V2TIMManager.getFriendshipManager().checkFriend(listOf(userId),V2TIMFriendInfo.V2TIM_FRIEND_TYPE_BOTH,
            object: V2TIMValueCallback<List<com.tencent.imsdk.v2.V2TIMFriendCheckResult>> {
                override fun onSuccess(value: List<com.tencent.imsdk.v2.V2TIMFriendCheckResult>?) {
                    if (cont.isActive) cont.resume(value.orEmpty().any { it.userID==userId && it.resultCode==0 && it.resultType==com.tencent.imsdk.v2.V2TIMFriendCheckResult.V2TIM_FRIEND_RELATION_TYPE_BOTH_WAY })
                }
                override fun onError(code: Int, desc: String?) { if (cont.isActive) cont.resumeWithException(Exception("好友关系尚未确认($code)")) }
            })
    }

    suspend fun addFriend(userId: String, wording: String = "请求添加你为好友"): AddFriendOutcome {
        val accountScope=LinLanFriendOwnership.sessionKey()
        val key="$accountScope:$userId"
        if (!outgoingFriendActions.add(key)) throw Exception("该好友申请正在处理中")
        try {
        val outcome = suspendCancellableCoroutine { cont ->
            val app = V2TIMFriendAddApplication(userId).apply {
                setAddWording(wording)
                setAddType(V2TIMFriendInfo.V2TIM_FRIEND_TYPE_BOTH)
            }
            V2TIMManager.getFriendshipManager().addFriend(
                app,
                object : V2TIMValueCallback<V2TIMFriendOperationResult> {
                    override fun onSuccess(t: V2TIMFriendOperationResult?) {
                        if (!cont.isActive) return
                        when (t?.resultCode ?: 30539) {
                            0 -> cont.resume(AddFriendOutcome.AddedDirectly)
                            30539 -> cont.resume(AddFriendOutcome.PendingConfirm)
                            30515 -> cont.resume(AddFriendOutcome.AlreadyFriends)
                            else -> {
                                val info = t?.resultInfo?.trim().orEmpty()
                                cont.resumeWithException(
                                    Exception(info.ifBlank { "添加好友失败(${t?.resultCode})" })
                                )
                            }
                        }
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("添加好友失败($code): $desc"))
                        }
                    }
                }
            )
        }
        if (LinLanFriendOwnership.sessionKey()!=accountScope) throw CancellationException("账号已变化")
        if (outcome == AddFriendOutcome.PendingConfirm) LinLanFriendOwnership.requestPending(userId)
        else {
            LinLanFriendOwnership.established(userId)
            if (outcome == AddFriendOutcome.AddedDirectly) runCatching { sendFriendGreetingMessage(userId) }
        }
        return outcome
        } finally { outgoingFriendActions.remove(key) }
    }

    suspend fun fetchIncomingFriendApplications(): List<V2TIMFriendApplication> =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getFriendshipManager().getFriendApplicationList(
                object : V2TIMValueCallback<V2TIMFriendApplicationResult> {
                    override fun onSuccess(result: V2TIMFriendApplicationResult?) {
                        val apps = result?.friendApplicationList
                            ?.filter { it.type == V2TIMFriendApplication.V2TIM_FRIEND_APPLICATION_COME_IN }
                            ?: emptyList()
                        if (cont.isActive) cont.resume(apps)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("获取好友申请失败($code): $desc"))
                        }
                    }
                }
            )
        }

    private val friendApplicationActions = java.util.concurrent.ConcurrentHashMap.newKeySet<String>()

    suspend fun acceptFriendApplication(app: V2TIMFriendApplication) = handleFriendApplication(app, true)
    suspend fun rejectFriendApplication(app: V2TIMFriendApplication) = handleFriendApplication(app, false)

    private suspend fun handleFriendApplication(app: V2TIMFriendApplication, accept: Boolean) {
        val owner = V2TIMManager.getInstance().loginUser.orEmpty()
        val accountScope=LinLanFriendOwnership.sessionKey()
        val userId = app.userID.orEmpty()
        val key = "$accountScope:$userId:${app.addTime}"
        if (!friendApplicationActions.add(key)) throw Exception("该申请正在处理中")
        var accepted = false
        try {
            try {
                suspendCancellableCoroutine<Unit> { cont ->
                    val callback = object : V2TIMValueCallback<V2TIMFriendOperationResult> {
                        override fun onSuccess(result: V2TIMFriendOperationResult?) {
                            if (!cont.isActive) return
                            val code = result?.resultCode ?: -1
                            if (code == 0) cont.resume(Unit)
                            else cont.resumeWithException(LinLanSendFailure(result?.resultInfo ?: "申请处理结果未确认", code))
                        }
                        override fun onError(code: Int, desc: String?) {
                            if (cont.isActive) cont.resumeWithException(LinLanSendFailure(desc ?: "申请处理失败", code))
                        }
                    }
                    if (accept) V2TIMManager.getFriendshipManager().acceptFriendApplication(app, V2TIMFriendApplication.V2TIM_FRIEND_ACCEPT_AGREE_AND_ADD, callback)
                    else V2TIMManager.getFriendshipManager().refuseFriendApplication(app, callback)
                }
                accepted = accept
            } catch (error: LinLanSendFailure) {
                if (error.code != 30614) throw error
                if (LinLanFriendOwnership.sessionKey() != accountScope) throw CancellationException("账号已切换")
                val alreadyFriend = areBothWayFriends(userId)
                if (!accept || !alreadyFriend) throw Exception(if (alreadyFriend) "已添加为好友，请刷新列表" else "该申请已失效，请重新申请")
                accepted=accept
            }
            if (LinLanFriendOwnership.sessionKey()!=accountScope) throw CancellationException("账号已变化")
            if (accepted) {LinLanFriendOwnership.established(userId);runCatching { sendFriendGreetingMessage(userId) }}
        } finally {
            friendApplicationActions.remove(key)
            if (V2TIMManager.getInstance().loginUser == owner) { notifyConversationsChanged(); notifyFriendsChanged() }
        }
    }

    /** Public 群自定义 ID：随机 9 位或 10 位数字（与 jizhangTimkotlin 一致，便于搜索/分享）。 */
    fun randomPublicGroupId(): String {
        val useTenDigits = kotlin.random.Random.nextBoolean()
        return if (useTenDigits) {
            (1_000_000_000L + kotlin.random.Random.nextLong(9_000_000_000L)).toString()
        } else {
            (100_000_000 + kotlin.random.Random.nextInt(900_000_000)).toString()
        }
    }

    /**
     * 保留：创建 Public（陌生人社交群，约 2000 人量级）。
     * 默认建群请用 [createCommunityGroup]。
     */
    suspend fun createPublicGroup(name: String, memberUserIds: List<String>): String {
        var lastError: Exception? = null
        repeat(5) {
            val customId = randomPublicGroupId()
            try {
                return createPublicGroupInternal(name, memberUserIds, customId)
            } catch (e: Exception) {
                lastError = e
                val msg = e.message.orEmpty()
                // 10021/10025：群 ID 已被占用
                val idConflict = msg.contains("10021") || msg.contains("10025") || msg.contains("已存在")
                if (!idConflict) throw e
            }
        }
        throw lastError ?: Exception("创建群聊失败")
    }

    private suspend fun createPublicGroupInternal(
        name: String,
        memberUserIds: List<String>,
        customGroupId: String,
    ): String = suspendCancellableCoroutine { cont ->
        val info = V2TIMGroupInfo().apply {
            groupType = "Public"
            groupName = name
            groupID = customGroupId
            // 默认允许邀请好友入群，避免 10007 this group does not enable invite
            groupAddOpt = V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
            groupApproveOpt = V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
        }
        val members = memberUserIds.map { uid ->
            V2TIMCreateGroupMemberInfo().apply { setUserID(uid) }
        }
        V2TIMManager.getGroupManager().createGroup(
            info,
            if (members.isEmpty()) null else members,
            object : V2TIMValueCallback<String> {
                override fun onSuccess(groupId: String?) {
                    if (cont.isActive) {
                        if (groupId.isNullOrBlank()) {
                            cont.resumeWithException(Exception("创建群失败：空 groupId"))
                        } else {
                            cont.resume(groupId)
                        }
                    }
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception(formatImCreateGroupError(code, desc)))
                    }
                }
            },
        )
    }

    /**
     * 默认建群：创建社群（Community，可达十万级）。
     * 不启用话题，消息直接发到社群 ID；已读回执在发送侧会自动关闭。
     */
    class PartialGroupCreation(val groupId: String, cause: Throwable) :
        Exception("群已创建，但追加成员未全部完成。请进入群聊后在群设置中继续邀请：${cause.message}", cause)

    suspend fun createCommunityGroup(name: String, memberUserIds: List<String>): String {
        val ids = memberUserIds.filter { it.isNotBlank() }.distinct()
        require(ids.size <= 100) { "最多选择 100 位好友" }
        val groupId = suspendCancellableCoroutine { cont ->
            val info = V2TIMGroupInfo().apply {
                groupType = "Community"
                groupName = name
                setSupportTopic(false)
                groupAddOpt = V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
                groupApproveOpt = V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY
            }
            val members = ids.take(20).map { uid ->
                V2TIMCreateGroupMemberInfo().apply { setUserID(uid) }
            }
            V2TIMManager.getCommunityManager().createCommunity(
                info,
                if (members.isEmpty()) null else members,
                object : V2TIMValueCallback<String> {
                    override fun onSuccess(id: String?) {
                        if (!cont.isActive) return
                        if (id.isNullOrBlank()) {
                            cont.resumeWithException(Exception("创建群失败：空 groupId"))
                            return
                        }
                        cont.resume(id)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception(formatImCreateGroupError(code, desc)))
                        }
                    }
                },
            )
        }
        communityTypeCache[groupId] = true
        try {
            inviteUsersToGroup(groupId, ids.drop(20))
        } catch (error: Exception) {
            throw PartialGroupCreation(groupId, error)
        }
        // 主动创建会话资料，避免消息列表里看不到新建社群。
        runCatching { fetchConversationOrNull("group_$groupId") }
        notifyConversationsChanged()
        return groupId
    }

    fun formatImCreateGroupError(code: Int, desc: String?): String {
        val raw = desc?.trim().orEmpty()
        val lower = raw.lowercase()
        return when {
            raw.contains("未开启建群权限") ||
                lower.contains("can create group") ||
                lower.contains("create group permission") ||
                (code == 10101 && (raw.contains("建群") || lower.contains("create"))) ->
                "当前账号未开启建群权限，请联系管理员"
            raw.contains("跨归属") || lower.contains("cross-owner") ->
                "跨归属建群不允许"
            raw.any { it in '\u4e00'..'\u9fff' } ->
                if (raw.startsWith("创建")) raw else "创建群聊失败：$raw"
            code != 0 -> "创建群聊失败（$code）"
            else -> "创建群聊失败"
        }
    }

    suspend fun setGroupInfo(info: V2TIMGroupInfo) = suspendCancellableCoroutine { cont ->
        V2TIMManager.getGroupManager().setGroupInfo(
            info,
            object : V2TIMCallback {
                override fun onSuccess() {
                    if (cont.isActive) cont.resume(Unit)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("更新群设置失败($code): $desc"))
                    }
                }
            }
        )
    }

    suspend fun quitGroup(groupId: String) = suspendCancellableCoroutine { cont ->
        V2TIMManager.getInstance().quitGroup(
            groupId,
            object : V2TIMCallback {
                override fun onSuccess() {
                    notifyConversationsChanged()
                    if (cont.isActive) cont.resume(Unit)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("退出群失败($code): $desc"))
                    }
                }
            },
        )
    }

    suspend fun dismissGroup(groupId: String) = suspendCancellableCoroutine { cont ->
        V2TIMManager.getInstance().dismissGroup(
            groupId,
            object : V2TIMCallback {
                override fun onSuccess() {
                    notifyConversationsChanged()
                    if (cont.isActive) cont.resume(Unit)
                }

                override fun onError(code: Int, desc: String?) {
                    if (cont.isActive) {
                        cont.resumeWithException(Exception("解散群失败($code): $desc"))
                    }
                }
            },
        )
    }

    suspend fun setGroupMute(groupId: String, muted: Boolean) =
        suspendCancellableCoroutine { cont ->
            val opt = if (muted) {
                V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
            } else {
                V2TIMMessage.V2TIM_RECEIVE_MESSAGE
            }
            V2TIMManager.getMessageManager().setGroupReceiveMessageOpt(
                groupId,
                opt,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("设置免打扰失败($code): $desc"))
                        }
                    }
                },
            )
        }

    suspend fun isGroupMuted(groupId: String): Boolean =
        suspendCancellableCoroutine { cont ->
            V2TIMManager.getConversationManager().getConversation(
                "group_$groupId",
                object : V2TIMValueCallback<V2TIMConversation> {
                    override fun onSuccess(t: V2TIMConversation?) {
                        if (cont.isActive) {
                            cont.resume(t?.recvOpt == V2TIMMessage.V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE)
                        }
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) cont.resume(false)
                    }
                },
            )
        }

    suspend fun muteAllGroupMembers(groupId: String, muted: Boolean) =
        suspendCancellableCoroutine { cont ->
        permissionMemo.invalidateGroup(groupId)
            V2TIMManager.getGroupManager().muteAllGroupMembers(
                groupId,
                muted,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("全员禁言失败($code): $desc"))
                        }
                    }
                },
            )
        }

    suspend fun setGroupMemberRole(groupId: String, userId: String, role: Int) =
        suspendCancellableCoroutine { cont ->
        permissionMemo.invalidateGroup(groupId)
            V2TIMManager.getGroupManager().setGroupMemberRole(
                groupId,
                userId,
                role,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("设置角色失败($code): $desc"))
                        }
                    }
                },
            )
        }

    suspend fun setSelfGroupNameCard(groupId: String, nameCard: String) =
        suspendCancellableCoroutine { cont ->
            val selfId = V2TIMManager.getInstance().loginUser
            if (selfId.isNullOrBlank()) {
                cont.resumeWithException(Exception("未登录"))
                return@suspendCancellableCoroutine
            }
            val info = V2TIMGroupMemberFullInfo().apply {
                userID = selfId
                setNameCard(nameCard)
            }
            V2TIMManager.getGroupManager().setGroupMemberInfo(
                groupId,
                info,
                object : V2TIMCallback {
                    override fun onSuccess() {
                        if (cont.isActive) cont.resume(Unit)
                    }

                    override fun onError(code: Int, desc: String?) {
                        if (cont.isActive) {
                            cont.resumeWithException(Exception("设置群昵称失败($code): $desc"))
                        }
                    }
                },
            )
        }

    fun isGroupMemberMuted(member: V2TIMGroupMemberFullInfo): Boolean {
        val until = member.muteUntil
        return until > 0 && until * 1000L > System.currentTimeMillis()
    }

    fun canManageGroup(role: Int): Boolean =
        role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_OWNER ||
            role == V2TIMGroupMemberFullInfo.V2TIM_GROUP_MEMBER_ROLE_ADMIN ||
            role == 400 ||
            role == 300

    data class GroupSendPermission(
        val allowsSend: Boolean,
        val blockedHint: String = "",
        val canRetry: Boolean = false,
    ) {
        companion object {
            val Allowed = GroupSendPermission(true)
        }
    }

    /** 全员禁言（普通成员）或个人禁言时不允许发言/转发。 */
    private val permissionMemo = PermissionMemo<GroupSendPermission>(scope)
    private fun announceGroupSendPermission(groupID: String) {
        permissionMemo.invalidateGroup(groupID)
        _groupSendPolicyChanged.tryEmit(groupID)
    }
    suspend fun resolveGroupSendPermission(groupId: String): GroupSendPermission {
        val owner = V2TIMManager.getInstance().loginUser.orEmpty()
        val appID = LinLanRuntimeConfig.sdkAppId
        if (owner.isBlank()) return GroupSendPermission(false, "聊天服务尚未登录")
        return try {
            kotlinx.coroutines.withTimeout(8_000) {
                permissionMemo.getCurrent("$appID:$owner:$groupId",
                    isCurrent = { owner == V2TIMManager.getInstance().loginUser && appID == LinLanRuntimeConfig.sdkAppId },
                    retry = { true }) { loadGroupSendPermission(groupId) }
            }
        } catch (_: kotlinx.coroutines.TimeoutCancellationException) {
            kotlinx.coroutines.currentCoroutineContext().ensureActive()
            GroupSendPermission(false, "群权限暂未确认，点此重试", canRetry = true)
        } catch (error: CancellationException) { throw error }
          catch (_: Exception) { GroupSendPermission(false, "群权限暂未确认，点此重试", canRetry = true) }
    }
    private suspend fun loadGroupSendPermission(groupId: String): GroupSendPermission {
        val id = groupId.trim()
        if (id.isEmpty()) return GroupSendPermission(false, "群ID无效")
        val owner = V2TIMManager.getInstance().loginUser.orEmpty()
        if (owner.isBlank()) return GroupSendPermission(false, "聊天服务尚未登录")
        return try {
            val info = fetchGroupInfo(id) ?: throw IllegalStateException("群资料待同步")
            val members: List<V2TIMGroupMemberFullInfo> = suspendCancellableCoroutine { cont ->
                V2TIMManager.getGroupManager().getGroupMembersInfo(id, listOf(owner), object : V2TIMValueCallback<List<V2TIMGroupMemberFullInfo>> {
                    override fun onSuccess(value: List<V2TIMGroupMemberFullInfo>?) { if (cont.isActive) cont.resume(value.orEmpty()) }
                    override fun onError(code: Int, desc: String?) { if (cont.isActive) cont.resumeWithException(Exception("群权限查询失败($code)")) }
                })
            }
            val member = members.firstOrNull { it.userID == owner }
                ?: throw IllegalStateException("群成员身份待同步")
            when {
                V2TIMManager.getInstance().loginUser != owner -> GroupSendPermission(false, "账号已切换")
                isGroupMemberMuted(member) -> GroupSendPermission(false, "已被个人禁言")
                info.isAllMuted && !canManageGroup(member.role) -> GroupSendPermission(false, "全体禁言")
                else -> GroupSendPermission.Allowed
            }
        } catch (error: Exception) { throw error }
    }

    fun groupAddOptLabel(opt: Int): String = when (opt) {
        V2TIMGroupInfo.V2TIM_GROUP_ADD_ANY -> "允许任何人"
        V2TIMGroupInfo.V2TIM_GROUP_ADD_AUTH -> "需要验证"
        V2TIMGroupInfo.V2TIM_GROUP_ADD_FORBID -> "禁止加群"
        else -> "未知"
    }

    fun formatImInviteError(code: Int, desc: String?): String {
        val raw = desc?.trim().orEmpty()
        val lower = raw.lowercase()
        return when {
            lower.contains("does not enable invite") -> "该群未开启邀请进群（$code）"
            code == 10007 -> "邀请入群失败（10007）：操作权限不足，请核对成员身份、角色及群类型"
            code == 10101 || lower.contains("cross-owner") ->
                "无法邀请该用户入群（归属不一致），请确认双方属于同一运营方"
            raw.any { it in '\u4e00'..'\u9fff' } -> "邀请入群失败：$raw"
            code != 0 -> "邀请入群失败（$code）"
            else -> "邀请入群失败"
        }
    }

    fun formatImSendError(code: Int, desc: String?): String {
        val raw = desc?.trim().orEmpty()
        val lower = raw.lowercase()
        when {
            lower.contains("read receipt") && lower.contains("community") ->
                return "社群不支持已读回执，请升级后重试"
            lower.contains("community group does not support") ->
                return "社群暂不支持发送此类消息"
            code == 10017 || lower.contains("muted") && lower.contains("send") ->
                return "您已被禁言，无法发送消息"
            lower.contains("only group member can send") ||
                lower.contains("only group member") ->
                return "你不是该群成员，无法发送消息"
            lower.contains("does not enable invite") ->
                return "该群未开启邀请进群"
            code == 10007 ->
                return "你不是该群成员或没有发送权限"
            code == 10010 ->
                return "群组不存在或已解散"
            code == 6017 ->
                return "参数无效，请稍后重试"
            code == 6014 || code == 6206 ->
                return "登录已失效，请重新登录"
            code == 6004 || lower.contains("network") || lower.contains("timeout") ->
                return "网络异常，请稍后重试"
        }
        if (raw.any { it in '\u4e00'..'\u9fff' }) {
            return "发送失败：$raw"
        }
        return if (code != 0) "发送失败，请稍后重试（$code）" else "发送失败，请稍后重试"
    }

    private fun groupCreateTipText(payload: JSONObject): String {
        val opUser = payload.optString("opUser").trim().ifBlank { "成员" }
        val content = payload.optString("content").trim().ifBlank {
            when (payload.optInt("cmd", 0)) {
                1 -> "创建社群"
                else -> "创建群组"
            }
        }
        return "$opUser $content"
    }

    fun isGroupCreateTipMessage(msg: V2TIMMessage): Boolean {
        if (msg.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_CUSTOM) return false
        val raw = msg.customElem?.data?.toString(Charsets.UTF_8).orEmpty()
        return runCatching {
            JSONObject(raw).optString("businessID") == GROUP_CREATE_BUSINESS_ID
        }.getOrDefault(false)
    }

    fun groupCreateTipText(msg: V2TIMMessage): String {
        val raw = msg.customElem?.data?.toString(Charsets.UTF_8).orEmpty()
        return runCatching { groupCreateTipText(JSONObject(raw)) }
            .getOrDefault("创建群组")
    }

    fun isFriendGreetingTipMessage(msg: V2TIMMessage): Boolean {
        if (msg.elemType != V2TIMMessage.V2TIM_ELEM_TYPE_TEXT) return false
        // 仅单聊问候语；群消息不当 tip
        if (!msg.groupID.isNullOrBlank()) return false
        val text = msg.textElem?.text?.trim().orEmpty()
        if (text.isEmpty()) return false
        if (FRIEND_GREETING_TEXTS.contains(text)) return true
        return text.contains("通过你的好友验证") ||
            text.contains("已成为好友") ||
            text.contains("accepted your friend request")
    }

    fun friendGreetingTipText(msg: V2TIMMessage): String =
        msg.textElem?.text?.trim().orEmpty().ifBlank { "你们已成为好友" }

    fun isGroupTipsMessage(msg: V2TIMMessage): Boolean =
        msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_GROUP_TIPS

    fun shouldShowGroupTipsInChat(msg: V2TIMMessage): Boolean {
        if (!isGroupTipsMessage(msg)) return true
        return groupTipsTipText(msg).isNotBlank()
    }

    fun groupTipsTipText(msg: V2TIMMessage): String {
        val tips = msg.groupTipsElem ?: return ""
        fun memberName(m: com.tencent.imsdk.v2.V2TIMGroupMemberInfo?): String {
            val card = m?.nameCard?.trim().orEmpty()
            if (card.isNotEmpty()) return card
            val nick = m?.nickName?.trim().orEmpty()
            if (nick.isNotEmpty()) return nick
            return m?.userID?.trim().orEmpty().ifBlank { "成员" }
        }
        val memberNames = tips.memberList.orEmpty().map { memberName(it) }.filter { it.isNotBlank() }.joinToString("、")
        val opName = memberName(tips.opMember).ifBlank { "有人" }
        return when (tips.type) {
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_JOIN ->
                if (memberNames.isBlank()) "新成员加入了群聊" else "$memberNames 加入了群聊"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_INVITE ->
                if (memberNames.isBlank()) "$opName 邀请成员加入群聊" else "$opName 邀请 $memberNames 加入群聊"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_QUIT -> "$opName 退出了群聊"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_KICKED ->
                if (memberNames.isBlank()) "$opName 移除了群成员" else "$opName 将 $memberNames 移出群聊"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_SET_ADMIN ->
                if (memberNames.isBlank()) "$opName 设置了管理员" else "$memberNames 已被设为管理员"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_CANCEL_ADMIN ->
                if (memberNames.isBlank()) "$opName 取消了管理员" else "$memberNames 已被取消管理员"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_GROUP_INFO_CHANGE -> {
                tips.groupChangeInfoList.orEmpty().mapNotNull { change ->
                    when (change.type) {
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_NAME ->
                            "$opName 修改群名称为「${change.value.orEmpty()}」"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_INTRODUCTION ->
                            "$opName 更新了群介绍"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_NOTIFICATION ->
                            "$opName 更新了群公告"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_FACE_URL ->
                            "$opName 修改了群头像"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_OWNER ->
                            "群主已转让给 ${change.value.orEmpty().ifBlank { "新群主" }}"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_SHUT_UP_ALL ->
                            if (change.boolValue) "$opName 开启了全体禁言" else "$opName 关闭了全体禁言"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_GROUP_ADD_OPT ->
                            "$opName 更新了主动加群方式"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_GROUP_APPROVE_OPT ->
                            "$opName 更新了邀请进群方式"
                        V2TIMGroupChangeInfo.V2TIM_GROUP_INFO_CHANGE_TYPE_CUSTOM -> {
                            when (change.key.orEmpty()) {
                                LinLanGroupPolicy.MEMBER_PROFILE_ACCESS_KEY ->
                                    if (change.value == "1") "$opName 允许查看成员资料" else "$opName 禁止查看成员资料"
                                LinLanGroupPolicy.MUTE_ADMIN_ALSO_KEY ->
                                    "$opName 更新了管理员禁言设置"
                                LinLanGroupPolicy.VOICE_CALL_KEY ->
                                    "$opName 更新了群语音通话设置"
                                LinLanGroupPolicy.VIDEO_CALL_KEY ->
                                    "$opName 更新了群视频通话设置"
                                else -> "$opName 更新了群设置"
                            }
                        }
                        else -> null
                    }
                }.distinct().joinToString("；").ifBlank { "$opName 更新了群资料" }
            }
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_MEMBER_INFO_CHANGE -> {
                val opId = tips.opMember?.userID.orEmpty()
                tips.memberChangeInfoList.orEmpty().map { change ->
                    val target = change.userID.orEmpty().ifBlank { "群成员" }
                    when {
                        change.muteTime > 0 -> "$opName 禁言了 $target"
                        target == opId -> "$opName 修改了群昵称"
                        else -> "$opName 取消了 $target 的禁言"
                    }
                }.distinct().joinToString("；").ifBlank { "$opName 更新了群成员资料" }
            }
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_TOPIC_INFO_CHANGE -> "$opName 更新了群话题"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_PINNED_MESSAGE_ADDED -> "$opName 置顶了一条消息"
            V2TIMGroupTipsElem.V2TIM_GROUP_TIPS_TYPE_PINNED_MESSAGE_DELETED -> "$opName 取消了一条置顶消息"
            else -> "群聊信息已更新"
        }
    }

    fun messagePreview(msg: V2TIMMessage): String {
        if (shouldSilentlyHideRevokedMessage(msg)) return ""
        if (isRevokedMessage(msg)) return revokedTipText(msg)
        if (isGroupTipsMessage(msg)) {
            return groupTipsTipText(msg).ifBlank { "" }
        }
        return when (msg.elemType) {
            V2TIMMessage.V2TIM_ELEM_TYPE_TEXT ->
                LinLanEmojiCatalog.previewText(msg.textElem?.text.orEmpty())
            V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> "[图片]"
            V2TIMMessage.V2TIM_ELEM_TYPE_SOUND -> "[语音]"
            V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> "[视频]"
            V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                val name = msg.fileElem?.fileName?.takeIf { it.isNotBlank() }
                if (name != null) "[文件] $name" else "[文件]"
            }
            V2TIMMessage.V2TIM_ELEM_TYPE_MERGER -> {
                val title = msg.mergerElem?.title?.takeIf { it.isNotBlank() }
                if (title != null) "[聊天记录] $title" else "[聊天记录]"
            }
            V2TIMMessage.V2TIM_ELEM_TYPE_CUSTOM -> {
                LinLanRtcSignalingPayload.parseFromMessage(msg)?.displayText(msg.isSelf)?.let { return it }
                val raw = msg.customElem?.data?.toString(Charsets.UTF_8).orEmpty()
                runCatching {
                    val o = JSONObject(raw)
                    when (o.optString("businessID")) {
                        GROUP_CREATE_BUSINESS_ID -> groupCreateTipText(o)
                        CARD_BUSINESS_ID ->
                            "[个人名片] ${o.optString("nickname").ifBlank { o.optString("userId") }}"
                        GROUP_CARD_BUSINESS_ID ->
                            "[群名片] ${o.optString("groupName").ifBlank { o.optString("groupId") }}"
                        else ->
                            msg.customElem?.description?.ifBlank { "[自定义消息]" } ?: "[自定义消息]"
                    }
                }.getOrDefault(msg.customElem?.description?.ifBlank { "[自定义消息]" } ?: "[自定义消息]")
            }
            else -> "[消息]"
        }
    }

    fun copyableText(msg: V2TIMMessage): String = when (msg.elemType) {
        V2TIMMessage.V2TIM_ELEM_TYPE_TEXT -> msg.textElem?.text.orEmpty()
        else -> messagePreview(msg)
    }
}
