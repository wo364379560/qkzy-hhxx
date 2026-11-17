//
//  LinLanEditProfileViewController.swift
//  TodayToTrackRun
//
//  对照睿之禾 EditProfileScreen：睿之禾号 / 改资料 / 上传头像
//

import PhotosUI
import SnapKit
import UIKit

final class LinLanEditProfileViewController: UIViewController, UIGestureRecognizerDelegate {
    private let scroll = UIScrollView()
    private let stack = UIStackView()

    private let headerCard = UIView()
    private let avatarView = UIImageView()
    private let headerName = UILabel()
    private let headerAt = UILabel()
    private let avatarHint = UILabel()

    private let nicknameValue = UILabel()
    private let signalValue = UILabel()
    private let phoneValue = UILabel()
    private let tipLabel = UILabel()

    private var profile = LinLanChatProfile()
    private var uploading = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "个人设置"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "保存",
            style: .done,
            target: self,
            action: #selector(saveNickname)
        )
        navigationItem.rightBarButtonItem?.tintColor = AppTheme.accent

        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        scroll.keyboardDismissMode = .onDrag
        stack.axis = .vertical
        stack.spacing = 14
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 32, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        buildHeader()
        buildInfoCard()

        tipLabel.font = .systemFont(ofSize: 13)
        tipLabel.textColor = AppTheme.accent
        tipLabel.numberOfLines = 0
        tipLabel.isHidden = true
        stack.addArrangedSubview(tipLabel)

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        tap.delegate = self
        view.addGestureRecognizer(tap)

        Task { await reload() }
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldReceive touch: UITouch) -> Bool {
        // 点在控件上时不抢手势，避免影响点击
        if touch.view is UIControl || touch.view is UITextField { return false }
        return true
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await reload() }
    }

    // MARK: - UI

    private func buildHeader() {
        headerCard.backgroundColor = AppTheme.accent
        headerCard.layer.cornerRadius = 16
        headerCard.clipsToBounds = true
        headerCard.isUserInteractionEnabled = true
        headerCard.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(pickAvatar)))

        avatarView.backgroundColor = UIColor.white.withAlphaComponent(0.25)
        avatarView.layer.cornerRadius = 16
        avatarView.clipsToBounds = true
        avatarView.contentMode = .scaleAspectFill
        avatarView.snp.makeConstraints { $0.size.equalTo(64) }

        headerName.font = .systemFont(ofSize: 22, weight: .bold)
        headerName.textColor = .white
        headerAt.font = .systemFont(ofSize: 14)
        headerAt.textColor = UIColor.white.withAlphaComponent(0.85)
        headerAt.text = "@"
        avatarHint.font = .systemFont(ofSize: 13, weight: .medium)
        avatarHint.textColor = UIColor.white.withAlphaComponent(0.95)
        avatarHint.text = "点击修改头像"

        let col = UIStackView(arrangedSubviews: [headerName, headerAt, avatarHint])
        col.axis = .vertical
        col.spacing = 4
        col.alignment = .leading

        headerCard.addSubview(avatarView)
        headerCard.addSubview(col)
        avatarView.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.top.bottom.equalToSuperview().inset(18)
        }
        col.snp.makeConstraints { make in
            make.leading.equalTo(avatarView.snp.trailing).offset(14)
            make.centerY.equalToSuperview()
            make.trailing.equalToSuperview().inset(16)
        }
        stack.addArrangedSubview(headerCard)
    }

    private func buildInfoCard() {
        let card = AppTheme.makeCard()
        let list = UIStackView()
        list.axis = .vertical
        card.addSubview(list)
        list.snp.makeConstraints { $0.edges.equalToSuperview() }

        list.addArrangedSubview(infoRow(title: "昵称", valueLabel: nicknameValue, action: #selector(editNickname)))
        list.addArrangedSubview(divider())
        list.addArrangedSubview(infoRow(title: "睿之禾号", valueLabel: signalValue, action: #selector(editSignal)))
        list.addArrangedSubview(divider())
        list.addArrangedSubview(infoRow(title: "手机号", valueLabel: phoneValue, action: #selector(copyPhone), showChevron: false))
        stack.addArrangedSubview(card)
    }

    private func infoRow(
        title: String,
        valueLabel: UILabel,
        action: Selector,
        showChevron: Bool = true
    ) -> UIView {
        let row = UIControl()
        row.addTarget(self, action: action, for: .touchUpInside)
        row.snp.makeConstraints { $0.height.equalTo(52) }

        let titleL = UILabel()
        titleL.text = title
        titleL.font = .systemFont(ofSize: 16, weight: .medium)
        titleL.textColor = AppTheme.textPrimary

        valueLabel.font = .systemFont(ofSize: 15)
        valueLabel.textColor = AppTheme.textSecondary
        valueLabel.textAlignment = .right
        valueLabel.lineBreakMode = .byTruncatingMiddle

        let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))
        chevron.tintColor = UIColor.tertiaryLabel
        chevron.isHidden = !showChevron
        chevron.setContentHuggingPriority(.required, for: .horizontal)

        row.addSubview(titleL)
        row.addSubview(valueLabel)
        row.addSubview(chevron)
        titleL.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
        }
        chevron.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(14)
            make.centerY.equalToSuperview()
            make.width.equalTo(showChevron ? 12 : 0)
        }
        valueLabel.snp.makeConstraints { make in
            make.trailing.equalTo(chevron.snp.leading).offset(showChevron ? -6 : 0)
            make.centerY.equalToSuperview()
            make.leading.greaterThanOrEqualTo(titleL.snp.trailing).offset(12)
        }
        return row
    }

    private func divider() -> UIView {
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

    // MARK: - Data

    private func reload() async {
        do {
            _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
            let p = try await LinLanAppContainer.apiClient().fetchProfile()
            profile = p
            applyUI()
        } catch {
            showTip(error.localizedDescription)
        }
    }

    private func applyUI() {
        let name = profile.nickname.isEmpty ? "未设置昵称" : profile.nickname
        headerName.text = name
        headerAt.text = profile.chuanluId.isEmpty ? "@" : "@\(profile.chuanluId)"
        nicknameValue.text = profile.nickname.isEmpty ? "未设置" : profile.nickname
        signalValue.text = profile.chuanluId.isEmpty ? "未设置" : profile.chuanluId
        signalValue.textColor = profile.chuanluId.isEmpty ? AppTheme.accent : AppTheme.textSecondary
        let phone = profile.phone.isEmpty ? LinLanAppContainer.appPrefs().accountPhone : profile.phone
        phoneValue.text = phone.isEmpty ? "未获取" : phone

        LinLanImageLoader.load(
            urlString: profile.avatarUrl,
            into: avatarView,
            placeholderLetter: String(name.prefix(1))
        )
        avatarHint.text = uploading ? "上传中…" : "点击修改头像"
    }

    private func showTip(_ text: String?) {
        tipLabel.text = text
        tipLabel.isHidden = (text == nil || text?.isEmpty == true)
    }

    private static func isValidSignalId(_ raw: String) -> Bool {
        raw.range(of: #"^[a-z][a-z0-9_-]{5,19}$"#, options: .regularExpression) != nil
    }

    // MARK: - Actions

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func editNickname() {
        let a = UIAlertController(title: "修改昵称", message: nil, preferredStyle: .alert)
        a.addTextField {
            $0.text = self.profile.nickname
            $0.placeholder = "请输入昵称"
        }
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "确定", style: .default) { [weak self] _ in
            guard let self else { return }
            let text = a.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !text.isEmpty else {
                self.showTip("请输入昵称")
                return
            }
            self.profile.nickname = text
            self.applyUI()
        })
        present(a, animated: true)
    }

    @objc private func saveNickname() {
        let name = profile.nickname.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else {
            showTip("请先设置昵称")
            return
        }
        Task {
            do {
                try await LinLanAppContainer.apiClient().updateProfile(nickname: name)
                await LinLanImManager.syncSelfProfile(nickname: name, faceUrl: nil)
                await MainActor.run { self.showTip("已保存") }
            } catch {
                await MainActor.run { self.showTip(error.localizedDescription) }
            }
        }
    }

    @objc private func editSignal() {
        let a = UIAlertController(
            title: profile.chuanluId.isEmpty ? "设置睿之禾号" : "修改睿之禾号",
            message: "6-20 位，以字母开头，可包含数字、下划线或短横线",
            preferredStyle: .alert
        )
        a.addTextField {
            $0.text = self.profile.chuanluId
            $0.placeholder = "睿之禾号"
            $0.autocapitalizationType = .none
            $0.autocorrectionType = .no
        }
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "确定", style: .default) { [weak self] _ in
            guard let self else { return }
            var value = (a.textFields?.first?.text ?? "")
                .lowercased()
                .filter { ($0 >= "a" && $0 <= "z") || ($0 >= "0" && $0 <= "9") || $0 == "_" || $0 == "-" }
            if value.count > 20 { value = String(value.prefix(20)) }
            guard !value.isEmpty else {
                self.showTip("请输入睿之禾号")
                return
            }
            guard Self.isValidSignalId(value) else {
                self.showTip("睿之禾号格式不正确")
                return
            }
            if value == self.profile.chuanluId { return }
            Task {
                do {
                    try await LinLanAppContainer.apiClient().updateProfile(wananId: value)
                    await MainActor.run {
                        self.profile.chuanluId = value
                        self.applyUI()
                        self.showTip("睿之禾号已更新")
                    }
                } catch {
                    await MainActor.run { self.showTip(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }

    @objc private func copyPhone() {
        let text = profile.phone.isEmpty ? LinLanAppContainer.appPrefs().accountPhone : profile.phone
        guard !text.isEmpty else { return }
        UIPasteboard.general.string = text
        showTip("已复制手机号")
    }

    @objc private func pickAvatar() {
        guard !uploading else { return }
        var config = PHPickerConfiguration(photoLibrary: .shared())
        config.filter = .images
        config.selectionLimit = 1
        let picker = PHPickerViewController(configuration: config)
        picker.delegate = self
        present(picker, animated: true)
    }

    private func uploadAvatar(image: UIImage) {
        guard let data = image.jpegData(compressionQuality: 0.85) else {
            showTip("图片处理失败")
            return
        }
        let path = FileManager.default.temporaryDirectory
            .appendingPathComponent("uavatar_\(UUID().uuidString).jpg")
        do {
            try data.write(to: path)
        } catch {
            showTip("保存临时文件失败")
            return
        }
        uploading = true
        applyUI()
        Task {
            do {
                var uid = profile.imUserId
                if uid.isEmpty {
                    uid = try await LinLanAppContainer.apiClient().fetchProfile().imUserId
                }
                guard !uid.isEmpty else { throw LinLanAPIError.message("用户未登录") }
                let url = try await LinLanAppContainer.apiClient().uploadUserAvatar(
                    userId: uid,
                    filePath: path.path
                )
                try await LinLanAppContainer.apiClient().updateProfile(avatarUrl: url)
                await LinLanImManager.syncSelfProfile(nickname: nil, faceUrl: url)
                await MainActor.run {
                    self.profile.avatarUrl = url
                    self.uploading = false
                    self.applyUI()
                    self.showTip("头像已更新")
                }
            } catch {
                await MainActor.run {
                    self.uploading = false
                    self.applyUI()
                    self.showTip(error.localizedDescription)
                }
            }
        }
    }
}

extension LinLanEditProfileViewController: PHPickerViewControllerDelegate {
    func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
        picker.dismiss(animated: true)
        guard let provider = results.first?.itemProvider, provider.canLoadObject(ofClass: UIImage.self) else { return }
        provider.loadObject(ofClass: UIImage.self) { [weak self] obj, _ in
            guard let self, let image = obj as? UIImage else { return }
            DispatchQueue.main.async { self.uploadAvatar(image: image) }
        }
    }
}
