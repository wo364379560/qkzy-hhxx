package com.balabalaim.gcloud.tim.push.appdev.run

import org.json.JSONArray
import org.json.JSONObject
import java.util.Locale
import kotlin.math.roundToInt

data class RunLocation(val latitude: Double, val longitude: Double)

/** Relative shape with explicitly opted-in start/end positions. */
data class RunShare(val startedAt: Long, val seconds: Long, val meters: Double, val route: List<RoutePosition>, val start: RunLocation? = null, val end: RunLocation? = null) {
    val distance: String get() = String.format(Locale.US, "%.2f 公里", meters / 1000)
    val duration: String get() = String.format(Locale.US, "%02d:%02d:%02d", seconds / 3600, seconds / 60 % 60, seconds % 60)
    val pace: String get() = if (meters < 10) "—" else (seconds * 1000 / meters).toLong().let { "${it / 60}′${(it % 60).toString().padStart(2, '0')}″ / 公里" }
    val summary: String get() = "[运动打卡] $distance · $duration\n平均配速 $pace\n一起动起来，每一步都算数。"

    fun encode(): String = JSONObject().put("businessID", "rzh_run_v2").put("rzh_run_v1", JSONObject()
        .put("start", start?.let { JSONArray().put(it.latitude).put(it.longitude) })
        .put("end", end?.let { JSONArray().put(it.latitude).put(it.longitude) })
        .put("startedAt", startedAt).put("seconds", seconds).put("meters", meters)
        .put("route", JSONArray().apply { route.forEach { p ->
            put(JSONArray().put((p.x * 1000).roundToInt()).put((p.y * 1000).roundToInt()).put(p.segment))
        } })).toString()

    companion object {
        fun from(run: RunSession, includeShape: Boolean, includeLocations: Boolean = false): RunShare {
            require(!run.active && run.points.isNotEmpty() && run.durationSeconds > 0) { "请先完成并保存一段运动" }
            val positions = if (includeShape) RouteProjection.project(run.points) else emptyList()
            // Sample each segment separately so pauses never become a connecting line.
            val sampled = if (positions.size <= 96) positions else {
                val stride = kotlin.math.ceil(positions.size / 90.0).toInt()
                positions.filterIndexed { index, _ -> index % stride == 0 }.take(95) + positions.last()
            }
            return RunShare(run.startedAt, run.durationSeconds, run.distanceMeters, sampled,
                if (includeLocations) run.points.first().let { RunLocation(it.latitude, it.longitude) } else null,
                if (includeLocations) run.points.last().let { RunLocation(it.latitude, it.longitude) } else null)
        }
        fun decode(raw: String?): RunShare? = runCatching {
            if (raw.isNullOrBlank() || raw.length > 8192) return null
            val obj = JSONObject(raw).getJSONObject("rzh_run_v1")
            val time = obj.getLong("startedAt"); val seconds = obj.getLong("seconds"); val meters = obj.getDouble("meters")
            require(time > 0 && seconds in 1..604800 && meters.isFinite() && meters in 0.0..2_000_000.0)
            val array = obj.optJSONArray("route") ?: JSONArray()
            require(array.length() <= 96)
            var previousSegment = -1
            val route = (0 until array.length()).map { i ->
                val p = array.getJSONArray(i)
                val x = p.getDouble(0); val y = p.getDouble(1); val segment = p.getInt(2)
                require(x.isFinite() && y.isFinite() && x in 0.0..1000.0 && y in 0.0..1000.0 && segment >= previousSegment && segment >= 0)
                previousSegment = segment
                RoutePosition((x / 1000).toFloat(), (y / 1000).toFloat(), segment)
            }
            fun location(key: String): RunLocation? {
                if (!obj.has(key)) return null
                val p = obj.getJSONArray(key)
                val lat = p.getDouble(0); val lon = p.getDouble(1)
                require(lat.isFinite() && lon.isFinite() && lat in -90.0..90.0 && lon in -180.0..180.0)
                return RunLocation(lat, lon)
            }
            RunShare(time, seconds, meters, route, location("start"), location("end"))
        }.getOrNull()
    }
}
