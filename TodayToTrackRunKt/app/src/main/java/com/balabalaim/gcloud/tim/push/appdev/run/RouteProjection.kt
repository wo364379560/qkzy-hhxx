package com.balabalaim.gcloud.tim.push.appdev.run

import kotlin.math.cos

data class RoutePosition(val x: Float, val y: Float, val segment: Int)

/** Local GPS route projection, north up, with equal scale on both axes. */
object RouteProjection {
    fun project(points: List<TrackPoint>): List<RoutePosition> {
        if (points.isEmpty()) return emptyList()
        val latitude = points.map { it.latitude }.average()
        val origin = points.first().longitude
        val longitudeScale = cos(Math.toRadians(latitude)).coerceAtLeast(0.000001)
        val xs = points.map {
            // Keep a local route continuous when it crosses the date line.
            val delta = ((it.longitude - origin + 540.0) % 360.0) - 180.0
            delta * longitudeScale * 111_320.0
        }
        val ys = points.map { (it.latitude - latitude) * 111_320.0 }
        val centerX = (xs.min() + xs.max()) / 2
        val centerY = (ys.min() + ys.max()) / 2
        val span = maxOf(xs.max() - xs.min(), ys.max() - ys.min(), 20.0)
        return points.indices.map { i ->
            RoutePosition((0.5 + (xs[i] - centerX) / span * 0.8).toFloat(),
                (0.5 - (ys[i] - centerY) / span * 0.8).toFloat(), points[i].segment)
        }
    }
}
