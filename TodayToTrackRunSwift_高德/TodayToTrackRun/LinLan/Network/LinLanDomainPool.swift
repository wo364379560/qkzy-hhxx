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
    private var loginPreparation: Task<String?, Never>?
    private var healthyUntil: [String: TimeInterval] = [:]
    private var cooldownUntil: [String: TimeInterval] = [:]
    private var probeCursor = 0
    private var seedOrigins = LinLanDomainPoolPolicy.seeds
    private var fetchedAt: TimeInterval = 0
    private var usingRemotePool = false
    private let wallNow: @Sendable () -> TimeInterval
    private let session: URLSession
    private let defaults: UserDefaults
    private let cacheKey = "linlan.domain_pool.last_good_origin.v5"
    private let documentKey = "linlan.domain_pool.validated_document.v1"
    private let fetchedKey = "linlan.domain_pool.validated_document_fetched.v1"

    init(defaults: UserDefaults = .standard, session: URLSession? = nil, wallNow: @escaping @Sendable () -> TimeInterval = { Date().timeIntervalSince1970 }) {
        self.wallNow = wallNow
        self.defaults = defaults
        self.session = session ?? RuiHTTPDNSNetwork.session(requestTimeout: 3, resourceTimeout: 3)
        if let url = Bundle.main.url(forResource: "appConfig", withExtension: "json"),
            let data = try? Data(contentsOf: url),
            let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let pool = root["domainPool"] as? [String: Any], let raw = pool["apiOrigins"] as? [String] {
            let normalized = raw.compactMap(LinLanDomainPoolPolicy.normalizeOrigin)
            if normalized.count == raw.count && !normalized.isEmpty && normalized.allSatisfy({ LinLanDomainPoolPolicy.businessOrigins.contains($0) }) { origins = normalized }
            if let config = pool["configOrigins"] as? [String] {
                let seeds = config.compactMap(LinLanDomainPoolPolicy.normalizeOrigin)
                    .filter { LinLanDomainPoolPolicy.bootstrapOrigins.contains($0) }
                if !seeds.isEmpty { bootstrapOrigins = seeds }
            }
        }
        seedOrigins = origins
        if let raw = defaults.data(forKey: documentKey),
            let cached = LinLanDomainPoolPolicy.parseDocument(raw) {
            document = cached
            let fetched = defaults.double(forKey: fetchedKey)
            fetchedAt = fetched
            if LinLanDomainPoolPolicy.cacheUsable(fetchedAt: fetched, now: wallNow(), ttl: cached.ttlSeconds) {
                origins = cached.origins
                usingRemotePool = true
                refreshAfter = fetched + Double(cached.ttlSeconds)
            }
        }
        activeOrigin = LinLanDomainPoolPolicy.preferMember(defaults.string(forKey: cacheKey), in: origins)[0]
    }

    func currentBaseURL() -> String {
        expireStalePoolIfNeeded()
        return activeOrigin + LinLanAppConstants.apiPrefix
    }

    // Keep the highest validated version even after its route cache expires.
    private func expireStalePoolIfNeeded() {
        guard usingRemotePool, let document,
            !LinLanDomainPoolPolicy.cacheUsable(fetchedAt: fetchedAt, now: wallNow(), ttl: document.ttlSeconds) else { return }
        usingRemotePool = false
        origins = seedOrigins
        generation += 1
        healthyUntil.removeAll()
        if !origins.contains(activeOrigin) { activeOrigin = origins[0] }
        probeCursor = 0
        refreshAfter = 0
    }

    private func beginRefresh(deadline: TimeInterval) -> Task<Void, Never> {
        if let refreshTask { return refreshTask }
        let task = Task {
            RuiHTTPDNSNetwork.prepareAfterConsent()
            await self.refreshRemotePool(deadline: deadline)
            self.refreshTask = nil
        }
        refreshTask = task
        return task
    }

    func ensureReady() async {
        expireStalePoolIfNeeded()
        guard defaults.bool(forKey: "linlan.prefs.privacy_accepted"),
            wallNow() >= refreshAfter, refreshTask == nil, loginPreparation == nil else { return }
        _ = beginRefresh(deadline: ProcessInfo.processInfo.systemUptime + 6)
    }

    // Return the verified URL snapshot, not a Bool followed by a separate mutable base lookup.
    func prepareForLogin(deadline: TimeInterval) async -> String? {
        expireStalePoolIfNeeded()
        guard defaults.bool(forKey: "linlan.prefs.privacy_accepted"),
            !Task.isCancelled, ProcessInfo.processInfo.systemUptime < deadline else { return nil }
        if origins.contains(activeOrigin), (healthyUntil[activeOrigin] ?? 0) > ProcessInfo.processInfo.systemUptime {
            return activeOrigin + LinLanAppConstants.apiPrefix
        }
        let task: Task<String?, Never>
        if let loginPreparation { task = loginPreparation }
        else {
            let budgetEnd = min(deadline, ProcessInfo.processInfo.systemUptime + 12)
            task = Task {
                let result = await self.prepareColdRoute(deadline: budgetEnd)
                self.loginPreparation = nil
                return result
            }
            loginPreparation = task
        }
        // Joining a single-flight operation must respect each caller's own shorter deadline.
        guard let result = await domainPoolTaskValue(task, deadline: deadline), let origin = result,
            ProcessInfo.processInfo.systemUptime < deadline, !Task.isCancelled else { return nil }
        expireStalePoolIfNeeded()
        guard origins.contains(origin), (healthyUntil[origin] ?? 0) > ProcessInfo.processInfo.systemUptime else { return nil }
        return origin + LinLanAppConstants.apiPrefix
    }

    private func prepareColdRoute(deadline: TimeInterval) async -> String? {
        RuiHTTPDNSNetwork.prepareAfterConsent()
        if wallNow() >= refreshAfter || refreshTask != nil {
            let pending = beginRefresh(deadline: min(deadline, ProcessInfo.processInfo.systemUptime + 4))
            _ = await domainPoolTaskValue(pending, deadline: min(deadline, ProcessInfo.processInfo.systemUptime + 4))
        }
        for _ in 0..<3 {
            expireStalePoolIfNeeded()
            let revision = generation
            if let origin = await probeCandidates(preferred: origins[probeCursor % origins.count], deadline: deadline) { return origin }
            // Old candidates can fail quickly while an already-started refresh is still pending.
            // Wait on that actual task, bounded by the same deadline, and then re-evaluate.
            if let pending = refreshTask { _ = await domainPoolTaskValue(pending, deadline: deadline) }
            guard ProcessInfo.processInfo.systemUptime < deadline, generation != revision else { return nil }
        }
        return nil
    }

    func markAvailable(origin: String) {
        guard origins.contains(origin) else { return }
        healthyUntil[origin] = ProcessInfo.processInfo.systemUptime + 60
        cooldownUntil.removeValue(forKey: origin)
    }

    func prepareFutureRoute(failedOrigin: String) {
        guard defaults.bool(forKey: "linlan.prefs.privacy_accepted"),
            failedOrigin == activeOrigin, !preparingFuture else { return }
        markFailed(origin: failedOrigin)
        preparingFuture = true
        Task {
            await self.failoverIfNeeded(failedOrigin: failedOrigin, budget: 8)
            self.preparingFuture = false
        }
    }

    private func markFailed(origin: String) {
        healthyUntil.removeValue(forKey: origin)
        cooldownUntil[origin] = ProcessInfo.processInfo.systemUptime + 30
    }

    func failoverIfNeeded(failedOrigin: String? = nil, budget: TimeInterval = 8) async {
        guard defaults.bool(forKey: "linlan.prefs.privacy_accepted") else { return }
        expireStalePoolIfNeeded()
        if let failedOrigin {
            markFailed(origin: failedOrigin)
            if failedOrigin != activeOrigin { return }
        } else { markFailed(origin: activeOrigin) }
        // A concurrent caller should not start another scan or inherit its deadline.
        guard budget > 0, failoverTask == nil else { return }
        generation += 1
        let index = origins.firstIndex(of: activeOrigin) ?? 0
        let next = origins[(index + 1) % origins.count]
        let task = Task {
            _ = await self.probeCandidates(preferred: next, deadline: ProcessInfo.processInfo.systemUptime + min(8, budget))
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
        let transport = session
        let task = Task<Data?, Never> {
            guard let (data, response) = try? await transport.data(for: request),
                let http = response as? HTTPURLResponse, http.statusCode == 200, data.count <= 65_536 else { return nil }
            return data
        }
        guard let result = await domainPoolTaskValue(task, deadline: min(deadline, ProcessInfo.processInfo.systemUptime + 3)) else {
            task.cancel()
            return nil
        }
        return result
    }

    private func refreshRemotePool(deadline: TimeInterval) async {
        for origin in LinLanDomainPoolPolicy.boundedCandidates(bootstrapOrigins, preferred: bootstrapOrigins[bootstrapCursor % bootstrapOrigins.count]) {
            guard let data = await get(origin: origin, path: LinLanAppConstants.bootstrapPath, deadline: deadline),
                let next = LinLanDomainPoolPolicy.parseBootstrap(data),
                LinLanDomainPoolPolicy.accepts(current: document, next: next) else { continue }
            document = next
            usingRemotePool = true
            origins = next.origins
            generation += 1
            if !origins.contains(activeOrigin) { activeOrigin = origins[0] }
            let now = wallNow()
            fetchedAt = now
            refreshAfter = now + Double(next.ttlSeconds)
            defaults.set(try? JSONEncoder().encode(next), forKey: documentKey)
            defaults.set(now, forKey: fetchedKey)
            return
        }
        bootstrapCursor = (bootstrapCursor + 1) % bootstrapOrigins.count
        refreshAfter = wallNow() + 30
    }

    private func probeCandidates(preferred: String, deadline: TimeInterval) async -> String? {
        // A background pool may arrive during a probe. Re-evaluate that pool
        // within the original deadline instead of failing the first login.
        for _ in 0..<3 {
            expireStalePoolIfNeeded()
            let snapshot = origins
            let revision = generation
            for origin in LinLanDomainPoolPolicy.recoveryCandidates(snapshot, preferred: preferred) {
                guard ProcessInfo.processInfo.systemUptime < deadline else { return nil }
                if generation != revision { break }
                if (cooldownUntil[origin] ?? 0) > ProcessInfo.processInfo.systemUptime { continue }
                probeCursor = ((snapshot.firstIndex(of: origin) ?? 0) + 1) % snapshot.count
                let data = await get(origin: origin, path: LinLanAppConstants.probePath, deadline: deadline)
                expireStalePoolIfNeeded()
                if generation != revision { break }
                guard let data, LinLanDomainPoolPolicy.healthyPing(data) else {
                    healthyUntil.removeValue(forKey: origin)
                    cooldownUntil[origin] = ProcessInfo.processInfo.systemUptime + 30
                    continue
                }
                guard origins.contains(origin) else { break }
                activeOrigin = origin
                markAvailable(origin: origin)
                defaults.set(origin, forKey: cacheKey)
                return origin
            }
            if generation == revision { return nil }
            if (healthyUntil[activeOrigin] ?? 0) > ProcessInfo.processInfo.systemUptime { return activeOrigin }
        }
        return nil
    }
}

