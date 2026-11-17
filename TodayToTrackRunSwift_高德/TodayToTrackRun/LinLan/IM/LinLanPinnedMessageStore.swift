//
//  LinLanPinnedMessageStore.swift
//  TodayToTrackRun
//
//  本地置顶（对照 Android PinnedMessageStore）
//

import Foundation

struct LinLanPinnedMessage: Codable, Equatable {
    let msgId: String
    let summary: String
    let sender: String

    var displayLine: String {
        let s = sender.trimmingCharacters(in: .whitespacesAndNewlines)
        if s.isEmpty { return summary }
        return "\(s): \(summary)"
    }
}

enum LinLanPinnedMessageStore {
    private static let prefix = "linlan.pinned.v2."
    private static let legacyPrefix = "linlan.pinned."

    static func list(conversationId: String) -> [LinLanPinnedMessage] {
        let key = prefix + conversationId
        if let data = UserDefaults.standard.data(forKey: key),
           let decoded = try? JSONDecoder().decode([LinLanPinnedMessage].self, from: data) {
            return decoded.filter { !LinLanImManager.isKnownRecalledMessage($0.msgId) }
        }
        // 兼容旧版仅存 msgId
        let legacy = UserDefaults.standard.stringArray(forKey: legacyPrefix + conversationId) ?? []
        return legacy.filter { !LinLanImManager.isKnownRecalledMessage($0) }.map { LinLanPinnedMessage(msgId: $0, summary: "[消息]", sender: "") }
    }

    static func pinnedMsgIds(conversationId: String) -> [String] {
        list(conversationId: conversationId).map(\.msgId)
    }

    static func isPinned(conversationId: String, msgId: String) -> Bool {
        list(conversationId: conversationId).contains { $0.msgId == msgId }
    }

    static func setPinned(
        conversationId: String,
        msgId: String,
        pinned: Bool,
        summary: String = "[消息]",
        sender: String = ""
    ) {
        var pins = list(conversationId: conversationId)
        pins.removeAll { $0.msgId == msgId }
        if pinned {
            pins.insert(
                LinLanPinnedMessage(msgId: msgId, summary: summary, sender: sender),
                at: 0
            )
        }
        persist(conversationId: conversationId, pins: pins)
    }

    private static func persist(conversationId: String, pins: [LinLanPinnedMessage]) {
        let key = prefix + conversationId
        if let data = try? JSONEncoder().encode(pins) {
            UserDefaults.standard.set(data, forKey: key)
        }
        UserDefaults.standard.removeObject(forKey: legacyPrefix + conversationId)
    }

    static func clear(conversationId: String) {
        UserDefaults.standard.removeObject(forKey: prefix + conversationId)
        UserDefaults.standard.removeObject(forKey: legacyPrefix + conversationId)
    }

    /// 退出账号时清理本地置顶（未按 userId 分库）
    static func clearAll() {
        let defaults = UserDefaults.standard
        let keys = defaults.dictionaryRepresentation().keys.filter {
            $0.hasPrefix(prefix) || $0.hasPrefix(legacyPrefix)
        }
        keys.forEach { defaults.removeObject(forKey: $0) }
    }
}
