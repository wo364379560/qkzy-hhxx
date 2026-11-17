//
//  LinLanImManager+Group.swift
//  TodayToTrackRun
//
//  对照睿之禾 ImManager 群管理能力
//

import Foundation
import ImSDK_Plus_Swift

struct LinLanGroupInfoSnapshot {
    let groupId: String
    let groupName: String
    let faceUrl: String?
    let notification: String
    let owner: String
    let role: UInt
    let memberCount: UInt
    let allMuted: Bool
    let groupAddOpt: V2TIMGroupAddOpt
    let groupApproveOpt: V2TIMGroupAddOpt
}

extension LinLanImManager {
    static var roleOwner: UInt { UInt(V2TIMGroupMemberRole.V2TIM_GROUP_MEMBER_ROLE_SUPER.rawValue) }
    static var roleAdmin: UInt { UInt(V2TIMGroupMemberRole.V2TIM_GROUP_MEMBER_ROLE_ADMIN.rawValue) }
    static var roleMember: UInt { UInt(V2TIMGroupMemberRole.V2TIM_GROUP_MEMBER_ROLE_MEMBER.rawValue) }

    static func isOwnerOrAdmin(role: UInt) -> Bool {
        role == roleOwner || role == roleAdmin || role == 400 || role == 300
    }

    struct GroupSendPermission {
        let allowsSend: Bool
        let blockedHint: String
        static let allowed = GroupSendPermission(allowsSend: true, blockedHint: "")
    }

    /// 全员禁言（普通成员）或个人禁言时不允许发言/转发。
    static func resolveGroupSendPermission(groupId: String) async -> GroupSendPermission {
        let owner = loginUser
        guard !groupId.isEmpty, !owner.isEmpty else { return GroupSendPermission(allowsSend: false, blockedHint: "群权限待确认") }
        do {
            let info = try await fetchGroupInfo(groupId: groupId)
            guard let me = await fetchSelfGroupMember(groupId: groupId), loginUser == owner else {
                return GroupSendPermission(allowsSend: false, blockedHint: "未确认群成员身份，请重试")
            }
            if me.isMuted { return GroupSendPermission(allowsSend: false, blockedHint: "已被个人禁言") }
            if info.allMuted && !isOwnerOrAdmin(role: me.role) { return GroupSendPermission(allowsSend: false, blockedHint: "全体禁言") }
            return .allowed
        } catch { return GroupSendPermission(allowsSend: false, blockedHint: "群权限查询失败，请稍后重试") }
    }

    static func fetchSelfGroupMember(groupId: String) async -> LinLanGroupMemberItem? {
        let selfId = loginUser
        guard !selfId.isEmpty else { return nil }
        let list: [V2TIMGroupMemberFullInfo] = await withCheckedContinuation { cont in
            V2TIMManager.shared.getGroupMembersInfo(
                groupID: groupId,
                memberList: [selfId],
                succ: { members in
                    cont.resume(returning: members)
                },
                fail: { _, _ in
                    cont.resume(returning: [])
                }
            )
        }
        guard let m = list.first(where: { $0.userID == selfId }) ?? list.first else { return nil }
        let card = m.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let name: String = {
            if !card.isEmpty { return card }
            if let n = m.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !n.isEmpty { return n }
            return m.userID
        }()
        return LinLanGroupMemberItem(
            userId: m.userID,
            displayName: name,
            nameCard: card,
            role: m.role,
            faceUrl: m.faceURL,
            muteUntil: m.muteUntil
        )
    }

    /// 关闭「允许查看成员资料」(禁止私聊) 后：
    /// 群主/管理员可查看任意成员；普通成员只能查看群主/管理员；自己始终可看。
    static func canViewMemberProfile(
        profileAccessEnabled: Bool,
        selfRole: UInt,
        targetRole: UInt,
        targetUserId: String
    ) -> Bool {
        if targetUserId == loginUser { return true }
        if profileAccessEnabled { return true }
        if isOwnerOrAdmin(role: selfRole) { return true }
        return isOwnerOrAdmin(role: targetRole)
    }

