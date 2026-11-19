//
//  LinLanLoginViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanLoginViewController: UIViewController {
    private let prefs = LinLanAppContainer.appPrefs()
    private let auth = LinLanAppContainer.authRepository()
    private let loginCredentials = LinLanAppContainer.loginCredentials()

    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let brand = UILabel()
    private let subtitle = UILabel()
    private let phoneField = LinLanAuthForm.textField(placeholder: "手机号")
    private let passwordField = LinLanAuthForm.textField(placeholder: "密码", secure: true)
    private let codeRow = LinLanSmsCodeRow()
    private let rememberPasswordRow = LinLanRememberPasswordRow()
    private let errorLabel = UILabel()
    private let loginButton = LinLanAuthForm.primaryButton(title: "登录")
    private let modeButton = LinLanAuthForm.linkButton(title: "验证码登录")
    private let forgotButton = LinLanAuthForm.linkButton(title: "忘记密码", color: AppTheme.textSecondary)
    private let registerButton = LinLanAuthForm.linkButton(title: "没有账号？去注册")
    private let legalRow = LinLanLegalCheckRow()

    private var useSms = false
    private var privacyDialogAccepted = false
    private var loading = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        navigationController?.setNavigationBarHidden(true, animated: false)
        privacyDialogAccepted = prefs.privacyAccepted
        phoneField.keyboardType = .phonePad
        phoneField.delegate = self
        passwordField.delegate = self
        passwordField.textContentType = .password
        codeRow.codeField.delegate = self
        setupUI()
        bind()
        restoreRememberedCredentials()
        applyMode()
        LinLanAuthForm.attachDoneToolbar(
            to: [phoneField, passwordField, codeRow.codeField],
            target: self,
            action: #selector(dismissKeyboard)
        )
        LinLanAuthForm.installTapToDismiss(on: view, target: self, action: #selector(dismissKeyboard))
        if !prefs.privacyAccepted {
            presentPrivacyGate()
        }
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    private func setupUI() {
        view.addSubview(scroll)
        scroll.keyboardDismissMode = .onDrag
        scroll.alwaysBounceVertical = true
        scroll.snp.makeConstraints { $0.edges.equalToSuperview() }

        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 72, left: 24, bottom: 40, right: 24)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        brand.text = LinLanAppConstants.appName
        brand.font = .systemFont(ofSize: 40, weight: .bold)
        brand.textColor = AppTheme.textPrimary

        subtitle.text = "跑起来，再连上身边的人"
        subtitle.font = .systemFont(ofSize: 15)
        subtitle.textColor = AppTheme.textSecondary

        errorLabel.font = .systemFont(ofSize: 13)
        errorLabel.textColor = .systemRed
        errorLabel.numberOfLines = 0
        errorLabel.isHidden = true

        stack.addArrangedSubview(brand)
        stack.addArrangedSubview(subtitle)
        stack.setCustomSpacing(28, after: subtitle)
        stack.addArrangedSubview(phoneField)
        stack.addArrangedSubview(passwordField)
        stack.addArrangedSubview(codeRow)
        stack.addArrangedSubview(rememberPasswordRow)
        stack.addArrangedSubview(errorLabel)
        stack.addArrangedSubview(legalRow)
        stack.addArrangedSubview(loginButton)
        stack.addArrangedSubview(modeButton)
        stack.addArrangedSubview(forgotButton)
        stack.addArrangedSubview(registerButton)
    }

    private func bind() {
        codeRow.phoneProvider = { [weak self] in self?.phoneField.text ?? "" }
        codeRow.onMessage = { [weak self] msg in self?.showError(msg) }
        legalRow.onOpenAgreement = { [weak self] in
            self?.openLegal(title: "用户协议", url: LinLanAppConstants.userAgreementURL)
        }
        legalRow.onOpenPrivacy = { [weak self] in
            self?.openLegal(title: "隐私政策", url: LinLanAppConstants.privacyPolicyURL)
        }
        loginButton.addTarget(self, action: #selector(loginTapped), for: .touchUpInside)
        modeButton.addTarget(self, action: #selector(toggleMode), for: .touchUpInside)
        forgotButton.addTarget(self, action: #selector(forgotTapped), for: .touchUpInside)
        registerButton.addTarget(self, action: #selector(registerTapped), for: .touchUpInside)
        rememberPasswordRow.onChanged = { [weak self] enabled in
            if !enabled { self?.loginCredentials.clear() }
        }
    }

    private func restoreRememberedCredentials() {
        guard let credentials = loginCredentials.load() else {
            rememberPasswordRow.setOn(false)
            return
        }
        phoneField.text = credentials.phone
        passwordField.text = credentials.password
        rememberPasswordRow.setOn(true)
    }

    private func applyMode() {
        passwordField.isHidden = useSms
        codeRow.isHidden = !useSms
        rememberPasswordRow.isHidden = useSms
        modeButton.setTitle(useSms ? "密码登录" : "验证码登录", for: .normal)
    }

    private func canSubmit() -> Bool {
        // 登录页的当前勾选是硬性条件：即使以前同意过，本次未勾选也不允许登录。
        legalRow.isChecked
    }

    @objc private func toggleMode() {
        useSms.toggle()
        applyMode()
    }

    @objc private func loginTapped() {
        view.endEditing(true)
        let phone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !phone.isEmpty else { showError("请填写手机号"); return }
        guard canSubmit() else { showError("请先同意用户协议与隐私政策"); return }
        guard !loading else { return }

        loading = true
        loginButton.isEnabled = false
        loginButton.setTitle("登录中…", for: .normal)
        showError(nil)

        Task {
            do {
                if self.useSms {
                    let code = self.codeRow.codeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                    guard !code.isEmpty else { throw LinLanAPIError.message("请填写验证码") }
                    _ = try await self.auth.loginWithCode(phone: phone, code: code)
                } else {
                    let password = self.passwordField.text ?? ""
                    guard !password.isEmpty else { throw LinLanAPIError.message("请填写密码") }
                    _ = try await self.auth.loginWithPassword(phone: phone, password: password)
                    if self.rememberPasswordRow.isOn {
                        _ = self.loginCredentials.save(phone: phone, password: password)
                    } else {
                        self.loginCredentials.clear()
                    }
                }
                if self.legalRow.isChecked {
                    self.prefs.privacyAccepted = true
                }
                await MainActor.run { LinLanRootSwitcher.showMain() }
            } catch {
                await MainActor.run {
                    self.showError(error.localizedDescription)
                    self.loading = false
                    self.loginButton.isEnabled = true
                    self.loginButton.setTitle("登录", for: .normal)
                }
            }
        }
    }

    @objc private func forgotTapped() {
        dismissKeyboard()
        navigationController?.setNavigationBarHidden(false, animated: true)
        linLanPush(LinLanForgotPasswordViewController())
    }

    @objc private func registerTapped() {
        dismissKeyboard()
        navigationController?.setNavigationBarHidden(false, animated: true)
        linLanPush(LinLanRegisterViewController())
    }

    private func openLegal(title: String, url: String) {
        let vc = LinLanLegalWebViewController(titleText: title, urlString: url)
        navigationController?.setNavigationBarHidden(false, animated: true)
        linLanPush(vc)
    }

    private func showError(_ text: String?) {
        errorLabel.text = text
        errorLabel.isHidden = (text == nil || text?.isEmpty == true)
    }

    private func presentPrivacyGate() {
        let alert = UIAlertController(
            title: "服务协议与隐私保护",
            message: "欢迎使用\(LinLanAppConstants.appName)。请仔细阅读《用户协议》和《隐私政策》，了解我们如何收集、使用和保护您的个人信息。选择「同意」可继续；选择「暂不使用」将退出应用，下次打开仍会提示。",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "用户协议", style: .default) { [weak self] _ in
            self?.openLegal(title: "用户协议", url: LinLanAppConstants.userAgreementURL)
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) { self?.presentPrivacyGate() }
        })
        alert.addAction(UIAlertAction(title: "隐私政策", style: .default) { [weak self] _ in
            self?.openLegal(title: "隐私政策", url: LinLanAppConstants.privacyPolicyURL)
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) { self?.presentPrivacyGate() }
        })
        // 不写 privacyAccepted，下次冷启动仍弹窗
        alert.addAction(UIAlertAction(title: "暂不使用", style: .destructive) { _ in
            exit(0)
        })
        alert.addAction(UIAlertAction(title: "同意", style: .default) { [weak self] _ in
            self?.privacyDialogAccepted = true
            // Network preflight/HTTPDNS require consent before authentication can succeed.
            self?.prefs.privacyAccepted = true
        })
        present(alert, animated: true)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
    }
}

extension LinLanLoginViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}
