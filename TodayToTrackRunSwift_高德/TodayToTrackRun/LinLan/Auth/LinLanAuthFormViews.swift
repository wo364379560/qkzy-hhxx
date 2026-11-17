//
//  LinLanAuthFormViews.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

enum LinLanAuthForm {
    static func textField(placeholder: String, secure: Bool = false) -> UITextField {
        let tf = UITextField()
        tf.placeholder = placeholder
        tf.borderStyle = .none
        tf.backgroundColor = UIColor(white: 1, alpha: 0.92)
        tf.layer.cornerRadius = 14
        tf.layer.borderWidth = 1
        tf.layer.borderColor = AppTheme.divider.cgColor
        tf.font = .systemFont(ofSize: 16)
        tf.textColor = AppTheme.textPrimary
        tf.isSecureTextEntry = secure
        tf.autocapitalizationType = .none
        tf.autocorrectionType = .no
        tf.returnKeyType = .done
        tf.leftView = UIView(frame: CGRect(x: 0, y: 0, width: 14, height: 1))
        tf.leftViewMode = .always
        tf.rightView = UIView(frame: CGRect(x: 0, y: 0, width: 14, height: 1))
        tf.rightViewMode = .always
        tf.snp.makeConstraints { $0.height.equalTo(50) }
        return tf
    }

    /// 数字键盘等无 Return 键时用工具条收起
    static func attachDoneToolbar(to fields: [UITextField], target: Any?, action: Selector) {
        let bar = UIToolbar(frame: CGRect(x: 0, y: 0, width: UIScreen.main.bounds.width, height: 44))
        bar.items = [
            UIBarButtonItem(barButtonSystemItem: .flexibleSpace, target: nil, action: nil),
            UIBarButtonItem(title: "完成", style: .done, target: target, action: action)
        ]
        bar.items?.last?.tintColor = AppTheme.accent
        fields.forEach { $0.inputAccessoryView = bar }
    }

    static func installTapToDismiss(on view: UIView, target: Any?, action: Selector) {
        let tap = UITapGestureRecognizer(target: target, action: action)
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)
    }

    static func primaryButton(title: String) -> UIButton {
        let b = UIButton(type: .system)
        b.setTitle(title, for: .normal)
        b.setTitleColor(.white, for: .normal)
        b.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        b.backgroundColor = AppTheme.accent
        b.layer.cornerRadius = 16
        b.snp.makeConstraints { $0.height.equalTo(52) }
        return b
    }

    static func linkButton(title: String, color: UIColor = AppTheme.accent) -> UIButton {
        let b = UIButton(type: .system)
        b.setTitle(title, for: .normal)
        b.setTitleColor(color, for: .normal)
        b.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
        return b
    }
}

final class LinLanRememberPasswordRow: UIView {
    var isOn: Bool { rememberSwitch.isOn }
    var onChanged: ((Bool) -> Void)?

    private let rememberSwitch = UISwitch()

    override init(frame: CGRect) {
        super.init(frame: frame)

        let label = UILabel()
        label.text = "记住密码"
        label.font = .systemFont(ofSize: 14, weight: .medium)
        label.textColor = AppTheme.textSecondary

        rememberSwitch.onTintColor = AppTheme.accent
        rememberSwitch.transform = CGAffineTransform(scaleX: 0.82, y: 0.82)
        rememberSwitch.addTarget(self, action: #selector(valueChanged), for: .valueChanged)

        let stack = UIStackView(arrangedSubviews: [label, UIView(), rememberSwitch])
        stack.axis = .horizontal
        stack.alignment = .center
        addSubview(stack)
        stack.snp.makeConstraints { make in
            make.top.bottom.equalToSuperview()
            make.leading.trailing.equalToSuperview().inset(2)
        }
        snp.makeConstraints { $0.height.equalTo(34) }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError() }

    func setOn(_ on: Bool) {
        rememberSwitch.setOn(on, animated: false)
    }

    @objc private func valueChanged() {
        onChanged?(rememberSwitch.isOn)
    }
}

final class LinLanSmsCodeRow: UIView {
    let codeField = LinLanAuthForm.textField(placeholder: "验证码")
    private let sendButton = UIButton(type: .system)
    private var seconds = 0
    private var timer: Timer?
    var phoneProvider: (() -> String)?
    var onMessage: ((String) -> Void)?

