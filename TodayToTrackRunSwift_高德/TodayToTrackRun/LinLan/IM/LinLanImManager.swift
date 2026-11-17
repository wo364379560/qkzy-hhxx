//
//  LinLanImManager.swift
//  TodayToTrackRun
//

import Foundation
import ImSDK_Plus_Swift

enum LinLanImError: LocalizedError {
    case loginFailed(Int, String)
    case logoutFailed(Int, String)
    case apiFailed(Int, String)
    case timeout

    var errorDescription: String? {
        switch self {
        case .loginFailed(let code, let desc):
            return "IM 登录失败(\(code)): \(desc)"
        case .logoutFailed(let code, let desc):
            return "IM 登出失败(\(code)): \(desc)"
        case .apiFailed(let code, let desc):
            return Self.friendlyMessage(code: code, raw: desc)
        case .timeout:
            return "IM 登录超时"
        }
    }

    /// 常见 TIM 错误转中文，避免直接暴露英文 desc / Err_SNS_xxx
    static func friendlyMessage(code: Int, raw: String) -> String {
        let info = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        let lower = info.lowercased()

        // 邀请 / 禁言等优先按文案识别（SDK 偶发 code 不一致）
        if lower.contains("does not enable invite") {
            return "该群未开启邀请进群，请将「邀请进群方式」改为允许或需要验证"
        }
        if code == 10017
            || code == 20012
            || (lower.contains("mute") && (lower.contains("send") || lower.contains("you") || lower.contains("member")))
            || lower.contains("shutup")
            || lower.contains("all members are muted")
            || lower.contains("group is muted") {
            return "您已被禁言，无法发送消息"
        }
        if lower.contains("only group member") {
            return "你不是该群成员，无法发送消息"
        }
        if lower.contains("community group does not support") {
            return "社群暂不支持发送此类消息"
        }

        switch code {
        case 20007: return "对方已将你拉黑，无法发送消息"
        case 20009: return "你们已不是好友，请先重新添加好友后再发消息"
        case 30001: return "用户不存在"
        case 30004, 30010: return "不能添加自己为好友"
        case 30515: return "对方已经是你的好友"
        case 30516: return "你已将对方拉黑，请先移出黑名单"
        case 30525: return "对方已将你拉黑，无法添加"
        case 30539: return "已发送好友申请，等待对方确认"
        case 30540: return "对方拒绝加好友"
        case 10007: return "该群未开启邀请进群，请将「邀请进群方式」改为允许或需要验证"
        case 10010: return "群组不存在或已解散"
        case 10101: return "无法邀请该用户入群，请确认双方账号归属一致"
        case 38000: return "当前账号属于其他服务空间无法直接添加。"
        case 6014, 6206: return "登录已失效，请重新登录"
        case 6017: return "参数无效，请稍后重试"
        case 6004: return "网络异常，请稍后重试"
        default:
            if info.localizedCaseInsensitiveContains("Reverse_Black_List")
                || info.localizedCaseInsensitiveContains("IN_REVERSE_BLACK") {
                return "对方已将你拉黑，无法添加"
            }
            if info.localizedCaseInsensitiveContains("In_Black_List")
                || info.localizedCaseInsensitiveContains("IN_BLACK_LIST") {
                return "你已将对方拉黑，请先移出黑名单"
            }
            if info.localizedCaseInsensitiveContains("SELF")
                || info.localizedCaseInsensitiveContains("自己") {
                return "不能添加自己为好友"
            }
            if info.isEmpty { return "操作失败(\(code))" }
            // 英文错误码 / 纯英文 desc：不直接展示给用户
            if lower.contains("cross-owner") || info.contains("其他服务空间") || info.contains("跨归属") {
                return "当前账号属于其他服务空间无法直接添加。"
            }
            if info.hasPrefix("Err_") || info.contains("_SNS_") {
                return "操作失败(\(code))"
            }
            let hasChinese = info.unicodeScalars.contains { $0.value >= 0x4E00 && $0.value <= 0x9FFF }
            if hasChinese { return info }
            if lower.contains("network") || lower.contains("timeout") {
                return "网络异常，请稍后重试"
            }
            return code != 0 ? "操作失败，请稍后重试(\(code))" : "操作失败，请稍后重试"
        }
    }
}

