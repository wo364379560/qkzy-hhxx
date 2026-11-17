//
//  RingProgressView.swift
//  TodayToTrackRun
//

import UIKit

final class RingProgressView: UIView {
    var progress: CGFloat = 0 {
        didSet { animatedSetProgress(progress) }
    }

    var lineWidth: CGFloat = 10
    var trackColor: UIColor = UIColor(white: 0.9, alpha: 1)
    var progressColor: UIColor = AppTheme.accent
    var centerTitle: String = "" {
        didSet { titleLabel.text = centerTitle }
    }
    var centerSubtitle: String = "" {
        didSet { subtitleLabel.text = centerSubtitle }
    }

    private let trackLayer = CAShapeLayer()
    private let progressLayer = CAShapeLayer()
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()

    override init(frame: CGRect) {
        super.init(frame: frame)
        layer.addSublayer(trackLayer)
        layer.addSublayer(progressLayer)
        trackLayer.fillColor = UIColor.clear.cgColor
        trackLayer.strokeColor = trackColor.cgColor
        trackLayer.lineWidth = lineWidth
        trackLayer.lineCap = .round

        progressLayer.fillColor = UIColor.clear.cgColor
        progressLayer.strokeColor = progressColor.cgColor
        progressLayer.lineWidth = lineWidth
        progressLayer.lineCap = .round
        progressLayer.strokeEnd = 0

        titleLabel.font = AppTheme.mono(22, weight: .bold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.textAlignment = .center
        subtitleLabel.font = .systemFont(ofSize: 12, weight: .medium)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.textAlignment = .center
        addSubview(titleLabel)
        addSubview(subtitleLabel)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func layoutSubviews() {
        super.layoutSubviews()
        let inset = lineWidth / 2 + 2
        let rect = bounds.insetBy(dx: inset, dy: inset)
        let path = UIBezierPath(
            arcCenter: CGPoint(x: bounds.midX, y: bounds.midY),
            radius: min(rect.width, rect.height) / 2,
            startAngle: -.pi / 2,
            endAngle: .pi * 1.5,
            clockwise: true
        )
        trackLayer.path = path.cgPath
        progressLayer.path = path.cgPath
        trackLayer.lineWidth = lineWidth
        progressLayer.lineWidth = lineWidth
        trackLayer.strokeColor = trackColor.cgColor
        progressLayer.strokeColor = progressColor.cgColor

        titleLabel.frame = CGRect(x: 12, y: bounds.midY - 22, width: bounds.width - 24, height: 28)
        subtitleLabel.frame = CGRect(x: 12, y: bounds.midY + 6, width: bounds.width - 24, height: 18)
    }

    private func animatedSetProgress(_ value: CGFloat) {
        let clamped = min(max(value, 0), 1)
        let animation = CABasicAnimation(keyPath: "strokeEnd")
        animation.fromValue = progressLayer.presentation()?.strokeEnd ?? progressLayer.strokeEnd
        animation.toValue = clamped
        animation.duration = 0.55
        animation.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
        progressLayer.strokeEnd = clamped
        progressLayer.add(animation, forKey: "progress")
    }
}