    override init(frame: CGRect) {
        super.init(frame: frame)
        codeField.keyboardType = .numberPad

        sendButton.setTitle("获取验证码", for: .normal)
        sendButton.setTitleColor(AppTheme.accent, for: .normal)
        sendButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        sendButton.contentEdgeInsets = UIEdgeInsets(top: 0, left: 10, bottom: 0, right: 10)
        sendButton.addTarget(self, action: #selector(sendTapped), for: .touchUpInside)

        let stack = UIStackView(arrangedSubviews: [codeField, sendButton])
        stack.axis = .horizontal
        stack.spacing = 10
        stack.alignment = .center
        addSubview(stack)
        stack.snp.makeConstraints { $0.edges.equalToSuperview() }
        sendButton.snp.makeConstraints { $0.width.greaterThanOrEqualTo(96) }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError() }

    deinit { timer?.invalidate() }

    @objc private func sendTapped() {
        let phone = phoneProvider?().trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !phone.isEmpty else {
            onMessage?("请先填写手机号")
            return
        }
        guard seconds == 0 else { return }
        sendButton.isEnabled = false
        Task {
            do {
                try await LinLanAppContainer.authRepository().sendSms(phone: phone)
                await MainActor.run {
                    self.onMessage?("验证码已发送")
                    self.startCountdown()
                }
            } catch {
                await MainActor.run {
                    self.sendButton.isEnabled = true
                    self.onMessage?(error.localizedDescription)
                }
            }
        }
    }

    private func startCountdown() {
        seconds = 60
        sendButton.isEnabled = false
        sendButton.setTitle("\(seconds)s", for: .normal)
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] t in
            guard let self else { t.invalidate(); return }
            self.seconds -= 1
            if self.seconds <= 0 {
                t.invalidate()
                self.sendButton.isEnabled = true
                self.sendButton.setTitle("获取验证码", for: .normal)
            } else {
                self.sendButton.setTitle("\(self.seconds)s", for: .normal)
            }
        }
    }
}

final class LinLanLegalCheckRow: UIView {
    var isChecked: Bool { check.isOn }
    var onOpenAgreement: (() -> Void)?
    var onOpenPrivacy: (() -> Void)?

    private let check = UISwitch()

    override init(frame: CGRect) {
        super.init(frame: frame)
        check.onTintColor = AppTheme.accent
        check.transform = CGAffineTransform(scaleX: 0.75, y: 0.75)

        let prefix = UILabel()
        prefix.text = "同意"
        prefix.font = .systemFont(ofSize: 13)
        prefix.textColor = AppTheme.textSecondary

        let agreement = UIButton(type: .system)
        agreement.setTitle("《用户协议》", for: .normal)
        agreement.setTitleColor(AppTheme.accent, for: .normal)
        agreement.titleLabel?.font = .systemFont(ofSize: 13, weight: .medium)
        agreement.addAction(UIAction { [weak self] _ in self?.onOpenAgreement?() }, for: .touchUpInside)

        let and = UILabel()
        and.text = "与"
        and.font = .systemFont(ofSize: 13)
        and.textColor = AppTheme.textSecondary

        let privacy = UIButton(type: .system)
        privacy.setTitle("《隐私政策》", for: .normal)
        privacy.setTitleColor(AppTheme.accent, for: .normal)
        privacy.titleLabel?.font = .systemFont(ofSize: 13, weight: .medium)
        privacy.addAction(UIAction { [weak self] _ in self?.onOpenPrivacy?() }, for: .touchUpInside)

        let textRow = UIStackView(arrangedSubviews: [prefix, agreement, and, privacy, UIView()])
        textRow.axis = .horizontal
        textRow.alignment = .center
        textRow.spacing = 0

        let row = UIStackView(arrangedSubviews: [check, textRow])
        row.axis = .horizontal
        row.alignment = .center
        row.spacing = 4
        addSubview(row)
        row.snp.makeConstraints { $0.edges.equalToSuperview() }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError() }
}
