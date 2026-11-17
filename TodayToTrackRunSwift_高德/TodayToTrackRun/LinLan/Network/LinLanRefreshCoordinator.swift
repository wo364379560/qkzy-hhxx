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
            let refreshed = try await remote(expected)
            guard refreshed.isLoggedIn, !refreshed.refreshToken.isEmpty, refreshed.userId == expected.userId, store.replaceSession(expected, with: refreshed) else {
                throw LinLanAPIError.message("登录状态已变化，请重新操作")
            }
            return refreshed
        }
        task = operation
        defer { task = nil }
        do {
            let result = try await operation.value
            failures = 0
            nextAttempt = .distantPast
            return result
        } catch {
            if (error as? LinLanAPIError)?.isUnauthorized == true {
                _ = store.replaceSession(expected, with: LinLanAuthSession())
            } else {
                failures += 1
                let serverDelay = (error as? LinLanAPIError)?.retryAfter ?? 0
                nextAttempt = Date().addingTimeInterval(max(serverDelay, min(30, pow(2, Double(min(failures, 5))))) + Double.random(in: 0...0.5))
            }
            throw error
        }
    }
}
