//
//  WeatherCityLocator.swift
//  TodayToTrackRun
//
//  对齐小确 JIzhangTodayDashboardVc：定位 → 逆地理 → 城市名，再去拉聚合天气。
//

import CoreLocation
import Foundation

enum WeatherCityLocator {
    /// 优先定位当前城市；失败则回退到已保存城市 / 杭州
    static func resolveCity() async -> String {
        do {
            let city = try await requestLocatedCity()
            let normalized = normalize(city)
            if !normalized.isEmpty {
                RunSettings.weatherCity = normalized
                return normalized
            }
        } catch {
            print("[Weather] locate failed:", error.localizedDescription)
        }
        let fallback = RunSettings.weatherCity.trimmingCharacters(in: .whitespacesAndNewlines)
        return fallback.isEmpty ? "杭州" : fallback
    }

    private static func normalize(_ city: String) -> String {
        var name = city.trimmingCharacters(in: .whitespacesAndNewlines)
        if name.hasSuffix("市") {
            name = String(name.dropLast())
        }
        return name
    }

    private static func requestLocatedCity() async throws -> String {
        let manager = OneShotLocationManager()
        let location = try await manager.requestLocation()
        let placemarks = try await CLGeocoder().reverseGeocodeLocation(location)
        if let locality = placemarks.first?.locality, !locality.isEmpty {
            return locality
        }
        if let admin = placemarks.first?.administrativeArea, !admin.isEmpty {
            return admin
        }
        throw LocateError.noCity
    }

    enum LocateError: LocalizedError {
        case denied
        case noCity
        case timeout

        var errorDescription: String? {
            switch self {
            case .denied: return "定位权限未开启"
            case .noCity: return "无法解析城市"
            case .timeout: return "定位超时"
            }
        }
    }
}

/// 一次性定位，用完即释放
private final class OneShotLocationManager: NSObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var continuation: CheckedContinuation<CLLocation, Error>?
    private var finished = false

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyKilometer
    }

    func requestLocation() async throws -> CLLocation {
        try await withCheckedThrowingContinuation { cont in
            self.continuation = cont
            switch manager.authorizationStatus {
            case .notDetermined:
                manager.requestWhenInUseAuthorization()
            case .authorizedWhenInUse, .authorizedAlways:
                manager.requestLocation()
            case .denied, .restricted:
                finish(.failure(WeatherCityLocator.LocateError.denied))
            @unknown default:
                finish(.failure(WeatherCityLocator.LocateError.denied))
            }

            DispatchQueue.main.asyncAfter(deadline: .now() + 8) { [weak self] in
                self?.finish(.failure(WeatherCityLocator.LocateError.timeout))
            }
        }
    }

    private func finish(_ result: Result<CLLocation, Error>) {
        guard !finished else { return }
        finished = true
        manager.stopUpdatingLocation()
        switch result {
        case .success(let loc):
            continuation?.resume(returning: loc)
        case .failure(let err):
            continuation?.resume(throwing: err)
        }
        continuation = nil
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            manager.requestLocation()
        case .denied, .restricted:
            finish(.failure(WeatherCityLocator.LocateError.denied))
        default:
            break
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        finish(.success(location))
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        finish(.failure(error))
    }
}
