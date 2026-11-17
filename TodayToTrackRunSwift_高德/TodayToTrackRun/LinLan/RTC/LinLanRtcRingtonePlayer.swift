//
//  LinLanRtcRingtonePlayer.swift
//  TodayToTrackRun
//

import AudioToolbox
import AVFoundation
import Foundation

enum LinLanRtcRingtoneKind {
    case outgoing
    case incoming
}

enum LinLanRtcRingtonePlayer {
    private static var timer: Timer?
    private static var soundId: SystemSoundID = 0

    static func play(_ kind: LinLanRtcRingtoneKind) {
        stop()
        let id: SystemSoundID = kind == .incoming ? 1005 : 1002
        soundId = id
        timer = Timer.scheduledTimer(withTimeInterval: 2.4, repeats: true) { _ in
            AudioServicesPlaySystemSound(id)
        }
        timer?.fire()
    }

    static func stop() {
        timer?.invalidate()
        timer = nil
        if soundId != 0 {
            AudioServicesDisposeSystemSoundID(soundId)
            soundId = 0
        }
    }
}
