//
//  ReverseGeocoder.swift
//  TodayToTrackRun
//

import AMapSearchKit
import CoreLocation
import Foundation

/// 高德逆地理：坐标 -> 城市名（如「杭州市」）
final class ReverseGeocoder: NSObject {
    static let shared = ReverseGeocoder()

    private let search = AMapSearchAPI()
    private var continuation: CheckedContinuation<String, Error>?

    private override init() {
        super.init()
        search?.delegate = self
    }

    func cityName(for coordinate: CLLocationCoordinate2D) async throws -> String {
        try await withCheckedThrowingContinuation { cont in
            if continuation != nil {
                cont.resume(throwing: GeocodeError.busy)
                return
            }
            continuation = cont
            let request = AMapReGeocodeSearchRequest()
            request.location = AMapGeoPoint.location(
                withLatitude: coordinate.latitude,
                longitude: coordinate.longitude
            )
            request.requireExtension = false
            search?.aMapReGoecodeSearch(request)

            DispatchQueue.main.asyncAfter(deadline: .now() + 8) { [weak self] in
                guard let self, let pending = self.continuation else { return }
                self.continuation = nil
                pending.resume(throwing: GeocodeError.timeout)
            }
        }
    }

    enum GeocodeError: LocalizedError {
        case empty
        case timeout
        case busy
        case sdk

        var errorDescription: String? {
            switch self {
            case .empty: return "未解析到城市"
            case .timeout: return "逆地理超时"
            case .busy: return "逆地理进行中"
            case .sdk: return "逆地理失败"
            }
        }
    }
}

extension ReverseGeocoder: AMapSearchDelegate {
    func onReGeocodeSearchDone(_ request: AMapReGeocodeSearchRequest!, response: AMapReGeocodeSearchResponse!) {
        guard let pending = continuation else { return }
        continuation = nil
        let component = response?.regeocode?.addressComponent
        let city = (component?.city?.isEmpty == false ? component?.city : nil)
            ?? (component?.province?.isEmpty == false ? component?.province : nil)
            ?? (component?.district?.isEmpty == false ? component?.district : nil)
        if let city, !city.isEmpty {
            pending.resume(returning: city)
        } else {
            pending.resume(throwing: GeocodeError.empty)
        }
    }

    func aMapSearchRequest(_ request: Any!, didFailWithError error: Error!) {
        guard let pending = continuation else { return }
        continuation = nil
        pending.resume(throwing: error ?? GeocodeError.sdk)
    }
}
