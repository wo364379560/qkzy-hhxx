package com.balabalaim.gcloud.tim.push.appdev.data.network

import org.json.JSONObject
import java.net.URI

/** Domain configuration and HTTPDNS answers deliberately have separate lifetimes. */
internal data class DomainPoolDocument(val version: Long, val ttlSeconds: Int, val origins: List<String>) {
    fun toJSON(): String = JSONObject().put("version", version).put("ttl_seconds", ttlSeconds)
        .put("api_origins", org.json.JSONArray(origins)).toString()
}

internal object DomainPoolPolicy {
    // Formal version-0 pool; CDN entries become business routes only after publication.
    val seeds = listOf("https://api.ruizhihe.com", "https://api.prfgsu.com", "https://api.uktjwg.cn")
    val bootstrapOrigins = listOf(
        "https://api.ruizhihe.com", "https://api.prfgsu.com", "https://api.uktjwg.cn",
        "https://api1.prfgsu.com", "https://api1.uktjwg.cn",
    )
    // Approved business hosts are independent of the bundled bootstrap and HTTPDNS scopes.
    val trustedBusinessOrigins = bootstrapOrigins.toSet() + setOf(
        "https://cdnapi1.sazuts.cn", "https://cdnapi2.sazuts.cn",
    )
    val httpDnsHosts = setOf("api.prfgsu.com", "api1.prfgsu.com", "api.uktjwg.cn", "api1.uktjwg.cn")

    internal fun relay(origin: String): String = when (URI(origin).host) {
        "api.prfgsu.com", "api1.prfgsu.com" -> "001"
        "api.uktjwg.cn", "api1.uktjwg.cn" -> "002"
        "cdnapi1.sazuts.cn", "cdnapi2.sazuts.cn" -> "sazuts"
        else -> origin
    }

    /** Bounded by the exact approved pool; prefer independent relay groups. */
    fun recoveryCandidates(origins: List<String>, preferred: String): List<String> {
        if (origins.isEmpty()) return emptyList()
        val start = origins.indexOf(preferred).coerceAtLeast(0)
        val rotated = origins.drop(start) + origins.take(start)
        val families = mutableSetOf<String>()
        val independent = rotated.filter { families.add(relay(it)) }
        return independent + rotated.filter { it !in independent }
    }

    fun boundedCandidates(origins: List<String>, preferred: String): List<String> = recoveryCandidates(origins, preferred).take(2)

    // Leave time for recovery and one safe GET retry when the first route hangs.
    fun attemptBudgetMillis(method: String, attempt: Int, remaining: Long): Long =
        if (method.equals("GET", ignoreCase = true) && attempt == 0) remaining.coerceAtLeast(0) / 2
        else remaining.coerceAtLeast(0)

    fun normalizeOrigin(raw: String): String? = runCatching {
        val u = URI(raw.trim())
        val host = u.host?.lowercase() ?: return null
        if (u.scheme?.lowercase() != "https" || u.rawUserInfo != null || u.rawQuery != null || u.rawFragment != null ||
            (u.rawPath != "" && u.rawPath != "/") || (u.port != -1 && u.port != 443) ||
            host.length > 253 || !host.contains('.') || host.endsWith(".local") ||
            host.all { it.isDigit() || it == '.' } ||
            host.split('.').any { !it.matches(Regex("[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?")) }) return null
        "https://$host"
    }.getOrNull()

    fun parseDocument(data: JSONObject): DomainPoolDocument? = runCatching {
        val pool = data.optJSONObject("domain_pool") ?: data
        val versionNumber = pool.opt("version") as? Number ?: return null
        val ttlNumber = pool.opt("ttl_seconds") as? Number ?: return null
        val version = versionNumber.toLong()
        val ttl = ttlNumber.toInt()
        if (version < 0 || versionNumber.toDouble() != version.toDouble() || ttl !in 60..86400 || ttlNumber.toDouble() != ttl.toDouble()) return null
        val array = pool.optJSONArray("api_origins") ?: return null
        if (array.length() !in 1..16) return null
        val origins = (0 until array.length()).map { normalizeOrigin(array.get(it) as? String ?: return null) ?: return null }.distinct()
        if (origins.any { it !in trustedBusinessOrigins }) return null
        DomainPoolDocument(version, ttl, origins)
    }.getOrNull()

    fun parseBootstrap(body: String): DomainPoolDocument? = runCatching {
        val root = JSONObject(body)
        if (root.opt("code") != 0) return null
        parseDocument(root.getJSONObject("data"))
    }.getOrNull()

    // A stale document may bridge a temporary outage for at most one day.
    fun usableCache(fetchedAt: Long, now: Long, ttlSeconds: Int = 0): Boolean = fetchedAt > 0 && now >= fetchedAt && now - fetchedAt <= ttlSeconds * 1000L + 86_400_000L

    fun accepts(current: DomainPoolDocument?, next: DomainPoolDocument): Boolean = current == null ||
        next.version > current.version || next == current

    fun healthyPing(body: String): Boolean = runCatching {
        val root = JSONObject(body)
        root.opt("code") == 0 && root.getJSONObject("data").optString("status") == "ok"
    }.getOrDefault(false)

    fun preferMember(origins: List<String>, cached: String?): List<String> {
        val normalized = cached?.let(::normalizeOrigin)
        return if (normalized != null && normalized in origins) listOf(normalized) + origins.filter { it != normalized } else origins
    }
}

/** Version trust outlives the route cache. Reads re-evaluate expiry while the app stays running. */
internal class ValidatedDomainPoolCache(
    private val seeds: List<String>,
    private val wallNow: () -> Long = System::currentTimeMillis,
) {
    private var document: DomainPoolDocument? = null
    private var fetchedAt = 0L
    @Synchronized fun restore(value: DomainPoolDocument, fetched: Long) { document = value; fetchedAt = fetched }
    @Synchronized fun publish(value: DomainPoolDocument): Boolean {
        if (!DomainPoolPolicy.accepts(document, value)) return false
        document = value; fetchedAt = wallNow()
        return true
    }
    @Synchronized fun origins(): List<String> {
        val value = document ?: return seeds
        return if (DomainPoolPolicy.usableCache(fetchedAt, wallNow(), value.ttlSeconds)) value.origins else seeds
    }
    @Synchronized fun refreshAfter(): Long {
        val value = document ?: return 0L
        return if (DomainPoolPolicy.usableCache(fetchedAt, wallNow(), value.ttlSeconds)) fetchedAt + value.ttlSeconds * 1000L else 0L
    }
}
