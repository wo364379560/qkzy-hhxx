package com.balabalaim.gcloud.tim.push.appdev.data.network

import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanSessionStore
import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import com.balabalaim.gcloud.tim.push.appdev.data.model.ChatProfile
import com.balabalaim.gcloud.tim.push.appdev.data.model.GroupNoticeResult
import com.balabalaim.gcloud.tim.push.appdev.data.model.SearchUserResult
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanReliability
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.util.concurrent.TimeUnit

data class AvatarPresignResult(
    val putUrl: String,
    val avatarUrl: String,
)

data class ChatAttachmentPresignResult(
    val fileId: String,
    val fileName: String,
    val putUrl: String,
    val downloadUrl: String,
    val previewUrl: String,
    val contentType: String,
    val isVideo: Boolean,
)

class DeviceApprovalRequired(val challenge: JSONObject) : Exception("请验证新设备登录")

class ApiException(message: String, val unauthorized: Boolean = false, val status: Int? = null, val stage: String? = null, val providerCode: String? = null, val requestId: String? = null) : Exception(message)

class LinLanApiClient(
    private val sessionStore: LinLanSessionStore,
) {
    private val jsonMedia = "application/json; charset=utf-8".toMediaType()
    private val client = LinLanHttpTransport.configure(OkHttpClient.Builder())
        .dns(LinLanHTTPDNS.dns)
        .followRedirects(false)
        .followSslRedirects(false)
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS)
        .callTimeout(30, TimeUnit.SECONDS)
        .build()

    private val uploadClient = LinLanHttpTransport.configure(OkHttpClient.Builder())
        .dns(LinLanHTTPDNS.dns)
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.MINUTES)
        .writeTimeout(10, TimeUnit.MINUTES)
        .build()

    private val routeRecovery = ApiRouteRecovery(
        currentBaseUrl = { LinLanDomainPoolManager.currentBaseUrl() },
        recover = { failed, budget -> LinLanDomainPoolManager.failoverIfNeeded(budgetMillis = budget, failedOrigin = failed) },
        prepareFuture = { failed -> LinLanDomainPoolManager.prepareFutureRoute(failed) },
        nowMillis = { android.os.SystemClock.elapsedRealtime() },
    )

    private val refreshCoordinator = SessionRefreshCoordinator(
        read = { sessionStore.currentSession() },
        refreshRemote = { current -> refreshSessionRemote(current) },
        save = { expected, next -> sessionStore.replaceSessionIfCurrent(expected, next) },
        clear = { expected -> sessionStore.replaceSessionIfCurrent(expected, AuthSession()) },
    )

    private suspend fun authorizationHeader(): String {
        var session = sessionStore.currentSession()
        if (AccessTokenPolicy.needsRefresh(session)) session = refreshCoordinator.refresh()
        return AccessTokenPolicy.header(session.token)
    }

    suspend fun antiFraudNotice(): JSONObject = getAuthData("/notices/anti-fraud")

    suspend fun realnameStatus(): JSONObject = getAuthData("/realname/status")

    suspend fun verifyRealname(name: String, id: String) {
        postAuthVoid("/realname/verify", JSONObject().put("real_name", name).put("id_number", id))
    }

    suspend fun peerRealname(userId: String): Boolean? {
        val items = postData("/realname/peers", JSONObject().put("im_user_ids", JSONArray().put(userId)), true)
            .optJSONArray("items") ?: return null
        for (i in 0 until items.length()) {
            val item = items.optJSONObject(i) ?: continue
            if (item.optString("im_user_id") == userId) return item.optBoolean("verified")
        }
        return null
    }

    suspend fun securitySettings(): JSONObject = getAuthData("/security/settings")

    suspend fun updateSecuritySetting(name: String, enabled: Boolean): JSONObject {
        require(name in setOf("remote_login_alert", "login_verification"))
        val body = JSONObject().put(name, enabled)
        if (name == "login_verification" && enabled) {
            body.put("device_id", sessionStore.deviceId())
                .put("device_name", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}")
                .put("platform", "android")
        }
        return executeForData(Request.Builder()
            .url(LinLanDomainPoolManager.currentBaseUrl() + "/security/settings")
            .header("Authorization", authorizationHeader())
            .put(body.toString().toRequestBody(jsonMedia)).build())
    }

    suspend fun loginDevices(): List<JSONObject> = jsonItems(getAuthData(
        "/devices?current_device_id=" + java.net.URLEncoder.encode(sessionStore.deviceId(), "UTF-8")))

    suspend fun loginChallenges(): List<JSONObject> = jsonItems(getAuthData("/login-challenges/pending"))

    suspend fun revokeDevice(id: String) = postAuthVoid("/devices/revoke", JSONObject().put("device_id", id))

    suspend fun decideChallenge(id: Long, approve: Boolean, trustDevice: Boolean = false) = postAuthVoid(
        "/login-challenges/$id/${if (approve) "approve" else "deny"}",
        JSONObject().put("current_device_id", sessionStore.deviceId()).put("trust_device", approve && trustDevice))

    suspend fun sendDeviceVerificationSms(phone: String) = postVoid("/sms/send", JSONObject()
        .put("phone", phone.trim()).put("scene", "device_verify").put("device_id", sessionStore.deviceId()))

    suspend fun challengeStatus(challenge: JSONObject): String {
        val url = (LinLanDomainPoolManager.currentBaseUrl() + "/login-challenges/${challenge.optLong("challenge_id")}")
            .toHttpUrl().newBuilder().addQueryParameter("pre_auth_token", challenge.optString("pre_auth_token")).build()
        return executeForData(Request.Builder().url(url).get().build()).optString("status")
    }

    suspend fun verifyChallengeSms(challenge: JSONObject, code: String) = postVoid(
        "/login-challenges/${challenge.optLong("challenge_id")}/verify-sms",
        JSONObject().put("pre_auth_token", challenge.optString("pre_auth_token")).put("code", code))

    suspend fun exchangeChallenge(challenge: JSONObject): AuthSession = authRequest(
        "/login-challenges/${challenge.optLong("challenge_id")}/exchange",
        JSONObject().put("pre_auth_token", challenge.optString("pre_auth_token")))

    private fun jsonItems(data: JSONObject): List<JSONObject> {
        val items = data.optJSONArray("items") ?: return emptyList()
        return (0 until items.length()).mapNotNull { items.optJSONObject(it) }
    }

    suspend fun initializeRouting() {
        LinLanDomainPoolManager.ensureReady()
    }

    suspend fun loginWithPassword(phone: String, password: String): AuthSession {
        val body = JSONObject()
            .put("phone", phone)
            .put("password", password)
            .put("device_id", sessionStore.deviceId())
            .put("device_name", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}")
            .put("platform", "android")
        return authRequest("/login", body)
    }

    suspend fun loginWithCode(phone: String, code: String): AuthSession {
        val body = JSONObject()
            .put("phone", phone)
            .put("code", code)
            .put("device_id", sessionStore.deviceId())
            .put("device_name", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}")
            .put("platform", "android")
        return authRequest("/login", body)
    }

    suspend fun register(
        phone: String,
        code: String?,
        nickname: String,
        password: String,
        inviteCode: String?,
    ): AuthSession {
        val body = JSONObject()
            .put("phone", phone)
            .put("nickname", nickname)
            .put("password", password)
            .put("register_type", "phone_sms")
            .put("device_id", sessionStore.deviceId())
            .put("device_name", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}")
            .put("platform", "android")
        if (!code.isNullOrBlank()) body.put("code", code)
        if (!inviteCode.isNullOrBlank()) body.put("invite_code", inviteCode)
        return authRequest("/register", body)
    }

    suspend fun sendSms(phone: String) = postVoid("/sms/send", JSONObject().put("phone", phone))

    suspend fun resetPassword(phone: String, code: String, newPassword: String) {
        val body = JSONObject()
            .put("phone", phone)
            .put("code", code)
            .put("new_password", newPassword)
        postVoid("/password/reset", body)
    }

    suspend fun updatePassword(currentPassword: String, newPassword: String) {
        val body = JSONObject()
            .put("current_password", currentPassword)
            .put("new_password", newPassword)
        postAuthVoid("/password/update", body)
    }

    suspend fun bindInviteCode(code: String) {
        postAuthVoid("/invite-code/bind", JSONObject().put("invite_code", code))
    }

    suspend fun fetchProfile(): ChatProfile = withContext(Dispatchers.IO) {
        val data = getAuthData("/profile")
        com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnerProtocol.profile(data)
    }

    suspend fun syncFriendOwnership(target: String, requestId: String, owner: String, generation: Long): com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnerAssignment = withContext(Dispatchers.IO) {
        require(target.isNotBlank() && target.length <= 64 && !target.contains('/'))
        val header = authorizationHeader()
        if (sessionStore.sessionGeneration() != generation || sessionStore.currentSession().userId != owner) throw CancellationException("账号已变化")
        val request = Request.Builder().url(LinLanDomainPoolManager.currentBaseUrl() + "/users/" + java.net.URLEncoder.encode(target, "UTF-8") + "/bind-owner-from-me")
            .header("Authorization", header).post(JSONObject().put("request_id", requestId).toString().toRequestBody(jsonMedia)).build()
        val data = executeForData(request, owner, generation)
        com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnerProtocol.assignment(data, requestId)
    }

    suspend fun fetchFriendOwnerBrief(target: String): JSONObject = postData("/users/briefs", JSONObject().put("im_user_ids", JSONArray().put(target)), true)

    suspend fun updateProfile(
        nickname: String? = null,
        avatarUrl: String? = null,
        selfSignature: String? = null,
        wananId: String? = null,
        allowType: Int? = null,
    ) {
        val body = JSONObject()
        if (!nickname.isNullOrBlank()) body.put("nickname", nickname)
        if (!avatarUrl.isNullOrBlank()) body.put("avatar_url", avatarUrl)
        if (selfSignature != null) body.put("self_signature", selfSignature)
        if (wananId != null) body.put("wanan_id", wananId.trim().lowercase())
        if (allowType != null) body.put("allow_type", allowType)
        if (body.length() == 0) return
        postAuthVoid("/profile/update", body)
    }

    /** GET /upload/avatar/presign — same as iOS. */
    suspend fun getUserAvatarPresign(userId: String, bizId: Int = 0): AvatarPresignResult =
        avatarPresign(type = "user", userId = userId, groupId = null, bizId = bizId)

    suspend fun getGroupAvatarPresign(groupId: String, bizId: Int = 0): AvatarPresignResult =
        avatarPresign(type = "group", userId = null, groupId = groupId, bizId = bizId)

    suspend fun putFileToCos(putUrl: String, filePath: String) = withContext(Dispatchers.IO) {
        val file = File(filePath)
        if (!file.exists()) throw ApiException("文件不存在")
        val request = Request.Builder()
            .url(putUrl)
            .put(file.asRequestBody("image/jpeg".toMediaType()))
            .header("Content-Type", "image/jpeg")
            .build()
        try {
          client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val xml = response.peekBody(16384).string()
                val code = Regex("<Code>([A-Za-z0-9_.:-]+)</Code>").find(xml)?.groupValues?.get(1)
                val requestId = response.header("X-Cos-Request-Id") ?: Regex("<RequestId>([A-Za-z0-9_.:-]+)</RequestId>").find(xml)?.groupValues?.get(1)
                val message = when {
                    response.code == 451 -> "头像存储账户欠费或冻结，请联系管理员"
                    code in listOf("SignatureDoesNotMatch", "InvalidAccessKeyId", "RequestTimeTooSkewed") -> "头像存储签名异常，请联系管理员"
                    response.code == 403 -> "头像存储权限不足，请联系管理员"
                    else -> "头像上传失败（HTTP ${response.code}）"
                }
                throw ApiException(message + (requestId?.let { "，请求ID：$it" } ?: ""), status = response.code, stage = "cos_put", providerCode = code, requestId = requestId)
            }
          }
        } catch (error: ApiException) { throw error }
          catch (error: java.net.SocketTimeoutException) { throw ApiException("头像上传超时，请重试", stage = "cos_put") }
          catch (error: javax.net.ssl.SSLException) { throw ApiException("头像存储TLS连接失败", stage = "cos_put") }
          catch (error: java.net.UnknownHostException) { throw ApiException("头像存储域名解析失败", stage = "cos_put") }
          catch (error: java.io.IOException) { throw ApiException("头像上传连接中断，请重试", stage = "cos_put") }
    }

    suspend fun getChatAttachmentUploadSettings(): JSONObject =
        getAuthData("/upload/attachment/settings")

    suspend fun presignChatAttachment(
        conversationId: String,
        fileName: String,
        fileSize: Long,
        contentType: String,
        isVideo: Boolean,
    ): ChatAttachmentPresignResult = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("conversation_id", conversationId)
            .put("file_name", fileName)
            .put("file_size", fileSize)
            .put("content_type", contentType)
            .put("is_video", isVideo)
        val data = postData("/upload/attachment/presign", body, auth = true)
        ChatAttachmentPresignResult(
            fileId = data.optString("file_id"),
            fileName = data.optString("file_name").ifBlank { fileName },
            putUrl = data.optString("put_url"),
            downloadUrl = data.optString("download_url"),
            previewUrl = data.optString("preview_url"),
            contentType = data.optString("content_type").ifBlank { contentType },
            isVideo = data.optBoolean("is_video", isVideo),
        )
    }

    suspend fun uploadAttachmentMultipart(file: File, fileName: String, conversationId: String, contentType: String,
        isVideo: Boolean, onProgress: (Float) -> Unit = {}, isActive: () -> Boolean = { true }): ChatAttachmentPresignResult = withContext(Dispatchers.IO) {
        val expected = sessionStore.currentSession()
        val generation = sessionStore.sessionGeneration()
        suspend fun requireCurrent() {
            kotlinx.coroutines.currentCoroutineContext().ensureActive()
            if (!isActive()) throw CancellationException("上传已停止")
            if (sessionStore.sessionGeneration() != generation || sessionStore.currentSession().userId != expected.userId) throw CancellationException("账号已切换，上传已暂停")
        }
        val hash = java.security.MessageDigest.getInstance("SHA-256")
        file.inputStream().use { input ->
            val buffer = ByteArray(1024 * 1024)
            while (true) { val count = input.read(buffer); if (count < 0) break; requireCurrent(); hash.update(buffer, 0, count) }
        }
        fun hex(bytes: ByteArray) = bytes.joinToString("") { "%02x".format(it) }
        val fingerprint = hex(hash.digest())
        val requestId = hex(hash.digest("$conversationId:$fileName:${file.length()}:$contentType:$isVideo:$fingerprint".toByteArray()))
        requireCurrent()
        var state = postData("/upload/attachment/multipart/init", JSONObject().put("request_id", requestId).put("content_hash", fingerprint)
            .put("conversation_id", conversationId).put("file_name", fileName).put("file_size", file.length()).put("content_type", contentType).put("is_video", isVideo), true)
        if (state.optString("content_hash") != fingerprint || state.optLong("file_size") != file.length()) throw ApiException("上传任务与原文件不一致")
        val fileId = state.getString("file_id")
        val path = "/upload/attachment/multipart/$fileId"
        val partSize = state.getInt("part_size")
        if (partSize != 8 * 1024 * 1024) throw ApiException("不支持的分片大小")
        if (state.optString("state") == "uploading") {
            val confirmed = mutableMapOf<Int, Long>()
            val rows = state.optJSONArray("parts") ?: JSONArray()
            for (i in 0 until rows.length()) rows.getJSONObject(i).let { confirmed[it.getInt("part_number")] = it.getLong("size") }
            var confirmedBytes = confirmed.values.sum()
            java.io.RandomAccessFile(file, "r").use { input ->
                var offset = 0L
                var part = 1
                while (offset < file.length()) {
                    requireCurrent()
                    val length = minOf(partSize.toLong(), file.length() - offset).toInt()
                    if (confirmed[part] != length.toLong()) {
                        val bytes = ByteArray(length)
                        input.seek(offset); input.readFully(bytes)
                        for (attempt in 0..2) {
                            try {
                                val signed = postData("$path/parts/$part", JSONObject(), true)
                                requireCurrent()
                                val request = Request.Builder().url(signed.getString("put_url")).put(bytes.toRequestBody(null)).build()
                                uploadClient.newCall(request).execute().use { response ->
                                    if (!response.isSuccessful) throw ApiException("分片上传被存储服务拒绝(HTTP ${response.code})")
                                }
                                break
                            } catch (error: Exception) {
                                requireCurrent()
                                val progress = getAuthData(path).optJSONArray("parts") ?: JSONArray()
                                if ((0 until progress.length()).any { progress.getJSONObject(it).let { row -> row.optInt("part_number") == part && row.optLong("size") == length.toLong() } }) break
                                if (attempt == 2) throw error
                                delay(500L shl attempt)
                            }
                        }
                        confirmedBytes += length
                    }
                    onProgress(confirmedBytes.toFloat() / file.length())
                    offset += length; part++
                }
            }
            requireCurrent()
            state = postData("$path/complete", JSONObject(), true)
        }
        val deadline = System.currentTimeMillis() + 10 * 60_000
        while (state.optString("state") != "uploaded") {
            requireCurrent()
            if (state.optString("state") in setOf("failed", "aborted", "cleaned")) throw ApiException("文件合并或内容校验失败")
            if (System.currentTimeMillis() > deadline) throw ApiException("文件仍在服务端确认；重新选择同一文件可继续查询，请勿重复发送")
            delay(1500)
            state = getAuthData(path)
        }
        requireCurrent()
        onProgress(1f)
        ChatAttachmentPresignResult(fileId, state.optString("file_name"), "", state.optString("download_url"), state.optString("preview_url"), state.optString("content_type"), state.optBoolean("is_video"))
    }

    suspend fun attachmentAccess(fileId: String): String = getAuthData("/files/${java.net.URLEncoder.encode(fileId, "UTF-8")}/access").getString("url")

    suspend fun completeChatAttachment(fileId: String) {
        postAuthVoid(
            "/upload/attachment/complete",
            JSONObject().put("file_id", fileId),
        )
    }

    suspend fun putAttachmentToCos(
        putUrl: String,
        filePath: String,
        contentType: String,
    ) = withContext(Dispatchers.IO) {
        val file = File(filePath)
        if (!file.exists()) throw ApiException("文件不存在")
        val mediaType = contentType.toMediaType()
        val request = Request.Builder()
            .url(putUrl)
            .put(file.asRequestBody(mediaType))
            .header("Content-Type", contentType)
            .build()
        uploadClient.newCall(request).execute().use { response ->
            if (!response.isSuccessful) throw ApiException("上传失败(${response.code})")
        }
    }

    /** Presign + PUT COS，返回最终 avatar_url。 */
    suspend fun uploadUserAvatar(userId: String, filePath: String): String {
        val presign = getUserAvatarPresign(userId)
        putFileToCos(presign.putUrl, filePath)
        return presign.avatarUrl
    }

    suspend fun uploadGroupAvatar(groupId: String, filePath: String): String {
        val presign = getGroupAvatarPresign(groupId)
        putFileToCos(presign.putUrl, filePath)
        return presign.avatarUrl
    }

    private suspend fun avatarPresign(
        type: String,
        userId: String?,
        groupId: String?,
        bizId: Int,
    ): AvatarPresignResult = withContext(Dispatchers.IO) {
        val builder = (LinLanDomainPoolManager.currentBaseUrl() + "/upload/avatar/presign")
            .toHttpUrl()
            .newBuilder()
            .addQueryParameter("type", type)
            .addQueryParameter("biz_id", bizId.toString())
        if (!userId.isNullOrBlank()) builder.addQueryParameter("user_id", userId)
        if (!groupId.isNullOrBlank()) builder.addQueryParameter("group_id", groupId)
        val request = Request.Builder()
            .url(builder.build())
            .get()
            .header("Authorization", authorizationHeader())
            .build()
        val data = executeForData(request)
        val putUrl = data.optString("put_url")
        val avatarUrl = data.optString("avatar_url")
        if (putUrl.isBlank() || avatarUrl.isBlank()) throw ApiException("获取上传地址失败")
        AvatarPresignResult(putUrl = putUrl, avatarUrl = avatarUrl)
    }

    suspend fun deleteAccount() {
        postAuthVoid("/account/delete", JSONObject())
    }

    suspend fun submitReport(
        targetId: String,
        targetType: String,
        targetName: String,
        category: String,
        description: String? = null,
    ) {
        val body = JSONObject()
            .put("target_id", targetId)
            .put("target_type", targetType)
            .put("target_name", targetName)
            .put("category", category)
        if (!description.isNullOrBlank()) body.put("description", description)
        postAuthVoid("/reports", body)
    }

    /** Uses the captured credential and never refreshes/replays logout under a new account. */
    suspend fun logoutSession(captured: AuthSession, deviceId: String) = withContext(Dispatchers.IO) {
        if (captured.refreshToken.isBlank()) throw ApiException("服务端注销未确认：缺少会话凭证")
        val request = Request.Builder()
            .url(LinLanDomainPoolManager.currentBaseUrl() + "/logout")
            .post(JSONObject().put("refresh_token", captured.refreshToken).put("device_id", deviceId)
                .toString().toRequestBody(jsonMedia)).build()
        client.newBuilder().callTimeout(5, TimeUnit.SECONDS).build().newCall(request).execute().use { response ->
            val result = runCatching { JSONObject(response.body?.string().orEmpty()) }.getOrNull()
            if (!response.isSuccessful || result?.optInt("code", -1) != 0) throw ApiException("服务端注销未确认")
        }
    }

    suspend fun refreshSession(): AuthSession = refreshCoordinator.refresh()

    private suspend fun refreshSessionRemote(current: AuthSession): AuthSession {
        val body = JSONObject()
            .put("refresh_token", current.refreshToken)
            .put("need_user_sig_refresh", true)
            .put("device_id", sessionStore.deviceId())
            .put("device_name", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}")
            .put("platform", "android")
        val data = postData("/auth/refresh", body, auth = false)
        val parsed = parseAuthSession(data)
        return parsed.copy(
            userId = parsed.userId.ifBlank { current.userId },
            userSig = parsed.userSig.ifBlank { current.userSig },
            userSigExpiresAt = parsed.userSigExpiresAt.ifBlank { current.userSigExpiresAt },
        )
    }

    suspend fun getGroupNotice(groupId: String): GroupNoticeResult = withContext(Dispatchers.IO) {
        val url = (LinLanDomainPoolManager.currentBaseUrl() + "/groups/notice")
            .toHttpUrl()
            .newBuilder()
            .addQueryParameter("group_id", groupId)
            .build()
        val request = Request.Builder()
            .url(url)
            .get()
            .header("Authorization", authorizationHeader())
            .build()
        val data = executeForData(request)
        GroupNoticeResult(
            groupId = data.optInt("groupID"),
            content = data.optString("content"),
            summary = data.optString("summary"),
            source = data.optString("source"),
            hasValue = data.optBoolean("hasValue"),
        )
    }

    /** 群主/管理员撤回群成员消息（无客户端 2 分钟时限）。 */
    suspend fun revokeGroupMessage(groupId: String, msgSeq: Long) {
        postAuthVoid(
            "/messages/group/revoke",
            JSONObject()
                .put("group_id", groupId)
                .put("msg_seq", msgSeq),
        )
    }

    suspend fun saveGroupNotice(groupId: String, content: String): GroupNoticeResult =
        withContext(Dispatchers.IO) {
            val data = postData(
                "/groups/notice",
                JSONObject().put("group_id", groupId).put("content", content),
                auth = true,
            )
            GroupNoticeResult(
                groupId = data.optInt("groupID"),
                content = data.optString("content"),
                summary = data.optString("summary"),
                source = data.optString("source"),
                hasValue = data.optBoolean("hasValue"),
            )
        }

    suspend fun searchUsers(keyword: String): List<SearchUserResult> =
        searchUsersInternal("/users/search", keyword)

    suspend fun searchUsersExact(keyword: String): List<SearchUserResult> =
        searchUsersInternal("/users/search-exact", keyword)

    /** Exact first, then fuzzy — same as iOS AddFriend. */
    suspend fun searchUsersPreferExact(keyword: String): List<SearchUserResult> {
        val exact = runCatching { searchUsersExact(keyword) }.getOrDefault(emptyList())
        if (exact.isNotEmpty()) return exact
        return searchUsers(keyword)
    }

    private suspend fun searchUsersInternal(path: String, keyword: String): List<SearchUserResult> =
        withContext(Dispatchers.IO) {
            val url = (LinLanDomainPoolManager.currentBaseUrl() + path)
                .toHttpUrl()
                .newBuilder()
                .addQueryParameter("q", keyword)
                .build()
            val request = Request.Builder()
                .url(url)
                .get()
                .header("Authorization", authorizationHeader())
                .build()
            val data = executeForData(request)
            val users = data.optJSONArray("users")
                ?: data.optJSONArray("list")
                ?: JSONArray()
            buildList {
                for (i in 0 until users.length()) {
                    val item = users.optJSONObject(i) ?: continue
                    val imUserId = item.optString("im_user_id").ifBlank {
                        item.optString("user_id")
                    }
                    if (imUserId.isBlank()) continue
                    add(
                        SearchUserResult(
                            imUserId = imUserId,
                            wananId = item.optString("wanan_id").ifBlank {
                                item.optString("chuanlu_id")
                            }.ifBlank { null },
                            nickname = item.optString("nickname"),
                            userTag = item.optString("user_tag").ifBlank { null },
                            avatarUrl = item.optString("avatar_url").ifBlank { null },
                            canAddFriend = if (item.has("can_add_friend")) {
                                item.optBoolean("can_add_friend", true)
                            } else {
                                true
                            },
                            addBlockedReason = item.optString("add_blocked_reason"),
                        )
                    )
                }
            }
        }

    private suspend fun authRequest(path: String, body: JSONObject): AuthSession = withContext(Dispatchers.IO) {
        val generation = sessionStore.sessionGeneration()
        val data = postData(path, body, auth = false)
        if (data.optBoolean("need_device_approval")) throw DeviceApprovalRequired(data)
        if (data.optBoolean("require_totp_verify") || data.optBoolean("require_totp_setup")) {
            throw ApiException("此账号需要完成验证器验证，请使用已支持该验证方式的客户端登录")
        }
        parseAuthSession(data).also {
            if (!it.isLoggedIn) throw ApiException("登录响应不完整，请重新登录")
            if (!sessionStore.saveSessionIfGeneration(it, generation)) throw CancellationException("登录状态已变化")
        }
    }

    private suspend fun postVoid(path: String, body: JSONObject, auth: Boolean = false) {
        postData(path, body, auth)
    }

    private suspend fun postAuthVoid(path: String, body: JSONObject) {
        postData(path, body, auth = true)
    }

    private suspend fun getAuthData(path: String): JSONObject = withContext(Dispatchers.IO) {
        val request = Request.Builder()
            .url(LinLanDomainPoolManager.currentBaseUrl() + path)
            .get()
            .header("Authorization", authorizationHeader())
            .build()
        executeForData(request)
    }

    private suspend fun postData(path: String, body: JSONObject, auth: Boolean): JSONObject {
        val builder = Request.Builder()
            .url(LinLanDomainPoolManager.currentBaseUrl() + path)
            .post(body.toString().toRequestBody(jsonMedia))
        if (auth) {
            builder.header("Authorization", authorizationHeader())
        }
        return executeForData(builder.build())
    }

    private suspend fun executeForData(request: Request, expectedOwner: String? = null, expectedGeneration: Long? = null): JSONObject = withContext(Dispatchers.IO) {
        LinLanDomainPoolManager.ensureReady()
        val requestGeneration = expectedGeneration ?: sessionStore.sessionGeneration()
        val requestOwner = expectedOwner ?: sessionStore.currentSession().userId
        if (sessionStore.sessionGeneration() != requestGeneration || sessionStore.currentSession().userId != requestOwner) throw CancellationException("账号已变化")
        var activeRequest = request
        val networkDeadline = android.os.SystemClock.elapsedRealtime() + 30_000
        var attempt = 0
        var authRetried = false
        while (true) {
            if (activeRequest.header("Authorization") != null && (requestGeneration != sessionStore.sessionGeneration() || sessionStore.currentSession().userId != requestOwner)) throw CancellationException("账号已切换，请重新操作")
            try {
                val remaining = networkDeadline - android.os.SystemClock.elapsedRealtime()
                if (remaining <= 0) throw ApiException("网络连接超时，请重试")
                val call = client.newCall(activeRequest)
                call.timeout().timeout(DomainPoolPolicy.attemptBudgetMillis(activeRequest.method, attempt, remaining).coerceAtLeast(1), TimeUnit.MILLISECONDS)
                return@withContext call.execute().use { response ->
                val text = response.body?.string().orEmpty()
                if (activeRequest.header("Authorization") != null && (requestGeneration != sessionStore.sessionGeneration() || sessionStore.currentSession().userId != requestOwner)) throw CancellationException("账号已切换，请忽略旧请求结果")
                if (!response.isSuccessful) {
                    val msg = runCatching { JSONObject(text).optString("message") }.getOrNull()
                        ?.takeIf { it.isNotBlank() } ?: "请求失败(${response.code})"
                    if (response.code == 401) throw ApiException(msg, unauthorized = true, status = 401)
                    throw ApiException(msg, status = response.code)
                }
                val root = JSONObject(text)
                if (root.optInt("code", -1) != 0) {
                    val msg = root.optString("message").ifBlank { "请求失败" }
                    throw ApiException(msg, unauthorized = root.optInt("code") == 401)
                }
                root.optJSONObject("data") ?: JSONObject()
                }
            } catch (error: Throwable) {
                if (error is CancellationException) throw error
                if (activeRequest.header("Authorization") != null && (requestGeneration != sessionStore.sessionGeneration() || sessionStore.currentSession().userId != requestOwner)) throw CancellationException("账号已切换，请忽略旧请求结果")
                if (error is ApiException && error.unauthorized && activeRequest.header("Authorization") != null) {
                    val rejected = activeRequest.header("Authorization").orEmpty().removePrefix("Bearer ")
                    if (!authRetried) {
                        authRetried = true
                        val refreshed = refreshCoordinator.refresh(rejected)
                        activeRequest = activeRequest.newBuilder().header("Authorization", AccessTokenPolicy.header(refreshed.token)).build()
                        continue
                    }
                    // This endpoint rejected a newly issued access token. The refresh session is still valid.
                    throw error
                }
                val next = routeRecovery.nextReadRequest(activeRequest, attempt, error, networkDeadline) {
                    sessionStore.sessionGeneration() == requestGeneration && sessionStore.currentSession().userId == requestOwner
                }
                if (next == null) {
                    if (error is ApiException) throw error
                    throw ApiException(error.message ?: "网络异常")
                }
                activeRequest = next
                delay(LinLanReliability.retryDelayMs(attempt))
                attempt++
            }
        }
        @Suppress("UNREACHABLE_CODE")
        throw ApiException("网络异常")
    }

    private fun parseAuthSession(data: JSONObject): AuthSession = AuthSession(
        sdkAppId = data.optInt("sdk_app_id", LinLanAppConstants.SDK_APP_ID),
        userId = data.optString("user_id"),
        userSig = data.optString("user_sig"),
        userSigExpiresAt = data.optString("user_sig_expires_at"),
        token = data.optString("token"),
        tokenExpiresAt = data.optString("token_expires_at"),
        refreshToken = data.optString("refresh_token"),
        mustChangePassword = data.optBoolean("must_change_password"),
    )
}
