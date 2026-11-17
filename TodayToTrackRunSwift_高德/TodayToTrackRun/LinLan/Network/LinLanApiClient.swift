//
//  LinLanApiClient.swift
//  TodayToTrackRun
//
//  对照 Android LinLanApiClient 鉴权相关接口
//

import Foundation
import UIKit

final class LinLanApiClient {
    private let sessionStore: LinLanSessionStore
    private let refreshCoordinator = LinLanRefreshCoordinator()
    private let session: URLSession

    init(sessionStore: LinLanSessionStore) {
        self.sessionStore = sessionStore
        self.session = RuiHTTPDNSNetwork.session(requestTimeout: 20, resourceTimeout: 30)
    }

    func initializeRouting() async {
        await LinLanDomainPool.shared.ensureReady()
    }

    func loginWithPassword(phone: String, password: String) async throws -> LinLanAuthSession {
        try await authRequest("/login", body: [
            "phone": phone,
            "password": password,
            "device_id": sessionStore.deviceId(),
        ])
    }

    func loginWithCode(phone: String, code: String) async throws -> LinLanAuthSession {
        try await authRequest("/login", body: [
            "phone": phone,
            "code": code,
            "device_id": sessionStore.deviceId(),
        ])
    }

    func register(
        phone: String,
        code: String?,
        nickname: String,
        password: String,
        inviteCode: String?
    ) async throws -> LinLanAuthSession {
        var body: [String: Any] = [
            "phone": phone,
            "nickname": nickname,
            "password": password,
            "register_type": "phone_sms",
            "device_id": sessionStore.deviceId(),
        ]
        if let code, !code.isEmpty { body["code"] = code }
        if let inviteCode, !inviteCode.isEmpty { body["invite_code"] = inviteCode }
        return try await authRequest("/register", body: body)
    }

    func sendSms(phone: String) async throws {
        _ = try await postData("/sms/send", body: ["phone": phone], auth: false)
    }

    func resetPassword(phone: String, code: String, newPassword: String) async throws {
        _ = try await postData("/password/reset", body: [
            "phone": phone,
            "code": code,
            "new_password": newPassword,
        ], auth: false)
    }

    func updatePassword(currentPassword: String, newPassword: String) async throws {
        _ = try await postData("/password/update", body: [
            "current_password": currentPassword,
            "new_password": newPassword,
        ], auth: true)
    }

    func deleteAccount() async throws {
        _ = try await postData("/account/delete", body: [:], auth: true)
    }

    func fetchSecuritySettings() async throws -> LinLanSecuritySettings {
        let data = try await getAuthData("/security/settings")
        return LinLanSecuritySettings(
            remoteLoginAlert: bool(data, "remote_login_alert", default: true),
            loginVerification: bool(data, "login_verification", default: false)
        )
    }

    func updateSecuritySettings(
        remoteLoginAlert: Bool? = nil,
        loginVerification: Bool? = nil
    ) async throws -> LinLanSecuritySettings {
        var body: [String: Any] = [:]
        if let remoteLoginAlert { body["remote_login_alert"] = remoteLoginAlert }
        if let loginVerification { body["login_verification"] = loginVerification }
        if loginVerification == true {
            body["device_id"] = sessionStore.deviceId()
            body["device_name"] = UIDevice.current.name
            body["platform"] = "ios"
        }
        let data = try await putData("/security/settings", body: body, auth: true)
        return LinLanSecuritySettings(
            remoteLoginAlert: bool(data, "remote_login_alert", default: remoteLoginAlert ?? true),
            loginVerification: bool(data, "login_verification", default: loginVerification ?? false)
        )
    }

