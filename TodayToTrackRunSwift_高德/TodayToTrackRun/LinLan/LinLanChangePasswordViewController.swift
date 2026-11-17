//
//  LinLanChangePasswordViewController.swift
//  TodayToTrackRun
//
//  对照 Android LinLanChangePasswordScreen → POST /password/update
//

import SnapKit
import UIKit

final class LinLanChangePasswordViewController: UIViewController {
    private let currentField = LinLanAuthForm.textField(placeholder: "当前密码", secure: true)
    private let nextField = LinLanAuthForm.textField(placeholder: "新密码（至少 8 位）", secure: true)
    private let confirmField = LinLanAuthForm.textField(placeholder: "确认新密码", secure: true)
    private let messageLabel = UILabel()
    private let submitButton = LinLanAuthForm.primaryButton(title: "确认修改")
    private var submitting = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "修改密码"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )

        messageLabel.font = .systemFont(ofSize: 13)
        messageLabel.textColor = .systemRed
        messageLabel.numberOfLines = 0
        messageLabel.isHidden = true

        let tip = UILabel()
        tip.text = "修改成功后请使用新密码登录。"
        tip.font = .systemFont(ofSize: 13)
        tip.textColor = AppTheme.textSecondary
        tip.numberOfLines = 0

        let stack = UIStackView(arrangedSubviews: [
            tip, currentField, nextField, confirmField, messageLabel, submitButton,
        ])
        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 24, left: 20, bottom: 24, right: 20)
        view.addSubview(stack)
        stack.snp.makeConstraints { $0.top.leading.trailing.equalTo(view.safeAreaLayoutGuide) }

        submitButton.addTarget(self, action: #selector(submit), for: .touchUpInside)

        let tap = UITapGestureRecognizer(target: self, action: #selector(endEdit))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }
    @objc private func endEdit() { view.endEditing(true) }

    @objc private func submit() {
        view.endEditing(true)
        guard !submitting else { return }

        let current = currentField.text ?? ""
        let next = nextField.text ?? ""
        let confirm = confirmField.text ?? ""

        if current.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            showMessage("请输入当前密码", ok: false)
            return
        }
        if next.count < 8 {
            showMessage("新密码至少 8 位", ok: false)
            return
        }
        if next != confirm {
            showMessage("两次密码不一致", ok: false)
            return
        }
        if next == current {
            showMessage("新密码不能与当前密码相同", ok: false)
            return
        }

        submitting = true
        submitButton.isEnabled = false
        submitButton.setTitle("提交中...", for: .normal)
        messageLabel.isHidden = true

        Task {
            do {
                try await LinLanAppContainer.authRepository().updatePassword(
                    currentPassword: current,
                    newPassword: next
                )
                await MainActor.run {
                    self.submitting = false
                    self.submitButton.isEnabled = true
                    self.submitButton.setTitle("确认修改", for: .normal)
                    self.showMessage("密码已修改", ok: true)
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) { [weak self] in
                        self?.navigationController?.popViewController(animated: true)
                    }
                }
            } catch {
                await MainActor.run {
                    self.submitting = false
                    self.submitButton.isEnabled = true
                    self.submitButton.setTitle("确认修改", for: .normal)
                    self.showMessage(error.localizedDescription, ok: false)
                }
            }
        }
    }

    private func showMessage(_ text: String, ok: Bool) {
        messageLabel.text = text
        messageLabel.textColor = ok ? AppTheme.accent : .systemRed
        messageLabel.isHidden = false
    }
}
