//
//  MainTabBarController.swift
//  TodayToTrackRun
//
//  LinLan 信息架构：运动 | 统计 | 消息 | 我的
//

import UIKit

final class MainTabBarController: UITabBarController {
    /// 保留自定义胶囊实现，当前统一使用系统 TabBar（含 iOS 26 Liquid Glass）。
    private let showsCustomFloatingBar = false
    private let floatingBar = LinLanFloatingTabBar(items: [
        ("运动", "figure.run"),
        ("统计", "chart.bar.fill"),
        ("消息", "bubble.left.and.bubble.right.fill"),
        ("我的", "person.fill"),
    ])
    private var badgeObservations: [NSKeyValueObservation] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        delegate = self
        configureTabs()
        configureAppearance()
        configureFloatingBar()
        // Splash 可能跳过等待；进主页后补齐 TIM login
        Task {
            _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
        }
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        syncFloatingBar()
    }

    private func configureTabs() {
        let sport = wrap(RunMapViewController(), title: "运动", systemImage: "figure.run")
        let stats = wrap(LinLanStatsViewController(), title: "统计", systemImage: "chart.bar.fill")
        let message = wrap(LinLanMessageContainerViewController(), title: "消息", systemImage: "bubble.left.and.bubble.right.fill")
        let me = wrap(MeHomeViewController(), title: "我的", systemImage: "person.fill")
        viewControllers = [sport, stats, message, me]
        selectedIndex = 0
        observeBadges()
    }

    private func wrap(_ root: UIViewController, title: String, systemImage: String) -> UINavigationController {
        root.title = title
        let nav = UINavigationController(rootViewController: root)
        nav.tabBarItem = UITabBarItem(
            title: title,
            image: UIImage(systemName: systemImage),
            selectedImage: UIImage(systemName: systemImage)
        )
        return nav
    }

    private func configureAppearance() {
        let appearance = UITabBarAppearance()
        appearance.configureWithDefaultBackground()
        appearance.stackedLayoutAppearance.normal.iconColor = AppTheme.textSecondary
        appearance.stackedLayoutAppearance.normal.titleTextAttributes = [
            .foregroundColor: AppTheme.textSecondary,
        ]
        appearance.stackedLayoutAppearance.selected.iconColor = AppTheme.accent
        appearance.stackedLayoutAppearance.selected.titleTextAttributes = [
            .foregroundColor: AppTheme.accent,
        ]
        tabBar.standardAppearance = appearance
        tabBar.scrollEdgeAppearance = appearance
        tabBar.tintColor = AppTheme.accent
        tabBar.unselectedItemTintColor = AppTheme.textSecondary
        tabBar.isTranslucent = true
        tabBar.isUserInteractionEnabled = true
        additionalSafeAreaInsets.bottom = 0
    }

    private func configureFloatingBar() {
        floatingBar.onSelect = { [weak self] index in
            guard let self else { return }
            self.selectedIndex = index
        }
        view.addSubview(floatingBar)
        floatingBar.isHidden = !showsCustomFloatingBar
    }

    private func syncFloatingBar() {
        guard showsCustomFloatingBar else {
            floatingBar.isHidden = true
            return
        }
        floatingBar.isHidden = tabBar.isHidden
        floatingBar.alpha = tabBar.alpha
        let overflow: CGFloat = 4
        var frame = tabBar.frame
        frame.origin.y -= overflow
        frame.size.height += overflow
        floatingBar.frame = frame
        view.bringSubviewToFront(floatingBar)
    }

    private func observeBadges() {
        badgeObservations.removeAll()
        for (index, item) in (tabBar.items ?? []).enumerated() {
            floatingBar.setBadge(item.badgeValue, at: index)
            let obs = item.observe(\.badgeValue, options: [.new]) { [weak self] tabItem, _ in
                self?.floatingBar.setBadge(tabItem.badgeValue, at: index)
            }
            badgeObservations.append(obs)
        }
    }
}

extension MainTabBarController: UITabBarControllerDelegate {
    func tabBarController(
        _ tabBarController: UITabBarController,
        shouldSelect viewController: UIViewController
    ) -> Bool {
        if let index = viewControllers?.firstIndex(of: viewController) {
            floatingBar.setSelectedIndex(index, animated: true)
        }
        return true
    }
}
