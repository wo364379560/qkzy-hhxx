package com.balabalaim.gcloud.tim.push.appdev.data.network

import java.io.IOException
import java.net.InetAddress
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.runBlocking
import okhttp3.Dns
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.SocketPolicy
import org.junit.Assert.*
import org.junit.Test

class HttpTransportTest {
    private fun client() = LinLanHttpTransport.configure(OkHttpClient.Builder().proxy(java.net.Proxy.NO_PROXY))
        .callTimeout(2, TimeUnit.SECONDS).readTimeout(1, TimeUnit.SECONDS).build()
    private fun write(server: MockWebServer, method: String) = Request.Builder().url(server.url("/write"))
        .method(method, if (method == "DELETE") null else "{\"request_id\":\"stable-test-id\"}".toRequestBody("application/json".toMediaType())).build()

    @Test fun unreachableIPv6FallsBackBeforeSendingOneWrite() {
        MockWebServer().use { server ->
            server.start(InetAddress.getByName("127.0.0.1"), 0)
            server.enqueue(MockResponse().setBody("ok"))
            val dns = object : Dns { override fun lookup(hostname: String) = listOf(InetAddress.getByName("::1"), InetAddress.getByName("127.0.0.1")) }
            var ipv6Attempts = 0
            // Force a pre-send IPv6 connection refusal; local VPNs may accept ::1 sockets.
            val sockets = object : javax.net.SocketFactory() {
                override fun createSocket(): java.net.Socket = object : java.net.Socket() {
                    override fun connect(endpoint: java.net.SocketAddress, timeout: Int) {
                        if ((endpoint as java.net.InetSocketAddress).address.address.size == 16) {
                            ipv6Attempts++; throw java.net.ConnectException("simulated IPv6 refusal")
                        }
                        super.connect(endpoint, timeout)
                    }
                }
                override fun createSocket(host: String, port: Int) = throw UnsupportedOperationException()
                override fun createSocket(host: String, port: Int, local: InetAddress, localPort: Int) = throw UnsupportedOperationException()
                override fun createSocket(host: InetAddress, port: Int) = throw UnsupportedOperationException()
                override fun createSocket(host: InetAddress, port: Int, local: InetAddress, localPort: Int) = throw UnsupportedOperationException()
            }
            val transport = LinLanHttpTransport.configure(OkHttpClient.Builder().proxy(java.net.Proxy.NO_PROXY).dns(dns).socketFactory(sockets))
                .callTimeout(3, TimeUnit.SECONDS).connectTimeout(1, TimeUnit.SECONDS).build()
            val request = write(server, "POST").newBuilder().url(server.url("/write").newBuilder().host("isolated.test").build()).build()
            transport.newCall(request).execute().use { assertEquals(200, it.code) }
            assertEquals(1, server.requestCount)
            assertEquals(1, ipv6Attempts)
            assertEquals("isolated.test:${server.port}", server.takeRequest().getHeader("Host"))
        }
    }

    @Test fun disablingConnectionRetryAloneStillReplays503WithRetryAfterZero() {
        MockWebServer().use { server ->
            server.enqueue(MockResponse().setResponseCode(503).setHeader("Retry-After", "0").setBody("unavailable"))
            server.enqueue(MockResponse().setBody("ok"))
            OkHttpClient.Builder().proxy(java.net.Proxy.NO_PROXY).retryOnConnectionFailure(false).build().newCall(write(server, "POST")).execute().use {
                assertEquals(200, it.code)
            }
            assertEquals("Baseline must expose the status follow-up missed by a flag-only fix", 2, server.requestCount)
        }
    }

    @Test fun everyWriteMethodIsSentOnceOn503IncludingBodylessDelete() {
        for (method in listOf("POST", "PUT", "PATCH", "DELETE", "OPTIONS")) {
            MockWebServer().use { server ->
                server.enqueue(MockResponse().setResponseCode(503).setHeader("Retry-After", "0").setBody("gateway"))
                server.enqueue(MockResponse().setBody("must not be reached"))
                client().newCall(write(server, method)).execute().use {
                    assertEquals(method, 503, it.code)
                    assertEquals("0", it.header("Retry-After"))
                    assertEquals("gateway", it.body!!.string())
                }
                assertEquals(method, 1, server.requestCount)
                val recorded = server.takeRequest(1, TimeUnit.SECONDS)!!
                assertEquals(method, recorded.method)
                assertEquals(if (method == "DELETE") "" else "{\"request_id\":\"stable-test-id\"}", recorded.body.readUtf8())
            }
        }
    }

