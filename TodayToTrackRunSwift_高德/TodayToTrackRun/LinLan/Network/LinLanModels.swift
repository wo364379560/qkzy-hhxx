//
//  LinLanModels.swift
//  TodayToTrackRun
//

import Foundation

struct LinLanAuthSession: Equatable, Sendable {
    var userId: String = ""
    var userSig: String = ""
    var userSigExpiresAt: String = ""
    var token: String = ""
    var tokenExpiresAt: String = ""
    var refreshToken: String = ""
    var mustChangePassword: Bool = false
    /// 登录回包；0 表示回落常量
    var sdkAppId: Int = 0

    var isLoggedIn: Bool {
        !userId.isEmpty && !userSig.isEmpty && !token.isEmpty
    }

    var canResume: Bool {
        isLoggedIn || (!userId.isEmpty && !refreshToken.isEmpty)
    }

    func needsAccessRefresh(at now: Date = Date()) -> Bool {
        token.isEmpty || Self.expiresSoon(tokenExpiresAt, now)
    }

    func needsSessionRefresh(at now: Date = Date()) -> Bool {
        !userId.isEmpty && !refreshToken.isEmpty &&
            (needsAccessRefresh(at: now) || userSig.isEmpty || Self.expiresSoon(userSigExpiresAt, now))
    }

    private static func expiresSoon(_ timestamp: String, _ now: Date) -> Bool {
        guard !timestamp.isEmpty else { return false } // Legacy sessions still handle a server 401.
        let formatter = ISO8601DateFormatter()
        var date = formatter.date(from: timestamp)
        if date == nil {
            formatter.formatOptions.insert(.withFractionalSeconds)
            date = formatter.date(from: timestamp)
        }
        return date.map { $0.timeIntervalSince(now) <= 60 } ?? false
    }

    var resolvedSdkAppId: Int {
        sdkAppId > 0 ? sdkAppId : LinLanAppConstants.sdkAppId
    }
}

struct LinLanChatProfile: Equatable, Sendable {
    var imUserId: String = ""
    var chuanluId: String = ""
    var nickname: String = ""
    var avatarUrl: String = ""
    var phone: String = ""
    var inviteCode: String = ""
    var canBindInviteCode: Bool = false
    var allowType: Int = 1
    var ownerId: UInt64 = 0
    var ownerName: String = ""
    var ownerRevision: UInt64 = 0
    var isAssigned: Bool? = nil
    var ownerState: String = "unknown"
    var ownerReasonCode: String = ""

    static func fromAPI(_ data: [String: Any]) -> LinLanChatProfile {
        func text(_ key: String) -> String { data[key] as? String ?? "" }
        func number(_ key: String) -> UInt64 { (data[key] as? NSNumber)?.uint64Value ?? UInt64(text(key)) ?? 0 }
        return LinLanChatProfile(imUserId: text("im_user_id"), chuanluId: text("chuanlu_id").isEmpty ? text("wanan_id") : text("chuanlu_id"),
            nickname: text("nickname"), avatarUrl: text("avatar_url"),
            phone: text("phone").isEmpty ? (text("mobile").isEmpty ? text("phone_number") : text("mobile")) : text("phone"),
            inviteCode: text("invite_code"),
            canBindInviteCode: data["can_bind_invite_code"] as? Bool ?? true, allowType: (data["allow_type"] as? NSNumber)?.intValue ?? 1,
            ownerId: number("owner_id"), ownerName: text("owner_name"), ownerRevision: number("owner_revision"),
            isAssigned: data["is_assigned"] as? Bool, ownerState: text("owner_state").isEmpty ? "unknown" : text("owner_state"), ownerReasonCode: text("owner_reason_code"))
    }

    var displaySignalId: String {
        chuanluId.isEmpty ? imUserId : chuanluId
    }
}

struct LinLanFriendOwnerAssignment: Equatable, Sendable {
    var state = "pending"
    var reasonCode = ""
    var requestId = ""
}

struct LinLanSecuritySettings: Equatable {
    var remoteLoginAlert = true
    var loginVerification = false
}

struct LinLanLoginDevice: Equatable {
    var id = ""
    var deviceId = ""
    var deviceName = "未知设备"
    var platform = ""
    var ip = ""
    var lastLoginAt = ""
    var isCurrent = false
    var isOnline = false

    var isWeb: Bool {
        ["web", "h5", "browser"].contains(platform.lowercased())
    }
}

struct LinLanSearchUser: Equatable {
    var imUserId: String = ""
    var wananId: String = ""
    var nickname: String = ""
    var avatarUrl: String = ""
    var canAddFriend: Bool = true
    var addBlockedReason: String = ""

    var displayName: String {
        if !nickname.isEmpty { return nickname }
        if !wananId.isEmpty { return wananId }
        return imUserId
    }
}

enum LinLanAPIError: LocalizedError {
    case message(String, unauthorized: Bool = false)
    case network(String)
    case http(String, status: Int, retryAfter: TimeInterval?)

    var errorDescription: String? {
        switch self {
        case .message(let msg, _): return msg
        case .network(let msg): return msg
        case .http(let msg, _, _): return msg
        }
    }

    var retryAfter: TimeInterval? {
        if case .http(_, _, let delay) = self { return delay }
        return nil
    }

