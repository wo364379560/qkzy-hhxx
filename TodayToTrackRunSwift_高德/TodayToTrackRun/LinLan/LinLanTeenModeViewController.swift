//
//  LinLanTeenModeViewController.swift
//  TodayToTrackRun
//
//  青少年模式：开启设 4 位数字密码，关闭需验证（对照合规截图）
//

import SnapKit
import UIKit

final class LinLanTeenModeViewController: UIViewController {
    private let prefs = LinLanAppContainer.appPrefs()
    private let modeSwitch = UISwitch()
    private let statusLabel = UILabel()
    private var updatingSwitch = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "青少年模式"
        view.backgroundColor = UIColor(red: 0.94, green: 0.95, blue: 0.97, alpha: 1)
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        let scroll = UIScrollView()
        scroll.alwaysBounceVertical = true
        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 14
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 28, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        stack.addArrangedSubview(makeInfoCard())
        stack.addArrangedSubview(makeToggleCard())
        stack.addArrangedSubview(makeRulesCard())
        reloadSwitch()
    }

    private func makeInfoCard() -> UIView {
        let card = UIView()
        card.backgroundColor = UIColor(red: 0.18, green: 0.36, blue: 0.78, alpha: 1)
        card.layer.cornerRadius = 16
        card.clipsToBounds = true

        let label = UILabel()
        label.text = "开启后将对使用时间与部分功能进行限制，帮助营造更健康的使用环境。关闭需验证家长设定的密码。"
        label.font = .systemFont(ofSize: 15)
        label.textColor = .white
        label.numberOfLines = 0
        card.addSubview(label)
        label.snp.makeConstraints { $0.edges.equalToSuperview().inset(18) }
        return card
    }

    private func makeToggleCard() -> UIView {
        let card = AppTheme.makeCard()
        let row = UIView()
        card.addSubview(row)
        row.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.height.equalTo(72)
        }

        let title = UILabel()
        title.text = "开启青少年模式"
        title.font = .systemFont(ofSize: 16, weight: .semibold)
        title.textColor = AppTheme.textPrimary

        statusLabel.font = .systemFont(ofSize: 13)
        statusLabel.textColor = AppTheme.textSecondary

        modeSwitch.onTintColor = AppTheme.accent
        modeSwitch.addTarget(self, action: #selector(switchChanged), for: .valueChanged)

        let textCol = UIStackView(arrangedSubviews: [title, statusLabel])
        textCol.axis = .vertical
        textCol.spacing = 4

        row.addSubview(textCol)
        row.addSubview(modeSwitch)
        textCol.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.trailing.lessThanOrEqualTo(modeSwitch.snp.leading).offset(-12)
        }
        modeSwitch.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        return card
    }

    private func makeRulesCard() -> UIView {
        let card = AppTheme.makeCard()
        let title = UILabel()
        title.text = "开启后的限制"
        title.font = .systemFont(ofSize: 16, weight: .semibold)
        title.textColor = AppTheme.textPrimary

        let rules = UIStackView()
        rules.axis = .vertical
        rules.spacing = 12
        for item in [
            ("使用时间限制", "每日使用时长不超过 40 分钟"),
            ("夜间使用限制", "22:00 - 06:00 期间无法使用"),
            ("消费限制", "无法进行任何充值和消费操作"),
            ("添加好友限制", "无法主动添加陌生人为好友"),
            ("通知限制", "仅接收重要系统通知"),
        ] {
            rules.addArrangedSubview(makeRuleRow(title: item.0, detail: item.1))
        }

        card.addSubview(title)
        card.addSubview(rules)
        title.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview().inset(16)
        }
        rules.snp.makeConstraints { make in
            make.top.equalTo(title.snp.bottom).offset(14)
            make.leading.trailing.bottom.equalToSuperview().inset(16)
        }
        return card
    }

    private func makeRuleRow(title: String, detail: String) -> UIView {
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 14, weight: .medium)
        t.textColor = AppTheme.textPrimary
        let d = UILabel()
        d.text = detail
        d.font = .systemFont(ofSize: 12)
        d.textColor = AppTheme.textSecondary
        let col = UIStackView(arrangedSubviews: [t, d])
        col.axis = .vertical
        col.spacing = 2
        return col
    }

    private func reloadSwitch() {
        updatingSwitch = true
        let on = prefs.teenModeEnabled
        modeSwitch.isOn = on
        statusLabel.text = on ? "已开启" : "未开启"
        updatingSwitch = false
    }

    @objc private func switchChanged() {
        guard !updatingSwitch else { return }
        if modeSwitch.isOn {
            showSetPasswordDialog()
        } else {
            showVerifyPasswordDialog()
        }
    }

    private func showSetPasswordDialog() {
        let alert = UIAlertController(
            title: "设置密码",
            message: "请设置 4 位数字密码，用于关闭青少年模式",
            preferredStyle: .alert
        )
        alert.addTextField {
            $0.placeholder = "4 位数字密码"
            $0.keyboardType = .numberPad
            $0.isSecureTextEntry = true
            $0.textAlignment = .center
        }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel) { [weak self] _ in
            self?.reloadSwitch()
        })
        alert.addAction(UIAlertAction(title: "确定", style: .default) { [weak self] _ in
            guard let self else { return }
            let pin = alert.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard pin.count == 4, pin.allSatisfy(\.isNumber) else {
                self.toast("请输入 4 位数字密码")
                self.reloadSwitch()
                return
            }
            self.prefs.teenModePin = pin
            self.prefs.teenModeEnabled = true
            self.reloadSwitch()
            self.toast("青少年模式已开启")
        })
        present(alert, animated: true)
    }

    private func showVerifyPasswordDialog() {
        let alert = UIAlertController(
            title: "验证密码",
            message: "请输入家长设定的 4 位密码以关闭青少年模式",
            preferredStyle: .alert
        )
        alert.addTextField {
            $0.placeholder = "4 位数字密码"
            $0.keyboardType = .numberPad
            $0.isSecureTextEntry = true
            $0.textAlignment = .center
        }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel) { [weak self] _ in
            self?.reloadSwitch()
        })
        alert.addAction(UIAlertAction(title: "确定", style: .default) { [weak self] _ in
            guard let self else { return }
            let pin = alert.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard pin == self.prefs.teenModePin else {
                self.toast("密码错误")
                self.reloadSwitch()
                return
            }
            self.prefs.teenModeEnabled = false
            self.reloadSwitch()
            self.toast("青少年模式已关闭")
        })
        present(alert, animated: true)
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}
