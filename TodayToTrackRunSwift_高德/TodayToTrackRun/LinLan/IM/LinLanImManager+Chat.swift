//
//  LinLanImManager+Chat.swift
//  TodayToTrackRun
//

import Foundation
import ImSDK_Plus_Swift

extension Notification.Name {
    static let linLanDidReadMessages = Notification.Name("linLanDidReadMessages")
    static let linLanDidRecvMessage = Notification.Name("linLanDidRecvMessage")
    static let linLanDidRevokeMessage = Notification.Name("linLanDidRevokeMessage")
    /// 全员禁言 / 个人禁言变化；object 为 groupID
    static let linLanGroupSendPolicyChanged = Notification.Name("linLanGroupSendPolicyChanged")
    /// 群成员关系变化；object 为 groupID。列表页收到后刷新，但不代表当前用户离群。
    static let linLanGroupMembershipChanged = Notification.Name("linLanGroupMembershipChanged")
    /// 当前登录用户已退出、被踢、群解散或回收；object 为 groupID。
    static let linLanCurrentUserRemovedFromGroup = Notification.Name("linLanCurrentUserRemovedFromGroup")
    /// 腾讯 IM 群消息置顶变化；object 为 groupID。
    static let linLanGroupPinnedMessagesChanged = Notification.Name("linLanGroupPinnedMessagesChanged")
    /// 语音播放状态变化；userInfo["msgId"] 为 String?（nil 表示已停止）
    static let linLanSoundPlaybackDidChange = Notification.Name("linLanSoundPlaybackDidChange")
}

struct LinLanGroupMemberItem {
    let userId: String
    var displayName: String
    /// 群昵称（可能为空；勿与个人昵称混用）
    var nameCard: String = ""
    var role: UInt
    var faceUrl: String? = nil
    var muteUntil: UInt = 0

    var isMuted: Bool {
        LinLanImManager.isMuteUntilActive(muteUntil)
    }

    var isOwner: Bool {
        role == LinLanImManager.roleOwner || role == 400
    }

    var isAdmin: Bool {
        role == LinLanImManager.roleAdmin || role == 300
    }
}

private final class LinLanMsgListenerBridge: NSObject, V2TIMAdvancedMsgListener {
    func onRecvC2CReadReceipt(receiptList: [V2TIMMessageReceipt]) {
        NotificationCenter.default.post(name: .linLanDidReadMessages, object: receiptList, userInfo: ["legacy": true])
    }
    func onRecvMessageReadReceipts(receiptList: [V2TIMMessageReceipt]) {
        NotificationCenter.default.post(name: .linLanDidReadMessages, object: receiptList)
    }

    func onRecvNewMessage(msg: V2TIMMessage) {
        NotificationCenter.default.post(name: .linLanDidRecvMessage, object: msg)
        if msg.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS,
           let gid = msg.groupID?.trimmingCharacters(in: .whitespacesAndNewlines),
           !gid.isEmpty {
            NotificationCenter.default.post(name: .linLanGroupSendPolicyChanged, object: gid)
        }
    }

    func onRecvMessageRevoked(msgID: String, operateUser: V2TIMUserInfo, reason: String?) {
        LinLanImManager.receivedRecall(msgID, actor: operateUser.userID)
    }

    func onRecvMessageRevoked(msgID: String) {
        LinLanImManager.receivedRecall(msgID)
    }

    func onGroupMessagePinned(
        groupID: String,
        message: V2TIMMessage,
        isPinned: Bool,
        opUser: V2TIMGroupMemberInfo
    ) {
        LinLanImManager.notifyGroupPinsChanged(groupID)
    }
}

private final class LinLanGroupListenerBridge: NSObject, V2TIMGroupListener {
    func onGroupInfoChanged(groupID: String, changeInfoList: [V2TIMGroupChangeInfo]) {
        notifyMute(groupID)
    }

    func onMemberInfoChanged(groupID: String, changeInfoList: [V2TIMGroupMemberChangeInfo]) {
        notifyMute(groupID)
    }

    func onMemberEnter(groupID: String, memberList: [V2TIMGroupMemberInfo]) {
        LinLanGroupMemberCache.invalidate(groupId: groupID)
        if memberList.contains(where: { $0.userID == LinLanImManager.loginUser }) {
            LinLanGroupMembershipState.reconcileJoined(groupIDs: Set([groupID]))
        }
        notifyMembership(groupID)
    }

    func onMemberInvited(
        groupID: String,
        opUser: V2TIMGroupMemberInfo,
        memberList: [V2TIMGroupMemberInfo]
    ) {
        LinLanGroupMemberCache.invalidate(groupId: groupID)
        if memberList.contains(where: { $0.userID == LinLanImManager.loginUser }) {
            LinLanGroupMembershipState.reconcileJoined(groupIDs: Set([groupID]))
        }
        notifyMembership(groupID)
    }

