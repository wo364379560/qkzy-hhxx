//
//  BundledRunSeeder.swift
//  TodayToTrackRun
//
//  首次启动时从 App Bundle 中导入两条本地跑步记录。
//

import CoreLocation
import Foundation

enum BundledRunSeeder {
    private static let importedVersionKey = "bundledRunSeeder.importedVersion"

    private struct Payload: Decodable {
        let version: Int
        let runs: [Run]
    }

    private struct Run: Decodable {
        let latitude: Double
        let longitude: Double
        let laps: Double
        let paceSecondsPerKm: Double
        let hoursAgo: Double
        let cityName: String
    }

    /// 每个 JSON 版本仅导入一次；升级 JSON 的 version 后可再次导入新样例。
    @discardableResult
    static func importIfNeeded(bundle: Bundle = .main, defaults: UserDefaults = .standard) throws -> [Int64] {
        guard let url = bundle.url(forResource: "demoRuns", withExtension: "json") else {
            throw ImportError.resourceMissing
        }

        let payload = try JSONDecoder().decode(Payload.self, from: Data(contentsOf: url))
        guard defaults.integer(forKey: importedVersionKey) < payload.version else {
            return []
        }

        var ids: [Int64] = []
        for run in payload.runs.prefix(2) {
            let endedAt = Date().addingTimeInterval(-run.hoursAgo * 3600)
            let snapshot = MockRunGenerator.makeSnapshot(
                center: CLLocationCoordinate2D(latitude: run.latitude, longitude: run.longitude),
                laps: run.laps,
                paceSecondsPerKm: run.paceSecondsPerKm,
                endedAt: endedAt
            )
            ids.append(try RunRecordStore.save(snapshot: snapshot, cityName: run.cityName))
        }

        defaults.set(payload.version, forKey: importedVersionKey)
        return ids
    }

    enum ImportError: LocalizedError {
        case resourceMissing

        var errorDescription: String? {
            "未找到本地跑步数据 demoRuns.json"
        }
    }
}