    @Test fun allMethodsKeep408WithoutTransparentFollowUp() {
        for (method in listOf("GET", "POST", "PUT", "DELETE")) {
            MockWebServer().use { server ->
                server.enqueue(MockResponse().setResponseCode(408).setBody("timeout"))
                server.enqueue(MockResponse().setBody("must not be reached"))
                client().newCall(if (method == "GET") Request.Builder().url(server.url("/read")).build() else write(server, method)).execute().use { assertEquals(408, it.code) }
                assertEquals(method, 1, server.requestCount)
            }
        }
    }

    @Test fun read503AlsoReturnsImmediatelyAndPreservesRetryAfter() {
        for (value in listOf("0", "00", "000", "120")) {
            MockWebServer().use { server ->
                server.enqueue(MockResponse().setResponseCode(503).setHeader("Retry-After", value).setBody("upstream failure"))
                server.enqueue(MockResponse().setBody("must not be reached"))
                client().newCall(Request.Builder().url(server.url("/read")).build()).execute().use {
                    assertEquals(503, it.code); assertEquals(value, it.header("Retry-After")); assertEquals("upstream failure", it.body!!.string())
                }
                assertEquals(1, server.requestCount)
            }
        }
    }

    @Test fun redirectNeverMovesAWriteToAnotherOrigin() {
        for (status in listOf(301, 302, 303, 307, 308)) {
            MockWebServer().use { primary -> MockWebServer().use { backup ->
                backup.start()
                primary.enqueue(MockResponse().setResponseCode(status).setHeader("Location", backup.url("/other")))
                client().newCall(write(primary, "POST")).execute().use { assertEquals(status, it.code) }
                assertEquals(1, primary.requestCount); assertEquals(0, backup.requestCount)
            } }
        }
    }

