//
//  LinLanSecurityPrivacyViewController.swift
//

import SnapKit
import UIKit

final class LinLanSecurityPrivacyViewController: UIViewController {
    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let friendVerifySwitch = UISwitch()
    private var updatingVerify = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "安全与隐私"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        stack.axis = .vertical
        stack.spacing = 14
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 28, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        stack.addArrangedSubview(section("加好友验证"))
        let verifyCard = AppTheme.makeCard()
        let verifyRow = UIView()
        verifyRow.snp.makeConstraints { $0.height.equalTo(56) }
        let verifyLabel = UILabel()
        verifyLabel.text = "需要验证才能添加我为好友"
        verifyLabel.font = .systemFont(ofSize: 16, weight: .medium)
        verifyLabel.textColor = AppTheme.textPrimary
        friendVerifySwitch.onTintColor = AppTheme.accent
        friendVerifySwitch.isOn = true
        friendVerifySwitch.addTarget(self, action: #selector(friendVerifyChanged), for: .valueChanged)
        verifyCard.addSubview(verifyRow)
        verifyRow.addSubview(verifyLabel)
        verifyRow.addSubview(friendVerifySwitch)
        verifyRow.snp.makeConstraints { $0.edges.equalToSuperview() }
        verifyLabel.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.trailing.lessThanOrEqualTo(friendVerifySwitch.snp.leading).offset(-8)
        }
        friendVerifySwitch.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        stack.addArrangedSubview(verifyCard)

        stack.addArrangedSubview(section("账号安全"))
        let accountCard = AppTheme.makeCard()
        let accountStack = UIStackView()
        accountStack.axis = .vertical
        accountCard.addSubview(accountStack)
        accountStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        accountStack.addArrangedSubview(nav("修改密码", #selector(linLanChangePassword)))
        accountStack.addArrangedSubview(div())
        accountStack.addArrangedSubview(nav("登录设备管理", #selector(linLanLoginDevices)))
        accountStack.addArrangedSubview(div())
        accountStack.addArrangedSubview(nav("账号保护", #selector(linLanAccountProtection)))
        accountStack.addArrangedSubview(div())
        accountStack.addArrangedSubview(nav("注销账号", #selector(linLanDeleteAccount), danger: true))
        stack.addArrangedSubview(accountCard)

        stack.addArrangedSubview(section("保护与协议"))
        let legalCard = AppTheme.makeCard()
        let legalStack = UIStackView()
        legalStack.axis = .vertical
        legalCard.addSubview(legalStack)
        legalStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        legalStack.addArrangedSubview(nav("隐私与合规", #selector(linLanCompliance)))
        stack.addArrangedSubview(legalCard)

        Task { await loadFriendVerify() }
    }

    private func loadFriendVerify() async {
        do {
            let profile = try await LinLanAppContainer.apiClient().fetchProfile()
            await MainActor.run {
                // 1=需要验证，0=允许任何人
                self.friendVerifySwitch.isOn = profile.allowType != 0
            }
        } catch {
            // 保持默认开
        }
    }

    private func section(_ t: String) -> UILabel {
        let l = UILabel()
        l.text = t
        l.font = .systemFont(ofSize: 13, weight: .medium)
        l.textColor = AppTheme.textSecondary
        return l
    }

    private func nav(_ title: String, _ action: Selector, danger: Bool = false) -> UIButton {
        let b = UIButton(type: .system)
        b.contentHorizontalAlignment = .left
        b.contentEdgeInsets = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 16)
        b.setTitle(title, for: .normal)
        b.setTitleColor(danger ? UIColor.systemRed : AppTheme.textPrimary, for: .normal)
        b.titleLabel?.font = .systemFont(ofSize: 16, weight: .medium)
        b.snp.makeConstraints { $0.height.equalTo(52) }
        b.addTarget(self, action: action, for: .touchUpInside)
        let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))
        chevron.tintColor = UIColor(white: 0.75, alpha: 1)
        b.addSubview(chevron)
        chevron.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        return b
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

    @objc private func friendVerifyChanged() {
        guard !updatingVerify else { return }
        let on = friendVerifySwitch.isOn
        let previous = !on
        updatingVerify = true
        friendVerifySwitch.isEnabled = false
        let allowType = on ? 1 : 0
        Task {
            do {
                try await LinLanAppContainer.apiClient().updateProfile(allowType: allowType)
                await LinLanImManager.syncSelfProfile(nickname: nil, faceUrl: nil, allowType: allowType)
                await MainActor.run {
                    self.updatingVerify = false
                    self.friendVerifySwitch.isEnabled = true
                    self.toast(on ? "已开启好友验证" : "已关闭好友验证")
                }
            } catch {
                await MainActor.run {
                    self.updatingVerify = false
                    self.friendVerifySwitch.isOn = previous
                    self.friendVerifySwitch.isEnabled = true
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func linLanChangePassword() {
        linLanPush(LinLanChangePasswordViewController())
    }

    @objc private func linLanLoginDevices() {
        linLanPush(LinLanLoginDevicesViewController())
    }

    @objc private func linLanAccountProtection() {
        linLanPush(LinLanAccountProtectionViewController())
    }

    @objc private func linLanDeleteAccount() {
        let a = UIAlertController(
            title: "注销账号",
            message: "注销后账号数据将无法恢复，确定继续？",
            preferredStyle: .alert
        )
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "确认注销", style: .destructive) { [weak self] _ in
            self?.performDeleteAccount()
        })
        present(a, animated: true)
    }

    private func performDeleteAccount() {
        Task {
            do {
                try await LinLanAppContainer.apiClient().deleteAccount()
                await LinLanAppContainer.authRepository().logout()
                await MainActor.run {
                    LinLanRootSwitcher.showLogin()
                }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    @objc private func linLanCompliance() {
        linLanPush(LinLanPrivacyComplianceViewController())
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}
