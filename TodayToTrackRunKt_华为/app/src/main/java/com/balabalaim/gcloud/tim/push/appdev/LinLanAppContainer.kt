package com.balabalaim.gcloud.tim.push.appdev

import android.content.Context
import com.balabalaim.gcloud.tim.push.appdev.data.LinLanAuthRepository
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanPinnedMessageStore
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanSessionStore
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanApiClient

object LinLanAppContainer {
    private lateinit var sessionStoreInstance: LinLanSessionStore
    private lateinit var apiClientInstance: LinLanApiClient
    private lateinit var authRepositoryInstance: LinLanAuthRepository
    private lateinit var pinnedMessageStoreInstance: LinLanPinnedMessageStore
    private lateinit var appPrefsInstance: LinLanAppPrefs

    fun init(context: Context) {
        if (::authRepositoryInstance.isInitialized) return
        val app = context.applicationContext
        sessionStoreInstance = LinLanSessionStore(app)
        apiClientInstance = LinLanApiClient(sessionStoreInstance)
        authRepositoryInstance = LinLanAuthRepository(app, sessionStoreInstance, apiClientInstance)
        pinnedMessageStoreInstance = LinLanPinnedMessageStore(app)
        appPrefsInstance = LinLanAppPrefs(app)
    }

    fun authRepository(): LinLanAuthRepository = authRepositoryInstance
    fun apiClient(): LinLanApiClient = apiClientInstance
    fun sessionStore(): LinLanSessionStore = sessionStoreInstance
    fun pinnedMessageStore(): LinLanPinnedMessageStore = pinnedMessageStoreInstance
    fun appPrefs(): LinLanAppPrefs = appPrefsInstance
}
