//
//  SceneDelegate.swift
//  TodayToTrackRun
//
//com.balabalaim.gcloud.tim.push.appdev 

import UIKit

class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    var window: UIWindow?

    func scene(
        _ scene: UIScene,
        willConnectTo session: UISceneSession,
        options connectionOptions: UIScene.ConnectionOptions
    ) {
        guard let windowScene = scene as? UIWindowScene else { return }
        let window = UIWindow(windowScene: windowScene)
        window.rootViewController = LinLanSplashViewController()
        window.makeKeyAndVisible()
        self.window = window
    }
}
