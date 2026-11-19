package com.balabalaim.gcloud.tim.push.appdev.data.network

import android.content.Context
import com.alibaba.sdk.android.httpdns.HttpDns
import com.alibaba.sdk.android.httpdns.HttpDnsService
import com.alibaba.sdk.android.httpdns.InitConfig
import com.alibaba.sdk.android.httpdns.RequestIpType
import com.balabalaim.gcloud.tim.push.appdev.BuildConfig
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import okhttp3.Dns

internal object LinLanHTTPDNS {
    @Volatile private var context: Context? = null
    @Volatile private var service: HttpDnsService? = null
    val dns: Dns = ScopedHttpDns(::resolve, diagnostic = { host, stage, elapsed -> NetworkDiagnostics.record(stage, host, elapsed) })

    fun configure(context: Context) { this.context = context.applicationContext }

    fun prepareAfterConsent() { runCatching { serviceAfterConsent() } }

    @Synchronized private fun serviceAfterConsent(): HttpDnsService? {
        service?.let { return it }
        val app = context ?: return null
        if (!LinLanAppPrefs(app).privacyAccepted || BuildConfig.HTTPDNS_ACCOUNT_ID.isBlank() || BuildConfig.HTTPDNS_SECRET_KEY.isBlank()) return null
        val config = InitConfig.Builder().setContext(app).setSecretKey(BuildConfig.HTTPDNS_SECRET_KEY)
            // Keep SDK TTL memory caching; disk cache restored empty answers on the tested Huawei/Android 10 device.
            .setEnableHttps(true).setEnableCacheIp(false).setEnableExpiredIp(false)
            .setEnableDegradationLocalDns(false).setTimeoutMillis(2_000)
            .setPreResolveAfterNetworkChanged(true).build()
        com.alibaba.sdk.android.httpdns.log.HttpDnsLog.enable(false)
        HttpDns.init(BuildConfig.HTTPDNS_ACCOUNT_ID, config)
        return HttpDns.getService(BuildConfig.HTTPDNS_ACCOUNT_ID).also {
            service = it
            it.setPreResolveHosts(DomainPoolPolicy.httpDnsHosts.toList(), RequestIpType.both)
        }
    }

    private fun resolve(host: String): List<java.net.InetAddress> {
        val sdk = serviceAfterConsent() ?: run {
            NetworkDiagnostics.record("httpdns_not_initialized", host, 0)
            return emptyList()
        }
        // Wait for the signed answer on an IO thread, including on first install.
        val result = sdk.getHttpDnsResultForHostSync(host, RequestIpType.both) ?: return emptyList()
        if (result.isFromLocalDns || result.isExpired) return emptyList()
        return (result.ips.orEmpty().toList() + result.ipv6s.orEmpty().toList()).mapNotNull(ScopedHttpDns::literal)
    }
}
