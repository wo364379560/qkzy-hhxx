//
//  LinLanMergerDetailViewController.swift
//  TodayToTrackRun
//
//  合并转发详情：可查看文本/图片/视频/文件
//

import AVFoundation
import ImSDK_Plus_Swift
import SnapKit
import UIKit

final class LinLanMergerDetailViewController: UIViewController {
    private let source: V2TIMMessage
    private var childMessages: [V2TIMMessage] = []
    private let tableView = UITableView(frame: .zero, style: .plain)
    private let emptyLabel = UILabel()
    private let loading = UIActivityIndicatorView(style: .medium)

    init(message: V2TIMMessage) {
        self.source = message
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        let card = LinLanImManager.parseMergerCard(source)
        title = card?.title ?? "聊天记录"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )

        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 72
        tableView.backgroundColor = .white
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "c")
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        emptyLabel.textAlignment = .center
        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.numberOfLines = 0
        emptyLabel.isHidden = true
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalToSuperview().inset(24) }

        loading.hidesWhenStopped = true
        view.addSubview(loading)
        loading.snp.makeConstraints { $0.center.equalToSuperview() }

        if card?.layersOverLimit == true {
            emptyLabel.isHidden = false
            emptyLabel.text = "聊天记录层数超限，无法查看"
            return
        }
        Task { await load() }
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }

    private func load() async {
        await MainActor.run { loading.startAnimating() }
        do {
            let list = try await LinLanImManager.downloadMergerMessage(source)
            await MainActor.run {
                self.loading.stopAnimating()
                self.childMessages = list
                self.emptyLabel.isHidden = !list.isEmpty
                self.emptyLabel.text = list.isEmpty ? "暂无内容" : nil
                self.tableView.reloadData()
            }
        } catch {
            await MainActor.run {
                self.loading.stopAnimating()
                self.emptyLabel.isHidden = false
                self.emptyLabel.text = error.localizedDescription
            }
        }
    }

    private func openChild(_ msg: V2TIMMessage) {
        LinLanChatMediaPresenter.open(message: msg, from: self)
    }

    private func openVideoAttachment(_ attachment: LinLanAttachmentCard) {
        guard attachment.isVideo else { return }
        loading.startAnimating()
        Task {
            do {
                let url = try await LinLanAppContainer.apiClient().attachmentAccess(fileID: attachment.fileID)
                await MainActor.run {
                    self.loading.stopAnimating()
                    self.present(
                        LinLanMediaPreviewViewController(kind: .video(pathOrUrl: url.absoluteString)),
                        animated: true
                    )
                }
            } catch {
                await MainActor.run {
                    self.loading.stopAnimating()
                    let alert = UIAlertController(title: nil, message: "视频暂时无法访问，请稍后重试", preferredStyle: .alert)
                    alert.addAction(UIAlertAction(title: "好", style: .default))
                    self.present(alert, animated: true)
                }
            }
        }
    }
}

extension LinLanMergerDetailViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        childMessages.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "c", for: indexPath)
        let msg = childMessages[indexPath.row]
        var content = cell.defaultContentConfiguration()
        content.text = LinLanQuoteHelper.senderName(msg)
        content.secondaryText = LinLanImManager.messagePreview(msg)
        content.textProperties.font = .systemFont(ofSize: 15, weight: .semibold)
        content.secondaryTextProperties.color = AppTheme.textSecondary
        content.secondaryTextProperties.numberOfLines = 3
        switch msg.elemType {
        case .V2TIM_ELEM_TYPE_IMAGE, .V2TIM_ELEM_TYPE_VIDEO, .V2TIM_ELEM_TYPE_FILE,
             .V2TIM_ELEM_TYPE_MERGER:
            cell.accessoryType = .disclosureIndicator
        default:
            let canOpen = LinLanAttachmentCard.parse(msg.textElem?.text)?.isVideo == true
                || LinLanImManager.parseMergerCard(msg) != nil
            cell.accessoryType = canOpen ? .disclosureIndicator : .none
        }
        cell.contentConfiguration = content
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let msg = childMessages[indexPath.row]
        switch msg.elemType {
        case .V2TIM_ELEM_TYPE_IMAGE, .V2TIM_ELEM_TYPE_VIDEO, .V2TIM_ELEM_TYPE_FILE,
             .V2TIM_ELEM_TYPE_MERGER:
            openChild(msg)
        default:
            if let attachment = LinLanAttachmentCard.parse(msg.textElem?.text), attachment.isVideo {
                openVideoAttachment(attachment)
            } else if LinLanImManager.parseMergerCard(msg) != nil {
                openChild(msg)
            } else if msg.elemType == .V2TIM_ELEM_TYPE_TEXT {
                let text = msg.textElem?.text ?? ""
                let a = UIAlertController(title: LinLanQuoteHelper.senderName(msg), message: text, preferredStyle: .alert)
                a.addAction(UIAlertAction(title: "好", style: .default))
                present(a, animated: true)
            }
        }
    }
}

