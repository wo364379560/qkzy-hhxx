//
//  LinLanPushService.swift
//  TodayToTrackRun
//
//  使用 IM SDK 自带 APNs（V2TIMManager.setAPNS），不接 TIMPush。
//

import Foundation
import ImSDK_Plus_Swift
import UIKit
import UserNotifications

enum LinLanPushService {
    private static let lock = NSLock()
    private static var cachedDeviceToken: Data?

    /// 申请通知权限并注册远程推送（系统 APNs）
    static func requestAuthorizationAndRegister() {
        UNUserNotificationCenter.current().getNotificationSettings { settings in
            switch settings.authorizationStatus {
            case .notDetermined:
                UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { granted, _ in
                    guard granted else { return }
                    DispatchQueue.main.async {
                        UIApplication.shared.registerForRemoteNotifications()
                    }
                }
            case .authorized, .provisional, .ephemeral:
                DispatchQueue.main.async {
                    UIApplication.shared.registerForRemoteNotifications()
                }
            default:
                break
            }
        }
    }

    /// AppDelegate 收到 deviceToken 后调用
    static func updateAPNSToken(_ deviceToken: Data) {
        lock.lock()
        cachedDeviceToken = deviceToken
        lock.unlock()
        applyAPNSToken(deviceToken)
    }

    /// IM 登录成功后重上报（token 可能比登录先到）
    static func resyncAPNSTokenIfNeeded() {
        lock.lock()
        let token = cachedDeviceToken
        lock.unlock()
        guard let token else {
            // 尚未拿到 token：再触发一次系统注册
            DispatchQueue.main.async {
                UIApplication.shared.registerForRemoteNotifications()
            }
            return
        }
        applyAPNSToken(token)
    }

    /// 登出时清除 IM 侧 APNs 绑定
    static func clearAPNSToken() {
        lock.lock()
        cachedDeviceToken = nil
        lock.unlock()
        V2TIMManager.shared.setAPNS(config: nil, succ: nil, fail: nil)
    }

    /// 发送消息时附带的离线推送内容（对方杀进程时靠这个展示）
    static func makeOfflinePushInfo(
        title: String,
        desc: String,
        peerID: String,
        isGroup: Bool,
        senderID: String
    ) -> V2TIMOfflinePushInfo {
        let info = V2TIMOfflinePushInfo()
        info.title = title.isEmpty ? LinLanAppConstants.appName : title
        info.desc = desc.isEmpty ? "发来一条消息" : desc
        info.disablePush = false
        info.iOSSound = kIOSOfflinePushDefaultSound
        info.AndroidXiaoMiChannelID = "157799"
        info.AndroidOPPOChannelID = "push_oplus_category_service"
        info.AndroidOPPOCategory = "IM"
        func json(_ object: [String: String]) -> String { (try? JSONSerialization.data(withJSONObject: object)).flatMap { String(data: $0, encoding: .utf8) } ?? "{}" }
        let vendor: [String: String] = ["xiaomiTemplateId": "M12762", "xiaomiTemplateParam": json(["keywords1": title, "keywords2": desc]),
            "oppoTemplateId": "6a8408344fae550145d04b8f", "oppoTitleParam": json(["senderName": title]), "oppoContentParam": json(["text": desc])]
        info.vendorParams = json(vendor)

        info.ext = makeOfflinePushExt(peerID: peerID, isGroup: isGroup, senderID: senderID, title: title)
        return info
    }

    private static func applyAPNSToken(_ deviceToken: Data) {
        let businessID = LinLanAppConstants.apnsBusinessID
        guard businessID > 0 else {
            print("[LinLanPush] apnsBusinessID 未配置，已拿到 deviceToken 但未上报 IM。请在控制台上传证书后填写证书 ID。")
            return
        }
        // 未登录时先缓存，登录后再 resync
        guard !LinLanImManager.loginUser.isEmpty else { return }

        let config = V2TIMAPNSConfig()
        config.token = deviceToken
        config.businessID = businessID
        V2TIMManager.shared.setAPNS(
            config: config,
            succ: {
                print("[LinLanPush] setAPNS succ, businessID=\(businessID)")
            },
            fail: { code, desc in
                print("[LinLanPush] setAPNS fail \(code): \(desc)")
            }
        )
    }

    private static func makeOfflinePushExt(
        peerID: String,
        isGroup: Bool,
        senderID: String,
        title: String
    ) -> String? {
        let entity: [String: Any] = [
            "action": 1,
            "chatType": isGroup ? 2 : 1,
            "sender": senderID,
            "groupID": isGroup ? peerID : "",
            "userID": isGroup ? "" : peerID,
            "nickname": title,
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: ["entity": entity]),
              let json = String(data: data, encoding: .utf8) else {
            return nil
        }
        return json
    }
}
