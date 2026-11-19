package com.balabalaim.gcloud.tim.push.appdev.data.network

import okhttp3.Dns
import java.net.InetAddress
import java.net.UnknownHostException

/** OkHttp retains the original URL, Host, TLS SNI and hostname verifier. */
internal class ScopedHttpDns(
    private val signedLookup: (String) -> List<InetAddress>,
    private val system: Dns = Dns.SYSTEM,
    private val diagnostic: (String, String, Long) -> Unit = { _, _, _ -> },
) : Dns {
    override fun lookup(hostname: String): List<InetAddress> {
        if (hostname.lowercase() !in DomainPoolPolicy.httpDnsHosts) return system.lookup(hostname)
        val started = System.nanoTime()
        val result = runCatching { signedLookup(hostname) }
        val signed = result.getOrDefault(emptyList()).filter(::isPublicAddress).distinct()
        diagnostic(hostname, if (result.isFailure) "httpdns_failed" else if (signed.isEmpty()) "httpdns_empty" else "httpdns_success", (System.nanoTime() - started) / 1_000_000)
        if (signed.isNotEmpty()) return interleave(signed)
        val systemAt = System.nanoTime()
        val fallback = try { system.lookup(hostname).filter(::isPublicAddress).distinct() }
        catch (error: UnknownHostException) {
            diagnostic(hostname, "system_dns_failed", (System.nanoTime() - systemAt) / 1_000_000)
            throw error
        }
        diagnostic(hostname, "system_dns_fallback", (System.nanoTime() - systemAt) / 1_000_000)
        if (fallback.isEmpty()) throw UnknownHostException("No public edge address for $hostname")
        return interleave(fallback)
    }

    companion object {
        fun interleave(addresses: List<InetAddress>): List<InetAddress> {
            val unique = addresses.distinct()
            val v4 = unique.filter { it.address.size == 4 }
            val v6 = unique.filter { it.address.size == 16 }
            return buildList {
                for (i in 0 until maxOf(v4.size, v6.size)) {
                    v4.getOrNull(i)?.let(::add)
                    v6.getOrNull(i)?.let(::add)
                }
            }
        }
        fun literal(value: String): InetAddress? {
            if (!value.matches(Regex("[0-9a-fA-F:.]+")) || (!value.contains('.') && !value.contains(':'))) return null
            return runCatching { InetAddress.getByName(value) }.getOrNull()?.takeIf(::isPublicAddress)
        }

        fun isPublicAddress(address: InetAddress): Boolean {
            if (address.isAnyLocalAddress || address.isLoopbackAddress || address.isLinkLocalAddress || address.isSiteLocalAddress || address.isMulticastAddress) return false
            val b = address.address.map { it.toInt() and 255 }
            return if (b.size == 4) {
                b[0] != 0 && b[0] != 10 && b[0] != 127 && b[0] < 224 &&
                    !(b[0] == 100 && b[1] in 64..127) && !(b[0] == 169 && b[1] == 254) &&
                    !(b[0] == 172 && b[1] in 16..31) &&
                    !(b[0] == 192 && (b[1] == 168 || (b[1] == 0 && b[2] in listOf(0, 2)))) &&
                    !(b[0] == 198 && (b[1] in 18..19 || (b[1] == 51 && b[2] == 100))) &&
                    !(b[0] == 203 && b[1] == 0 && b[2] == 113)
            } else {
                if (b.size != 16) return false
                if (b.take(12) == listOf(0, 0x64, 0xff, 0x9b, 0, 0, 0, 0, 0, 0, 0, 0)) {
                    return isPublicAddress(InetAddress.getByAddress(address.address.copyOfRange(12, 16)))
                }
                (b[0] and 0xe0) == 0x20 && !(b[0] == 0x20 && b[1] == 1 &&
                    ((b[2] == 0x0d && b[3] == 0xb8) || b[2] < 2))
            }
        }
    }
}
