//
//  RunDetailViewController.swift
//  TodayToTrackRun
//

import MAMapKit
import SnapKit
import UIKit

final class RunDetailViewController: UIViewController {
    private let detail: RunRecordDetail
    /// 必须在高德 SDK configure 之后创建
    private var mapView: MAMapView!
    private var polyline: MAPolyline?
    private var startAnnotation: MAPointAnnotation?
    private var endAnnotation: MAPointAnnotation?
    private var didFitTrack = false

    private let bottomPanel = UIView()
    private let statsLabel = UILabel()
    private let hrTitle = UILabel()
    private let hrChart = HeartRateChartView()

    init(detail: RunRecordDetail) {
        self.detail = detail
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        AMapBootstrap.configureIfNeeded()
        title = "轨迹详情"
        view.backgroundColor = .black

        let created: MAMapView! = MAMapView(frame: .zero)
        guard let map = created else {
            assertionFailure("MAMapView 创建失败：请确认高德 Key / 隐私协议已配置")
            return
        }
        mapView = map
        map.showsUserLocation = false
        map.showsCompass = false
        map.showsScale = false
        map.mapType = .satellite
        map.delegate = self
        view.addSubview(map)

        bottomPanel.backgroundColor = UIColor.black.withAlphaComponent(0.82)
        bottomPanel.layer.cornerRadius = 16
        bottomPanel.layer.maskedCorners = [.layerMinXMinYCorner, .layerMaxXMinYCorner]
        view.addSubview(bottomPanel)

        statsLabel.numberOfLines = 0
        statsLabel.textColor = .white
        statsLabel.font = .systemFont(ofSize: 15, weight: .medium)
        statsLabel.textAlignment = .center

        hrTitle.text = "心率曲线（估算）"
        hrTitle.textColor = UIColor.white.withAlphaComponent(0.7)
        hrTitle.font = .systemFont(ofSize: 12, weight: .medium)

        bottomPanel.addSubview(statsLabel)
        bottomPanel.addSubview(hrTitle)
        bottomPanel.addSubview(hrChart)

        mapView.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
            make.bottom.equalTo(bottomPanel.snp.top).offset(20)
        }
        bottomPanel.snp.makeConstraints { make in
            make.leading.trailing.bottom.equalToSuperview()
        }
        statsLabel.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview().inset(16)
        }
        hrTitle.snp.makeConstraints { make in
            make.top.equalTo(statsLabel.snp.bottom).offset(10)
            make.leading.trailing.equalToSuperview().inset(16)
        }
        hrChart.snp.makeConstraints { make in
            make.top.equalTo(hrTitle.snp.bottom).offset(4)
            make.leading.trailing.equalToSuperview().inset(8)
            make.height.equalTo(88)
            make.bottom.equalTo(view.safeAreaLayoutGuide.snp.bottom).offset(-12)
        }

        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "分享",
            style: .plain,
            target: self,
            action: #selector(shareTapped)
        )

        fillStats()
        drawTrack()
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        fitTrackToVisibleArea(animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) { [weak self] in
            self?.fitTrackToVisibleArea(animated: true, force: true)
        }
    }

    private func fillStats() {
        let s = detail.summary
        let place = s.cityName ?? "本地跑步"
        let kcal = s.caloriesKcal > 0
            ? s.caloriesKcal
            : RunMetricsEstimator.estimatedCalories(
                distanceMeters: s.distanceMeters,
                durationSeconds: s.durationSeconds
            )
        let hr = s.averageHeartRate > 0
            ? s.averageHeartRate
            : RunMetricsEstimator.estimatedAverageHeartRate(
                paceSecondsPerKm: s.averagePaceSecondsPerKm,
                durationSeconds: s.durationSeconds
            )
        statsLabel.text = """
        \(s.formattedDistance)  ·  \(s.formattedDuration)  ·  \(s.formattedPace)
        \(place)  ·  \(kcal) kcal  ·  均心率 \(hr > 0 ? "\(hr)" : "--")
        """
        hrChart.values = RunMetricsEstimator.heartRateSeries(
            averageBPM: hr,
            durationSeconds: s.durationSeconds
        )
    }

    private func drawTrack() {
        let coords = detail.points.map(\.coordinate)
        guard coords.count >= 2 else { return }
        var mutable = coords
        let line = MAPolyline(coordinates: &mutable, count: UInt(mutable.count))
        polyline = line
        mapView.add(line)

        let start = MAPointAnnotation()
        start.coordinate = coords.first!
        start.title = "起点"
        startAnnotation = start

        let end = MAPointAnnotation()
        end.coordinate = coords.last!
        end.title = "终点"
        endAnnotation = end

        mapView.addAnnotations([start, end])
    }

    private func fitTrackToVisibleArea(animated: Bool, force: Bool = false) {
        guard let polyline else { return }
        guard mapView.bounds.width > 1, mapView.bounds.height > 1 else { return }
        if didFitTrack, !force { return }
        mapView.layoutIfNeeded()
        mapView.showOverlays(
            [polyline],
            edgePadding: UIEdgeInsets(top: 100, left: 48, bottom: 220, right: 48),
            animated: animated
        )
        didFitTrack = true
    }

    @objc private func shareTapped() {
        let s = detail.summary
        let text = "跑步 \(s.formattedDistance)，用时 \(s.formattedDuration)，配速 \(s.formattedPace)"
        mapView.takeSnapshot(in: mapView.bounds) { [weak self] image, _ in
            guard let self else { return }
            var items: [Any] = [text]
            if let image { items.append(image) }
            let vc = UIActivityViewController(activityItems: items, applicationActivities: nil)
            self.present(vc, animated: true)
        }
    }
}

