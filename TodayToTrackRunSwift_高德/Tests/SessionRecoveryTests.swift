import Foundation

@main struct SessionRecoveryTests {
    static func main() async throws {
        let suite = "rui-session-tests-" + UUID().uuidString
        let defaults = UserDefaults(suiteName: suite)!
        defer { defaults.removePersistentDomain(forName: suite) }
        let store = LinLanSessionStore(defaults: defaults)
        let now = Date(timeIntervalSince1970: 0)
        let partial = LinLanAuthSession(userId: "u", refreshToken: "old")
        let fresh = LinLanAuthSession(userId: "u", userSig: "sig", userSigExpiresAt: "1970-01-01T03:00:00Z",
                                      token: "access", tokenExpiresAt: "1970-01-01T01:00:00Z", refreshToken: "new")
        precondition(!LinLanAuthSession().canResume && !LinLanAuthSession().needsSessionRefresh(at: now))
        precondition(partial.canResume && partial.needsSessionRefresh(at: now))
        precondition(!fresh.needsSessionRefresh(at: now))
        var stale = fresh
        stale.userSigExpiresAt = "1970-01-01T00:00:30.100Z"
        precondition(stale.needsSessionRefresh(at: now) && !stale.needsAccessRefresh(at: now))
        stale = fresh
        stale.tokenExpiresAt = "1970-01-01T08:00:10+08:00"
        precondition(stale.needsAccessRefresh(at: now) && stale.needsSessionRefresh(at: now))

        store.saveSession(partial)
        let renewed = try await LinLanRefreshCoordinator().refresh(store: store) { old in
            precondition(old == partial)
            return fresh
        }
        precondition(renewed == fresh && store.currentSession() == fresh)
        for status in [503, 401] {
            do {
                _ = try await LinLanRefreshCoordinator().refresh(store: store) { _ in
                    throw LinLanAPIError.http("test", status: status, retryAfter: nil)
                }
                preconditionFailure("refresh should fail")
            } catch {
                precondition(store.currentSession().canResume == (status == 503))
            }
        }
        store.saveSession(partial)
        do {
            _ = try await LinLanRefreshCoordinator().refresh(store: store) { _ in
                store.clearSession()
                return fresh
            }
            preconditionFailure("late renewal should not undo logout")
        } catch { precondition(!store.currentSession().canResume) }
        // An old refresh failure/success must neither clear nor reject a newer same-user login.
        for rejected in [true, false] {
            store.saveSession(partial)
            let manager = LinLanRefreshCoordinator()
            let result = try await manager.refresh(store: store) { _ in
                store.saveSession(fresh)
                if rejected { throw LinLanAPIError.http("revoked", status: 401, retryAfter: nil) }
                return LinLanAuthSession(userId: "u", userSig: "old-sig", token: "late-access", refreshToken: "late-refresh")
            }
            precondition(result == fresh && store.currentSession() == fresh)
        }
        store.saveSession(partial)
        do {
            _ = try await LinLanRefreshCoordinator().refresh(store: store) { _ in
                store.saveSession(LinLanAuthSession(userId: "other", userSig: "other-sig", token: "other-token", refreshToken: "other-refresh"))
                throw LinLanAPIError.http("revoked", status: 401, retryAfter: nil)
            }
            preconditionFailure("old account should cancel")
        } catch { precondition(error is CancellationError && store.currentSession().userId == "other") }
        print("Passed: empty, fresh, partial, expiring access/signature, rotation, 503, 401 and late refresh")
    }
}
