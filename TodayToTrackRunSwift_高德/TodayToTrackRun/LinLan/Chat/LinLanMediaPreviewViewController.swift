//
//  LinLanMediaPreviewViewController.swift
//  TodayToTrackRun
//
//  图片放大/保存、视频播放
//

import AVKit
import Photos
import SnapKit
import UIKit

enum LinLanMediaPreviewKind {
    case image(urlOrPath: String)
    case video(pathOrUrl: String)
}

final class LinLanMediaPreviewViewController: UIViewController, UIScrollViewDelegate {
    private let kind: LinLanMediaPreviewKind
    private let scrollView = UIScrollView()
    private let imageView = UIImageView()
    private var playerVC: AVPlayerViewController?
    private let loading = UIActivityIndicatorView(style: .large)
    private let topBar = UIView()
    private let closeButton = UIButton(type: .system)
    private let saveButton = UIButton(type: .system)

    init(kind: LinLanMediaPreviewKind) {
        self.kind = kind
        super.init(nibName: nil, bundle: nil)
        modalPresentationStyle = .fullScreen
        modalTransitionStyle = .crossDissolve
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black

        loading.color = .white
        loading.hidesWhenStopped = true
        view.addSubview(loading)
        loading.snp.makeConstraints { $0.center.equalToSuperview() }

        switch kind {
        case .image(let urlOrPath):
            setupImage(urlOrPath)
        case .video(let pathOrUrl):
            setupVideo(pathOrUrl)
        }
        setupTopBar(showSave: {
            if case .image = kind { return true }
            return false
        }())
    }

    private func setupTopBar(showSave: Bool) {
        topBar.backgroundColor = UIColor.black.withAlphaComponent(0.45)
        view.addSubview(topBar)
        topBar.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
            make.bottom.equalTo(view.safeAreaLayoutGuide.snp.top).offset(52)
        }

        // 关闭：白底黑 X，更醒目
        closeButton.setImage(
            UIImage(systemName: "xmark", withConfiguration: UIImage.SymbolConfiguration(pointSize: 15, weight: .bold)),
            for: .normal
        )
        closeButton.tintColor = .black
        closeButton.backgroundColor = .white
        closeButton.layer.cornerRadius = 18
        closeButton.clipsToBounds = true
        closeButton.accessibilityLabel = "关闭"
        closeButton.addTarget(self, action: #selector(closeTapped), for: .touchUpInside)
        topBar.addSubview(closeButton)
        closeButton.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.bottom.equalToSuperview().inset(8)
            make.size.equalTo(36)
        }

        if showSave {
            saveButton.setTitle("保存", for: .normal)
            saveButton.setTitleColor(.black, for: .normal)
            saveButton.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
            saveButton.backgroundColor = .white
            saveButton.layer.cornerRadius = 18
            saveButton.clipsToBounds = true
            saveButton.addTarget(self, action: #selector(saveImage), for: .touchUpInside)
            topBar.addSubview(saveButton)
            saveButton.snp.makeConstraints { make in
                make.leading.equalToSuperview().offset(16)
                make.centerY.equalTo(closeButton)
                make.height.equalTo(36)
                make.width.greaterThanOrEqualTo(72)
            }
            saveButton.contentEdgeInsets = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 16)
        }
    }

    private func setupImage(_ urlOrPath: String) {
        scrollView.delegate = self
        scrollView.minimumZoomScale = 1
        scrollView.maximumZoomScale = 4
        scrollView.showsVerticalScrollIndicator = false
        scrollView.showsHorizontalScrollIndicator = false
        scrollView.backgroundColor = .black
        imageView.contentMode = .scaleAspectFit
        imageView.isUserInteractionEnabled = true
        view.insertSubview(scrollView, at: 0)
        scrollView.addSubview(imageView)
        scrollView.snp.makeConstraints { $0.edges.equalToSuperview() }
        imageView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalTo(scrollView)
            make.height.equalTo(scrollView)
        }

        // 底部再放一个大保存按钮，避免只靠顶部时被忽略
        let bottomSave = UIButton(type: .system)
        bottomSave.setTitle("保存到相册", for: .normal)
        bottomSave.setTitleColor(.black, for: .normal)
        bottomSave.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        bottomSave.backgroundColor = .white
        bottomSave.layer.cornerRadius = 22
        bottomSave.clipsToBounds = true
        bottomSave.addTarget(self, action: #selector(saveImage), for: .touchUpInside)
        view.addSubview(bottomSave)
        bottomSave.snp.makeConstraints { make in
            make.centerX.equalToSuperview()
            make.bottom.equalTo(view.safeAreaLayoutGuide).inset(24)
            make.height.equalTo(44)
            make.width.equalTo(160)
        }

        loading.startAnimating()
        Task {
            let data = await LinLanImManager.loadImageData(urlString: urlOrPath)
            await MainActor.run {
                self.loading.stopAnimating()
                if let data, let img = UIImage(data: data) {
                    self.imageView.image = img
                } else {
                    self.toast("图片加载失败")
                }
                self.view.bringSubviewToFront(self.topBar)
                self.view.bringSubviewToFront(bottomSave)
            }
        }
    }

    private func setupVideo(_ pathOrUrl: String) {
        let url: URL? = {
            if pathOrUrl.hasPrefix("http://") || pathOrUrl.hasPrefix("https://") {
                return URL(string: pathOrUrl)
            }
            if pathOrUrl.hasPrefix("file://") {
                return URL(string: pathOrUrl)
            }
            return URL(fileURLWithPath: pathOrUrl)
        }()
        guard let url else {
            toast("无法播放该视频")
            return
        }
        let player = AVPlayer(url: url)
        let pvc = AVPlayerViewController()
        pvc.player = player
        pvc.showsPlaybackControls = true
        addChild(pvc)
        view.insertSubview(pvc.view, at: 0)
        pvc.view.snp.makeConstraints { $0.edges.equalToSuperview() }
        pvc.didMove(toParent: self)
        playerVC = pvc
        player.play()
        view.bringSubviewToFront(topBar)
    }

    func viewForZooming(in scrollView: UIScrollView) -> UIView? { imageView }

    @objc private func closeTapped() {
        playerVC?.player?.pause()
        dismiss(animated: true)
    }

    @objc private func saveImage() {
        guard let image = imageView.image else {
            toast("图片加载中，请稍后再试")
            return
        }
        PHPhotoLibrary.requestAuthorization(for: .addOnly) { [weak self] status in
            guard let self else { return }
            guard status == .authorized || status == .limited else {
                DispatchQueue.main.async { self.toast("无相册写入权限") }
                return
            }
            UIImageWriteToSavedPhotosAlbum(
                image,
                self,
                #selector(self.saveFinished(_:didFinishSavingWithError:contextInfo:)),
                nil
            )
        }
    }

    @objc private func saveFinished(
        _ image: UIImage,
        didFinishSavingWithError error: Error?,
        contextInfo: UnsafeRawPointer?
    ) {
        toast(error == nil ? "已保存到相册" : (error?.localizedDescription ?? "保存失败"))
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}