    func fetchLoginDevices() async throws -> [LinLanLoginDevice] {
        let data = try await getAuthData(
            "/devices",
            query: ["current_device_id": sessionStore.deviceId()]
        )
        let raw = data["items"] as? [Any] ?? []
        return raw.compactMap { item in
            guard let d = item as? [String: Any] else { return nil }
            return LinLanLoginDevice(
                id: String(describing: d["id"] ?? ""),
                deviceId: string(d, "device_id"),
                deviceName: string(d, "device_name").isEmpty ? "未知设备" : string(d, "device_name"),
                platform: string(d, "platform"),
                ip: string(d, "ip"),
                lastLoginAt: string(d, "last_login_at").isEmpty
                    ? string(d, "last_used_at") : string(d, "last_login_at"),
                isCurrent: bool(d, "is_current"),
                isOnline: bool(d, "is_online")
            )
        }
    }

    func revokeLoginDevice(deviceId: String) async throws {
        _ = try await postData("/devices/revoke", body: ["device_id": deviceId], auth: true)
    }

    func refreshSession() async throws -> LinLanAuthSession {
        try await refreshCoordinator.refresh(store: sessionStore) { current in
            let data = try await self.postData("/auth/refresh", body: [
                "refresh_token": current.refreshToken,
                "need_user_sig_refresh": true,
                "device_id": self.sessionStore.deviceId(),
            ], auth: false)
            return self.parseAuthSession(data)
        }
    }

    func logoutSession(_ captured: LinLanAuthSession, deviceID: String) async throws {
        guard !captured.refreshToken.isEmpty else { throw LinLanAPIError.message("服务端注销未确认") }
        let base = await LinLanDomainPool.shared.currentBaseURL()
        guard let url = URL(string: base + "/logout") else { throw LinLanAPIError.message("注销地址无效") }
        var request = URLRequest(url: url, timeoutInterval: 5)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONSerialization.data(withJSONObject: ["refresh_token": captured.refreshToken, "device_id": deviceID])
        _ = try await executeForData(request, allowRefresh: false)
    }

    func attachmentAccess(fileID: String) async throws -> URL {
        let body = try await getAuthData("/files/\(fileID)/access")
        guard let url = URL(string: string(body, "url")), url.scheme == "https" || url.scheme == "http" else { throw LinLanAPIError.message("文件地址无效") }
        return url
    }

    func uploadAttachmentMultipart(path: String, fileName: String, conversation: String, contentType: String, isVideo: Bool) async throws -> [String: Any] {
        let owner = sessionStore.currentSession().userId
        let generation = sessionStore.generation
        return try await LinLanMultipartUploader.upload(path: path, fileName: fileName, conversation: conversation, contentType: contentType, isVideo: isVideo,
            active: { self.sessionStore.generation == generation && self.sessionStore.currentSession().userId == owner }) { method, path, body in
                if method == "GET" { return try await self.getAuthData(path) }
                return try await self.postData(path, body: body, auth: true)
            }
    }

    func fetchProfile() async throws -> LinLanChatProfile {
        let data = try await getAuthData("/profile")
        return LinLanChatProfile.fromAPI(data)
    }

    func syncFriendOwnership(_ target: String, requestID: String, owner: String, generation: UInt64) async throws -> LinLanFriendOwnerAssignment {
        let stored=sessionStore.currentSession()
        guard stored.userId==owner,sessionStore.generation==generation else {throw LinLanAPIError.message("账号已变化")}
        let safe=target.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? ""
        let base=await LinLanDomainPool.shared.currentBaseURL()
        guard let url=URL(string:base+"/users/"+safe+"/bind-owner-from-me") else {throw LinLanAPIError.message("用户ID无效")}
        var request=URLRequest(url:url);request.httpMethod="POST"
        request.setValue("application/json",forHTTPHeaderField:"Content-Type")
        request.setValue("Bearer \(stored.token)",forHTTPHeaderField:"Authorization")
        request.httpBody=try JSONSerialization.data(withJSONObject:["request_id":requestID])
        let data=try await executeForData(request,expectedOwner:owner,expectedGeneration:generation)
        let assignment=data["assignment"] as? [String:Any] ?? [:]
        return LinLanFriendOwnerAssignment(state:string(assignment,"state").isEmpty ? "pending" : string(assignment,"state"),reasonCode:string(assignment,"reason_code"),requestId:string(assignment,"request_id"))
    }