/// 聊天/合并详情共用的媒体打开入口（防连点重复 push / present）
enum LinLanChatMediaPresenter {
    private static var isOpening = false
    private static var openingToken = 0
    private static var soundPlayer: AVAudioPlayer?
    private static var soundPlayingMsgId: String?
    private static let soundBridge = LinLanSoundPlayerBridge()

    /// 当前正在播放的语音消息 id（无则 nil）
    static var playingSoundMsgId: String? { soundPlayingMsgId }

    static func open(message: V2TIMMessage, from vc: UIViewController) {
        // 已在打开中，或已有预览层，忽略连点
        if isOpening { return }
        if vc.presentedViewController is LinLanMediaPreviewViewController { return }
        if vc.navigationController?.topViewController is LinLanFilePreviewViewController { return }

        switch message.elemType {
        case .V2TIM_ELEM_TYPE_SOUND:
            playSound(message, from: vc)

        case .V2TIM_ELEM_TYPE_IMAGE:
            beginOpening()
            showBusy(true, on: vc, text: "图片加载中…")
            let token = openingToken
            Task {
                let path = await LinLanImManager.resolveImagePlayable(message)
                await MainActor.run {
                    guard token == openingToken else { return }
                    showBusy(false, on: vc)
                    endOpening()
                    guard let path, !path.isEmpty else {
                        toast("图片加载失败", from: vc)
                        return
                    }
                    presentImage(path, from: vc)
                }
            }

        case .V2TIM_ELEM_TYPE_VIDEO:
            beginOpening()
            showBusy(true, on: vc, text: "视频加载中…")
            let token = openingToken
            Task {
                let path = await LinLanImManager.resolveVideoPlayable(message)
                await MainActor.run {
                    guard token == openingToken else { return }
                    showBusy(false, on: vc)
                    endOpening()
                    guard let path, !path.isEmpty else {
                        toast("视频加载失败", from: vc)
                        return
                    }
                    guard vc.presentedViewController == nil else { return }
                    let preview = LinLanMediaPreviewViewController(kind: .video(pathOrUrl: path))
                    vc.present(preview, animated: true)
                }
            }

        case .V2TIM_ELEM_TYPE_FILE:
            beginOpening()
            showBusy(true, on: vc, text: "文件加载中…")
            let token = openingToken
            Task {
                let resolved = await LinLanImManager.resolveFileLocal(message)
                await MainActor.run {
                    guard token == openingToken else { return }
                    showBusy(false, on: vc)
                    endOpening()
                    guard let resolved else {
                        toast("文件加载失败", from: vc)
                        return
                    }
                    // 顶层已是文件预览则不再 push
                    if vc.navigationController?.topViewController is LinLanFilePreviewViewController {
                        return
                    }
                    vc.linLanPush(
                        LinLanFilePreviewViewController(
                            pathOrUrl: resolved.pathOrUrl,
                            fileName: resolved.fileName
                        )
                    )
                }
            }

        case .V2TIM_ELEM_TYPE_MERGER:
            pushMerger(message, from: vc)

        default:
            if LinLanImManager.parseMergerCard(message) != nil {
                pushMerger(message, from: vc)
            }
        }
    }

    private static func playSound(_ message: V2TIMMessage, from vc: UIViewController) {
        let msgId = message.msgID
        // 再点同一条：停止
        if soundPlayingMsgId == msgId, soundPlayer?.isPlaying == true {
            stopSoundPlayback()
            return
        }
        beginOpening()
        showBusy(true, on: vc, text: "语音加载中…")
        let token = openingToken
        Task {
            let pathOrUrl = await LinLanImManager.resolveSoundPlayable(message)
            var playerData: Data?
            var localPath: String?
            if let pathOrUrl, !pathOrUrl.isEmpty {
                if pathOrUrl.hasPrefix("http://") || pathOrUrl.hasPrefix("https://") {
                    if let remote = URL(string: pathOrUrl) {
                        playerData = try? await URLSession.shared.data(from: remote).0
                    }
                } else {
                    localPath = pathOrUrl.hasPrefix("file://")
                        ? (URL(string: pathOrUrl)?.path ?? pathOrUrl)
                        : pathOrUrl
                }
            }
            await MainActor.run {
                guard token == openingToken else { return }
                showBusy(false, on: vc)
                endOpening()
                do {
                    try activateVoicePlaybackSession()
                    let player: AVAudioPlayer
                    if let playerData, !playerData.isEmpty {
                        player = try AVAudioPlayer(data: playerData)
                    } else if let localPath, FileManager.default.fileExists(atPath: localPath) {
                        player = try AVAudioPlayer(contentsOf: URL(fileURLWithPath: localPath))
                    } else {
                        toast("语音加载失败", from: vc)
                        return
                    }
                    player.delegate = soundBridge
                    player.prepareToPlay()
                    soundPlayer?.stop()
                    soundPlayer = player
                    guard player.play() else {
                        stopSoundPlayback()
                        toast("无法播放语音", from: vc)
                        return
                    }
                    setPlayingSoundMsgId(msgId)
                } catch {
                    #if DEBUG
                    print("[LinLanSound] play error: \(error)")
                    #endif
                    stopSoundPlayback()
                    toast("无法播放语音", from: vc)
                }
            }
        }
    }

