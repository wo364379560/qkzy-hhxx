//
//  LinLanGroupPolicy.swift
//  TodayToTrackRun
//
//  对照睿之禾 GroupPolicy
//

import Foundation

enum LinLanGroupPolicy {
    static let memberProfileAccessKey = "inner_attr_member_profile_access"
    static let muteAdminAlsoKey = "inner_attr_mute_admin_also"
    static let voiceCallKey = "inner_attr_group_voice_call"
    static let videoCallKey = "inner_attr_group_video_call"

    static func isMemberProfileAccessEnabled(_ attrs: [String: String]) -> Bool {
        isEnabled(attrs[memberProfileAccessKey], defaultValue: true)
    }

    static func isMuteAdminAlsoEnabled(_ attrs: [String: String]) -> Bool {
        isEnabled(attrs[muteAdminAlsoKey], defaultValue: false)
    }

    static func isVoiceCallEnabled(_ attrs: [String: String]) -> Bool {
        isEnabled(attrs[voiceCallKey], defaultValue: false)
    }

    static func isVideoCallEnabled(_ attrs: [String: String]) -> Bool {
        isEnabled(attrs[videoCallKey], defaultValue: false)
    }

    static func fetchAll(groupId: String) async -> [String: String] {
        (try? await LinLanImManager.getGroupAttributes(
            groupId: groupId,
            keys: [memberProfileAccessKey, muteAdminAlsoKey, voiceCallKey, videoCallKey]
        )) ?? [:]
    }

    /// 拉取「允许查看成员资料」。失败按禁止处理（fail-closed），与 Android 一致。
    static func fetchMemberProfileAccessEnabled(groupId: String) async -> Bool {
        for _ in 0..<3 {
            do {
                let attrs = try await LinLanImManager.getGroupAttributes(
                    groupId: groupId,
                    keys: [memberProfileAccessKey]
                )
                return isMemberProfileAccessEnabled(attrs)
            } catch {
                try? await Task.sleep(nanoseconds: 200_000_000)
            }
        }
        return false
    }

    static func updateMemberProfileAccess(groupId: String, enabled: Bool) async throws {
        try await LinLanImManager.setGroupAttributes(
            groupId: groupId,
            attributes: [memberProfileAccessKey: enabled ? "1" : "0"]
        )
    }

    static func updateMuteAdminAlso(groupId: String, enabled: Bool) async throws {
        try await LinLanImManager.setGroupAttributes(
            groupId: groupId,
            attributes: [muteAdminAlsoKey: enabled ? "1" : "0"]
        )
    }

    static func updateVoiceCallAccess(groupId: String, enabled: Bool) async throws {
        try await LinLanImManager.setGroupAttributes(
            groupId: groupId,
            attributes: [voiceCallKey: enabled ? "1" : "0"]
        )
    }

    static func updateVideoCallAccess(groupId: String, enabled: Bool) async throws {
        try await LinLanImManager.setGroupAttributes(
            groupId: groupId,
            attributes: [videoCallKey: enabled ? "1" : "0"]
        )
    }

    private static func isEnabled(_ raw: String?, defaultValue: Bool) -> Bool {
        let n = raw?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        if n.isEmpty { return defaultValue }
        return n == "1" || n == "true" || n == "yes" || n == "on"
    }
}
