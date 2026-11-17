//
//  RunMapViewController.swift
//  TodayToTrackRun
//

import AMapFoundationKit
import CoreLocation
import MAMapKit
import SnapKit
import UIKit

/// 跑步页：高德卫星底图 + 黄线轨迹 + 开始/暂停/结束 + 系统分享。
final class RunMapViewController: UIViewController {
    /// 必须在 `AMapBootstrap.configureIfNeeded()` 之后创建，否则 MAMapView() 会返回 nil 并导致 SnapKit 崩
    private var mapView: MAMapView!
    private let tracker = RunLocationTracker()
    private var polyline: MAPolyline?
    private var isFollowingUser = true
    private let permissionManager = CLLocationManager()

    private let statsCard = UIView()
    private let distanceValueLabel = UILabel()
    private let durationValueLabel = UILabel()
    private let paceValueLabel = UILabel()
    private let thumbnailView = TrackThumbnailView()

    private let startButton = UIButton(type: .system)
    private let pauseButton = UIButton(type: .system)
    private let stopButton = UIButton(type: .system)
    private let shareButton = UIButton(type: .system)
    private let satelliteButton = UIButton(type: .system)
    private let simulateButton = UIButton(type: .system)

    private var simulator: SimulatedRunController?
    private var isSimulating = false

    private var lastSnapshot = RunSessionSnapshot(
        points: [],
        distanceMeters: 0,
        duration: 0,
        isPaused: false,
        isRunning: false
    )