    func fetchFriendOwnerBrief(_ target: String) async throws -> LinLanChatProfile? {
        let data=try await postData("/users/briefs",body:["im_user_ids":[target]],auth:true)
        guard let rows=data["users"] as? [[String:Any]],let peer=rows.first(where:{string($0,"im_user_id")==target}) else {return nil}
        return LinLanChatProfile.fromAPI(peer)
    }

    func updateProfile(
        nickname: String? = nil,
        avatarUrl: String? = nil,
        wananId: String? = nil,
        allowType: Int? = nil
    ) async throws {
        var body: [String: Any] = [:]
        if let nickname, !nickname.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            body["nickname"] = nickname.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        if let avatarUrl, !avatarUrl.isEmpty { body["avatar_url"] = avatarUrl }
        if let wananId {
            body["wanan_id"] = wananId.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        }
        if let allowType { body["allow_type"] = allowType }
        guard !body.isEmpty else { return }
        _ = try await postData("/profile/update", body: body, auth: true)
    }

    func bindInviteCode(_ code: String) async throws {
        _ = try await postData(
            "/invite-code/bind",
            body: ["invite_code": code.trimmingCharacters(in: .whitespacesAndNewlines)],
            auth: true
        )
    }

    /// 群主/管理员撤回他人消息（无 2 分钟限制）
    func revokeGroupMessage(groupId: String, msgSeq: UInt) async throws {
        _ = try await postData(
            "/messages/group/revoke",
            body: [
                "group_id": groupId,
                "msg_seq": msgSeq,
            ],
            auth: true
        )
    }

    func submitReport(
        targetId: String,
        targetType: String,
        targetName: String,
        category: String,
        description: String?
    ) async throws {
        var body: [String: Any] = [
            "target_id": targetId,
            "target_type": targetType,
            "target_name": targetName,
            "category": category,
        ]
        if let description {
            let d = description.trimmingCharacters(in: .whitespacesAndNewlines)
            if !d.isEmpty { body["description"] = String(d.prefix(200)) }
        }
        _ = try await postData("/reports", body: body, auth: true)
    }

    func getGroupNotice(groupId: String) async throws -> String {
        let data = try await getAuthData("/groups/notice", query: ["group_id": groupId])
        return string(data, "content")
    }

    func saveGroupNotice(groupId: String, content: String) async throws -> String {
        let data = try await postData(
            "/groups/notice",
            body: ["group_id": groupId, "content": content],
            auth: true
        )
        return string(data, "content")
    }

    func uploadGroupAvatar(groupId: String, filePath: String) async throws -> String {
        let data = try await getAuthData("/upload/avatar/presign", query: [
            "type": "group",
            "group_id": groupId,
            "biz_id": "0",
        ])
        let putUrl = string(data, "put_url")
        let avatarUrl = string(data, "avatar_url")
        guard !putUrl.isEmpty, !avatarUrl.isEmpty else {
            throw LinLanAPIError.message("获取上传地址失败")
        }
        try await putFileToCos(putUrl: putUrl, filePath: filePath, contentType: "image/jpeg")
        return avatarUrl
    }

    /// Presign + PUT COS，返回最终 avatar_url（对照 Android uploadUserAvatar）
    func uploadUserAvatar(userId: String, filePath: String) async throws -> String {
        let data = try await getAuthData("/upload/avatar/presign", query: [
            "type": "user",
            "user_id": userId,
            "biz_id": "0",
        ])
        let putUrl = string(data, "put_url")
        let avatarUrl = string(data, "avatar_url")
        guard !putUrl.isEmpty, !avatarUrl.isEmpty else {
            throw LinLanAPIError.message("获取上传地址失败")
        }
        try await putFileToCos(putUrl: putUrl, filePath: filePath, contentType: "image/jpeg")
        return avatarUrl
    }

