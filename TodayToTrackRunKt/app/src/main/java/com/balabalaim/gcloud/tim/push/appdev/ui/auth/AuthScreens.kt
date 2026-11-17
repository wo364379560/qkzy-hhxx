package com.balabalaim.gcloud.tim.push.appdev.ui.auth

import android.app.Activity
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.ClickableText
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.zIndex
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val DeepTeal = Color(0xFF202C40)
private val Leaf = Color(0xFFD26336)
private val WarmSand = Color(0xFFF7F1E8)
private val Clay = Color(0xFFD97745)

@Composable
fun LinLanSplashScreen(
    onFinished: (loggedIn: Boolean) -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit = { _, _ -> },
) {
    LaunchedEffect(Unit) {
        val canResume = try {
            LinLanAppContainer.authRepository().decideSplashLoggedIn()
        } catch (e: kotlinx.coroutines.CancellationException) {
            throw e
        } catch (_: java.io.IOException) {
            false // No readable local session (e.g. failed preference storage).
        }
        onFinished(canResume)
    }
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Brush.verticalGradient(listOf(DeepTeal, Color(0xFF34465F)))),
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                LinLanAppConstants.APP_NAME,
                color = WarmSand,
                fontSize = 40.sp,
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Serif,
            )
            Text("连接可信的人", color = Leaf, modifier = Modifier.padding(top = 8.dp))
            Spacer(modifier = Modifier.height(28.dp))
            CircularProgressIndicator(color = Leaf)
            Text("正在检查登录状态…", color = WarmSand)
        }
    }
}

@Composable
fun LinLanLoginScreen(
    onLoggedIn: () -> Unit,
    onRegister: () -> Unit,
    onForgotPassword: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
) {
    val context = LocalContext.current
    val prefs = remember { LinLanAppContainer.appPrefs() }
    var privacyDialogAccepted by remember { mutableStateOf(prefs.privacyAccepted) }
    var showPrivacyGate by remember { mutableStateOf(!prefs.privacyAccepted) }
    var agreed by remember { mutableStateOf(false) }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var useSms by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    var challenge by remember { mutableStateOf<org.json.JSONObject?>(null) }
    challenge?.let { pending ->
        DeviceApprovalDialog(pending, phone.trim(), onDone = { challenge = null; onLoggedIn() }, onCancel = { challenge = null })
    }
    val auth = remember { LinLanAppContainer.authRepository() }

    if (showPrivacyGate) {
        LinLanPrivacyConsentDialog(
            onAgree = {
                prefs.privacyAccepted = true
                privacyDialogAccepted = true
                showPrivacyGate = false
            },
            onDisagree = {
                (context as? Activity)?.finishAffinity()
            },
            onOpenLegal = onOpenLegal,
        )
        return
    }

    AuthShell(brandLine = "欢迎回来") {
        AuthField(phone, { phone = it }, "手机号")
        Spacer(Modifier.height(12.dp))
        if (useSms) {
            AuthCodeRow(
                code = code,
                onCodeChange = { code = it },
                phone = phone,
                onSent = { error = it },
            )
        } else {
            AuthField(password, { password = it }, "密码", password = true)
        }
        error?.let {
            Text(it, color = Color(0xFFC43E38), modifier = Modifier.padding(top = 8.dp))
        }
        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = agreed, enabled = !loading, onCheckedChange = { agreed = it })
            LegalLinks(
                prefix = "已阅读并同意",
                onOpenAgreement = {
                    onOpenLegal("用户协议", LinLanAppConstants.USER_AGREEMENT_URL)
                },
                onOpenPrivacy = {
                    onOpenLegal("隐私政策", LinLanAppConstants.PRIVACY_POLICY_URL)
                },
            )
        }
        Spacer(Modifier.height(12.dp))
        AuthPrimaryButton(
            text = if (loading) "登录中…" else "登录",
            enabled = !loading && canSubmitPrivacyConsent(prefs.privacyAccepted, privacyDialogAccepted, agreed),
            onClick = {
                if (loading) return@AuthPrimaryButton
                if (!canSubmitPrivacyConsent(prefs.privacyAccepted, privacyDialogAccepted, agreed)) {
                    error = "请先阅读并勾选用户协议和隐私政策"
                    return@AuthPrimaryButton
                }
                scope.launch {
                    loading = true
                    error = null
                    runCatching {
                        if (useSms) auth.loginWithCode(phone.trim(), code.trim())
                        else auth.loginWithPassword(phone.trim(), password)
                        if (agreed) {
                            prefs.privacyAccepted = true
                        }
                        onLoggedIn()
                    }.onFailure {
                        if (it is com.balabalaim.gcloud.tim.push.appdev.data.network.DeviceApprovalRequired) challenge = it.challenge
                        else error = it.message
                    }
                    loading = false
                }
            },
        )
        TextButton(onClick = { useSms = !useSms }) {
            Text(if (useSms) "密码登录" else "验证码登录", color = Leaf)
        }
        TextButton(onClick = onForgotPassword) { Text("忘记密码", color = DeepTeal.copy(0.8f)) }
        TextButton(onClick = onRegister) { Text("没有账号？去注册", color = Leaf) }
    }
}

