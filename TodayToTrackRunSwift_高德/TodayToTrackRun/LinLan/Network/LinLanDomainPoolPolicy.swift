import Foundation

nonisolated struct LinLanDomainPoolDocument: Codable, Equatable, Sendable {
    let version: Int
    let ttlSeconds: Int
    let origins: [String]

    enum CodingKeys: String, CodingKey {
        case version
        case ttlSeconds = "ttl_seconds"
        case origins = "api_origins"
    }
}

nonisolated enum LinLanDomainPoolPolicy {
    // Formal API routes, bootstrap trust, and resolver scope are separate sets.
    static let seeds = ["https://api.ruizhihe.com", "https://api.prfgsu.com", "https://api.uktjwg.cn"]
    static let bootstrapOrigins = seeds + ["https://api1.prfgsu.com", "https://api1.uktjwg.cn"]

    private static func relay(_ origin: String) -> String {
        switch URL(string: origin)?.host {
        case "api.prfgsu.com", "api1.prfgsu.com": "001"
        case "api.uktjwg.cn", "api1.uktjwg.cn": "002"
        default: origin
        }
    }

    static func boundedCandidates(_ origins: [String], preferred: String) -> [String] {
        guard !origins.isEmpty else { return [] }
        let start = origins.firstIndex(of: preferred) ?? 0
        let rotated = Array(origins[start...]) + Array(origins[..<start])
        let first = rotated[0]
        let rest = Array(rotated.dropFirst())
        guard let second = rest.first(where: { relay($0) != relay(first) }) ?? rest.first else { return [first] }
        return [first, second]
    }

    static func attemptTimeout(method: String, mayRetry: Bool, remaining: TimeInterval, requested: TimeInterval) -> TimeInterval {
        let available = max(0, remaining)
        return min(requested, method == "GET" && mayRetry ? available / 2 : available)
    }

    static func normalizeOrigin(_ raw: String) -> String? {
        guard let u = URLComponents(string: raw.trimmingCharacters(in: .whitespacesAndNewlines)),
            u.scheme?.lowercased() == "https", let host = u.host?.lowercased(),
            u.user == nil, u.password == nil, u.query == nil, u.fragment == nil,
            u.path.isEmpty || u.path == "/", u.percentEncodedPath.isEmpty || u.percentEncodedPath == "/",
            u.port == nil || u.port == 443, host.count <= 253,
            host.contains("."), !host.hasSuffix(".local"),
            !host.allSatisfy({ $0.isNumber || $0 == "." }),
            host.split(separator: ".", omittingEmptySubsequences: false).allSatisfy({
                $0.range(of: "^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$", options: .regularExpression) != nil
            }) else { return nil }
        return "https://" + host
    }

    static func parseDocument(_ data: Data) -> LinLanDomainPoolDocument? {
        guard data.count <= 65_536,
            let document = try? JSONDecoder().decode(LinLanDomainPoolDocument.self, from: data),
            document.version >= 0, (60...86400).contains(document.ttlSeconds),
            (1...16).contains(document.origins.count) else { return nil }
        let normalized = document.origins.compactMap(normalizeOrigin)
        guard normalized.count == document.origins.count, normalized.allSatisfy({ bootstrapOrigins.contains($0) }) else { return nil }
        var seen = Set<String>()
        return LinLanDomainPoolDocument(version: document.version, ttlSeconds: document.ttlSeconds,
            origins: normalized.filter { seen.insert($0).inserted })
    }

    static func parseBootstrap(_ data: Data) -> LinLanDomainPoolDocument? {
        struct Envelope: Decodable { let code: Int; let data: Payload }
        struct Payload: Decodable {
            let version: Int?
            let ttl_seconds: Int?
            let api_origins: [String]?
            let domain_pool: LinLanDomainPoolDocument?
        }
        guard data.count <= 65_536, let root = try? JSONDecoder().decode(Envelope.self, from: data), root.code == 0 else { return nil }
        let value: LinLanDomainPoolDocument
        if let nested = root.data.domain_pool { value = nested }
        else if let version = root.data.version, let ttl = root.data.ttl_seconds, let origins = root.data.api_origins {
            value = LinLanDomainPoolDocument(version: version, ttlSeconds: ttl, origins: origins)
        } else { return nil }
        guard let encoded = try? JSONEncoder().encode(value) else { return nil }
        return parseDocument(encoded)
    }

    static func accepts(current: LinLanDomainPoolDocument?, next: LinLanDomainPoolDocument) -> Bool {
        guard let current else { return true }
        return next.version > current.version || next == current
    }

    static func healthyPing(_ data: Data) -> Bool {
        struct Ping: Decodable { struct Status: Decodable { let status: String }; let code: Int; let data: Status }
        guard data.count <= 65_536, let response = try? JSONDecoder().decode(Ping.self, from: data) else { return false }
        return response.code == 0 && response.data.status == "ok"
    }

    static func preferMember(_ cached: String?, in origins: [String]) -> [String] {
        guard let cached, let origin = normalizeOrigin(cached), origins.contains(origin) else { return origins }
        return [origin] + origins.filter { $0 != origin }
    }
    static func isGatewayStatus(_ status: Int) -> Bool { [502, 503, 504].contains(status) }

    /// Only the first GET may be replayed. All other failures prepare a future route.
    @MainActor
    static func recoveryRequest(
        _ request: URLRequest, allowReadRetry: Bool, deadline: TimeInterval,
        now: () -> TimeInterval = { ProcessInfo.processInfo.systemUptime },
        currentBaseURL: @MainActor () async -> String,
        recover: @MainActor (String, TimeInterval) async -> Void,
        prepareFuture: @MainActor (String) async -> Void,
        sessionIsCurrent: @MainActor () -> Bool
    ) async throws -> URLRequest? {
        guard sessionIsCurrent() else { throw CancellationError() }
        guard let original = request.url, let components = URLComponents(url: original, resolvingAgainstBaseURL: false),
            let scheme = components.scheme, let host = components.host else { return nil }
        let failed = scheme + "://" + host + (components.port.map { ":\($0)" } ?? "")
        let remaining = deadline - now()
        guard (request.httpMethod ?? "GET") == "GET", allowReadRetry, remaining > 0 else {
            await prepareFuture(failed)
            return nil
        }
        await recover(failed, min(8, remaining))
        guard sessionIsCurrent() else { throw CancellationError() }
        guard now() < deadline, let next = URLComponents(string: await currentBaseURL()),
            let nextHost = next.host, let nextScheme = next.scheme,
            nextHost != host || nextScheme != scheme || (next.port ?? 443) != (components.port ?? 443) else { return nil }
        var target = components
        target.scheme = nextScheme; target.host = nextHost; target.port = next.port
        guard let url = target.url else { return nil }
        var retried = request; retried.url = url
        return retried
    }

}
