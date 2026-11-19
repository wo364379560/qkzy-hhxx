package com.balabalaim.gcloud.tim.push.appdev.data.network

import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withTimeoutOrNull

/** Only anonymous probes run here. Business writes are constructed after this gate succeeds. */
internal class LoginRoutePreflight(
    private val now: () -> Long,
    private val current: () -> String,
    private val candidates: () -> List<String>,
    private val refreshPool: suspend (Long) -> Unit,
    private val probe: suspend (String, Long) -> Boolean,
    private val select: (String) -> Unit,
    private val awaitPendingPool: suspend (Long) -> Unit = {},
) {
    private val mutex = Mutex()
    private val healthy = mutableMapOf<String, Long>()
    private val failedUntil = mutableMapOf<String, Long>()
    private var nextPreferred: String? = null

    @Synchronized fun succeeded(origin: String) {
        healthy[origin] = now() + 60_000
        failedUntil.remove(origin)
        nextPreferred = null
    }

    @Synchronized fun failed(origin: String) {
        healthy.remove(origin)
        failedUntil[origin] = now() + 30_000
        val routes = candidates()
        if (routes.isNotEmpty()) nextPreferred = routes[(routes.indexOf(origin).coerceAtLeast(0) + 1) % routes.size]
    }

    @Synchronized private fun isHealthy(origin: String) = (healthy[origin] ?: 0) > now()
    @Synchronized private fun preferred() = nextPreferred ?: current()
    @Synchronized private fun readyCandidates() = candidates().filter { (failedUntil[it] ?: 0) <= now() }

    suspend fun ensure(deadline: Long): String? {
        val snapshot = current()
        if (snapshot in candidates() && isHealthy(snapshot)) return snapshot
        val remaining = deadline - now()
        if (remaining <= 0) return null
        return withTimeoutOrNull(remaining) {
            mutex.withLock {
                val active = current()
                if (active in candidates() && isHealthy(active)) return@withLock active
                refreshPool(minOf(deadline, now() + minOf(4_000L, (deadline - now()).coerceAtLeast(0) / 2)))
                repeat(3) {
                    val poolSnapshot = candidates()
                    val routes = DomainPoolPolicy.recoveryCandidates(readyCandidates(), preferred())
                    for (origin in routes) {
                        if (now() >= deadline) return@withLock null
                        if (poolSnapshot != candidates()) break
                        if (probe(origin, deadline)) {
                            if (origin !in candidates()) continue
                            succeeded(origin)
                            select(origin)
                            return@withLock origin
                        }
                        failed(origin)
                    }
                    // A real in-flight refresh can arrive after fast failures of the old pool.
                    // Await it without polling or inheriting a different caller's budget.
                    if (poolSnapshot == candidates()) awaitPendingPool(deadline)
                    if (poolSnapshot == candidates() || now() >= deadline) return@withLock null
                }
                null
            }
        }
    }
}
