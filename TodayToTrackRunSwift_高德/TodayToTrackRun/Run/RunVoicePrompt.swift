//
//  RunVoicePrompt.swift
//  TodayToTrackRun
//
//  对齐 YSRun YSVoicePrompt：每公里播报本地 mp3。
//  资源在 girlmp3/，文件名历史拼写为 gril_（非 girl_）。
//

import AVFoundation
import CoreLocation
import Foundation

final class RunVoicePrompt {
    static let shared = RunVoicePrompt()

    private var player: AVAudioPlayer?
    private var promptedKilometers = Set<Int>()
    private let maxKilometerPrompt = 20
    private var sessionConfigured = false
    private var lastPaceCoachAt: Date?

    private init() {}

    func resetSession() {
        promptedKilometers.removeAll()
        lastPaceCoachAt = nil
        stop()
        configureAudioSessionIfNeeded()
    }

    func stop() {
        player?.stop()
        player = nil
    }

    func update(distanceMeters: CLLocationDistance) {
        guard RunSettings.voiceBroadcastEnabled else { return }
        let km = Int(distanceMeters / 1000.0)
        guard km >= 1, km <= maxKilometerPrompt else { return }
        guard !promptedKilometers.contains(km) else { return }
        promptedKilometers.insert(km)
        play(resourceName: "gril_\(km)km")
    }

    func coachIfNeeded(paceSecondsPerKm: Double?, duration: TimeInterval) {
        guard RunSettings.voiceBroadcastEnabled else { return }
        guard duration >= 180, let pace = paceSecondsPerKm, pace > 0 else { return }
        if let last = lastPaceCoachAt, Date().timeIntervalSince(last) < 180 { return }

        if pace < 240 {
            lastPaceCoachAt = Date()
            play(resourceName: "gril_strength_too_high_slowdown_prompt")
        } else if pace > 480 {
            lastPaceCoachAt = Date()
            play(resourceName: "gril_speed_up_prompt")
        } else if pace >= 300, pace <= 390, Int(duration) % 600 < 2 {
            lastPaceCoachAt = Date()
            play(resourceName: "gril_keep_burning_fat")
        }
    }

    private func configureAudioSessionIfNeeded() {
        guard !sessionConfigured else { return }
        do {
            let session = AVAudioSession.sharedInstance()
            try session.setCategory(.playback, mode: .spokenAudio, options: [.mixWithOthers, .duckOthers])
            try session.setActive(true)
            sessionConfigured = true
        } catch {
            print("[Voice] session failed:", error.localizedDescription)
        }
    }

    private func play(resourceName: String) {
        configureAudioSessionIfNeeded()
        guard let url = resolveURL(named: resourceName) else {
            print("[Voice] missing:", resourceName)
            return
        }
        do {
            let next = try AVAudioPlayer(contentsOf: url)
            next.prepareToPlay()
            player?.stop()
            player = next
            next.play()
        } catch {
            print("[Voice] play failed:", error.localizedDescription)
        }
    }

    private func resolveURL(named name: String) -> URL? {
        if let url = Bundle.main.url(forResource: name, withExtension: "mp3", subdirectory: "girlmp3") {
            return url
        }
        if let url = Bundle.main.url(forResource: name, withExtension: "mp3") {
            return url
        }
        let direct = Bundle.main.bundleURL
            .appendingPathComponent("girlmp3", isDirectory: true)
            .appendingPathComponent("\(name).mp3")
        if FileManager.default.fileExists(atPath: direct.path) {
            return direct
        }
        return nil
    }
}
