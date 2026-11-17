//
//  LinLanFilePreviewViewController.swift
//  TodayToTrackRun
//
//  应用内查看文件：本地 QuickLook / 远程 WKWebView
//

import QuickLook
import SnapKit
import UIKit
import WebKit

final class LinLanFilePreviewViewController: UIViewController, QLPreviewControllerDataSource, WKNavigationDelegate {
    private let pathOrUrl: String
    private let fileName: String
    private var localURL: URL?
    private let webView = WKWebView()
    private let loading = UIActivityIndicatorView(style: .large)
    private let statusLabel = UILabel()
    private var didStart = false

    init(pathOrUrl: String, fileName: String) {
        self.pathOrUrl = pathOrUrl
        self.fileName = fileName
        super.init(nibName: nil, bundle: nil)
        title = fileName
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )

        loading.hidesWhenStopped = true
        statusLabel.font = .systemFont(ofSize: 13)
        statusLabel.textColor = AppTheme.textSecondary
        statusLabel.textAlignment = .center
        view.addSubview(webView)
        view.addSubview(loading)
        view.addSubview(statusLabel)
        webView.isHidden = true
        webView.navigationDelegate = self
        webView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        loading.snp.makeConstraints { $0.centerX.equalToSuperview(); $0.centerY.equalToSuperview().offset(-12) }
        statusLabel.snp.makeConstraints { make in
            make.top.equalTo(loading.snp.bottom).offset(10)
            make.leading.trailing.equalToSuperview().inset(24)
        }

        // 仅启动一次，避免重复加载
        guard !didStart else { return }
        didStart = true
        Task { await open() }
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }

    private func open() async {
        await MainActor.run {
            loading.startAnimating()
            statusLabel.text = "正在打开…"
        }
        let resolved = await resolveLocalFile()
        await MainActor.run {
            self.loading.stopAnimating()
            self.statusLabel.text = nil
            if let local = resolved {
                self.localURL = local
                let ql = QLPreviewController()
                ql.dataSource = self
                self.addChild(ql)
                self.view.insertSubview(ql.view, at: 0)
                ql.view.snp.makeConstraints { $0.edges.equalTo(self.view.safeAreaLayoutGuide) }
                ql.didMove(toParent: self)
            } else if let url = URL(string: self.pathOrUrl),
                      self.pathOrUrl.hasPrefix("http://") || self.pathOrUrl.hasPrefix("https://") {
                self.webView.isHidden = false
                self.statusLabel.text = "正在加载预览…"
                self.webView.load(URLRequest(url: url))
            } else {
                self.toast("无法打开该文件")
            }
        }
    }

    /// 优先复用本地缓存，远程仅下载一次到磁盘
    private func resolveLocalFile() async -> URL? {
        if pathOrUrl.hasPrefix("http://") || pathOrUrl.hasPrefix("https://") {
            guard let remote = URL(string: pathOrUrl) else { return nil }
            let dir = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)[0]
                .appendingPathComponent("linlan_chat_media", isDirectory: true)
            try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
            let safe = fileName.replacingOccurrences(of: "/", with: "_")
            let key = String(pathOrUrl.hashValue)
            let out = dir.appendingPathComponent("preview_\(key)_\(safe)")
            if FileManager.default.fileExists(atPath: out.path),
               (try? out.resourceValues(forKeys: [.fileSizeKey]).fileSize) ?? 0 > 0 {
                return out
            }
            do {
                let (tmp, _) = try await URLSession.shared.download(from: remote)
                if FileManager.default.fileExists(atPath: out.path) {
                    try? FileManager.default.removeItem(at: out)
                }
                try FileManager.default.moveItem(at: tmp, to: out)
                return out
            } catch {
                return nil
            }
        }
        if pathOrUrl.hasPrefix("file://"), let url = URL(string: pathOrUrl) {
            return FileManager.default.fileExists(atPath: url.path) ? url : nil
        }
        let url = URL(fileURLWithPath: pathOrUrl)
        return FileManager.default.fileExists(atPath: url.path) ? url : nil
    }

    func numberOfPreviewItems(in controller: QLPreviewController) -> Int { localURL == nil ? 0 : 1 }

    func previewController(_ controller: QLPreviewController, previewItemAt index: Int) -> QLPreviewItem {
        (localURL ?? URL(fileURLWithPath: "/")) as QLPreviewItem
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}
