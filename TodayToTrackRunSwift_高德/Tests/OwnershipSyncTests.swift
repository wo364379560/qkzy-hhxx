import Foundation

private final class Fixture: @unchecked Sendable {
    let lock = NSLock()
    var live = true
    var calls: [String] = []
    var refreshes = 0
    func active() -> Bool { lock.lock(); defer { lock.unlock() }; return live }
    func deactivate() { lock.lock(); live = false; lock.unlock() }
    func record(_ request: String) -> Int { lock.lock(); defer { lock.unlock() }; calls.append(request); return calls.count }
    func refreshed() { lock.lock(); refreshes += 1; lock.unlock() }
}

@main struct OwnershipSyncTests {
    static func main() async {
        let old = LinLanChatProfile.fromAPI(["im_user_id":"old","nickname":"old client"])
        precondition(old.imUserId == "old" && old.isAssigned == nil && old.ownerRevision == 0 && old.ownerState == "unknown")
        let extended = LinLanChatProfile.fromAPI(["im_user_id":"u","owner_revision":UInt64(45),"owner_id":8,"is_assigned":true,"owner_state":"assigned"])
        precondition(extended.ownerRevision == 45 && extended.isAssigned == true && extended.ownerId == 8)
        let fixture = Fixture(), engine = LinLanOwnerSyncEngine()
        await withTaskGroup(of: LinLanFriendOwnerAssignment?.self) { group in
            for _ in 0..<20 { group.addTask {
                await engine.synchronize(target:"peer",scope:"sdk:user:1",active:{fixture.active()},run:{request in
                    _ = fixture.record(request);try? await Task.sleep(nanoseconds:50_000_000)
                    return LinLanFriendOwnerAssignment(state:"migrated")
                },refresh:{fixture.refreshed()})
            } }
            for await result in group { precondition(result?.state == "migrated") }
        }
        precondition(fixture.calls.count == 1 && fixture.refreshes == 1)
        let retry = Fixture()
        let result = await engine.synchronize(target:"retry",scope:"sdk:user:1",active:{retry.active()},run:{request in
            LinLanFriendOwnerAssignment(state:retry.record(request) < 3 ? "pending" : "unchanged")
        },refresh:{retry.refreshed()})
        precondition(result?.state == "unchanged" && retry.calls.count == 3 && Set(retry.calls).count == 1 && retry.refreshes == 3)
        let switched = Fixture()
        let stale = await engine.synchronize(target:"peer",scope:"sdk:user:old-generation",active:{switched.active()},run:{request in
            _ = switched.record(request); switched.deactivate()
            return LinLanFriendOwnerAssignment(state:"migrated")
        },refresh:{switched.refreshed()})
        precondition(stale == nil && switched.refreshes == 0)
        print("PASS: Swift ownership coalescing, stable bounded retries, generation fence, old and extended profile compatibility")
    }
}
