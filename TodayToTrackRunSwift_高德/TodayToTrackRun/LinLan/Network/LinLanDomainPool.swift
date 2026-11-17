import Foundation

actor LinLanDomainPool {
    static let shared = LinLanDomainPool()
    private var origins = LinLanDomainPoolPolicy.seeds
    private var activeOrigin = LinLanDomainPoolPolicy.seeds[0]
    private var bootstrapOrigins = LinLanDomainPoolPolicy.bootstrapOrigins
    private var bootstrapCursor = 0
    private var document: LinLanDomainPoolDocument?
    private var refreshAfter: TimeInterval = 0
    private var generation = 0
    private var refreshTask: Task<Void, Never>?
    private var failoverTask: Task<Void, Never>?
    private var preparingFuture = false
    private let session = RuiHTTPDNSNetwork.session(requestTimeout: 3, resourceTimeout: 3)
    private let cacheKey = "linlan.domain_pool.last_good_origin.v5"
    private let documentKey = "linlan.domain_pool.validated_document.v1"
    private let fetchedKey = "linlan.domain_pool.validated_document_fetched.v1"

    private init() {
        if let url = Bundle.main.url(forResource: "appConfig", withExtension: "json"),
            let data = try? Data(contentsOf: url),
            let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let pool = root["domainPool"] as? [String: Any], let raw = pool["apiOrigins"] as? [String] {
            let normalized = raw.compactMap(LinLanDomainPoolPolicy.normalizeOrigin)
            if normalized.count == raw.count && !normalized.isEmpty { origins = normalized }
            if let config = pool["configOrigins"] as? [String] {
                let seeds = config.compactMap(LinLanDomainPoolPolicy.normalizeOrigin)
                    .filter { LinLanDomainPoolPolicy.bootstrapOrigins.contains($0) }
                if !seeds.isEmpty { bootstrapOrigins = seeds }
            }
        }
        if let raw = UserDefaults.standard.data(forKey: documentKey),
            let cached = LinLanDomainPoolPolicy.parseDocument(raw) {
            document = cached
            origins = cached.origins
            refreshAfter = UserDefaults.standard.double(forKey: fetchedKey) + Double(cached.ttlSeconds)
        }
        activeOrigin = LinLanDomainPoolPolicy.preferMember(UserDefaults.standard.string(forKey: cacheKey), in: origins)[0]
    }

    func currentBaseURL() -> String { activeOrigin + LinLanAppConstants.apiPrefix }

    func ensureReady() async {
        guard UserDefaults.standard.bool(forKey: "linlan.prefs.privacy_accepted"),
            Date().timeIntervalSince1970 >= refreshAfter, refreshTask == nil else { return }
        refreshTask = Task {
            RuiHTTPDNSNetwork.prepareAfterConsent()
            let deadline = ProcessInfo.processInfo.systemUptime + 18
            await self.refreshRemotePool(deadline: deadline)
            await self.probeCandidates(preferred: self.activeOrigin, deadline: deadline)
            self.refreshTask = nil
        }
    }

    func prepareFutureRoute(failedOrigin: String) {
        guard UserDefaults.standard.bool(forKey: "linlan.prefs.privacy_accepted"),
            failedOrigin == activeOrigin, !preparingFuture else { return }
        preparingFuture = true
        Task {
            await self.failoverIfNeeded(failedOrigin: failedOrigin, budget: 8)
            self.preparingFuture = false
        }
    }

    func failoverIfNeeded(failedOrigin: String? = nil, budget: TimeInterval = 8) async {
        guard UserDefaults.standard.bool(forKey: "linlan.prefs.privacy_accepted") else { return }
        if let failedOrigin, failedOrigin != activeOrigin { return }
        // A concurrent caller should not start another scan or inherit its deadline.
        guard budget > 0, failoverTask == nil else { return }
        generation += 1
        let index = origins.firstIndex(of: activeOrigin) ?? 0
        let next = origins[(index + 1) % origins.count]
        let task = Task {
            await self.probeCandidates(preferred: next, deadline: ProcessInfo.processInfo.systemUptime + min(8, budget))
        }
        failoverTask = task
        await task.value
        failoverTask = nil
    }

    private func get(origin: String, path: String, deadline: TimeInterval) async -> Data? {
        let remaining = deadline - ProcessInfo.processInfo.systemUptime
        guard remaining > 0, let url = URL(string: origin + path) else { return nil }
        var request = URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: min(3, remaining))
        request.setValue("no-cache", forHTTPHeaderField: "Cache-Control")
        guard let (data, response) = try? await session.data(for: request),
            let http = response as? HTTPURLResponse, http.statusCode == 200, data.count <= 65_536 else { return nil }
        return data
    }

    private func refreshRemotePool(deadline: TimeInterval) async {
        for origin in LinLanDomainPoolPolicy.boundedCandidates(bootstrapOrigins, preferred: bootstrapOrigins[bootstrapCursor % bootstrapOrigins.count]) {
            guard let data = await get(origin: origin, path: LinLanAppConstants.bootstrapPath, deadline: deadline),
                let next = LinLanDomainPoolPolicy.parseBootstrap(data),
                LinLanDomainPoolPolicy.accepts(current: document, next: next) else { continue }
            document = next
            origins = next.origins
            generation += 1
            if !origins.contains(activeOrigin) { activeOrigin = origins[0] }
            let now = Date().timeIntervalSince1970
            refreshAfter = now + Double(next.ttlSeconds)
            UserDefaults.standard.set(try? JSONEncoder().encode(next), forKey: documentKey)
            UserDefaults.standard.set(now, forKey: fetchedKey)
            return
        }
        bootstrapCursor = (bootstrapCursor + 1) % bootstrapOrigins.count
        refreshAfter = Date().timeIntervalSince1970 + 30
    }

    private func probeCandidates(preferred: String, deadline: TimeInterval) async {
        let snapshot = origins
        let revision = generation
        for origin in LinLanDomainPoolPolicy.boundedCandidates(snapshot, preferred: preferred) {
            guard ProcessInfo.processInfo.systemUptime < deadline, generation == revision else { return }
            guard let data = await get(origin: origin, path: LinLanAppConstants.probePath, deadline: deadline),
                LinLanDomainPoolPolicy.healthyPing(data) else { continue }
            guard generation == revision, origins.contains(origin) else { return }
            activeOrigin = origin
            UserDefaults.standard.set(origin, forKey: cacheKey)
            return
        }
    }
}
