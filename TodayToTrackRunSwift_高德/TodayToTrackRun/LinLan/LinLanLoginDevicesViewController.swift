import SnapKit
import UIKit

final class LinLanLoginDevicesViewController: UIViewController {
    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let loading = UIActivityIndicatorView(style: .medium)
    private var devices: [LinLanLoginDevice] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "登录设备管理"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.rightBarButtonItem = UIBarButtonItem(image: UIImage(systemName: "arrow.clockwise"), style: .plain, target: self, action: #selector(refresh))
        setupUI()
        loadDevices()
    }

    private func setupUI() {
        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 28, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { $0.edges.width.equalToSuperview() }

        let info = UILabel()
        info.text = "以下为云端记录的登录设备（含 Web）。可将其他设备远程退出；当前设备会标记为「本机」。"
        info.font = .systemFont(ofSize: 14)
        info.textColor = AppTheme.textSecondary
        info.numberOfLines = 0
        let infoCard = AppTheme.makeCard()
        infoCard.addSubview(info)
        info.snp.makeConstraints { $0.edges.equalToSuperview().inset(18) }
        stack.addArrangedSubview(infoCard)
        stack.addArrangedSubview(loading)
    }

    @objc private func refresh() { loadDevices() }

    private func loadDevices() {
        loading.startAnimating()
        navigationItem.rightBarButtonItem?.isEnabled = false
        Task {
            do {
                devices = try await LinLanAppContainer.apiClient().fetchLoginDevices()
                await MainActor.run { self.renderDevices() }
            } catch {
                await MainActor.run { self.renderError(error.localizedDescription) }
            }
        }
    }

    private func clearDeviceViews() {
        stack.arrangedSubviews.dropFirst(2).forEach { $0.removeFromSuperview() }
        loading.stopAnimating()
        navigationItem.rightBarButtonItem?.isEnabled = true
    }

    private func renderDevices() {
        clearDeviceViews()
        guard !devices.isEmpty else {
            stack.addArrangedSubview(messageCard("暂无设备记录"))
            return
        }
        let online = devices.filter(\.isOnline)
        if online.isEmpty {
            stack.addArrangedSubview(messageCard("暂无在线设备"))
        } else {
            addSection("当前在线", devices: online)
        }
    }

    private func renderError(_ text: String) {
        clearDeviceViews()
        stack.addArrangedSubview(messageCard("加载失败：\(text)\n请稍后点击右上角刷新"))
    }

    private func addSection(_ title: String, devices: [LinLanLoginDevice]) {
        let label = UILabel()
        label.text = title
        label.font = .systemFont(ofSize: 13, weight: .medium)
        label.textColor = AppTheme.textSecondary
        stack.addArrangedSubview(label)
        devices.forEach { stack.addArrangedSubview(deviceCard($0)) }
    }

    private func deviceCard(_ device: LinLanLoginDevice) -> UIView {
        let card = AppTheme.makeCard()
        let icon = UIImageView(image: UIImage(systemName: device.isWeb ? "globe" : "iphone"))
        icon.tintColor = AppTheme.accent
        icon.contentMode = .scaleAspectFit
        let title = UILabel()
        title.text = device.deviceName + (device.isCurrent ? "  · 本机" : "")
        title.font = .systemFont(ofSize: 16, weight: .semibold)
        title.textColor = AppTheme.textPrimary
        let platform = device.platform.isEmpty ? "设备" : device.platform
        let time = formattedTime(device.lastLoginAt)
        let detail = UILabel()
        detail.text = "\(platform)\(device.isOnline ? " · 在线" : "")\n最近登录：\(time)\nIP：\(device.ip.isEmpty ? "—" : device.ip)"
        detail.font = .systemFont(ofSize: 12)
        detail.textColor = AppTheme.textSecondary
        detail.numberOfLines = 3
        let labels = UIStackView(arrangedSubviews: [title, detail])
        labels.axis = .vertical
        labels.spacing = 5
        let action = UIButton(type: .system)
        action.setImage(UIImage(systemName: device.isCurrent ? "rectangle.portrait.and.arrow.right" : "trash"), for: .normal)
        action.tintColor = device.isCurrent ? AppTheme.textSecondary : .systemRed
        action.addAction(UIAction { [weak self] _ in self?.confirmRemove(device) }, for: .touchUpInside)
        card.addSubview(icon); card.addSubview(labels); card.addSubview(action)
        icon.snp.makeConstraints { $0.leading.equalToSuperview().offset(16); $0.centerY.equalToSuperview(); $0.width.height.equalTo(34) }
        labels.snp.makeConstraints { $0.leading.equalTo(icon.snp.trailing).offset(14); $0.top.bottom.equalToSuperview().inset(16); $0.trailing.lessThanOrEqualTo(action.snp.leading).offset(-8) }
        action.snp.makeConstraints { $0.trailing.equalToSuperview().inset(16); $0.centerY.equalToSuperview(); $0.width.height.equalTo(40) }
        return card
    }

    private func confirmRemove(_ device: LinLanLoginDevice) {
        guard !device.deviceId.isEmpty else { toast("该设备记录无法远程退出"); return }
        let alert = UIAlertController(
            title: device.isCurrent ? "退出本机登录" : "远程退出设备",
            message: device.isCurrent ? "确定退出本机吗？退出后需要重新登录。" : "确定让「\(device.deviceName)」下线吗？",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .destructive) { [weak self] _ in self?.remove(device) })
        present(alert, animated: true)
    }

    private func remove(_ device: LinLanLoginDevice) {
        Task {
            if device.isCurrent {
                await LinLanAppContainer.authRepository().logout()
                await MainActor.run { LinLanRootSwitcher.showLogin() }
                return
            }
            do {
                try await LinLanAppContainer.apiClient().revokeLoginDevice(deviceId: device.deviceId)
                await MainActor.run { self.toast("已远程退出该设备"); self.loadDevices() }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func formattedTime(_ raw: String) -> String {
        guard !raw.isEmpty else { return "—" }
        let iso = ISO8601DateFormatter()
        guard let date = iso.date(from: raw) ?? iso.date(from: raw.replacingOccurrences(of: " ", with: "T") + "Z") else { return raw }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter.string(from: date)
    }

    private func messageCard(_ text: String) -> UIView {
        let card = AppTheme.makeCard()
        let label = UILabel()
        label.text = text; label.font = .systemFont(ofSize: 14); label.textColor = AppTheme.textSecondary; label.textAlignment = .center; label.numberOfLines = 0
        card.addSubview(label); label.snp.makeConstraints { $0.edges.equalToSuperview().inset(20) }
        return card
    }

    private func toast(_ text: String) { let a = UIAlertController(title: nil, message: text, preferredStyle: .alert); present(a, animated: true); DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) } }
}
