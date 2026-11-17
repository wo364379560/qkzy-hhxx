import SnapKit
import UIKit

final class LinLanAccountProtectionViewController: UIViewController {
    private let remoteSwitch = UISwitch()
    private let verificationSwitch = UISwitch()
    private var settings = LinLanSecuritySettings()
    private var saving = false
    private let defaults = UserDefaults.standard
    private let remoteKey = "linlan.security.remoteLoginAlert"
    private let verificationKey = "linlan.security.loginVerification"

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "账号保护"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        setupUI()
        loadSettings()
    }

    private func setupUI() {
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 12
        view.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(16)
            make.leading.trailing.equalToSuperview().inset(16)
        }

        let info = UILabel()
        info.text = "开启以下保护后，系统会在检测到异常登录行为时提醒你，降低账号被盗风险。设置将同步到云端。"
        info.font = .systemFont(ofSize: 14)
        info.textColor = AppTheme.textSecondary
        info.numberOfLines = 0
        let infoCard = AppTheme.makeCard()
        infoCard.addSubview(info)
        info.snp.makeConstraints { $0.edges.equalToSuperview().inset(18) }
        stack.addArrangedSubview(infoCard)

        remoteSwitch.onTintColor = AppTheme.accent
        verificationSwitch.onTintColor = AppTheme.accent
        remoteSwitch.addTarget(self, action: #selector(remoteChanged), for: .valueChanged)
        verificationSwitch.addTarget(self, action: #selector(verificationChanged), for: .valueChanged)
        stack.addArrangedSubview(switchCard(symbol: "location.fill", title: "异地登录提醒", subtitle: "登录成功后通过安全助手会话提醒你", toggle: remoteSwitch))
        stack.addArrangedSubview(switchCard(symbol: "checkmark.shield.fill", title: "新设备登录验证", subtitle: "新设备登录需在另一台已登录设备上确认", toggle: verificationSwitch))
    }

    private func switchCard(symbol: String, title: String, subtitle: String, toggle: UISwitch) -> UIView {
        let card = AppTheme.makeCard()
        let icon = UIImageView(image: UIImage(systemName: symbol))
        icon.tintColor = AppTheme.accent
        icon.contentMode = .scaleAspectFit
        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        let sub = UILabel()
        sub.text = subtitle
        sub.font = .systemFont(ofSize: 12)
        sub.textColor = AppTheme.textSecondary
        sub.numberOfLines = 2
        let labels = UIStackView(arrangedSubviews: [titleLabel, sub])
        labels.axis = .vertical
        labels.spacing = 3
        card.addSubview(icon); card.addSubview(labels); card.addSubview(toggle)
        icon.snp.makeConstraints { $0.leading.equalToSuperview().offset(16); $0.centerY.equalToSuperview(); $0.width.height.equalTo(30) }
        labels.snp.makeConstraints { $0.leading.equalTo(icon.snp.trailing).offset(12); $0.top.bottom.equalToSuperview().inset(16); $0.trailing.lessThanOrEqualTo(toggle.snp.leading).offset(-10) }
        toggle.snp.makeConstraints { $0.trailing.equalToSuperview().inset(16); $0.centerY.equalToSuperview() }
        return card
    }

    private func loadSettings() {
        settings.remoteLoginAlert = defaults.object(forKey: remoteKey) == nil ? true : defaults.bool(forKey: remoteKey)
        settings.loginVerification = defaults.bool(forKey: verificationKey)
        applySettings()
        Task {
            do {
                settings = try await LinLanAppContainer.apiClient().fetchSecuritySettings()
                cacheAndApply()
            } catch {
                await MainActor.run { self.toast("已显示本机缓存，云端设置加载失败") }
            }
        }
    }

    @objc private func remoteChanged() { save(remote: remoteSwitch.isOn, verification: nil) }
    @objc private func verificationChanged() { save(remote: nil, verification: verificationSwitch.isOn) }

    private func save(remote: Bool?, verification: Bool?) {
        guard !saving else { applySettings(); return }
        saving = true
        setSwitchesEnabled(false)
        Task {
            do {
                settings = try await LinLanAppContainer.apiClient().updateSecuritySettings(remoteLoginAlert: remote, loginVerification: verification)
                cacheAndApply()
                await MainActor.run { self.toast(remote != nil ? (self.settings.remoteLoginAlert ? "已开启异地登录提醒" : "已关闭异地登录提醒") : (self.settings.loginVerification ? "已开启新设备登录验证" : "已关闭新设备登录验证")) }
            } catch {
                await MainActor.run { self.applySettings(); self.toast(error.localizedDescription) }
            }
            await MainActor.run { self.saving = false; self.setSwitchesEnabled(true) }
        }
    }

    @MainActor private func cacheAndApply() {
        defaults.set(settings.remoteLoginAlert, forKey: remoteKey)
        defaults.set(settings.loginVerification, forKey: verificationKey)
        applySettings()
    }

    private func applySettings() { remoteSwitch.isOn = settings.remoteLoginAlert; verificationSwitch.isOn = settings.loginVerification }
    private func setSwitchesEnabled(_ enabled: Bool) { remoteSwitch.isEnabled = enabled; verificationSwitch.isEnabled = enabled }
    private func toast(_ text: String) { let a = UIAlertController(title: nil, message: text, preferredStyle: .alert); present(a, animated: true); DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) } }
}
