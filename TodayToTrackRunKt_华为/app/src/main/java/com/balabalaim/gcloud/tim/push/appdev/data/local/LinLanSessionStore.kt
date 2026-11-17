package com.balabalaim.gcloud.tim.push.appdev.data.local

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.intPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import java.util.UUID

private val Context.sessionDataStore by preferencesDataStore("auth_session")

class LinLanSessionStore(private val context: Context) {
    private val generation = java.util.concurrent.atomic.AtomicLong()
    fun sessionGeneration(): Long = generation.get()
    private object Keys {
        val SDK_APP_ID = intPreferencesKey("sdk_app_id")
        val USER_ID = stringPreferencesKey("user_id")
        val USER_SIG = stringPreferencesKey("user_sig")
        val USER_SIG_EXPIRES = stringPreferencesKey("user_sig_expires_at")
        val TOKEN = stringPreferencesKey("token")
        val TOKEN_EXPIRES = stringPreferencesKey("token_expires_at")
        val REFRESH_TOKEN = stringPreferencesKey("refresh_token")
        val DEVICE_ID = stringPreferencesKey("device_id")
    }

    val sessionFlow: Flow<AuthSession> = context.sessionDataStore.data.map { prefs ->
        AuthSession(
            sdkAppId = prefs[Keys.SDK_APP_ID] ?: 0,
            userId = prefs[Keys.USER_ID].orEmpty(),
            userSig = prefs[Keys.USER_SIG].orEmpty(),
            userSigExpiresAt = prefs[Keys.USER_SIG_EXPIRES].orEmpty(),
            token = prefs[Keys.TOKEN].orEmpty(),
            tokenExpiresAt = prefs[Keys.TOKEN_EXPIRES].orEmpty(),
            refreshToken = prefs[Keys.REFRESH_TOKEN].orEmpty(),
        )
    }

    suspend fun currentSession(): AuthSession = sessionFlow.first()

    suspend fun saveSession(session: AuthSession) {
        context.sessionDataStore.edit { prefs ->
            prefs[Keys.SDK_APP_ID] = session.sdkAppId
            prefs[Keys.USER_ID] = session.userId
            prefs[Keys.USER_SIG] = session.userSig
            prefs[Keys.USER_SIG_EXPIRES] = session.userSigExpiresAt
            prefs[Keys.TOKEN] = session.token
            prefs[Keys.TOKEN_EXPIRES] = session.tokenExpiresAt
            prefs[Keys.REFRESH_TOKEN] = session.refreshToken
        }
    }

    suspend fun replaceSessionIfCurrent(expected: AuthSession, next: AuthSession): Boolean {
        var replaced = false
        context.sessionDataStore.edit { prefs ->
            if (prefs[Keys.USER_ID].orEmpty() != expected.userId ||
                prefs[Keys.TOKEN].orEmpty() != expected.token ||
                prefs[Keys.REFRESH_TOKEN].orEmpty() != expected.refreshToken) return@edit
            prefs[Keys.SDK_APP_ID] = next.sdkAppId
            prefs[Keys.USER_ID] = next.userId
            prefs[Keys.USER_SIG] = next.userSig
            prefs[Keys.USER_SIG_EXPIRES] = next.userSigExpiresAt
            prefs[Keys.TOKEN] = next.token
            prefs[Keys.TOKEN_EXPIRES] = next.tokenExpiresAt
            prefs[Keys.REFRESH_TOKEN] = next.refreshToken
            replaced = true
        }
        return replaced
    }

    suspend fun saveSessionIfGeneration(session: AuthSession, expected: Long): Boolean {
        var saved = false
        context.sessionDataStore.edit { prefs ->
            if (generation.get() != expected) return@edit
            prefs[Keys.SDK_APP_ID] = session.sdkAppId
            prefs[Keys.USER_ID] = session.userId
            prefs[Keys.USER_SIG] = session.userSig
            prefs[Keys.USER_SIG_EXPIRES] = session.userSigExpiresAt
            prefs[Keys.TOKEN] = session.token
            prefs[Keys.TOKEN_EXPIRES] = session.tokenExpiresAt
            prefs[Keys.REFRESH_TOKEN] = session.refreshToken
            saved = true
        }
        return saved
    }

    suspend fun clearSession() {
        generation.incrementAndGet()
        context.sessionDataStore.edit { prefs ->
            prefs.remove(Keys.SDK_APP_ID)
            prefs.remove(Keys.USER_ID)
            prefs.remove(Keys.USER_SIG)
            prefs.remove(Keys.USER_SIG_EXPIRES)
            prefs.remove(Keys.TOKEN)
            prefs.remove(Keys.TOKEN_EXPIRES)
            prefs.remove(Keys.REFRESH_TOKEN)
        }
    }

    suspend fun deviceId(): String {
        val existing = context.sessionDataStore.data.first()[Keys.DEVICE_ID]
        if (!existing.isNullOrBlank()) return existing
        val created = UUID.randomUUID().toString()
        context.sessionDataStore.edit { prefs -> prefs[Keys.DEVICE_ID] = created }
        return created
    }
}
