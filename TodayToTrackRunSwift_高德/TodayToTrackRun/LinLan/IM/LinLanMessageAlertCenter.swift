//
//  LinLanMessageAlertCenter.swift
//  TodayToTrackRun
//
//  前台：系统提示音；后台：本地通知。尊重全局开关与会话免打扰。
//

import AudioToolbox
import Foundation
import ImSDK_Plus_Swift
import UIKit
import UserNotifications

enum LinLanMessageAlertCenter {
    private static let lock = NSLock()
    private static var started = false
    private static var appInForeground = true
    private static var _activeConversationId: String?

    static var activeConversationId: String? {
        lock.lock(); defer { lock.unlock() }
        return _activeConversationId
    }

    static func start() {
        lock.lock()
        let already = started
        if !already { started = true }
        lock.unlock()
        guard !already else { return }

        UNUserNotificationCenter.current().delegate = LinLanLocalNotifyDelegate.shared
        requestAuthorizationIfNeeded()

        NotificationCenter.default.addObserver(
            forName: .linLanDidRecvMessage,
            object: nil,
            queue: .main
        ) { note in
            guard let msg = note.object as? V2TIMMessage else { return }
            Task { await handleIncoming(msg) }
        }
        NotificationCenter.default.addObserver(
            forName: UIApplication.didBecomeActiveNotification,
            object: nil,
            queue: .main
        ) { _ in
            appInForeground = true
        }
        NotificationCenter.default.addObserver(
            forName: UIApplication.willResignActiveNotification,
            object: nil,
            queue: .main
        ) { _ in
            appInForeground = false
        }
    }

    static func setActiveConversation(_ conversationId: String?) {
        lock.lock()
        _activeConversationId = conversationId?.trimmingCharacters(in: .whitespacesAndNewlines)
        if _activeConversationId?.isEmpty == true { _activeConversationId = nil }
        lock.unlock()
    }

    static func requestAuthorizationIfNeeded() {
        LinLanPushService.requestAuthorizationAndRegister()
    }

    private static func handleIncoming(_ msg: V2TIMMessage) async {
        if msg.isSelf { return }
        if msg.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS { return }
        let session = LinLanAppContainer.sessionStore().currentSession()
        guard session.isLoggedIn else { return }

        let prefs = LinLanAppContainer.appPrefs()
        let muted = await LinLanImManager.isMessageMuted(msg)
        let convId = LinLanImManager.conversationId(of: msg)
        let inActiveChat = {
            lock.lock(); defer { lock.unlock() }
            return !(_activeConversationId ?? "").isEmpty && _activeConversationId == convId
        }()

        // 关闭「接收新消息通知」或会话免打扰 → 全静音
        if muted || !prefs.messageNotifyEnabled { return }

        if appInForeground {
            // 正在看该会话时不响；提示音开关关闭时不响
            if prefs.soundNotifyEnabled && !inActiveChat {
                playSystemSound()
            }
            return
        }

        // 后台：本地通知；提示音开关控制是否带系统声音
        await showLocalNotification(msg: msg, withSound: prefs.soundNotifyEnabled)
    }

    private static func playSystemSound() {
        // 系统「新短信」提示音
        AudioServicesPlaySystemSound(1007)
    }

    private static func showLocalNotification(msg: V2TIMMessage, withSound: Bool) async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        guard settings.authorizationStatus == .authorized || settings.authorizationStatus == .provisional else {
            return
        }

        let title: String = {
            if let n = msg.nickName, !n.isEmpty { return n }
            if let r = msg.friendRemark, !r.isEmpty { return r }
            if let c = msg.nameCard, !c.isEmpty { return c }
            if let s = msg.sender, !s.isEmpty { return s }
            return "新消息"
        }()
        let body = LinLanImManager.messagePreview(msg)
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body.isEmpty ? "发来一条消息" : body
        content.sound = withSound ? .default : nil
        content.userInfo = [
            "conversationId": LinLanImManager.conversationId(of: msg),
            "isGroup": msg.groupID?.isEmpty == false
        ]

        let id = msg.msgID.isEmpty
            ? "linlan_\(Int(Date().timeIntervalSince1970 * 1000))"
            : msg.msgID
        let req = UNNotificationRequest(identifier: id, content: content, trigger: nil)
        try? await UNUserNotificationCenter.current().add(req)
    }
}

private final class LinLanLocalNotifyDelegate: NSObject, UNUserNotificationCenterDelegate {
    static let shared = LinLanLocalNotifyDelegate()

    /// 前台若仍投递本地通知：不弹横幅（前台已用系统提示音）
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([])
    }
}
