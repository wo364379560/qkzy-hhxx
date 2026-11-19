import Foundation

actor LinLanRefreshCoordinator {
    private var task: Task<LinLanAuthSession, Error>?
    private var nextAttempt = Date.distantPast
    private var failures = 0
    func refresh(store: LinLanSessionStore, remote: @escaping (LinLanAuthSession) async throws -> LinLanAuthSession) async throws -> LinLanAuthSession {
        if let task { return try await task.value }
        guard Date() >= nextAttempt else { throw LinLanAPIError.message("连接暂时不可用，请稍后重试") }
        let expected = store.currentSession()
        guard !expected.refreshToken.isEmpty else { throw LinLanAPIError.message("登录已过期", unauthorized: true) }
        let operation = Task {
            do {
                let refreshed = try await remote(expected)
                guard refreshed.isLoggedIn, !refreshed.refreshToken.isEmpty, refreshed.userId == expected.userId else {
                    throw LinLanAPIError.message("续签响应不完整，请稍后重试")
                }
                if store.replaceSession(expected, with: refreshed) { return refreshed }
                let latest = store.currentSession()
                if latest.isLoggedIn && latest.userId == expected.userId { return latest }
                throw CancellationError()
            } catch {
                let latest = store.currentSession()
                if latest != expected {
                    if latest.isLoggedIn && latest.userId == expected.userId { return latest }
                    throw CancellationError()
                }
                if (error as? LinLanAPIError)?.isUnauthorized == true {
                    _ = store.replaceSession(expected, with: LinLanAuthSession())
                }
                throw error
            }
        }
        task = operation
        defer { task = nil }
        do {
            let result = try await operation.value
            failures = 0
            nextAttempt = .distantPast
            return result
        } catch {
            if store.currentSession() != expected || error is CancellationError {
                failures = 0
                nextAttempt = .distantPast
            } else if (error as? LinLanAPIError)?.isUnauthorized != true {
                failures += 1
                let serverDelay = (error as? LinLanAPIError)?.retryAfter ?? 0
                nextAttempt = Date().addingTimeInterval(max(serverDelay, min(30, pow(2, Double(min(failures, 5))))) + Double.random(in: 0...0.5))
            }
            throw error
        }
    }
}