    func onMemberLeave(groupID: String, member: V2TIMGroupMemberInfo) {
        LinLanGroupMemberCache.invalidate(groupId: groupID)
        if member.userID == LinLanImManager.loginUser {
            LinLanImManager.handleCurrentUserRemovedFromGroup(groupID)
        } else {
            notifyMembership(groupID)
        }
    }

    func onMemberKicked(
        groupID: String,
        opUser: V2TIMGroupMemberInfo,
        memberList: [V2TIMGroupMemberInfo]
    ) {
        LinLanGroupMemberCache.invalidate(groupId: groupID)
        if memberList.contains(where: { $0.userID == LinLanImManager.loginUser }) {
            LinLanImManager.handleCurrentUserRemovedFromGroup(groupID)
        } else {
            // 其他成员被移除只刷新成员数据，不让当前用户退出群聊。
            notifyMembership(groupID)
        }
    }

    func onQuitFromGroup(groupID: String) {
        LinLanImManager.handleCurrentUserRemovedFromGroup(groupID)
    }

    func onGroupDismissed(groupID: String, opUser: V2TIMGroupMemberInfo) {
        LinLanImManager.handleCurrentUserRemovedFromGroup(groupID)
    }

    func onGroupRecycled(groupID: String, opUser: V2TIMGroupMemberInfo) {
        LinLanImManager.handleCurrentUserRemovedFromGroup(groupID)
    }

    func onGroupCreated(groupID: String) {
        LinLanGroupMembershipState.reconcileJoined(groupIDs: Set([groupID]))
        notifyMembership(groupID)
    }

    func onAllGroupMembersMuted(groupID: String, isMute: Bool) {
        notifyMute(groupID)
    }

    private func notifyMute(_ groupID: String) {
        let gid = groupID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { return }
        NotificationCenter.default.post(name: .linLanGroupSendPolicyChanged, object: gid)
    }

    private func notifyMembership(_ groupID: String) {
        let gid = groupID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { return }
        DispatchQueue.main.async {
            NotificationCenter.default.post(name: .linLanGroupMembershipChanged, object: gid)
        }
    }
}

extension LinLanImManager {
    static let atAllDisplay = "全体成员"
    static let atAllTag = kImSDK_MesssageAtALL
    static let historyPageSize = 30

    static func notifyGroupPinsChanged(_ groupID: String) {
        let gid = groupID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { return }
        DispatchQueue.main.async {
            NotificationCenter.default.post(name: .linLanGroupPinnedMessagesChanged, object: gid)
        }
    }

    static func handleCurrentUserRemovedFromGroup(_ groupID: String) {
        let gid = groupID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { return }
        LinLanGroupMembershipState.markRemoved(groupIDs: Set([gid]))
        LinLanGroupMemberCache.invalidate(groupId: gid)
        DispatchQueue.main.async {
            NotificationCenter.default.post(name: .linLanCurrentUserRemovedFromGroup, object: gid)
            NotificationCenter.default.post(name: .linLanGroupMembershipChanged, object: gid)
        }
        Task {
            try? await deleteConversation(conversationID: conversationId(userID: nil, groupID: gid))
        }
    }

    static func observeRecalls(_ messages: [V2TIMMessage]) -> [V2TIMMessage] {
        guard let scope = recallScope() else { return messages }
        for message in messages where message.status == .V2TIM_MSG_STATUS_LOCAL_REVOKED {
            recalled.receive(scope, id: message.msgID, actor: message.revokerInfo?.userID ?? "", sender: message.sender ?? "", group: message.groupID ?? "")
        }
        return messages
    }
    static func receivedRecall(_ id: String, actor: String = "") {
        guard let scope = recallScope(), recalled.receive(scope, id: id, actor: actor) else { return }
        NotificationCenter.default.post(name: .linLanDidRevokeMessage, object: id)
        Task {
            guard recalled.isCurrent(scope) else { return }
            let found = try? await findMessages([id])
            guard recalled.isCurrent(scope) else { return }
            if let message = found?.first {
                recalled.receive(scope, id: id, actor: actor.isEmpty ? message.revokerInfo?.userID ?? "" : actor, sender: message.sender ?? "", group: message.groupID ?? "")
            }
            NotificationCenter.default.post(name: .linLanDidRevokeMessage, object: id)
        }
    }
    static func findMessages(_ ids: [String]) async throws -> [V2TIMMessage] {
        let scope = recallScope()
        let messages: [V2TIMMessage] = try await withCheckedThrowingContinuation { continuation in
            V2TIMManager.shared.findMessages(messageIDList: ids, succ: { continuation.resume(returning: $0) },
                fail: { continuation.resume(throwing: LinLanImError.apiFailed($0, $1)) })
        }
        guard recalled.isCurrent(scope) else { throw CancellationError() }
        return observeRecalls(messages)
    }

