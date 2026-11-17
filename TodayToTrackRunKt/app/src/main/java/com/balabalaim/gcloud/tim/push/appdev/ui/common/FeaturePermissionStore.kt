package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.content.Context
import android.content.pm.PackageManager
import androidx.core.content.ContextCompat

class FeaturePermissionStore(context: Context, name: String = "rui_feature_permissions_v1") {
    private val app = context.applicationContext
    private val prefs = app.getSharedPreferences(name, Context.MODE_PRIVATE)
    val policy = FeaturePermissionPolicy(
        readChoice = ::choice,
        writeChoice = { feature, value -> prefs.edit().putString(feature.key, value.name).apply() },
        wasSystemRequested = { prefs.getBoolean("requested:$it", false) },
        writeSystemRequested = { permission, value -> prefs.edit().putBoolean("requested:$permission", value).apply() },
    )

    fun choice(feature: FeaturePermission): FeaturePermissionPolicy.Choice = runCatching {
        FeaturePermissionPolicy.Choice.valueOf(prefs.getString(feature.key, "UNKNOWN") ?: "UNKNOWN")
    }.getOrDefault(FeaturePermissionPolicy.Choice.UNKNOWN)

    fun granted(permissions: List<String>): Set<String> = permissions.filter {
        ContextCompat.checkSelfPermission(app, it) == PackageManager.PERMISSION_GRANTED
    }.toSet()

    fun action(features: List<FeaturePermission>, permissions: List<String>) = policy.action(features, permissions, granted(permissions))
}
