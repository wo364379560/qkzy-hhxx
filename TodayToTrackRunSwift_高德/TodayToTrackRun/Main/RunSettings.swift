//
//  RunSettings.swift
//  TodayToTrackRun
//

import Foundation

enum RunSettings {
    private static let voiceKey = "settings.voiceBroadcast"
    private static let autoPauseKey = "settings.autoPause"
    private static let weatherCityKey = "settings.weatherCity"

    static var voiceBroadcastEnabled: Bool {
        get {
            if UserDefaults.standard.object(forKey: voiceKey) == nil { return true }
            return UserDefaults.standard.bool(forKey: voiceKey)
        }
        set { UserDefaults.standard.set(newValue, forKey: voiceKey) }
    }

    static var autoPauseEnabled: Bool {
        get {
            if UserDefaults.standard.object(forKey: autoPauseKey) == nil { return true }
            return UserDefaults.standard.bool(forKey: autoPauseKey)
        }
        set { UserDefaults.standard.set(newValue, forKey: autoPauseKey) }
    }

    static var weatherCity: String {
        get { UserDefaults.standard.string(forKey: weatherCityKey) ?? "杭州" }
        set { UserDefaults.standard.set(newValue, forKey: weatherCityKey) }
    }
}
