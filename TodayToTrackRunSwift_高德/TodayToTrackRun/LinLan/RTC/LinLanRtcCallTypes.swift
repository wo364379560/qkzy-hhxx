//
//  LinLanRtcCallTypes.swift
//  TodayToTrackRun
//
//  对照 Android LinLanRtcCallTypes
//

import Foundation

enum LinLanRtcCallStatus {
    case none
    case waiting
    case accept
}

enum LinLanRtcCallMediaType: Int {
    case audio = 1
    case video = 2

    static func from(_ value: Int) -> LinLanRtcCallMediaType {
        LinLanRtcCallMediaType(rawValue: value) ?? .audio
    }
}

enum LinLanRtcCallRole {
    case caller
    case called
}

enum LinLanRtcCallEndReason {
    case hangup
    case reject
    case noResponse
    case busy
    case cancel
}

struct LinLanRtcCallPeer {
    var userId: String
    var displayName: String
    var avatarUrl: String?
}

struct LinLanRtcCallContext {
    var inviteId: String
    var roomId: String
    var peer: LinLanRtcCallPeer
    var mediaType: LinLanRtcCallMediaType
    var role: LinLanRtcCallRole
}
