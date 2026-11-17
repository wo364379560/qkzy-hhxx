//
//  LinLanGroupMemberCache.swift
//  TodayToTrackRun
//
//  群成员缓存：避免每次 @/资料/管理都全量拉几千人
//

import Foundation

enum LinLanGroupMemberCache {
    /// @ / 轻量场景默认上限，防止大群内存顶满
    static let mentionLimit = 400
    /// 成员列表页可拉更高上限
    static let listLimit = 1500
    private static let ttl: TimeInterval = 90

    private static let lock = NSLock()
    private static var store: [String: Entry] = [:]

    private struct Entry {
        let at: Date
        let list: [LinLanGroupMemberItem]
        let truncated: Bool
    }

    static func invalidate(groupId: String) {
        lock.lock()
        store.removeValue(forKey: groupId)
        lock.unlock()
    }

    static func clearAll() {
        lock.lock()
        store.removeAll()
        lock.unlock()
    }

    static func members(
        groupId: String,
        maxCount: Int = mentionLimit,
        force: Bool = false
    ) async throws -> [LinLanGroupMemberItem] {
        lock.lock()
        let hit = store[groupId]
        lock.unlock()
        if !force,
           let hit,
           Date().timeIntervalSince(hit.at) < ttl,
           hit.list.count >= min(maxCount, hit.list.count),
           (!hit.truncated || hit.list.count >= maxCount) {
            return Array(hit.list.prefix(maxCount))
        }

        let fetched = try await LinLanImManager.fetchGroupMembers(
            groupId: groupId,
            maxCount: maxCount
        )
        lock.lock()
        store[groupId] = Entry(
            at: Date(),
            list: fetched.list,
            truncated: fetched.truncated
        )
        lock.unlock()
        return fetched.list
    }
}
