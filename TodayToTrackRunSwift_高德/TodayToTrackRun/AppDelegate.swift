//
//  AppDelegate.swift
//  TodayToTrackRun

/**
 APP 软件名称: 睿之禾
 软 件 包 名: com.balabalaim.gcloud.tim.push.appdev
 
 三种情况   我给你列一下 你就明白了

 第一种 就是新注册 没有归属的  添加A 同意添加好友以后  自动归属到A名下

 第二种 已经归属在A的 添加A 就正常流程 -- 这个我还没有懂意思，客户提出来的

 第三种 已经归属在别的名下 去搜索账号A 添加  提示 当前账号属于其他服务空间无法直接添加。

 然后 新增一个功能  后台针对某个账号 开起关闭 可以被全局搜索到
 
 xiaoque/ChengNanDaodas 域名池解析的时候：账号密码在线上 API 可用，但客户端 code/data 的类型转换脆弱，且后台域名须写完整 https://api.xxx。正在加固登录解析并核对域名池格式。
 
 com.yunmaihutong.iosdev
 
 */
import UIKit
import UserNotifications

@main
class AppDelegate: UIResponder, UIApplicationDelegate {

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        do {
            try AppDatabase.setup()
            do {
                try BundledRunSeeder.importIfNeeded()
            } catch {
                // 本地样例导入失败不应阻止 App 启动。
                print("[BundledRunSeeder] import failed: \(error.localizedDescription)")
            }
        } catch {
            assertionFailure("GRDB setup failed: \(error)")
        }
        LinLanMessageAlertCenter.start()
        // 高德隐私状态尽早初始化；推送仍放到首帧后
        AMapBootstrap.configureIfNeeded()
        DispatchQueue.main.async {
            LinLanPushService.requestAuthorizationAndRegister()
        }
        return true
    }

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        LinLanPushService.updateAPNSToken(deviceToken)
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        print("[LinLanPush] register APNs failed: \(error.localizedDescription)")
    }

    func application(
        _ application: UIApplication,
        configurationForConnecting connectingSceneSession: UISceneSession,
        options: UIScene.ConnectionOptions
    ) -> UISceneConfiguration {
        UISceneConfiguration(name: "Default Configuration", sessionRole: connectingSceneSession.role)
    }
}
