package com.balabalaim.gcloud.tim.push.appdev.data.network

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class LoginRoutePreflightTest {
    private val pool = listOf("https://api.prfgsu.com", "https://api.uktjwg.cn", "https://api1.prfgsu.com", "https://api1.uktjwg.cn", "https://cdnapi1.sazuts.cn", "https://cdnapi2.sazuts.cn")

    @Test fun coldInstallProbesBeforeTheFirstWriteAndHealthyRouteDoesNotWait() = runBlocking {
        var now = 100L; var origin = DomainPoolPolicy.seeds.first(); var candidates = DomainPoolPolicy.seeds
        val events = mutableListOf<String>()
        val gate = LoginRoutePreflight({ now }, { origin }, { candidates }, {
            events += "pool"; candidates = pool
        }, { target, _ -> events += target; target == pool[1] }, { origin = it })
        assertNotNull(gate.ensure(12_100))
        events += "business_write"
        assertEquals(listOf("pool", pool[0], pool[1], "business_write"), events)
        assertEquals(pool[1], origin)
        assertFalse(events.any { it.contains("ruizhihe") })
        events.clear(); now += 1_000
        assertNotNull(gate.ensure(now + 12_000)); assertTrue(events.isEmpty())
    }

    @Test fun failedCandidatesCoolDownAndBothNewCDNsCanBeReached() = runBlocking {
        for (healthy in pool.takeLast(2)) {
            var origin = pool.first()
            val attempted = mutableListOf<String>()
            val gate = LoginRoutePreflight({ 100L }, { origin }, { pool }, {}, { target, _ -> attempted += target; target == healthy }, { origin = it })
            assertEquals(healthy, gate.ensure(12_100)); assertEquals(healthy, origin)
            assertTrue(attempted.size <= pool.size)
            assertEquals(attempted.distinct(), attempted)
        }
    }

    @Test fun concurrentLoginPreparationsShareOneProbe() = runBlocking {
        var origin = pool[0]; var probes = 0
        val gate = LoginRoutePreflight({ 100L }, { origin }, { pool }, {}, { _, _ -> probes++; delay(20); true }, { origin = it })
        val results = (1..8).map { async { gate.ensure(12_100) } }.awaitAll()
        assertTrue(results.all { it == pool[0] }); assertEquals(1, probes)
    }

    @Test fun preparationTimeoutNeverAllowsBusinessSend() = runBlocking {
        var sends = 0
        val gate = LoginRoutePreflight({ 100L }, { pool[0] }, { pool }, {}, { _, _ -> delay(200); true }, {})
        if (gate.ensure(120) != null) sends++
        assertEquals(0, sends)
    }

    @Test fun failedKnownRouteIsRecheckedAndRetiredProbeCannotBeSelected() = runBlocking {
        var candidates = pool; var selected = pool[0]
        val gate = LoginRoutePreflight({ 100L }, { selected }, { candidates }, {}, { route, _ ->
            if (route == pool[1]) candidates = pool.takeLast(2)
            true
        }, { selected = it })
        gate.succeeded(pool[0]); gate.failed(pool[0])
        val verified = gate.ensure(12_100)
        assertEquals(pool[4], verified)
        selected = pool[5]
        assertEquals(pool[4], verified) // immutable result does not follow the mutable active route
    }
    @Test fun lateInFlightPoolIsAwaitedWithinBudgetThenReevaluated() = runBlocking {
        var origins = DomainPoolPolicy.seeds
        var active = origins[0]
        val pending = CompletableDeferred<Unit>()
        val now = { System.nanoTime() / 1_000_000 }
        val publish = launch { delay(40); origins = pool; pending.complete(Unit) }
        val gate = LoginRoutePreflight(now, { active }, { origins }, {}, { origin, _ -> origin == pool.last() }, { active = it },
            awaitPendingPool = { deadline -> withTimeoutOrNull((deadline-now()).coerceAtLeast(1)) { pending.await() }; Unit })
        assertEquals(pool.last(), gate.ensure(now()+1_000))
        publish.join()
    }

    @Test fun runningCacheExpiresButVersionWatermarkSurvives() {
        var wall = 100_000L
        val cache = ValidatedDomainPoolCache(DomainPoolPolicy.seeds) { wall }
        val current = DomainPoolDocument(4, 600, pool)
        assertTrue(cache.publish(current)); assertEquals(pool, cache.origins())
        wall += 600_000 + 86_400_001
        assertEquals(DomainPoolPolicy.seeds, cache.origins())
        assertFalse(cache.publish(current.copy(version=3)))
        assertEquals(DomainPoolPolicy.seeds, cache.origins())
        assertTrue(cache.publish(current)); assertEquals(pool, cache.origins())
    }

}
