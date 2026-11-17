//
//  LinLanGroupMembershipState.swift
//  TodayToTrackRun
//
//  记录“当前账号已离开的群”，用于实时回调缺失或离线重启时过滤旧会话。
//  key 按 SDKAppID + IM userID 隔离；成功拉到完整已加入群列表后再做权威校准。
//

import Foundation

enum LinLanGroupMembershipPolicy {
    /// Topic 会话 ID 由父 Community ID 加 `@TOPIC#_...` 组成。
    /// 成员关系属于父 Community，所以过滤会话时必须按父群归属比较。
    static func membershipGroupID(_ value: String) -> String {
        let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let range = normalized.range(of: "@TOPIC#_") else { return normalized }
        let parent = String(normalized[..<range.lowerBound])
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return parent.isEmpty ? normalized : parent
    }

    static func belongsToJoinedGroup(_ conversationGroupID: String, joinedGroupIDs: Set<String>) -> Bool {
        let exact = conversationGroupID.trimmingCharacters(in: .whitespacesAndNewlines)
        if joinedGroupIDs.contains(exact) { return true }
        let membershipID = membershipGroupID(exact)
        return joinedGroupIDs.contains { membershipGroupID($0) == membershipID }
    }

    static func isKnownRemoved(_ conversationGroupID: String, removedGroupIDs: Set<String>) -> Bool {
        let exact = conversationGroupID.trimmingCharacters(in: .whitespacesAndNewlines)
        if removedGroupIDs.contains(exact) { return true }
        let membershipID = membershipGroupID(exact)
        return removedGroupIDs.contains { membershipGroupID($0) == membershipID }
    }

    /// 完整快照可作为权威成员关系；快照不完整/拉取失败时仅过滤已确认离开的群，
    /// 避免一次网络失败把所有群（特别是 Community）误清空。
    static func visibleGroupIDs(
        conversationGroupIDs: Set<String>,
        joinedGroupIDs: Set<String>?,
        removedGroupIDs: Set<String>
    ) -> Set<String> {
        if let joinedGroupIDs {
            return Set(conversationGroupIDs.filter {
                belongsToJoinedGroup($0, joinedGroupIDs: joinedGroupIDs)
            })
        }
        return Set(conversationGroupIDs.filter {
            !isKnownRemoved($0, removedGroupIDs: removedGroupIDs)
        })
    }
}

struct LinLanGroupMembershipToken: Equatable {
    let scope: String
    let revision: UInt64
}

enum LinLanGroupMembershipState {
    private static let keyPrefix = "linlan.removed-groups.v1."
    private static let lock = NSLock()
    /// 只用于阻止异步群快照覆盖更新的踢人/重新入群事件，不需持久化。
    private static var revisions: [String: UInt64] = [:]

    static func capture(
        scope: String = LinLanImManager.currentAccountStorageScope
    ) -> LinLanGroupMembershipToken {
        lock.lock()
        defer { lock.unlock() }
        return LinLanGroupMembershipToken(scope: scope, revision: revisions[scope, default: 0])
    }

    static func isCurrent(_ token: LinLanGroupMembershipToken) -> Bool {
        guard !token.scope.isEmpty,
              token.scope == LinLanImManager.currentAccountStorageScope else { return false }
        lock.lock()
        defer { lock.unlock() }
        return revisions[token.scope, default: 0] == token.revision
    }

    static func removedGroupIDs(
        scope: String = LinLanImManager.currentAccountStorageScope,
        defaults: UserDefaults = .standard
    ) -> Set<String> {
        guard !scope.isEmpty else { return [] }
        lock.lock()
        defer { lock.unlock() }
        return Set(defaults.stringArray(forKey: keyPrefix + scope) ?? [])
    }

    static func isRemoved(
        groupID: String,
        scope: String = LinLanImManager.currentAccountStorageScope,
        defaults: UserDefaults = .standard
    ) -> Bool {
        removedGroupIDs(scope: scope, defaults: defaults).contains(normalize(groupID))
    }

