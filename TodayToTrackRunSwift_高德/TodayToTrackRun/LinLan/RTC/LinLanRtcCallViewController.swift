//
//  LinLanRtcCallViewController.swift
//  TodayToTrackRun
//
//  对照 Android LinLanRtcCallActivity
//

import SnapKit
import UIKit

final class LinLanRtcCallViewController: UIViewController, LinLanRtcCallUiBridge {
    private let remoteView = UIView()
    private let localView = UIView()
    private let nameLabel = UILabel()
    private let statusLabel = UILabel()
    private let avatarLetter = UILabel()
    private let acceptButton = UIButton(type: .system)
    private let rejectButton = UIButton(type: .system)
    private let cancelButton = UIButton(type: .system)
    private let hangupButton = UIButton(type: .system)
    private let micButton = UIButton(type: .system)
    private let speakerButton = UIButton(type: .system)
    private let cameraFlipButton = UIButton(type: .system)
    private let waitingStack = UIStackView()
    private let inCallStack = UIStackView()

    private var viewReady = false
    private var micOn = true
    private var speakerOn = false
    private var durationTimer: Timer?
    var onViewReady: (() -> Void)?

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = UIColor(white: 0.08, alpha: 1)
        LinLanRtcCallManager.shared.activeUi = self
        setupUI()
        refreshFromManager()
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        viewReady = true
        onViewReady?()
        onViewReady = nil
        if let ctx = LinLanRtcCallManager.shared.callContext,
           LinLanRtcCallManager.shared.status == .accept {
            LinLanTrtcProvider.shared.rebindMediaViews(
                localPreview: localView,
                remoteView: remoteView,
                mediaType: ctx.mediaType,
                remoteUserId: LinLanTrtcProvider.shared.remoteUserId ?? ctx.peer.userId
            )
        }
    }

    deinit {
        stopDurationTimer()
        if LinLanRtcCallManager.shared.activeUi === self {
            LinLanRtcCallManager.shared.activeUi = nil
        }
    }

    func localPreviewView() -> UIView? { localView }
    func remoteVideoView() -> UIView? { remoteView }
    func isViewReady() -> Bool { viewReady }

    func refreshFromManager() {
        guard let ctx = LinLanRtcCallManager.shared.callContext else { return }
        let status = LinLanRtcCallManager.shared.status
        let isVideo = ctx.mediaType == .video
        let inCall = status == .accept
        nameLabel.text = ctx.peer.displayName
        avatarLetter.text = String(ctx.peer.displayName.prefix(1))
        if inCall {
            statusLabel.text = LinLanRtcSignalingPayload.formatClock(LinLanRtcCallManager.shared.connectedElapsedSec())
            startDurationTimer()
        } else {
            stopDurationTimer()
            statusLabel.text = {
                if ctx.role == .called { return isVideo ? "邀请你视频通话…" : "邀请你语音通话…" }
                return "正在等待对方接听…"
            }()
        }
        remoteView.isHidden = !(isVideo && inCall)
        localView.isHidden = !isVideo
        waitingStack.isHidden = inCall
        inCallStack.isHidden = !inCall
        acceptButton.isHidden = !(ctx.role == .called && !inCall)
        rejectButton.isHidden = !(ctx.role == .called && !inCall)
        cancelButton.isHidden = !(ctx.role == .caller && !inCall)
        hangupButton.isHidden = !inCall
        cameraFlipButton.isHidden = !isVideo
        if inCall && !speakerOn && isVideo {
            speakerOn = true
            LinLanTrtcProvider.shared.setSpeakerEnabled(true)
        }
        refreshControlColors()
    }

    func onRemoteUserJoined() {
        refreshFromManager()
    }

    func dismissForEnd(_ reason: LinLanRtcCallEndReason) {
        stopDurationTimer()
        dismiss(animated: true)
    }

    private func startDurationTimer() {
        if durationTimer != nil { return }
        durationTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            self?.refreshDurationText()
        }
    }

    private func stopDurationTimer() {
        durationTimer?.invalidate()
        durationTimer = nil
    }

    private func refreshDurationText() {
        guard LinLanRtcCallManager.shared.status == .accept else { return }
        statusLabel.text = LinLanRtcSignalingPayload.formatClock(LinLanRtcCallManager.shared.connectedElapsedSec())
    }

    private func setupUI() {
        remoteView.backgroundColor = .black
        localView.backgroundColor = UIColor(white: 0.15, alpha: 1)
        localView.layer.cornerRadius = 8
        localView.clipsToBounds = true

        nameLabel.font = .systemFont(ofSize: 26, weight: .semibold)
        nameLabel.textColor = .white
        nameLabel.textAlignment = .center
        statusLabel.font = .systemFont(ofSize: 15)
        statusLabel.textColor = UIColor(white: 0.8, alpha: 1)
        statusLabel.textAlignment = .center
        avatarLetter.font = .systemFont(ofSize: 36, weight: .bold)
        avatarLetter.textColor = .white
        avatarLetter.textAlignment = .center
        avatarLetter.backgroundColor = AppTheme.accent
        avatarLetter.layer.cornerRadius = 44
        avatarLetter.clipsToBounds = true

        styleAction(acceptButton, title: "接听", color: UIColor.systemGreen)
        styleAction(rejectButton, title: "拒绝", color: UIColor.systemRed)
        styleAction(cancelButton, title: "取消", color: UIColor.systemRed)
        styleAction(hangupButton, title: "挂断", color: UIColor.systemRed)
        styleIcon(micButton, system: "mic.fill")
        styleIcon(speakerButton, system: "speaker.wave.2.fill")
        styleIcon(cameraFlipButton, system: "camera.rotate")

        acceptButton.addTarget(self, action: #selector(onAccept), for: .touchUpInside)
        rejectButton.addTarget(self, action: #selector(onReject), for: .touchUpInside)
        cancelButton.addTarget(self, action: #selector(onCancel), for: .touchUpInside)
        hangupButton.addTarget(self, action: #selector(onHangup), for: .touchUpInside)
        micButton.addTarget(self, action: #selector(onMic), for: .touchUpInside)
        speakerButton.addTarget(self, action: #selector(onSpeaker), for: .touchUpInside)
        cameraFlipButton.addTarget(self, action: #selector(onFlip), for: .touchUpInside)

        waitingStack.axis = .horizontal
        waitingStack.spacing = 28
        waitingStack.distribution = .fillEqually
        waitingStack.addArrangedSubview(rejectButton)
        waitingStack.addArrangedSubview(acceptButton)
        waitingStack.addArrangedSubview(cancelButton)

        inCallStack.axis = .horizontal
        inCallStack.spacing = 20
        inCallStack.distribution = .fillEqually
        inCallStack.addArrangedSubview(micButton)
        inCallStack.addArrangedSubview(speakerButton)
        inCallStack.addArrangedSubview(cameraFlipButton)
        inCallStack.addArrangedSubview(hangupButton)

        view.addSubview(remoteView)
        view.addSubview(avatarLetter)
        view.addSubview(nameLabel)
        view.addSubview(statusLabel)
        view.addSubview(localView)
        view.addSubview(waitingStack)
        view.addSubview(inCallStack)

        remoteView.snp.makeConstraints { $0.edges.equalToSuperview() }
        avatarLetter.snp.makeConstraints { make in
            make.centerX.equalToSuperview()
            make.top.equalTo(view.safeAreaLayoutGuide).offset(80)
            make.size.equalTo(88)
        }
        nameLabel.snp.makeConstraints { make in
            make.top.equalTo(avatarLetter.snp.bottom).offset(16)
            make.leading.trailing.equalToSuperview().inset(24)
        }
        statusLabel.snp.makeConstraints { make in
            make.top.equalTo(nameLabel.snp.bottom).offset(8)
            make.leading.trailing.equalToSuperview().inset(24)
        }
        localView.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.top.equalTo(view.safeAreaLayoutGuide).offset(16)
            make.width.equalTo(110)
            make.height.equalTo(160)
        }
        waitingStack.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview().inset(40)
            make.bottom.equalTo(view.safeAreaLayoutGuide).offset(-28)
            make.height.equalTo(64)
        }
        inCallStack.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview().inset(24)
            make.bottom.equalTo(view.safeAreaLayoutGuide).offset(-28)
            make.height.equalTo(64)
        }
    }

    private func styleAction(_ button: UIButton, title: String, color: UIColor) {
        button.setTitle(title, for: .normal)
        button.setTitleColor(.white, for: .normal)
        button.backgroundColor = color
        button.layer.cornerRadius = 32
        button.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
    }

    private func styleIcon(_ button: UIButton, system: String) {
        button.setImage(UIImage(systemName: system), for: .normal)
        button.tintColor = .white
        button.backgroundColor = UIColor(white: 0.25, alpha: 1)
        button.layer.cornerRadius = 28
    }

    private func refreshControlColors() {
        micButton.backgroundColor = micOn ? UIColor(white: 0.25, alpha: 1) : UIColor.systemRed
        speakerButton.backgroundColor = speakerOn ? AppTheme.accent : UIColor(white: 0.25, alpha: 1)
    }

    @objc private func onAccept() { LinLanRtcCallManager.shared.acceptIncomingCall() }
    @objc private func onReject() { LinLanRtcCallManager.shared.rejectIncomingCall() }
    @objc private func onCancel() { LinLanRtcCallManager.shared.cancelOutgoingCall() }
    @objc private func onHangup() { LinLanRtcCallManager.shared.hangup() }
    @objc private func onMic() {
        micOn.toggle()
        LinLanTrtcProvider.shared.setMicEnabled(micOn)
        refreshControlColors()
    }
    @objc private func onSpeaker() {
        speakerOn.toggle()
        LinLanTrtcProvider.shared.setSpeakerEnabled(speakerOn)
        refreshControlColors()
    }
    @objc private func onFlip() {
        LinLanTrtcProvider.shared.switchCamera()
    }
}
