package com.balabalaim.gcloud.tim.push.appdev.run

/** Reject stale GPS fixes, poor accuracy, stationary jitter and impossible running speeds. */
object RunRules {
    fun accept(accuracyMeters: Float, ageSeconds: Double, hasPrevious: Boolean, elapsedSeconds: Double, distanceMeters: Double): Boolean {
        if (!accuracyMeters.isFinite() || accuracyMeters < 0 || accuracyMeters > 40 || ageSeconds !in 0.0..15.0) return false
        if (!hasPrevious) return true
        if (!elapsedSeconds.isFinite() || elapsedSeconds <= 0) return false
        // After a GPS gap, record a new segment without connecting the unknown route.
        if (elapsedSeconds > 30) return true
        return distanceMeters.isFinite() && distanceMeters >= 3 && distanceMeters / elapsedSeconds <= 12
    }
}