    var isUnauthorized: Bool {
        if case .message(_, let unauthorized) = self { return unauthorized }
        if case .http(_, let status, _) = self { return status == 401 }
        return false
    }
}

// MARK: - 朋友圈 / 社区动态（对齐 ShangLan /community/posts）

struct LinLanCommunityAuthor {
    var userId: String = ""
    var imUserId: String = ""
    var nickname: String = ""
    var avatarUrl: String = ""

    var displayName: String {
        if !nickname.isEmpty { return nickname }
        if !imUserId.isEmpty { return imUserId }
        return userId.isEmpty ? "用户" : userId
    }

    static func parse(_ dict: [String: Any]?) -> LinLanCommunityAuthor? {
        guard let dict else { return nil }
        var a = LinLanCommunityAuthor()
        a.userId = (dict["user_id"] as? String) ?? ""
        a.imUserId = (dict["im_user_id"] as? String) ?? ""
        a.nickname = (dict["nickname"] as? String) ?? ""
        a.avatarUrl = (dict["avatar_url"] as? String) ?? ""
        return a
    }
}

struct LinLanCommunityPost {
    var id: Int = 0
    var postType: String = ""
    var title: String = ""
    var content: String = ""
    var imageUrls: [String] = []
    var likeCount: Int = 0
    var commentCount: Int = 0
    var isLiked: Bool = false
    var isMine: Bool = false
    var createdAt: String = ""
    var author: LinLanCommunityAuthor?

    static func parse(_ dict: [String: Any]) -> LinLanCommunityPost? {
        var p = LinLanCommunityPost()
        if let i = dict["id"] as? Int { p.id = i }
        else if let n = dict["id"] as? NSNumber { p.id = n.intValue }
        guard p.id > 0 else { return nil }
        p.postType = (dict["post_type"] as? String) ?? ""
        p.title = (dict["title"] as? String) ?? ""
        p.content = (dict["content"] as? String) ?? ""
        if let urls = dict["image_urls"] as? [String] {
            p.imageUrls = urls.filter { !$0.isEmpty }
        }
        p.likeCount = (dict["like_count"] as? Int)
            ?? (dict["like_count"] as? NSNumber)?.intValue ?? 0
        p.commentCount = (dict["comment_count"] as? Int)
            ?? (dict["comment_count"] as? NSNumber)?.intValue ?? 0
        p.isLiked = (dict["is_liked"] as? Bool)
            ?? (dict["is_liked"] as? NSNumber)?.boolValue ?? false
        p.isMine = (dict["is_mine"] as? Bool)
            ?? (dict["is_mine"] as? NSNumber)?.boolValue ?? false
        p.createdAt = (dict["created_at"] as? String) ?? ""
        p.author = LinLanCommunityAuthor.parse(dict["author"] as? [String: Any])
        return p
    }
}

struct LinLanCommunityComment {
    var id: Int = 0
    var content: String = ""
    var createdAt: String = ""
    var author: LinLanCommunityAuthor?

    static func parse(_ dict: [String: Any]) -> LinLanCommunityComment? {
        var c = LinLanCommunityComment()
        if let i = dict["id"] as? Int { c.id = i }
        else if let n = dict["id"] as? NSNumber { c.id = n.intValue }
        c.content = (dict["content"] as? String) ?? ""
        c.createdAt = (dict["created_at"] as? String) ?? ""
        c.author = LinLanCommunityAuthor.parse(dict["author"] as? [String: Any])
        return c
    }
}

enum LinLanCommunityImagesHelper {
    /// 与 ShangLan 共用 marker，保证双端互通
    private static let markerStart = "<!--shanglan_images:"
    private static let markerEnd = "-->"

    static func parse(content: String) -> (text: String, images: [String]) {
        guard let startRange = content.range(of: markerStart),
              let endRange = content.range(of: markerEnd, range: startRange.upperBound..<content.endIndex) else {
            return (content.trimmingCharacters(in: .whitespacesAndNewlines), [])
        }
        let jsonString = String(content[startRange.upperBound..<endRange.lowerBound])
        var images: [String] = []
        if let data = jsonString.data(using: .utf8),
           let urls = try? JSONSerialization.jsonObject(with: data) as? [String] {
            images = urls.map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }
        }
        var cleaned = content
        cleaned.removeSubrange(startRange.lowerBound..<endRange.upperBound)
        return (cleaned.trimmingCharacters(in: .whitespacesAndNewlines), images)
    }

    static func embed(content: String, images: [String]) -> String {
        let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !images.isEmpty,
              let data = try? JSONSerialization.data(withJSONObject: images),
              let json = String(data: data, encoding: .utf8) else {
            return trimmed
        }
        if trimmed.isEmpty { return "\(markerStart)\(json)\(markerEnd)" }
        return trimmed + "\n\(markerStart)\(json)\(markerEnd)"
    }

    static func images(for post: LinLanCommunityPost) -> [String] {
        if !post.imageUrls.isEmpty { return post.imageUrls }
        return parse(content: post.content).images
    }

    static func displayContent(for post: LinLanCommunityPost) -> String {
        if !post.imageUrls.isEmpty {
            return post.content.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        return parse(content: post.content).text
    }
}
