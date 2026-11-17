//
//  LinLanQuoteHelper.swift
//  TodayToTrackRun
//

import Foundation
import ImSDK_Plus_Swift

struct LinLanMessageQuote {
    let quotedMsgId: String
    let displayLine: String
}

enum LinLanQuoteHelper {
    private static let cloudKey = "atomicx_quote_v1"

    static func resolveQuote(_ message: V2TIMMessage) -> LinLanMessageQuote? {
        guard let data = message.cloudCustomData else { return nil }
        return decodeQuote(data)
    }

    static func decodeQuote(_ data: Data) -> LinLanMessageQuote? {
        guard let root = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] else { return nil }
        for key in ["messageReply", cloudKey] {
            guard let payload = root[key] as? [String: Any] else { continue }
            let native = key == cloudKey
            let id = (payload[native ? "msgID" : "messageID"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !id.isEmpty else { continue }
            let sender = payload[native ? "msgSender" : "messageSender"] as? String ?? ""
            let summary = payload[native ? "msgAbstract" : "messageAbstract"] as? String ?? ""
            return LinLanMessageQuote(quotedMsgId: id, displayLine: displayLine(sender: sender, abstract: summary.isEmpty ? "[消息]" : summary))
        }
        return nil
    }

    static func embedQuoteCloudCustomData(quoted: V2TIMMessage, senderName: String) -> Data? {
        let summary = LinLanImManager.messagePreview(quoted)
        let root: [String: Any] = [
            cloudKey: ["msgID": quoted.msgID, "msgSender": senderName, "msgAbstract": summary,
                       "messageTime": Int(quoted.timestamp.timeIntervalSince1970), "messageSequence": quoted.seq, "messageType": quoted.elemType.rawValue],
            "messageReply": ["messageID": quoted.msgID, "messageSender": senderName, "messageAbstract": summary,
                             "messageTime": Int(quoted.timestamp.timeIntervalSince1970), "messageSequence": quoted.seq, "messageType": quoted.elemType.rawValue, "version": 1]
        ]
        return try? JSONSerialization.data(withJSONObject: root)
    }

    static func senderName(_ message: V2TIMMessage) -> String {
        if let s = message.nameCard?.trimmingCharacters(in: .whitespacesAndNewlines), !s.isEmpty { return s }
        if let s = message.friendRemark?.trimmingCharacters(in: .whitespacesAndNewlines), !s.isEmpty { return s }
        if let s = message.nickName?.trimmingCharacters(in: .whitespacesAndNewlines), !s.isEmpty { return s }
        return message.sender ?? message.userID ?? "未知"
    }

    private static func displayLine(sender: String, abstract: String) -> String {
        let name = sender.trimmingCharacters(in: .whitespacesAndNewlines)
        let content = abstract.trimmingCharacters(in: .whitespacesAndNewlines)
        if name.isEmpty { return content }
        if content.isEmpty { return name }
        return "\(name): \(content)"
    }
}