// Await a shared task without cancelling it when one caller times out. Unlike a
// task-group race, this does not wait for the losing shared task to finish.
nonisolated private final class DomainPoolDeadlineResult<Value: Sendable>: @unchecked Sendable {
    private let lock = NSLock()
    private var completed = false
    private var result: Value?
    private var continuation: CheckedContinuation<Value?, Never>?
    func install(_ continuation: CheckedContinuation<Value?, Never>) {
        lock.lock()
        let done = completed; let value = result
        if !done { self.continuation = continuation }
        lock.unlock()
        if done { continuation.resume(returning: value) }
    }
    func complete(_ value: Value?) {
        lock.lock()
        guard !completed else { lock.unlock(); return }
        completed = true; result = value
        let continuation = self.continuation; self.continuation = nil
        lock.unlock()
        continuation?.resume(returning: value)
    }
}

nonisolated private func domainPoolTaskValue<Value: Sendable>(_ task: Task<Value, Never>, deadline: TimeInterval) async -> Value? {
    let remaining = deadline - ProcessInfo.processInfo.systemUptime
    guard remaining > 0, !Task.isCancelled else { return nil }
    let result = DomainPoolDeadlineResult<Value>()
    return await withTaskCancellationHandler {
        await withCheckedContinuation { continuation in
            result.install(continuation)
            Task { result.complete(await task.value) }
            Task {
                try? await Task.sleep(nanoseconds: UInt64(min(remaining, 60) * 1_000_000_000))
                result.complete(nil)
            }
        }
    } onCancel: { result.complete(nil) }
}
