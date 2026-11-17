//
//  HeartRateChartView.swift
//  TodayToTrackRun
//

import UIKit

final class HeartRateChartView: UIView {
    var values: [Int] = [] {
        didSet { setNeedsDisplay() }
    }

    var lineColor: UIColor = UIColor(red: 0.95, green: 0.35, blue: 0.35, alpha: 1)
    var fillColor: UIColor = UIColor(red: 0.95, green: 0.35, blue: 0.35, alpha: 0.18)

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = .clear
        isOpaque = false
        contentMode = .redraw
    }

    required init?(coder: NSCoder) { fatalError() }

    override func draw(_ rect: CGRect) {
        guard values.count >= 2, let context = UIGraphicsGetCurrentContext() else { return }
        let inset = UIEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)
        let drawRect = rect.inset(by: inset)
        let minV = Double(values.min() ?? 0)
        let maxV = Double(values.max() ?? 1)
        let span = max(maxV - minV, 8)

        let path = UIBezierPath()
        let fill = UIBezierPath()
        for (index, value) in values.enumerated() {
            let x = drawRect.minX + drawRect.width * CGFloat(index) / CGFloat(values.count - 1)
            let yNorm = (Double(value) - minV) / span
            let y = drawRect.maxY - CGFloat(yNorm) * drawRect.height
            let point = CGPoint(x: x, y: y)
            if index == 0 {
                path.move(to: point)
                fill.move(to: CGPoint(x: x, y: drawRect.maxY))
                fill.addLine(to: point)
            } else {
                path.addLine(to: point)
                fill.addLine(to: point)
            }
        }
        fill.addLine(to: CGPoint(x: drawRect.maxX, y: drawRect.maxY))
        fill.close()

        context.setFillColor(fillColor.cgColor)
        fill.fill()

        context.setStrokeColor(lineColor.cgColor)
        context.setLineWidth(2)
        context.setLineJoin(.round)
        context.setLineCap(.round)
        path.stroke()
    }
}
