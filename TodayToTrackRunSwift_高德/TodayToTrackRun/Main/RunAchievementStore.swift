//
//  RunAchievementStore.swift
//  TodayToTrackRun
//

import Foundation

struct RunAchievement: Equatable {
    let id: String
    let title: String
    let symbolName: String
    let unlocked: Bool
}

enum RunAchievementStore {
    static func achievements(from summaries: [RunRecordSummary]) -> [RunAchievement] {
        let totalDistance = summaries.reduce(0.0) { $0 + $1.distanceMeters }
        let totalCount = summaries.count
        let streak = GoalPreferences.streakDays()
        let bestPace = summaries.compactMap(\.averagePaceSecondsPerKm).min()
        let hasNight = summaries.contains { Calendar.current.component(.hour, from: $0.startedAt) >= 20 }

        return [
            RunAchievement(id: "first", title: "首次跑步", symbolName: "figure.run", unlocked: totalCount >= 1),
            RunAchievement(id: "streak7", title: "连续7天", symbolName: "flame.fill", unlocked: streak >= 7),
            RunAchievement(id: "10k", title: "10K达成", symbolName: "medal.fill", unlocked: totalDistance >= 10_000 || summaries.contains { $0.distanceMeters >= 10_000 }),
            RunAchievement(id: "pace5", title: "配速5分", symbolName: "bolt.fill", unlocked: (bestPace ?? 9999) <= 300),
            RunAchievement(id: "night", title: "夜跑达人", symbolName: "moon.stars.fill", unlocked: hasNight),
            RunAchievement(id: "half", title: "半马完赛", symbolName: "mountain.2.fill", unlocked: summaries.contains { $0.distanceMeters >= 21_097 })
        ]
    }

    static func cumulative(from summaries: [RunRecordSummary]) -> (km: Double, count: Int, kcal: Int) {
        let km = summaries.reduce(0.0) { $0 + $1.distanceMeters } / 1000.0
        let count = summaries.count
        let kcal = summaries.reduce(0) { partial, item in
            let value = item.caloriesKcal > 0
                ? item.caloriesKcal
                : RunMetricsEstimator.estimatedCalories(
                    distanceMeters: item.distanceMeters,
                    durationSeconds: item.durationSeconds
                )
            return partial + value
        }
        return (km, count, kcal)
    }
}