    /// 精确优先，再模糊（对照 Android searchUsersPreferExact）
    func searchUsersPreferExact(keyword: String) async throws -> [LinLanSearchUser] {
        let exact = (try? await searchUsers(path: "/users/search-exact", keyword: keyword)) ?? []
        if !exact.isEmpty { return exact }
        return try await searchUsers(path: "/users/search", keyword: keyword)
    }

    // MARK: - 朋友圈 /community

    func fetchCommunityFeed(feed: String = "recommended") async throws -> [LinLanCommunityPost] {
        let data = try await getAuthData("/community/posts", query: ["feed": feed])
        let raw = data["items"] ?? data["posts"]
        let list: [[String: Any]] = {
            if let arr = raw as? [[String: Any]] { return arr }
            if let arr = raw as? [Any] { return arr.compactMap { $0 as? [String: Any] } }
            return []
        }()
        return list.compactMap { LinLanCommunityPost.parse($0) }
    }

    func fetchCommunityPostDetail(postID: Int) async throws -> (LinLanCommunityPost, [LinLanCommunityComment]) {
        let data = try await getAuthData("/community/posts/\(postID)")
        guard let postDict = data["post"] as? [String: Any],
              let post = LinLanCommunityPost.parse(postDict) else {
            throw LinLanAPIError.message("动态不存在")
        }
        let raw = data["comments"]
        let comments: [[String: Any]] = {
            if let arr = raw as? [[String: Any]] { return arr }
            if let arr = raw as? [Any] { return arr.compactMap { $0 as? [String: Any] } }
            return []
        }()
        return (post, comments.compactMap { LinLanCommunityComment.parse($0) })
    }

    func createCommunityPost(
        title: String,
        content: String,
        imageURLs: [String] = [],
        postType: String = "experience_share"
    ) async throws -> Int {
        let bodyContent = LinLanCommunityImagesHelper.embed(content: content, images: imageURLs)
        let data = try await postData(
            "/community/posts",
            body: [
                "post_type": postType,
                "title": title,
                "content": bodyContent
            ],
            auth: true
        )
        return int(data, "id", default: 0)
    }

    func createCommunityComment(postID: Int, content: String) async throws -> LinLanCommunityComment {
        let data = try await postData(
            "/community/posts/\(postID)/comments",
            body: ["content": content],
            auth: true
        )
        if let c = data["comment"] as? [String: Any], let parsed = LinLanCommunityComment.parse(c) {
            return parsed
        }
        if let c = data["reply"] as? [String: Any], let parsed = LinLanCommunityComment.parse(c) {
            return parsed
        }
        if let parsed = LinLanCommunityComment.parse(data) { return parsed }
        throw LinLanAPIError.message("评论失败")
    }

    func toggleCommunityPostLike(postID: Int) async throws -> (isLiked: Bool, likeCount: Int) {
        let data = try await postData("/community/posts/\(postID)/like", body: [:], auth: true)
        return (
            bool(data, "is_liked"),
            int(data, "like_count", default: 0)
        )
    }

    func deleteCommunityPost(postID: Int) async throws {
        _ = try await postData("/community/posts/\(postID)/delete", body: [:], auth: true)
    }

    // MARK: - Durable batch forwarding

    func initForwardJob(_ body: [String: Any]) async throws -> [String: Any] {
        try await postData("/forward-jobs/init", body: body, auth: true)
    }

    func uploadForwardTargets(requestID: String, offset: Int, targets: [[String: Any]]) async throws -> [String: Any] {
        try await putData("/forward-jobs/\(requestID)/targets", body: ["offset": offset, "targets": targets], auth: true)
    }

    func uploadForwardMessages(requestID: String, offset: Int, messages: [[String: Any]]) async throws -> [String: Any] {
        try await putData("/forward-jobs/\(requestID)/messages", body: ["offset": offset, "messages": messages], auth: true)
    }