    @Test fun lostResponseAfterReceivingTheEntireWriteCannotReplayOnAnotherAddress() {
        MockWebServer().use { server ->
            server.start(InetAddress.getByName("127.0.0.1"), 0)
            server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AFTER_REQUEST))
            server.enqueue(MockResponse().setBody("must not be reached"))
            val address = InetAddress.getByName("127.0.0.1")
            val dns = object : Dns { override fun lookup(hostname: String): List<InetAddress> = listOf(address, address) }
            val client = LinLanHttpTransport.configure(OkHttpClient.Builder().proxy(java.net.Proxy.NO_PROXY).dns(dns)).callTimeout(2, TimeUnit.SECONDS).build()
            val request = write(server, "POST").newBuilder().url(server.url("/write").newBuilder().host("isolated.test").build()).build()
            try { client.newCall(request).execute().close(); fail("Expected a lost response") } catch (_: IOException) { }
            assertEquals(1, server.requestCount)
            assertEquals("{\"request_id\":\"stable-test-id\"}", server.takeRequest(1, TimeUnit.SECONDS)!!.body.readUtf8())
        }
    }

    @Test fun aFreshExplicitCallCanReuseTheBodyAfterDefinitive401Rejection() {
        MockWebServer().use { server ->
            server.enqueue(MockResponse().setResponseCode(401)); server.enqueue(MockResponse().setBody("ok"))
            val transport = client(); val original = write(server, "POST").newBuilder().header("Authorization", "Bearer old").build()
            transport.newCall(original).execute().use { assertEquals(401, it.code) }
            assertEquals(1, server.requestCount)
            transport.newCall(original.newBuilder().header("Authorization", "Bearer refreshed").build()).execute().use { assertEquals(200, it.code) }
            assertEquals(2, server.requestCount)
            val first = server.takeRequest(); val second = server.takeRequest()
            assertEquals(first.body.readUtf8(), second.body.readUtf8())
            assertEquals("Bearer old", first.getHeader("Authorization")); assertEquals("Bearer refreshed", second.getHeader("Authorization"))
        }
    }

    @Test fun gatewayGETSwitchesOnceWhilePreservingPathQueryAndToken() = runBlocking {
        for (status in listOf(502, 503, 504)) {
            MockWebServer().use { primary -> MockWebServer().use { backup ->
                backup.start(); primary.enqueue(MockResponse().setResponseCode(status).setHeader("Retry-After", "0"))
                backup.enqueue(MockResponse().setBody("{\"code\":0,\"data\":{\"status\":\"ok\"}}"))
                backup.enqueue(MockResponse().setBody("ok"))
                val transport=client(); var base=primary.url("/api/v1/chat").toString(); var scans=0
                val recovery=ApiRouteRecovery({base}, { _, budget ->
                    scans++; assertTrue(budget in 1..8000)
                    transport.newCall(Request.Builder().url(backup.url("/api/v1/chat/ping")).build()).execute().use {
                        assertTrue(DomainPoolPolicy.healthyPing(it.body!!.string()))
                    }
                    base=backup.url("/api/v1/chat").toString()
                }, { fail("Initial GET should await its bounded scan") }, { System.nanoTime()/1_000_000 })
                val request=Request.Builder().url(primary.url("/api/v1/chat/profile?q=exact%20value")).header("Authorization","Bearer same-session").build()
                val error=transport.newCall(request).execute().use { ApiException("gateway",status=it.code) }
                val next=recovery.nextReadRequest(request,0,error,System.nanoTime()/1_000_000+3000){true}!!
                transport.newCall(next).execute().use { assertEquals(200,it.code) }
                assertEquals(1,scans);assertEquals(1,primary.requestCount);assertEquals(2,backup.requestCount)
                assertNull(backup.takeRequest().getHeader("Authorization"))
                val retried=backup.takeRequest();assertEquals("/api/v1/chat/profile?q=exact%20value",retried.path);assertEquals("Bearer same-session",retried.getHeader("Authorization"))
            } }
        }
    }

    @Test fun writeGatewayFailureOnlyPreparesTheNextExplicitAction() = runBlocking {
        MockWebServer().use { primary ->
            primary.enqueue(MockResponse().setResponseCode(503).setHeader("Retry-After","0"))
            val request=write(primary,"POST");val transport=client()
            val error=transport.newCall(request).execute().use { ApiException("gateway",status=it.code) }
            var future=0
            val recovery=ApiRouteRecovery({primary.url("/").toString()},{_,_->fail("No awaited replay for write")},{future++},{100L})
            assertNull(recovery.nextReadRequest(request,0,error,100L){true})
            assertEquals(1,future);assertEquals(1,primary.requestCount)
        }
    }

    @Test fun businessAuthRateLimitAndCancellationNeverStartAProbe() = runBlocking {
        val request=Request.Builder().url("https://api.ruizhihe.com/api/v1/chat/profile").build()
        val recovery=ApiRouteRecovery({"https://api.prfgsu.com"},{_,_->fail("Unexpected probe")},{fail("Unexpected background probe")},{100L})
        for (error in listOf(ApiException("business"),ApiException("unauthorized",true,status=401),ApiException("forbidden",status=403),ApiException("rate limited",status=429),ApiException("application error",status=500),IllegalArgumentException("bad local data"),CancellationException("canceled"))) {
            assertNull(recovery.nextReadRequest(request,0,error,1000){true})
        }
    }

    @Test fun expiredDeadlineNeverReplaysButCanPrepareTheFuture() = runBlocking {
        var future=0
        val recovery=ApiRouteRecovery({"https://api.prfgsu.com"},{_,_->fail("Expired request must not wait")},{future++},{100L})
        val request=Request.Builder().url("https://api.ruizhihe.com/api/v1/chat/profile").build()
        assertNull(recovery.nextReadRequest(request,0,IOException("lost"),100){true});assertEquals(1,future)
    }

    @Test fun accountChangeDuringProbeCannotGenerateAReplayedRequest() = runBlocking {
        var current=true
        val recovery=ApiRouteRecovery({"https://api.prfgsu.com"},{_,_->current=false},{},{100L})
        val request=Request.Builder().url("https://api.ruizhihe.com/api/v1/chat/profile").header("Authorization","Bearer old").build()
        try { recovery.nextReadRequest(request,0,ApiException("gateway",status=502),1000){current};fail("Old account must stop") }
        catch (_: CancellationException) { }
    }

    @Test fun noHealthyAlternativeNeverResendsTheSameFailedRead() = runBlocking {
        val request=Request.Builder().url("https://api.ruizhihe.com/api/v1/chat/profile").build()
        var scans=0;var future=0
        val recovery=ApiRouteRecovery({"https://api.ruizhihe.com/api/v1/chat"},{_,_->scans++},{future++},{100L})
        assertNull(recovery.nextReadRequest(request,0,ApiException("gateway",status=504),1000){true})
        assertEquals(1,scans);assertEquals(0,future)
        assertNull(recovery.nextReadRequest(request,1,ApiException("gateway",status=503),1000){true});assertEquals(1,future)
    }
}
