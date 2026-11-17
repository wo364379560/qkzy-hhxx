#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
work=$(mktemp -d /tmp/linlan-quote.XXXXXX)
trap 'rm -rf "$work"' EXIT
cat > "$work/check.swift" <<'SWIFT'
import Foundation
enum Elem: Int { case text = 1 }
final class V2TIMMessage {
    var cloudCustomData: Data?
    var msgID = "ios-original"
    var timestamp = Date(timeIntervalSince1970: 1700000000)
    var seq: UInt = 42
    var elemType = Elem.text
    var nameCard: String?
    var friendRemark: String?
    var nickName: String?
    var sender: String?
    var userID: String?
}
enum LinLanImManager {
    static func messagePreview(_ message: V2TIMMessage) -> String { "带引号\"和换行\n的原消息 😀" }
}
SWIFT
sed '/^import ImSDK_Plus_Swift$/d' TodayToTrackRun/LinLan/IM/LinLanQuoteHelper.swift >> "$work/check.swift"
cat >> "$work/check.swift" <<'SWIFT'
let message = V2TIMMessage()
let encoded = LinLanQuoteHelper.embedQuoteCloudCustomData(quoted: message, senderName: "群名片")!
let root = try JSONSerialization.jsonObject(with: encoded) as! [String: Any]
let web = root["messageReply"] as! [String: Any]
assert(web["messageID"] as? String == "ios-original")
assert(web["messageTime"] as? Int == 1700000000)
assert(web["messageSequence"] as? Int == 42)
assert(web["messageType"] as? Int == 1)
for key in ["messageReply", "atomicx_quote_v1"] {
    let payload = try JSONSerialization.data(withJSONObject: [key: root[key]!])
    let quote = LinLanQuoteHelper.decodeQuote(payload)!
    assert(quote.quotedMsgId == "ios-original")
    assert(quote.displayLine == "群名片: 带引号\"和换行\n的原消息 😀")
}
let fallback = try JSONSerialization.data(withJSONObject: ["messageReply": ["messageID": NSNull()], "atomicx_quote_v1": root["atomicx_quote_v1"]!])
assert(LinLanQuoteHelper.decodeQuote(fallback)?.quotedMsgId == "ios-original")
for invalid in ["", "null", "[]", "{", "{}", "{\"messageReply\":{\"messageID\":null}}"] {
    assert(LinLanQuoteHelper.decodeQuote(invalid.data(using: .utf8)!) == nil)
}
print("iOS quote interoperability: encode/decode, native and web envelopes, timestamps, Unicode and malformed metadata passed")
SWIFT
xcrun swift "$work/check.swift"
