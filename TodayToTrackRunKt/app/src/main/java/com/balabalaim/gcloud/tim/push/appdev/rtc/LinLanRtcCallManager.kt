package com.balabalaim.gcloud.tim.push.appdev.rtc

import android.content.Context
import android.content.Intent
import android.os.Handler
import android.os.Looper
import android.widget.Toast
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.tencent.imsdk.v2.V2TIMCallback
import com.tencent.imsdk.v2.V2TIMManager
import com.tencent.imsdk.v2.V2TIMSignalingListener
import com.tencent.imsdk.v2.V2TIMSignalingManager
import com.tencent.imsdk.v2.V2TIMMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Aligns with iOS chenDao_RTCCallManager:
 * IM signaling invite/accept/reject + LiteAVSDK_TRTC room.
 */
object LinLanRtcCallManager {
    private const val INVITE_TIMEOUT_SEC = 45

    private val mainHandler = Handler(Looper.getMainLooper())
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val provider = LinLanTrtcProvider.shared

    private var appContext: Context? = null
    private var selfUserId: String = ""
    private var selfDisplayName: String = "我"
    private var selfAvatarUrl: String? = null
    private var userSig: String = ""
    private var boundSignaling: V2TIMSignalingManager? = null
    private var boundListener: V2TIMSignalingListener? = null
    private var boundAppId = 0
    private var signalingEpoch = 0L
    private var callEpoch = 0L
    private var ringTimeout: Runnable? = null
    private var callConnectedAtMs: Long? = null
    private var endingBySelf = false

    private val _status = MutableStateFlow(LinLanRtcCallStatus.NONE)
    val status: StateFlow<LinLanRtcCallStatus> = _status.asStateFlow()

    private val _context = MutableStateFlow<LinLanRtcCallContext?>(null)
    val callContext: StateFlow<LinLanRtcCallContext?> = _context.asStateFlow()

    private val _toast = MutableStateFlow<String?>(null)
    val toast: StateFlow<String?> = _toast.asStateFlow()

    var activeUi: LinLanRtcCallUiBridge? = null

    private fun signalingListener(epoch: Long) = object : V2TIMSignalingListener() {
        private fun dispatch(block: () -> Unit) { mainHandler.post { if (epoch == signalingEpoch && selfUserId.isNotBlank() && V2TIMManager.getInstance().loginUser == selfUserId) block() } }
        override fun onReceiveNewInvitation(
            inviteID: String?,
            inviter: String?,
            groupID: String?,
            inviteeList: MutableList<String>?,
            data: String?,
        ) {
            if (!groupID.isNullOrBlank()) return
            val id = inviteID ?: return
            val from = inviter ?: return
            dispatch { handleNewInvitation(id, from, data) }
        }

        override fun onInviteeAccepted(inviteID: String?, invitee: String?, data: String?) {
            val id = inviteID ?: return
            dispatch { if (invitee == _context.value?.peer?.userId) handleInviteAccepted(id) }
        }

        override fun onInviteeRejected(inviteID: String?, invitee: String?, data: String?) {
            val id = inviteID ?: return
            dispatch { handleInviteRejected(id, LinLanRtcSignalingPayload.parse(data)?.action == LinLanRtcSignalingAction.BUSY) }
        }

        override fun onInvitationCancelled(inviteID: String?, inviter: String?, data: String?) {
            val id = inviteID ?: return
            dispatch { handleInviteCancelled(id) }
        }

        override fun onInvitationTimeout(inviteID: String?, inviteeList: MutableList<String>?) {
            val id = inviteID ?: return
            dispatch { handleInviteTimeout(id) }
        }
    }

    fun setup(context: Context) {
        appContext = context.applicationContext
        provider.attach(context.applicationContext)
        val sdkAppId = com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager.currentSdkAppId()
        if (sdkAppId > 0) {
            val manager = V2TIMManager.getSignalingManager()
            if (boundSignaling !== manager || boundAppId != sdkAppId || boundListener == null) {
                unbindSignaling()
                val listener = signalingListener(++signalingEpoch)
                manager.addSignalingListener(listener)
                boundSignaling = manager; boundListener = listener; boundAppId = sdkAppId
            }
        }
        provider.onRemoteUserJoined = { uid -> handleRemoteJoined(uid) }
        provider.onRemoteUserLeft = { uid -> if (uid == _context.value?.peer?.userId) finishCall(LinLanRtcCallEndReason.HANGUP) }
        provider.onRemoteVideoAvailable = { uid, available ->
            if (uid == _context.value?.peer?.userId) {
                if (available) activeUi?.remoteVideoView()?.let { provider.bindRemoteView(it, uid) }
                activeUi?.onRemoteVideoAvailabilityChanged(available)
            }
        }
    }

