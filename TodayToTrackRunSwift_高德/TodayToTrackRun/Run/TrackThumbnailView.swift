//
//  TrackThumbnailView.swift
//  TodayToTrackRun
//

import CoreLocation
import UIKit

/// 列表用轨迹缩略图：不嵌地图 SDK，纯 Path 绘制（COROS 列表同款思路）。
final class TrackThumbnailView: UIView {
    var coordinates: [CLLocationCoordinate2D] = [] {
        didSet { setNeedsDisplay() }
    }

    var strokeColor: UIColor = UIColor(red: 1, green: 0.84, blue: 0.2, alpha: 1)
    var lineWidth: CGFloat = 3

    override init(frame: CGRect) {
        super.init(frame: frame)
        commonInit()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        commonInit()
    }

    private func commonInit() {
        backgroundColor = UIColor(white: 0.08, alpha: 1)
        isOpaque = true
        layer.cornerRadius = 8
        clipsToBounds = true
    }

    override func draw(_ rect: CGRect) {
        guard coordinates.count >= 2,
              let context = UIGraphicsGetCurrentContext() else {
            return
        }

        let lats = coordinates.map(\.latitude)
        let lons = coordinates.map(\.longitude)
        guard let minLat = lats.min(), let maxLat = lats.max(),
              let minLon = lons.min(), let maxLon = lons.max() else {
            return
        }

        let latSpan = max(maxLat - minLat, 0.00001)
        let lonSpan = max(maxLon - minLon, 0.00001)
        let inset: CGFloat = 10
        let drawRect = rect.insetBy(dx: inset, dy: inset)

        let path = UIBezierPath()
        for (index, coordinate) in coordinates.enumerated() {
            let x = drawRect.minX + CGFloat((coordinate.longitude - minLon) / lonSpan) * drawRect.width
            let y = drawRect.maxY - CGFloat((coordinate.latitude - minLat) / latSpan) * drawRect.height
            let point = CGPoint(x: x, y: y)
            if index == 0 {
                path.move(to: point)
            } else {
                path.addLine(to: point)
            }
        }

        context.setStrokeColor(strokeColor.cgColor)
        context.setLineWidth(lineWidth)
        context.setLineJoin(.round)
        context.setLineCap(.round)
        context.addPath(path.cgPath)
        context.strokePath()
    }
}
