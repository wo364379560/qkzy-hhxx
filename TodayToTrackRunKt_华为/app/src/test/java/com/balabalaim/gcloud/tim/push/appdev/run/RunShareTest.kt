package com.balabalaim.gcloud.tim.push.appdev.run

import org.junit.Assert.*
import org.junit.Test

class RunShareTest {
    private fun run() = RunSession(startedAt = 100000, durationSeconds = 600, distanceMeters = 1800.0,
        owner = "private-account", points = (0..1000).map { TrackPoint(30.123456 + it * .00001, 120.654321 + it * .00001, it.toLong(), if (it < 500) 0 else 1) })
    @Test fun sharedShapeIsBoundedAndNeverIncludesGpsOrOwner() {
        val raw = RunShare.from(run(), true).encode()
        assertFalse(raw.contains("latitude")); assertFalse(raw.contains("longitude")); assertFalse(raw.contains("private-account"))
        assertFalse(raw.contains("30.123456")); assertFalse(raw.contains("120.654321"))
        assertTrue(raw.toByteArray().size < 8192)
        val card = RunShare.decode(raw)!!
        assertTrue(card.route.size <= 96); assertEquals(setOf(0, 1), card.route.map { it.segment }.toSet())
        assertEquals(1800.0, card.meters, 0.01); assertEquals(600L, card.seconds)
    }
    @Test fun sharingWithoutShapeRetainsOnlySummary() {
        val card = RunShare.decode(RunShare.from(run(), false).encode())!!
        assertTrue(card.route.isEmpty()); assertTrue(card.summary.contains("1.80 公里"))
    }
    @Test fun malformedOrUnsupportedPayloadsFallBackToText() {
        listOf(null, "", "{}", "[]", "broken", "{\"rzh_run_v1\":{\"startedAt\":1,\"seconds\":-1,\"meters\":0}}")
            .forEach { assertNull(RunShare.decode(it)) }
        val bad = RunShare(1, 1, 1.0, listOf(RoutePosition(2f, 0f, 0))).encode()
        assertNull(RunShare.decode(bad))
    }
    @Test fun cannotShareAnActiveOrEmptyRecording() {
        assertTrue(runCatching { RunShare.from(run().copy(active = true), true) }.isFailure)
        assertTrue(runCatching { RunShare.from(run().copy(points = emptyList()), true) }.isFailure)
    }
    @Test fun explicitLocationSharingRoundTripsStartAndEnd() {
        val original = run()
        val card = RunShare.decode(RunShare.from(original, true, true).encode())!!
        assertEquals(original.points.first().latitude, card.start!!.latitude, .00000001)
        assertEquals(original.points.last().longitude, card.end!!.longitude, .00000001)
        val private = RunShare.decode(RunShare.from(original, true).encode())!!
        assertNull(private.start); assertNull(private.end)
    }
    @Test fun rejectsOutOfRangeCoordinatesAndSupportsLegacyCards() {
        val bad = RunShare(1, 1, 10.0, emptyList(), RunLocation(100.0, 120.0)).encode()
        assertNull(RunShare.decode(bad))
        val legacy = """{"rzh_run_v1":{"startedAt":100,"seconds":60,"meters":200,"route":[]}}"""
        assertNotNull(RunShare.decode(legacy))
    }
}
