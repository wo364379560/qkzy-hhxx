//
//  WeatherModels.swift
//  TodayToTrackRun
//

import Foundation

struct WeatherIndex: Equatable {
    let emoji: String
    let title: String
    let value: String
}

struct WeatherFutureDay: Equatable {
    let week: String
    let emoji: String
    let weather: String
    let tempRange: String
}

struct WeatherInfo: Equatable {
    let city: String
    let temp: String
    let condition: String
    let tempRange: String
    let emoji: String
    let windDirection: String
    let humidity: String
    let windLevel: String
    let indices: [WeatherIndex]
    let future: [WeatherFutureDay]

    /// 跑步友好提示：优先用「晨练指数」
    var runAdvice: String {
        if let exercise = indices.first(where: { $0.title.contains("晨练") }) {
            return "晨练：\(exercise.value)"
        }
        return condition
    }

    static let placeholder = WeatherInfo(
        city: "本地",
        temp: "--",
        condition: "天气加载中",
        tempRange: "--",
        emoji: "🌤️",
        windDirection: "--",
        humidity: "--",
        windLevel: "--",
        indices: [
            WeatherIndex(emoji: "🏃", title: "晨练指数", value: "--")
        ],
        future: []
    )
}
