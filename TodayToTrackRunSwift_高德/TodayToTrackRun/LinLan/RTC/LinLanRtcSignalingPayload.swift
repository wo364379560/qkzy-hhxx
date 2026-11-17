//
//  LinLanRtcSignalingPayload.swift
//  TodayToTrackRun
//
//  对照 Android LinLanRtcSignalingPayload：businessID = rtc_call_v1
//

import Foundation
import ImSDK_Plus_Swift

enum LinLanRtcSignalingAction: String {
    case invite
    case accept
    case reject
    case cancel
    case hangup
    case noResponse = "no_response"

    static func from(_ raw: String?) -> LinLanRtcSignalingAction {
        LinLanRtcSignalingAction(rawValue: raw ?? "") ?? .invite
    }
}

struct LinLanRtcSignalingPayload {
    static let business = "rtc_call_v1"
    static let currentVersion = 1

    var action: LinLanRtcSignalingAction
    var mediaType: LinLanRtcCallMediaType
    var roomId: String
    var peerName: String?
    var peerAvatar: String?
    var durationSec: Int?
    var businessId: String = business
    var version: Int = currentVersion

    func toJson() -> String? {
        var obj: [String: Any] = [
            "businessID": businessId,
            "action": action.rawValue,
            "version": version,
            "mediaType": mediaType.rawValue,
            "roomId": roomId,
        ]
        if let peerName, !peerName.isEmpty { obj["peerName"] = peerName }
        if let peerAvatar, !peerAvatar.isEmpty { obj["peerAvatar"] = peerAvatar }
        if let durationSec { obj["durationSec"] = durationSec }
        guard let data = try? JSONSerialization.data(withJSONObject: obj),
              let text = String(data: data, encoding: .utf8) else { return nil }
        return text
    }

    func displayText(isSelf: Bool) -> String {
        let media = mediaType == .video ? "视频" : "语音"
        switch action {
        case .invite:
            return isSelf ? "[发起\(media)通话]" : "[\(media)通话]"
        case .accept:
            return "[已接听]"
        case .reject:
            return isSelf ? "[已拒绝]" : "[对方已拒绝]"
        case .cancel:
            return isSelf ? "[已取消]" : "[对方已取消]"
        case .hangup:
            let sec = durationSec ?? 0
            if sec > 0 {
                return "通话时长 \(Self.formatClock(sec))"
            }
            return "[\(media)通话结束]"
        case .noResponse:
            return isSelf ? "[对方无应答]" : "[未接听]"
        }
    }

    static func formatClock(_ sec: Int) -> String {
        let safe = max(0, sec)
        return String(format: "%02d:%02d", safe / 60, safe % 60)
    }

    static func shouldHideInChat(_ payload: LinLanRtcSignalingPayload) -> Bool {
        payload.action == .invite || payload.action == .accept
    }

    static func parse(_ json: String?) -> LinLanRtcSignalingPayload? {
        guard let json, !json.isEmpty,
              let data = json.data(using: .utf8),
              let raw = try? JSONSerialization.jsonObject(with: data) else { return nil }
        guard let obj = unwrap(raw) else { return nil }
        let business = (obj["businessID"] as? String) ?? ""
        if business == Self.business || (obj["roomId"] != nil && obj["mediaType"] != nil) {
            let media = (obj["mediaType"] as? Int)
                ?? (obj["mediaType"] as? NSNumber)?.intValue
                ?? 1
            return LinLanRtcSignalingPayload(
                action: .from(obj["action"] as? String),
                mediaType: .from(media),
                roomId: (obj["roomId"] as? String) ?? "",
                peerName: (obj["peerName"] as? String)?.nilIfBlank,
                peerAvatar: (obj["peerAvatar"] as? String)?.nilIfBlank,
                durationSec: (obj["durationSec"] as? Int) ?? (obj["durationSec"] as? NSNumber)?.intValue,
                businessId: business.isEmpty ? Self.business : business,
                version: (obj["version"] as? Int) ?? currentVersion
            )
        }
        return nil
    }

    static func parseFromMessage(_ msg: V2TIMMessage) -> LinLanRtcSignalingPayload? {
        if let data = msg.customElem?.data,
           let raw = String(data: data, encoding: .utf8),
           let payload = parse(raw) {
            return payload
        }
        if let desc = msg.customElem?.desc, let payload = parse(desc) {
            return payload
        }
        if let info = V2TIMManager.shared.getSignallingInfo(msg: msg) {
            if let payload = parse(info.data) {
                var copied = payload
                copied.action = actionFromSignaling(info, fallback: payload.action)
                return copied
            }
            if !(info.inviteID ?? "").isEmpty || info.actionType.rawValue > 0 {
                return LinLanRtcSignalingPayload(
                    action: actionFromSignaling(info, fallback: .invite),
                    mediaType: .audio,
                    roomId: ""
                )
            }
        }
        return nil
    }

    private static func unwrap(_ raw: Any) -> [String: Any]? {
        guard let obj = raw as? [String: Any] else { return nil }
        let business = obj["businessID"] as? String
        if business == Self.business || (obj["roomId"] != nil && obj["mediaType"] != nil) {
            return obj
        }
        if let nested = obj["data"] as? String,
           nested.trimmingCharacters(in: .whitespacesAndNewlines).hasPrefix("{"),
           let data = nested.data(using: .utf8),
           let inner = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
            return inner
        }
        return obj
    }

    private static func actionFromSignaling(
        _ info: V2TIMSignalingInfo,
        fallback: LinLanRtcSignalingAction
    ) -> LinLanRtcSignalingAction {
        switch info.actionType {
        case .V2TIMSignalingActionType_Invite: return .invite
        case .V2TIMSignalingActionType_Cancel_Invite: return .cancel
        case .V2TIMSignalingActionType_Accept_Invite: return .accept
        case .V2TIMSignalingActionType_Reject_Invite: return .reject
        case .V2TIMSignalingActionType_Invite_Timeout: return .noResponse
        default: return fallback
        }
    }
}

private extension String {
    var nilIfBlank: String? {
        let t = trimmingCharacters(in: .whitespacesAndNewlines)
        return t.isEmpty ? nil : t
    }
}
