//
//  LinLanAuthRepository.swift
//  TodayToTrackRun
//
//  对照 Android：HTTP 鉴权 + TIM login
//

import Foundation
import Photos

final class LinLanAuthRepository {
    private let sessionStore: LinLanSessionStore
    private let lifecycle = LinLanLifecycleQueue()
    private let apiClient: LinLanApiClient

    init(sessionStore: LinLanSessionStore, apiClient: LinLanApiClient) {
        self.sessionStore = sessionStore
        self.apiClient = apiClient
    }

    func bootstrap() async {
        await apiClient.initializeRouting()
        LinLanImManager.initSDK(sdkAppId: sessionStore.currentSession().resolvedSdkAppId)
    }

    func currentSession() -> LinLanAuthSession {
        sessionStore.currentSession()
    }

    /// Only wait for renewal when saved credentials are missing or close to expiry.
    func decideSplashLoggedIn() async -> Bool {
        let local = sessionStore.currentSession()
        if local.canResume {
            if local.needsSessionRefresh() {
                _ = try? await apiClient.refreshSession()
            }
            let current = sessionStore.currentSession()
            guard current.canResume else { return false }
            LinLanImManager.initSDK(sdkAppId: current.resolvedSdkAppId)
            Task {
                await self.apiClient.initializeRouting()
                // 预热相册权限，减轻首次选图/视频慢
                _ = await PHPhotoLibrary.requestAuthorization(for: .readWrite)
            }
            return true
        }

        // 未登录：短暂探测域名后进登录页，超时也不挡 UI
        _ = await withSoftTimeout(seconds: 0.8) {
            await self.bootstrap()
            return true
        }
        Task { await self.bootstrap() }
        return false
    }

    func ensureSessionReady() async throws {
        let session = sessionStore.currentSession()
        if session.needsSessionRefresh() {
            let renewed = try await apiClient.refreshSession()
            guard renewed.userId == session.userId else { throw CancellationError() }
            try await completeImLogin(renewed)
            return
        }
        guard session.isLoggedIn else { return }
        if LinLanImManager.isLoggedIn(as: session.userId) { return }
        // IM 未登录或账号不一致：强制按 HTTP session 重新登录 IM
        do {
            try await completeImLogin(session)
        } catch {
            guard !(error is CancellationError), !session.refreshToken.isEmpty,
                  sessionStore.currentSession() == session else { throw error }
            let renewed = try await apiClient.refreshSession()
            try await completeImLogin(renewed)
        }
    }

    @discardableResult
    func loginWithPassword(phone: String, password: String) async throws -> LinLanAuthSession {
        let session = try await apiClient.loginWithPassword(phone: phone, password: password)
        try await completeImLogin(session)
        LinLanAppContainer.appPrefs().accountPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        return session
    }

    @discardableResult
    func loginWithCode(phone: String, code: String) async throws -> LinLanAuthSession {
        let session = try await apiClient.loginWithCode(phone: phone, code: code)
        try await completeImLogin(session)
        LinLanAppContainer.appPrefs().accountPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        return session
    }

    @discardableResult
    func register(
        phone: String,
        code: String?,
        nickname: String,
        password: String,
        inviteCode: String?
    ) async throws -> LinLanAuthSession {
        let session = try await apiClient.register(
            phone: phone,
            code: code,
            nickname: nickname,
            password: password,
            inviteCode: inviteCode
        )
        try await completeImLogin(session)
        LinLanAppContainer.appPrefs().accountPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        return session
    }

    func logout() async {
        let captured = sessionStore.currentSession()
        let deviceID = sessionStore.deviceId()
        sessionStore.clearSession()
        let remote = Task { try await self.apiClient.logoutSession(captured, deviceID: deviceID) }
        try? await lifecycle.run {
            LinLanPushService.clearAPNSToken()
            await LinLanRtcCallManager.shared.logoutCleanup()
            if !(await LinLanImManager.logout()) { print("IM退出未确认，本地已退出") }
        }
        clearAccountLocalState()
        do { try await remote.value } catch { print("服务端注销未确认，本地已退出") }
    }

    /// 切号时清理应用层缓存（好友/会话本身走 TIM，靠 logout+relogin 隔离）
    private func clearAccountLocalState() {
        LinLanGroupMemberCache.clearAll()
        LinLanPinnedMessageStore.clearAll()
        LinLanMessageAlertCenter.setActiveConversation(nil)
        let prefs = LinLanAppContainer.appPrefs()
        prefs.cachedNickname = ""
        prefs.cachedSignalId = ""
        prefs.accountPhone = ""
    }

    func sendSms(phone: String) async throws {
        try await apiClient.sendSms(phone: phone)
    }

