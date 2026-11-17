//
//  SimulatedRunController.swift
//  TodayToTrackRun
//
//  室内模拟跑步：按时间吐出假点，驱动地图 UI。
//

import CoreLocation
import Foundation

protocol SimulatedRunControllerDelegate: AnyObject {
    func simulatedRun(_ controller: SimulatedRunController, didUpdate snapshot: RunSessionSnapshot)
    func simulatedRunDidFinish(_ controller: SimulatedRunController, snapshot: RunSessionSnapshot)
}

final class SimulatedRunController {
    weak var delegate: SimulatedRunControllerDelegate?

    private var points: [TrackPoint] = []
    private var cursor = 0
    private var timer: Timer?
    private var startedAt: Date?
    private let allPoints: [TrackPoint]
    private let totalDistance: CLLocationDistance
    private let totalDuration: TimeInterval

    /// speedFactor: 1=真实配速时长，10=十倍速便于室内调试
    init(laps: Double = 2, paceSecondsPerKm: Double = 330, speedFactor: Double = 12) {
        let raw = MockRunGenerator.ovalPoints(laps: laps, paceSecondsPerKm: paceSecondsPerKm)
        // 压缩时间戳间隔，便于快速演示
        let now = Date()
        let compressedDuration = max(8, raw.duration / max(speedFactor, 1))
        var rebuilt: [TrackPoint] = []
        for (i, p) in raw.points.enumerated() {
            let t = now.addingTimeInterval(compressedDuration * Double(i) / Double(max(raw.points.count - 1, 1)))
            rebuilt.append(
                TrackPoint(
                    coordinate: p.coordinate,
                    timestamp: t,
                    altitude: p.altitude,
                    speed: p.speed
                )
            )
        }
        allPoints = rebuilt
        totalDistance = raw.distance
        totalDuration = compressedDuration
    }

    var isRunning: Bool { timer != nil }

    func start() {
        stopTimer()
        points = []
        cursor = 0
        startedAt = Date()
        // 约 8～20 秒播完
        let interval = max(0.12, totalDuration / Double(max(allPoints.count, 1)))
        timer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
            self?.tick()
        }
        tick()
    }

    func stop() -> RunSessionSnapshot {
        stopTimer()
        return snapshot(isRunning: false)
    }

    private func tick() {
        guard cursor < allPoints.count else {
            let finalSnap = snapshot(isRunning: false)
            stopTimer()
            delegate?.simulatedRunDidFinish(self, snapshot: finalSnap)
            return
        }
        points.append(allPoints[cursor])
        cursor += 1
        let snap = snapshot(isRunning: true)
        delegate?.simulatedRun(self, didUpdate: snap)
    }

    private func snapshot(isRunning: Bool) -> RunSessionSnapshot {
        let progress = allPoints.isEmpty ? 1 : Double(points.count) / Double(allPoints.count)
        let distance = totalDistance * progress
        let duration: TimeInterval
        if let first = points.first, let last = points.last {
            duration = max(1, last.timestamp.timeIntervalSince(first.timestamp))
        } else {
            duration = 0
        }
        return RunSessionSnapshot(
            points: points,
            distanceMeters: distance,
            duration: duration,
            isPaused: false,
            isRunning: isRunning
        )
    }

    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
}
