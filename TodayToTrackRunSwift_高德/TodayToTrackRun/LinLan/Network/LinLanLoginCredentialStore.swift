//
//  LinLanLoginCredentialStore.swift
//  TodayToTrackRun
//
//  “记住密码”凭据只保存在本机 Keychain，不写入 UserDefaults。
//

import Foundation
import Security

struct LinLanRememberedCredentials: Codable {
    let phone: String
    let password: String
}

final class LinLanLoginCredentialStore {
    private let service = (Bundle.main.bundleIdentifier ?? "TodayToTrackRun") + ".remembered-login"
    private let account = "password-login"

    func load() -> LinLanRememberedCredentials? {
        var result: CFTypeRef?
        let query = baseQuery().merging([
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]) { _, new in new }

        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data
        else { return nil }

        return try? JSONDecoder().decode(LinLanRememberedCredentials.self, from: data)
    }

    @discardableResult
    func save(phone: String, password: String) -> Bool {
        let normalizedPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedPhone.isEmpty, !password.isEmpty,
              let data = try? JSONEncoder().encode(
                  LinLanRememberedCredentials(phone: normalizedPhone, password: password)
              )
        else { return false }

        let updateStatus = SecItemUpdate(
            baseQuery() as CFDictionary,
            [kSecValueData as String: data] as CFDictionary
        )
        if updateStatus == errSecSuccess { return true }
        guard updateStatus == errSecItemNotFound else { return false }

        let item = baseQuery().merging([
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]) { _, new in new }
        return SecItemAdd(item as CFDictionary, nil) == errSecSuccess
    }

    func clear() {
        SecItemDelete(baseQuery() as CFDictionary)
    }

    private func baseQuery() -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }
}
