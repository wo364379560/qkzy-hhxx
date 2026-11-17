//
//  LinLanCreateMomentViewController.swift
//  TodayToTrackRun
//
//  对照 ShangLan_CreateCommunityPostVc
//

import PhotosUI
import SnapKit
import UIKit

final class LinLanCreateMomentViewController: UIViewController, UITextViewDelegate, PHPickerViewControllerDelegate {
    var onPosted: (() -> Void)?

    private let textView = UITextView()
    private let placeholder = UILabel()
    private let imageStack = UIStackView()
    private var images: [UIImage] = []
    private var submitting = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "发朋友圈"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .close,
            target: self,
            action: #selector(close)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "发布",
            style: .done,
            target: self,
            action: #selector(submit)
        )
        navigationItem.rightBarButtonItem?.tintColor = AppTheme.accent

        textView.font = .systemFont(ofSize: 16)
        textView.backgroundColor = .white
        textView.layer.cornerRadius = 12
        textView.textContainerInset = UIEdgeInsets(top: 12, left: 10, bottom: 12, right: 10)
        textView.delegate = self
        textView.inputAccessoryView = makeDoneBar()

        placeholder.text = "这一刻的想法…"
        placeholder.font = .systemFont(ofSize: 16)
        placeholder.textColor = AppTheme.textSecondary

        imageStack.axis = .horizontal
        imageStack.spacing = 8
        imageStack.alignment = .center

        let addBtn = UIButton(type: .system)
        addBtn.setTitle("+ 图片", for: .normal)
        addBtn.setTitleColor(AppTheme.accent, for: .normal)
        addBtn.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
        addBtn.addTarget(self, action: #selector(pickImages), for: .touchUpInside)

        view.addSubview(textView)
        textView.addSubview(placeholder)
        view.addSubview(imageStack)
        view.addSubview(addBtn)

        textView.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(16)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(180)
        }
        placeholder.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.leading.equalToSuperview().offset(15)
        }
        imageStack.snp.makeConstraints { make in
            make.top.equalTo(textView.snp.bottom).offset(12)
            make.leading.equalToSuperview().inset(16)
            make.height.equalTo(72)
        }
        addBtn.snp.makeConstraints { make in
            make.top.equalTo(imageStack.snp.bottom).offset(12)
            make.leading.equalToSuperview().inset(16)
        }
    }

    private func makeDoneBar() -> UIToolbar {
        let bar = UIToolbar(frame: CGRect(x: 0, y: 0, width: UIScreen.main.bounds.width, height: 44))
        bar.items = [
            UIBarButtonItem(barButtonSystemItem: .flexibleSpace, target: nil, action: nil),
            UIBarButtonItem(title: "完成", style: .done, target: self, action: #selector(dismissKeyboard))
        ]
        bar.items?.last?.tintColor = AppTheme.accent
        return bar
    }

    @objc private func dismissKeyboard() { view.endEditing(true) }
    @objc private func close() { dismiss(animated: true) }

    func textViewDidChange(_ textView: UITextView) {
        placeholder.isHidden = !(textView.text ?? "").isEmpty
    }

    @objc private func pickImages() {
        var config = PHPickerConfiguration(photoLibrary: .shared())
        config.filter = .images
        config.selectionLimit = max(0, 9 - images.count)
        guard config.selectionLimit > 0 else {
            toast("最多 9 张图片")
            return
        }
        let picker = PHPickerViewController(configuration: config)
        picker.delegate = self
        present(picker, animated: true)
    }

    func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
        picker.dismiss(animated: true)
        guard !results.isEmpty else { return }
        for r in results {
            r.itemProvider.loadObject(ofClass: UIImage.self) { [weak self] obj, _ in
                guard let img = obj as? UIImage else { return }
                DispatchQueue.main.async {
                    guard let self, self.images.count < 9 else { return }
                    self.images.append(img)
                    self.refreshThumbs()
                }
            }
        }
    }

    private func refreshThumbs() {
        imageStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        for img in images {
            let iv = UIImageView(image: img)
            iv.contentMode = .scaleAspectFill
            iv.clipsToBounds = true
            iv.layer.cornerRadius = 8
            iv.snp.makeConstraints { $0.size.equalTo(72) }
            imageStack.addArrangedSubview(iv)
        }
    }

    @objc private func submit() {
        view.endEditing(true)
        guard !submitting else { return }
        let text = textView.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !text.isEmpty || !images.isEmpty else {
            toast("请输入内容或选择图片")
            return
        }
        submitting = true
        navigationItem.rightBarButtonItem?.isEnabled = false
        Task {
            do {
                _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
                let userId = LinLanAppContainer.sessionStore().currentSession().userId
                var urls: [String] = []
                for img in images {
                    let path = FileManager.default.temporaryDirectory
                        .appendingPathComponent("moment_\(UUID().uuidString).jpg")
                    guard let data = img.jpegData(compressionQuality: 0.85) else { continue }
                    try data.write(to: path)
                    let url = try await LinLanAppContainer.apiClient().uploadUserAvatar(
                        userId: userId,
                        filePath: path.path
                    )
                    urls.append(url)
                }
                let title = derivedTitle(from: text)
                _ = try await LinLanAppContainer.apiClient().createCommunityPost(
                    title: title,
                    content: text,
                    imageURLs: urls
                )
                await MainActor.run {
                    self.submitting = false
                    self.onPosted?()
                    self.dismiss(animated: true)
                }
            } catch {
                await MainActor.run {
                    self.submitting = false
                    self.navigationItem.rightBarButtonItem?.isEnabled = true
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    private func derivedTitle(from text: String) -> String {
        let t = text.trimmingCharacters(in: .whitespacesAndNewlines)
        if t.isEmpty { return "分享图片" }
        return t.count > 20 ? String(t.prefix(20)) : t
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        a.addAction(UIAlertAction(title: "好", style: .default))
        present(a, animated: true)
    }
}