    fun unbindSignaling() {
        ++signalingEpoch
        boundListener?.let { boundSignaling?.removeSignalingListener(it) }
        boundListener = null; boundSignaling = null; boundAppId = 0
    }

    fun onIncomingMessage(message: V2TIMMessage) {
        val payload = LinLanRtcSignalingPayload.parseFromMessage(message) ?: return
        val ctx = _context.value ?: return
        if (message.isSelf || message.sender != ctx.peer.userId || payload.roomId != ctx.roomId || payload.roomType != ctx.roomType) return
        if (payload.inviteId != null && payload.inviteId != ctx.inviteId) return
        if (payload.action == LinLanRtcSignalingAction.HANGUP) mainHandler.post { if (_context.value?.inviteId == ctx.inviteId) finishCall(LinLanRtcCallEndReason.HANGUP) }
    }

    fun updateSelfProfile(name: String?, avatarUrl: String?) {
        if (!name.isNullOrBlank()) selfDisplayName = name
        selfAvatarUrl = avatarUrl
    }

    fun refreshLoginState(userId: String, sig: String) {
        if (selfUserId.isNotBlank() && selfUserId != userId) { unbindSignaling(); resetCall(); if (provider.isInRoom) provider.exitRoom() }
        selfUserId = userId
        userSig = sig
        appContext?.let { setup(it) }
    }

    fun startCall(
        activityContext: Context,
        peer: RtcCallPeer,
        mediaType: LinLanRtcCallMediaType,
    ) {
        setup(activityContext)
        scope.launch {
            val session = LinLanAppContainer.sessionStore().currentSession()
            refreshLoginState(session.userId, session.userSig)
            if (selfUserId.isBlank()) {
                toast("请先登录")
                return@launch
            }
            if (!com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager.isReadyFor(session.userId, session.sdkAppId)) {
                toast("聊天尚未连接，请稍后重试")
                return@launch
            }
            if (_status.value != LinLanRtcCallStatus.NONE) {
                toast("当前正在通话中")
                return@launch
            }
            if (peer.userId.isBlank() || peer.userId == selfUserId) return@launch
            beginOutgoingCall(activityContext, peer, mediaType)
        }
    }

    private fun beginOutgoingCall(
        activityContext: Context,
        peer: RtcCallPeer,
        mediaType: LinLanRtcCallMediaType,
    ) {
        val epoch = ++callEpoch
        val roomId = makeRoomId(selfUserId, peer.userId)
        val data = encodeSignaling(LinLanRtcSignalingAction.INVITE, mediaType, roomId) ?: return
        updateStatus(LinLanRtcCallStatus.WAITING)
        val inviteId = V2TIMManager.getSignalingManager().invite(
            peer.userId,
            data,
            false,
            com.balabalaim.gcloud.tim.push.appdev.push.LinLanOfflinePushInfo.message(selfDisplayName, null, if (mediaType == LinLanRtcCallMediaType.VIDEO) "视频来电" else "语音来电"),
            INVITE_TIMEOUT_SEC,
            object : V2TIMCallback {
                override fun onSuccess() = Unit
                override fun onError(code: Int, desc: String?) {
                    mainHandler.post {
                        if (epoch != callEpoch) return@post
                        finishCall(LinLanRtcCallEndReason.CANCEL)
                        toast(desc ?: "发起通话失败")
                    }
                }
            },
        )
        if (inviteId.isNullOrBlank()) {
            resetCall()
            toast("发起通话失败")
            return
        }
        _context.value = LinLanRtcCallContext(
            inviteId = inviteId,
            roomId = roomId,
            peer = peer,
            mediaType = mediaType,
            role = LinLanRtcCallRole.CALLER,
        )
        appContext?.let { LinLanRtcRingtonePlayer.play(it, LinLanRtcRingtonePlayer.Kind.OUTGOING) }
        presentCallUi(activityContext)
        startRingTimeout()
    }