    override func viewDidLoad() {
        super.viewDidLoad()
        AMapBootstrap.configureIfNeeded()
        title = "运动"
        view.backgroundColor = .black
        tracker.delegate = self
        setupMap()
        setupStats()
        setupControls()
        updateControls(for: lastSnapshot)
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleExternalSimulate),
            name: .todayTrackStartSimulatedRun,
            object: nil
        )
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    private func setupMap() {
        // Key / 隐私协议必须先就绪，再 new MAMapView（属性阶段创建会得到 nil）
        AMapBootstrap.configureIfNeeded()
        let created: MAMapView! = MAMapView(frame: .zero)
        guard let map = created else {
            assertionFailure("MAMapView 创建失败：请确认高德 Key / 隐私协议已配置")
            return
        }
        mapView = map
        map.delegate = self
        map.showsUserLocation = true
        map.userTrackingMode = .follow
        map.showsCompass = false
        map.showsScale = false
        map.mapType = .satellite
        map.zoomLevel = 17
        view.addSubview(map)
        map.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
    }

    private func setupStats() {
        statsCard.backgroundColor = UIColor.black.withAlphaComponent(0.72)
        statsCard.layer.cornerRadius = 14
        view.addSubview(statsCard)

        // 三列等分：数值 + 标题分行，避免大字号把「km/配速」挤成省略号
        let distanceCol = makeStatColumn(
            valueLabel: distanceValueLabel,
            caption: "距离",
            valueFontSize: 26
        )
        let durationCol = makeStatColumn(
            valueLabel: durationValueLabel,
            caption: "时间",
            valueFontSize: 22
        )
        let paceCol = makeStatColumn(
            valueLabel: paceValueLabel,
            caption: "配速",
            valueFontSize: 18
        )

        let stack = UIStackView(arrangedSubviews: [distanceCol, durationCol, paceCol])
        stack.axis = .horizontal
        stack.distribution = .fillEqually
        stack.alignment = .fill
        stack.spacing = 4

        statsCard.addSubview(stack)
        statsCard.addSubview(thumbnailView)
        thumbnailView.layer.cornerRadius = 8
        thumbnailView.clipsToBounds = true
        thumbnailView.backgroundColor = UIColor.white.withAlphaComponent(0.08)

        statsCard.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide.snp.top).offset(12)
            make.leading.trailing.equalToSuperview().inset(12)
        }
        // 指标条铺满宽度三等分；缩略图改到下方，不再挤占列宽
        stack.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview().inset(12)
        }
        thumbnailView.snp.makeConstraints { make in
            make.top.equalTo(stack.snp.bottom).offset(10)
            make.trailing.equalToSuperview().inset(12)
            make.bottom.equalToSuperview().inset(12)
            make.width.height.equalTo(48)
        }

        distanceValueLabel.text = "0.00 km"
        durationValueLabel.text = "00:00"
        paceValueLabel.text = "--'--\"/km"
    }

    private func makeStatColumn(
        valueLabel: UILabel,
        caption: String,
        valueFontSize: CGFloat
    ) -> UIView {
        valueLabel.textColor = .white
        valueLabel.textAlignment = .center
        valueLabel.numberOfLines = 1
        valueLabel.font = .monospacedDigitSystemFont(ofSize: valueFontSize, weight: .bold)
        valueLabel.adjustsFontSizeToFitWidth = true
        valueLabel.minimumScaleFactor = 0.55
        valueLabel.lineBreakMode = .byClipping

        let captionLabel = UILabel()
        captionLabel.text = caption
        captionLabel.textColor = UIColor.white.withAlphaComponent(0.78)
        captionLabel.font = .systemFont(ofSize: 12, weight: .medium)
        captionLabel.textAlignment = .center

        let col = UIStackView(arrangedSubviews: [valueLabel, captionLabel])
        col.axis = .vertical
        col.alignment = .fill
        col.spacing = 4
        return col
    }

    private func setupControls() {
        configure(button: startButton, title: "开始", color: .systemGreen)
        configure(button: pauseButton, title: "暂停", color: .systemOrange)
        configure(button: stopButton, title: "结束", color: .systemRed)
        configure(button: shareButton, title: "分享", color: .systemBlue)
        configure(button: satelliteButton, title: "卫星", color: .darkGray)
        configure(button: simulateButton, title: "模拟", color: UIColor(red: 0.45, green: 0.35, blue: 0.85, alpha: 1))

        startButton.addTarget(self, action: #selector(startTapped), for: .touchUpInside)
        pauseButton.addTarget(self, action: #selector(pauseTapped), for: .touchUpInside)
        stopButton.addTarget(self, action: #selector(stopTapped), for: .touchUpInside)
        shareButton.addTarget(self, action: #selector(shareTapped), for: .touchUpInside)
        satelliteButton.addTarget(self, action: #selector(toggleMapType), for: .touchUpInside)
        simulateButton.addTarget(self, action: #selector(simulateTapped), for: .touchUpInside)

        let row = UIStackView(arrangedSubviews: [
            startButton, pauseButton, stopButton, shareButton, satelliteButton, simulateButton
        ])
        row.axis = .horizontal
        row.spacing = 8
        row.distribution = .fillEqually
        view.addSubview(row)
        row.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview().inset(10)
            make.bottom.equalTo(view.safeAreaLayoutGuide.snp.bottom).offset(-16)
            make.height.equalTo(48)
        }
    }

    private func configure(button: UIButton, title: String, color: UIColor) {
        button.setTitle(title, for: .normal)
        button.setTitleColor(.white, for: .normal)
        button.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
        button.backgroundColor = color
        button.layer.cornerRadius = 10
    }

    private func updateControls(for snapshot: RunSessionSnapshot) {
        if isSimulating {
            startButton.isEnabled = false
            startButton.alpha = 0.45
            pauseButton.isEnabled = false
            pauseButton.alpha = 0.45
            stopButton.isEnabled = true
            stopButton.alpha = 1
            simulateButton.isEnabled = false
            simulateButton.alpha = 0.45
            return
        }
        simulateButton.isEnabled = !snapshot.isRunning
        simulateButton.alpha = simulateButton.isEnabled ? 1 : 0.45
        if !snapshot.isRunning {
            startButton.isEnabled = true
            startButton.alpha = 1
            pauseButton.isEnabled = false
            pauseButton.alpha = 0.45
            pauseButton.setTitle("暂停", for: .normal)
            stopButton.isEnabled = snapshot.points.count >= 2
            stopButton.alpha = stopButton.isEnabled ? 1 : 0.45
        } else if snapshot.isPaused {
            startButton.isEnabled = false
            startButton.alpha = 0.45
            pauseButton.isEnabled = true
            pauseButton.alpha = 1
            pauseButton.setTitle("继续", for: .normal)
            stopButton.isEnabled = true
            stopButton.alpha = 1
        } else {
            startButton.isEnabled = false
            startButton.alpha = 0.45
            pauseButton.isEnabled = true
            pauseButton.alpha = 1
            pauseButton.setTitle("暂停", for: .normal)
            stopButton.isEnabled = true
            stopButton.alpha = 1
        }
    }

    private func render(snapshot: RunSessionSnapshot) {
        lastSnapshot = snapshot
        distanceValueLabel.text = snapshot.formattedDistance
        durationValueLabel.text = snapshot.formattedDuration
        let pace = snapshot.formattedPace
        paceValueLabel.text = pace.contains("/km") ? pace : pace + "/km"
        thumbnailView.coordinates = snapshot.points.map(\.coordinate)
        updateControls(for: snapshot)
        redrawPolyline(with: snapshot.points.map(\.coordinate))

        if snapshot.isRunning, !snapshot.isPaused {
            RunVoicePrompt.shared.update(distanceMeters: snapshot.distanceMeters)
            RunVoicePrompt.shared.coachIfNeeded(
                paceSecondsPerKm: snapshot.averagePaceSecondsPerKm,
                duration: snapshot.duration
            )
        }
    }

    private func redrawPolyline(with coordinates: [CLLocationCoordinate2D]) {
        if let polyline {
            mapView.remove(polyline)
            self.polyline = nil
        }
        guard coordinates.count >= 2 else { return }

        var coords = coordinates
        let line = MAPolyline(coordinates: &coords, count: UInt(coords.count))
        polyline = line
        mapView.add(line)
    }

    @objc private func startTapped() {
        guard !isSimulating else { return }
        requestLocationPermissionIfNeeded()
        isFollowingUser = true
        mapView.userTrackingMode = .follow
        RunVoicePrompt.shared.resetSession()
        tracker.start()
    }

    @objc private func pauseTapped() {
        guard !isSimulating else { return }
        if lastSnapshot.isPaused {
            tracker.resume()
        } else {
            tracker.pause()
        }
    }

    @objc private func stopTapped() {
        RunVoicePrompt.shared.stop()
        let snapshot: RunSessionSnapshot
        if isSimulating {
            snapshot = simulator?.stop() ?? lastSnapshot
            finishSimulation(snapshot: snapshot, autoSave: true)
            return
        }
        snapshot = tracker.stop()
        isFollowingUser = false
        mapView.userTrackingMode = .none
        render(snapshot: snapshot)
        if let polyline {
            mapView.showOverlays(
                [polyline],
                edgePadding: UIEdgeInsets(top: 140, left: 40, bottom: 120, right: 40),
                animated: true
            )
        }
        persist(snapshot: snapshot)
    }

    @objc private func simulateTapped() {
        startSimulation()
    }

    @objc private func handleExternalSimulate() {
        startSimulation()
    }

    private func startSimulation() {
        guard !isSimulating, !lastSnapshot.isRunning else { return }
        let center = CLLocationCoordinate2D(latitude: 30.2532, longitude: 120.2105)
        mapView.setCenter(center, animated: true)
        mapView.setZoomLevel(16.5, animated: true)
        mapView.userTrackingMode = .none
        isFollowingUser = false
        isSimulating = true
        RunVoicePrompt.shared.resetSession()
        // 约 1.2km，便于室内听到 1 公里语音
        let sim = SimulatedRunController(laps: 3, paceSecondsPerKm: 330, speedFactor: 15)
        sim.delegate = self
        simulator = sim
        sim.start()
        updateControls(for: lastSnapshot)
    }

    private func finishSimulation(snapshot: RunSessionSnapshot, autoSave: Bool) {
        isSimulating = false
        simulator = nil
        RunVoicePrompt.shared.stop()
        render(snapshot: snapshot)
        if let polyline {
            mapView.showOverlays(
                [polyline],
                edgePadding: UIEdgeInsets(top: 140, left: 40, bottom: 120, right: 40),
                animated: true
            )
        }
        if autoSave {
            persist(snapshot: snapshot)
        }
        updateControls(for: snapshot)
    }

    @objc private func openHistory() {
        linLanPush(RunHistoryViewController())
    }

    private func persist(snapshot: RunSessionSnapshot) {
        do {
            let isDemoTrack: Bool = {
                guard let c = snapshot.points.first?.coordinate else { return false }
                return abs(c.latitude - 30.2532) < 0.02 && abs(c.longitude - 120.2105) < 0.02
            }()
            let cityHint = isDemoTrack ? "杭州市" : nil
            let metrics = RunMetricsEstimator.metrics(for: snapshot)
            let runID = try RunRecordStore.save(snapshot: snapshot, cityName: cityHint)
            GoalPreferences.markRunCompletedToday()

            let completion = RunCompletionViewController(
                snapshot: snapshot,
                cityName: cityHint,
                calories: metrics.calories,
                avgHR: metrics.avgHR
            )
            completion.onViewHistory = { [weak self] in
                self?.openHistory()
            }
            present(completion, animated: true)

            if cityHint == nil {
                enrichCityName(runID: runID, snapshot: snapshot)
            }
        } catch RunRecordStore.StoreError.tooShort {
            let alert = UIAlertController(
                title: "未保存",
                message: "轨迹太短，请再跑一段后再结束",
                preferredStyle: .alert
            )
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
        } catch {
            let alert = UIAlertController(
                title: "保存失败",
                message: error.localizedDescription,
                preferredStyle: .alert
            )
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
        }
    }

    private func enrichCityName(runID: Int64, snapshot: RunSessionSnapshot) {
        guard let coordinate = snapshot.points.last?.coordinate else { return }
        Task { @MainActor in
            do {
                let city = try await ReverseGeocoder.shared.cityName(for: coordinate)
                try RunRecordStore.updateCityName(runID: runID, cityName: city)
            } catch {
                // 逆地理失败不影响本地保存
                print("[Geo] city resolve failed:", error.localizedDescription)
            }
        }
    }

    @objc private func shareTapped() {
        let text = """
        今日跑步 \(lastSnapshot.formattedDistance)
        用时 \(lastSnapshot.formattedDuration)
        配速 \(lastSnapshot.formattedPace)
        """
        mapView.takeSnapshot(in: mapView.bounds) { [weak self] image, _ in
            guard let self else { return }
            var items: [Any] = [text]
            if let image {
                items.append(image)
            }
            let vc = UIActivityViewController(activityItems: items, applicationActivities: nil)
            if let pop = vc.popoverPresentationController {
                pop.sourceView = self.shareButton
                pop.sourceRect = self.shareButton.bounds
            }
            self.present(vc, animated: true)
        }
    }

    @objc private func toggleMapType() {
        if mapView.mapType == .satellite {
            mapView.mapType = .standard
            satelliteButton.setTitle("标准", for: .normal)
        } else {
            mapView.mapType = .satellite
            satelliteButton.setTitle("卫星", for: .normal)
        }
    }

    private func requestLocationPermissionIfNeeded() {
        switch permissionManager.authorizationStatus {
        case .notDetermined:
            permissionManager.requestWhenInUseAuthorization()
        default:
            break
        }
    }
}

extension RunMapViewController: RunLocationTrackerDelegate {
    func runLocationTracker(_ tracker: RunLocationTracker, didUpdate snapshot: RunSessionSnapshot) {
        guard !isSimulating else { return }
        render(snapshot: snapshot)
    }

    func runLocationTracker(_ tracker: RunLocationTracker, didFail error: Error) {
        guard !isSimulating else { return }
        let alert = UIAlertController(
            title: "定位失败",
            message: error.localizedDescription,
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "好", style: .default))
        present(alert, animated: true)
    }

    func runLocationTrackerDidAutoPause(_ tracker: RunLocationTracker) {
        // 轻提示：状态栏文案已通过 pause 按钮「继续」体现
        pauseButton.setTitle("继续", for: .normal)
    }

    func runLocationTrackerDidAutoResume(_ tracker: RunLocationTracker) {
        pauseButton.setTitle("暂停", for: .normal)
    }
}

extension RunMapViewController: SimulatedRunControllerDelegate {
    func simulatedRun(_ controller: SimulatedRunController, didUpdate snapshot: RunSessionSnapshot) {
        render(snapshot: snapshot)
        if let last = snapshot.points.last {
            mapView.setCenter(last.coordinate, animated: false)
        }
    }

    func simulatedRunDidFinish(_ controller: SimulatedRunController, snapshot: RunSessionSnapshot) {
        finishSimulation(snapshot: snapshot, autoSave: true)
    }
}

extension RunMapViewController: MAMapViewDelegate {
    func mapView(_ mapView: MAMapView!, rendererFor overlay: MAOverlay!) -> MAOverlayRenderer! {
        guard let polyline = overlay as? MAPolyline else { return nil }
        let renderer = MAPolylineRenderer(polyline: polyline)
        renderer?.lineWidth = 6
        renderer?.strokeColor = UIColor(red: 1, green: 0.84, blue: 0.2, alpha: 1)
        renderer?.lineJoinType = kMALineJoinRound
        renderer?.lineCapType = kMALineCapRound
        return renderer
    }

    func mapView(_ mapView: MAMapView!, didSingleTappedAt coordinate: CLLocationCoordinate2D) {
        isFollowingUser = false
        mapView.userTrackingMode = .none
    }
}
