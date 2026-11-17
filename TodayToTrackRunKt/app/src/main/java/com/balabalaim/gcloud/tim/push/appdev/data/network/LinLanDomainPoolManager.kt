package com.balabalaim.gcloud.tim.push.appdev.data.network

import android.content.Context
import android.os.SystemClock
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.balabalaim.gcloud.tim.push.appdev.data.local.LinLanAppPrefs
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
    private val refreshing = AtomicBoolean(false)
    private val preparingFuture = AtomicBoolean(false)
    private val bgScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val client = LinLanHttpTransport.configure(OkHttpClient.Builder()).dns(LinLanHTTPDNS.dns)
        .followRedirects(false).followSslRedirects(false)
        .connectTimeout(2, TimeUnit.SECONDS).readTimeout(3, TimeUnit.SECONDS)
        .callTimeout(3, TimeUnit.SECONDS).build()

    @Volatile private var origins = DomainPoolPolicy.seeds
    @Volatile private var activeOrigin = DomainPoolPolicy.seeds.first()
    @Volatile private var refreshAfterMillis = 0L
    private var document: DomainPoolDocument? = null
    private var bootstrapOrigins = DomainPoolPolicy.bootstrapOrigins
    private var bootstrapCursor = 0

    fun init(context: Context) {
        appContext = context.applicationContext
        LinLanHTTPDNS.configure(appContext)
        loadSeedOrigins()
        prefs().getString(KEY_DOCUMENT, null)?.let { raw ->
            runCatching { DomainPoolPolicy.parseDocument(JSONObject(raw)) }.getOrNull()?.let { cached ->
                document = cached
                origins = cached.origins
                refreshAfterMillis = prefs().getLong(KEY_FETCHED, 0) + cached.ttlSeconds * 1000L
            }
        }
        // A retired origin in an old preference must never rejoin a newer pool.
        activeOrigin = DomainPoolPolicy.preferMember(origins, prefs().getString(KEY_LAST_ORIGIN, null)).first()
    }

    fun currentBaseUrl(): String = activeOrigin + LinLanAppConstants.API_PREFIX

    /** Coalesced refresh runs off the UI thread and never starts the SDK before consent. */
    suspend fun ensureReady() {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted) return
        if (System.currentTimeMillis() < refreshAfterMillis || !refreshing.compareAndSet(false, true)) return
        bgScope.launch {
            try {
                LinLanHTTPDNS.prepareAfterConsent()
                mutex.withLock {
                    val deadline = SystemClock.elapsedRealtime() + 18_000
                    refreshRemotePool(deadline)
                    probeCandidates(deadline, activeOrigin)
                }
            } finally { refreshing.set(false) }
        }
    }

    /** No business payload/authentication is sent by these bounded future-route probes. */
    fun prepareFutureRoute(failedOrigin: String) {
        if (!::appContext.isInitialized || !LinLanAppPrefs(appContext).privacyAccepted || failedOrigin != activeOrigin) return
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
            mutex.withLock {
                if (failedOrigin != null && failedOrigin != activeOrigin) return@withLock
                val index = origins.indexOf(activeOrigin).coerceAtLeast(0)
                val next = origins[(index + 1) % origins.size]
                probeCandidates(deadline, next)
            }
        }
        Unit
    }

    private fun prefs() = appContext.getSharedPreferences("domain_pool_cache", Context.MODE_PRIVATE)

    private fun loadSeedOrigins() {
        runCatching {
            val pool = JSONObject(appContext.assets.open("appConfig.json").bufferedReader().use { it.readText() }).getJSONObject("domainPool")
            val list = pool.getJSONArray("apiOrigins")
            val parsed = (0 until list.length()).map { DomainPoolPolicy.normalizeOrigin(list.getString(it)) ?: return@runCatching }
            if (parsed.isNotEmpty()) origins = parsed.distinct()
            val config = pool.optJSONArray("configOrigins") ?: return@runCatching
            // Bootstrap trust comes from the bundled five exact origins, not a remote pool.
            val seeds = (0 until config.length()).mapNotNull { DomainPoolPolicy.normalizeOrigin(config.optString(it)) }
                .filter { it in DomainPoolPolicy.bootstrapOrigins }.distinct()
            if (seeds.isNotEmpty()) bootstrapOrigins = seeds
        }
    }

    private fun get(origin: String, path: String, deadline: Long): String? {
        val remaining = deadline - SystemClock.elapsedRealtime()
        if (remaining <= 0) return null
        return runCatching {
            val call = client.newCall(Request.Builder().url(origin + path).header("Cache-Control", "no-cache").get().build())
            call.timeout().timeout(minOf(3_000, remaining), TimeUnit.MILLISECONDS)
            call.execute().use { response ->
                if (response.code != 200 || (response.body?.contentLength() ?: 0) > 65_536) return@use null
                val source = response.body?.source() ?: return@use null
                val buffer = okio.Buffer()
                while (buffer.size <= 65_536) {
                    if (source.read(buffer, 65_537 - buffer.size) == -1L) break
                }
                if (buffer.size > 65_536) null else buffer.readUtf8()
            }
        }.getOrNull()
    }

    private fun refreshRemotePool(deadline: Long) {
        val candidates = DomainPoolPolicy.boundedCandidates(bootstrapOrigins, bootstrapOrigins[bootstrapCursor % bootstrapOrigins.size])
        for (origin in candidates) {
            val body = get(origin, LinLanAppConstants.BOOTSTRAP_PATH, deadline) ?: continue
            val next = DomainPoolPolicy.parseBootstrap(body) ?: continue
            if (!DomainPoolPolicy.accepts(document, next)) continue
            document = next
            origins = next.origins
            if (activeOrigin !in origins) activeOrigin = origins.first()
            refreshAfterMillis = System.currentTimeMillis() + next.ttlSeconds * 1000L
            prefs().edit().putString(KEY_DOCUMENT, next.toJSON()).putLong(KEY_FETCHED, System.currentTimeMillis()).apply()
            return
        }
        bootstrapCursor = (bootstrapCursor + 1) % bootstrapOrigins.size
        refreshAfterMillis = System.currentTimeMillis() + 30_000
    }

    private fun probeCandidates(deadline: Long, preferred: String) {
        for (origin in DomainPoolPolicy.boundedCandidates(origins, preferred)) {
            if (SystemClock.elapsedRealtime() >= deadline) return
            val body = get(origin, LinLanAppConstants.PROBE_PATH, deadline) ?: continue
            if (!DomainPoolPolicy.healthyPing(body)) continue
            activeOrigin = origin
            prefs().edit().putString(KEY_LAST_ORIGIN, origin).apply()
            return
        }
        // Keep the previous route; an unverified candidate is never cached as healthy.
    }

    private const val KEY_LAST_ORIGIN = "last_good_origin_v4"
    private const val KEY_DOCUMENT = "validated_document_v1"
    private const val KEY_FETCHED = "validated_document_fetched_at_v1"
}
