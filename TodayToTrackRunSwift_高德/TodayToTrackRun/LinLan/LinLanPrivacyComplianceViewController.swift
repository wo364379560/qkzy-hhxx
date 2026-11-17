//
//  LinLanPrivacyComplianceViewController.swift
//  TodayToTrackRun
//
//  隐私与合规：青少年模式 + 协议清单 + 注销（对照合规截图）
//

import SnapKit
import UIKit

final class LinLanPrivacyComplianceViewController: UIViewController {
    private let scroll = UIScrollView()
    private let stack = UIStackView()

    private struct RowItem {
        let title: String
        let symbol: String
        let tint: UIColor
        let action: Selector
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "隐私与合规"
        view.backgroundColor = UIColor(red: 0.93, green: 0.96, blue: 1.0, alpha: 1)
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        view.addSubview(scroll)
        scroll.alwaysBounceVertical = true
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        stack.axis = .vertical
        stack.spacing = 10
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 28, right: 14)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        stack.addArrangedSubview(makeIntroCard())

        let items: [RowItem] = [
            RowItem(title: "青少年模式", symbol: "face.smiling", tint: UIColor(red: 0.20, green: 0.48, blue: 0.96, alpha: 1), action: #selector(openTeen)),
            RowItem(title: "隐私政策", symbol: "shield.lefthalf.filled", tint: UIColor(red: 0.22, green: 0.55, blue: 0.95, alpha: 1), action: #selector(openPrivacy)),
            RowItem(title: "用户协议", symbol: "doc.text.fill", tint: UIColor(red: 0.20, green: 0.50, blue: 0.95, alpha: 1), action: #selector(openAgreement)),
            RowItem(title: "个人信息收集清单", symbol: "checklist", tint: UIColor(red: 0.25, green: 0.45, blue: 0.95, alpha: 1), action: #selector(openPersonalInfo)),
            RowItem(title: "个人信息共享清单", symbol: "point.3.connected.trianglepath.dotted", tint: UIColor(red: 0.28, green: 0.42, blue: 0.95, alpha: 1), action: #selector(openShareList)),
            RowItem(title: "免责声明", symbol: "hammer.fill", tint: UIColor(red: 0.30, green: 0.48, blue: 0.92, alpha: 1), action: #selector(openStatement)),
            RowItem(title: "未成年人保护条款", symbol: "book.fill", tint: UIColor(red: 0.22, green: 0.52, blue: 0.94, alpha: 1), action: #selector(openMinor)),
           // RowItem(title: "第三方 SDK 说明", symbol: "square.stack.3d.up.fill", tint: UIColor(red: 0.26, green: 0.46, blue: 0.93, alpha: 1), action: #selector(openSDK)),
            RowItem(title: "注销账号", symbol: "trash.fill", tint: UIColor(red: 0.95, green: 0.35, blue: 0.30, alpha: 1), action: #selector(openDelete)),
        ]
        items.forEach { stack.addArrangedSubview(makeLinkCard($0)) }
    }

    private func makeIntroCard() -> UIView {
        let card = AppTheme.makeCard()
        let label = UILabel()
        label.text = "以下内容为隐私政策、用户协议及相关合规说明，便于你随时查阅。协议页面将在应用内打开。"
        label.font = .systemFont(ofSize: 14)
        label.textColor = AppTheme.textSecondary
        label.numberOfLines = 0
        card.addSubview(label)
        label.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }
        return card
    }

    private func makeLinkCard(_ item: RowItem) -> UIView {
        let card = AppTheme.makeCard()
        let button = UIButton(type: .system)
        button.addTarget(self, action: item.action, for: .touchUpInside)
        card.addSubview(button)
        button.snp.makeConstraints { $0.edges.equalToSuperview() }
        button.snp.makeConstraints { $0.height.equalTo(56) }

        let iconWrap = UIView()
        iconWrap.isUserInteractionEnabled = false
        iconWrap.backgroundColor = item.tint.withAlphaComponent(0.12)
        iconWrap.layer.cornerRadius = 10
        let icon = UIImageView(image: UIImage(systemName: item.symbol))
        icon.tintColor = item.tint
        icon.contentMode = .scaleAspectFit
        iconWrap.addSubview(icon)

        let title = UILabel()
        title.isUserInteractionEnabled = false
        title.text = item.title
        title.font = .systemFont(ofSize: 16, weight: .medium)
        title.textColor = AppTheme.textPrimary

        let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))
        chevron.isUserInteractionEnabled = false
        chevron.tintColor = UIColor(white: 0.72, alpha: 1)
        chevron.contentMode = .scaleAspectFit

        button.addSubview(iconWrap)
        button.addSubview(title)
        button.addSubview(chevron)

        iconWrap.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(14)
            make.centerY.equalToSuperview()
            make.size.equalTo(34)
        }
        icon.snp.makeConstraints { make in
            make.center.equalToSuperview()
            make.size.equalTo(18)
        }
        title.snp.makeConstraints { make in
            make.leading.equalTo(iconWrap.snp.trailing).offset(12)
            make.centerY.equalToSuperview()
            make.trailing.lessThanOrEqualTo(chevron.snp.leading).offset(-8)
        }
        chevron.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(14)
            make.centerY.equalToSuperview()
            make.width.equalTo(10)
            make.height.equalTo(14)
        }
        return card
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func openTeen() {
        linLanPush(LinLanTeenModeViewController())
    }

    @objc private func openPrivacy() {
        openLegal("隐私政策", LinLanAppConstants.privacyPolicyURL)
    }

    @objc private func openAgreement() {
        openLegal("用户协议", LinLanAppConstants.userAgreementURL)
    }

    @objc private func openPersonalInfo() {
        openLegal("个人信息收集清单", LinLanAppConstants.personalInfoProtocolURL)
    }

    @objc private func openShareList() {
        openLegal("个人信息共享清单", LinLanAppConstants.thirdInfoShareURL)
    }

    @objc private func openStatement() {
        openLegal("免责声明", LinLanAppConstants.statementProtocolURL)
    }

    @objc private func openMinor() {
        openLegal("未成年人保护条款", LinLanAppConstants.minorProtectionURL)
    }

    @objc private func openSDK() {
        openLegal("第三方 SDK 说明", LinLanAppConstants.thirdPartySDKURL)
    }

    private func openLegal(_ title: String, _ url: String) {
        linLanPush(LinLanLegalWebViewController(titleText: title, urlString: url))
    }

    @objc private func openDelete() {
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
                await MainActor.run { LinLanRootSwitcher.showLogin() }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}
