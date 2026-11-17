//
//  LinLanMessageNotifyViewController.swift
//

import SnapKit
import UIKit

final class LinLanMessageNotifyViewController: UIViewController {
    private let master = UISwitch()
    private let sound = UISwitch()
    private let prefs = LinLanAppContainer.appPrefs()

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "消息通知"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        let card = AppTheme.makeCard()
        let stack = UIStackView()
        stack.axis = .vertical
        card.addSubview(stack)
        stack.snp.makeConstraints { $0.edges.equalToSuperview() }
        master.onTintColor = AppTheme.accent
        sound.onTintColor = AppTheme.accent
        master.isOn = prefs.messageNotifyEnabled
        sound.isOn = prefs.soundNotifyEnabled
        master.addTarget(self, action: #selector(masterChanged), for: .valueChanged)
        sound.addTarget(self, action: #selector(soundChanged), for: .valueChanged)
        stack.addArrangedSubview(row("接收新消息通知", master))
        stack.addArrangedSubview(div())
        stack.addArrangedSubview(row("提示音", sound))

        view.addSubview(card)
        card.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(16)
            make.leading.trailing.equalToSuperview().inset(16)
        }

        let tip = UILabel()
        tip.text = "前台用系统提示音与本地通知；离线走 IM SDK 自带 APNs（需在控制台配置证书 ID）。会话免打扰时静音。"
        tip.font = .systemFont(ofSize: 12)
        tip.textColor = AppTheme.textSecondary
        tip.numberOfLines = 0
        view.addSubview(tip)
        tip.snp.makeConstraints { make in
            make.top.equalTo(card.snp.bottom).offset(12)
            make.leading.trailing.equalToSuperview().inset(20)
        }

        refreshSoundEnabled()
    }

    @objc private func masterChanged() {
        prefs.messageNotifyEnabled = master.isOn
        if master.isOn {
            LinLanMessageAlertCenter.requestAuthorizationIfNeeded()
        }
        refreshSoundEnabled()
    }

    @objc private func soundChanged() {
        prefs.soundNotifyEnabled = sound.isOn
    }

    private func refreshSoundEnabled() {
        sound.isEnabled = master.isOn
        sound.alpha = master.isOn ? 1 : 0.45
    }

    private func row(_ title: String, _ sw: UISwitch) -> UIView {
        let v = UIView()
        v.snp.makeConstraints { $0.height.equalTo(52) }
        let l = UILabel()
        l.text = title
        l.font = .systemFont(ofSize: 16, weight: .medium)
        l.textColor = AppTheme.textPrimary
        v.addSubview(l)
        v.addSubview(sw)
        l.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
        }
        sw.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        return v
    }

    private func div() -> UIView {
        let w = UIView()
        w.snp.makeConstraints { $0.height.equalTo(1) }
        let line = UIView()
        line.backgroundColor = AppTheme.divider
        w.addSubview(line)
        line.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.trailing.equalToSuperview()
            make.centerY.equalToSuperview()
            make.height.equalTo(1 / UIScreen.main.scale)
        }
        return w
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }
}
