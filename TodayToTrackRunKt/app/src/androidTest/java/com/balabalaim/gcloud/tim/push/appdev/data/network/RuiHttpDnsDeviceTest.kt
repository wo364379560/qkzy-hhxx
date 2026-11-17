package com.balabalaim.gcloud.tim.push.appdev.data.network

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import okhttp3.Connection
import okhttp3.EventListener
import okhttp3.OkHttpClient
import okhttp3.Request
import org.junit.Assert.*
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith
import java.net.InetAddress
import java.net.UnknownHostException
import java.util.concurrent.TimeUnit

/** Explicitly opted-in, read-only public endpoint checks on a selected device. */
@RunWith(AndroidJUnit4::class)
class RuiHttpDnsDeviceTest {
    private fun optIn() = assumeTrue(InstrumentationRegistry.getArguments().getString("ruiHttpDnsLive") == "true")

    @Test(timeout = 120_000) fun realSignedAnswersReachEveryManagedHostWithSystemDNSDisabled() {
        optIn()
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val prefs = LinLanAppPrefs(context)
        val accepted = prefs.privacyAccepted
        var activity: android.app.Activity? = null
        val diagnostics = java.util.Collections.synchronizedSet(mutableSetOf<String>())
        // Keep native SDK logcat disabled. Retain only enumerated, non-secret error categories.
        val diagnosticLogger = com.alibaba.sdk.android.httpdns.ILogger { message ->
            for (category in listOf("NonWhitelistDomain", "SignatureError", "InvalidSignature", "SignatureExpired", "RRNotExist", "Unauthorized", "InvalidAccount", "SSLHandshakeException", "SocketTimeoutException", "UnknownHostException", "HTTP 403", "HTTP 200", "403", "200", "Code:Success", "Network is unreachable", "init error", "service is disabled", "init fail, crash defend", "sdk is not safe to run", "sdk will not run any more", "init httpdns with null context", "host is invalid", "httpdns service is init", "request success", "response data is empty", "response data decrypt fail", "解析失败", "request in main thread", "response don\'t have code", "network is not available", "SocketException", "SSLException")) {
                if (message.contains(category)) diagnostics.add(category)
            }
        }
        com.alibaba.sdk.android.httpdns.log.HttpDnsLog.enable(false)
        com.alibaba.sdk.android.httpdns.log.HttpDnsLog.setLogger(diagnosticLogger)
        try {
            // This test process restores the existing consent state on exit.
            prefs.privacyAccepted = true
            val intent = android.content.Intent(context, com.balabalaim.gcloud.tim.push.appdev.MainActivity::class.java)
                .addFlags(android.content.Intent.FLAG_ACTIVITY_NEW_TASK)
            activity = InstrumentationRegistry.getInstrumentation().startActivitySync(intent)
            LinLanHTTPDNS.configure(context)
            LinLanHTTPDNS.prepareAfterConsent()
            val resolver = LinLanHTTPDNS::class.java.getDeclaredMethod("resolve", String::class.java).apply { isAccessible = true }
            for (host in DomainPoolPolicy.httpDnsHosts) {
                InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply { putString("rui_probe", "host=$host phase=signed-resolve") })
                val warmed = java.util.concurrent.CompletableFuture<com.alibaba.sdk.android.httpdns.HTTPDNSResult>()
                com.alibaba.sdk.android.httpdns.HttpDns.getService(com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_ACCOUNT_ID)
                    .getHttpDnsResultForHostAsync(host, com.alibaba.sdk.android.httpdns.RequestIpType.both) { result -> warmed.complete(result) }
                warmed.get(5, TimeUnit.SECONDS)
                @Suppress("UNCHECKED_CAST")
                val initialAnswers = resolver.invoke(LinLanHTTPDNS, host) as List<InetAddress>
                if (initialAnswers.isEmpty()) {
                    val instance = com.alibaba.sdk.android.httpdns.HttpDns.getService(com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_ACCOUNT_ID)
                    val raw = instance.getHttpDnsResultForHostSync(host, com.alibaba.sdk.android.httpdns.RequestIpType.both)
                    fail("SDK result for $host: present=${raw != null} v4=${raw?.ips?.size ?: 0} v6=${raw?.ipv6s?.size ?: 0} expired=${raw?.isExpired} local=${raw?.isFromLocalDns} ttl=${raw?.ttl} impl=${instance.javaClass.name} hostMatches=${raw?.host == host} categories=$diagnostics")
                }
                val actualAnswers = mutableSetOf<InetAddress>()
                var systemCalls = 0
                val blockedSystem = object : okhttp3.Dns {
                    override fun lookup(hostname: String): List<InetAddress> { systemCalls++; throw UnknownHostException("System DNS disabled for $hostname in this test client") }
                }
                val dns = ScopedHttpDns({ queried ->
                    @Suppress("UNCHECKED_CAST")
                    val answers = resolver.invoke(LinLanHTTPDNS, queried) as List<InetAddress>
                    assertTrue("Signed SDK returned no usable answers for $host", answers.isNotEmpty())
                    actualAnswers.addAll(answers)
                    answers
                }, blockedSystem)
                val client = LinLanHttpTransport.configure(OkHttpClient.Builder()).dns(dns).followRedirects(false).followSslRedirects(false)
                    .connectTimeout(2, TimeUnit.SECONDS).readTimeout(3, TimeUnit.SECONDS).callTimeout(10, TimeUnit.SECONDS)
                    .eventListener(object : EventListener() {
                        override fun connectionAcquired(call: okhttp3.Call, connection: Connection) {
                            assertEquals(host, call.request().url.host)
                            assertNotNull("Expected verified TLS", connection.handshake())
                            assertTrue("Connection must use a signed SDK answer", connection.route().socketAddress.address in actualAnswers)
                        }
                    }).build()
                try {
                    for (path in listOf("/api/v1/chat/ping", "/api/v1/public/bootstrap/domain-pool")) {
                        client.newCall(Request.Builder().url("https://$host$path").get().build()).execute().use { response ->
                            assertEquals("$host$path", 200, response.code)
                            assertTrue(response.header("Cache-Control").orEmpty().contains("no-store"))
                            val text = response.body!!.string()
                            if (path.endsWith("/ping")) assertTrue(DomainPoolPolicy.healthyPing(text))
                            else assertNotNull(DomainPoolPolicy.parseBootstrap(text))
                            InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply {
                                putString("rui_probe", "host=$host path=$path http=200 dns=signed-sdk tls=verified system_fallback=false")
                            })
                        }
                    }
                    assertEquals(0, systemCalls)
                } finally {
                    client.connectionPool.evictAll()
                    client.dispatcher.executorService.shutdown()
                }
            }
        } finally {
            com.alibaba.sdk.android.httpdns.log.HttpDnsLog.removeLogger(diagnosticLogger)
            prefs.privacyAccepted = accepted
            activity?.let { running -> InstrumentationRegistry.getInstrumentation().runOnMainSync { running.finish() } }
        }
    }

    @Test(timeout = 20_000) fun cfRemainsOnTheProductionSystemDNSPath() {
        optIn()
        assertFalse(DomainPoolPolicy.httpDnsHosts.contains("api.ruizhihe.com"))
        val client = LinLanHttpTransport.configure(OkHttpClient.Builder()).dns(LinLanHTTPDNS.dns).followRedirects(false).callTimeout(10, TimeUnit.SECONDS).build()
        try {
            client.newCall(Request.Builder().url("https://api.ruizhihe.com/api/v1/chat/ping").build()).execute().use {
                assertEquals(200, it.code)
                assertTrue(DomainPoolPolicy.healthyPing(it.body!!.string()))
                println("RUI_QA host=api.ruizhihe.com http=200 dns=system tls=verified")
            }
        } finally { client.connectionPool.evictAll(); client.dispatcher.executorService.shutdown() }
    }
    @Test(timeout = 20_000) fun signedV2ChannelDiagnostic() {
        optIn()
        val q = sortedMapOf("id" to com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_ACCOUNT_ID,
            "m" to "0", "dn" to "api.prfgsu.com", "q" to "4,6", "v" to "1.0",
            "exp" to (System.currentTimeMillis() / 1000 + 600).toString())
        val key = com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_SECRET_KEY.chunked(2).map { it.toInt(16).toByte() }.toByteArray()
        val mac = javax.crypto.Mac.getInstance("HmacSHA256").apply { init(javax.crypto.spec.SecretKeySpec(key, "HmacSHA256")) }
        q["s"] = mac.doFinal(q.entries.joinToString("&") { "${it.key}=${it.value}" }.toByteArray()).joinToString("") { "%02x".format(it) }
        val url = okhttp3.HttpUrl.Builder().scheme("https").host("203.107.1.1").addPathSegment("v2").addPathSegment("d")
        q.forEach { (k, v) -> url.addQueryParameter(k, v) }
        val client = LinLanHttpTransport.configure(OkHttpClient.Builder()).connectTimeout(2, TimeUnit.SECONDS).readTimeout(3, TimeUnit.SECONDS).callTimeout(5, TimeUnit.SECONDS).build()
        try {
            client.newCall(Request.Builder().url(url.build()).build()).execute().use { response ->
                val json = org.json.JSONObject(response.body!!.string())
                val answers = json.optJSONObject("data")?.optJSONArray("answers")
                val answer = answers?.optJSONObject(0)
                val count = (answer?.optJSONObject("v4")?.optJSONArray("ips")?.length() ?: 0) + (answer?.optJSONObject("v6")?.optJSONArray("ips")?.length() ?: 0)
                InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply {
                    putString("rui_v2_channel", "http=${response.code} code=${json.optString("code")} answer_count=$count tls=verified")
                })
                assertEquals(200, response.code)
                assertTrue("No answers from the signed v2 channel", count > 0)
            }
        } catch (error: Exception) {
            throw AssertionError("Native signed v2 channel failed: ${error.javaClass.simpleName}")
        } finally { client.connectionPool.evictAll(); client.dispatcher.executorService.shutdown() }
    }

    @Test(timeout = 15_000) fun nativeSdkColdLookupDiagnostic() {
        optIn()
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val disk = InstrumentationRegistry.getArguments().getString("ruiDiskCache") == "true"
        val config = com.alibaba.sdk.android.httpdns.InitConfig.Builder().setContext(context)
            .setSecretKey(com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_SECRET_KEY)
            .setEnableHttps(true).setEnableCacheIp(disk).setEnableExpiredIp(false)
            .setEnableDegradationLocalDns(false).setTimeoutMillis(2000).build()
        val account = com.balabalaim.gcloud.tim.push.appdev.BuildConfig.HTTPDNS_ACCOUNT_ID
        com.alibaba.sdk.android.httpdns.HttpDns.init(account, config)
        val sdk = com.alibaba.sdk.android.httpdns.HttpDns.getService(account)
        val result = sdk.getHttpDnsResultForHostSync("api.prfgsu.com", com.alibaba.sdk.android.httpdns.RequestIpType.both)
        val count = (result?.ips?.size ?: 0) + (result?.ipv6s?.size ?: 0)
        InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply {
            putString("rui_sdk_cold", "disk_cache=$disk count=$count hostMatches=${result?.host == "api.prfgsu.com"}")
        })
        assertTrue("Cold SDK lookup yielded no answers", count > 0)
    }

}
