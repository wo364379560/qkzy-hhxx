package com.balabalaim.gcloud.tim.push.appdev.run

import org.junit.Assert.*
import org.junit.Test

class RouteProjectionTest {
    private fun point(lat: Double, lon: Double, segment: Int = 0) = TrackPoint(lat, lon, 0, segment)

    @Test fun emptyAndStationaryRoutesStayFiniteAndCentered() {
        assertTrue(RouteProjection.project(emptyList()).isEmpty())
        for (points in listOf(listOf(point(30.0, 120.0)), List(3) { point(30.0, 120.0) })) {
            RouteProjection.project(points).forEach {
                assertEquals(0.5f, it.x, 0.0001f)
                assertEquals(0.5f, it.y, 0.0001f)
            }
        }
    }

    @Test fun northIsUpEastIsRightAndRouteFits() {
        val result = RouteProjection.project(listOf(point(30.0, 120.0), point(30.001, 120.001)))
        assertTrue(result.last().x > result.first().x)
        assertTrue(result.last().y < result.first().y)
        result.forEach { assertTrue(it.x in 0.09f..0.91f && it.y in 0.09f..0.91f) }
    }

    @Test fun preservesAspectRatioAndPauseSegments() {
        val result = RouteProjection.project(listOf(point(0.0, 0.0), point(0.0, 0.002), point(0.001, 0.002, 1)))
        assertEquals(2f, (result[1].x - result[0].x) / (result[1].y - result[2].y), 0.001f)
        assertEquals(listOf(0, 0, 1), result.map { it.segment })
    }

    @Test fun dateLineCrossingRemainsALocalEastwardRoute() {
        val result = RouteProjection.project(listOf(point(0.0, 179.999), point(0.0, -179.999)))
        assertTrue(result.last().x > result.first().x)
        assertEquals(result.first().y, result.last().y, 0.0001f)
    }
}
