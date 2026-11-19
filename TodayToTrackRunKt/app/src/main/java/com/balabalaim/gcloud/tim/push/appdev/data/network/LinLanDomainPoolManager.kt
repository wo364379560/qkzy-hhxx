package com.balabalaim.gcloud.tim.push.appdev.data.network

import android.content.Context
import android.os.SystemClock
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import okhttp3.Call
import okhttp3.Callback
import okhttp3.Response
import java.io.IOException
import kotlinx.coroutines.Deferred
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.async
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

object LinLanDomainPoolManager {
    private lateinit var appContext: Context
    private val mutex = Mutex()
    @Volatile private var refreshTask: Deferred<Unit>? = null
    private val preparingFuture = AtomicBoolean(false)
    private val bgScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client = LinLanHttpTransport.configure(OkHttpClient.Builder()).dns(LinLanHTTPDNS.dns)
        .eventListenerFactory { NetworkDiagnostics.listener() }
        .followRedirects(false).followSslRedirects(false)
        .connectTimeout(2, TimeUnit.SECONDS).readTimeout(3, TimeUnit.SECONDS)
        .callTimeout(3, TimeUnit.SECONDS).build()

    private var cache = ValidatedDomainPoolCache(DomainPoolPolicy.seeds)
    private val origins: List<String> get() = cache.origins()
    @Volatile private var activeOrigin = DomainPoolPolicy.seeds.first()
    @Volatile private var refreshAfterMillis = 0L
    private var bootstrapOrigins = DomainPoolPolicy.bootstrapOrigins
    private var bootstrapCursor = 0

    private val preflight = LoginRoutePreflight(
        now = { SystemClock.elapsedRealtime() }, current = { activeOrigin }, candidates = { origins },
        refreshPool = { deadline ->
            if (System.currentTimeMillis() >= refreshAfterMillis) {
                withTimeoutOrNull((deadline - SystemClock.elapsedRealtime()).coerceAtLeast(1)) {
                    beginRefresh(deadline).await()
                }
            }
        },
        probe = { origin, deadline -> get(origin, LinLanAppConstants.PROBE_PATH, deadline)?.let(DomainPoolPolicy::healthyPing) == true },
        select = { origin ->
            activeOrigin = origin
            prefs().edit().putString(KEY_LAST_ORIGIN, origin).apply()
        },
        awaitPendingPool = { deadline ->
            refreshTask?.takeIf { it.isActive }?.let { pending ->
                withTimeoutOrNull((deadline - SystemClock.elapsedRealtime()).coerceAtLeast(1)) { pending.await() }
            }
        },
    )