    private static let msgBridge = LinLanMsgListenerBridge()
    private static let groupBridge = LinLanGroupListenerBridge()
    private static var listenersReady = false

    static func registerMessageListeners() {
        lock.lock()
        defer { lock.unlock() }
        guard !listenersReady else { return }
        V2TIMManager.shared.addAdvancedMsgListener(listener: msgBridge)
        V2TIMManager.shared.addGroupListener(listener: groupBridge)
        listenersReady = true
    }

    static func conversationId(userID: String?, groupID: String?) -> String {
        if let g = groupID, !g.isEmpty { return "group_\(g)" }
        if let u = userID, !u.isEmpty { return "c2c_\(u)" }
        return ""
    }

    static func conversationId(of msg: V2TIMMessage) -> String {
        if let g = msg.groupID, !g.isEmpty { return "group_\(g)" }
        let peer: String? = msg.isSelf
            ? (msg.userID ?? msg.sender)
            : (msg.sender ?? msg.userID)
        guard let p = peer?.trimmingCharacters(in: .whitespacesAndNewlines), !p.isEmpty else {
            return ""
        }
        return "c2c_\(p)"
    }

    /// 会话免打扰 / 不接收消息
    static func isMessageMuted(_ msg: V2TIMMessage) async -> Bool {
        let cid = conversationId(of: msg)
        guard !cid.isEmpty, cid != "c2c_", cid != "group_" else { return false }
        return await withCheckedContinuation { cont in
            V2TIMManager.shared.getConversation(conversationID: cid, succ: { conv in
                let opt = conv.recvOpt
                let muted = opt == .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
                    || opt == .V2TIM_NOT_RECEIVE_MESSAGE
                cont.resume(returning: muted)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
    }

    /// 首次 lastMsg=nil；加载更早传当前列表最旧一条
    static func fetchHistory(
        conversationId: String,
        count: Int = historyPageSize,
        lastMsg: V2TIMMessage? = nil
    ) async throws -> [V2TIMMessage] {
        let option = V2TIMMessageListGetOption()
        option.getType = .V2TIM_GET_CLOUD_OLDER_MSG
        option.count = UInt(count)
        option.lastMsg = lastMsg
        if conversationId.hasPrefix("group_") {
            option.groupID = String(conversationId.dropFirst("group_".count))
        } else {
            option.userID = String(conversationId.dropFirst("c2c_".count))
        }
        let scope = recallScope()
        let messages: [V2TIMMessage] = try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getHistoryMessageList(option: option, succ: { list in
                cont.resume(returning: list)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
        guard recalled.isCurrent(scope) else { throw CancellationError() }
        return observeRecalls(messages)
    }

    static func markChatRead(userID: String?, groupID: String?, messages: [V2TIMMessage]) {
        if let userID, !userID.isEmpty {
            V2TIMManager.shared.markC2CMessageAsRead(userID: userID, succ: nil, fail: nil)
        } else if let groupID, !groupID.isEmpty {
            V2TIMManager.shared.markGroupMessageAsRead(groupID: groupID, succ: nil, fail: nil)
        }
        let targets = messages.filter { !$0.isSelf && $0.needReadReceipt && !looksLikeCommunity($0.groupID) }
        for start in stride(from: 0, to: targets.count, by: 30) {
            V2TIMManager.shared.sendMessageReadReceipts(messageList: Array(targets[start..<min(start + 30, targets.count)]), succ: nil, fail: nil)
        }
    }

    static func refreshReadReceipts(_ messages: [V2TIMMessage]) {
        let targets = messages.filter { $0.isSelf && $0.needReadReceipt && ($0.groupID?.isEmpty ?? true) }
        for start in stride(from: 0, to: targets.count, by: 30) {
            V2TIMManager.shared.getMessageReadReceipts(messageList: Array(targets[start..<min(start + 30, targets.count)]), succ: { receipts in
                NotificationCenter.default.post(name: .linLanDidReadMessages, object: receipts)
            }, fail: { _, _ in })
        }
    }

    static func sendTextMessage(
        text: String,
        receiver: String?,
        groupId: String?,
        cloudCustomData: Data? = nil,
        atUserIds: [String] = []
    ) async throws -> V2TIMMessage {
        guard var msg = V2TIMManager.shared.createTextMessage(text: text) else {
            throw LinLanImError.apiFailed(-1, "创建文本消息失败")
        }
        if let groupId, !groupId.isEmpty, !atUserIds.isEmpty {
            if let atMsg = V2TIMManager.shared.createAtSignedGroupMessage(message: msg, atUserList: atUserIds) {
                msg = atMsg
            }
        }
        if let cloudCustomData {
            msg.cloudCustomData = cloudCustomData
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    static func sendImageMessage(path: String, receiver: String?, groupId: String?) async throws -> V2TIMMessage {
        guard let msg = V2TIMManager.shared.createImageMessage(imagePath: path) else {
            throw LinLanImError.apiFailed(-1, "创建图片消息失败")
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    static func sendSoundMessage(
        path: String,
        durationSec: Int,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        guard let msg = V2TIMManager.shared.createSoundMessage(audioFilePath: path, duration: durationSec) else {
            throw LinLanImError.apiFailed(-1, "创建语音消息失败")
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    static func sendVideoMessage(
        videoPath: String,
        type: String,
        durationSec: Int,
        snapshotPath: String,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        if attachmentFileSize(videoPath) > 100 * 1024 * 1024 {
            return try await sendLargeAttachment(path: videoPath, fileName: URL(fileURLWithPath: videoPath).lastPathComponent, receiver: receiver, groupId: groupId, isVideo: true)
        }
        guard let msg = V2TIMManager.shared.createVideoMessage(
            videoFilePath: videoPath,
            type: type,
            duration: durationSec,
            snapshotPath: snapshotPath
        ) else {
            throw LinLanImError.apiFailed(-1, "创建视频消息失败")
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    static func sendFileMessage(
        path: String,
        fileName: String,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        if attachmentFileSize(path) > 100 * 1024 * 1024 {
            return try await sendLargeAttachment(path: path, fileName: fileName, receiver: receiver, groupId: groupId, isVideo: false)
        }
        guard let msg = V2TIMManager.shared.createFileMessage(filePath: path, fileName: fileName) else {
            throw LinLanImError.apiFailed(-1, "创建文件消息失败")
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    private static func attachmentFileSize(_ path: String) -> Int64 {
        (try? FileManager.default.attributesOfItem(atPath: path)[.size] as? NSNumber)?.int64Value ?? 0
    }

    private static func sendLargeAttachment(path: String, fileName: String, receiver: String?, groupId: String?, isVideo: Bool) async throws -> V2TIMMessage {
        let owner = loginUser
        let conversation = (groupId?.isEmpty == false) ? "group_\(groupId!)" : "c2c_\(receiver ?? "")"
        let type = isVideo ? "video/mp4" : "application/octet-stream"
        let result = try await LinLanAppContainer.apiClient().uploadAttachmentMultipart(path: path, fileName: fileName, conversation: conversation, contentType: type, isVideo: isVideo)
        guard loginUser == owner, LinLanAppContainer.sessionStore().currentSession().userId == owner else { throw LinLanForwardFailure.notDispatched("账号已切换，附件未发送") }
        let body: [String: Any] = ["file_id": result["file_id"] ?? "", "name": result["file_name"] ?? fileName, "size": attachmentFileSize(path), "is_video": isVideo, "content_type": result["content_type"] ?? type]
        let data = try JSONSerialization.data(withJSONObject: body)
        let marker = "文件已上传，点击卡片预览或下载\n\(fileName)\nCN_ATTACHMENT_V1:" + (String(data: data, encoding: .utf8) ?? "")
        return try await sendTextMessage(text: marker, receiver: receiver, groupId: groupId)
    }

    static func sendRtcCallRecord(
        peerUserId: String,
        payload: LinLanRtcSignalingPayload
    ) async throws -> V2TIMMessage {
        guard let json = payload.toJson(),
              let data = json.data(using: .utf8) else {
            throw LinLanImError.apiFailed(-1, "创建通话记录失败")
        }
        let text = payload.displayText(isSelf: true)
        guard let msg = V2TIMManager.shared.createCustomMessage(data: data, desc: text, ext: text) else {
            throw LinLanImError.apiFailed(-1, "创建通话记录失败")
        }
        let sent = try await sendCreated(msg, receiver: peerUserId, groupId: nil)
        NotificationCenter.default.post(name: .linLanDidRecvMessage, object: sent)
        return sent
    }

    static func shouldHideRtcSignalingInChat(_ msg: V2TIMMessage) -> Bool {
        guard let payload = LinLanRtcSignalingPayload.parseFromMessage(msg) else { return false }
        return LinLanRtcSignalingPayload.shouldHideInChat(payload)
    }

    /// 被移出成员的具体身份只对群主/管理员展示。
    /// 普通成员以及角色尚未确认时隐藏 KICKED；其他群通知保持不变。
    static func shouldHideGroupRemovalTip(_ msg: V2TIMMessage, viewerRole: UInt? = nil) -> Bool {
        guard msg.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS,
              let type = msg.groupTipsElem?.type else { return false }
        return shouldHideGroupTipType(type, viewerRole: viewerRole)
    }

    static func shouldHideGroupTipType(_ type: V2TIMGroupTipsType, viewerRole: UInt?) -> Bool {
        guard type == .V2TIM_GROUP_TIPS_TYPE_KICKED else { return false }
        guard let viewerRole else { return true }
        return !isOwnerOrAdmin(role: viewerRole)
    }

    static func fetchPinnedGroupMessages(groupId: String) async throws -> [V2TIMMessage] {
        let gid = groupId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { return [] }
        return try await withCheckedThrowingContinuation { cont in
            V2TIMManager.shared.getPinnedGroupMessageList(groupID: gid, succ: { messages in
                cont.resume(returning: messages)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    static func setGroupMessagePinned(
        groupId: String,
        message: V2TIMMessage,
        pinned: Bool
    ) async throws {
        let gid = groupId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !gid.isEmpty else { throw LinLanImError.apiFailed(-1, "群组不存在") }
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            V2TIMManager.shared.pinGroupMessage(
                groupID: gid,
                message: message,
                isPinned: pinned,
                succ: { cont.resume() },
                fail: { code, desc in
                    cont.resume(throwing: LinLanImError.apiFailed(code, desc))
                }
            )
        }
    }

    /// 群主/管理员撤回他人：消息直接消失，不展示 tip
    static func shouldSilentlyHideRevokedMessage(_ msg: V2TIMMessage) -> Bool {
        recalled.shouldHide(recallScope(), id: msg.msgID, group: msg.groupID ?? "", sender: msg.sender ?? "",
            actor: msg.revokerInfo?.userID ?? "", sdkRevoked: msg.status == .V2TIM_MSG_STATUS_LOCAL_REVOKED)
    }
    static func isKnownRecalledMessage(_ id: String) -> Bool {
        recalled.isRevoked(recallScope(), id: id, sdkRevoked: false)
    }
    static func isRevokedMessage(_ msg: V2TIMMessage) -> Bool {
        recalled.isRevoked(recallScope(), id: msg.msgID, sdkRevoked: msg.status == .V2TIM_MSG_STATUS_LOCAL_REVOKED)
    }

    static func revokedTipText(_ msg: V2TIMMessage) -> String {
        msg.isSelf ? "你撤回了一条消息" : "对方撤回了一条消息"
    }

    static func canRevokeMessage(
        _ msg: V2TIMMessage,
        selfGroupRole: UInt = 0,
        senderMemberRole: UInt = 0
    ) -> Bool {
        if isRevokedMessage(msg) { return false }
        if msg.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS { return false }
        if msg.status != .V2TIM_MSG_STATUS_SEND_SUCC { return false }

        if msg.isSelf {
            return Date().timeIntervalSince(msg.timestamp) <= 120
        }

        let groupId = (msg.groupID ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !groupId.isEmpty, isOwnerOrAdmin(role: selfGroupRole) else { return false }
        let selfId = loginUser
        guard !selfId.isEmpty else { return false }
        return canRevokeGroupMemberMessage(
            senderUserId: (msg.sender ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
            senderRole: senderMemberRole,
            selfUserId: selfId,
            selfRole: selfGroupRole
        )
    }

    static func canRevokeGroupMemberMessage(
        senderUserId: String,
        senderRole: UInt,
        selfUserId: String,
        selfRole: UInt
    ) -> Bool {
        if senderUserId.isEmpty || senderUserId == selfUserId { return false }
        if selfRole == roleOwner || selfRole == 400 { return true }
        if selfRole == roleAdmin || selfRole == 300 {
            return senderRole == roleMember || senderRole == 0 || senderRole == 200
        }
        return false
    }

    /// 自己撤回走 SDK（≤2 分钟）；群主/管理员撤回他人走 REST
    static func revokeMessage(_ msg: V2TIMMessage) async throws {
        guard let scope = recallScope() else { throw LinLanImError.apiFailed(-1, "聊天服务尚未登录") }
        let groupId = (msg.groupID ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let kind: LinLanRecalledMessageState.Kind = !msg.isSelf && !groupId.isEmpty ? .moderator : .own
        let intent = try recalled.begin(scope, id: msg.msgID, kind: kind, sender: msg.sender ?? "", group: groupId, actor: loginUser)
        do {
            if kind == .moderator {
                guard msg.seq > 0 else { throw LinLanImError.apiFailed(-1, "撤回失败：消息序号无效") }
                try await LinLanAppContainer.apiClient().revokeGroupMessage(groupId: groupId, msgSeq: msg.seq)
            } else {
                try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
                    V2TIMManager.shared.revokeMessage(msg: msg, succ: { cont.resume() },
                        fail: { cont.resume(throwing: LinLanImError.apiFailed($0, $1)) })
                }
            }
            guard recalled.confirm(intent) else { throw CancellationError() }
            NotificationCenter.default.post(name: .linLanDidRevokeMessage, object: msg.msgID)
        } catch {
            recalled.cancel(intent)
            throw error
        }
    }

    static func canForwardMessage(_ msg: V2TIMMessage) -> Bool {
        msg.status == .V2TIM_MSG_STATUS_SEND_SUCC
            && !isRevokedMessage(msg)
            && msg.elemType != .V2TIM_ELEM_TYPE_GROUP_TIPS
    }

    static func forwardMessage(
        _ source: V2TIMMessage,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        guard let msg = V2TIMManager.shared.createForwardMessage(message: source) else {
            throw LinLanImError.apiFailed(-1, "创建转发消息失败")
        }
        return try await sendCreated(msg, receiver: receiver, groupId: groupId)
    }

    static func forwardMessagesOneByOne(
        _ sources: [V2TIMMessage],
        receiver: String?,
        groupId: String?
    ) async throws {
        let sorted = sources.filter(canForwardMessage).sorted {
            if $0.timestamp != $1.timestamp { return $0.timestamp < $1.timestamp }
            return $0.seq < $1.seq
        }
        guard !sorted.isEmpty else { throw LinLanImError.apiFailed(-1, "没有可转发的消息") }
        for source in sorted {
            _ = try await forwardMessage(source, receiver: receiver, groupId: groupId)
        }
    }

    static func forwardMessagesMerged(
        _ sources: [V2TIMMessage],
        title: String,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        let sorted = sources.filter(canForwardMessage).sorted {
            if $0.timestamp != $1.timestamp { return $0.timestamp < $1.timestamp }
            return $0.seq < $1.seq
        }.prefix(100)
        guard !sorted.isEmpty else { throw LinLanImError.apiFailed(-1, "没有可转发的消息") }
        let list = Array(sorted)
        let abstracts = list.prefix(5).map { msg in
            "\(LinLanQuoteHelper.senderName(msg))：\(messagePreview(msg))"
        }
        // compatibleText 不要用「自定义」语义，接收端旧版会展示这段文案
        guard let merger = V2TIMManager.shared.createMergerMessage(
            messageList: list,
            title: title.isEmpty ? "聊天记录" : title,
            abstractList: Array(abstracts),
            compatibleText: "[聊天记录]"
        ) else {
            throw LinLanImError.apiFailed(-1, "创建合并消息失败")
        }
        return try await sendCreated(merger, receiver: receiver, groupId: groupId)
    }

    /// 分页拉取，带上限，避免大群一次灌满内存
    static func fetchGroupMembers(
        groupId: String,
        maxCount: Int = 400
    ) async throws -> (list: [LinLanGroupMemberItem], truncated: Bool) {
        var all: [LinLanGroupMemberItem] = []
        var nextSeq: UInt = 0
        var guardCount = 0
        var truncated = false
        repeat {
            let page: (members: [V2TIMGroupMemberFullInfo], next: UInt) = try await withCheckedThrowingContinuation { cont in
                V2TIMManager.shared.getGroupMemberList(
                    groupID: groupId,
                    filter: .V2TIM_GROUP_MEMBER_FILTER_ALL,
                    nextSeq: nextSeq,
                    succ: { next, members in
                        cont.resume(returning: (members, next))
                    },
                    fail: { code, desc in
                        cont.resume(throwing: LinLanImError.apiFailed(code, desc))
                    }
                )
            }
            for m in page.members {
                if all.count >= maxCount {
                    truncated = true
                    break
                }
                let card = m.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                let name: String = {
                    if !card.isEmpty { return card }
                    if let n = m.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !n.isEmpty { return n }
                    return m.userID
                }()
                all.append(
                    LinLanGroupMemberItem(
                        userId: m.userID,
                        displayName: name,
                        nameCard: card,
                        role: m.role,
                        faceUrl: m.faceURL,
                        muteUntil: m.muteUntil
                    )
                )
            }
            if truncated { break }
            nextSeq = page.next
            guardCount += 1
        } while nextSeq != 0 && guardCount < 40
        if nextSeq != 0 { truncated = true }
        return (all, truncated)
    }

    static func fetchAllGroupMembers(groupId: String) async throws -> [LinLanGroupMemberItem] {
        try await LinLanGroupMemberCache.members(
            groupId: groupId,
            maxCount: LinLanGroupMemberCache.listLimit
        )
    }

    /// 群 tip 文案：进退群、踢人/拉人、公告、群名、加群方式、禁言等
    static func groupTipsTipText(_ msg: V2TIMMessage) -> String {
        guard msg.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS,
              let tip = msg.groupTipsElem else { return "" }

        func memberName(_ m: V2TIMGroupMemberInfo?) -> String {
            guard let m else { return "成员" }
            if let c = m.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines), !c.isEmpty { return c }
            if let n = m.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !n.isEmpty { return n }
            let id = m.userID.trimmingCharacters(in: .whitespacesAndNewlines)
            return id.isEmpty ? "成员" : id
        }

        let op = memberName(tip.opMember)
        let list = tip.memberList ?? []
        let names = list.prefix(5).map { memberName($0) }.filter { !$0.isEmpty }
        let more = list.count > 5 ? " 等\(list.count)人" : ""
        let joined = names.joined(separator: "、")

        switch tip.type {
        case .V2TIM_GROUP_TIPS_TYPE_JOIN:
            return joined.isEmpty ? "新成员加入了群聊" : "\(joined)\(more) 加入了群聊"
        case .V2TIM_GROUP_TIPS_TYPE_INVITE:
            return joined.isEmpty
                ? "\(op) 邀请成员加入群聊"
                : "\(op) 邀请 \(joined)\(more) 加入群聊"
        case .V2TIM_GROUP_TIPS_TYPE_QUIT:
            return "\(op) 退出了群聊"
        case .V2TIM_GROUP_TIPS_TYPE_KICKED:
            return joined.isEmpty
                ? "\(op) 将成员移出群聊"
                : "\(op) 将 \(joined)\(more) 移出群聊"
        case .V2TIM_GROUP_TIPS_TYPE_SET_ADMIN:
            return joined.isEmpty ? "\(op) 设置了管理员" : "\(op) 将 \(joined)\(more) 设为管理员"
        case .V2TIM_GROUP_TIPS_TYPE_CANCEL_ADMIN:
            return joined.isEmpty ? "\(op) 取消了管理员" : "\(op) 取消了 \(joined)\(more) 的管理员"
        case .V2TIM_GROUP_TIPS_TYPE_GROUP_INFO_CHANGE:
            return groupInfoChangeTipText(op: op, changes: tip.groupChangeInfoList ?? [])
        case .V2TIM_GROUP_TIPS_TYPE_MEMBER_INFO_CHANGE:
            return memberInfoChangeTipText(op: op, changes: tip.memberChangeInfoList ?? [], members: list)
        default:
            return ""
        }
    }

    private static func groupInfoChangeTipText(op: String, changes: [V2TIMGroupChangeInfo]) -> String {
        guard !changes.isEmpty else { return "\(op) 修改了群资料" }
        let parts: [String] = changes.compactMap { change in
            switch change.type {
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_NAME:
                let name = (change.value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
                return name.isEmpty ? "\(op) 修改了群名称" : "\(op) 修改群名称为「\(name)」"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_NOTIFICATION:
                let notice = (change.value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
                if notice.isEmpty { return "\(op) 清空了群公告" }
                let short = notice.count > 40 ? String(notice.prefix(40)) + "…" : notice
                return "\(op) 更新了群公告：\(short)"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_INTRODUCTION:
                return "\(op) 修改了群简介"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_FACE:
                return "\(op) 修改了群头像"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_OWNER:
                let owner = (change.value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
                return owner.isEmpty ? "\(op) 转让了群主" : "群主已转让给 \(owner)"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_SHUT_UP_ALL:
                return change.boolValue ? "\(op) 开启了全员禁言" : "\(op) 关闭了全员禁言"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_GROUP_ADD_OPT:
                let label = groupAddOptLabel(V2TIMGroupAddOpt(rawValue: change.intValue) ?? .V2TIM_GROUP_ADD_AUTH)
                return "\(op) 将主动加群方式改为「\(label)」"
            case .V2TIM_GROUP_INFO_CHANGE_TYPE_GROUP_APPROVE_OPT:
                let label = groupAddOptLabel(V2TIMGroupAddOpt(rawValue: change.intValue) ?? .V2TIM_GROUP_ADD_AUTH)
                return "\(op) 将邀请进群方式改为「\(label)」"
            default:
                return nil
            }
        }
        if parts.isEmpty { return "\(op) 修改了群资料" }
        return parts.joined(separator: "；")
    }

    private static func memberInfoChangeTipText(
        op: String,
        changes: [V2TIMGroupMemberChangeInfo],
        members: [V2TIMGroupMemberInfo]
    ) -> String {
        if !changes.isEmpty {
            let parts = changes.map { change -> String in
                let uid = (change.userID ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
                let target = members.first(where: { $0.userID == uid }).map { m -> String in
                    if let c = m.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines), !c.isEmpty { return c }
                    if let n = m.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !n.isEmpty { return n }
                    return uid.isEmpty ? "成员" : uid
                } ?? (uid.isEmpty ? "成员" : uid)
                if change.muteTime > 0 {
                    return "\(op) 禁言了 \(target)"
                }
                return "\(op) 解除了 \(target) 的禁言"
            }
            return parts.joined(separator: "；")
        }
        // 部分端修改群昵称会带成员列表、无 mute 变更
        if let m = members.first {
            let name: String = {
                if let c = m.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines), !c.isEmpty { return c }
                if let n = m.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !n.isEmpty { return n }
                return m.userID
            }()
            return "\(name) 修改了群昵称"
        }
        return "\(op) 修改了群成员资料"
    }

    static func canAtAll(members: [LinLanGroupMemberItem]) -> Bool {
        let selfId = loginUser
        guard let me = members.first(where: { $0.userId == selfId }) else { return false }
        let superRole = UInt(V2TIMGroupMemberRole.V2TIM_GROUP_MEMBER_ROLE_SUPER.rawValue)
        let adminRole = UInt(V2TIMGroupMemberRole.V2TIM_GROUP_MEMBER_ROLE_ADMIN.rawValue)
        return me.role == superRole || me.role == adminRole || me.role == 400 || me.role == 300
    }

    static func belongsToConversation(_ msg: V2TIMMessage, userID: String?, groupID: String?) -> Bool {
        if let g = groupID, !g.isEmpty {
            return isSameGroup(msg.groupID, g)
        }
        if let u = userID, !u.isEmpty {
            let peer = msg.isSelf ? (msg.userID ?? "") : (msg.sender ?? msg.userID ?? "")
            return peer == u || (msg.userID ?? "") == u
        }
        return false
    }

    /// Community 群 ID 可能是 `@TGS#_@TGS#xxx` 或截断形式。
    static func isSameGroup(_ lhs: String?, _ rhs: String?) -> Bool {
        let a = lhs?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let b = rhs?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if a.isEmpty || b.isEmpty { return false }
        if a == b { return true }
        if LinLanGroupMembershipPolicy.membershipGroupID(a)
            == LinLanGroupMembershipPolicy.membershipGroupID(b) { return true }
        return a.hasSuffix(b) || b.hasSuffix(a)
    }

    /// TIM 可能下发解禁时间戳，也可能下发剩余秒数（如 2592000）。
    static func isMuteUntilActive(_ muteUntil: UInt) -> Bool {
        if muteUntil == 0 { return false }
        let now = UInt(Date().timeIntervalSince1970)
        if muteUntil > 1_000_000_000 {
            return muteUntil > now
        }
        return true
    }

    private static func looksLikeCommunity(_ groupId: String?) -> Bool {
        let id = groupId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.hasPrefix("@TGS#_") || id.contains("@TOPIC#_")
    }

    private static func sendCreated(
        _ message: V2TIMMessage,
        receiver: String?,
        groupId: String?
    ) async throws -> V2TIMMessage {
        let owner = loginUser
        guard !owner.isEmpty else { throw LinLanForwardFailure.notDispatched("聊天服务尚未登录") }
        if let groupId, !groupId.isEmpty {
            let permission = await resolveGroupSendPermission(groupId: groupId)
            guard permission.allowsSend else { throw LinLanForwardFailure.notDispatched(permission.blockedHint) }
        }
        guard loginUser == owner else { throw LinLanForwardFailure.notDispatched("账号已切换") }
        try Task.checkCancellation()
        message.needReadReceipt = groupId == nil || groupId?.isEmpty == true || !looksLikeCommunity(groupId)
        let isGroup = !(groupId ?? "").isEmpty
        let peerID = isGroup ? (groupId ?? "") : (receiver ?? "")
        let title: String = {
            let nick = LinLanAppContainer.appPrefs().cachedNickname
                .trimmingCharacters(in: .whitespacesAndNewlines)
            return nick.isEmpty ? LinLanAppConstants.appName : nick
        }()
        let push = LinLanPushService.makeOfflinePushInfo(
            title: title,
            desc: messagePreview(message),
            peerID: peerID,
            isGroup: isGroup,
            senderID: loginUser
        )
        return try await withCheckedThrowingContinuation { cont in
            _ = V2TIMManager.shared.sendMessage(
                message: message,
                receiver: receiver,
                groupID: groupId,
                priority: .V2TIM_PRIORITY_NORMAL,
                onlineUserOnly: false,
                offlinePushInfo: push,
                progress: nil,
                succ: {
                    cont.resume(returning: message)
                },
                fail: { code, desc in
                    cont.resume(throwing: LinLanImError.apiFailed(code, desc))
                }
            )
        }
    }
}
