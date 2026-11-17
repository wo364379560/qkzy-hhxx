//
//  RunLocationTracker.swift
//  TodayToTrackRun
//

import AMapLocationKit
import CoreLocation
import Foundation

protocol RunLocationTrackerDelegate: AnyObject {
    func runLocationTracker(_ tracker: RunLocationTracker, didUpdate snapshot: RunSessionSnapshot)
    func runLocationTracker(_ tracker: RunLocationTracker, didFail error: Error)
    func runLocationTrackerDidAutoPause(_ tracker: RunLocationTracker)
    func runLocationTrackerDidAutoResume(_ tracker: RunLocationTracker)
}

extension RunLocationTrackerDelegate {
    func runLocationTrackerDidAutoPause(_ tracker: RunLocationTracker) {}
    func runLocationTrackerDidAutoResume(_ tracker: RunLocationTracker) {}
}

/// 负责持续定位与轨迹点累积（高德 AMapLocation）。
final class RunLocationTracker: NSObject {
    weak var delegate: RunLocationTrackerDelegate?

    private let locationManager = AMapLocationManager()
    private var points: [TrackPoint] = []
    private var distanceMeters: CLLocationDistance = 0
    private var startedAt: Date?
    private var accumulatedPaused: TimeInterval = 0
    private var pauseBeganAt: Date?
    private var running = false
    private var paused = false
    /// 手动暂停会停定位；自动暂停保持定位以便检测恢复移动
    private var isManualPause = false

    private var stationarySince: Date?
    private let autoPauseSpeed: CLLocationSpeed = 0.45
    private let autoResumeSpeed: CLLocationSpeed = 0.9
    private let autoPauseHold: TimeInterval = 8

    private var timer: Timer?

    override init() {
        super.init()
        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyBest
        locationManager.distanceFilter = 5
        locationManager.locationTimeout = 10
        locationManager.reGeocodeTimeout = 5
        locationManager.allowsBackgroundLocationUpdates = true
        locationManager.pausesLocationUpdatesAutomatically = false
        locationManager.locatingWithReGeocode = false
    }

    func start() {
        guard !running else { return }
        points.removeAll()
        distanceMeters = 0
        accumulatedPaused = 0
        pauseBeganAt = nil
        startedAt = Date()
        running = true
        paused = false
        isManualPause = false
        stationarySince = nil
        locationManager.startUpdatingLocation()
        startTimer()
        publish()
    }

    func pause() {
        guard running, !paused else { return }
        paused = true
        isManualPause = true
        pauseBeganAt = Date()
        stationarySince = nil
        locationManager.stopUpdatingLocation()
        publish()
    }

    func resume() {
        guard running, paused else { return }
        finishPauseClock()
        let wasAuto = !isManualPause
        paused = false
        isManualPause = false
        stationarySince = nil
        locationManager.startUpdatingLocation()
        publish()
        if wasAuto {
            delegate?.runLocationTrackerDidAutoResume(self)
        }
    }

    func stop() -> RunSessionSnapshot {
        locationManager.stopUpdatingLocation()
        stopTimer()
        if paused {
            finishPauseClock()
            paused = false
            isManualPause = false
        }
        running = false
        stationarySince = nil
        let snapshot = currentSnapshot()
        publish()
        return snapshot
    }

    var snapshot: RunSessionSnapshot { currentSnapshot() }

    private func finishPauseClock() {
        if let pauseBeganAt {
            accumulatedPaused += Date().timeIntervalSince(pauseBeganAt)
        }
        pauseBeganAt = nil
    }

    private func enterAutoPause() {
        guard running, !paused else { return }
        paused = true
        isManualPause = false
        pauseBeganAt = Date()
        // 保持定位，以便速度恢复后自动继续
        publish()
        delegate?.runLocationTrackerDidAutoPause(self)
    }

    private func currentSnapshot() -> RunSessionSnapshot {
        RunSessionSnapshot(
            points: points,
            distanceMeters: distanceMeters,
            duration: elapsedDuration(),
            isPaused: paused,
            isRunning: running
        )
    }

    private func elapsedDuration() -> TimeInterval {
        guard let startedAt else { return 0 }
        var pausedExtra = accumulatedPaused
        if paused, let pauseBeganAt {
            pausedExtra += Date().timeIntervalSince(pauseBeganAt)
        }
        return max(0, Date().timeIntervalSince(startedAt) - pausedExtra)
    }

    private func append(location: CLLocation) {
        guard running else { return }
        guard location.horizontalAccuracy >= 0, location.horizontalAccuracy <= 50 else { return }

        let speed = max(0, location.speed)

        // 自动暂停：仍收点，但不累计距离；速度恢复则继续
        if paused {
            if !isManualPause, RunSettings.autoPauseEnabled, speed >= autoResumeSpeed {
                resume()
            } else {
                return
            }
        }

        if RunSettings.autoPauseEnabled {
            if speed >= 0, speed < autoPauseSpeed {
                if stationarySince == nil {
                    stationarySince = Date()
                } else if let since = stationarySince,
                          Date().timeIntervalSince(since) >= autoPauseHold {
                    enterAutoPause()
                    return
                }
            } else if speed >= autoResumeSpeed {
                stationarySince = nil
            }
        }

        let point = TrackPoint(
            coordinate: location.coordinate,
            timestamp: location.timestamp,
            altitude: location.altitude,
            speed: speed
        )

        if let last = points.last {
            let lastLocation = CLLocation(
                latitude: last.coordinate.latitude,
                longitude: last.coordinate.longitude
            )
            let delta = location.distance(from: lastLocation)
            if delta < 2 { return }
            if delta > 80 { return }
            distanceMeters += delta
        }

        points.append(point)
        publish()
    }

    private func publish() {
        delegate?.runLocationTracker(self, didUpdate: currentSnapshot())
    }

    private func startTimer() {
        stopTimer()
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            self?.publish()
        }
    }

    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
}

extension RunLocationTracker: AMapLocationManagerDelegate {
    func amapLocationManager(_ manager: AMapLocationManager!, didUpdate location: CLLocation!, reGeocode: AMapLocationReGeocode!) {
        guard let location else { return }
        append(location: location)
    }

    func amapLocationManager(_ manager: AMapLocationManager!, didFailWithError error: Error!) {
        guard let error else { return }
        delegate?.runLocationTracker(self, didFail: error)
    }

    func amapLocationManager(_ manager: AMapLocationManager!, doRequireLocationAuth locationManager: CLLocationManager!) {
        locationManager.requestAlwaysAuthorization()
    }
}
