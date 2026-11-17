//
//  AppDatabase.swift
//  TodayToTrackRun
//

import Foundation
import GRDB

enum AppDatabase {
    static var shared: DatabasePool!

    static func setup() throws {
        let folder = try FileManager.default.url(
            for: .applicationSupportDirectory,
            in: .userDomainMask,
            appropriateFor: nil,
            create: true
        ).appendingPathComponent("TodayToTrackRun", isDirectory: true)
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)

        let dbURL = folder.appendingPathComponent("runs.sqlite")
        var config = Configuration()
        #if DEBUG
        config.prepareDatabase { db in
            db.trace { print("[GRDB]", $0) }
        }
        #endif

        let pool = try DatabasePool(path: dbURL.path, configuration: config)
        try migrator.migrate(pool)
        shared = pool
    }

    private static var migrator: DatabaseMigrator {
        var migrator = DatabaseMigrator()
        migrator.registerMigration("v1_run_tables") { db in
            try db.create(table: "run_record") { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("started_at", .datetime).notNull()
                t.column("ended_at", .datetime).notNull()
                t.column("duration_seconds", .double).notNull()
                t.column("distance_meters", .double).notNull()
                t.column("avg_pace_seconds_per_km", .double)
                t.column("point_count", .integer).notNull().defaults(to: 0)
                t.column("city_name", .text)
                t.column("created_at", .datetime).notNull()
            }

            try db.create(table: "run_point") { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("run_id", .integer).notNull()
                    .references("run_record", onDelete: .cascade)
                t.column("latitude", .double).notNull()
                t.column("longitude", .double).notNull()
                t.column("altitude", .double).notNull().defaults(to: 0)
                t.column("speed", .double).notNull().defaults(to: 0)
                t.column("recorded_at", .datetime).notNull()
                t.column("sequence", .integer).notNull()
            }

            try db.create(
                index: "run_point_run_id_seq",
                on: "run_point",
                columns: ["run_id", "sequence"]
            )
        }

        // 二期同步预留：服务端 id + 同步时间
        migrator.registerMigration("v2_sync_fields") { db in
            try db.alter(table: "run_record") { t in
                t.add(column: "server_id", .text)
                t.add(column: "synced_at", .datetime)
            }
            try db.create(
                index: "run_record_unsynced",
                on: "run_record",
                columns: ["synced_at"]
            )
        }

        // 估消耗 / 平均心率（无手表时本地估算）
        migrator.registerMigration("v3_run_metrics") { db in
            try db.alter(table: "run_record") { t in
                t.add(column: "calories_kcal", .integer).defaults(to: 0)
                t.add(column: "avg_heart_rate", .integer).defaults(to: 0)
            }
        }
        return migrator
    }
}