    static func markRemoved(
        groupIDs: Set<String>,
        scope: String = LinLanImManager.currentAccountStorageScope,
        defaults: UserDefaults = .standard
    ) {
        guard !scope.isEmpty else { return }
        let normalized = Set(groupIDs.map(normalize).filter { !$0.isEmpty })
        guard !normalized.isEmpty else { return }
        lock.lock()
        var stored = Set(defaults.stringArray(forKey: keyPrefix + scope) ?? [])
        stored.formUnion(normalized)
        persist(stored, scope: scope, defaults: defaults)
        revisions[scope, default: 0] &+= 1
        lock.unlock()
    }

    /// 只在拉取期间账号和成员关系都没有变化时写入，防止旧快照把刚被踢的群重新放回。
    static func markRemovedIfCurrent(
        groupIDs: Set<String>,
        token: LinLanGroupMembershipToken,
        defaults: UserDefaults = .standard
    ) -> LinLanGroupMembershipToken? {
        guard !token.scope.isEmpty,
              token.scope == LinLanImManager.currentAccountStorageScope else { return nil }
        let normalized = Set(groupIDs.map(normalize).filter { !$0.isEmpty })
        lock.lock()
        defer { lock.unlock() }
        guard revisions[token.scope, default: 0] == token.revision else { return nil }
        if !normalized.isEmpty {
            var stored = Set(defaults.stringArray(forKey: keyPrefix + token.scope) ?? [])
            stored.formUnion(normalized)
            persist(stored, scope: token.scope, defaults: defaults)
            revisions[token.scope, default: 0] &+= 1
        }
        return LinLanGroupMembershipToken(
            scope: token.scope,
            revision: revisions[token.scope, default: 0]
        )
    }

    /// 只清除确实出现在成功结果里的群，支持被重新合法邀请/申请入群后恢复。
    static func reconcileJoined(
        groupIDs: Set<String>,
        scope: String = LinLanImManager.currentAccountStorageScope,
        defaults: UserDefaults = .standard
    ) {
        guard !scope.isEmpty else { return }
        let normalized = Set(groupIDs.map(normalize).filter { !$0.isEmpty })
        lock.lock()
        var stored = Set(defaults.stringArray(forKey: keyPrefix + scope) ?? [])
        stored.subtract(normalized)
        persist(stored, scope: scope, defaults: defaults)
        revisions[scope, default: 0] &+= 1
        lock.unlock()
    }

    /// 快照式校准：仅在 token 仍有效时清除“已离群”标记，并返回写入后 token。
    static func reconcileJoinedIfCurrent(
        groupIDs: Set<String>,
        token: LinLanGroupMembershipToken,
        defaults: UserDefaults = .standard
    ) -> LinLanGroupMembershipToken? {
        guard !token.scope.isEmpty,
              token.scope == LinLanImManager.currentAccountStorageScope else { return nil }
        let normalized = Set(groupIDs.map(normalize).filter { !$0.isEmpty })
        lock.lock()
        defer { lock.unlock() }
        guard revisions[token.scope, default: 0] == token.revision else { return nil }
        var stored = Set(defaults.stringArray(forKey: keyPrefix + token.scope) ?? [])
        let original = stored
        stored.subtract(normalized)
        if stored != original {
            persist(stored, scope: token.scope, defaults: defaults)
            revisions[token.scope, default: 0] &+= 1
        }
        return LinLanGroupMembershipToken(
            scope: token.scope,
            revision: revisions[token.scope, default: 0]
        )
    }

    private static func normalize(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private static func persist(_ values: Set<String>, scope: String, defaults: UserDefaults) {
        let key = keyPrefix + scope
        if values.isEmpty {
            defaults.removeObject(forKey: key)
        } else {
            defaults.set(values.sorted(), forKey: key)
        }
    }
}