    fun acceptIncomingCall() {
        val ctx = _context.value ?: return
        if (_status.value != LinLanRtcCallStatus.WAITING || ctx.role != LinLanRtcCallRole.CALLED) return
        val epoch = callEpoch
        val data = encodeSignaling(LinLanRtcSignalingAction.ACCEPT, ctx.mediaType, ctx.roomId)
        V2TIMManager.getSignalingManager().accept(
            ctx.inviteId,
            data,
            object : V2TIMCallback {
                override fun onSuccess() {
                    mainHandler.post { if (epoch == callEpoch && _context.value?.inviteId == ctx.inviteId) enterRtcAfterAccepted() }
                }

                override fun onError(code: Int, desc: String?) {
                    mainHandler.post { if (epoch == callEpoch && _context.value?.inviteId == ctx.inviteId) toast(desc ?: "接听失败") }
                }
            },
        )
    }

    fun rejectIncomingCall() {
        val ctx = _context.value ?: return
        if (ctx.role != LinLanRtcCallRole.CALLED) return
        val data = encodeSignaling(LinLanRtcSignalingAction.REJECT, ctx.mediaType, ctx.roomId)
        V2TIMManager.getSignalingManager().reject(ctx.inviteId, data, null)
        finishCall(LinLanRtcCallEndReason.REJECT)
    }

    fun cancelOutgoingCall() {
        val ctx = _context.value ?: return
        if (ctx.role != LinLanRtcCallRole.CALLER || _status.value != LinLanRtcCallStatus.WAITING) return
        val data = encodeSignaling(LinLanRtcSignalingAction.CANCEL, ctx.mediaType, ctx.roomId)
        V2TIMManager.getSignalingManager().cancel(ctx.inviteId, data, null)
        finishCall(LinLanRtcCallEndReason.CANCEL)
    }

    fun hangup() {
        endingBySelf = true
        finishCall(LinLanRtcCallEndReason.HANGUP)
    }

    fun connectedElapsedSec(): Int {
        val start = callConnectedAtMs ?: return 0
        return ((System.currentTimeMillis() - start) / 1000L).toInt().coerceAtLeast(0)
    }

    fun enterRtcAfterAccepted() {
        val ctx = _context.value ?: return
        val epoch = callEpoch
        if (selfUserId.isBlank() || userSig.isBlank()) {
            scope.launch {
                val session = LinLanAppContainer.sessionStore().currentSession()
                if (epoch != callEpoch || _context.value?.inviteId != ctx.inviteId || session.userId != selfUserId) return@launch
                refreshLoginState(session.userId, session.userSig)
                if (selfUserId.isBlank() || userSig.isBlank()) {
                    toast("UserSig 无效")
                    finishCall(LinLanRtcCallEndReason.HANGUP)
                    return@launch
                }
                doEnterRoom(ctx)
            }
            return
        }
        doEnterRoom(ctx)
    }

    private fun doEnterRoom(ctx: LinLanRtcCallContext) {
        if (_context.value?.inviteId != ctx.inviteId) return
        val epoch = callEpoch
        updateStatus(LinLanRtcCallStatus.ACCEPT)
        if (callConnectedAtMs == null) {
            callConnectedAtMs = System.currentTimeMillis()
        }
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        val ui = activeUi
        if (ui == null) {
            appContext?.let { presentCallUi(it) }
        }
        val bridge = activeUi
        val join = {
            val local = bridge?.localPreviewView()
            val remote = bridge?.remoteVideoView()
            if (epoch == callEpoch && _context.value?.inviteId == ctx.inviteId && activeUi === bridge && remote != null && !provider.isInRoom) {
                appContext?.let { androidx.core.content.ContextCompat.startForegroundService(it, Intent(it, CallForegroundService::class.java)) }
                provider.enterRoom(
                    roomId = ctx.roomId,
                    userId = selfUserId,
                    userSig = userSig,
                    mediaType = ctx.mediaType,
                    localPreview = local,
                    remoteView = remote,
                    roomType = ctx.roomType,
                )
                provider.setSpeakerEnabled(ctx.mediaType == LinLanRtcCallMediaType.VIDEO)
                stopRingTimeout()
            }
        }
        if (bridge?.isViewReady() == true) join()
        else bridge?.onViewReady = join
        bridge?.refreshFromManager()
    }

