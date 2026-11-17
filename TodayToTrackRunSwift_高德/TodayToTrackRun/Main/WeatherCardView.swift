//
//  WeatherCardView.swift
//  TodayToTrackRun
//

import UIKit

final class WeatherCardView: UIControl {
    private let cityLabel = UILabel()
    private let tempLabel = UILabel()
    private let conditionLabel = UILabel()
    private let rangeLabel = UILabel()
    private let emojiLabel = UILabel()
    private let adviceLabel = UILabel()
    private let gradient = CAGradientLayer()

    override init(frame: CGRect) {
        super.init(frame: frame)
        layer.cornerRadius = AppTheme.cardRadius
        clipsToBounds = true
        gradient.colors = [AppTheme.weatherBlue.cgColor, AppTheme.weatherBlueEnd.cgColor]
        gradient.startPoint = CGPoint(x: 0, y: 0)
        gradient.endPoint = CGPoint(x: 1, y: 1)
        layer.insertSublayer(gradient, at: 0)

        cityLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        cityLabel.textColor = .white
        tempLabel.font = AppTheme.mono(40, weight: .bold)
        tempLabel.textColor = .white
        conditionLabel.font = .systemFont(ofSize: 14, weight: .medium)
        conditionLabel.textColor = UIColor.white.withAlphaComponent(0.92)
        rangeLabel.font = .systemFont(ofSize: 13)
        rangeLabel.textColor = UIColor.white.withAlphaComponent(0.85)
        rangeLabel.textAlignment = .right
        emojiLabel.font = .systemFont(ofSize: 36)
        adviceLabel.font = .systemFont(ofSize: 12, weight: .medium)
        adviceLabel.textColor = UIColor.white.withAlphaComponent(0.9)

        [cityLabel, tempLabel, conditionLabel, rangeLabel, emojiLabel, adviceLabel].forEach(addSubview)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func layoutSubviews() {
        super.layoutSubviews()
        gradient.frame = bounds
        cityLabel.frame = CGRect(x: 16, y: 14, width: bounds.width - 80, height: 20)
        tempLabel.frame = CGRect(x: 16, y: 34, width: 140, height: 48)
        conditionLabel.frame = CGRect(x: 16, y: bounds.height - 42, width: bounds.width * 0.55, height: 18)
        adviceLabel.frame = CGRect(x: 16, y: bounds.height - 24, width: bounds.width - 32, height: 16)
        emojiLabel.frame = CGRect(x: bounds.width - 64, y: 12, width: 48, height: 48)
        rangeLabel.frame = CGRect(x: bounds.width - 140, y: bounds.height - 42, width: 124, height: 18)
    }

    func configure(_ weather: WeatherInfo, loading: Bool) {
        cityLabel.text = weather.city
        tempLabel.text = loading ? "--°" : "\(weather.temp)°"
        conditionLabel.text = weather.condition
        rangeLabel.text = weather.tempRange
        emojiLabel.text = weather.emoji
        adviceLabel.text = weather.runAdvice
    }
}
