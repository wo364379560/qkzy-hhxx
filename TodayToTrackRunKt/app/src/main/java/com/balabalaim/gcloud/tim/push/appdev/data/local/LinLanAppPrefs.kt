package com.balabalaim.gcloud.tim.push.appdev.data.local

import android.content.Context

class LinLanAppPrefs(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences("app_prefs", Context.MODE_PRIVATE)

    var privacyAccepted: Boolean
        get() = prefs.getBoolean(KEY_PRIVACY, false)
        set(value) = prefs.edit().putBoolean(KEY_PRIVACY, value).apply()

    var teenModeEnabled: Boolean
        get() = prefs.getBoolean(KEY_TEEN, false)
        set(value) = prefs.edit().putBoolean(KEY_TEEN, value).apply()

    var teenModePin: String
        get() = prefs.getString(KEY_TEEN_PIN, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_TEEN_PIN, value).apply()

    var messageNotifyEnabled: Boolean
        get() = prefs.getBoolean(KEY_NOTIFY, true)
        set(value) = prefs.edit().putBoolean(KEY_NOTIFY, value).apply()

    var soundNotifyEnabled: Boolean
        get() = prefs.getBoolean(KEY_SOUND, true)
        set(value) = prefs.edit().putBoolean(KEY_SOUND, value).apply()

    /** The phone used for the current account. The chat profile API intentionally does not expose it. */
    var accountPhone: String
        get() = prefs.getString(KEY_ACCOUNT_PHONE, "").orEmpty()
        set(value) = prefs.edit().putString(KEY_ACCOUNT_PHONE, value).apply()

    companion object {
        private const val KEY_PRIVACY = "privacy_accepted"
        private const val KEY_TEEN = "teen_mode"
        private const val KEY_TEEN_PIN = "teen_pin"
        private const val KEY_NOTIFY = "msg_notify"
        private const val KEY_SOUND = "sound_notify"
        private const val KEY_ACCOUNT_PHONE = "account_phone"
    }
}
