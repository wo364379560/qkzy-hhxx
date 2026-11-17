//
//  LinLanAppPrefs.swift
//  TodayToTrackRun
//

import Foundation

final class LinLanAppPrefs {
    private let defaults = UserDefaults.standard
    private let prefix = "linlan.prefs."

    var privacyAccepted: Bool {
        get { defaults.bool(forKey: key("privacy_accepted")) }
        set { defaults.set(newValue, forKey: key("privacy_accepted")) }
    }

    var teenModeEnabled: Bool {
        get { defaults.bool(forKey: key("teen_mode")) }
        set { defaults.set(newValue, forKey: key("teen_mode")) }
    }

    var teenModePin: String {
        get { defaults.string(forKey: key("teen_pin")) ?? "" }
        set { defaults.set(newValue, forKey: key("teen_pin")) }
    }

    var messageNotifyEnabled: Bool {
        get {
            if defaults.object(forKey: key("msg_notify")) == nil { return true }
            return defaults.bool(forKey: key("msg_notify"))
        }
        set { defaults.set(newValue, forKey: key("msg_notify")) }
    }

    var soundNotifyEnabled: Bool {
        get {
            if defaults.object(forKey: key("sound_notify")) == nil { return true }
            return defaults.bool(forKey: key("sound_notify"))
        }
        set { defaults.set(newValue, forKey: key("sound_notify")) }
    }

    var cachedNickname: String {
        get { defaults.string(forKey: key("cached_nickname")) ?? "" }
        set { defaults.set(newValue, forKey: key("cached_nickname")) }
    }

    var cachedSignalId: String {
        get { defaults.string(forKey: key("cached_signal_id")) ?? "" }
        set { defaults.set(newValue, forKey: key("cached_signal_id")) }
    }

    var accountPhone: String {
        get { defaults.string(forKey: key("account_phone")) ?? "" }
        set { defaults.set(newValue, forKey: key("account_phone")) }
    }

    private func key(_ name: String) -> String { prefix + name }
}
