package com.balabalaim.gcloud.tim.push.appdev.im

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class GroupPermissionRefreshTest {
    @Test fun firstReadWithoutAVersionEntryMustBeAccepted() = runBlocking {
        val memo = PermissionMemo<Boolean>(this)
        assertTrue(memo.get("app:user:new-group") { true })
        assertTrue(memo.get("app:user:new-group") { fail("Expected the confirmed cache"); false })
    }

    @Test fun invalidatedReadAutomaticallyUsesTheNewConfirmedDecision() = runBlocking {
        val memo = PermissionMemo<Boolean>(this)
        val entered = CompletableDeferred<Unit>(); val release = CompletableDeferred<Unit>()
        var reads = 0
        val result = async {
            memo.getCurrent("app:user:group", { true }) {
                reads++
                if (reads == 1) { entered.complete(Unit); release.await(); true } else false
            }
        }
        entered.await(); memo.invalidateGroup("group"); release.complete(Unit)
        assertFalse("Old allowed snapshot must not survive a mute change", result.await())
        assertEquals(2, reads)
    }

    @Test fun simultaneousReadersShareBothTheInvalidatedAndReplacementQueries() = runBlocking {
        val memo = PermissionMemo<Int>(this)
        val entered = CompletableDeferred<Unit>(); val release = CompletableDeferred<Unit>()
        var reads = 0
        val load: suspend () -> Int = { reads++; if (reads == 1) { entered.complete(Unit); release.await() }; reads }
        val a = async { memo.getCurrent("app:user:group", { true }, load = load) }
        entered.await()
        val b = async(start = CoroutineStart.UNDISPATCHED) { memo.getCurrent("app:user:group", { true }, load = load) }
        memo.invalidateGroup("group"); release.complete(Unit)
        assertEquals(2, a.await()); assertEquals(2, b.await()); assertEquals(2, reads)
    }

    @Test fun logoutCannotTurnTheOldReadIntoANewSessionQuery() = runBlocking {
        val memo = PermissionMemo<Int>(this)
        var reads = 0
        val entered = CompletableDeferred<Unit>(); val release = CompletableDeferred<Unit>()
        val result = async {
            runCatching { memo.getCurrent("app:user:group", { true }) { reads++; entered.complete(Unit); release.await(); 1 } }
        }
        entered.await(); memo.clear(); release.complete(Unit)
        assertTrue(result.await().exceptionOrNull() is CancellationException)
        assertEquals(1, reads)
    }

    @Test fun changedAccountCannotPublishEvenAnUninvalidatedSuccessfulReply() = runBlocking {
        val memo = PermissionMemo<Int>(this)
        var current = true
        val result = runCatching { memo.getCurrent("app:user:group", { current }) { current = false; 1 } }
        assertTrue(result.exceptionOrNull() is CancellationException)
    }

    @Test fun temporarySDKFailureCanRecoverButIsNeverCachedAsADenial() = runBlocking {
        val queryScope = CoroutineScope(SupervisorJob() + Dispatchers.Unconfined)
        val memo = PermissionMemo<Boolean>(queryScope)
        var reads = 0
        val result = memo.getCurrent("app:user:group", { true }, retry = { true }) {
            reads++; if (reads == 1) error("membership syncing") else true
        }
        assertTrue(result); assertEquals(2, reads)
        queryScope.cancel()
    }

    @Test fun confirmedMuteRemainsDeniedWithoutExtraQueries() = runBlocking {
        val memo = PermissionMemo<Boolean>(this)
        var reads = 0
        repeat(2) { assertFalse(memo.getCurrent("app:user:group", { true }, retry = { true }) { reads++; false }) }
        assertEquals(1, reads)
    }

    @Test fun repeatedSDKFailureStopsAfterThreeReadAttempts() = runBlocking {
        val queryScope = CoroutineScope(SupervisorJob() + Dispatchers.Unconfined)
        val memo = PermissionMemo<Int>(queryScope)
        var reads = 0
        val result = runCatching { memo.getCurrent("app:user:group", { true }, retry = { true }) { reads++; error("offline") } }
        assertTrue(result.isFailure); assertEquals(3, reads)
        queryScope.cancel()
    }

    @Test fun cancellationNeverProducesAnAllowedFallbackOrAnotherQuery() = runBlocking {
        val memo = PermissionMemo<Int>(this)
        var reads = 0
        val result = runCatching { memo.getCurrent("app:user:group", { true }, retry = { true }) { reads++; throw CancellationException("left chat") } }
        assertTrue(result.exceptionOrNull() is CancellationException); assertEquals(1, reads)
    }
}
