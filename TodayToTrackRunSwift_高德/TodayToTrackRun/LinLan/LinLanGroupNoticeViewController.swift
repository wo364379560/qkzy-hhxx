//
//  LinLanGroupNoticeViewController.swift
//

import ImSDK_Plus_Swift
import SnapKit
import UIKit

final class LinLanGroupNoticeViewController: UIViewController, UITextViewDelegate {
    static let maxBytes = 5000

    private let groupId: String
    private let canEdit: Bool
    private let textView = UITextView()
    private let countLabel = UILabel()

    init(groupId: String, canEdit: Bool) {
        self.groupId = groupId
        self.canEdit = canEdit
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "群公告"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        if canEdit {
            navigationItem.rightBarButtonItem = UIBarButtonItem(
                title: "保存",
                style: .done,
                target: self,
                action: #selector(save)
            )
            navigationItem.rightBarButtonItem?.tintColor = AppTheme.accent
        }

        let noticeFont = UIFont.systemFont(ofSize: 16)
        textView.font = noticeFont
        // 页面背景固定为白色，文字颜色也必须明确指定；否则部分系统/外观配置下
        // 普通文字会变成白色，只有彩色 Emoji 看得见。
        textView.textColor = AppTheme.textPrimary
        textView.tintColor = AppTheme.accent
        textView.typingAttributes = [
            .font: noticeFont,
            .foregroundColor: AppTheme.textPrimary
        ]
        textView.isEditable = canEdit
        textView.isSelectable = true
        textView.isScrollEnabled = true
        textView.backgroundColor = .white
        textView.layer.cornerRadius = 0
        textView.textContainerInset = UIEdgeInsets(top: 16, left: 14, bottom: 48, right: 14)
        textView.textContainer.lineBreakMode = .byCharWrapping
        textView.textContainer.widthTracksTextView = true
        textView.delegate = self
        textView.alwaysBounceVertical = true
        textView.keyboardDismissMode = .interactive
        if canEdit {
            textView.inputAccessoryView = makeKeyboardDoneBar()
        }
        countLabel.font = .systemFont(ofSize: 12)
        countLabel.textAlignment = .right
        countLabel.backgroundColor = UIColor.white.withAlphaComponent(0.92)
        countLabel.isUserInteractionEnabled = true
        countLabel.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard)))
        view.addSubview(textView)
        view.addSubview(countLabel)
        textView.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide)
            make.leading.trailing.equalToSuperview()
            make.bottom.equalTo(view.keyboardLayoutGuide.snp.top)
        }
        countLabel.snp.makeConstraints { make in
            make.trailing.equalTo(view.safeAreaLayoutGuide).inset(16)
            make.bottom.equalTo(view.keyboardLayoutGuide.snp.top).offset(-10)
        }

        // 点空白区域（导航栏下方空白）收起；textView 占满时靠「完成」工具条
        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)

        Task { await load() }
    }

    private func makeKeyboardDoneBar() -> UIToolbar {
        let bar = UIToolbar(frame: CGRect(x: 0, y: 0, width: UIScreen.main.bounds.width, height: 44))
        bar.items = [
            UIBarButtonItem(barButtonSystemItem: .flexibleSpace, target: nil, action: nil),
            UIBarButtonItem(title: "完成", style: .done, target: self, action: #selector(dismissKeyboard))
        ]
        bar.items?.last?.tintColor = AppTheme.accent
        return bar
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    private func load() async {
        let content = (try? await LinLanAppContainer.apiClient().getGroupNotice(groupId: groupId)) ?? ""
        await MainActor.run {
            textView.text = content.isEmpty
                ? (canEdit ? "" : "暂无公告")
                : content
            textView.textColor = AppTheme.textPrimary
            textView.setContentOffset(.zero, animated: false)
            refreshCount()
        }
    }

    private func utf8ByteCount(_ text: String) -> Int {
        text.utf8.count
    }

    private func refreshCount() {
        let n = utf8ByteCount(textView.text ?? "")
        countLabel.text = "\(n)/\(Self.maxBytes) 字节"
        countLabel.textColor = n > Self.maxBytes ? .systemRed : AppTheme.textSecondary
    }

    func textViewDidChange(_ textView: UITextView) {
        refreshCount()
    }

    @objc private func back() {
        view.endEditing(true)
        navigationController?.popViewController(animated: true)
    }

    @objc private func save() {
        view.endEditing(true)
        let content = textView.text ?? ""
        let bytes = utf8ByteCount(content)
        guard bytes <= Self.maxBytes else {
            toast("群公告不能超过 \(Self.maxBytes) 字节")
            return
        }
        Task {
            do {
                _ = try await LinLanAppContainer.apiClient().saveGroupNotice(
                    groupId: groupId,
                    content: content
                )
                let info = V2TIMGroupInfo()
                info.groupID = groupId
                info.notification = String(content.prefix(200))
                try? await LinLanImManager.setGroupInfo(info)
                await MainActor.run {
                    self.toast("公告已保存")
                    self.navigationController?.popViewController(animated: true)
                }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak a] in a?.dismiss(animated: true) }
    }
}
