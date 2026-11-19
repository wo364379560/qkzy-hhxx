package com.balabalaim.gcloud.tim.push.appdev.data.network

import android.util.Log
import okhttp3.Call
import okhttp3.EventListener
import okhttp3.Protocol
import okhttp3.Response
import java.io.IOException
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.Proxy
import java.util.UUID

/** No URL path/query, body, headers, user identity, IP address or exception message is logged. */
internal object NetworkDiagnostics {
    fun record(stage: String, host: String, elapsedMs: Long, code: Int? = null, id: String = "none") {
        if ("https://$host" !in DomainPoolPolicy.trustedBusinessOrigins) return
        Log.i("RuiNetwork", "id=$id host=$host stage=$stage elapsed_ms=$elapsedMs code=${code ?: 0}")
    }

    fun listener(): EventListener = object : EventListener() {
        private val id = UUID.randomUUID().toString().take(8)
        private var started = 0L
        private var dnsAt = 0L
        private var connectAt = 0L
        private fun now() = System.nanoTime() / 1_000_000
        override fun callStart(call: Call) { started = now() }
        override fun dnsStart(call: Call, domainName: String) { dnsAt = now() }
        override fun dnsEnd(call: Call, domainName: String, inetAddressList: List<InetAddress>) {
            record("dns", domainName, now() - dnsAt, id = id)
        }
        override fun connectStart(call: Call, inetSocketAddress: InetSocketAddress, proxy: Proxy) { connectAt = now() }
        override fun connectEnd(call: Call, inetSocketAddress: InetSocketAddress, proxy: Proxy, protocol: Protocol?) {
            record("connect", call.request().url.host, now() - connectAt, id = id)
        }
        override fun responseHeadersEnd(call: Call, response: Response) {
            val serverId = response.header("X-Request-ID")?.takeIf { it.matches(Regex("[A-Za-z0-9-]{1,64}")) }
            record("response", call.request().url.host, now() - started, response.code, if (serverId == null) id else "$id/$serverId")
        }
        override fun callFailed(call: Call, ioe: IOException) {
            record("network_failed", call.request().url.host, now() - started, id = id)
        }
    }
}
