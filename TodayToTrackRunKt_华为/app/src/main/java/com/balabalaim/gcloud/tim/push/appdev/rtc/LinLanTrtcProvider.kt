package com.balabalaim.gcloud.tim.push.appdev.rtc

import android.os.Handler
import android.os.Looper
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.tencent.rtmp.ui.TXCloudVideoView
import com.tencent.trtc.TRTCCloud
import com.tencent.trtc.TRTCCloudDef
import com.tencent.trtc.TRTCCloudListener

/**
 * Aligns with iOS chenDao_TRTCProvider — LiteAVSDK_TRTC direct, no TUICallKit.
 */
class LinLanTrtcProvider private constructor() {
    private val mainHandler = Handler(Looper.getMainLooper())
    private var cloud: TRTCCloud? = null
    private var remoteUserId: String? = null

    var isInRoom: Boolean = false
        private set

    var onRemoteUserJoined: ((String) -> Unit)? = null
    var onRemoteUserLeft: ((String) -> Unit)? = null
    var onRemoteVideoAvailable: ((String, Boolean) -> Unit)? = null

    private val listener = object : TRTCCloudListener() {
        override fun onEnterRoom(result: Long) {
            android.util.Log.i("RuiCall", "TRTC enterRoom result=$result")
        }

        override fun onUserAudioAvailable(userId: String?, available: Boolean) {
            android.util.Log.i("RuiCall", "remote audio available=$available")
        }

        override fun onRemoteUserEnterRoom(userId: String?) {
            val uid = userId ?: return
            notifyJoined(uid)
        }

        override fun onRemoteUserLeaveRoom(userId: String?, reason: Int) {
            val uid = userId ?: return
            notifyLeft(uid)
        }

        override fun onUserVideoAvailable(userId: String?, available: Boolean) {
            android.util.Log.i("RuiCall", "remote video available=$available")
            val uid = userId ?: return
            runOnMain { onRemoteVideoAvailable?.invoke(uid, available) }
        }
    }

    fun attach(context: android.content.Context) {
        if (cloud != null) return
        cloud = TRTCCloud.sharedInstance(context.applicationContext).also {
            it.setListener(listener)
        }
    }

    fun enterRoom(
        roomId: String,
        userId: String,
        userSig: String,
        mediaType: LinLanRtcCallMediaType,
        localPreview: TXCloudVideoView?,
        remoteView: TXCloudVideoView?,
        roomType: String = "string",
    ) {
        runOnMain {
            val trtc = cloud ?: return@runOnMain
            val params = TRTCCloudDef.TRTCParams().apply {
                sdkAppId = LinLanRuntimeConfig.sdkAppId
                this.userId = userId
                this.userSig = userSig
                if (roomType == "number") this.roomId = roomId.toLong().toInt() else strRoomId = roomId
            }
            val scene = if (mediaType == LinLanRtcCallMediaType.VIDEO) {
                TRTCCloudDef.TRTC_APP_SCENE_VIDEOCALL
            } else {
                TRTCCloudDef.TRTC_APP_SCENE_AUDIOCALL
            }
            trtc.enterRoom(params, scene)
            trtc.startLocalAudio(TRTCCloudDef.TRTC_AUDIO_QUALITY_DEFAULT)
            if (mediaType == LinLanRtcCallMediaType.VIDEO && localPreview != null) {
                trtc.startLocalPreview(true, localPreview)
            } else {
                trtc.stopLocalPreview()
            }
            remoteUserId = null
            isInRoom = true
            remoteView?.tag = "remote"
        }
    }

    fun exitRoom() {
        runOnMain {
            val trtc = cloud ?: return@runOnMain
            trtc.stopLocalPreview()
            trtc.stopLocalAudio()
            trtc.stopAllRemoteView()
            trtc.exitRoom()
            isInRoom = false
            remoteUserId = null
        }
    }

    fun setMicEnabled(enabled: Boolean) {
        cloud?.muteLocalAudio(!enabled)
    }

    fun setSpeakerEnabled(enabled: Boolean) {
        runCatching {
            cloud?.setAudioRoute(
                if (enabled) TRTCCloudDef.TRTC_AUDIO_ROUTE_SPEAKER
                else TRTCCloudDef.TRTC_AUDIO_ROUTE_EARPIECE
            )
        }
    }

    fun switchCamera() {
        runCatching {
            val dm = cloud?.deviceManager ?: return
            dm.switchCamera(!dm.isFrontCamera)
        }
    }

    fun setVideoEnabled(enabled: Boolean) {
        runCatching {
            cloud?.muteLocalVideo(TRTCCloudDef.TRTC_VIDEO_STREAM_TYPE_BIG, !enabled)
        }.onFailure {
            runCatching { cloud?.muteLocalVideo(!enabled) }
        }
    }

    fun bindRemoteView(view: TXCloudVideoView, userId: String) {
        runOnMain {
            remoteUserId = userId
            cloud?.startRemoteView(userId, TRTCCloudDef.TRTC_VIDEO_STREAM_TYPE_BIG, view)
        }
    }

    fun rebindMediaViews(
        localPreview: TXCloudVideoView?,
        remoteView: TXCloudVideoView?,
        mediaType: LinLanRtcCallMediaType,
        remoteUserId: String?,
    ) {
        runOnMain {
            if (mediaType == LinLanRtcCallMediaType.VIDEO && localPreview != null) {
                cloud?.startLocalPreview(true, localPreview)
            }
            if (!remoteUserId.isNullOrBlank() && remoteView != null) {
                bindRemoteView(remoteView, remoteUserId)
            }
        }
    }

    fun currentRemoteUserId(): String? = remoteUserId

    private fun notifyJoined(userId: String) {
        runOnMain { onRemoteUserJoined?.invoke(userId) }
    }

    private fun notifyLeft(userId: String) {
        runOnMain { onRemoteUserLeft?.invoke(userId) }
    }

    private fun runOnMain(block: () -> Unit) {
        if (Looper.myLooper() == Looper.getMainLooper()) block()
        else mainHandler.post(block)
    }

    companion object {
        val shared: LinLanTrtcProvider by lazy { LinLanTrtcProvider() }
    }
}
