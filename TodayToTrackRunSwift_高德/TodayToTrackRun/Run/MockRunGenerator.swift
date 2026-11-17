//
//  MockRunGenerator.swift
//  TodayToTrackRun
//
//  室内调试用：生成标准跑道椭圆轨迹，无需真跑。
//

import CoreLocation
import Foundation

enum MockRunGenerator {
    /// 杭州上城区体育中心附近（演示用）
    private static let defaultCenter = CLLocationCoordinate2D(latitude: 30.2532, longitude: 120.2105)

    /// 生成一圈约 400m 椭圆的 GPS 点
    static func ovalPoints(
        center: CLLocationCoordinate2D = defaultCenter,
        laps: Double = 5,
        pointsPerLap: Int = 40,
        paceSecondsPerKm: Double = 330,
        endedAt: Date = Date()
    ) -> (points: [TrackPoint], distance: CLLocationDistance, duration: TimeInterval) {
        // 约 400m 标准跑道：长短半轴近似（米转经纬度）
        let metersPerDegLat = 111_320.0
        let metersPerDegLon = 111_320.0 * cos(center.latitude * .pi / 180)
        let radiusLat = 50.0 / metersPerDegLat   // ~南北半轴
        let radiusLon = 80.0 / metersPerDegLon   // ~东西半轴

        let totalPoints = max(Int(laps * Double(pointsPerLap)), pointsPerLap)
        let lapMeters = 400.0
        let distance = laps * lapMeters
        let duration = (distance / 1000.0) * paceSecondsPerKm
        let startedAt = endedAt.addingTimeInterval(-duration)

        var points: [TrackPoint] = []
        points.reserveCapacity(totalPoints + 1)

        for i in 0...totalPoints {
            let progress = Double(i) / Double(pointsPerLap)
            let angle = progress * 2 * .pi - .pi / 2
            let lat = center.latitude + radiusLat * sin(angle)
            let lon = center.longitude + radiusLon * cos(angle)
            let t = startedAt.addingTimeInterval(duration * Double(i) / Double(totalPoints))
            let speed = 1000.0 / paceSecondsPerKm
            points.append(
                TrackPoint(
                    coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                    timestamp: t,
                    altitude: 12,
                    speed: speed
                )
            )
        }
        return (points, distance, duration)
    }

    static func makeSnapshot(
        center: CLLocationCoordinate2D = defaultCenter,
        laps: Double = 5,
        paceSecondsPerKm: Double = 330,
        endedAt: Date = Date()
    ) -> RunSessionSnapshot {
        let result = ovalPoints(
            center: center,
            laps: laps,
            paceSecondsPerKm: paceSecondsPerKm,
            endedAt: endedAt
        )
        return RunSessionSnapshot(
            points: result.points,
            distanceMeters: result.distance,
            duration: result.duration,
            isPaused: false,
            isRunning: false
        )
    }

    /// 写入 1～N 条演示记录（不同时间），方便列表/详情调试
    @discardableResult
    static func seedDemoRecords(count: Int = 3) throws -> [Int64] {
        var ids: [Int64] = []
        let samples: [(laps: Double, pace: Double, hoursAgo: Double, city: String)] = [
            (5, 323, 1, "杭州市"),
            (2.5, 340, 26, "杭州市"),
            (7.5, 310, 50, "杭州市")
        ]
        for (index, sample) in samples.prefix(count).enumerated() {
            let ended = Date().addingTimeInterval(-sample.hoursAgo * 3600)
            let snapshot = makeSnapshot(
                laps: sample.laps,
                paceSecondsPerKm: sample.pace,
                endedAt: ended
            )
            let id = try RunRecordStore.save(snapshot: snapshot, cityName: sample.city)
            ids.append(id)
            if index == 0 {
                GoalPreferences.markRunCompletedToday()
            }
        }
        return ids
    }
}
