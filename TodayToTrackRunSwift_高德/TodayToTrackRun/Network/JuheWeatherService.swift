//
//  JuheWeatherService.swift
//  TodayToTrackRun
//
//  从 chixin demo 的 JuheService 移植：聚合天气 API。
//

import Foundation

actor JuheWeatherService {
    static let shared = JuheWeatherService()

    /// 与 Flutter demo 同一套聚合 Key（同主体产品复用）
    private let weatherKey = "009d51be2ee5e2d7ff0b28ce0865e61e"

    private(set) var currentCity = "杭州"
    private(set) var weather: WeatherInfo = .placeholder

    func refresh(city: String? = nil) async -> WeatherInfo {
        let target = city ?? currentCity
        if let fetched = await fetchWeather(city: target) {
            currentCity = fetched.city
            weather = fetched
            return fetched
        }
        return weather
    }

    private func fetchWeather(city: String) async -> WeatherInfo? {
        var components = URLComponents(string: "http://v.juhe.cn/weather/index")
        components?.queryItems = [
            URLQueryItem(name: "key", value: weatherKey),
            URLQueryItem(name: "cityname", value: city),
            URLQueryItem(name: "dtype", value: "json"),
            URLQueryItem(name: "format", value: "2")
        ]
        guard let url = components?.url else { return nil }
        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            guard
                let root = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                (root["error_code"] as? Int) == 0,
                let result = root["result"] as? [String: Any]
            else { return nil }
            return parse(result: result, fallbackCity: city)
        } catch {
            return nil
        }
    }

    private func parse(result: [String: Any], fallbackCity: String) -> WeatherInfo {
        let sk = result["sk"] as? [String: Any]
        let today = result["today"] as? [String: Any]
        let placeholder = WeatherInfo.placeholder

        let city = str(today?["city"]).isEmpty ? fallbackCity : str(today?["city"])
        var temp = str(sk?["temp"])
        if temp.isEmpty {
            temp = str(today?["temperature"])
                .components(separatedBy: "~")
                .first?
                .filter(\.isNumber) ?? ""
        }
        let condition = str(today?["weather"])
        let tempRange = str(today?["temperature"])
        let weatherId = today?["weather_id"] as? [String: Any]
        let emoji = emojiForCode(str(weatherId?["fa"]))

        let windDirection = [str(sk?["wind_direction"]), str(sk?["wind_strength"])]
            .filter { !$0.isEmpty }
            .joined(separator: " ")

        let indices: [WeatherIndex] = [
            .init(emoji: "👔", title: "穿衣指数", value: nonempty(str(today?["dressing_index"]))),
            .init(emoji: "☀️", title: "紫外线", value: nonempty(str(today?["uv_index"]))),
            .init(emoji: "🚗", title: "洗车指数", value: nonempty(str(today?["wash_index"]))),
            .init(emoji: "🏃", title: "晨练指数", value: nonempty(str(today?["exercise_index"]))),
            .init(emoji: "🧳", title: "旅游指数", value: nonempty(str(today?["travel_index"]))),
            .init(emoji: "😌", title: "舒适度", value: nonempty(str(today?["comfort_index"])))
        ]

        var future: [WeatherFutureDay] = []
        for (index, day) in futureDays(result["future"]).enumerated() {
            let week = index == 0 ? "今天" : nonempty(str(day["week"]), fallback: "未来")
            let dayWeatherId = day["weather_id"] as? [String: Any]
            future.append(
                WeatherFutureDay(
                    week: week,
                    emoji: emojiForCode(str(dayWeatherId?["fa"])),
                    weather: str(day["weather"]),
                    tempRange: str(day["temperature"])
                )
            )
        }

        return WeatherInfo(
            city: city,
            temp: temp.isEmpty ? placeholder.temp : temp,
            condition: condition.isEmpty ? placeholder.condition : condition,
            tempRange: tempRange.isEmpty ? placeholder.tempRange : tempRange,
            emoji: emoji,
            windDirection: windDirection.isEmpty ? placeholder.windDirection : windDirection,
            humidity: nonempty(str(sk?["humidity"]), fallback: placeholder.humidity),
            windLevel: nonempty(str(today?["wind"]), fallback: placeholder.windLevel),
            indices: indices,
            future: future
        )
    }

    private func futureDays(_ raw: Any?) -> [[String: Any]] {
        if let list = raw as? [[String: Any]] { return list }
        if let list = raw as? [Any] {
            return list.compactMap { $0 as? [String: Any] }
        }
        if let map = raw as? [String: Any] {
            return map.values.compactMap { $0 as? [String: Any] }
        }
        return []
    }

    private func str(_ value: Any?) -> String {
        (value as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    }

    private func nonempty(_ value: String, fallback: String = "--") -> String {
        value.isEmpty ? fallback : value
    }

    private func emojiForCode(_ code: String) -> String {
        switch code {
        case "00": return "☀️"
        case "01": return "🌤️"
        case "02": return "☁️"
        case "03", "04", "05": return "⛈️"
        case "06", "07", "08", "09", "10", "11", "12": return "🌧️"
        case "13", "14", "15", "16", "17": return "❄️"
        case "18": return "🌫️"
        case "19", "20": return "💨"
        default: return "🌤️"
        }
    }
}