    func commitForwardJob(requestID: String) async throws -> [String: Any] {
        try await postData("/forward-jobs/\(requestID)/commit", body: [:], auth: true)
    }

    func forwardJobStatus(requestID: String) async throws -> [String: Any] {
        try await getAuthData("/forward-jobs/\(requestID)/status")
    }

    func forwardJobPage(requestID: String, cursor: String = "0") async throws -> [String: Any] {
        try await getAuthData("/forward-jobs/\(requestID)", query: ["cursor": cursor])
    }

    func cancelForwardJob(requestID: String) async throws -> [String: Any] {
        try await postData("/forward-jobs/\(requestID)/cancel", body: [:], auth: true)
    }

    // MARK: - Private

    private func searchUsers(path: String, keyword: String) async throws -> [LinLanSearchUser] {
        let data = try await getAuthData(path, query: ["q": keyword])
        let raw = data["users"] ?? data["list"]
        let users: [[String: Any]] = {
            if let arr = raw as? [[String: Any]] { return arr }
            if let arr = raw as? [Any] {
                return arr.compactMap { $0 as? [String: Any] }
            }
            return []
        }()
        return users.compactMap { item in
            var im = string(item, "im_user_id")
            if im.isEmpty { im = string(item, "user_id") }
            guard !im.isEmpty else { return nil }
            var signal = string(item, "wanan_id")
            if signal.isEmpty { signal = string(item, "chuanlu_id") }
            return LinLanSearchUser(
                imUserId: im,
                wananId: signal,
                nickname: string(item, "nickname"),
                avatarUrl: string(item, "avatar_url"),
                canAddFriend: bool(item, "can_add_friend", default: true),
                addBlockedReason: string(item, "add_blocked_reason")
            )
        }
    }

    private func authRequest(_ path: String, body: [String: Any]) async throws -> LinLanAuthSession {
        let generation = sessionStore.generation
        let data = try await postData(path, body: body, auth: false)
        let session = parseAuthSession(data)
        guard session.isLoggedIn, sessionStore.saveSession(session, ifGeneration: generation) else { throw LinLanAPIError.message("登录状态已变化，请重试") }
        return session
    }

