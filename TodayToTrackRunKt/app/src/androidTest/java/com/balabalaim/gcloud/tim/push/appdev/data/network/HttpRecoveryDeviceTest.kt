package com.balabalaim.gcloud.tim.push.appdev.data.network

import android.content.Context
import android.content.ContextWrapper
import android.os.SystemClock
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import java.lang.reflect.InvocationTargetException
import java.net.InetAddress
import java.net.Proxy
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.runBlocking
import kotlin.coroutines.Continuation
import kotlin.coroutines.intrinsics.suspendCoroutineUninterceptedOrReturn
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import okhttp3.mockwebserver.SocketPolicy
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith

/** All injected failures stay inside this app process and 127.0.0.1. No account/IM operations. */
@RunWith(AndroidJUnit4::class)
class HttpRecoveryDeviceTest {
    private fun optIn() = assumeTrue(InstrumentationRegistry.getArguments().getString("ruiFaultTests") == "true")
    private fun report(text: String) = InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply { putString("rui_wire", text) })
    private fun client() = LinLanApiClient(LinLanAppContainer.sessionStore())
    private fun field(name: String) = LinLanDomainPoolManager::class.java.getDeclaredField(name).apply { isAccessible = true }
    private fun transport(api: LinLanApiClient, name: String): OkHttpClient = LinLanApiClient::class.java.getDeclaredField(name).apply { isAccessible = true }.get(api) as OkHttpClient
    private fun server() = MockWebServer().apply { start(InetAddress.getByName("127.0.0.1"), 0) }
    private fun write(server: MockWebServer, method: String = "POST") = Request.Builder().url(server.url("/api/v1/chat/isolated-write"))
        .method(method, if (method == "DELETE") null else "{\"request_id\":\"isolated-stable-id\"}".toRequestBody()).build()
    private suspend fun execute(api: LinLanApiClient, request: Request): JSONObject = suspendCoroutineUninterceptedOrReturn { continuation ->
        val method = LinLanApiClient::class.java.getDeclaredMethod("executeForData", Request::class.java, String::class.java, java.lang.Long::class.java, Continuation::class.java).apply { isAccessible = true }
        try { method.invoke(api, request, null, null, continuation) }
        catch (error: InvocationTargetException) { throw error.targetException }
    }
    private fun waitForFutureProbe() {
        val deadline = SystemClock.elapsedRealtime() + 9000
        val pending = field("preparingFuture").get(LinLanDomainPoolManager) as java.util.concurrent.atomic.AtomicBoolean
        while (pending.get() && SystemClock.elapsedRealtime() < deadline) Thread.sleep(10)
        assertFalse("Future probe must finish under its deadline", pending.get())
    }
    private fun <T> routes(origins: List<String>, block: () -> T): T {
        val names = listOf("origins", "activeOrigin", "refreshAfterMillis", "appContext")
        val saved = names.associateWith { field(it).get(LinLanDomainPoolManager) }
        val target = InstrumentationRegistry.getInstrumentation().targetContext
        val isolated = target
        val context = object : ContextWrapper(target) {
            override fun getApplicationContext(): Context = this
            override fun getSharedPreferences(name: String, mode: Int) = isolated.getSharedPreferences("rui_r3_fault_$name", mode)
        }
        // Only the fixture's own preference namespace is changed.
        LinLanAppPrefs(context).privacyAccepted = true
        field("appContext").set(LinLanDomainPoolManager, context)
        field("origins").set(LinLanDomainPoolManager, origins)
        field("activeOrigin").set(LinLanDomainPoolManager, origins[0])
        field("refreshAfterMillis").set(LinLanDomainPoolManager, Long.MAX_VALUE)
        try { return block() }
        finally {
            waitForFutureProbe()
            names.forEach { field(it).set(LinLanDomainPoolManager, saved[it]) }
        }
    }
    private fun ping() = MockResponse().setHeader("Cache-Control", "no-store").setBody("{\"code\":0,\"data\":{\"status\":\"ok\"}}")
    private fun success() = MockResponse().setBody("{\"code\":0,\"data\":{\"fixture\":\"passed\"}}")

    @Test(timeout = 30_000) fun actualBusinessAndUploadClientsDoNotReplayStatusFollowUps() {
        optIn()
        val api = client()
        for (name in listOf("client", "uploadClient")) {
            val real = transport(api, name)
            assertFalse(real.retryOnConnectionFailure); assertFalse(real.followRedirects); assertFalse(real.followSslRedirects)
            for (method in listOf("POST", "PUT", "DELETE")) for (status in listOf(408, 503)) server().use { server ->
                server.enqueue(MockResponse().setResponseCode(status).setHeader("Retry-After", "0")); server.enqueue(success())
                real.newCall(write(server, method)).execute().use { assertEquals(status, it.code) }
                assertEquals(1, server.requestCount)
                report("client=$name method=$method status=$status writes_received=1")
            }
        }
    }

    @Test(timeout = 30_000) fun lostWriteResponseAndRedirectCannotProduceASecondWrite() {
        optIn(); val api=client()
        for (name in listOf("client", "uploadClient")) {
            val real=transport(api,name)
            server().use { server ->
                server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AFTER_REQUEST));server.enqueue(success())
                try { real.newCall(write(server)).execute().close();fail("Expected the lost response") } catch (_: java.io.IOException) { }
                assertEquals(1,server.requestCount)
                report("client=$name case=response-lost-after-body writes_received=1")
            }
            server().use { primary -> server().use { backup ->
                primary.enqueue(MockResponse().setResponseCode(307).setHeader("Location",backup.url("/redirected")))
                real.newCall(write(primary)).execute().use { assertEquals(307,it.code) }
                assertEquals(1,primary.requestCount);assertEquals(0,backup.requestCount)
            } }
        }
    }

    @Test(timeout = 45_000) fun actualApiGatewayGetSwitchesThroughHealthyPublicProbe() {
        optIn()
        for (status in listOf(502,503,504)) server().use { primary -> server().use { backup ->
            primary.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=MockResponse().setResponseCode(status).setHeader("Retry-After","0").setBody("{\"code\":$status,\"message\":\"isolated gateway\"}") }
            backup.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=if(request.path!!.endsWith("/ping")) ping() else success() }
            routes(listOf(primary.url("/").toString().removeSuffix("/"),backup.url("/").toString().removeSuffix("/"))) {
                val api=client()
                val result=runBlocking { execute(api,Request.Builder().url(primary.url("/api/v1/chat/isolated-get?q=preserved")).header("Authorization","Bearer isolated-test").build()) }
                assertEquals("passed",result.getString("fixture"));assertEquals(1,primary.requestCount);assertEquals(2,backup.requestCount)
                assertNull(backup.takeRequest().getHeader("Authorization"))
                val recovered=backup.takeRequest();assertEquals("/api/v1/chat/isolated-get?q=preserved",recovered.path);assertEquals("Bearer isolated-test",recovered.getHeader("Authorization"))
                report("case=actual-api-get status=$status primary_gets=1 backup_health=1 backup_gets=1")
            }
        } }
    }

    @Test(timeout = 30_000) fun actualApiWrite503OnlyPreparesTheNextManualAction() {
        optIn();server().use { primary -> server().use { backup ->
            primary.enqueue(MockResponse().setResponseCode(503).setHeader("Retry-After","0"))
            backup.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=if(request.path!!.endsWith("/ping")) ping() else success() }
            routes(listOf(primary.url("/").toString().removeSuffix("/"),backup.url("/").toString().removeSuffix("/"))) {
                val api=client()
                runBlocking {
                    try { execute(api,write(primary));fail("Original write must return its error") }
                    catch (error: ApiException) { assertEquals(503,error.status) }
                }
                waitForFutureProbe()
                assertEquals(1,primary.requestCount);assertEquals(1,backup.requestCount)
                assertEquals("GET",backup.takeRequest().method)
                val next=Request.Builder().url(LinLanDomainPoolManager.currentBaseUrl()+"/isolated-write").post("manual-second-action".toRequestBody()).build()
                assertEquals("passed",runBlocking { execute(api,next) }.getString("fixture"))
                assertEquals("POST",backup.takeRequest().method);assertEquals(2,backup.requestCount)
                report("case=actual-api-post503 original_writes=1 automatic_backup_writes=0 next_manual_write=1")
            }
        } }
    }

    @Test(timeout = 30_000) fun authRateLimitAndBusinessErrorsKeepTheCurrentRoute() {
        optIn()
        for (status in listOf(401,403,429,500,200)) server().use { primary -> server().use { backup ->
            primary.enqueue(MockResponse().setResponseCode(status).setBody("{\"code\":502,\"message\":\"isolated business rejection\"}"))
            routes(listOf(primary.url("/").toString().removeSuffix("/"),backup.url("/").toString().removeSuffix("/"))) {
                try { runBlocking { execute(client(),Request.Builder().url(primary.url("/api/v1/chat/isolated-get")).build()) };fail("Expected rejection") }
                catch (error: ApiException) { if(status!=200)assertEquals(status,error.status) else assertNull(error.status) }
                assertEquals(1,primary.requestCount);assertEquals(0,backup.requestCount)
            }
        } }
        report("case=401-403-429-500-and-http200-business-error probes=0")
    }

    @Test(timeout = 30_000) fun primaryAndFirstBackupFailureCanSelectTheRemainingHealthyRoute() {
        optIn();server().use { primary -> server().use { first -> server().use { last ->
            primary.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=MockResponse().setResponseCode(502) }
            first.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=MockResponse().setResponseCode(503) }
            last.dispatcher=object : Dispatcher() { override fun dispatch(request: RecordedRequest)=if(request.path!!.endsWith("/ping"))ping() else success() }
            routes(listOf(primary,first,last).map{it.url("/").toString().removeSuffix("/")}) {
                val data=runBlocking { execute(client(),Request.Builder().url(primary.url("/api/v1/chat/isolated-get")).build()) }
                assertEquals("passed",data.getString("fixture"));assertEquals(1,primary.requestCount);assertEquals(1,first.requestCount);assertEquals(2,last.requestCount)
                report("case=two-routes-failed remaining_healthy_route_selected=true")
            }
        } } }
    }
}
