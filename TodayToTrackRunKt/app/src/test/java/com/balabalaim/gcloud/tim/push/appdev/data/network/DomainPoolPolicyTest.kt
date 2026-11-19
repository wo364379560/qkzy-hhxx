package com.balabalaim.gcloud.tim.push.appdev.data.network

import okhttp3.Dns
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.net.InetAddress
import java.net.UnknownHostException

class DomainPoolPolicyTest {
    @Test fun productionProbePathMatchesAnonymousBackendRoute() {
        assertEquals("/api/v1/chat/ping", LinLanAppConstants.PROBE_PATH)
        assertEquals("/api/v1/public/bootstrap/domain-pool", LinLanAppConstants.BOOTSTRAP_PATH)
    }
    private fun systemDns(resolve: () -> List<InetAddress>) = object : Dns {
        override fun lookup(hostname: String) = resolve()
    }
    private fun body(version: Int, origin: String = "https://api1.prfgsu.com") =
        """{"code":0,"data":{"version":$version,"ttl_seconds":3600,"api_origins":["$origin"]}}"""

    @Test fun currentV4PoolAcceptsCDNsWithoutBroadeningBootstrapOrHTTPDNS() {
        val origins = listOf("https://api.prfgsu.com", "https://api.uktjwg.cn", "https://api1.prfgsu.com", "https://api1.uktjwg.cn", "https://cdnapi1.sazuts.cn", "https://cdnapi2.sazuts.cn")
        val json = JSONObject().put("version", 4).put("ttl_seconds", 600).put("api_origins", org.json.JSONArray(origins))
        val pool = DomainPoolPolicy.parseDocument(json)!!
        assertEquals(origins, pool.origins)
        assertFalse(pool.origins.contains(DomainPoolPolicy.seeds.first()))
        assertEquals(5, DomainPoolPolicy.bootstrapOrigins.size)
        assertEquals(4, DomainPoolPolicy.httpDnsHosts.size)
        assertFalse(DomainPoolPolicy.httpDnsHosts.any { it.endsWith("sazuts.cn") })
        assertEquals(pool, DomainPoolPolicy.parseDocument(JSONObject(pool.toJSON())))
        assertNull(DomainPoolPolicy.parseDocument(JSONObject(json.toString()).put("api_origins", org.json.JSONArray(origins + "https://cdnapi3.sazuts.cn"))))
        assertTrue(DomainPoolPolicy.usableCache(100, 86_400_100))
        assertFalse(DomainPoolPolicy.usableCache(100, 86_400_101))
        assertFalse(DomainPoolPolicy.usableCache(100, 99))
    }

    @Test fun signedLookupExceptionOrTimeoutFallsBackAndReportsTheStage() {
        val public = InetAddress.getByName("8.8.8.8")
        for (error in listOf(UnknownHostException(), java.net.SocketTimeoutException())) {
            val stages = mutableListOf<String>()
            val resolver = ScopedHttpDns({ throw error }, systemDns { listOf(public) }, { _, stage, _ -> stages += stage })
            assertEquals(listOf(public), resolver.lookup("api.prfgsu.com"))
            assertEquals(listOf("httpdns_failed", "system_dns_fallback"), stages)
        }
    }

    @Test fun formalRoutesBootstrapAndResolverHaveIndependentScopes() {
        assertEquals(listOf("https://api.ruizhihe.com", "https://api.prfgsu.com", "https://api.uktjwg.cn"), DomainPoolPolicy.seeds)
        assertEquals(5, DomainPoolPolicy.bootstrapOrigins.size)
        assertTrue(DomainPoolPolicy.bootstrapOrigins.containsAll(DomainPoolPolicy.seeds))
        assertEquals(4, DomainPoolPolicy.httpDnsHosts.size)
        assertFalse(DomainPoolPolicy.httpDnsHosts.contains("api.ruizhihe.com"))
        val routes = listOf("https://api.prfgsu.com", "https://api1.prfgsu.com", "https://api.uktjwg.cn")
        assertEquals(listOf(routes[0], routes[2]), DomainPoolPolicy.boundedCandidates(routes, routes[0]))
        assertEquals(listOf(routes[0]), DomainPoolPolicy.boundedCandidates(routes.take(1), routes[0]))
    }

    @Test fun validatesWholeBootstrapAndRejectsHTMLAndPartialURLs() {
        assertNotNull(DomainPoolPolicy.parseBootstrap(body(0)))
        for (origin in listOf("http://api.prfgsu.com", "https://api.prfgsu.com/api/v1", "https://user:pass@api.prfgsu.com", "https://api.prfgsu.com?x=1", "https://127.0.0.1", "https://[::1]", "https://api.prfgsu.com:8080", "https://untrusted.example")) {
            assertNull(origin, DomainPoolPolicy.parseBootstrap(body(1, origin)))
        }
        assertNull(DomainPoolPolicy.parseBootstrap("<html>challenge</html>"))
        assertNull(DomainPoolPolicy.parseBootstrap(body(-1)))
        assertNull(DomainPoolPolicy.parseBootstrap(body(1).replace("3600", "0")))
        assertNull(DomainPoolPolicy.parseBootstrap(body(1).replace("\"code\":0", "\"code\":403")))
    }

