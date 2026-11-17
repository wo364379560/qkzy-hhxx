//
//  LinLanSessionStore.swift
//  TodayToTrackRun
//

import Foundation

final class LinLanSessionStore {
    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) { self.defaults = defaults }
    private let lock = NSRecursiveLock()
    private var epoch: UInt64 = 0
    var generation: UInt64 { lock.lock(); defer { lock.unlock() }; return epoch }
    private let prefix = "linlan.session."

    func currentSession() -> LinLanAuthSession {
        lock.lock(); defer { lock.unlock() }
        return LinLanAuthSession(
            userId: string("user_id"),
            userSig: string("user_sig"),
            userSigExpiresAt: string("user_sig_expires_at"),
            token: string("token"),
            tokenExpiresAt: string("token_expires_at"),
            refreshToken: string("refresh_token"),
            mustChangePassword: defaults.bool(forKey: key("must_change_password")),
            sdkAppId: defaults.integer(forKey: key("sdk_app_id"))
        )
    }

    func saveSession(_ session: LinLanAuthSession) {
        lock.lock(); defer { lock.unlock() }
        defaults.set(session.userId, forKey: key("user_id"))
        defaults.set(session.userSig, forKey: key("user_sig"))
        defaults.set(session.userSigExpiresAt, forKey: key("user_sig_expires_at"))
        defaults.set(session.token, forKey: key("token"))
        defaults.set(session.tokenExpiresAt, forKey: key("token_expires_at"))
        defaults.set(session.refreshToken, forKey: key("refresh_token"))
        defaults.set(session.mustChangePassword, forKey: key("must_change_password"))
        defaults.set(session.sdkAppId, forKey: key("sdk_app_id"))
    }

    func clearSession() {
        lock.lock(); defer { lock.unlock() }
        epoch &+= 1
        [
            "user_id", "user_sig", "user_sig_expires_at",
            "token", "token_expires_at", "refresh_token", "must_change_password",
            "sdk_app_id",
        ].forEach { defaults.removeObject(forKey: key($0)) }
    }

    @discardableResult
    func replaceSession(_ expected: LinLanAuthSession, with next: LinLanAuthSession) -> Bool {
        lock.lock(); defer { lock.unlock() }
        guard currentSession() == expected else { return false }
        saveSession(next)
        return true
    }

    @discardableResult
    func saveSession(_ session: LinLanAuthSession, ifGeneration expected: UInt64) -> Bool {
        lock.lock(); defer { lock.unlock() }
        guard epoch == expected else { return false }
        saveSession(session)
        return true
    }

    func deviceId() -> String {
        lock.lock(); defer { lock.unlock() }
        let k = key("device_id")
        if let existing = defaults.string(forKey: k), !existing.isEmpty {
            return existing
        }
        let created = UUID().uuidString
        defaults.set(created, forKey: k)
        return created
    }

    private func key(_ name: String) -> String { prefix + name }

    private func string(_ name: String) -> String {
        defaults.string(forKey: key(name)) ?? ""
    }
}