extension RunDetailViewController: MAMapViewDelegate {
    func mapView(_ mapView: MAMapView!, rendererFor overlay: MAOverlay!) -> MAOverlayRenderer! {
        guard let polyline = overlay as? MAPolyline else { return nil }
        let renderer = MAPolylineRenderer(polyline: polyline)
        renderer?.lineWidth = 6
        renderer?.strokeColor = UIColor(red: 1, green: 0.84, blue: 0.2, alpha: 1)
        renderer?.lineJoinType = kMALineJoinRound
        renderer?.lineCapType = kMALineCapRound
        return renderer
    }

    func mapView(_ mapView: MAMapView!, viewFor annotation: MAAnnotation!) -> MAAnnotationView! {
        guard let point = annotation as? MAPointAnnotation else { return nil }
        let id = "runEndpoint"
        let view = mapView.dequeueReusableAnnotationView(withIdentifier: id)
            ?? MAAnnotationView(annotation: point, reuseIdentifier: id)
        view?.annotation = point
        view?.canShowCallout = true

        let isStart = point === startAnnotation || point.title == "起点"
        let size: CGFloat = 16
        let dot = UIView(frame: CGRect(x: 0, y: 0, width: size, height: size))
        dot.layer.cornerRadius = size / 2
        dot.backgroundColor = isStart
            ? UIColor(red: 0.25, green: 0.85, blue: 0.45, alpha: 1)
            : UIColor(red: 0.95, green: 0.35, blue: 0.35, alpha: 1)
        dot.layer.borderWidth = 2
        dot.layer.borderColor = UIColor.white.cgColor
        UIGraphicsBeginImageContextWithOptions(dot.bounds.size, false, 0)
        if let ctx = UIGraphicsGetCurrentContext() {
            dot.layer.render(in: ctx)
            view?.image = UIGraphicsGetImageFromCurrentImageContext()
        }
        UIGraphicsEndImageContext()
        view?.centerOffset = CGPoint(x: 0, y: -size / 2)
        return view
    }

    func mapViewDidFinishLoadingMap(_ mapView: MAMapView!) {
        fitTrackToVisibleArea(animated: false)
    }
}