    suspend fun prepareLoginRoute(deadline: Long): String {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted) throw ApiException("请先同意隐私政策")
        val verified = preflight.ensure(minOf(deadline, SystemClock.elapsedRealtime() + 12_000))
            ?: throw ApiException("暂时无法连接服务，请稍后重试", stage = "route_preflight")
        if (verified !in origins) throw ApiException("线路已更新，请重试", stage = "route_preflight")
        return verified + LinLanAppConstants.API_PREFIX
    }

    fun recordSuccess(origin: String) = preflight.succeeded(origin)
    fun recordFailure(origin: String) = preflight.failed(origin)

    fun init(context: Context) {
        appContext = context.applicationContext
        LinLanHTTPDNS.configure(appContext)
        loadSeedOrigins()
        prefs().getString(KEY_DOCUMENT, null)?.let { raw ->
            runCatching { DomainPoolPolicy.parseDocument(JSONObject(raw)) }.getOrNull()?.let { cached ->
                cache.restore(cached, prefs().getLong(KEY_FETCHED, 0))
                refreshAfterMillis = cache.refreshAfter()
            }
        }
        // A retired origin in an old preference must never rejoin a newer pool.
        activeOrigin = DomainPoolPolicy.preferMember(origins, prefs().getString(KEY_LAST_ORIGIN, null)).first()
    }

    fun currentBaseUrl(): String {
        val legal = origins
        if (activeOrigin !in legal) activeOrigin = legal.first()
        return activeOrigin + LinLanAppConstants.API_PREFIX
    }

    @Synchronized private fun beginRefresh(deadline: Long): Deferred<Unit> {
        refreshTask?.takeIf { it.isActive }?.let { return it }
        val task = bgScope.async(start = CoroutineStart.LAZY) {
            LinLanHTTPDNS.prepareAfterConsent()
            mutex.withLock { refreshRemotePool(deadline) }
        }
        refreshTask = task
        task.start()
        return task
    }

    /** Coalesced refresh runs off the UI thread and never starts the SDK before consent. */
    suspend fun ensureReady() {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted) return
        if (System.currentTimeMillis() < refreshAfterMillis) return
        beginRefresh(SystemClock.elapsedRealtime() + 6_000)
    }

    /** No business payload/authentication is sent by these bounded future-route probes. */
    fun prepareFutureRoute(failedOrigin: String) {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted || failedOrigin != activeOrigin) return
        preflight.failed(failedOrigin)
        if (!preparingFuture.compareAndSet(false, true)) return
        bgScope.launch {
            try { failoverIfNeeded(budgetMillis = 8_000, failedOrigin = failedOrigin) }
            finally { preparingFuture.set(false) }
        }
    }

    suspend fun failoverIfNeeded(cause: Throwable? = null, budgetMillis: Long = 8_000, failedOrigin: String? = null) = withContext(Dispatchers.IO) {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted) return@withContext
        val budget = budgetMillis.coerceIn(0, 8_000)
        if (budget == 0L) return@withContext
        val deadline = SystemClock.elapsedRealtime() + budget
        withTimeoutOrNull(budget) {
            if (failedOrigin != null && failedOrigin != activeOrigin) return@withTimeoutOrNull
            if (failedOrigin != null) preflight.failed(failedOrigin)
            preflight.ensure(deadline)
        }
        Unit
    }

    private fun prefs() = appContext.getSharedPreferences("domain_pool_cache", Context.MODE_PRIVATE)

    private fun loadSeedOrigins() {
        runCatching {
            val pool = JSONObject(appContext.assets.open("appConfig.json").bufferedReader().use { it.readText() }).getJSONObject("domainPool")
            val list = pool.getJSONArray("apiOrigins")
            val parsed = (0 until list.length()).map { DomainPoolPolicy.normalizeOrigin(list.getString(it)) ?: return@runCatching }
            if (parsed.isNotEmpty() && parsed.all { it in DomainPoolPolicy.trustedBusinessOrigins }) cache = ValidatedDomainPoolCache(parsed.distinct())
            val config = pool.optJSONArray("configOrigins") ?: return@runCatching
            // Bootstrap trust comes from the bundled five exact origins, not a remote pool.
            val seeds = (0 until config.length()).mapNotNull { DomainPoolPolicy.normalizeOrigin(config.optString(it)) }
                .filter { it in DomainPoolPolicy.bootstrapOrigins }.distinct()
            if (seeds.isNotEmpty()) bootstrapOrigins = seeds
        }
    }

    private suspend fun get(origin: String, path: String, deadline: Long): String? {
        val remaining = deadline - SystemClock.elapsedRealtime()
        if (remaining <= 0) return null
        return withTimeoutOrNull(minOf(3_000, remaining)) {
            suspendCancellableCoroutine { continuation ->
                val call = client.newCall(Request.Builder().url(origin + path).header("Cache-Control", "no-cache").get().build())
                call.timeout().timeout(minOf(3_000, remaining), TimeUnit.MILLISECONDS)
                continuation.invokeOnCancellation { call.cancel() }
                call.enqueue(object : Callback {
                    override fun onFailure(call: Call, error: IOException) {
                        if (continuation.isActive) continuation.resume(null)
                    }
                    override fun onResponse(call: Call, response: Response) {
                        val body = response.use {
                            if (it.code != 200 || (it.body?.contentLength() ?: 0) > 65_536) return@use null
                            runCatching {
                                val source = it.body?.source() ?: return@runCatching null
                                val buffer = okio.Buffer()
                                while (buffer.size <= 65_536) {
                                    if (source.read(buffer, 65_537 - buffer.size) == -1L) break
                                }
                                if (buffer.size > 65_536) null else buffer.readUtf8()
                            }.getOrNull()
                        }
                        if (continuation.isActive) continuation.resume(body)
                    }
                })
            }
        }
    }

    private suspend fun refreshRemotePool(deadline: Long) {
        val candidates = DomainPoolPolicy.boundedCandidates(bootstrapOrigins, bootstrapOrigins[bootstrapCursor % bootstrapOrigins.size])
        for (origin in candidates) {
            val body = get(origin, LinLanAppConstants.BOOTSTRAP_PATH, deadline) ?: continue
            val next = DomainPoolPolicy.parseBootstrap(body) ?: continue
            if (!cache.publish(next)) continue
            if (activeOrigin !in origins) activeOrigin = origins.first()
            refreshAfterMillis = System.currentTimeMillis() + next.ttlSeconds * 1000L
            prefs().edit().putString(KEY_DOCUMENT, next.toJSON()).putLong(KEY_FETCHED, System.currentTimeMillis()).apply()
            return
        }
        bootstrapCursor = (bootstrapCursor + 1) % bootstrapOrigins.size
        refreshAfterMillis = System.currentTimeMillis() + 30_000
    }

    private const val KEY_LAST_ORIGIN = "last_good_origin_v4"
    private const val KEY_DOCUMENT = "validated_document_v1"
    private const val KEY_FETCHED = "validated_document_fetched_at_v1"
}