    fileprivate static func handleSoundFinished() {
        stopSoundPlayback()
    }

    private static func stopSoundPlayback() {
        soundPlayer?.stop()
        soundPlayer = nil
        setPlayingSoundMsgId(nil)
    }

    private static func setPlayingSoundMsgId(_ msgId: String?) {
        soundPlayingMsgId = msgId
        var info: [AnyHashable: Any] = [:]
        if let msgId { info["msgId"] = msgId }
        NotificationCenter.default.post(
            name: .linLanSoundPlaybackDidChange,
            object: nil,
            userInfo: info
        )
    }

    /// 播放会话：不要用 defaultToSpeaker（仅 playAndRecord 可用）；用 override 强制外放
    private static func activateVoicePlaybackSession() throws {
        let session = AVAudioSession.sharedInstance()
        try? session.setActive(false, options: .notifyOthersOnDeactivation)
        try session.setCategory(.playback, mode: .default, options: [])
        try session.setActive(true)
        try? session.overrideOutputAudioPort(.speaker)
    }

    private static func pushMerger(_ message: V2TIMMessage, from vc: UIViewController) {
        beginOpening()
        vc.linLanPush(LinLanMergerDetailViewController(message: message))
        // push 完成下一个主线程周期即可响应详情页媒体点击。
        DispatchQueue.main.async { endOpening() }
    }

    private static func presentImage(_ url: String, from vc: UIViewController) {
        beginOpening()
        let preview = LinLanMediaPreviewViewController(kind: .image(urlOrPath: url))
        vc.present(preview, animated: true) {
            endOpening()
        }
        // present 失败兜底
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) {
            if isOpening { endOpening() }
        }
    }

    private static func beginOpening() {
        isOpening = true
        openingToken += 1
    }

    private static func endOpening() {
        isOpening = false
    }

    private static func showBusy(_ show: Bool, on vc: UIViewController, text: String = "") {
        assert(Thread.isMainThread)
        let tag = 9_720_331
        guard vc.isViewLoaded else { return }
        if show {
            if vc.view.viewWithTag(tag) != nil { return }
            let overlay = UIView()
            overlay.tag = tag
            overlay.translatesAutoresizingMaskIntoConstraints = false
            overlay.backgroundColor = UIColor.black.withAlphaComponent(0.25)
            let box = UIView()
            box.translatesAutoresizingMaskIntoConstraints = false
            box.backgroundColor = UIColor(white: 0.15, alpha: 0.92)
            box.layer.cornerRadius = 12
            let spinner = UIActivityIndicatorView(style: .medium)
            spinner.color = .white
            spinner.startAnimating()
            let label = UILabel()
            label.text = text
            label.textColor = .white
            label.font = .systemFont(ofSize: 14, weight: .medium)
            let stack = UIStackView(arrangedSubviews: [spinner, label])
            stack.translatesAutoresizingMaskIntoConstraints = false
            stack.axis = .horizontal
            stack.spacing = 10
            stack.alignment = .center
            vc.view.addSubview(overlay)
            overlay.addSubview(box)
            box.addSubview(stack)
            NSLayoutConstraint.activate([
                overlay.topAnchor.constraint(equalTo: vc.view.topAnchor),
                overlay.leadingAnchor.constraint(equalTo: vc.view.leadingAnchor),
                overlay.trailingAnchor.constraint(equalTo: vc.view.trailingAnchor),
                overlay.bottomAnchor.constraint(equalTo: vc.view.bottomAnchor),
                box.centerXAnchor.constraint(equalTo: overlay.centerXAnchor),
                box.centerYAnchor.constraint(equalTo: overlay.centerYAnchor),
                stack.topAnchor.constraint(equalTo: box.topAnchor, constant: 14),
                stack.leadingAnchor.constraint(equalTo: box.leadingAnchor, constant: 18),
                stack.trailingAnchor.constraint(equalTo: box.trailingAnchor, constant: -18),
                stack.bottomAnchor.constraint(equalTo: box.bottomAnchor, constant: -14),
            ])
        } else {
            vc.view.viewWithTag(tag)?.removeFromSuperview()
        }
    }

    private static func toast(_ msg: String, from vc: UIViewController) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        vc.present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}

private final class LinLanSoundPlayerBridge: NSObject, AVAudioPlayerDelegate {
    func audioPlayerDidFinishPlaying(_ player: AVAudioPlayer, successfully flag: Bool) {
        DispatchQueue.main.async {
            LinLanChatMediaPresenter.handleSoundFinished()
        }
    }

    func audioPlayerDecodeErrorDidOccur(_ player: AVAudioPlayer, error: Error?) {
        DispatchQueue.main.async {
            LinLanChatMediaPresenter.handleSoundFinished()
        }
    }
}
