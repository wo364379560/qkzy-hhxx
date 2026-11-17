package com.balabalaim.gcloud.tim.push.appdev.core

/** Runtime values signed and returned by the business server. */
object LinLanRuntimeConfig {
    @Volatile
    var sdkAppId: Int = LinLanAppConstants.SDK_APP_ID
        private set

    fun applySdkAppId(value: Int) {
        sdkAppId = value.takeIf { it > 0 } ?: LinLanAppConstants.SDK_APP_ID
    }
}
