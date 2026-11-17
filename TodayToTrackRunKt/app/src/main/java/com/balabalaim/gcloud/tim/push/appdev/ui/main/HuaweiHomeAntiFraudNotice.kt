package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.content.Context
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.*
import java.security.MessageDigest
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay

/** Uses the same content-version acknowledgement contract as LingDoSpace. */
@Composable
fun HuaweiHomeAntiFraudNotice(onResolved: () -> Unit) {
    if (!isHuaweiDevice()) return
    val context = LocalContext.current
    val prefs = remember { context.getSharedPreferences("anti_fraud_notice", Context.MODE_PRIVATE) }
    val session by LinLanAppContainer.sessionStore().sessionFlow.collectAsState(initial = null)
    val userId = session?.userId.orEmpty()
    val resolved by rememberUpdatedState(onResolved)
    var content by rememberSaveable(userId) { mutableStateOf("") }
    var countdown by rememberSaveable(userId) { mutableIntStateOf(7) }
    var checked by rememberSaveable(userId) { mutableStateOf(false) }

    LaunchedEffect(userId) {
        if (userId.isBlank()) return@LaunchedEffect
        if (checked) {
            if (content.isBlank()) resolved()
            return@LaunchedEffect
        }
        try {
            val notice = LinLanAppContainer.apiClient().antiFraudNotice()
            val body = notice.optString("content").trim()
            if (notice.optBoolean("enabled") && body.isNotBlank() &&
                prefs.getString("ack_$userId", "") != noticeVersion(body)) {
                countdown = notice.optInt("countdown_sec", 7).coerceIn(0, 120)
                content = body
            }
            checked = true
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (_: Exception) {
            // A failed notice fetch must not block the home screen; retry on its next entry.
        }
        if (content.isBlank()) resolved()
    }

    if (content.isNotBlank()) {
        AntiFraudReadingDialog(content, countdown) {
            prefs.edit().putString("ack_$userId", noticeVersion(content)).apply()
            content = ""
            resolved()
        }
    }
}

private fun noticeVersion(content: String): String = MessageDigest.getInstance("SHA-256")
    .digest(content.toByteArray(Charsets.UTF_8)).joinToString("") { "%02x".format(it) }

@Composable
private fun AntiFraudReadingDialog(content: String, seconds: Int, onConfirm: () -> Unit) {
    var remaining by rememberSaveable(content) { mutableIntStateOf(seconds) }
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(content, lifecycle) {
        // Time spent in the background does not count as reading time.
        lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) {
            while (remaining > 0) {
                delay(1_000)
                remaining--
            }
        }
    }
    Dialog(onDismissRequest = {}, properties = DialogProperties(
        dismissOnBackPress = false, dismissOnClickOutside = false, usePlatformDefaultWidth = false,
    )) {
        Surface(Modifier.fillMaxWidth().padding(horizontal = 20.dp).fillMaxHeight(0.86f),
            shape = RoundedCornerShape(22.dp), color = LinLanCanvas) {
            Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                Text("防范电信网络诈骗提示", Modifier.fillMaxWidth(), fontSize = 20.sp,
                    fontWeight = FontWeight.Bold, textAlign = TextAlign.Center, color = LinLanInk)
                Text("请完整阅读以下安全提醒，阅读倒计时结束后可确认。提高警惕，保护好个人信息与财产安全。",
                    fontSize = 13.sp, lineHeight = 20.sp, color = LinLanInkSecondary,
                    textAlign = TextAlign.Center)
                Column(Modifier.weight(1f).fillMaxWidth()
                    .background(LinLanSurface, RoundedCornerShape(14.dp))
                    .verticalScroll(rememberScrollState()).padding(16.dp)) {
                    Text(content, fontSize = 16.sp, lineHeight = 27.sp, color = LinLanInk)
                }
                Text(if (remaining > 0) "请认真阅读，${remaining} 秒后可点击确认" else "请牢记安全提醒，谨防网络诈骗",
                    Modifier.fillMaxWidth(), fontSize = 13.sp, color = LinLanAccent, textAlign = TextAlign.Center)
                Button(onClick = { if (remaining == 0) onConfirm() }, enabled = remaining == 0,
                    modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp), shape = RoundedCornerShape(14.dp)) {
                    Text("我已知晓", fontSize = 17.sp, fontWeight = FontWeight.SemiBold)
                }
            }
        }
    }
}