    private func getAuthData(
        _ path: String,
        query: [String: String] = [:]
    ) async throws -> [String: Any] {
        let base = await LinLanDomainPool.shared.currentBaseURL()
        guard var components = URLComponents(string: base + path) else {
            throw LinLanAPIError.network("无效地址")
        }
        if !query.isEmpty {
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        guard let url = components.url else {
            throw LinLanAPIError.network("无效地址")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.setValue("Bearer \(sessionStore.currentSession().token)", forHTTPHeaderField: "Authorization")
        return try await executeForData(request)
    }

    private func postData(_ path: String, body: [String: Any], auth: Bool) async throws -> [String: Any] {
        let base = await LinLanDomainPool.shared.currentBaseURL()
        guard let url = URL(string: base + path) else {
            throw LinLanAPIError.network("无效地址")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json; charset=utf-8", forHTTPHeaderField: "Content-Type")
        if auth {
            request.setValue(
                "Bearer \(sessionStore.currentSession().token)",
                forHTTPHeaderField: "Authorization"
            )
        }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        return try await executeForData(request)
    }

    private func putData(_ path: String, body: [String: Any], auth: Bool) async throws -> [String: Any] {
        let base = await LinLanDomainPool.shared.currentBaseURL()
        guard let url = URL(string: base + path) else {
            throw LinLanAPIError.network("无效地址")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("application/json; charset=utf-8", forHTTPHeaderField: "Content-Type")
        if auth {
            request.setValue("Bearer \(sessionStore.currentSession().token)", forHTTPHeaderField: "Authorization")
        }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        return try await executeForData(request)
    }

    private func putFileToCos(putUrl: String, filePath: String, contentType: String) async throws {
        guard let url = URL(string: putUrl) else {
            throw LinLanAPIError.network("无效上传地址")
        }
        let fileURL = URL(fileURLWithPath: filePath)
        guard FileManager.default.fileExists(atPath: filePath) else {
            throw LinLanAPIError.message("文件不存在")
        }
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue(contentType, forHTTPHeaderField: "Content-Type")
        do {
            let (_, response) = try await session.upload(for: request, fromFile: fileURL)
            let code = (response as? HTTPURLResponse)?.statusCode ?? -1
            guard (200..<300).contains(code) else {
                throw LinLanAPIError.message("上传失败(\(code))")
            }
        } catch let error as LinLanAPIError {
            throw error
        } catch {
            throw LinLanAPIError.network(error.localizedDescription)
        }
    }

    private func executeForData(_ request: URLRequest, allowRefresh: Bool = true, expectedOwner: String? = nil, expectedGeneration: UInt64? = nil, allowRouteRetry: Bool = true, networkDeadline: TimeInterval? = nil) async throws -> [String: Any] {
        var request = request
        let deadline = networkDeadline ?? ProcessInfo.processInfo.systemUptime + 30
        let owner = expectedOwner ?? sessionStore.currentSession().userId
        let generation = expectedGeneration ?? sessionStore.generation
        guard sessionStore.currentSession().userId==owner, sessionStore.generation==generation else {throw LinLanAPIError.message("账号已变化")}
        if allowRefresh, let bearer = request.value(forHTTPHeaderField: "Authorization") {
            let saved = sessionStore.currentSession()
            if saved.userId == owner, bearer == "Bearer \(saved.token)", !saved.refreshToken.isEmpty, saved.needsAccessRefresh() {
                let renewed = try await refreshSession()
                guard renewed.userId == owner, sessionStore.generation == generation else { throw CancellationError() }
                request.setValue("Bearer \(renewed.token)", forHTTPHeaderField: "Authorization")
            }
        }
        await LinLanDomainPool.shared.ensureReady()
        guard sessionStore.currentSession().userId == owner, sessionStore.generation == generation else { throw CancellationError() }
        let method = request.httpMethod ?? "GET"
        let url = request.url?.path ?? "(nil)"
        do {
            let remaining = deadline - ProcessInfo.processInfo.systemUptime
            guard remaining > 0 else { throw LinLanAPIError.network("网络连接超时，请重试") }
            var bounded = request
            bounded.timeoutInterval = LinLanDomainPoolPolicy.attemptTimeout(method: method, mayRetry: allowRouteRetry, remaining: remaining, requested: request.timeoutInterval)
            let (data, response) = try await session.data(for: bounded)
            let http = response as? HTTPURLResponse
            let code = http?.statusCode ?? -1
            let root = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] ?? [:]
            if request.value(forHTTPHeaderField: "Authorization") != nil && (sessionStore.generation != generation || sessionStore.currentSession().userId != owner) {
                throw LinLanAPIError.message("账号已切换，请重新操作")
            }
            if !(200..<300).contains(code) {
                let msg = string(root, "message").isEmpty
                    ? "请求失败(\(code))"
                    : string(root, "message")
                if code == 401, allowRefresh, request.value(forHTTPHeaderField: "Authorization") != nil {
                    let refreshed = try await refreshSession()
                    guard refreshed.userId == owner, sessionStore.generation == generation else { throw LinLanAPIError.message("账号已切换") }
                    var retried = request
                    retried.setValue("Bearer \(refreshed.token)", forHTTPHeaderField: "Authorization")
                    return try await executeForData(retried, allowRefresh: false, expectedOwner:owner, expectedGeneration:generation, allowRouteRetry:allowRouteRetry, networkDeadline:deadline)
                }
                throw LinLanAPIError.http(msg, status: code, retryAfter: http?.value(forHTTPHeaderField: "Retry-After").flatMap(TimeInterval.init))
            }
            let biz = root["code"] as? Int ?? -1
            if biz != 0 {
                let msg = string(root, "message").isEmpty ? "请求失败" : string(root, "message")
                throw LinLanAPIError.message(msg)
            }
            return (root["data"] as? [String: Any]) ?? [:]
        } catch let error as LinLanAPIError {
            #if DEBUG
            print("┌─ LinLan API Error\n│ \(method) \(url)\n│ \(error.localizedDescription)\n└─")
            #endif
            if case .http(_, let status, _) = error, LinLanDomainPoolPolicy.isGatewayStatus(status),
                let retried = try await recoverReadRequest(request, owner: owner, generation: generation, allowReadRetry: allowRouteRetry, deadline: deadline) {
                return try await executeForData(retried, allowRefresh: allowRefresh, expectedOwner: owner, expectedGeneration: generation, allowRouteRetry: false, networkDeadline: deadline)
            }
            throw error
        } catch is CancellationError {
            throw CancellationError()
        } catch let error as URLError where error.code == .cancelled {
            throw CancellationError()
        } catch {
            #if DEBUG
            print("┌─ LinLan API Network Error\n│ \(method) \(url)\n│ \(error.localizedDescription)\n└─")
            #endif
            if let retried = try await recoverReadRequest(request, owner: owner, generation: generation, allowReadRetry: allowRouteRetry, deadline: deadline) {
                return try await executeForData(retried, allowRefresh: allowRefresh, expectedOwner: owner, expectedGeneration: generation, allowRouteRetry: false, networkDeadline: deadline)
            }
            throw LinLanAPIError.network(error.localizedDescription)
        }
    }

    private func recoverReadRequest(_ request: URLRequest, owner: String, generation: UInt64, allowReadRetry: Bool, deadline: TimeInterval) async throws -> URLRequest? {
        try await LinLanDomainPoolPolicy.recoveryRequest(request, allowReadRetry: allowReadRetry, deadline: deadline,
            currentBaseURL: { await LinLanDomainPool.shared.currentBaseURL() },
            recover: { failed, budget in await LinLanDomainPool.shared.failoverIfNeeded(failedOrigin: failed, budget: budget) },
            prepareFuture: { failed in await LinLanDomainPool.shared.prepareFutureRoute(failedOrigin: failed) },
            sessionIsCurrent: { self.sessionStore.currentSession().userId == owner && self.sessionStore.generation == generation })
    }

    private static func prettyJSONString(from data: Data?) -> String? {
        guard let data, !data.isEmpty else { return nil }
        guard
            let obj = try? JSONSerialization.jsonObject(with: data),
            let pretty = try? JSONSerialization.data(withJSONObject: obj, options: [.prettyPrinted, .sortedKeys])
        else {
            return String(data: data, encoding: .utf8)
        }
        return String(data: pretty, encoding: .utf8)
    }

    private func parseAuthSession(_ data: [String: Any]) -> LinLanAuthSession {
        LinLanAuthSession(
            userId: string(data, "user_id"),
            userSig: string(data, "user_sig"),
            userSigExpiresAt: string(data, "user_sig_expires_at"),
            token: string(data, "token"),
            tokenExpiresAt: string(data, "token_expires_at"),
            refreshToken: string(data, "refresh_token"),
            mustChangePassword: bool(data, "must_change_password"),
            sdkAppId: int(data, "sdk_app_id", default: 0)
        )
    }

    private func string(_ dict: [String: Any], _ key: String) -> String {
        if let s = dict[key] as? String { return s }
        if let n = dict[key] as? NSNumber { return n.stringValue }
        return ""
    }

    private func bool(_ dict: [String: Any], _ key: String, default def: Bool = false) -> Bool {
        if let b = dict[key] as? Bool { return b }
        if let n = dict[key] as? NSNumber { return n.boolValue }
        return def
    }

    private func int(_ dict: [String: Any], _ key: String, default def: Int) -> Int {
        if let i = dict[key] as? Int { return i }
        if let n = dict[key] as? NSNumber { return n.intValue }
        return def
    }
}
