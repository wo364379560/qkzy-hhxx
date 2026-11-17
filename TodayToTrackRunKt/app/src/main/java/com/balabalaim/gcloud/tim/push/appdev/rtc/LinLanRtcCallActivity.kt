package com.balabalaim.gcloud.tim.push.appdev.rtc

import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.ImageButton
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import com.balabalaim.gcloud.tim.push.appdev.R
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.ui.common.PermissionCopy
import com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermission
import com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermissionPolicy
import com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermissionStore
import com.tencent.rtmp.ui.TXCloudVideoView

/**
 * Aligns with iOS chenDao_RTCCallPageVc — fullscreen call UI over TRTC Views.
 */
class LinLanRtcCallActivity : ComponentActivity(), LinLanRtcCallUiBridge {
    private lateinit var remoteVideo: TXCloudVideoView
    private lateinit var localVideo: TXCloudVideoView
    private lateinit var blurBackdrop: View
    private lateinit var infoPanel: LinearLayout
    private lateinit var nameLabel: TextView
    private lateinit var statusLabel: TextView
    private lateinit var durationLabel: TextView
    private lateinit var avatarLetter: TextView
    private lateinit var controlsRow: LinearLayout
    private lateinit var rejectWrap: LinearLayout
    private lateinit var acceptWrap: LinearLayout
    private lateinit var rejectButton: Button
    private lateinit var acceptButton: Button
    private lateinit var cancelButton: Button
    private lateinit var hangupButton: Button
    private lateinit var micButton: ImageButton
    private lateinit var speakerButton: ImageButton
    private lateinit var cameraFlipButton: ImageButton
    private lateinit var cameraOffButton: ImageButton
    private lateinit var minimizeButton: ImageButton

    private var micOn = true
    private var speakerOn = false
    private var cameraOn = true
    private var viewReady = false
    private var minimizing = false
    /** Avoid resetting speaker every refresh (would wipe user's toggle). */
    private var appliedInCallAudioDefaults = false
    private val durationTicker = object : Runnable {
        override fun run() {
            refreshDurationText()
            durationLabel.postDelayed(this, 1000)
        }
    }
    private var durationTicking = false

    override var onViewReady: (() -> Unit)? = null

