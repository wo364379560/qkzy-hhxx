//
//  LinLanAppContainer.swift
//  TodayToTrackRun
//

import Foundation
import UIKit

enum LinLanAppContainer {
    private static let lock = NSLock()
    private static var _sessionStore: LinLanSessionStore?
    private static var _prefs: LinLanAppPrefs?
    private static var _loginCredentials: LinLanLoginCredentialStore?
    private static var _api: LinLanApiClient?
    private static var _auth: LinLanAuthRepository?

    static func sessionStore() -> LinLanSessionStore {
        lock.lock(); defer { lock.unlock() }
        if let s = _sessionStore { return s }
        let created = LinLanSessionStore()
        _sessionStore = created
        return created
    }

    static func appPrefs() -> LinLanAppPrefs {
        lock.lock(); defer { lock.unlock() }
        if let p = _prefs { return p }
        let created = LinLanAppPrefs()
        _prefs = created
        return created
    }

    static func loginCredentials() -> LinLanLoginCredentialStore {
        lock.lock(); defer { lock.unlock() }
        if let store = _loginCredentials { return store }
        let created = LinLanLoginCredentialStore()
        _loginCredentials = created
        return created
    }

    static func apiClient() -> LinLanApiClient {
        lock.lock(); defer { lock.unlock() }
        if let a = _api { return a }
        if _sessionStore == nil { _sessionStore = LinLanSessionStore() }
        let created = LinLanApiClient(sessionStore: _sessionStore!)
        _api = created
        return created
    }

    static func authRepository() -> LinLanAuthRepository {
        lock.lock(); defer { lock.unlock() }
        if let a = _auth { return a }
        if _sessionStore == nil { _sessionStore = LinLanSessionStore() }
        if _api == nil { _api = LinLanApiClient(sessionStore: _sessionStore!) }
        let created = LinLanAuthRepository(sessionStore: _sessionStore!, apiClient: _api!)
        _auth = created
        return created
    }
}

extension Notification.Name {
    static let linLanAuthDidChange = Notification.Name("linLanAuthDidChange")
}

enum LinLanRootSwitcher {
    static func keyWindow() -> UIWindow? {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        for scene in scenes {
            if let key = scene.windows.first(where: \.isKeyWindow) { return key }
        }
        return scenes.flatMap(\.windows).first
    }

    static func applyRoot(loggedIn: Bool, animated: Bool = true) {
        guard let window = keyWindow() else { return }

        let next: UIViewController = loggedIn
            ? MainTabBarController()
            : UINavigationController(rootViewController: LinLanLoginViewController())

        if animated {
            UIView.transition(with: window, duration: 0.25, options: .transitionCrossDissolve) {
                window.rootViewController = next
            }
        } else {
            window.rootViewController = next
        }
        window.makeKeyAndVisible()
    }

    static func showMain() {
        applyRoot(loggedIn: true)
        NotificationCenter.default.post(name: .linLanAuthDidChange, object: nil)
    }

    static func showLogin() {
        applyRoot(loggedIn: false)
        NotificationCenter.default.post(name: .linLanAuthDidChange, object: nil)
    }
}
