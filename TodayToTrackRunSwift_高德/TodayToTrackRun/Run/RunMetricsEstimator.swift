//
//  RunMetricsEstimator.swift
//  TodayToTrackRun
//
//  无心率带时：用配速/时长估消耗与平均心率，并生成演示用心率曲线。
//

import Foundation

enum RunMetricsEstimator {
    /// 约 65 kcal / km（中等体重跑步经验值）
    static func estimatedCalories(distanceMeters: Double, durationSeconds: TimeInterval) -> Int {
        let byDistance = distanceMeters / 1000.0 * 65.0
        let byTime = durationSeconds / 60.0 * 8.0
        return max(0, Int(((byDistance + byTime) / 2.0).rounded()))
    }

    /// 用平均配速估平均心率（bpm）。配速越快，心率越高。
    static func estimatedAverageHeartRate(paceSecondsPerKm: Double?, durationSeconds: TimeInterval) -> Int {
        guard durationSeconds >= 30 else { return 0 }
        let pace = paceSecondsPerKm ?? 360
        // 7'/km → ~128，5'/km → ~155，4'/km → ~168
        let clamped = min(max(pace, 240), 540)
        let hr = 200.0 - clamped / 6.0
        return Int(min(175, max(110, hr)).rounded())
    }

    /// 生成一条带起伏的假心率序列，点数随时长变化（详情页曲线用）
    static func heartRateSeries(
        averageBPM: Int,
        durationSeconds: TimeInterval,
        sampleCount: Int = 48
    ) -> [Int] {
        guard averageBPM > 0, durationSeconds > 0 else { return [] }
        let count = max(12, min(sampleCount, Int(durationSeconds / 8) + 8))
        var series: [Int] = []
        series.reserveCapacity(count)
        for i in 0..<count {
            let t = Double(i) / Double(max(count - 1, 1))
            // 热身抬升 → 平台 → 冲刺略抬 → 收尾略降
            let envelope: Double
            if t < 0.12 {
                envelope = 0.82 + 0.18 * (t / 0.12)
            } else if t > 0.88 {
                envelope = 1.05 - 0.12 * ((t - 0.88) / 0.12)
            } else {
                envelope = 0.96 + 0.08 * sin(t * .pi * 4)
            }
            let wobble = sin(t * .pi * 11) * 4 + cos(t * .pi * 3.5) * 3
            let value = Double(averageBPM) * envelope + wobble
            series.append(Int(min(190, max(95, value)).rounded()))
        }
        return series
    }

    static func metrics(for snapshot: RunSessionSnapshot) -> (calories: Int, avgHR: Int) {
        let calories = estimatedCalories(
            distanceMeters: snapshot.distanceMeters,
            durationSeconds: snapshot.duration
        )
        let avgHR = estimatedAverageHeartRate(
            paceSecondsPerKm: snapshot.averagePaceSecondsPerKm,
            durationSeconds: snapshot.duration
        )
        return (calories, avgHR)
    }
}
