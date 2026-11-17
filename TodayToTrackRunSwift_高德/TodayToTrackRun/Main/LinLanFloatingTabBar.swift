//
//  LinLanFloatingTabBar.swift
//  TodayToTrackRun
//
//  底部胶囊 Tab 始终走玻璃：
//  iOS 26 用系统 Liquid Glass；更早系统用自定义玻璃（模糊 + 半透白 + 高光边）。
//

import UIKit

final class LinLanFloatingTabBar: UIView {
    var onSelect: ((Int) -> Void)?

    private let chrome = UIVisualEffectView()
    private let glassFill = UIView()
    private let glassSheen = UIView()
    private let sheenGradient = CAGradientLayer()
    private let highlight = UIVisualEffectView()
    private var buttons: [UIButton] = []
    private var badgeLabels: [UILabel] = []
    private var selectedIndex = 0
    private let usesSystemLiquidGlass: Bool = {
        if #available(iOS 26.0, *) { return true }
        return false
    }()

    init(items: [(title: String, systemImage: String)]) {
        super.init(frame: .zero)
        isOpaque = false
        backgroundColor = .clear

        chrome.isUserInteractionEnabled = true
        chrome.clipsToBounds = !usesSystemLiquidGlass
        chrome.layer.cornerCurve = .continuous
        addSubview(chrome)
        applyChromeMaterial()

        glassFill.isUserInteractionEnabled = false
        glassFill.backgroundColor = UIColor.white.withAlphaComponent(0.42)
        chrome.contentView.addSubview(glassFill)

        glassSheen.isUserInteractionEnabled = false
        glassSheen.layer.addSublayer(sheenGradient)
        sheenGradient.colors = [
            UIColor.white.withAlphaComponent(0.55).cgColor,
            UIColor.white.withAlphaComponent(0.08).cgColor,
            UIColor.clear.cgColor,
        ]
        sheenGradient.locations = [0, 0.38, 1]
        sheenGradient.startPoint = CGPoint(x: 0.5, y: 0)
        sheenGradient.endPoint = CGPoint(x: 0.5, y: 1)
        chrome.contentView.addSubview(glassSheen)

        highlight.isUserInteractionEnabled = false
        highlight.clipsToBounds = true
        highlight.layer.cornerCurve = .continuous
        chrome.contentView.addSubview(highlight)
        applyHighlightMaterial()

        chrome.layer.shadowColor = UIColor.black.cgColor
        chrome.layer.shadowOpacity = usesSystemLiquidGlass ? 0.08 : 0.14
        chrome.layer.shadowRadius = 16
        chrome.layer.shadowOffset = CGSize(width: 0, height: 6)
        if !usesSystemLiquidGlass {
            chrome.layer.borderWidth = 0.6
            chrome.layer.borderColor = UIColor.white.withAlphaComponent(0.62).cgColor
        }

        // 系统液态玻璃自带高光和折射，不再叠自定义白膜
        glassFill.isHidden = usesSystemLiquidGlass
        glassSheen.isHidden = usesSystemLiquidGlass

        for (index, item) in items.enumerated() {
            var config = UIButton.Configuration.plain()
            config.image = UIImage(
                systemName: item.systemImage,
                withConfiguration: UIImage.SymbolConfiguration(pointSize: 20, weight: .medium)
            )
            config.title = item.title
            config.imagePlacement = .top
            config.imagePadding = 4
            config.contentInsets = NSDirectionalEdgeInsets(top: 7, leading: 2, bottom: 6, trailing: 2)
            config.baseForegroundColor = AppTheme.textPrimary
            config.titleTextAttributesTransformer = UIConfigurationTextAttributesTransformer { incoming in
                var out = incoming
                out.font = .systemFont(ofSize: 10, weight: .medium)
                return out
            }

            let button = UIButton(configuration: config)
            button.tag = index
            button.addTarget(self, action: #selector(tap(_:)), for: .touchUpInside)
            chrome.contentView.addSubview(button)
            buttons.append(button)

            let badge = UILabel()
            badge.font = .systemFont(ofSize: 10, weight: .bold)
            badge.textColor = .white
            badge.backgroundColor = AppTheme.accent
            badge.textAlignment = .center
            badge.layer.cornerRadius = 8
            badge.clipsToBounds = true
            badge.isHidden = true
            chrome.contentView.addSubview(badge)
            badgeLabels.append(badge)
        }
        chrome.contentView.sendSubviewToBack(highlight)
        chrome.contentView.sendSubviewToBack(glassSheen)
        chrome.contentView.sendSubviewToBack(glassFill)
        applySelection(0, animated: false)
    }

    required init?(coder: NSCoder) { nil }

    func setSelectedIndex(_ index: Int, animated: Bool) {
        guard buttons.indices.contains(index) else { return }
        let changed = selectedIndex != index
        selectedIndex = index
        if changed || !animated {
            applySelection(index, animated: animated && changed)
        }
    }

    func setBadge(_ value: String?, at index: Int) {
        guard badgeLabels.indices.contains(index) else { return }
        let label = badgeLabels[index]
        let text = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        label.text = text
        label.isHidden = text.isEmpty
        setNeedsLayout()
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        let home = safeAreaInsets.bottom
        let pillH: CGFloat = 64
        let insetX: CGFloat = 16
        let bottomLift: CGFloat = home > 0 ? 10 : 8
        chrome.frame = CGRect(
            x: insetX,
            y: bounds.height - pillH - bottomLift,
            width: max(0, bounds.width - insetX * 2),
            height: pillH
        )

        if usesSystemLiquidGlass, #available(iOS 26.0, *) {
            chrome.cornerConfiguration = .capsule()
            highlight.cornerConfiguration = .capsule()
        } else {
            chrome.layer.cornerRadius = pillH / 2
            highlight.layer.cornerRadius = (pillH - 12) / 2
        }

        glassFill.frame = chrome.bounds
        glassSheen.frame = chrome.bounds
        sheenGradient.frame = glassSheen.bounds

        let count = CGFloat(max(buttons.count, 1))
        let itemW = chrome.bounds.width / count
        for (i, button) in buttons.enumerated() {
            button.frame = CGRect(x: CGFloat(i) * itemW, y: 0, width: itemW, height: chrome.bounds.height)
            let badge = badgeLabels[i]
            let w = max(16, (badge.text?.count ?? 0) > 2 ? 28 : 16)
            badge.frame = CGRect(x: Int(button.frame.midX) + 8, y: 6, width: w, height: 16)
        }
        highlight.frame = highlightFrame(for: selectedIndex)
        chrome.layer.shadowPath = UIBezierPath(
            roundedRect: chrome.bounds,
            cornerRadius: pillH / 2
        ).cgPath
    }

