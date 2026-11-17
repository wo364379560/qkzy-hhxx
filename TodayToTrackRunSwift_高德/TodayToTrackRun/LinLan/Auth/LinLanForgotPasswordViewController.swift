//
//  LinLanForgotPasswordViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanForgotPasswordViewController: UIViewController, UITextFieldDelegate {
    private let auth = LinLanAppContainer.authRepository()

    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let phoneField = LinLanAuthForm.textField(placeholder: "手机号")
    private let codeRow = LinLanSmsCodeRow()
    private let passwordField = LinLanAuthForm.textField(placeholder: "新密码（至少 6 位）", secure: true)
    private let messageLabel = UILabel()
    private let submitButton = LinLanAuthForm.primaryButton(title: "确认重置")

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "重置密码"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationController?.setNavigationBarHidden(false, animated: false)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        navigationController?.navigationBar.tintColor = AppTheme.accent

        phoneField.keyboardType = .phonePad
        phoneField.delegate = self
        passwordField.delegate = self
        codeRow.codeField.delegate = self

        view.addSubview(scroll)
        scroll.keyboardDismissMode = .onDrag
        scroll.alwaysBounceVertical = true
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        stack.axis = .vertical
        stack.spacing = 14
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 28, left: 24, bottom: 40, right: 24)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        let titleL = UILabel()
        titleL.text = "找回密码"
        titleL.font = .systemFont(ofSize: 28, weight: .bold)
        titleL.textColor = AppTheme.textPrimary

        let tip = UILabel()
        tip.text = "验证手机号后设置新密码，完成后返回登录即可。"
        tip.font = .systemFont(ofSize: 14)
        tip.textColor = AppTheme.textSecondary
        tip.numberOfLines = 0

        messageLabel.font = .systemFont(ofSize: 13)
        messageLabel.textColor = AppTheme.accent
        messageLabel.numberOfLines = 0
        messageLabel.isHidden = true

        codeRow.phoneProvider = { [weak self] in self?.phoneField.text ?? "" }
        codeRow.onMessage = { [weak self] msg in self?.showMessage(msg, ok: false) }
        submitButton.addTarget(self, action: #selector(submit), for: .touchUpInside)

        stack.addArrangedSubview(titleL)
        stack.addArrangedSubview(tip)
        stack.setCustomSpacing(24, after: tip)
        stack.addArrangedSubview(phoneField)
        stack.addArrangedSubview(codeRow)
        stack.addArrangedSubview(passwordField)
        stack.addArrangedSubview(messageLabel)
        stack.addArrangedSubview(submitButton)

        LinLanAuthForm.attachDoneToolbar(
            to: [phoneField, passwordField, codeRow.codeField],
            target: self,
            action: #selector(dismissKeyboard)
        )
        LinLanAuthForm.installTapToDismiss(on: view, target: self, action: #selector(dismissKeyboard))
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
    }

    @objc private func back() {
        dismissKeyboard()
        navigationController?.popViewController(animated: true)
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    @objc private func submit() {
        dismissKeyboard()
        let phone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let code = codeRow.codeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text ?? ""
        guard !phone.isEmpty, !code.isEmpty, !password.isEmpty else {
            showMessage("请填写完整信息", ok: false)
            return
        }
        guard password.count >= 6 else {
            showMessage("密码长度不能少于 6 位", ok: false)
            return
        }
        submitButton.isEnabled = false
        Task {
            do {
                try await self.auth.resetPassword(phone: phone, code: code, newPassword: password)
                await MainActor.run {
                    self.showMessage("密码已重置，请返回登录", ok: true)
                    self.submitButton.isEnabled = true
                }
            } catch {
                await MainActor.run {
                    self.showMessage(error.localizedDescription, ok: false)
                    self.submitButton.isEnabled = true
                }
            }
        }
    }

    private func showMessage(_ text: String, ok: Bool) {
        messageLabel.text = text
        messageLabel.textColor = ok ? AppTheme.accent : .systemRed
        messageLabel.isHidden = false
    }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}