    private fun handleNewInvitation(inviteId: String, inviter: String, data: String?) {
        val payload = LinLanRtcSignalingPayload.parse(data) ?: return
        if (payload.action != LinLanRtcSignalingAction.INVITE || payload.roomId.isBlank() || inviter == selfUserId) return
        if (_context.value?.inviteId == inviteId) return
        if (_status.value != LinLanRtcCallStatus.NONE) {
            V2TIMManager.getSignalingManager().reject(inviteId, payload.copy(action = LinLanRtcSignalingAction.BUSY).toJson(), null)
            return
        }
        val peerName = payload.peerName?.takeIf { it.isNotBlank() } ?: inviter
        val peer = RtcCallPeer(inviter, peerName, payload.peerAvatar)
        ++callEpoch
        _context.value = LinLanRtcCallContext(
            inviteId = inviteId,
            roomId = payload.roomId,
            roomType = payload.roomType,
            peer = peer,
            mediaType = payload.mediaType,
            role = LinLanRtcCallRole.CALLED,
        )
        updateStatus(LinLanRtcCallStatus.WAITING)
        startRingTimeout()
        appContext?.let {
            LinLanRtcRingtonePlayer.play(it, LinLanRtcRingtonePlayer.Kind.INCOMING)
            if (!com.balabalaim.gcloud.tim.push.appdev.notify.LinLanMessageAlertCenter.appInForeground) {
                com.balabalaim.gcloud.tim.push.appdev.notify.LinLanLocalNotifyHelper.showIncomingCallNotification(
                    context = it,
                    callerName = peerName,
                    isVideo = payload.mediaType == LinLanRtcCallMediaType.VIDEO,
                )
            }
            presentCallUi(it)
        }
    }

    private fun handleInviteAccepted(inviteId: String) {
        val ctx = _context.value ?: return
        if (ctx.inviteId != inviteId || ctx.role != LinLanRtcCallRole.CALLER) return
        enterRtcAfterAccepted()
    }

    private fun handleInviteRejected(inviteId: String, busy: Boolean = false) {
        if (_context.value?.inviteId != inviteId) return
        finishCall(if (busy) LinLanRtcCallEndReason.BUSY else LinLanRtcCallEndReason.REJECT)
    }

    private fun handleInviteCancelled(inviteId: String) {
        if (_context.value?.inviteId != inviteId) return
        finishCall(LinLanRtcCallEndReason.CANCEL)
    }

    private fun handleInviteTimeout(inviteId: String) {
        if (_context.value?.inviteId != inviteId) return
        finishCall(LinLanRtcCallEndReason.NO_RESPONSE)
    }

    private fun handleRemoteJoined(userId: String) {
        val peerId = _context.value?.peer?.userId ?: return
        if (userId != peerId) return
        activeUi?.remoteVideoView()?.let { provider.bindRemoteView(it, userId) }
        activeUi?.onRemoteUserJoined()
    }

    private fun presentCallUi(context: Context) {
        if (activeUi != null) return
        LinLanCallFloatWindow.hide()
        val intent = Intent(context, LinLanRtcCallActivity::class.java).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        context.startActivity(intent)
    }

    /** 从悬浮窗点回全屏通话页。 */
    fun restoreCallUi(context: Context) {
        if (_status.value == LinLanRtcCallStatus.NONE || _context.value == null) {
            LinLanCallFloatWindow.hide()
            return
        }
        presentCallUi(context)
    }

    private fun finishCall(reason: LinLanRtcCallEndReason) {
        ++callEpoch
        val ctx = _context.value
        val connectedAt = callConnectedAtMs
        val sendHangupRecord = endingBySelf &&
            reason == LinLanRtcCallEndReason.HANGUP &&
            ctx != null &&
            connectedAt != null
        val durationSec = if (connectedAt != null) {
            ((System.currentTimeMillis() - connectedAt) / 1000L).toInt().coerceAtLeast(1)
        } else {
            0
        }
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        callConnectedAtMs = null
        endingBySelf = false
        LinLanCallFloatWindow.hide()
        if (provider.isInRoom) provider.exitRoom()
        activeUi?.onViewReady = null
        activeUi?.dismissForEnd(reason)
        activeUi = null
        _context.value = null
        updateStatus(LinLanRtcCallStatus.NONE)
        if (sendHangupRecord && ctx != null) {
            val payload = LinLanRtcSignalingPayload(
                action = LinLanRtcSignalingAction.HANGUP,
                mediaType = ctx.mediaType,
                roomId = ctx.roomId,
                durationSec = durationSec,
                roomType = ctx.roomType, inviteId = ctx.inviteId,
            )
            val owner = selfUserId
            val appId = boundAppId
            scope.launch {
                if (owner != V2TIMManager.getInstance().loginUser || appId != boundAppId) return@launch
                runCatching {
                    com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager.sendRtcCallRecord(
                        ctx.peer.userId,
                        payload,
                    )
                }
            }
        }
        val message = when (reason) {
            LinLanRtcCallEndReason.HANGUP -> "通话已结束"
            LinLanRtcCallEndReason.REJECT -> "对方已拒绝"
            LinLanRtcCallEndReason.NO_RESPONSE -> "对方无应答"
            LinLanRtcCallEndReason.BUSY -> "对方忙线中"
            LinLanRtcCallEndReason.CANCEL -> "已取消"
        }
        toast(message)
    }

