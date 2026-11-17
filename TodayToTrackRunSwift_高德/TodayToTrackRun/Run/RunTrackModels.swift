//
//  RunTrackModels.swift
//  TodayToTrackRun
//

import CoreLocation
import Foundation

struct TrackPoint: Equatable {
    let coordinate: CLLocationCoordinate2D
    let timestamp: Date
    let altitude: CLLocationDistance
    let speed: CLLocationSpeed

    static func == (lhs: TrackPoint, rhs: TrackPoint) -> Bool {
        lhs.coordinate.latitude == rhs.coordinate.latitude
            && lhs.coordinate.longitude == rhs.coordinate.longitude
            && lhs.timestamp == rhs.timestamp
    }
}

struct RunSessionSnapshot {
    let points: [TrackPoint]
    let distanceMeters: CLLocationDistance
    let duration: TimeInterval
    let isPaused: Bool
    let isRunning: Bool

    var averagePaceSecondsPerKm: Double? {
        guard distanceMeters >= 1 else { return nil }
        let km = distanceMeters / 1000
        guard km > 0 else { return nil }
        return duration / km
    }

    var formattedDistance: String {
        if distanceMeters >= 1000 {
            return String(format: "%.2f km", distanceMeters / 1000)
        }
        return String(format: "%.0f m", distanceMeters)
    }

    var formattedDuration: String {
        let total = Int(duration.rounded())
        let h = total / 3600
        let m = (total % 3600) / 60
        let s = total % 60
        if h > 0 {
            return String(format: "%d:%02d:%02d", h, m, s)
        }
        return String(format: "%02d:%02d", m, s)
    }

    var formattedPace: String {
        guard let pace = averagePaceSecondsPerKm, pace.isFinite, pace > 0 else {
            return "--'--\""
        }
        let total = Int(pace.rounded())
        let min = total / 60
        let sec = total % 60
        return String(format: "%d'%02d\"/km", min, sec)
    }
}
