package com.balabalaim.gcloud.tim.push.appdev

import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallActivity
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallManager
import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanLocalNotifyHelper
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanNotifyRouter
import com.balabalaim.gcloud.tim.push.appdev.ui.navigation.LinLanAppNavHost
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanTheme

class MainActivity : ComponentActivity() {
    private val notifyPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { /* 用户拒绝则仅不弹横幅，不影响聊天 */ }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        LinLanAppContainer.init(applicationContext)
        LinLanLocalNotifyHelper.ensureChannels(applicationContext)
        handleNotifyIntent(intent)
        resumeCallFromLauncher(intent)
        if (LinLanAppContainer.appPrefs().privacyAccepted) {
            maybeRequestNotifyPermission()
        }
        enableEdgeToEdge()
        setContent {
            LinLanTheme {
                com.balabalaim.gcloud.tim.push.appdev.ui.main.DeviceLockGate { LinLanAppNavHost() }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        handleNotifyIntent(intent)
        resumeCallFromLauncher(intent)
    }

    private fun resumeCallFromLauncher(intent: Intent?) {
        if (intent?.action == Intent.ACTION_MAIN && intent.hasCategory(Intent.CATEGORY_LAUNCHER)
            && LinLanRtcCallManager.callContext.value != null) {
            startActivity(Intent(this, LinLanRtcCallActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_REORDER_TO_FRONT or Intent.FLAG_ACTIVITY_SINGLE_TOP))
        }
    }

    private fun handleNotifyIntent(intent: Intent?) {
        if (intent == null) return
        if (intent.getBooleanExtra(LinLanLocalNotifyHelper.EXTRA_IS_CALL, false)) {
            LinLanNotifyRouter.offerCall()
            return
        }
        val peerId = intent.getStringExtra(LinLanLocalNotifyHelper.EXTRA_PEER_ID).orEmpty()
        if (peerId.isBlank()) return
        val isGroup = intent.getBooleanExtra(LinLanLocalNotifyHelper.EXTRA_IS_GROUP, false)
        val title = intent.getStringExtra(LinLanLocalNotifyHelper.EXTRA_TITLE).orEmpty().ifBlank { peerId }
        LinLanNotifyRouter.offerChat(peerId, isGroup, title)
        // 避免重复消费
        intent.removeExtra(LinLanLocalNotifyHelper.EXTRA_PEER_ID)
    }

    private fun maybeRequestNotifyPermission() {
        if (Build.VERSION.SDK_INT < 33) return
        val granted = ContextCompat.checkSelfPermission(
            this,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
        if (!granted) {
            notifyPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }
}