struct LinLanFriendItem {
    let userId: String
    let displayName: String
    let remark: String
    let nickName: String
}

struct LinLanGroupItem {
    let groupId: String
    let groupName: String
    let role: UInt
}

struct LinLanJoinedGroupSnapshot {
    let groups: [LinLanGroupItem]
    /// `getJoinedGroupList` 与 `getJoinedCommunityList` 都成功才是完整快照。
    let isComplete: Bool
    /// 拉取完成并校准本地状态后的 token；使用结果前还要再校验，防止刚好收到踢人/切号回调。
    let membershipToken: LinLanGroupMembershipToken
}

enum LinLanImManager {
    static let lock = NSLock()
    private static var initializedAppId: Int32 = 0
    private static var recallSession: UInt64 = 0
    /// 会话列表只有最后一条消息，不携带当前用户在群内的角色。
    /// 按 SDK 空间+账号缓存已查到的角色，确保切号不串数据。
    private static var groupRolesByScope: [String: [String: UInt]] = [:]
    static let recalled = LinLanRecalledMessageState(
        read: { UserDefaults.standard.data(forKey: "linlan.recalled.v1." + $0) },
        write: { UserDefaults.standard.set($1, forKey: "linlan.recalled.v1." + $0) })
    private static func recallSessionSnapshot() -> (generation: UInt64, app: Int32) {
        lock.lock(); defer { lock.unlock() }
        return (recallSession, initializedAppId)
    }
    static func recallScope() -> LinLanRecalledMessageState.Scope? {
        let app = recallSessionSnapshot().app
        return recalled.current().flatMap { $0.key == "\(app):\(loginUser)" ? $0 : nil }
    }
    private static func changeRecallSession() -> UInt64 {
        lock.lock(); defer { lock.unlock() }
        recallSession &+= 1; recalled.deactivate(); return recallSession
    }

    static var loginUser: String {
        V2TIMManager.shared.getLoginUser() ?? ""
    }

    /// 本地持久状态必须同时按 SDK 空间和账号隔离，防止切号串数据。
    static var currentAccountStorageScope: String {
        let user = loginUser.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !user.isEmpty else { return "" }
        lock.lock()
        let appID = initializedAppId
        lock.unlock()
        return "\(appID):\(user)"
    }

    static func cacheGroupRoles(_ groups: [LinLanGroupItem], scope: String) {
        guard !scope.isEmpty else { return }
        lock.lock()
        defer { lock.unlock() }
        var roles = groupRolesByScope[scope] ?? [:]
        for group in groups {
            let groupID = LinLanGroupMembershipPolicy.membershipGroupID(group.groupId)
            guard !groupID.isEmpty, group.role > 0 else { continue }
            roles[groupID] = group.role
        }
        groupRolesByScope[scope] = roles
    }

    static func cacheGroupRole(_ role: UInt, groupID: String, scope: String) {
        guard !scope.isEmpty, role > 0 else { return }
        let membershipID = LinLanGroupMembershipPolicy.membershipGroupID(groupID)
        guard !membershipID.isEmpty else { return }
        lock.lock()
        var roles = groupRolesByScope[scope] ?? [:]
        roles[membershipID] = role
        groupRolesByScope[scope] = roles
        lock.unlock()
    }

    static func cachedGroupRole(groupID: String?) -> UInt? {
        guard let groupID, !groupID.isEmpty else { return nil }
        let scope = currentAccountStorageScope
        guard !scope.isEmpty else { return nil }
        let membershipID = LinLanGroupMembershipPolicy.membershipGroupID(groupID)
        lock.lock()
        let role = groupRolesByScope[scope]?[membershipID]
        lock.unlock()
        guard let role, role > 0 else { return nil }
        return role
    }

    static func initSDK(sdkAppId: Int? = nil) {
        let id = Int32(sdkAppId ?? LinLanAppConstants.sdkAppId)
        guard id > 0 else { return }

        lock.lock()
        defer { lock.unlock() }

        if initializedAppId == id { return }
        if initializedAppId != 0 {
            recallSession &+= 1; recalled.deactivate()
            LinLanFriendOwnershipListener.unbind()
            V2TIMManager.shared.unInitSDK()
            initializedAppId = 0
        }

        let config = V2TIMSDKConfig()
        config.logLevel = .V2TIM_LOG_INFO
        _ = V2TIMManager.shared.initSDK(sdkAppID: id, config: config)
        initializedAppId = id
    }

