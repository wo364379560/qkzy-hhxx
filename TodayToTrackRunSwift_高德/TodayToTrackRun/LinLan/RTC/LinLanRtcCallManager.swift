//
//  LinLanRtcCallManager.swift
//  TodayToTrackRun
//
//  对照 Android LinLanRtcCallManager：IM 信令 + TRTC 房间
//

import AVFoundation
import Foundation
import ImSDK_Plus_Swift
import UIKit

protocol LinLanRtcCallUiBridge: AnyObject {
    func localPreviewView() -> UIView?
    func remoteVideoView() -> UIView?
    func isViewReady() -> Bool
    var onViewReady: (() -> Void)? { get set }
    func refreshFromManager()
    func onRemoteUserJoined()
    func dismissForEnd(_ reason: LinLanRtcCallEndReason)
}

final class LinLanRtcCallManager: NSObject, V2TIMSignalingListener {
    static let shared = LinLanRtcCallManager()
    private static let inviteTimeoutSec = 45

    private let provider = LinLanTrtcProvider.shared
    private var signalingBound = false
    private var ringTimeout: Timer?
    private var callConnectedAt: Date?
    private var endingBySelf = false

    private(set) var status: LinLanRtcCallStatus = .none
    private(set) var callContext: LinLanRtcCallContext?
    private var selfUserId = ""
    private var selfDisplayName = "我"
    private var selfAvatarUrl: String?
    private var userSig = ""
    private var sdkAppId = LinLanAppConstants.sdkAppId

    weak var activeUi: LinLanRtcCallUiBridge?

    private override init() {
        super.init()
    }

    func setup(userId: String, userSig: String, sdkAppId: Int, displayName: String?) {
        selfUserId = userId
        self.userSig = userSig
        if sdkAppId > 0 { self.sdkAppId = sdkAppId }
        if let displayName, !displayName.isEmpty { selfDisplayName = displayName }
        provider.attach()
        if !signalingBound {
            V2TIMManager.shared.addSignalingListener(listener: self)
            signalingBound = true
        }
        provider.onRemoteUserJoined = { [weak self] uid in
            self?.handleRemoteJoined(uid)
        }
        provider.onRemoteUserLeft = { [weak self] _ in
            self?.finishCall(.hangup)
        }
    }

    func updateSelfProfile(name: String?, avatarUrl: String?) {
        if let name, !name.isEmpty { selfDisplayName = name }
        selfAvatarUrl = avatarUrl
    }

    func refreshLoginState(userId: String, userSig: String, sdkAppId: Int? = nil) {
        selfUserId = userId
        self.userSig = userSig
        if let sdkAppId, sdkAppId > 0 { self.sdkAppId = sdkAppId }
    }

    func startCall(peerUserId: String, peerName: String, mediaType: LinLanRtcCallMediaType) {
        if selfUserId.isEmpty || userSig.isEmpty {
            let session = LinLanAppContainer.authRepository().currentSession()
            refreshLoginState(userId: session.userId, userSig: session.userSig, sdkAppId: session.resolvedSdkAppId)
        }
        guard !selfUserId.isEmpty else {
            toast("请先登录")
            return
        }
        guard status == .none else {
            toast("当前正在通话中")
            return
        }
        guard !peerUserId.isEmpty, peerUserId != selfUserId else { return }
        let roomId = makeRoomId(caller: selfUserId, callee: peerUserId)
        guard let data = encodeSignaling(action: .invite, mediaType: mediaType, roomId: roomId) else { return }
        status = .waiting
        let inviteId = V2TIMManager.shared.invite(
            invitee: peerUserId,
            data: data,
            onlineUserOnly: false,
            offlinePushInfo: nil,
            timeout: Self.inviteTimeoutSec,
            succ: nil,
            fail: { [weak self] _, desc in
                DispatchQueue.main.async {
                    self?.resetCall()
                    self?.toast(desc ?? "发起通话失败")
                }
            }
        )
        guard let inviteId, !inviteId.isEmpty else {
            resetCall()
            toast("发起通话失败")
            return
        }
        callContext = LinLanRtcCallContext(
            inviteId: inviteId,
            roomId: roomId,
            peer: LinLanRtcCallPeer(userId: peerUserId, displayName: peerName, avatarUrl: nil),
            mediaType: mediaType,
            role: .caller
        )
        LinLanRtcRingtonePlayer.play(.outgoing)
        presentCallUi()
        startRingTimeout()
    }