    static func fetchGroupInfo(groupId: String) async throws -> LinLanGroupInfoSnapshot {
        let accountScope = currentAccountStorageScope
        let list: [V2TIMGroupInfoResult] = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getGroupsInfo(groupIDList: [groupId], succ: { results in
                cont.resume(returning: results)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        guard let result = list.first, let info = result.info else {
            throw LinLanImError.apiFailed(-1, "群资料不存在")
        }
        guard currentAccountStorageScope == accountScope else { throw CancellationError() }
        let snapshot = LinLanGroupInfoSnapshot(
            groupId: info.groupID ?? groupId,
            groupName: info.groupName ?? groupId,
            faceUrl: info.faceURL,
            notification: info.notification ?? "",
            owner: info.owner,
            role: info.role,
            memberCount: info.memberCount,
            allMuted: info.allMuted,
            groupAddOpt: info.groupAddOpt,
            groupApproveOpt: info.groupApproveOpt
        )
        cacheGroupRole(snapshot.role, groupID: snapshot.groupId, scope: accountScope)
        return snapshot
    }

    static func getGroupAttributes(groupId: String, keys: [String]?) async throws -> [String: String] {
        try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getGroupAttributes(groupID: groupId, keys: keys, succ: { map in
                cont.resume(returning: map)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupAttributes(groupId: String, attributes: [String: String]) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setGroupAttributes(groupID: groupId, attributes: attributes, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupName(groupId: String, name: String) async throws {
        let info = V2TIMGroupInfo()
        info.groupID = groupId
        info.groupName = name
        try await setGroupInfo(info)
    }

    static func setGroupFaceUrl(groupId: String, faceUrl: String) async throws {
        let info = V2TIMGroupInfo()
        info.groupID = groupId
        info.faceURL = faceUrl
        try await setGroupInfo(info)
    }

    static func setGroupAddOpt(groupId: String, opt: V2TIMGroupAddOpt) async throws {
        let info = V2TIMGroupInfo()
        info.groupID = groupId
        info.groupAddOpt = opt
        try await setGroupInfo(info)
    }

    static func setGroupApproveOpt(groupId: String, opt: V2TIMGroupAddOpt) async throws {
        let info = V2TIMGroupInfo()
        info.groupID = groupId
        info.groupApproveOpt = opt
        try await setGroupInfo(info)
    }

    static func setGroupInfo(_ info: V2TIMGroupInfo) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setGroupInfo(info: info, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setSelfGroupNameCard(groupId: String, nameCard: String) async throws {
        let selfId = loginUser
        guard !selfId.isEmpty else { throw LinLanImError.apiFailed(-1, "未登录") }
        let info = V2TIMGroupMemberFullInfo()
        info.userID = selfId
        info.nameCard = nameCard
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setGroupMemberInfo(groupID: groupId, info: info, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func inviteUsersToGroup(groupId: String, userIds: [String]) async throws {
        var seen = Set<String>()
        let ids = userIds.filter { !$0.isEmpty && seen.insert($0).inserted }
        for index in stride(from: 0, to: ids.count, by: 20) {
            try Task.checkCancellation()
            try await inviteUsersToGroupBatch(groupId: groupId, userIds: Array(ids[index..<min(index + 20, ids.count)]))
        }
    }

    private static func inviteUsersToGroupBatch(groupId: String, userIds: [String]) async throws {
        guard !userIds.isEmpty else { return }
        do {
            try await inviteUsersToGroupRaw(groupId: groupId, userIds: userIds)
        } catch {
            // TIM 将 inviteOpt=FORBID 时连群主也会 10007；群主/管理员临时放开为「需要验证」再邀请
            let isForbid: Bool = {
                if case LinLanImError.apiFailed(let code, let desc) = error {
                    return code == 10007
                        || desc.localizedCaseInsensitiveContains("does not enable invite")
                }
                return false
            }()
            guard isForbid else { throw error }
            let info = try await fetchGroupInfo(groupId: groupId)
            guard isOwnerOrAdmin(role: info.role) else {
                throw LinLanImError.apiFailed(
                    10007,
                    "该群未开启邀请进群"
                )
            }
            let previous = info.groupApproveOpt
            try await setGroupApproveOpt(groupId: groupId, opt: .V2TIM_GROUP_ADD_AUTH)
            do {
                try await inviteUsersToGroupRaw(groupId: groupId, userIds: userIds)
            } catch {
                try? await setGroupApproveOpt(groupId: groupId, opt: previous)
                throw error
            }
            try? await setGroupApproveOpt(groupId: groupId, opt: previous)
        }
    }

    private static func inviteUsersToGroupRaw(groupId: String, userIds: [String]) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.inviteUserToGroup(groupID: groupId, userList: userIds, succ: { results in
                let incomplete = results.filter {
                    $0.result != .V2TIM_GROUP_MEMBER_RESULT_SUCC && $0.result != .V2TIM_GROUP_MEMBER_RESULT_INVALID
                }
                if !incomplete.isEmpty {
                    cont.resume(throwing: LinLanImError.apiFailed(-1, "部分好友未入群或正在等待确认，请在群设置中查看"))
                } else { cont.resume() }
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    /// 待处理的入群申请 / 群邀请（含「邀请需被邀请人同意」）
    static func fetchPendingGroupApplications() async throws -> [V2TIMGroupApplication] {
        let result: V2TIMGroupApplicationResult = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getGroupApplicationList(succ: { res in
                cont.resume(returning: res)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        return result.applicationList.filter {
            $0.handleStatus == .V2TIM_GROUP_APPLICATION_HANDLE_STATUS_UNHANDLED
        }
    }

    static func acceptGroupApplication(_ application: V2TIMGroupApplication, reason: String? = nil) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.acceptGroupApplication(application: application, reason: reason, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func refuseGroupApplication(_ application: V2TIMGroupApplication, reason: String? = nil) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.refuseGroupApplication(application: application, reason: reason, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupApplicationRead() async {
        await withCheckedContinuation { (cont: CheckedContinuation<Void, Never>) in
            V2TIMManager.shared.setGroupApplicationRead(succ: {
                cont.resume()
            }, fail: { _, _ in
                cont.resume()
            })
        }
    }

    static func groupApplicationTitle(_ app: V2TIMGroupApplication) -> String {
        let from = (app.fromUserNickName?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap {
            $0.isEmpty ? nil : $0
        } ?? (app.fromUser ?? "有人")
        switch app.applicationType {
        case .V2TIM_GROUP_INVITE_APPLICATION_NEED_APPROVED_BY_INVITEE:
            return "\(from) 邀请你加入群"
        case .V2TIM_GROUP_INVITE_APPLICATION_NEED_APPROVED_BY_ADMIN:
            return "\(from) 邀请成员入群，待管理员审批"
        case .V2TIM_GROUP_JOIN_APPLICATION_NEED_APPROVED_BY_ADMIN:
            return "\(from) 申请加入群"
        default:
            return "\(from) 的群相关申请"
        }
    }

    static func groupApplicationSubtitle(_ app: V2TIMGroupApplication) -> String {
        let msg = (app.requestMsg ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let gid = app.groupID
        if msg.isEmpty { return "群 ID：\(gid)" }
        return "\(msg) · \(gid)"
    }

    static func kickGroupMember(groupId: String, userIds: [String]) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.kickGroupMember(
                groupID: groupId,
                memberList: userIds,
                reason: nil,
                duration: 0,
                succ: { _ in cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func muteGroupMember(groupId: String, userId: String, seconds: UInt) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.muteGroupMember(
                groupID: groupId,
                memberUserID: userId,
                muteTimeSeconds: seconds,
                succ: { cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func muteAllGroupMembers(groupId: String, muted: Bool) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.muteAllGroupMembers(groupID: groupId, isMute: muted, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupMemberRole(groupId: String, userId: String, role: V2TIMGroupMemberRole) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setGroupMemberRole(
                groupID: groupId,
                memberUserID: userId,
                newRole: role,
                succ: { cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func quitGroup(groupId: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.quitGroup(groupID: groupId, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func dismissGroup(groupId: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.dismissGroup(groupID: groupId, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupMute(groupId: String, muted: Bool) async throws {
        let opt: V2TIMReceiveMessageOpt = muted
            ? .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
            : .V2TIM_RECEIVE_MESSAGE
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setGroupReceiveMessageOpt(groupID: groupId, opt: opt, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func isGroupMuted(groupId: String) async -> Bool {
        await withCheckedContinuation { cont in
            V2TIMManager.shared.getConversation(conversationID: "group_\(groupId)", succ: { conv in
                cont.resume(returning: conv.recvOpt == .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
    }

    static func groupAddOptLabel(_ opt: V2TIMGroupAddOpt) -> String {
        switch opt {
        case .V2TIM_GROUP_ADD_ANY: return "允许任何人"
        case .V2TIM_GROUP_ADD_AUTH: return "需要验证"
        case .V2TIM_GROUP_ADD_FORBID: return "禁止加群"
        default: return "未知"
        }
    }

    static func nextGroupAddOpt(_ opt: V2TIMGroupAddOpt) -> V2TIMGroupAddOpt {
        switch opt {
        case .V2TIM_GROUP_ADD_ANY: return .V2TIM_GROUP_ADD_AUTH
        case .V2TIM_GROUP_ADD_AUTH: return .V2TIM_GROUP_ADD_FORBID
        default: return .V2TIM_GROUP_ADD_ANY
        }
    }

    static func selfNameCard(in members: [LinLanGroupMemberItem]) -> String {
        members.first(where: { $0.userId == loginUser })?.displayName ?? ""
    }
}
