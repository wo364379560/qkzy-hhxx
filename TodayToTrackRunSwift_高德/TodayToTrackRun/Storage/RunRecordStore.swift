//
//  RunRecordStore.swift
//  TodayToTrackRun
//

import CoreLocation
import Foundation
import GRDB

struct RunRecordSummary: Identifiable {
    let id: Int64
    let startedAt: Date
    let endedAt: Date
    let durationSeconds: TimeInterval
    let distanceMeters: CLLocationDistance
    let averagePaceSecondsPerKm: Double?
    let pointCount: Int
    let cityName: String?
    let serverID: String?
    let syncedAt: Date?
    let caloriesKcal: Int
    let averageHeartRate: Int
    /// 用于缩略图的抽样坐标
    let previewCoordinates: [CLLocationCoordinate2D]

    var isSynced: Bool { syncedAt != nil && serverID != nil }

    var formattedDistance: String {
        if distanceMeters >= 1000 {
            return String(format: "%.2f km", distanceMeters / 1000)
        }
        return String(format: "%.0f m", distanceMeters)
    }

    var formattedDuration: String {
        let total = Int(durationSeconds.rounded())
        let h = total / 3600
        let m = (total % 3600) / 60
        let s = total % 60
        if h > 0 { return String(format: "%d:%02d:%02d", h, m, s) }
        return String(format: "%02d:%02d", m, s)
    }

    var formattedPace: String {
        guard let pace = averagePaceSecondsPerKm, pace > 0, pace.isFinite else {
            return "--'--\"/km"
        }
        let total = Int(pace.rounded())
        return String(format: "%d'%02d\"/km", total / 60, total % 60)
    }
}

struct RunRecordDetail {
    let summary: RunRecordSummary
    let points: [TrackPoint]
}

enum RunRecordStore {
    static func save(snapshot: RunSessionSnapshot, cityName: String? = nil) throws -> Int64 {
        guard snapshot.points.count >= 2, snapshot.distanceMeters > 1 else {
            throw StoreError.tooShort
        }

        let startedAt = snapshot.points.first?.timestamp ?? Date().addingTimeInterval(-snapshot.duration)
        let endedAt = snapshot.points.last?.timestamp ?? Date()
        let metrics = RunMetricsEstimator.metrics(for: snapshot)

        return try AppDatabase.shared.write { db in
            try db.execute(
                sql: """
                INSERT INTO run_record (
                    started_at, ended_at, duration_seconds, distance_meters,
                    avg_pace_seconds_per_km, point_count, city_name, created_at,
                    server_id, synced_at, calories_kcal, avg_heart_rate
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?)
                """,
                arguments: [
                    startedAt,
                    endedAt,
                    snapshot.duration,
                    snapshot.distanceMeters,
                    snapshot.averagePaceSecondsPerKm,
                    snapshot.points.count,
                    cityName,
                    Date(),
                    metrics.calories,
                    metrics.avgHR
                ]
            )
            let runID = db.lastInsertedRowID
            for (index, point) in snapshot.points.enumerated() {
                try db.execute(
                    sql: """
                    INSERT INTO run_point (
                        run_id, latitude, longitude, altitude, speed, recorded_at, sequence
                    ) VALUES (?, ?, ?, ?, ?, ?, ?)
                    """,
                    arguments: [
                        runID,
                        point.coordinate.latitude,
                        point.coordinate.longitude,
                        point.altitude,
                        point.speed,
                        point.timestamp,
                        index
                    ]
                )
            }
            return runID
        }
    }

    static func updateCityName(runID: Int64, cityName: String) throws {
        try AppDatabase.shared.write { db in
            try db.execute(
                sql: "UPDATE run_record SET city_name = ? WHERE id = ?",
                arguments: [cityName, runID]
            )
        }
    }

    static func markSynced(runID: Int64, serverID: String, syncedAt: Date) throws {
        try AppDatabase.shared.write { db in
            try db.execute(
                sql: "UPDATE run_record SET server_id = ?, synced_at = ? WHERE id = ?",
                arguments: [serverID, syncedAt, runID]
            )
        }
    }

    static func unsyncedCount() throws -> Int {
        try AppDatabase.shared.read { db in
            try Int.fetchOne(
                db,
                sql: "SELECT COUNT(*) FROM run_record WHERE synced_at IS NULL"
            ) ?? 0
        }
    }