    @Test fun oldOrConflictingVersionCannotReplaceCurrent() {
        val current = DomainPoolPolicy.parseBootstrap(body(8))!!
        assertFalse(DomainPoolPolicy.accepts(current, DomainPoolPolicy.parseBootstrap(body(7))!!))
        assertFalse(DomainPoolPolicy.accepts(current, DomainPoolPolicy.parseBootstrap(body(8, "https://api.uktjwg.cn"))!!))
        assertTrue(DomainPoolPolicy.accepts(current, DomainPoolPolicy.parseBootstrap(body(9))!!))
        assertTrue(DomainPoolPolicy.accepts(current, current))
        assertEquals(current, DomainPoolPolicy.parseDocument(JSONObject(current.toJSON())))
    }

    @Test fun retiredLastGoodOriginIsNeverAddedBack() {
        val current = listOf("https://api1.uktjwg.cn")
        assertEquals(current, DomainPoolPolicy.preferMember(current, "https://api.prfgsu.com"))
        assertEquals(listOf(DomainPoolPolicy.seeds[2]) + DomainPoolPolicy.seeds.filter { it != DomainPoolPolicy.seeds[2] },
            DomainPoolPolicy.preferMember(DomainPoolPolicy.seeds, DomainPoolPolicy.seeds[2]))
    }

    @Test fun healthRequiresExactBusinessContract() {
        assertTrue(DomainPoolPolicy.healthyPing("""{"code":0,"data":{"status":"ok"}}"""))
        for (text in listOf("<html>200 OK</html>", "{}", """{"code":403,"data":{"status":"ok"}}""", """{"code":0,"data":{"status":"failed"}}""")) {
            assertFalse(DomainPoolPolicy.healthyPing(text))
        }
    }

    @Test fun firstReadCannotSpendTheRetryBudgetAndWritesKeepTheirOwnBudget() {
        assertEquals(15_000L, DomainPoolPolicy.attemptBudgetMillis("GET", 0, 30_000))
        assertEquals(12_000L, DomainPoolPolicy.attemptBudgetMillis("GET", 1, 12_000))
        for (method in listOf("POST", "PUT", "DELETE")) assertEquals(30_000L, DomainPoolPolicy.attemptBudgetMillis(method, 0, 30_000))
        assertEquals(0L, DomainPoolPolicy.attemptBudgetMillis("GET", 0, -1))
    }

    @Test fun alternatesAddressFamiliesWithoutDiscardingProviderOrder() {
        val a = InetAddress.getByName("8.8.8.8")
        val b = InetAddress.getByName("1.1.1.1")
        val c = InetAddress.getByName("2001:4860:4860::8888")
        assertEquals(listOf(a, c, b), ScopedHttpDns.interleave(listOf(a, b, c, a)))
        assertEquals(listOf(a, c, b), ScopedHttpDns({ listOf(a, b, c, a) }, systemDns { emptyList() }).lookup("api.prfgsu.com"))
        assertEquals(listOf(a, c, b), ScopedHttpDns({ emptyList() }, systemDns { listOf(a, b, c, a) }).lookup("api1.uktjwg.cn"))
        assertEquals(listOf(c), ScopedHttpDns.interleave(listOf(c)))
    }

    @Test fun allFourExactHostsUseSignedAnswersAndCFDoesNot() {
        var signedCalls = 0
        var systemCalls = 0
        val address = InetAddress.getByName("8.8.8.8")
        val dns = ScopedHttpDns({ signedCalls++; listOf(address) }, systemDns { systemCalls++; listOf(address) })
        DomainPoolPolicy.httpDnsHosts.forEach { assertEquals(listOf(address), dns.lookup(it)) }
        assertEquals(4, signedCalls)
        assertEquals(0, systemCalls)
        dns.lookup("api.ruizhihe.com")
        dns.lookup("thirdparty.example")
        assertEquals(4, signedCalls)
        assertEquals(2, systemCalls)
    }

    @Test fun rejectsPrivateAnswersAndUsesValidatedFallback() {
        val public = InetAddress.getByName("8.8.8.8")
        val private = InetAddress.getByName("127.0.0.1")
        assertEquals(listOf(public), ScopedHttpDns({ listOf(private) }, systemDns { listOf(public) }).lookup("api.prfgsu.com"))
        try {
            ScopedHttpDns({ listOf(private) }, systemDns { listOf(private) }).lookup("api.prfgsu.com")
            fail("private address accepted")
        } catch (_: UnknownHostException) { }
        for (ip in listOf("0.0.0.0", "10.1.2.3", "100.64.0.1", "169.254.1.1", "172.16.0.1", "192.168.1.1", "192.0.2.1", "198.51.100.1", "203.0.113.1", "::1", "fc00::1", "fe80::1", "2001:db8::1", "2001::1", "64:ff9b::7f00:1", "hostname.example")) assertNull(ip, ScopedHttpDns.literal(ip))
        assertNotNull(ScopedHttpDns.literal("2001:4860:4860::8888"))
        assertNotNull(ScopedHttpDns.literal("64:ff9b::808:808"))
    }
}
