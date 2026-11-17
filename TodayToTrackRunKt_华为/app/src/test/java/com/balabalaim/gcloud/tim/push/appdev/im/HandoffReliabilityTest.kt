package com.balabalaim.gcloud.tim.push.appdev.im

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class HandoffReliabilityTest {
    data class Message(val id: String, val time: Long, val state: String = "history", val sequence: Long = 0)
    @Test fun lateHistoryKeepsLiveMessagesAndRevokedState() {
        val live = listOf(Message("a", 1, "revoked"), Message("new1", 4), Message("new2", 5), Message("new3", 6))
        val merged = mergeHistoryTimeline(live, listOf(Message("a", 1), Message("old", 0), Message("removed", 2)), { it.id }, { it.time }, removedIDs = setOf("removed"))
        assertEquals(listOf("old", "a", "new1", "new2", "new3"), merged.map { it.id })
        assertEquals("revoked", merged.first { it.id == "a" }.state)
    }
    @Test fun messagesWithinOneSecondUseSDKSequenceInsteadOfRandomIDs() {
        val values = listOf(Message("z-first", 100, sequence = 1), Message("a-second", 100, sequence = 2))
        assertEquals(listOf("z-first", "a-second"), mergeHistoryTimeline(emptyList(), values, { it.id }, { it.time }, sequence = { it.sequence }).map { it.id })
    }
    @Test fun hiddenPagesAdvanceRawCursorAndStayBounded() = runBlocking {
        var calls = 0
        val pager = RawHistoryPager(2, { it: Int -> it.toString() }, { it < 2 }) { cursor: Int? ->
            calls++; val start = cursor?.minus(1) ?: 12
            if (start < 1) emptyList() else listOf(start, start - 1)
        }
        assertTrue(pager.load(true) { true }.isEmpty()); assertEquals(4, calls); assertTrue(pager.hasMore)
        val more = pager.load(false) { true }; assertTrue(more.contains(1)); assertTrue(calls > 4)
    }
    @Test fun changedScopeCannotCommitCursor() = runBlocking {
        var active = true
        val pager = RawHistoryPager(2, { it: Int -> it.toString() }, { true }) { _: Int? -> active = false; listOf(2, 1) }
        assertTrue(runCatching { pager.load(true) { active } }.exceptionOrNull() is CancellationException)
        assertNull(pager.cursor)
    }
    @Test fun groupTopicsAndPrivateMessagesNeverMix() {
        assertTrue(matchesChatConversation("group_@TGS#_a", "@TGS#_a", null, "friend"))
        assertFalse(matchesChatConversation("group_@TGS#_a", "@TGS#_a@TOPIC#_one", null, "friend"))
        assertFalse(matchesChatConversation("group_@TGS#_a@TOPIC#_one", "@TGS#_a@TOPIC#_two", null, "friend"))
        assertFalse(matchesChatConversation("c2c_friend", "other-group", null, "friend"))
    }
    @Test fun oneVideoFailureDoesNotRepeatSuccesses() = runBlocking {
        val success = mutableListOf<Int>(); val failures = mutableListOf<Int>()
        runIndependentVideoItems(listOf(1, 2, 3), { true }, { if (it == 2) error("bad video") else success += it }, { item, _ -> failures += item })
        assertEquals(listOf(1, 3), success); assertEquals(listOf(2), failures)
        runIndependentVideoItems(failures.toList(), { true }, { success += it }, { _, _ -> fail() })
        assertEquals(listOf(1, 3, 2), success)
        assertFalse(VideoBatchPolicy.canRetry("success")); assertFalse(VideoBatchPolicy.canRetry("unknown")); assertTrue(VideoBatchPolicy.canRetry("failed"))
    }
    @Test fun videoCancelStopsRemainingItemsAndCapacityIsBounded() = runBlocking {
        var active = true; val sent = mutableListOf<Int>()
        runIndependentVideoItems(listOf(1, 2, 3), { active }, { sent += it; active = false }, { _, _ -> fail() })
        assertEquals(listOf(1), sent)
        assertTrue(runCatching { VideoBatchPolicy.validateSize(600L * 1024 * 1024, 0, VideoBatchPolicy.MAX_FILE_BYTES) }.isFailure)
        assertTrue(runCatching { VideoBatchPolicy.validateSize(400, VideoBatchPolicy.MAX_TOTAL_BYTES - 100, 500) }.isFailure)
    }
    @Test fun targetConcurrencyHasSharedPacingAndNoTrailingDelay() = runBlocking {
        var clock = 0L; val starts = mutableListOf<Long>(); var inFlight = 0; var highWater = 0
        val pacer = AccountForwardPacer(500, { clock }) { clock += it }
        val byTarget = mutableMapOf<Int, MutableList<Int>>()
        runBoundedForwardTargets((1..10).toList(), 3, { true }, process = { target ->
            repeat(5) { index ->
                pacer.acquire { true }; starts += clock; inFlight++; highWater = maxOf(highWater, inFlight)
                yield(); byTarget.getOrPut(target) { mutableListOf() } += index; inFlight--
            }
        }, failed = { _, error -> throw error })
        assertTrue(highWater <= 3); assertTrue(highWater > 1)
        assertEquals(50, starts.size); assertTrue(starts.zipWithNext().all { (a, b) -> b - a >= 500 })
        assertEquals(49 * 500L, clock)
        assertTrue(byTarget.values.all { it == listOf(0, 1, 2, 3, 4) })
    }
    @Test fun targetFailureIsIsolatedAndPermissionInvalidationFencesLateReads() = runBlocking {
        val completed = mutableListOf<Int>(); val failures = mutableListOf<Int>()
        runBoundedForwardTargets(listOf(1, 2, 3), 2, { true }, { if (it == 2) error("denied") else completed += it }, { item, _ -> failures += item })
        assertEquals(setOf(1, 3), completed.toSet()); assertEquals(listOf(2), failures)
        val gate = CompletableDeferred<Unit>(); var reads = 0; var clock = 0L
        val memo = PermissionMemo<Int>(this, 100) { clock }
        val a = async { runCatching { memo.get("app:owner:group") { reads++; gate.await(); 1 } } }
        val b = async { runCatching { memo.get("app:owner:group") { reads++; gate.await(); 1 } } }
        repeat(5) { yield() }; assertEquals(1, reads)
        memo.invalidateGroup("group"); gate.complete(Unit)
        assertTrue(a.await().exceptionOrNull() is PermissionChanged); assertTrue(b.await().exceptionOrNull() is PermissionChanged)
        assertEquals(2, memo.get("app:owner:group") { reads++; 2 })
        assertEquals(2, memo.get("app:owner:group") { fail(); 3 })
        clock = 101
        assertEquals(4, memo.get("app:owner:group") { 4 })
    }
}