@Composable
fun LinLanRegisterScreen(
    onRegistered: () -> Unit,
    onBack: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
) {
    val context = LocalContext.current
    val prefs = remember { LinLanAppContainer.appPrefs() }
    var privacyDialogAccepted by remember { mutableStateOf(prefs.privacyAccepted) }
    var showPrivacyGate by remember { mutableStateOf(!prefs.privacyAccepted) }
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var nickname by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var agreed by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val auth = remember { LinLanAppContainer.authRepository() }

    if (showPrivacyGate) {
        LinLanPrivacyConsentDialog(
            onAgree = {
                prefs.privacyAccepted = true
                privacyDialogAccepted = true
                showPrivacyGate = false
            },
            onDisagree = {
                (context as? Activity)?.finishAffinity()
            },
            onOpenLegal = onOpenLegal,
        )
        return
    }

    AuthShell(
        brandLine = "开启新连接",
        topTrailing = {
            TextButton(onClick = onBack) {
                Text("返回登录", color = WarmSand, fontWeight = FontWeight.Medium)
            }
        },
    ) {
        AuthField(phone, { phone = it }, "手机号")
        Spacer(Modifier.height(10.dp))
        AuthCodeRow(
            code = code,
            onCodeChange = { code = it },
            phone = phone,
            onSent = { error = it },
        )
        Spacer(Modifier.height(10.dp))
        AuthField(nickname, { nickname = it }, "昵称")
        Spacer(Modifier.height(10.dp))
        AuthField(password, { password = it }, "密码", password = true)
        error?.let { Text(it, color = Color(0xFFC43E38), modifier = Modifier.padding(top = 8.dp)) }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = agreed, enabled = !loading, onCheckedChange = { agreed = it })
            LegalLinks(
                prefix = "同意",
                onOpenAgreement = {
                    onOpenLegal("用户协议", LinLanAppConstants.USER_AGREEMENT_URL)
                },
                onOpenPrivacy = {
                    onOpenLegal("隐私政策", LinLanAppConstants.PRIVACY_POLICY_URL)
                },
            )
        }
        AuthPrimaryButton(
            text = if (loading) "注册中…" else "创建账号",
            enabled = !loading && canSubmitPrivacyConsent(prefs.privacyAccepted, privacyDialogAccepted, agreed),
            onClick = {
                if (loading) return@AuthPrimaryButton
                if (!canSubmitPrivacyConsent(prefs.privacyAccepted, privacyDialogAccepted, agreed)) {
                    error = "请先阅读并勾选用户协议和隐私政策"
                    return@AuthPrimaryButton
                }
                val phoneVal = phone.trim()
                val nickVal = nickname.trim()
                val localError = when {
                    phoneVal.isEmpty() -> "请填写手机号"
                    nickVal.isEmpty() -> "请填写昵称"
                    password.isEmpty() -> "请填写密码"
                    password.length < 6 -> "密码长度不能少于6位"
                    else -> null
                }
                if (localError != null) {
                    error = localError
                    return@AuthPrimaryButton
                }
                scope.launch {
                    loading = true
                    error = null
                    runCatching {
                        auth.register(
                            phoneVal,
                            code.trim().ifBlank { null },
                            nickVal,
                            password,
                            inviteCode = null,
                        )
                        if (agreed) {
                            prefs.privacyAccepted = true
                        }
                        onRegistered()
                    }.onFailure { error = it.message }
                    loading = false
                }
            },
        )
    }
}

@Composable
fun LinLanForgotPasswordScreen(onBack: () -> Unit) {
    var phone by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }

    AuthShell(brandLine = "重置密码") {
        AuthField(phone, { phone = it }, "手机号")
        Spacer(Modifier.height(10.dp))
        AuthCodeRow(
            code = code,
            onCodeChange = { code = it },
            phone = phone,
            onSent = { message = it },
        )
        Spacer(Modifier.height(10.dp))
        AuthField(password, { password = it }, "新密码", password = true)
        message?.let { Text(it, color = Leaf, modifier = Modifier.padding(top = 8.dp)) }
        AuthPrimaryButton("确认重置", true) {
            scope.launch {
                runCatching { api.resetPassword(phone.trim(), code.trim(), password) }
                    .onSuccess { message = "密码已重置，请返回登录" }
                    .onFailure { message = it.message }
            }
        }
        TextButton(onClick = onBack) { Text("返回", color = WarmSand.copy(0.8f)) }
    }
}

internal fun canSubmitPrivacyConsent(
    persistedAccepted: Boolean,
    dialogAccepted: Boolean,
    checkboxChecked: Boolean,
): Boolean {
    return checkboxChecked && (persistedAccepted || dialogAccepted)
}