    private func applyChromeMaterial() {
        if #available(iOS 26.0, *) {
            let glass = UIGlassEffect(style: .regular)
            glass.isInteractive = true
            chrome.effect = glass
        } else {
            chrome.effect = UIBlurEffect(style: .systemUltraThinMaterialLight)
        }
    }

    private func applyHighlightMaterial() {
        if #available(iOS 26.0, *) {
            let glass = UIGlassEffect(style: .clear)
            glass.tintColor = UIColor.white.withAlphaComponent(0.42)
            highlight.effect = glass
            highlight.backgroundColor = .clear
        } else {
            highlight.effect = UIBlurEffect(style: .systemUltraThinMaterialLight)
            highlight.backgroundColor = UIColor.white.withAlphaComponent(0.38)
        }
    }

    private func highlightFrame(for index: Int) -> CGRect {
        guard buttons.indices.contains(index) else { return .zero }
        return buttons[index].frame.insetBy(dx: 3, dy: 5)
    }

    private func applySelection(_ index: Int, animated: Bool) {
        let updates = {
            self.highlight.frame = self.highlightFrame(for: index)
            for (i, button) in self.buttons.enumerated() {
                let on = i == index
                var config = button.configuration
                config?.baseForegroundColor = on ? AppTheme.accent : AppTheme.textPrimary
                button.configuration = config
                button.tintColor = on ? AppTheme.accent : AppTheme.textPrimary
            }
        }
        guard animated else {
            updates()
            return
        }
        UIView.animate(
            withDuration: 0.42,
            delay: 0,
            usingSpringWithDamping: 0.78,
            initialSpringVelocity: 0.55,
            options: [.allowUserInteraction, .beginFromCurrentState],
            animations: updates
        )
    }

    @objc private func tap(_ sender: UIButton) {
        setSelectedIndex(sender.tag, animated: true)
        onSelect?(sender.tag)
    }
}
