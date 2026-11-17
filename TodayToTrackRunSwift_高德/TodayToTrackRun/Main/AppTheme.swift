//
//  AppTheme.swift
//  TodayToTrackRun
//
//  视觉方向：米白底 + 珊瑚橙强调（不跟荧光绿深色系）
//  布局语言对齐参考稿：大圆角卡片、等宽数字、分组信息块。
//

import UIKit

enum AppTheme {
    static let accent = UIColor(red: 0.93, green: 0.45, blue: 0.28, alpha: 1)
    static let accentSoft = UIColor(red: 0.93, green: 0.45, blue: 0.28, alpha: 0.14)
    static let pageBackground = UIColor(red: 0.97, green: 0.96, blue: 0.94, alpha: 1)
    static let cardBackground = UIColor.white
    static let textPrimary = UIColor(white: 0.12, alpha: 1)
    static let textSecondary = UIColor(white: 0.45, alpha: 1)
    static let divider = UIColor(white: 0.92, alpha: 1)
    static let weatherBlue = UIColor(red: 0.30, green: 0.55, blue: 0.95, alpha: 1)
    static let weatherBlueEnd = UIColor(red: 0.40, green: 0.70, blue: 1.0, alpha: 1)

    static let cardRadius: CGFloat = 16

    static func mono(_ size: CGFloat, weight: UIFont.Weight = .bold) -> UIFont {
        .monospacedDigitSystemFont(ofSize: size, weight: weight)
    }

    static func title(_ size: CGFloat = 17) -> UIFont {
        .systemFont(ofSize: size, weight: .bold)
    }

    static func makeCard() -> UIView {
        let v = UIView()
        v.backgroundColor = cardBackground
        v.layer.cornerRadius = cardRadius
        v.clipsToBounds = true
        return v
    }

    static func applyFrostedNav(to navigationBar: UINavigationBar?) {
        guard let navigationBar else { return }
        let appearance = UINavigationBarAppearance()
        appearance.configureWithTransparentBackground()
        appearance.backgroundEffect = UIBlurEffect(style: .systemChromeMaterialLight)
        appearance.backgroundColor = pageBackground.withAlphaComponent(0.72)
        appearance.titleTextAttributes = [
            .foregroundColor: textPrimary,
            .font: UIFont.systemFont(ofSize: 17, weight: .semibold)
        ]
        appearance.shadowColor = .clear
        navigationBar.standardAppearance = appearance
        navigationBar.scrollEdgeAppearance = appearance
        navigationBar.compactAppearance = appearance
        navigationBar.tintColor = accent
    }
}
