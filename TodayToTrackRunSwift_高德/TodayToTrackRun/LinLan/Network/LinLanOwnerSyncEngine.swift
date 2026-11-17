import Foundation

// Independent from UI/SDK so lifecycle and duplicate-event behavior can be
// exercised using the actual coordinator with a deterministic test backend.
actor LinLanOwnerSyncEngine {
    private var pending: [String: Task<LinLanFriendOwnerAssignment?, Never>] = [:]

    func synchronize(target: String, scope: String,
        active: @escaping @Sendable () -> Bool,
        run: @escaping @Sendable (String) async throws -> LinLanFriendOwnerAssignment,
        refresh: @escaping @Sendable () async -> Void) async -> LinLanFriendOwnerAssignment? {
        let key = "\(scope):\(target)"
        guard !target.isEmpty, active() else { return nil }
        if let work = pending[key] { return await work.value }
        guard pending.count < 32 else { return LinLanFriendOwnerAssignment(state: "pending", reasonCode: "client_queue_full") }
        let requestID = UUID().uuidString
        let work = Task<LinLanFriendOwnerAssignment?, Never> {
            var result = LinLanFriendOwnerAssignment(state: "pending", reasonCode: "synchronization_unavailable", requestId: requestID)
            for wait in [UInt64(0), 500_000_000, 1_500_000_000] {
                if wait > 0 { try? await Task.sleep(nanoseconds: wait) }
                guard active() else { return nil }
                if let confirmed = try? await run(requestID) { result = confirmed }
                guard active() else { return nil }
                await refresh()
                guard active() else { return nil }
                if result.state != "pending" { break }
            }
            return active() ? result : nil
        }
        pending[key] = work
        let value = await work.value
        pending.removeValue(forKey: key)
        return value
    }
}