    func acceptIncomingCall() {
        guard let ctx = callContext, status == .waiting, ctx.role == .called else { return }
        let data = encodeSignaling(action: .accept, mediaType: ctx.mediaType, roomId: ctx.roomId)
        V2TIMManager.shared.accept(inviteID: ctx.inviteId, data: data, succ: { [weak self] in
            DispatchQueue.main.async { self?.enterRtcAfterAccepted() }
        }, fail: { [weak self] _, desc in
            DispatchQueue.main.async { self?.toast(desc ?? "接听失败") }
        })
    }

    func rejectIncomingCall() {
        guard let ctx = callContext, ctx.role == .called else { return }
        let data = encodeSignaling(action: .reject, mediaType: ctx.mediaType, roomId: ctx.roomId)
        V2TIMManager.shared.reject(inviteID: ctx.inviteId, data: data, succ: nil, fail: nil)
        finishCall(.reject)
    }

    func cancelOutgoingCall() {
        guard let ctx = callContext, ctx.role == .caller, status == .waiting else { return }
        let data = encodeSignaling(action: .cancel, mediaType: ctx.mediaType, roomId: ctx.roomId)
        V2TIMManager.shared.cancel(inviteID: ctx.inviteId, data: data, succ: nil, fail: nil)
        finishCall(.cancel)
    }

    func hangup() {
        endingBySelf = true
        finishCall(.hangup)
    }

    func connectedElapsedSec() -> Int {
        guard let start = callConnectedAt else { return 0 }
        return max(0, Int(Date().timeIntervalSince(start)))
    }

    func enterRtcAfterAccepted() {
        guard status == .waiting, let ctx = callContext else { return }
        if selfUserId.isEmpty || userSig.isEmpty {
            let session = LinLanAppContainer.authRepository().currentSession()
            refreshLoginState(userId: session.userId, userSig: session.userSig, sdkAppId: session.resolvedSdkAppId)
        }
        guard !selfUserId.isEmpty, !userSig.isEmpty else {
            toast("UserSig 无效")
            finishCall(.hangup)
            return
        }
        doEnterRoom(ctx)
    }

    func onReceiveNewInvitation(inviteID: String, inviter: String, groupID: String?, inviteeList: [String], data: String?) {
        if let groupID, !groupID.isEmpty { return }
        DispatchQueue.main.async { self.handleNewInvitation(inviteID, from: inviter, data: data) }
    }

    func onInviteeAccepted(inviteID: String, invitee: String, data: String?) {
        DispatchQueue.main.async { self.handleInviteAccepted(inviteID) }
    }

    func onInviteeRejected(inviteID: String, invitee: String, data: String?) {
        DispatchQueue.main.async { self.handleInviteRejected(inviteID) }
    }

    func onInvitationCancelled(inviteID: String, inviter: String, data: String?) {
        DispatchQueue.main.async { self.handleInviteCancelled(inviteID) }
    }

    func onInvitationTimeout(inviteID: String, inviteeList: [String]) {
        DispatchQueue.main.async { self.handleInviteTimeout(inviteID) }
    }

    func logoutCleanup() async {
        await MainActor.run {
            if self.status != .none { self.hangup() }
        }
    }

    static func requestCallPermissions(needCamera: Bool) async -> Bool {
        let mic = await AVCaptureDevice.requestAccess(for: .audio)
        guard mic else { return false }
        if needCamera {
            return await AVCaptureDevice.requestAccess(for: .video)
        }
        return true
    }

    private func doEnterRoom(_ ctx: LinLanRtcCallContext) {
        status = .accept
        if callConnectedAt == nil {
            callConnectedAt = Date()
        }
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        if activeUi == nil { presentCallUi() }
        // Audio must not depend on viewDidAppear; the UI can bind video surfaces later.
        provider.enterRoom(
            roomId: ctx.roomId, userId: selfUserId, userSig: userSig, sdkAppId: sdkAppId,
            mediaType: ctx.mediaType, localPreview: activeUi?.localPreviewView(),
            remoteView: activeUi?.remoteVideoView()
        )
        provider.setSpeakerEnabled(ctx.mediaType == .video)
        activeUi?.refreshFromManager()
    }

    private func handleNewInvitation(_ inviteId: String, from inviter: String, data: String?) {
        if status != .none {
            V2TIMManager.shared.reject(inviteID: inviteId, data: nil, succ: nil, fail: nil)
            return
        }
        guard let payload = LinLanRtcSignalingPayload.parse(data) else { return }
        let name = payload.peerName?.isEmpty == false ? payload.peerName! : inviter
        callContext = LinLanRtcCallContext(
            inviteId: inviteId,
            roomId: payload.roomId,
            peer: LinLanRtcCallPeer(userId: inviter, displayName: name, avatarUrl: payload.peerAvatar),
            mediaType: payload.mediaType,
            role: .called
        )
        status = .waiting
        LinLanRtcRingtonePlayer.play(.incoming)
        presentCallUi()
    }

