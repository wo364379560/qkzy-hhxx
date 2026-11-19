package com.balabalaim.gcloud.tim.push.appdev.data

import android.content.Context
import android.util.Log
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanSessionStore
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import com.balabalaim.gcloud.tim.push.appdev.data.network.AccessTokenPolicy
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanApiClient
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.push.LinLanPushManager
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallManager
import com.tencent.imsdk.v2.V2TIMManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.async
import kotlinx.coroutines.supervisorScope
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

class LinLanAuthRepository(
    private val context: Context,
    private val sessionStore: LinLanSessionStore,
    private val apiClient: LinLanApiClient,
) {
    private val bgScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val imLifecycle = Mutex()
    private val appPrefs = LinLanAppPrefs(context)

    suspend fun bootstrap() {
        runCatching { apiClient.initializeRouting() }
            .onFailure { Log.w(TAG, "domain pool: ${it.message}") }
        val local = sessionStore.currentSession()
        val sdkAppId = local.sdkAppId.takeIf { it > 0 } ?: LinLanAppConstants.SDK_APP_ID
        LinLanRuntimeConfig.applySdkAppId(sdkAppId)
        LinLanImManager.init(context, sdkAppId)
        LinLanRtcCallManager.setup(context)
    }

    suspend fun currentSession(): AuthSession = sessionStore.currentSession()

    private val restoreLock = Mutex()

    suspend fun decideSplashLoggedIn(): Boolean {
        val session = sessionStore.currentSession()
        bgScope.launch { runCatching { apiClient.initializeRouting() } }
        if (AccessTokenPolicy.needsSessionRefresh(session)) {
            try {
                apiClient.refreshSession()
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // A rejected refresh clears its own session. Offline/5xx keeps it for retry.
            }
        }
        val current = sessionStore.currentSession()
        return current.isLoggedIn || (current.userId.isNotBlank() && current.refreshToken.isNotBlank())
    }

    suspend fun ensureSessionReady() = restoreLock.withLock {
        val session = sessionStore.currentSession()
        if (session.userId.isBlank()) return@withLock
        if (AccessTokenPolicy.needsSessionRefresh(session)) {
            val renewed = apiClient.refreshSession()
            if (renewed.userId == session.userId) completeImLogin(renewed)
            return@withLock
        }
        val sdk = session.sdkAppId.takeIf { it > 0 } ?: LinLanAppConstants.SDK_APP_ID
        if (LinLanImManager.isReadyFor(session.userId, sdk)) return@withLock
        if (session.isLoggedIn) {
            try {
                completeImLogin(session)
                return@withLock
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                Log.w(TAG, "Cached IM login failed, refreshing credentials")
            }
        }
        if (sessionStore.currentSession().userId == session.userId) restoreSession()
        Unit
    }

    suspend fun loginWithPassword(phone: String, password: String): AuthSession {
        val session = apiClient.loginWithPassword(phone, password)
        completeImLogin(session)
        appPrefs.accountPhone = phone.trim()
        return session
    }

    suspend fun completeDeviceApproval(challenge: org.json.JSONObject, phone: String) {
        val session = apiClient.exchangeChallenge(challenge)
        completeImLogin(session)
        appPrefs.accountPhone = phone.trim()
    }

    suspend fun loginWithCode(phone: String, code: String): AuthSession {
        val session = apiClient.loginWithCode(phone, code)
        completeImLogin(session)
        appPrefs.accountPhone = phone.trim()
        return session
    }

    suspend fun register(
        phone: String,
        code: String?,
        nickname: String,
        password: String,
        inviteCode: String?,
    ): AuthSession {
        val session = apiClient.register(phone, code, nickname, password, inviteCode)
        completeImLogin(session)
        appPrefs.accountPhone = phone.trim()
        return session
    }

    /**
     * Restore session for splash. Returns null if not logged in OR restore fails,
     * so UI never sticks on splash.
     */
    suspend fun restoreSession(): AuthSession? {
        val session = sessionStore.currentSession()
        if (!session.isLoggedIn && (session.userId.isBlank() || session.refreshToken.isBlank())) return null
        return kotlinx.coroutines.coroutineScope {
            runCatching {
                val refreshed = apiClient.refreshSession()
                completeImLogin(refreshed)
                refreshed
            }.getOrElse { failure ->
                if (failure is com.balabalaim.gcloud.tim.push.appdev.data.network.ApiException && failure.unauthorized) {
                    return@coroutineScope null
                }
                if (failure is CancellationException) throw failure
                if (sessionStore.currentSession().userId != session.userId) return@coroutineScope null
                runCatching {
                    completeImLogin(session)
                    session
                }.getOrNull()
            }
        }
    }

    suspend fun logout() {
        val captured = sessionStore.currentSession()
        val deviceId = sessionStore.deviceId()
        sessionStore.clearSession()
        supervisorScope {
            val remote = async { runCatching { apiClient.logoutSession(captured, deviceId) } }

        if (com.balabalaim.gcloud.tim.push.appdev.run.RunState.session.value.active) {
            context.startService(android.content.Intent(context, com.balabalaim.gcloud.tim.push.appdev.run.RunTrackingService::class.java).setAction("stop"))
        }
        imLifecycle.withLock {
            runCatching { LinLanRtcCallManager.logoutCleanup() }
            LinLanPushManager.unregister()
            val im = withTimeoutOrNull(5_000) { runCatching { LinLanImManager.logout() } }
            if (im == null || im.isFailure) Log.w(TAG, "IM退出未确认")
        }
        if (remote.await().isFailure) Log.w(TAG, "服务端注销未确认，本地已退出")
        }
        appPrefs.accountPhone = ""
    }

    private suspend fun completeImLogin(session: AuthSession) = imLifecycle.withLock {
        val imStarted = System.nanoTime()
        fun requireCurrent(current: AuthSession) {
            if (current.userId != session.userId || current.refreshToken != session.refreshToken) throw CancellationException("登录状态已变化")
        }
        requireCurrent(sessionStore.currentSession())
        val sdkAppId = session.sdkAppId.takeIf { it > 0 } ?: LinLanAppConstants.SDK_APP_ID
        LinLanRuntimeConfig.applySdkAppId(sdkAppId)
        LinLanImManager.init(context, sdkAppId)
        withTimeoutOrNull(20_000) {
            LinLanImManager.login(session.userId, session.userSig)
        } ?: throw Exception("IM 登录超时")
        requireCurrent(sessionStore.currentSession())
        com.balabalaim.gcloud.tim.push.appdev.data.network.NetworkDiagnostics.record(
            "im_ready", java.net.URI(com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanDomainPoolManager.currentBaseUrl()).host,
            (System.nanoTime() - imStarted) / 1_000_000,
        )
        LinLanRtcCallManager.setup(context)
        LinLanRtcCallManager.refreshLoginState(session.userId, session.userSig)
        LinLanPushManager.registerIfAvailable(context, session.userId)
        val profileGeneration = sessionStore.sessionGeneration()
        runCatching {
            val profile = apiClient.fetchProfile()
            com.balabalaim.gcloud.tim.push.appdev.im.LinLanFriendOwnership.publishProfile(profile, profileGeneration, session.userId)
            LinLanRtcCallManager.updateSelfProfile(profile.nickname, profile.avatarUrl)
        }
    }

    companion object {
        private const val TAG = "LinLanAuthRepository"
    }
}
