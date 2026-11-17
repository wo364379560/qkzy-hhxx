//
//  LinLanRegisterViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanRegisterViewController: UIViewController, UITextFieldDelegate {
    private let prefs = LinLanAppContainer.appPrefs()
    private let auth = LinLanAppContainer.authRepository()

    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let phoneField = LinLanAuthForm.textField(placeholder: "手机号")
    private let codeRow = LinLanSmsCodeRow()
    private let nicknameField = LinLanAuthForm.textField(placeholder: "昵称")
    private let passwordField = LinLanAuthForm.textField(placeholder: "密码（至少 6 位）", secure: true)
    private let inviteField = LinLanAuthForm.textField(placeholder: "邀请码（选填）")
    private let errorLabel = UILabel()
    private let legalRow = LinLanLegalCheckRow()
    private let submitButton = LinLanAuthForm.primaryButton(title: "创建账号")

    private var privacyDialogAccepted = false
    private var loading = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "注册"
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

        privacyDialogAccepted = prefs.privacyAccepted
        phoneField.keyboardType = .phonePad
        [phoneField, nicknameField, passwordField, inviteField, codeRow.codeField].forEach {
            $0.delegate = self
        }
        setupUI()
        LinLanAuthForm.attachDoneToolbar(
            to: [phoneField, nicknameField, passwordField, inviteField, codeRow.codeField],
            target: self,
            action: #selector(dismissKeyboard)
        )
        LinLanAuthForm.installTapToDismiss(on: view, target: self, action: #selector(dismissKeyboard))
        if !prefs.privacyAccepted {
            presentPrivacyGate()
        }
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

    private func setupUI() {
        view.addSubview(scroll)
        scroll.keyboardDismissMode = .onDrag
        scroll.alwaysBounceVertical = true
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 20, left: 24, bottom: 40, right: 24)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        let brand = UILabel()
        brand.text = "开启新连接"
        brand.font = .systemFont(ofSize: 28, weight: .bold)
        brand.textColor = AppTheme.textPrimary

        let tip = UILabel()
        tip.text = "填写资料创建账号，即可开始使用。"
        tip.font = .systemFont(ofSize: 14)
        tip.textColor = AppTheme.textSecondary
        tip.numberOfLines = 0

        errorLabel.font = .systemFont(ofSize: 13)
        errorLabel.textColor = .systemRed
        errorLabel.numberOfLines = 0
        errorLabel.isHidden = true

        codeRow.phoneProvider = { [weak self] in self?.phoneField.text ?? "" }
        codeRow.onMessage = { [weak self] msg in self?.showError(msg) }
        legalRow.onOpenAgreement = { [weak self] in
            self?.openLegal(title: "用户协议", url: LinLanAppConstants.userAgreementURL)
        }
        legalRow.onOpenPrivacy = { [weak self] in
            self?.openLegal(title: "隐私政策", url: LinLanAppConstants.privacyPolicyURL)
        }
        submitButton.addTarget(self, action: #selector(submit), for: .touchUpInside)

        stack.addArrangedSubview(brand)
        stack.addArrangedSubview(tip)
        stack.setCustomSpacing(20, after: tip)
        stack.addArrangedSubview(phoneField)
        stack.addArrangedSubview(codeRow)
        stack.addArrangedSubview(nicknameField)
        stack.addArrangedSubview(passwordField)
        //stack.addArrangedSubview(inviteField)
        stack.addArrangedSubview(errorLabel)
        stack.addArrangedSubview(legalRow)
        stack.addArrangedSubview(submitButton)
    }

    private func canSubmit() -> Bool {
        if prefs.privacyAccepted { return true }
        return privacyDialogAccepted && legalRow.isChecked
    }

    @objc private func submit() {
        dismissKeyboard()
        let phone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let nick = nicknameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text ?? ""
        let code = codeRow.codeField.text?.trimmingCharacters(in: .whitespacesAndNewlines)
        let invite = inviteField.text?.trimmingCharacters(in: .whitespacesAndNewlines)

        if phone.isEmpty { showError("请填写手机号"); return }
        if nick.isEmpty { showError("请填写昵称"); return }
        if password.isEmpty { showError("请填写密码"); return }
        if password.count < 6 { showError("密码长度不能少于6位"); return }
        guard canSubmit() else { showError("请先同意用户协议与隐私政策"); return }
        guard !loading else { return }

        loading = true
        submitButton.isEnabled = false
        submitButton.setTitle("注册中…", for: .normal)
        showError(nil)

        Task {
            do {
                _ = try await self.auth.register(
                    phone: phone,
                    code: code?.isEmpty == true ? nil : code,
                    nickname: nick,
                    password: password,
                    inviteCode: invite?.isEmpty == true ? nil : invite
                )
                if self.legalRow.isChecked {
                    self.prefs.privacyAccepted = true
                }
                await MainActor.run { LinLanRootSwitcher.showMain() }
            } catch {
                await MainActor.run {
                    self.showError(error.localizedDescription)
                    self.loading = false
                    self.submitButton.isEnabled = true
                    self.submitButton.setTitle("创建账号", for: .normal)
                }
            }
        }
    }

    private func openLegal(title: String, url: String) {
        linLanPush(LinLanLegalWebViewController(titleText: title, urlString: url))
    }

    private func showError(_ text: String?) {
        errorLabel.text = text
        errorLabel.isHidden = (text == nil || text?.isEmpty == true)
    }

    private func presentPrivacyGate() {
        let alert = UIAlertController(
            title: "服务协议与隐私保护",
            message: "欢迎使用\(LinLanAppConstants.appName)。请仔细阅读《用户协议》和《隐私政策》。选择「暂不使用」将退出应用，下次打开仍会提示。",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "同意", style: .default) { [weak self] _ in
            self?.privacyDialogAccepted = true
        })
        alert.addAction(UIAlertAction(title: "暂不使用", style: .destructive) { _ in
            exit(0)
        })
        present(alert, animated: true)
    }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}
