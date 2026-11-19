package com.balabalaim.gcloud.tim.push.appdev.data.network

import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import java.text.SimpleDateFormat
import java.util.Locale

/** A rotating refresh token may be consumed only once, including during splash cancellation. */
internal class SessionRefreshCoordinator(
    private val read: suspend () -> AuthSession,
    private val refreshRemote: suspend (AuthSession) -> AuthSession,
    private val save: suspend (AuthSession, AuthSession) -> Boolean,
    private val clear: suspend (AuthSession) -> Unit,
) {
    private val mutex = Mutex()

    suspend fun refresh(rejectedToken: String? = null): AuthSession {
        val observed = read()
        return withContext(NonCancellable + Dispatchers.IO) {
            mutex.withLock {
                val current = read()
                if (current.userId != observed.userId) throw kotlinx.coroutines.CancellationException("账号已切换")
                if (current.isLoggedIn && current.refreshToken != observed.refreshToken) return@withLock current
                if (current.isLoggedIn && rejectedToken != null && current.token != rejectedToken) return@withLock current
                if (current.refreshToken.isBlank()) {
                    if (read() == current) clear(current)
                    throw ApiException("登录已失效，请重新登录", unauthorized = true)
                }
                try {
                    val refreshed = refreshRemote(current)
                    if (!refreshed.isLoggedIn || refreshed.refreshToken.isBlank() || refreshed.userId != current.userId) {
                        throw ApiException("续签响应不完整，请稍后重试")
                    }
                    // A logout or account switch while HTTP is in flight must not be undone.
                    val latest = read()
                    if (latest != current) {
                        if (latest.isLoggedIn && latest.userId == current.userId) return@withLock latest
                        throw ApiException("登录已失效，请重新登录", unauthorized = true)
                    }
                    if (save(current, refreshed)) refreshed else {
                        val changed = read()
                        if (changed.isLoggedIn && changed.userId == current.userId) changed else throw ApiException("登录已失效，请重新登录", unauthorized = true)
                    }
                } catch (error: ApiException) {
                    val latest = read()
                    if (latest != current) {
                        if (latest.isLoggedIn && latest.userId == current.userId) return@withLock latest
                        throw kotlinx.coroutines.CancellationException("登录状态已变化")
                    }
                    if (error.unauthorized) clear(current)
                    throw error
                }
            }
        }
    }
}

internal object AccessTokenPolicy {
    fun needsRefresh(session: AuthSession, now: Long = System.currentTimeMillis()): Boolean {
        if (session.token.isBlank()) return true
        return expiresSoon(session.tokenExpiresAt, now)
    }

    fun needsSessionRefresh(session: AuthSession, now: Long = System.currentTimeMillis()): Boolean =
        session.userId.isNotBlank() && session.refreshToken.isNotBlank() &&
            (needsRefresh(session, now) || session.userSig.isBlank() || expiresSoon(session.userSigExpiresAt, now))

    private fun expiresSoon(timestamp: String, now: Long): Boolean {
        if (timestamp.isBlank()) return false
        val expiry = listOf("yyyy-MM-dd'T'HH:mm:ssXXX", "yyyy-MM-dd'T'HH:mm:ss.SSSXXX")
            .firstNotNullOfOrNull { pattern -> runCatching { SimpleDateFormat(pattern, Locale.US).parse(timestamp)?.time }.getOrNull() }
        return expiry != null && expiry <= now + 60_000
    }

    fun header(token: String): String {
        val value = token.trim()
        if (value.isEmpty() || value.any { it.isWhitespace() }) throw ApiException("登录已失效，请重新登录", unauthorized = true)
        return "Bearer $value"
    }
}
