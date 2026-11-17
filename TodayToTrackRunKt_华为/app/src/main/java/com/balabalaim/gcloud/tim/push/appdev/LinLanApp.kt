package com.balabalaim.gcloud.tim.push.appdev

import android.app.Application
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanDomainPoolManager
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanMessageAlertCenter
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcCallManager

class LinLanApp : Application() {
    override fun onCreate() {
        super.onCreate()
        LinLanDomainPoolManager.init(this)
        LinLanAppContainer.init(this)
        LinLanRtcCallManager.setup(this)
        LinLanMessageAlertCenter.start(this)
    }
}