    static func unsyncedIDs(limit: Int = 50) throws -> [Int64] {
        try AppDatabase.shared.read { db in
            try Int64.fetchAll(
                db,
                sql: """
                SELECT id FROM run_record
                WHERE synced_at IS NULL
                ORDER BY started_at ASC
                LIMIT ?
                """,
                arguments: [limit]
            )
        }
    }

    static func listSummaries(limit: Int = 100) throws -> [RunRecordSummary] {
        try AppDatabase.shared.read { db in
            let rows = try Row.fetchAll(
                db,
                sql: """
                SELECT id, started_at, ended_at, duration_seconds, distance_meters,
                       avg_pace_seconds_per_km, point_count, city_name,
                       server_id, synced_at, calories_kcal, avg_heart_rate
                FROM run_record
                ORDER BY started_at DESC
                LIMIT ?
                """,
                arguments: [limit]
            )

            return try rows.map { row in
                let id: Int64 = row["id"]
                let preview = try previewCoordinates(db: db, runID: id)
                return makeSummary(row: row, preview: preview)
            }
        }
    }

    static func detail(runID: Int64) throws -> RunRecordDetail {
        try AppDatabase.shared.read { db in
            guard let row = try Row.fetchOne(
                db,
                sql: """
                SELECT id, started_at, ended_at, duration_seconds, distance_meters,
                       avg_pace_seconds_per_km, point_count, city_name,
                       server_id, synced_at, calories_kcal, avg_heart_rate
                FROM run_record WHERE id = ?
                """,
                arguments: [runID]
            ) else {
                throw StoreError.notFound
            }

            let pointRows = try Row.fetchAll(
                db,
                sql: """
                SELECT latitude, longitude, altitude, speed, recorded_at
                FROM run_point
                WHERE run_id = ?
                ORDER BY sequence ASC
                """,
                arguments: [runID]
            )

            let points: [TrackPoint] = pointRows.map { p in
                TrackPoint(
                    coordinate: CLLocationCoordinate2D(
                        latitude: p["latitude"],
                        longitude: p["longitude"]
                    ),
                    timestamp: p["recorded_at"],
                    altitude: p["altitude"],
                    speed: p["speed"]
                )
            }

            let summary = makeSummary(row: row, preview: points.map(\.coordinate))
            return RunRecordDetail(summary: summary, points: points)
        }
    }

    static func delete(runID: Int64) throws {
        try AppDatabase.shared.write { db in
            try db.execute(sql: "DELETE FROM run_record WHERE id = ?", arguments: [runID])
        }
    }

    private static func makeSummary(row: Row, preview: [CLLocationCoordinate2D]) -> RunRecordSummary {
        let calories = (row["calories_kcal"] as Int?) ?? 0
        let hr = (row["avg_heart_rate"] as Int?) ?? 0
        return RunRecordSummary(
            id: row["id"],
            startedAt: row["started_at"],
            endedAt: row["ended_at"],
            durationSeconds: row["duration_seconds"],
            distanceMeters: row["distance_meters"],
            averagePaceSecondsPerKm: row["avg_pace_seconds_per_km"],
            pointCount: row["point_count"],
            cityName: row["city_name"],
            serverID: row["server_id"],
            syncedAt: row["synced_at"],
            caloriesKcal: calories,
            averageHeartRate: hr,
            previewCoordinates: preview
        )
    }

    private static func previewCoordinates(db: Database, runID: Int64, maxCount: Int = 40) throws -> [CLLocationCoordinate2D] {
        let total = try Int.fetchOne(
            db,
            sql: "SELECT COUNT(*) FROM run_point WHERE run_id = ?",
            arguments: [runID]
        ) ?? 0
        guard total > 0 else { return [] }

        let step = max(1, total / maxCount)
        let rows = try Row.fetchAll(
            db,
            sql: """
            SELECT latitude, longitude FROM run_point
            WHERE run_id = ? AND (sequence % ?) = 0
            ORDER BY sequence ASC
            """,
            arguments: [runID, step]
        )
        return rows.map {
            CLLocationCoordinate2D(latitude: $0["latitude"], longitude: $0["longitude"])
        }
    }

    enum StoreError: LocalizedError {
        case tooShort
        case notFound

        var errorDescription: String? {
            switch self {
            case .tooShort: return "轨迹太短，未保存"
            case .notFound: return "记录不存在"
            }
        }
    }
}
