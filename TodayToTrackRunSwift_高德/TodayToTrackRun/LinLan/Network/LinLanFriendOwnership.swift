import Foundation
import ImSDK_Plus_Swift

struct LinLanFriendOwnerScope: Sendable {
    let owner: String
    let generation: UInt64
    let sdk: Int
    var key: String { "\(sdk):\(owner):\(generation)" }
    static func capture() -> LinLanFriendOwnerScope {
        let store = LinLanAppContainer.sessionStore(), session = store.currentSession()
        return LinLanFriendOwnerScope(owner: session.userId, generation: store.generation, sdk: session.resolvedSdkAppId)
    }
    var active: Bool {
        let store = LinLanAppContainer.sessionStore(), session = store.currentSession()
        return !owner.isEmpty && store.generation == generation && session.userId == owner
            && session.resolvedSdkAppId == sdk && LinLanImManager.loginUser == owner
    }
}

extension Notification.Name {
    static let linLanFriendOwnershipChanged = Notification.Name("linLanFriendOwnershipChanged")
}

actor LinLanFriendOwnership {
    static let shared = LinLanFriendOwnership()
    private let engine = LinLanOwnerSyncEngine()
    private var profiles: [String: LinLanChatProfile] = [:]
    private var waiting: Set<String> = []
    func markWaiting(_ target: String, scope: String) { waiting.insert("\(scope):\(target)") }
    func isWaiting(_ target: String, scope: String) -> Bool { waiting.contains("\(scope):\(target)") }

    func synchronize(_ target: String) async -> LinLanFriendOwnerAssignment? {
        let scope = LinLanFriendOwnerScope.capture()
        let result = await engine.synchronize(target: target, scope: scope.key, active: { scope.active }, run: { requestID in
            guard try await LinLanImManager.areBothWayFriends(target), scope.active else { throw LinLanAPIError.message("好友关系尚未确认") }
            return try await LinLanAppContainer.apiClient().syncFriendOwnership(target, requestID: requestID, owner: scope.owner, generation: scope.generation)
        }, refresh: {
            let api = LinLanAppContainer.apiClient()
            if let profile = try? await api.fetchProfile(), scope.active, profile.imUserId == scope.owner {
                await self.publish(profile, scope: scope.key)
            }
            if let peer = try? await api.fetchFriendOwnerBrief(target), scope.active { await self.publish(peer, scope: "\(scope.key):\(target)") }
        })
        guard scope.active, let result else { return nil }
        if result.state != "pending" { waiting.remove("\(scope.key):\(target)") }
        await MainActor.run {
            guard scope.active else { return }
            NotificationCenter.default.post(name: .linLanFriendOwnershipChanged, object: result, userInfo: ["scope": scope.key, "target": target])
        }
        return result
    }

    private func publish(_ profile: LinLanChatProfile, scope: String) {
        if let previous = profiles[scope], previous.ownerRevision > profile.ownerRevision { return }
        profiles[scope] = profile
    }
}

final class LinLanFriendOwnershipListener: V2TIMFriendshipListener {
    private static let lock = NSLock()
    private static var bound: LinLanFriendOwnershipListener?
    private let scope: LinLanFriendOwnerScope
    private init(scope: LinLanFriendOwnerScope) { self.scope = scope }
    static func bind() {
        lock.lock(); defer { lock.unlock() }
        if let bound { V2TIMManager.shared.removeFriendListener(listener: bound) }
        let listener = LinLanFriendOwnershipListener(scope: .capture())
        bound = listener; V2TIMManager.shared.addFriendListener(listener: listener)
    }
    static func unbind() {
        lock.lock(); defer { lock.unlock() }
        if let bound { V2TIMManager.shared.removeFriendListener(listener: bound) }
        bound = nil
    }
    func onFriendListAdded(infoList: [V2TIMFriendInfo]) {
        guard scope.active else { return }
        for info in infoList.prefix(16) {
            let target = info.userID
            Task { _ = await LinLanFriendOwnership.shared.synchronize(target) }
        }
    }
}
