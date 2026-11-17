//
//  LinLanReportViewController.swift
//  TodayToTrackRun
//
//  对照睿之禾举报：push 进入，提交在导航栏右上角
//

import SnapKit
import UIKit

final class LinLanReportViewController: UIViewController {
    private let targetId: String
    private let targetType: String
    private let targetName: String

    private let reasons: [(key: String, title: String)] = [
        ("spam", "垃圾信息 / 广告"),
        ("abuse", "骚扰 / 辱骂"),
        ("violence", "暴力 / 威胁"),
        ("fraud", "欺诈 / 诈骗"),
        ("porn", "色情 / 低俗"),
        ("illegal", "违法违规"),
        ("other", "其他"),
    ]

    private var selectedKey: String?
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let descView = UITextView()
    private var submitting = false

    init(targetId: String, targetType: String, targetName: String) {
        self.targetId = targetId
        self.targetType = targetType
        self.targetName = targetName
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    static func push(
        from: UIViewController,
        targetId: String,
        targetType: String,
        targetName: String
    ) {
        from.linLanPush(
            LinLanReportViewController(
                targetId: targetId,
                targetType: targetType,
                targetName: targetName
            )
        )
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "举报"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "提交",
            style: .done,
            target: self,
            action: #selector(submit)
        )
        navigationItem.rightBarButtonItem?.tintColor = .systemRed

        tableView.dataSource = self
        tableView.delegate = self
        tableView.backgroundColor = AppTheme.pageBackground
        tableView.keyboardDismissMode = .onDrag
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "r")

        descView.font = .systemFont(ofSize: 15)
        descView.layer.cornerRadius = 10
        descView.layer.borderWidth = 1 / UIScreen.main.scale
        descView.layer.borderColor = AppTheme.divider.cgColor
        descView.textContainerInset = UIEdgeInsets(top: 10, left: 8, bottom: 10, right: 8)
        descView.backgroundColor = .white
        descView.returnKeyType = .done
        descView.delegate = self
        descView.snp.makeConstraints { $0.height.equalTo(100) }

        let tip = UILabel()
        tip.text = "补充说明（选填，最多 200 字）"
        tip.font = .systemFont(ofSize: 13)
        tip.textColor = AppTheme.textSecondary

        let footer = UIStackView(arrangedSubviews: [tip, descView])
        footer.axis = .vertical
        footer.spacing = 10
        footer.isLayoutMarginsRelativeArrangement = true
        footer.layoutMargins = UIEdgeInsets(top: 8, left: 16, bottom: 28, right: 16)
        let wrap = UIView()
        wrap.addSubview(footer)
        footer.snp.makeConstraints { $0.edges.equalToSuperview() }
        wrap.frame = CGRect(x: 0, y: 0, width: view.bounds.width, height: 160)
        tableView.tableFooterView = wrap

        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        tableView.addGestureRecognizer(tap)

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(kbShow(_:)),
            name: UIResponder.keyboardWillShowNotification,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(kbHide(_:)),
            name: UIResponder.keyboardWillHideNotification,
            object: nil
        )
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    @objc private func back() {
        dismissKeyboard()
        navigationController?.popViewController(animated: true)
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    @objc private func kbShow(_ n: Notification) {
        guard
            let frame = n.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect,
            let duration = n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? Double
        else { return }
        let inset = frame.intersection(view.convert(view.bounds, to: nil)).height
            - view.safeAreaInsets.bottom
        let bottom = max(0, inset)
        UIView.animate(withDuration: duration) {
            self.tableView.contentInset.bottom = bottom + 16
            self.tableView.verticalScrollIndicatorInsets.bottom = bottom + 16
        }
        // 把补充说明滚进可视区
        if let footer = tableView.tableFooterView {
            let rect = footer.convert(footer.bounds, to: tableView)
            tableView.scrollRectToVisible(rect.insetBy(dx: 0, dy: -20), animated: true)
        }
    }

    @objc private func kbHide(_ n: Notification) {
        let duration = (n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? Double) ?? 0.25
        UIView.animate(withDuration: duration) {
            self.tableView.contentInset.bottom = 0
            self.tableView.verticalScrollIndicatorInsets.bottom = 0
        }
    }

    @objc private func submit() {
        dismissKeyboard()
        guard !submitting else { return }
        guard let key = selectedKey else {
            toast("请选择举报原因")
            return
        }
        submitting = true
        navigationItem.rightBarButtonItem?.isEnabled = false
        navigationItem.rightBarButtonItem?.title = "提交中"
        Task {
            do {
                try await LinLanAppContainer.apiClient().submitReport(
                    targetId: targetId,
                    targetType: targetType,
                    targetName: targetName,
                    category: key,
                    description: descView.text
                )
                await MainActor.run {
                    self.toast("举报已提交")
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.6) {
                        self.navigationController?.popViewController(animated: true)
                    }
                }
            } catch {
                await MainActor.run {
                    self.submitting = false
                    self.navigationItem.rightBarButtonItem?.isEnabled = true
                    self.navigationItem.rightBarButtonItem?.title = "提交"
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak a] in
            a?.dismiss(animated: true)
        }
    }
}

extension LinLanReportViewController: UITableViewDataSource, UITableViewDelegate, UITextViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { reasons.count }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        "选择举报原因"
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "r", for: indexPath)
        let r = reasons[indexPath.row]
        cell.textLabel?.text = r.title
        cell.accessoryType = selectedKey == r.key ? .checkmark : .none
        cell.tintColor = AppTheme.accent
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        dismissKeyboard()
        selectedKey = reasons[indexPath.row].key
        tableView.reloadData()
    }

    func textView(_ textView: UITextView, shouldChangeTextIn range: NSRange, replacementText text: String) -> Bool {
        if text == "\n" {
            dismissKeyboard()
            return false
        }
        let current = textView.text as NSString? ?? ""
        let next = current.replacingCharacters(in: range, with: text)
        return next.count <= 200
    }
}
