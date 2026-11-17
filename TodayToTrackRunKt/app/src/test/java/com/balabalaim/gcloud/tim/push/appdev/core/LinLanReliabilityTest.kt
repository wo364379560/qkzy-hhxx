package com.balabalaim.gcloud.tim.push.appdev.core

import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LinLanReliabilityTest {
    @Test
    fun retryPolicy_onlyRetriesFirstIdempotentNetworkFailure() {
        assertTrue(LinLanReliability.shouldRetryRequest("GET", 0, IOException("offline")))
        assertFalse(LinLanReliability.shouldRetryRequest("POST", 0, IOException("offline")))
        assertFalse(LinLanReliability.shouldRetryRequest("GET", 1, IOException("offline")))
        assertFalse(LinLanReliability.shouldRetryRequest("GET", 0, IllegalStateException("bad json")))
        assertEquals(listOf(250L, 500L, 1_000L, 2_000L), (0..3).map(LinLanReliability::retryDelayMs))
    }

    @Test
    fun cacheName_blocksTraversalAndInvalidCharacters() {
        assertEquals("passwd", LinLanReliability.safeCacheName("../../etc/passwd"))
        assertEquals("report_2026_.pdf", LinLanReliability.safeCacheName("folder/report:2026?.pdf"))
        assertEquals("file", LinLanReliability.safeCacheName(".."))
    }

    @Test
    fun largeGroupAccumulatorHandlesOneHundredThousandMembersWithoutDuplicates() {
        val first = (0 until 50_000).map { "user_$it" }
        val overlapAndNext = (25_000 until 100_000).map { "user_$it" }
        val merged = LinLanReliability.appendDistinct(
            current = first,
            incoming = overlapAndNext,
            keyOf = { it },
            limit = 100_000,
        )
        assertEquals(100_000, merged.size)
        assertEquals("user_0", merged.first())
        assertEquals("user_99999", merged.last())
        assertEquals(100_000, merged.toSet().size)
    }

    @Test
    fun forwardSelectionLimitMatchesProductRule() {
        val messages = (1..120).toList()
        val selected = LinLanReliability.appendDistinct(
            current = emptyList(),
            incoming = messages,
            keyOf = { it },
            limit = LinLanReliability.MAX_FORWARD_MESSAGES,
        )
        assertEquals(50, selected.size)
    }
}
