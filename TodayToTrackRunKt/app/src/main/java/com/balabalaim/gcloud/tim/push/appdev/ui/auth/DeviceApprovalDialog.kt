package com.balabalaim.gcloud.tim.push.appdev.ui.auth

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.json.JSONObject

@Composable
fun DeviceApprovalDialog(challenge: JSONObject, phone: String, onDone: () -> Unit, onCancel: () -> Unit) {
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }
    val verification = remember(challenge) { Mutex() }
    var code by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("请在已登录设备的「登录设备管理」中确认本次登录。") }
    var busy by remember { mutableStateOf(false) }
    var terminal by remember { mutableStateOf(false) }
    var cooldown by remember { mutableIntStateOf(0) }
    var exchanging by remember { mutableStateOf(false) }
    suspend fun exchange() {
        if (exchanging) return
        exchanging = true
        try {
            LinLanAppContainer.authRepository().completeDeviceApproval(challenge, phone)
            terminal = true
            onDone()
        } catch (cancelled: CancellationException) { throw cancelled }
        catch (error: Exception) { message = error.message ?: "登录失败" }
        finally { exchanging = false }
    }
    LaunchedEffect(challenge) {
        while (!terminal) {
            try { verification.withLock {
                if (terminal) return@withLock
                when (api.challengeStatus(challenge)) {
                    "approved" -> exchange()
                    "denied", "rejected" -> { terminal = true; message = "本次登录已被拒绝，请取消后重试。" }
                    "expired", "consumed" -> { terminal = true; message = "验证已过期或已使用，请重新登录。" }
                }
            } } catch (cancelled: CancellationException) { throw cancelled }
            catch (error: Exception) { message = error.message ?: "网络异常，正在重试" }
            delay(3000)
        }
    }
    LaunchedEffect(cooldown) { if (cooldown > 0) { delay(1000); cooldown-- } }
    AlertDialog(onDismissRequest = { if (!busy && !exchanging) onCancel() }, title = { Text("新设备登录验证") }, text = {
        Column(Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("本次验证标识：${challenge.optString("challenge_short_code")}")
            Text(message)
            Text("没有其他已登录设备？使用短信验证码确认。")
            val maskedPhone = if (phone.length >= 7) phone.take(3) + "****" + phone.takeLast(4) else phone
            Text("验证码将发送至 $maskedPhone")
            OutlinedTextField(code, { code = it.filter(Char::isDigit).take(6) }, label = { Text("短信验证码") }, singleLine = true,
                enabled = !terminal && !busy && !exchanging, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number))
            TextButton(enabled = !terminal && !busy && !exchanging && cooldown == 0, onClick = { scope.launch {
                busy = true
                try {
                    api.sendDeviceVerificationSms(phone)
                    cooldown = 60; message = "验证码已发送，请填写后点击验证并登录。"
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (error: Exception) { message = error.message.orEmpty() }
                finally { busy = false }
            } }) { Text(if (cooldown > 0) "${cooldown}秒后重发" else "发送验证码") }
        }
    }, confirmButton = { TextButton(enabled = !terminal && !busy && !exchanging && code.length >= 4, onClick = { scope.launch {
        busy = true
        try { verification.withLock {
            if (!terminal) { api.verifyChallengeSms(challenge, code); exchange() }
        } } catch (cancelled: CancellationException) { throw cancelled }
        catch (error: Exception) { message = error.message.orEmpty() }
        finally { busy = false }
    } }) { Text(if (busy || exchanging) "正在验证…" else "验证并登录") } }, dismissButton = {
        TextButton(enabled = !busy && !exchanging, onClick = onCancel) { Text("取消") }
    })
}