    private fun resetCall() {
        ++callEpoch
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        callConnectedAtMs = null
        endingBySelf = false
        activeUi?.onViewReady = null
        activeUi?.dismissForEnd(LinLanRtcCallEndReason.CANCEL)
        activeUi = null
        LinLanCallFloatWindow.hide()
        _context.value = null
        updateStatus(LinLanRtcCallStatus.NONE)
    }

    private fun updateStatus(status: LinLanRtcCallStatus) {
        if (status == LinLanRtcCallStatus.NONE) appContext?.let { it.stopService(Intent(it, CallForegroundService::class.java)) }
        _status.value = status
        activeUi?.refreshFromManager()
    }

    private fun startRingTimeout() {
        stopRingTimeout()
        val task = Runnable {
            val ctx = _context.value
            if (_status.value == LinLanRtcCallStatus.WAITING && ctx != null) {
                val data = encodeSignaling(LinLanRtcSignalingAction.NO_RESPONSE, ctx.mediaType, ctx.roomId)
                if (ctx.role == LinLanRtcCallRole.CALLER) V2TIMManager.getSignalingManager().cancel(ctx.inviteId, data, null)
                else V2TIMManager.getSignalingManager().reject(ctx.inviteId, data, null)
                finishCall(LinLanRtcCallEndReason.NO_RESPONSE)
            }
        }
        ringTimeout = task
        mainHandler.postDelayed(task, INVITE_TIMEOUT_SEC * 1000L)
    }

    private fun stopRingTimeout() {
        ringTimeout?.let { mainHandler.removeCallbacks(it) }
        ringTimeout = null
    }

    private fun makeRoomId(caller: String, callee: String): String {
        return "rui_" + java.util.UUID.randomUUID().toString().replace("-", "")
    }

    private fun encodeSignaling(
        action: LinLanRtcSignalingAction,
        mediaType: LinLanRtcCallMediaType,
        roomId: String,
        durationSec: Int? = null,
    ): String? = LinLanRtcSignalingPayload(
        action = action,
        mediaType = mediaType,
        roomId = roomId,
        peerName = selfDisplayName,
        peerAvatar = selfAvatarUrl,
        durationSec = durationSec,
        roomType = _context.value?.roomType ?: "string",
        inviteId = _context.value?.inviteId,
    ).toJson()

    private fun toast(msg: String) {
        _toast.value = msg
        appContext?.let {
            Toast.makeText(it, msg, Toast.LENGTH_SHORT).show()
        }
    }

    suspend fun logoutCleanup() {
        if (_status.value != LinLanRtcCallStatus.NONE) {
            when { _status.value == LinLanRtcCallStatus.ACCEPT -> hangup()
                _context.value?.role == LinLanRtcCallRole.CALLER -> cancelOutgoingCall()
                else -> rejectIncomingCall() }
        }
        unbindSignaling(); selfUserId = ""; userSig = ""
    }
}

interface LinLanRtcCallUiBridge {
    fun localPreviewView(): com.tencent.rtmp.ui.TXCloudVideoView?
    fun remoteVideoView(): com.tencent.rtmp.ui.TXCloudVideoView?
    fun isViewReady(): Boolean
    var onViewReady: (() -> Unit)?
    fun refreshFromManager()
    fun onRemoteUserJoined()
    fun onRemoteVideoAvailabilityChanged(available: Boolean) {}
    fun dismissForEnd(reason: LinLanRtcCallEndReason)
}
