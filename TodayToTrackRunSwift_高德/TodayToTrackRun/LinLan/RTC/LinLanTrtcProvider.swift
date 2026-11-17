//
//  LinLanTrtcProvider.swift
//  TodayToTrackRun
//
//  对照 Android LinLanTrtcProvider：LiteAVSDK_TRTC 直连，不用 TUICallKit
//

import Foundation
import TXLiteAVSDK_TRTC
import UIKit

final class LinLanTrtcProvider: NSObject, TRTCCloudDelegate {
    static let shared = LinLanTrtcProvider()

    private var cloud: TRTCCloud?
    private(set) var isInRoom = false
    private(set) var remoteUserId: String?

    var onRemoteUserJoined: ((String) -> Void)?
    var onRemoteUserLeft: ((String) -> Void)?

    func attach() {
        if cloud != nil { return }
        let instance = TRTCCloud.sharedInstance()
        instance.addDelegate(self)
        cloud = instance
    }

    func enterRoom(
        roomId: String,
        userId: String,
        userSig: String,
        sdkAppId: Int,
        mediaType: LinLanRtcCallMediaType,
        localPreview: UIView?,
        remoteView: UIView?
    ) {
        attach()
        guard let cloud else { return }
        let params = TRTCParams()
        params.sdkAppId = UInt32(sdkAppId)
        params.userId = userId
        params.userSig = userSig
        params.strRoomId = roomId
        let scene: TRTCAppScene = mediaType == .video ? .videoCall : .audioCall
        cloud.enterRoom(params, appScene: scene)
        cloud.startLocalAudio(.default)
        if mediaType == .video, let localPreview {
            cloud.startLocalPreview(true, view: localPreview)
        } else {
            cloud.stopLocalPreview()
        }
        remoteUserId = nil
        isInRoom = true
        _ = remoteView
    }

    func exitRoom() {
        guard let cloud else { return }
        cloud.stopLocalPreview()
        cloud.stopLocalAudio()
        cloud.stopAllRemoteView()
        cloud.exitRoom()
        isInRoom = false
        remoteUserId = nil
    }

    func setMicEnabled(_ enabled: Bool) {
        cloud?.muteLocalAudio(!enabled)
    }

    func setSpeakerEnabled(_ enabled: Bool) {
        cloud?.setAudioRoute(enabled ? .modeSpeakerphone : .modeEarpiece)
    }

    func switchCamera() {
        guard let dm = cloud?.getDeviceManager() else { return }
        dm.switchCamera(!dm.isFrontCamera())
    }

    func setVideoEnabled(_ enabled: Bool) {
        cloud?.muteLocalVideo(.big, mute: !enabled)
    }

    func bindRemoteView(_ view: UIView, userId: String) {
        remoteUserId = userId
        cloud?.startRemoteView(userId, streamType: .big, view: view)
    }

    func rebindMediaViews(
        localPreview: UIView?,
        remoteView: UIView?,
        mediaType: LinLanRtcCallMediaType,
        remoteUserId: String?
    ) {
        if mediaType == .video, let localPreview {
            cloud?.startLocalPreview(true, view: localPreview)
        }
        if let remoteUserId, !remoteUserId.isEmpty, let remoteView {
            bindRemoteView(remoteView, userId: remoteUserId)
        }
    }

    func onRemoteUserEnterRoom(_ userId: String) {
        DispatchQueue.main.async { self.onRemoteUserJoined?(userId) }
    }

    func onRemoteUserLeaveRoom(_ userId: String, reason: Int) {
        DispatchQueue.main.async { self.onRemoteUserLeft?(userId) }
    }

    func onUserVideoAvailable(_ userId: String, available: Bool) {
        DispatchQueue.main.async {
            if available {
                self.onRemoteUserJoined?(userId)
            } // A disabled camera is not a room departure.
        }
    }
}