    static func isLoggedIn(as userId: String) -> Bool {
        !userId.isEmpty && loginUser == userId && V2TIMManager.shared.getLoginStatus() == .V2TIM_STATUS_LOGINED
    }

    static func login(userId: String, userSig: String) async throws {
        let current = loginUser
        if isLoggedIn(as: userId) {
            recalled.activate("\(recallSessionSnapshot().app):\(userId)")
            return
        }
        // 切号：必须先退出旧 IM，否则会话/好友仍读旧账号本地库
        if !current.isEmpty, current != userId {
            await logout()
        }

        let recallGeneration = changeRecallSession()
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.login(userID: userId, userSig: userSig, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.loginFailed(code, desc))
            })
        }
        guard loginUser == userId else {
            throw LinLanImError.loginFailed(-1, "IM 登录态未切换到目标账号")
        }
        let snapshot = recallSessionSnapshot()
        guard snapshot.generation == recallGeneration else { throw CancellationError() }
        recalled.activate("\(snapshot.app):\(userId)")
    }

    @discardableResult
    static func logout() async -> Bool {
        _ = changeRecallSession()
        LinLanFriendOwnershipListener.unbind()
        // 即使 getLoginUser 为空也调一次，避免残留态
        return await withCheckedContinuation { (cont: CheckedContinuation<Bool, Never>) in
            V2TIMManager.shared.logout(succ: {
                cont.resume(returning: true)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
    }

    // MARK: - Conversations

    static func fetchConversations() async throws -> [V2TIMConversation] {
        let accountScope = currentAccountStorageScope
        guard !accountScope.isEmpty else { throw CancellationError() }
        var all: [V2TIMConversation] = []
        var nextSeq: UInt = 0
        var finished = false
        while !finished {
            let page = try await getConversationPage(nextSeq: nextSeq, count: 100)
            guard currentAccountStorageScope == accountScope else { throw CancellationError() }
            all.append(contentsOf: page.list)
            nextSeq = page.nextSeq
            finished = page.isFinished || page.list.isEmpty
        }

        // 补拉已加入群会话，并过滤已退出/被踢后仍残留在本地数据库的旧群会话。
        // Community 单独查询；任一查询失败时不把“不完整结果”当成权威空列表。
        let joinedSnapshot = try? await fetchJoinedGroupSnapshot()
        let groups = joinedSnapshot?.groups ?? []
        let joinedIDs = Set(groups.map(\.groupId))
        let rawGroupIDs = Set(all.compactMap { conversationGroupID($0) })
        let removedIDs = LinLanGroupMembershipState.removedGroupIDs()
        var authorityToken = joinedSnapshot?.membershipToken
        let snapshotIsCurrent = authorityToken.map(LinLanGroupMembershipState.isCurrent) ?? false
        let authoritativeJoinedIDs = joinedSnapshot?.isComplete == true && snapshotIsCurrent ? joinedIDs : nil
        let visibleGroupIDs = LinLanGroupMembershipPolicy.visibleGroupIDs(
            conversationGroupIDs: rawGroupIDs,
            joinedGroupIDs: authoritativeJoinedIDs,
            removedGroupIDs: removedIDs
        )
        if let authoritativeJoinedIDs, let token = authorityToken {
            let missing = Set(rawGroupIDs.filter {
                !LinLanGroupMembershipPolicy.belongsToJoinedGroup($0, joinedGroupIDs: authoritativeJoinedIDs)
            })
            authorityToken = LinLanGroupMembershipState.markRemovedIfCurrent(
                groupIDs: missing,
                token: token
            )
        }
        all.removeAll { conversation in
            guard let groupID = conversationGroupID(conversation) else { return false }
            return !visibleGroupIDs.contains(groupID)
        }
        var known = Set(all.compactMap { $0.conversationID })
        for g in groups {
            let cid = "group_\(g.groupId)"
            if known.contains(cid) { continue }
            if let conv = try? await getConversation(conversationID: cid) {
                guard currentAccountStorageScope == accountScope else { throw CancellationError() }
                all.append(conv)
                known.insert(cid)
            }
        }

        // 每个 getConversation 也是异步边界。期间如果收到 kick/重新入群，
        // 废弃已过期的权威快照，仅按当前已确认离群集合做安全过滤。
        guard currentAccountStorageScope == accountScope else { throw CancellationError() }
        let finalJoinedIDs: Set<String>? = {
            guard joinedSnapshot?.isComplete == true,
                  let authorityToken,
                  LinLanGroupMembershipState.isCurrent(authorityToken) else { return nil }
            return joinedIDs
        }()
        let finalVisibleGroupIDs = LinLanGroupMembershipPolicy.visibleGroupIDs(
            conversationGroupIDs: Set(all.compactMap { conversationGroupID($0) }),
            joinedGroupIDs: finalJoinedIDs,
            removedGroupIDs: LinLanGroupMembershipState.removedGroupIDs(scope: accountScope)
        )
        all.removeAll { conversation in
            guard let groupID = conversationGroupID(conversation) else { return false }
            return !finalVisibleGroupIDs.contains(groupID)
        }

        return all.sorted { lhs, rhs in
            if lhs.isPinned != rhs.isPinned { return lhs.isPinned && !rhs.isPinned }
            return lhs.orderKey > rhs.orderKey
        }
    }

    static func messagePreview(_ msg: V2TIMMessage?, viewerRole: UInt? = nil) -> String {
        guard let msg else { return "" }
        let resolvedRole = viewerRole ?? cachedGroupRole(groupID: msg.groupID)
        if shouldHideGroupRemovalTip(msg, viewerRole: resolvedRole) { return "" }
        if shouldSilentlyHideRevokedMessage(msg) { return "" }
        if isRevokedMessage(msg) { return revokedTipText(msg) }
        if let card = parseMergerCard(msg) {
            return "[聊天记录] \(card.title)"
        }
        switch msg.elemType {
        case .V2TIM_ELEM_TYPE_TEXT:
            return msg.textElem?.text ?? ""
        case .V2TIM_ELEM_TYPE_IMAGE:
            return "[图片]"
        case .V2TIM_ELEM_TYPE_SOUND:
            return "[语音]"
        case .V2TIM_ELEM_TYPE_VIDEO:
            return "[视频]"
        case .V2TIM_ELEM_TYPE_FILE:
            let name = msg.fileElem?.filename ?? ""
            return name.isEmpty ? "[文件]" : "[文件] \(name)"
        case .V2TIM_ELEM_TYPE_LOCATION:
            return "[位置]"
        case .V2TIM_ELEM_TYPE_FACE:
            return "[表情]"
        case .V2TIM_ELEM_TYPE_GROUP_TIPS:
            let tip = groupTipsTipText(msg)
            return tip.isEmpty ? "[群提示]" : tip
        case .V2TIM_ELEM_TYPE_MERGER:
            let title = msg.mergerElem?.title ?? ""
            return title.isEmpty ? "[聊天记录]" : "[聊天记录] \(title)"
        case .V2TIM_ELEM_TYPE_CUSTOM:
            if let rtc = LinLanRtcSignalingPayload.parseFromMessage(msg) {
                return rtc.displayText(isSelf: msg.isSelf)
            }
            if let data = msg.customElem?.data,
               let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                let bid = (root["businessID"] as? String) ?? ""
                switch bid {
                case "contact_card_v1":
                    let name = (root["nickname"] as? String).flatMap { $0.isEmpty ? nil : $0 }
                        ?? (root["userId"] as? String) ?? ""
                    return "[个人名片] \(name)"
                case "group_card_v1":
                    let name = (root["groupName"] as? String).flatMap { $0.isEmpty ? nil : $0 }
                        ?? (root["groupId"] as? String) ?? ""
                    return "[群名片] \(name)"
                default: break
                }
            }
            if let desc = msg.customElem?.desc?.trimmingCharacters(in: .whitespacesAndNewlines), !desc.isEmpty {
                return desc
            }
            return "[自定义消息]"
        default:
            return "[消息]"
        }
    }

    static func formatTime(_ date: Date?) -> String {
        guard let date else { return "" }
        let cal = Calendar.current
        let fmt = DateFormatter()
        if cal.isDateInToday(date) {
            fmt.dateFormat = "HH:mm"
        } else if cal.isDateInYesterday(date) {
            return "昨天"
        } else {
            fmt.dateFormat = "MM/dd"
        }
        return fmt.string(from: date)
    }

    // MARK: - Friends

    static func fetchFriendList() async throws -> [LinLanFriendItem] {
        let list: [V2TIMFriendInfo] = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getFriendList(succ: { infos in
                cont.resume(returning: infos)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        return list.map { info in
            let nick = info.userFullInfo.nickName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let remark = info.friendRemark?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let name = !remark.isEmpty ? remark : (!nick.isEmpty ? nick : info.userID)
            return LinLanFriendItem(
                userId: info.userID,
                displayName: name,
                remark: remark,
                nickName: nick
            )
        }.sorted { $0.displayName.localizedCompare($1.displayName) == .orderedAscending }
    }

    static func fetchIncomingFriendApplications() async throws -> (unread: UInt, list: [V2TIMFriendApplication]) {
        let result: V2TIMFriendApplicationResult = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getFriendApplicationList(succ: { res in
                cont.resume(returning: res)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        let incoming = result.applicationList.filter {
            $0.type == .V2TIM_FRIEND_APPLICATION_COME_IN || $0.type == .V2TIM_FRIEND_APPLICATION_BOTH
        }
        return (result.unreadCount, incoming)
    }

    private static let friendApplicationActions = LinLanFriendApplicationLock()
    static func acceptFriendApplication(_ application: V2TIMFriendApplication) async throws { try await handleFriendApplication(application, accept: true) }
    static func refuseFriendApplication(_ application: V2TIMFriendApplication) async throws { try await handleFriendApplication(application, accept: false) }

    private static func handleFriendApplication(_ application: V2TIMFriendApplication, accept: Bool) async throws {
        let owner = loginUser
        let scope=LinLanFriendOwnerScope.capture()
        let key = "\(scope.key):\(application.userID):\(application.addTime)"
        guard await friendApplicationActions.begin(key) else { throw LinLanAPIError.message("该申请正在处理中") }
        var accepted = false
        do {
            do {
                try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
                    let success: (V2TIMFriendOperationResult) -> Void = { result in
                        if result.resultCode == 0 { cont.resume() }
                        else { cont.resume(throwing: LinLanImError.apiFailed(result.resultCode, result.resultInfo)) }
                    }
                    if accept { V2TIMManager.shared.acceptFriendApplication(application: application, acceptType: .V2TIM_FRIEND_ACCEPT_AGREE_AND_ADD, succ: success, fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }) }
                    else { V2TIMManager.shared.refuseFriendApplication(application: application, succ: success, fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }) }
                }
                accepted = accept
            } catch LinLanImError.apiFailed(30614, _) {
                guard scope.active else { throw CancellationError() }
                let added = try await areBothWayFriends(application.userID)
                if !accept || !added { throw LinLanAPIError.message(added ? "已添加为好友，请刷新列表" : "申请已失效，请重新申请") }
                accepted=accept
            }
            guard scope.active else {throw CancellationError()}
            if accepted {Task {_ = await LinLanFriendOwnership.shared.synchronize(application.userID)}; _ = try? await sendC2CText(userId: application.userID, text: friendGreetingText) }
            await friendApplicationActions.end(key)
        } catch {
            await friendApplicationActions.end(key)
            throw error
        }
    }

    // MARK: - Blacklist

    static func fetchBlackList() async throws -> [LinLanFriendItem] {
        let list: [V2TIMFriendInfo] = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getBlackList(succ: { infos in
                cont.resume(returning: infos)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        return list.map { info in
            let nick = info.userFullInfo.nickName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let name = !nick.isEmpty ? nick : info.userID
            return LinLanFriendItem(
                userId: info.userID,
                displayName: name,
                remark: "",
                nickName: nick
            )
        }
    }

    static func removeFromBlackList(userId: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.deleteFromBlackList(
                userIDList: [userId],
                succ: { _ in cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    // MARK: - Add / manage friends

    enum AddFriendOutcome {
        case addedDirectly
        case pendingConfirm
        case alreadyFriends

        var toast: String {
            switch self {
            case .addedDirectly: return "已添加为好友"
            case .pendingConfirm: return "已发送好友申请"
            case .alreadyFriends: return "对方已经是你的好友"
            }
        }
    }

    static let friendGreetingText = "你我已成为好友，开始聊天吧"

    private static let outgoingFriendActions = LinLanFriendApplicationLock()
    static func areBothWayFriends(_ target: String) async throws -> Bool {
        try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.checkFriend(userIDList:[target],checkType:.V2TIM_FRIEND_TYPE_BOTH,succ:{ rows in
                cont.resume(returning:rows.contains { $0.userID==target && $0.resultCode==0 && $0.relationType == .V2TIM_FRIEND_RELATION_TYPE_BOTH_WAY })
            },fail:{ code,desc in cont.resume(throwing:LinLanImError.apiFailed(code,desc)) })
        }
    }

    static func addFriend(userId: String, wording: String = "请求添加你为好友") async throws -> AddFriendOutcome {
        let uid = userId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !uid.isEmpty else {
            throw LinLanImError.apiFailed(-1, "用户 ID 无效")
        }
        // 自己不能加自己
        if !loginUser.isEmpty, uid == loginUser {
            throw LinLanImError.apiFailed(30004, "不能添加自己为好友")
        }
        let scope=LinLanFriendOwnerScope.capture()
        let actionKey="\(scope.key):\(uid)"
        guard scope.active else {throw LinLanAPIError.message("聊天尚未就绪")}
        guard !(await LinLanFriendOwnership.shared.isWaiting(uid, scope: scope.key)) else {throw LinLanAPIError.message("好友申请待验证")}
        guard await outgoingFriendActions.begin(actionKey) else {throw LinLanAPIError.message("好友申请正在处理中")}
        do {
        let result: V2TIMFriendOperationResult = try await withCheckedThrowingContinuation { cont in
            let app = V2TIMFriendAddApplication()
            app.userID = uid
            app.addWording = wording
            app.addType = .V2TIM_FRIEND_TYPE_BOTH
            V2TIMManager.shared.addFriend(application: app, succ: { res in
                cont.resume(returning: res)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        let outcome: AddFriendOutcome
        switch result.resultCode {
        case 0: outcome = .addedDirectly
        case 30539: outcome = .pendingConfirm
        case 30515: outcome = .alreadyFriends
        default:
            let info = result.resultInfo.trimmingCharacters(in: .whitespacesAndNewlines)
            throw LinLanImError.apiFailed(
                result.resultCode,
                info.isEmpty ? "添加好友失败" : info
            )
        }
        guard scope.active else {throw CancellationError()}
        if outcome == .pendingConfirm {await LinLanFriendOwnership.shared.markWaiting(uid,scope:scope.key)}
        else {Task {_ = await LinLanFriendOwnership.shared.synchronize(uid)}}
        if outcome == .addedDirectly { _ = try? await sendC2CText(userId: userId, text: friendGreetingText) }
        await outgoingFriendActions.end(actionKey)
        return outcome
        } catch {await outgoingFriendActions.end(actionKey);throw error}
    }

    static func syncSelfProfile(nickname: String?, faceUrl: String?, allowType: Int? = nil) async {
        let info = V2TIMUserFullInfo()
        var hasChange = false
        if let nickname, !nickname.isEmpty {
            info.nickName = nickname
            hasChange = true
        }
        if let faceUrl, !faceUrl.isEmpty {
            info.faceURL = faceUrl
            hasChange = true
        }
        if let allowType, let typed = V2TIMFriendAllowType(rawValue: allowType) {
            // 0=AllowAny 1=NeedConfirm 2=DenyAny，与业务库 / TIM 一致
            info.allowType = typed
            hasChange = true
        }
        guard hasChange else { return }
        await withCheckedContinuation { (cont: CheckedContinuation<Void, Never>) in
            V2TIMManager.shared.setSelfInfo(info: info, succ: {
                cont.resume()
            }, fail: { _, _ in
                cont.resume()
            })
        }
    }

    static func isFriend(userId: String) async -> Bool {
        let list = (try? await fetchFriendList()) ?? []
        return list.contains { $0.userId == userId }
    }

    static func fetchFriendRemark(userId: String) async -> String {
        let list = (try? await fetchFriendList()) ?? []
        return list.first(where: { $0.userId == userId })?.remark ?? ""
    }

    static func fetchUserBrief(userId: String) async -> (nickName: String, faceUrl: String?)? {
        await withCheckedContinuation { cont in
            V2TIMManager.shared.getUsersInfo(userIDList: [userId], succ: { list in
                guard let info = list.first else {
                    cont.resume(returning: nil)
                    return
                }
                cont.resume(returning: (info.nickName ?? "", info.faceURL))
            }, fail: { _, _ in
                cont.resume(returning: nil)
            })
        }
    }

    static func setFriendRemark(userId: String, remark: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            let info = V2TIMFriendInfo()
            info.userID = userId
            info.friendRemark = remark
            V2TIMManager.shared.setFriendInfo(info: info, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func deleteFriend(userId: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.deleteFromFriendList(
                userIDList: [userId],
                deleteType: .V2TIM_FRIEND_TYPE_BOTH,
                succ: { _ in cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func addToBlackList(userId: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.addToBlackList(
                userIDList: [userId],
                succ: { _ in cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func deleteConversation(conversationID: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.deleteConversation(conversation: conversationID, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func pinConversation(conversationID: String, pinned: Bool) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.pinConversation(
                conversationID: conversationID,
                isPinned: pinned,
                succ: { cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func isConversationPinned(conversationID: String) async -> Bool {
        await withCheckedContinuation { cont in
            V2TIMManager.shared.getConversation(conversationID: conversationID, succ: { conv in
                cont.resume(returning: conv.isPinned)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
    }

    static func setC2CMute(userId: String, muted: Bool) async throws {
        let opt: V2TIMReceiveMessageOpt = muted
            ? .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
            : .V2TIM_RECEIVE_MESSAGE
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.setC2CReceiveMessageOpt(
                userIDList: [userId],
                opt: opt,
                succ: { cont.resume() },
                fail: { code, desc in cont.resume(throwing: LinLanImError.apiFailed(code, desc)) }
            )
        }
    }

    static func isC2CMuted(userId: String) async -> Bool {
        await withCheckedContinuation { cont in
            V2TIMManager.shared.getC2CReceiveMessageOpt(userIDList: [userId], succ: { list in
                let muted = list.first?.receiveOpt == .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
                cont.resume(returning: muted)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
    }

    static func sendC2CText(userId: String, text: String) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            _ = V2TIMManager.shared.sendC2CTextMessage(text: text, to: userId, succ: {
                cont.resume()
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    // MARK: - Create group

    /// 默认建社群 Community（对照 Android createCommunityGroup）
    struct PartialGroupCreation: LocalizedError {
        let groupId: String
        let reason: String
        var errorDescription: String? { "群已创建，但追加成员未全部完成。请进入群聊后在群设置中继续邀请：\(reason)" }
    }

    static func createCommunityGroup(name: String, memberUserIds: [String]) async throws -> String {
        var seen = Set<String>()
        let ids = memberUserIds.filter { !$0.isEmpty && seen.insert($0).inserted }
        guard ids.count <= 100 else { throw LinLanImError.apiFailed(-1, "最多选择 100 位好友") }
        let groupId: String = try await withCheckedThrowingContinuation { cont in
            let info = V2TIMGroupInfo()
            info.groupType = "Community"
            info.groupName = name
            info.isSupportTopic = false
            info.groupAddOpt = .V2TIM_GROUP_ADD_ANY
            info.groupApproveOpt = .V2TIM_GROUP_ADD_ANY
            let members: [V2TIMCreateGroupMemberInfo]? = ids.isEmpty
                ? nil
                : ids.prefix(20).map { uid in
                    let m = V2TIMCreateGroupMemberInfo()
                    m.userID = uid
                    return m
                }
            V2TIMManager.shared.createCommunity(info: info, memberList: members, succ: { gid in
                cont.resume(returning: gid)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        do {
            try await inviteUsersToGroup(groupId: groupId, userIds: Array(ids.dropFirst(20)))
        } catch {
            throw PartialGroupCreation(groupId: groupId, reason: error.localizedDescription)
        }
        await withCheckedContinuation { (cont: CheckedContinuation<Void, Never>) in
            V2TIMManager.shared.getConversation(conversationID: "group_\(groupId)", succ: { _ in
                cont.resume()
            }, fail: { _, _ in
                cont.resume()
            })
        }
        return groupId
    }

    // MARK: - Groups

    static func fetchJoinedGroups() async throws -> [LinLanGroupItem] {
        try await fetchJoinedGroupSnapshot().groups
    }

    static func fetchJoinedGroupSnapshot() async throws -> LinLanJoinedGroupSnapshot {
        let membershipToken = LinLanGroupMembershipState.capture()
        guard !membershipToken.scope.isEmpty else { throw CancellationError() }
        var map: [String: LinLanGroupItem] = [:]

        let joined: [V2TIMGroupInfo] = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getJoinedGroupList(succ: { infos in
                cont.resume(returning: infos)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        guard currentAccountStorageScope == membershipToken.scope else { throw CancellationError() }
        for g in joined {
            guard let gid = g.groupID?.trimmingCharacters(in: .whitespacesAndNewlines), !gid.isEmpty else { continue }
            let name = (g.groupName?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? gid
            map[gid] = LinLanGroupItem(groupId: gid, groupName: name, role: g.role)
        }

        let communityResult: Result<[V2TIMGroupInfo], Error> = await withCheckedContinuation { cont in
            V2TIMManager.shared.getJoinedCommunityList(succ: { infos in
                cont.resume(returning: .success(infos))
            }, fail: { code, desc in
                cont.resume(returning: .failure(LinLanImError.apiFailed(code, desc)))
            })
        }
        let communities: [V2TIMGroupInfo]
        let isComplete: Bool
        switch communityResult {
        case .success(let infos):
            communities = infos
            isComplete = true
        case .failure:
            communities = []
            isComplete = false
        }
        for g in communities {
            guard let gid = g.groupID?.trimmingCharacters(in: .whitespacesAndNewlines), !gid.isEmpty else { continue }
            let name = (g.groupName?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? gid
            map[gid] = LinLanGroupItem(groupId: gid, groupName: name, role: g.role)
        }

        guard currentAccountStorageScope == membershipToken.scope else { throw CancellationError() }
        let groups = map.values.sorted { $0.groupName.localizedCompare($1.groupName) == .orderedAscending }
        cacheGroupRoles(groups, scope: membershipToken.scope)
        let reconciledToken = LinLanGroupMembershipState.reconcileJoinedIfCurrent(
            groupIDs: Set(groups.map(\.groupId)),
            token: membershipToken
        )
        return LinLanJoinedGroupSnapshot(
            groups: groups,
            // 两个 SDK 查询都成功，且查询期间账号/成员关系未变化，才能当权威快照。
            isComplete: isComplete && reconciledToken != nil,
            membershipToken: reconciledToken ?? membershipToken
        )
    }

    private static func conversationGroupID(_ conversation: V2TIMConversation) -> String? {
        if let id = conversation.groupID?.trimmingCharacters(in: .whitespacesAndNewlines), !id.isEmpty {
            return id
        }
        let cid = conversation.conversationID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard cid.hasPrefix("group_") else { return nil }
        let id = String(cid.dropFirst("group_".count)).trimmingCharacters(in: .whitespacesAndNewlines)
        return id.isEmpty ? nil : id
    }

    // MARK: - Private helpers

    private static func getConversationPage(nextSeq: UInt, count: Int32) async throws -> (
        list: [V2TIMConversation],
        nextSeq: UInt,
        isFinished: Bool
    ) {
        try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getConversationList(nextSeq: nextSeq, count: count, succ: { list, seq, finished in
                cont.resume(returning: (list, seq, finished))
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    private static func getConversation(conversationID: String) async throws -> V2TIMConversation {
        try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getConversation(conversationID: conversationID, succ: { conv in
                cont.resume(returning: conv)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }
}

private actor LinLanFriendApplicationLock {
    private var keys = Set<String>()
    func begin(_ key: String) -> Bool { keys.insert(key).inserted }
    func end(_ key: String) { keys.remove(key) }
}
