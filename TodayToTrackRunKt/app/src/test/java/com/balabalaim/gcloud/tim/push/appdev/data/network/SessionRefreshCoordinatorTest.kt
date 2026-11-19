package com.balabalaim.gcloud.tim.push.appdev.data.network

import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test
import java.util.concurrent.atomic.AtomicReference

class SessionRefreshCoordinatorTest {
    private val initial = AuthSession(userId = "user", userSig = "sig", token = "old-access", refreshToken = "old-refresh")
    private val refreshed = initial.copy(token = "new-access", refreshToken = "new-refresh")
    private fun coordinator(state: AtomicReference<AuthSession>, refresh: suspend (AuthSession) -> AuthSession) =
        SessionRefreshCoordinator({ state.get() }, refresh, { expected, next -> state.compareAndSet(expected, next) }, { expected -> state.compareAndSet(expected, AuthSession()); Unit })

    @Test fun concurrentUnauthorizedRequestsConsumeRefreshTokenOnlyOnce() = runBlocking {
        val state = AtomicReference(initial)
        var calls = 0
        val coordinator = coordinator(state) { calls++; delay(40); refreshed }
        val sessions = (1..12).map { async { coordinator.refresh("old-access") } }.awaitAll()
        assertEquals(1, calls)
        assertTrue(sessions.all { it.token == "new-access" })
    }

    @Test fun logoutDuringRefreshDoesNotRestorePreviousAccount() = runBlocking {
        val state = AtomicReference(initial)
        val entered = CompletableDeferred<Unit>()
        val release = CompletableDeferred<Unit>()
        val coordinator = coordinator(state) { entered.complete(Unit); release.await(); refreshed }
        val result = async { runCatching { coordinator.refresh() } }
        entered.await(); state.set(AuthSession()); release.complete(Unit)
        assertTrue(result.await().isFailure)
        assertFalse(state.get().isLoggedIn)
    }

    @Test fun interruptedCallerStillSavesRotatedToken() = runBlocking {
        val state = AtomicReference(initial)
        val entered = CompletableDeferred<Unit>()
        val release = CompletableDeferred<Unit>()
        val coordinator = coordinator(state) { entered.complete(Unit); release.await(); refreshed }
        val caller = launch { coordinator.refresh() }
        entered.await(); caller.cancel(); release.complete(Unit); caller.join()
        assertEquals("new-refresh", state.get().refreshToken)
    }

    @Test fun invalidRefreshResponseCannotOverwriteSessionWithEmptyToken() = runBlocking {
        val state = AtomicReference(initial)
        val error = runCatching { coordinator(state) { AuthSession() }.refresh() }.exceptionOrNull()
        assertTrue(error is ApiException && !error.unauthorized)
        assertEquals(initial, state.get())
    }

    @Test fun startupRenewsPartialAndExpiringSessionsOnly() {
        val healthy = initial.copy(tokenExpiresAt = "1970-01-01T01:00:00Z", userSigExpiresAt = "1970-01-01T02:00:00Z")
        assertFalse(AccessTokenPolicy.needsSessionRefresh(AuthSession(), 0))
        assertFalse(AccessTokenPolicy.needsSessionRefresh(healthy, 0))
        assertTrue(AccessTokenPolicy.needsSessionRefresh(healthy.copy(token = ""), 0))
        assertTrue(AccessTokenPolicy.needsSessionRefresh(healthy.copy(userSig = ""), 0))
        assertTrue(AccessTokenPolicy.needsSessionRefresh(healthy.copy(userSigExpiresAt = "1970-01-01T00:00:15Z"), 0))
        assertTrue(AccessTokenPolicy.needsSessionRefresh(healthy.copy(tokenExpiresAt = "1970-01-01T00:00:05Z"), 0))
        assertFalse(AccessTokenPolicy.needsSessionRefresh(healthy.copy(token = "", refreshToken = ""), 0))
    }

    @Test fun partialSessionCanRenewButExplicitRejectionClearsIt() = runBlocking {
        val state = AtomicReference(initial.copy(token = "", userSig = ""))
        assertEquals(refreshed, coordinator(state) { refreshed }.refresh())
        runCatching { coordinator(state) { throw ApiException("expired", unauthorized = true) }.refresh() }
        assertEquals(AuthSession(), state.get())
    }

    @Test fun temporaryNetworkFailureKeepsSession() = runBlocking {
        val state = AtomicReference(initial)
        runCatching { coordinator(state) { throw ApiException("offline") }.refresh() }
        assertEquals(initial, state.get())
    }

    @Test fun temporaryStatusCodesNeverClearBusinessSession() = runBlocking {
        for (status in listOf(429, 500, 502, 503, 504)) {
            val state = AtomicReference(initial)
            runCatching { coordinator(state) { throw ApiException("temporary", status = status, providerCode = "SESSION_TEMPORARY") }.refresh() }
            assertEquals(initial, state.get())
        }
    }

    @Test fun lateInvalidRefreshCannotClearOrRejectNewLogin() = runBlocking {
        val state = AtomicReference(initial)
        val entered = CompletableDeferred<Unit>(); val release = CompletableDeferred<Unit>()
        val manager = coordinator(state) { entered.complete(Unit); release.await(); throw ApiException("invalid", unauthorized = true, status = 401, providerCode = "SESSION_REVOKED") }
        val result = async { manager.refresh() }
        entered.await(); state.set(refreshed); release.complete(Unit)
        assertEquals(refreshed, result.await()); assertEquals(refreshed, state.get())
    }

    @Test fun emptyOrMalformedTokenNeverProducesBearerOnlyHeader() {
        listOf("", "  ", "Bearer duplicate", "token\nvalue").forEach { token ->
            assertTrue(runCatching { AccessTokenPolicy.header(token) }.exceptionOrNull() is ApiException)
        }
        assertEquals("Bearer valid-token", AccessTokenPolicy.header(" valid-token "))
    }

    @Test fun refreshesBeforeExpiryButNotForValidAccessToken() {
        assertTrue(AccessTokenPolicy.needsRefresh(initial.copy(token = ""), 0))
        assertTrue(AccessTokenPolicy.needsRefresh(initial.copy(tokenExpiresAt = "1970-01-01T00:00:30Z"), 0))
        assertFalse(AccessTokenPolicy.needsRefresh(initial.copy(tokenExpiresAt = "1970-01-01T01:00:00Z"), 0))
    }
}