@Composable
private fun LinLanPrivacyConsentDialog(
    onAgree: () -> Unit,
    onDisagree: () -> Unit,
    onOpenLegal: (title: String, url: String) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDisagree,
        properties = DialogProperties(
            dismissOnBackPress = true,
            dismissOnClickOutside = false,
        ),
        title = { Text("服务协议与隐私保护") },
        text = {
            Column {
                Text(
                    "欢迎使用${LinLanAppConstants.APP_NAME}。请仔细阅读《用户协议》和《隐私政策》，" +
                        "了解我们如何收集、使用和保护您的个人信息。您可选择「同意」继续使用，或选择「拒绝」退出应用。",
                )
                Spacer(Modifier.height(8.dp))
                LegalLinks(
                    prefix = "查看",
                    onOpenAgreement = {
                        onOpenLegal("用户协议", LinLanAppConstants.USER_AGREEMENT_URL)
                    },
                    onOpenPrivacy = {
                        onOpenLegal("隐私政策", LinLanAppConstants.PRIVACY_POLICY_URL)
                    },
                )
            }
        },
        confirmButton = {
            TextButton(onClick = onAgree) { Text("同意") }
        },
        dismissButton = {
            TextButton(onClick = onDisagree) { Text("拒绝") }
        },
    )
}

@Composable
private fun LegalLinks(
    prefix: String,
    onOpenAgreement: () -> Unit,
    onOpenPrivacy: () -> Unit,
) {
    val annotated = buildAnnotatedString {
        append("$prefix ")
        pushStringAnnotation("ua", "ua")
        withStyle(SpanStyle(color = Leaf, textDecoration = TextDecoration.Underline)) {
            append("用户协议")
        }
        pop()
        append(" 与 ")
        pushStringAnnotation("pp", "pp")
        withStyle(SpanStyle(color = Leaf, textDecoration = TextDecoration.Underline)) {
            append("隐私政策")
        }
        pop()
    }
    ClickableText(
        text = annotated,
        onClick = { offset ->
            annotated.getStringAnnotations(offset, offset).firstOrNull()?.let {
                when (it.tag) {
                    "ua" -> onOpenAgreement()
                    "pp" -> onOpenPrivacy()
                }
            }
        },
    )
}

@Composable
private fun AuthShell(
    brandLine: String,
    topTrailing: (@Composable () -> Unit)? = null,
    content: @Composable () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    listOf(DeepTeal, Color(0xFF34465F), Color(0xFF465269))
                )
            ),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .windowInsetsPadding(WindowInsets.statusBars)
                .padding(horizontal = 24.dp)
                .padding(top = if (topTrailing != null) 48.dp else 16.dp, bottom = 24.dp),
        ) {
            Text(
                LinLanAppConstants.APP_NAME,
                color = WarmSand,
                fontSize = 42.sp,
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Serif,
            )
            Text(brandLine, color = Leaf, fontSize = 18.sp, modifier = Modifier.padding(top = 6.dp, bottom = 28.dp))
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(WarmSand.copy(alpha = 0.96f), RoundedCornerShape(24.dp))
                    .padding(20.dp),
            ) { content() }
        }
        // 必须叠在滚动层之上，否则「返回登录」会被全屏 Column 挡住点击
        if (topTrailing != null) {
            Box(
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .zIndex(1f)
                    .windowInsetsPadding(WindowInsets.statusBars)
                    .padding(horizontal = 4.dp, vertical = 2.dp),
            ) {
                topTrailing()
            }
        }
    }
}

@Composable
private fun AuthCodeRow(
    code: String,
    onCodeChange: (String) -> Unit,
    phone: String,
    onSent: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val api = remember { LinLanAppContainer.apiClient() }
    var coolDown by remember { mutableIntStateOf(0) }
    LaunchedEffect(coolDown) {
        if (coolDown <= 0) return@LaunchedEffect
        delay(1000)
        coolDown -= 1
    }
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AuthField(
            value = code,
            onChange = onCodeChange,
            label = "验证码",
            modifier = Modifier.weight(1f),
        )
        Spacer(modifier = Modifier.width(8.dp))
        TextButton(
            onClick = {
                if (phone.isBlank() || coolDown > 0) return@TextButton
                scope.launch {
                    runCatching { api.sendSms(phone.trim()) }
                        .onSuccess {
                            onSent("验证码已发送")
                            coolDown = 60
                        }
                        .onFailure { onSent(it.message ?: "发送失败") }
                }
            },
            enabled = coolDown <= 0 && phone.isNotBlank(),
        ) {
            Text(
                if (coolDown > 0) "${coolDown}s" else "获取验证码",
                color = if (coolDown > 0) Color.Gray else Clay,
                fontWeight = FontWeight.Medium,
            )
        }
    }
}

@Composable
private fun AuthField(
    value: String,
    onChange: (String) -> Unit,
    label: String,
    password: Boolean = false,
    modifier: Modifier = Modifier.fillMaxWidth(),
) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        singleLine = true,
        modifier = modifier,
        visualTransformation = if (password) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = Leaf,
            focusedLabelColor = DeepTeal,
            cursorColor = Leaf,
        ),
        shape = RoundedCornerShape(14.dp),
    )
}

@Composable
private fun AuthPrimaryButton(text: String, enabled: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 8.dp),
        colors = ButtonDefaults.buttonColors(containerColor = DeepTeal, contentColor = WarmSand),
        shape = RoundedCornerShape(14.dp),
    ) { Text(text, fontWeight = FontWeight.SemiBold) }
}