    private func handleInviteAccepted(_ inviteId: String) {
        guard let ctx = callContext, ctx.inviteId == inviteId, ctx.role == .caller else { return }
        enterRtcAfterAccepted()
    }

    private func handleInviteRejected(_ inviteId: String) {
        guard callContext?.inviteId == inviteId else { return }
        finishCall(.reject)
    }

    private func handleInviteCancelled(_ inviteId: String) {
        guard callContext?.inviteId == inviteId else { return }
        finishCall(.cancel)
    }

    private func handleInviteTimeout(_ inviteId: String) {
        guard callContext?.inviteId == inviteId else { return }
        finishCall(.noResponse)
    }

    private func handleRemoteJoined(_ userId: String) {
        guard let peerId = callContext?.peer.userId, userId == peerId else { return }
        if let view = activeUi?.remoteVideoView() {
            provider.bindRemoteView(view, userId: userId)
        }
        activeUi?.onRemoteUserJoined()
    }

    private func presentCallUi() {
        if activeUi != nil { return }
        guard let top = Self.topViewController() else { return }
        if top is LinLanRtcCallViewController { return }
        let vc = LinLanRtcCallViewController()
        vc.modalPresentationStyle = .fullScreen
        top.present(vc, animated: true)
    }

    private func finishCall(_ reason: LinLanRtcCallEndReason) {
        let ctx = callContext
        let connectedAt = callConnectedAt
        let sendHangupRecord = endingBySelf && reason == .hangup && ctx != nil && connectedAt != nil
        let duration = connectedAt.map { max(1, Int(Date().timeIntervalSince($0))) } ?? 0
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        callConnectedAt = nil
        endingBySelf = false
        if provider.isInRoom { provider.exitRoom() }
        activeUi?.dismissForEnd(reason)
        activeUi = nil
        callContext = nil
        status = .none
        if sendHangupRecord, let ctx {
            let payload = LinLanRtcSignalingPayload(
                action: .hangup,
                mediaType: ctx.mediaType,
                roomId: ctx.roomId,
                durationSec: duration
            )
            Task {
                _ = try? await LinLanImManager.sendRtcCallRecord(peerUserId: ctx.peer.userId, payload: payload)
            }
        }
        let message: String
        switch reason {
        case .hangup: message = "通话已结束"
        case .reject: message = "对方已拒绝"
        case .noResponse: message = "对方无应答"
        case .busy: message = "对方忙线中"
        case .cancel: message = "已取消"
        }
        toast(message)
    }

    private func resetCall() {
        LinLanRtcRingtonePlayer.stop()
        stopRingTimeout()
        callConnectedAt = nil
        endingBySelf = false
        callContext = nil
        status = .none
    }

    private func startRingTimeout() {
        stopRingTimeout()
        ringTimeout = Timer.scheduledTimer(withTimeInterval: TimeInterval(Self.inviteTimeoutSec), repeats: false) { [weak self] _ in
            guard let self, self.status == .waiting, let ctx = self.callContext, ctx.role == .caller else { return }
            let data = self.encodeSignaling(action: .noResponse, mediaType: ctx.mediaType, roomId: ctx.roomId)
            V2TIMManager.shared.cancel(inviteID: ctx.inviteId, data: data, succ: nil, fail: nil)
            self.finishCall(.noResponse)
        }
    }

    private func stopRingTimeout() {
        ringTimeout?.invalidate()
        ringTimeout = nil
    }

    private func makeRoomId(caller: String, callee: String) -> String {
        return UUID().uuidString
    }

    private func encodeSignaling(
        action: LinLanRtcSignalingAction,
        mediaType: LinLanRtcCallMediaType,
        roomId: String,
        durationSec: Int? = nil
    ) -> String? {
        LinLanRtcSignalingPayload(
            action: action,
            mediaType: mediaType,
            roomId: roomId,
            peerName: selfDisplayName,
            peerAvatar: selfAvatarUrl,
            durationSec: durationSec
        ).toJson()
    }

    private func toast(_ msg: String) {
        DispatchQueue.main.async {
            guard let top = Self.topViewController() else { return }
            let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
            top.present(a, animated: true)
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { a.dismiss(animated: true) }
        }
    }

    private static func topViewController() -> UIViewController? {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        let window = scenes.flatMap(\.windows).first(where: { $0.isKeyWindow }) ?? scenes.flatMap(\.windows).first
        var top = window?.rootViewController
        while let presented = top?.presentedViewController {
            top = presented
        }
        return top
    }
}