    func resetPassword(phone: String, code: String, newPassword: String) async throws {
        try await apiClient.resetPassword(phone: phone, code: code, newPassword: newPassword)
        let credentials = LinLanAppContainer.loginCredentials()
        if credentials.load()?.phone == phone.trimmingCharacters(in: .whitespacesAndNewlines) {
            _ = credentials.save(phone: phone, password: newPassword)
        }
    }

    func updatePassword(currentPassword: String, newPassword: String) async throws {
        try await apiClient.updatePassword(
            currentPassword: currentPassword,
            newPassword: newPassword
        )
        let credentials = LinLanAppContainer.loginCredentials()
        if let remembered = credentials.load(), remembered.password == currentPassword {
            _ = credentials.save(phone: remembered.phone, password: newPassword)
        }
        var session = sessionStore.currentSession()
        if session.mustChangePassword {
            session.mustChangePassword = false
            sessionStore.saveSession(session)
        }
    }

    private func completeImLogin(_ session: LinLanAuthSession) async throws {
        let generation = sessionStore.generation
        try await lifecycle.run {
            guard self.sessionStore.generation == generation, self.sessionStore.currentSession().userId == session.userId else { throw CancellationError() }
            try await self.performImLogin(session, generation: generation)
        }
    }

    private func performImLogin(_ session: LinLanAuthSession, generation: UInt64) async throws {
        guard session.isLoggedIn else {
            throw LinLanAPIError.message("登录数据不完整")
        }

        LinLanImManager.initSDK(sdkAppId: session.resolvedSdkAppId)
        // 目标账号与当前不一致时，先清本地业务缓存再切 IM
        if !LinLanImManager.loginUser.isEmpty, LinLanImManager.loginUser != session.userId {
            clearAccountLocalState()
        }
        try await withHardTimeout(seconds: 8) {
            try await LinLanImManager.login(userId: session.userId, userSig: session.userSig)
        }
        guard sessionStore.generation == generation, sessionStore.currentSession().userId == session.userId, LinLanImManager.loginUser == session.userId else {
            throw LinLanAPIError.message("IM 登录失败，请重试")
        }
        LinLanImManager.registerMessageListeners()
        LinLanFriendOwnershipListener.bind()
        LinLanPushService.resyncAPNSTokenIfNeeded()
        LinLanRtcCallManager.shared.setup(
            userId: session.userId,
            userSig: session.userSig,
            sdkAppId: session.resolvedSdkAppId,
            displayName: LinLanAppContainer.appPrefs().cachedNickname
        )

        // 个人资料不是完成登录的必要条件。放到后台更新，避免该接口超时或线路较慢时
        // 登录页一直停留在“登录中…”，影响正常进入首页及 App 审核。
        refreshProfileAfterLogin(
            expectedUserId: session.userId,
            expectedGeneration: generation
        )
    }

    private func refreshProfileAfterLogin(expectedUserId: String, expectedGeneration: UInt64) {
        Task { [weak self] in
            guard let self,
                  self.sessionStore.generation == expectedGeneration,
                  self.sessionStore.currentSession().userId == expectedUserId,
                  let profile = try? await self.apiClient.fetchProfile(),
                  self.sessionStore.generation == expectedGeneration,
                  self.sessionStore.currentSession().userId == expectedUserId
            else { return }

            let prefs = LinLanAppContainer.appPrefs()
            prefs.cachedNickname = profile.nickname
            prefs.cachedSignalId = profile.displaySignalId
            if !profile.phone.isEmpty { prefs.accountPhone = profile.phone }
            LinLanRtcCallManager.shared.updateSelfProfile(
                name: profile.nickname,
                avatarUrl: profile.avatarUrl
            )
        }
    }

    private func withSoftTimeout<T: Sendable>(
        seconds: Double,
        operation: @escaping @Sendable () async -> T
    ) async -> T? {
        await withTaskGroup(of: T?.self) { group in
            group.addTask { await operation() }
            group.addTask {
                try? await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000))
                return nil
            }
            let first = await group.next() ?? nil
            group.cancelAll()
            return first ?? nil
        }
    }

    private func withHardTimeout(
        seconds: Double,
        operation: @escaping @Sendable () async throws -> Void
    ) async throws {
        try await withThrowingTaskGroup(of: Void.self) { group in
            group.addTask { try await operation() }
            group.addTask {
                try await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000))
                throw LinLanImError.timeout
            }
            try await group.next()
            group.cancelAll()
        }
    }
}

private actor LinLanLifecycleQueue {
    private var tail: Task<Void, Never>?
    func run(_ action: @escaping () async throws -> Void) async throws {
        let prior = tail
        let next = Task { if let prior { await prior.value }; try await action() }
        tail = Task { _ = try? await next.value }
        try await next.value
    }
}
