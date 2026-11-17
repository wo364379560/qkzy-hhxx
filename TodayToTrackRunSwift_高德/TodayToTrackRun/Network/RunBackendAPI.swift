//
//  RunBackendAPI.swift
//  TodayToTrackRun
//
//  二期接口契约：先本地，后接现有服务端账号模块。
//

import CoreLocation
import Foundation

/// 后端环境配置（二期填入真实 baseURL）
enum RunAPIConfig {
    /// 例如 https://api.example.com/api/v1/run
    static var baseURL: URL?
    static var isConfigured: Bool { baseURL != nil }
}

struct AuthTokenDTO: Codable, Equatable {
    let accessToken: String
    let userID: String
    let nickname: String?
}

struct RunUploadPointDTO: Codable, Equatable {
    let latitude: Double
    let longitude: Double
    let altitude: Double
    let speed: Double
    let recordedAt: TimeInterval
    let sequence: Int
}

struct RunUploadRequestDTO: Codable, Equatable {
    /// 本地主键，便于服务端做幂等
    let localID: Int64
    let startedAt: TimeInterval
    let endedAt: TimeInterval
    let durationSeconds: Double
    let distanceMeters: Double
    let averagePaceSecondsPerKm: Double?
    let cityName: String?
    let caloriesKcal: Int
    let averageHeartRate: Int
    let points: [RunUploadPointDTO]
}

struct RunUploadResponseDTO: Codable, Equatable {
    let serverID: String
    let syncedAt: TimeInterval
}

struct RunRemoteSummaryDTO: Codable, Equatable {
    let serverID: String
    let startedAt: TimeInterval
    let endedAt: TimeInterval
    let durationSeconds: Double
    let distanceMeters: Double
    let averagePaceSecondsPerKm: Double?
    let cityName: String?
    let pointCount: Int
}

enum RunBackendError: LocalizedError {
    case notConfigured
    case unauthorized
    case server(String)

    var errorDescription: String? {
        switch self {
        case .notConfigured:
            return "二期接口尚未配置 baseURL"
        case .unauthorized:
            return "未登录或登录已过期"
        case .server(let message):
            return message
        }
    }
}

/// 跑步业务后端能力（登录 + 轨迹同步）
protocol RunBackendAPI: AnyObject {
    func login(phone: String, password: String) async throws -> AuthTokenDTO
    func register(phone: String, password: String, nickname: String) async throws -> AuthTokenDTO
    func uploadRun(_ body: RunUploadRequestDTO) async throws -> RunUploadResponseDTO
    func fetchRemoteRuns(since: Date?) async throws -> [RunRemoteSummaryDTO]
}

/// 默认桩：本地开发期直接返回 notConfigured，避免误调空接口
final class StubRunBackendAPI: RunBackendAPI {
    func login(phone: String, password: String) async throws -> AuthTokenDTO {
        throw RunBackendError.notConfigured
    }

    func register(phone: String, password: String, nickname: String) async throws -> AuthTokenDTO {
        throw RunBackendError.notConfigured
    }

    func uploadRun(_ body: RunUploadRequestDTO) async throws -> RunUploadResponseDTO {
        throw RunBackendError.notConfigured
    }

    func fetchRemoteRuns(since: Date?) async throws -> [RunRemoteSummaryDTO] {
        throw RunBackendError.notConfigured
    }
}

/// 二期可换成真实 HTTP 实现，并在 App 启动时注入
enum RunBackend {
    static var shared: RunBackendAPI = StubRunBackendAPI()
}

/// 本地 -> 上传 DTO 组装
enum RunSyncMapper {
    static func uploadRequest(from detail: RunRecordDetail) -> RunUploadRequestDTO {
        let s = detail.summary
        return RunUploadRequestDTO(
            localID: s.id,
            startedAt: s.startedAt.timeIntervalSince1970,
            endedAt: s.endedAt.timeIntervalSince1970,
            durationSeconds: s.durationSeconds,
            distanceMeters: s.distanceMeters,
            averagePaceSecondsPerKm: s.averagePaceSecondsPerKm,
            cityName: s.cityName,
            caloriesKcal: s.caloriesKcal,
            averageHeartRate: s.averageHeartRate,
            points: detail.points.enumerated().map { index, p in
                RunUploadPointDTO(
                    latitude: p.coordinate.latitude,
                    longitude: p.coordinate.longitude,
                    altitude: p.altitude,
                    speed: p.speed,
                    recordedAt: p.timestamp.timeIntervalSince1970,
                    sequence: index
                )
            }
        )
    }
}
