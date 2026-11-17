//
//  RunSyncService.swift
//  TodayToTrackRun
//

import Foundation

/// 二期同步入口：把本地未同步记录推到 RunBackendAPI
actor RunSyncService {
    static let shared = RunSyncService()

    func syncPending() async {
        guard RunAPIConfig.isConfigured else { return }
        do {
            let ids = try RunRecordStore.unsyncedIDs()
            for id in ids {
                let detail = try RunRecordStore.detail(runID: id)
                let body = RunSyncMapper.uploadRequest(from: detail)
                let response = try await RunBackend.shared.uploadRun(body)
                try RunRecordStore.markSynced(
                    runID: id,
                    serverID: response.serverID,
                    syncedAt: Date(timeIntervalSince1970: response.syncedAt)
                )
            }
        } catch {
            print("[RunSync] failed:", error.localizedDescription)
        }
    }
}