    private val permissionStore by lazy { FeaturePermissionStore(this) }
    private fun requiredFeatures() = if (LinLanRtcCallManager.callContext.value?.mediaType == LinLanRtcCallMediaType.VIDEO)
        listOf(FeaturePermission.MICROPHONE, FeaturePermission.CAMERA) else listOf(FeaturePermission.MICROPHONE)
    private fun requiredPermissions() = requiredFeatures().flatMap { it.systemPermissions }
    private val permissionLauncher = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) {
        if (permissionStore.action(requiredFeatures(), requiredPermissions()) == FeaturePermissionPolicy.Action.RUN) resumeReadyCall()
        else abortForPermission()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_rtc_call)
        bindViews()
        LinLanRtcCallManager.activeUi = this
        LinLanCallFloatWindow.hide()
        lifecycle.addObserver(object : androidx.lifecycle.DefaultLifecycleObserver {
            override fun onResume(owner: androidx.lifecycle.LifecycleOwner) {
                resumeReadyCall()
            }
        })
        ensurePermissions()
        refreshFromManager()
        wireActions()
    }

    private fun resumeReadyCall() {
        if (isFinishing || !lifecycle.currentState.isAtLeast(androidx.lifecycle.Lifecycle.State.RESUMED)) return
        if (permissionStore.action(requiredFeatures(), requiredPermissions()) != FeaturePermissionPolicy.Action.RUN) return
        viewReady = true
        onViewReady?.invoke()
        onViewReady = null
        val ctx = LinLanRtcCallManager.callContext.value
        if (ctx != null && LinLanRtcCallManager.status.value == LinLanRtcCallStatus.ACCEPT) {
            if (!LinLanTrtcProvider.shared.isInRoom) {
                LinLanRtcCallManager.enterRtcAfterAccepted()
                return
            }
            LinLanTrtcProvider.shared.rebindMediaViews(
                localPreview = localVideo,
                remoteView = remoteVideo,
                mediaType = ctx.mediaType,
                remoteUserId = LinLanTrtcProvider.shared.currentRemoteUserId() ?: ctx.peer.userId,
            )
        }
    }

    override fun onDestroy() {
        onViewReady = null
        if (LinLanRtcCallManager.activeUi === this) {
            LinLanRtcCallManager.activeUi = null
        }
        stopDurationTicker()
        super.onDestroy()
    }

    override fun localPreviewView(): TXCloudVideoView = localVideo

    override fun remoteVideoView(): TXCloudVideoView = remoteVideo

    override fun isViewReady(): Boolean = viewReady

    override fun refreshFromManager() {
        val ctx = LinLanRtcCallManager.callContext.value ?: return
        val status = LinLanRtcCallManager.status.value
        nameLabel.text = ctx.peer.displayName
        avatarLetter.text = ctx.peer.displayName.trim().firstOrNull()?.uppercaseChar()?.toString() ?: "?"
        val isVideo = ctx.mediaType == LinLanRtcCallMediaType.VIDEO
        val inCall = status == LinLanRtcCallStatus.ACCEPT

        remoteVideo.visibility = if (isVideo && inCall) View.VISIBLE else View.INVISIBLE
        localVideo.visibility = if (isVideo) View.VISIBLE else View.GONE
        blurBackdrop.visibility = if (isVideo && inCall) View.GONE else View.VISIBLE
        infoPanel.visibility = if (isVideo && inCall) View.GONE else View.VISIBLE
        minimizeButton.visibility = View.VISIBLE

        // 与微信一致：接听后展示通话时长
        if (inCall) {
            val clock = LinLanRtcSignalingPayload.formatClock(LinLanRtcCallManager.connectedElapsedSec())
            statusLabel.text = clock
            durationLabel.text = clock
            durationLabel.visibility = if (isVideo) View.VISIBLE else View.GONE
            startDurationTicker()
        } else {
            stopDurationTicker()
            durationLabel.visibility = View.GONE
            statusLabel.text = when {
                ctx.role == LinLanRtcCallRole.CALLED && isVideo -> "邀请你视频通话…"
                ctx.role == LinLanRtcCallRole.CALLED -> "邀请你语音通话…"
                else -> "正在等待对方接听…"
            }
        }

        if (inCall) {
            if (!appliedInCallAudioDefaults) {
                speakerOn = isVideo
                LinLanTrtcProvider.shared.setSpeakerEnabled(speakerOn)
                appliedInCallAudioDefaults = true
            }
        } else {
            appliedInCallAudioDefaults = false
        }
        refreshBottomBar(ctx, inCall)
        refreshControlSelection()
    }

    override fun onRemoteVideoAvailabilityChanged(available: Boolean) {
        remoteVideo.visibility = if (available) View.VISIBLE else View.INVISIBLE
    }

    override fun onRemoteUserJoined() {
        refreshDurationText()
        refreshFromManager()
    }

    override fun dismissForEnd(reason: LinLanRtcCallEndReason) {
        stopDurationTicker()
        LinLanCallFloatWindow.hide()
        if (!isFinishing) finish()
    }

    private fun refreshDurationText() {
        if (LinLanRtcCallManager.status.value != LinLanRtcCallStatus.ACCEPT) return
        val clock = LinLanRtcSignalingPayload.formatClock(LinLanRtcCallManager.connectedElapsedSec())
        statusLabel.text = clock
        durationLabel.text = clock
    }

    private fun startDurationTicker() {
        if (durationTicking) return
        durationTicking = true
        durationLabel.post(durationTicker)
    }

    private fun stopDurationTicker() {
        durationTicking = false
        durationLabel.removeCallbacks(durationTicker)
    }

    private fun bindViews() {
        remoteVideo = findViewById(R.id.remoteVideo)
        localVideo = findViewById(R.id.localVideo)
        blurBackdrop = findViewById(R.id.blurBackdrop)
        infoPanel = findViewById(R.id.infoPanel)
        nameLabel = findViewById(R.id.nameLabel)
        statusLabel = findViewById(R.id.statusLabel)
        durationLabel = findViewById(R.id.durationLabel)
        avatarLetter = findViewById(R.id.avatarLetter)
        controlsRow = findViewById(R.id.controlsRow)
        rejectWrap = findViewById(R.id.rejectWrap)
        acceptWrap = findViewById(R.id.acceptWrap)
        rejectButton = findViewById(R.id.rejectButton)
        acceptButton = findViewById(R.id.acceptButton)
        cancelButton = findViewById(R.id.cancelButton)
        hangupButton = findViewById(R.id.hangupButton)
        micButton = findViewById(R.id.micButton)
        speakerButton = findViewById(R.id.speakerButton)
        cameraFlipButton = findViewById(R.id.cameraFlipButton)
        cameraOffButton = findViewById(R.id.cameraOffButton)
        minimizeButton = findViewById(R.id.minimizeButton)
    }

    private fun wireActions() {
        hangupButton.setOnClickListener { LinLanRtcCallManager.hangup() }
        cancelButton.setOnClickListener { LinLanRtcCallManager.cancelOutgoingCall() }
        rejectButton.setOnClickListener { LinLanRtcCallManager.rejectIncomingCall() }
        acceptButton.setOnClickListener { LinLanRtcCallManager.acceptIncomingCall() }
        minimizeButton.setOnClickListener { minimizeCall() }
        micButton.setOnClickListener {
            micOn = !micOn
            LinLanTrtcProvider.shared.setMicEnabled(micOn)
            refreshControlSelection()
        }
        speakerButton.setOnClickListener {
            speakerOn = !speakerOn
            LinLanTrtcProvider.shared.setSpeakerEnabled(speakerOn)
            refreshControlSelection()
        }
        cameraFlipButton.setOnClickListener { LinLanTrtcProvider.shared.switchCamera() }
        cameraOffButton.setOnClickListener {
            cameraOn = !cameraOn
            LinLanTrtcProvider.shared.setVideoEnabled(cameraOn)
            refreshControlSelection()
        }
    }

    private fun refreshControlSelection() {
        // selected = muted / speaker-on / camera-off
        micButton.isSelected = !micOn
        speakerButton.isSelected = speakerOn
        cameraOffButton.isSelected = !cameraOn
    }

    private fun minimizeCall() {
        val ctx = LinLanRtcCallManager.callContext.value ?: return
        if (!LinLanCallFloatWindow.canDrawOverlays(this)) {
            Toast.makeText(this, "请开启「显示在其他应用上层」以使用悬浮通话", Toast.LENGTH_LONG).show()
            LinLanCallFloatWindow.requestOverlayPermission(this)
            return
        }
        minimizing = true
        LinLanCallFloatWindow.show(
            context = this,
            peerName = ctx.peer.displayName,
            isVideo = ctx.mediaType == LinLanRtcCallMediaType.VIDEO,
        )
        finish()
    }

    private fun refreshBottomBar(ctx: LinLanRtcCallContext, inCall: Boolean) {
        val isVideo = ctx.mediaType == LinLanRtcCallMediaType.VIDEO
        controlsRow.visibility = if (inCall) View.VISIBLE else View.GONE
        cameraFlipButton.visibility = if (inCall && isVideo) View.VISIBLE else View.GONE
        cameraOffButton.visibility = if (inCall && isVideo) View.VISIBLE else View.GONE

        when {
            inCall -> {
                hangupButton.visibility = View.VISIBLE
                cancelButton.visibility = View.GONE
                acceptWrap.visibility = View.GONE
                rejectWrap.visibility = View.GONE
            }
            ctx.role == LinLanRtcCallRole.CALLED -> {
                hangupButton.visibility = View.GONE
                cancelButton.visibility = View.GONE
                acceptWrap.visibility = View.VISIBLE
                rejectWrap.visibility = View.VISIBLE
                acceptButton.visibility = View.VISIBLE
                rejectButton.visibility = View.VISIBLE
            }
            else -> {
                // 拨出等待：仅居中显示「取消」
                hangupButton.visibility = View.GONE
                cancelButton.visibility = View.VISIBLE
                acceptWrap.visibility = View.GONE
                rejectWrap.visibility = View.GONE
            }
        }
    }

    private fun abortForPermission() {
        viewReady = false
        Toast.makeText(this, "权限未开启，可到“我的 → 功能权限”重新设置", Toast.LENGTH_LONG).show()
        val call = LinLanRtcCallManager.callContext.value
        if (LinLanRtcCallManager.status.value == LinLanRtcCallStatus.WAITING) {
            if (call?.role == LinLanRtcCallRole.CALLED) LinLanRtcCallManager.rejectIncomingCall()
            else LinLanRtcCallManager.cancelOutgoingCall()
        } else LinLanRtcCallManager.hangup()
        finish()
    }

    private fun ensurePermissions() {
        val features = requiredFeatures()
        val permissions = requiredPermissions()
        when (permissionStore.action(features, permissions)) {
            FeaturePermissionPolicy.Action.RUN -> return
            FeaturePermissionPolicy.Action.BLOCKED_FEATURE, FeaturePermissionPolicy.Action.BLOCKED_SYSTEM -> abortForPermission()
            FeaturePermissionPolicy.Action.REQUEST_SYSTEM -> {
                val missing = permissions.filter { it !in permissionStore.granted(permissions) }
                permissionStore.policy.requested(missing)
                permissionLauncher.launch(missing.toTypedArray())
            }
            FeaturePermissionPolicy.Action.EXPLAIN -> android.app.AlertDialog.Builder(this)
                .setTitle("${LinLanAppConstants.APP_NAME}权限说明")
                .setMessage(if (FeaturePermission.CAMERA in features) PermissionCopy.CAMERA_AND_MIC else PermissionCopy.MICROPHONE)
                .setPositiveButton("允许") { _, _ ->
                    permissionStore.policy.choose(features, FeaturePermissionPolicy.Choice.ALLOWED)
                    ensurePermissions()
                    if (lifecycle.currentState.isAtLeast(androidx.lifecycle.Lifecycle.State.RESUMED)) resumeReadyCall()
                }
                .setNegativeButton("拒绝") { _, _ ->
                    permissionStore.policy.choose(features, FeaturePermissionPolicy.Choice.DENIED)
                    abortForPermission()
                }
                .setCancelable(false).show()
        }
    }
}
