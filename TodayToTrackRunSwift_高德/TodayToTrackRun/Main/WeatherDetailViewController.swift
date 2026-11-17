//
//  WeatherDetailViewController.swift
//  TodayToTrackRun
//
//  对齐小确天气详情：实况 + 生活指数 + 未来几天预报
//

import SnapKit
import UIKit

final class WeatherDetailViewController: UIViewController {
    private var weather: WeatherInfo
    private let scroll = UIScrollView()
    private let stack = UIStackView()

    init(weather: WeatherInfo) {
        self.weather = weather
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "天气详情"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "刷新",
            style: .plain,
            target: self,
            action: #selector(refreshTapped)
        )
        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 24, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }
        rebuild()
    }

    private func rebuild() {
        stack.arrangedSubviews.forEach { $0.removeFromSuperview() }

        let header = AppTheme.makeCard()
        header.backgroundColor = AppTheme.weatherBlue
        let city = UILabel()
        city.text = weather.city
        city.font = .systemFont(ofSize: 16, weight: .semibold)
        city.textColor = .white
        let emoji = UILabel()
        emoji.text = weather.emoji
        emoji.font = .systemFont(ofSize: 44)
        let temp = UILabel()
        temp.text = "\(weather.temp)°"
        temp.font = AppTheme.mono(44, weight: .bold)
        temp.textColor = .white
        let cond = UILabel()
        cond.text = "\(weather.condition)  ·  \(weather.tempRange)"
        cond.font = .systemFont(ofSize: 14)
        cond.textColor = UIColor.white.withAlphaComponent(0.92)
        cond.numberOfLines = 0
        header.addSubview(city)
        header.addSubview(emoji)
        header.addSubview(temp)
        header.addSubview(cond)
        city.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().inset(16)
        }
        emoji.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(16)
            make.trailing.equalToSuperview().offset(-16)
        }
        temp.snp.makeConstraints { make in
            make.top.equalTo(city.snp.bottom).offset(6)
            make.leading.equalToSuperview().inset(16)
        }
        cond.snp.makeConstraints { make in
            make.top.equalTo(temp.snp.bottom).offset(6)
            make.leading.trailing.bottom.equalToSuperview().inset(16)
        }
        stack.addArrangedSubview(header)

        stack.addArrangedSubview(sectionTitle("实况"))
        let detailCard = AppTheme.makeCard()
        let detailStack = UIStackView()
        detailStack.axis = .vertical
        detailStack.spacing = 10
        detailCard.addSubview(detailStack)
        detailStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(14) }
        [
            ("风向风力", weather.windDirection),
            ("湿度", weather.humidity),
            ("风力", weather.windLevel),
            ("温度范围", weather.tempRange)
        ].forEach { title, value in
            detailStack.addArrangedSubview(kvRow(title, value))
        }
        stack.addArrangedSubview(detailCard)

        stack.addArrangedSubview(sectionTitle("生活指数"))
        let indexCard = AppTheme.makeCard()
        let indexStack = UIStackView()
        indexStack.axis = .vertical
        indexStack.spacing = 10
        indexCard.addSubview(indexStack)
        indexStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(14) }
        for item in weather.indices {
            indexStack.addArrangedSubview(kvRow("\(item.emoji) \(item.title)", item.value))
        }
        stack.addArrangedSubview(indexCard)

        stack.addArrangedSubview(sectionTitle("未来几天"))
        let futureCard = AppTheme.makeCard()
        let futureStack = UIStackView()
        futureStack.axis = .vertical
        futureStack.spacing = 0
        futureCard.addSubview(futureStack)
        futureStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(8) }

        let days = weather.future.isEmpty
            ? [WeatherFutureDay(week: "暂无", emoji: "🌤️", weather: "下拉或点刷新重试", tempRange: "--")]
            : weather.future
        for (index, day) in days.enumerated() {
            let row = UIView()
            row.snp.makeConstraints { $0.height.equalTo(48) }
            let week = UILabel()
            week.text = day.week
            week.font = .systemFont(ofSize: 14, weight: index == 0 ? .semibold : .regular)
            week.textColor = AppTheme.textPrimary
            week.snp.makeConstraints { $0.width.equalTo(48) }
            let icon = UILabel()
            icon.text = day.emoji
            icon.font = .systemFont(ofSize: 20)
            icon.textAlignment = .center
            icon.snp.makeConstraints { $0.width.equalTo(32) }
            let desc = UILabel()
            desc.text = day.weather
            desc.font = .systemFont(ofSize: 13)
            desc.textColor = AppTheme.textSecondary
            let range = UILabel()
            range.text = day.tempRange
            range.font = AppTheme.mono(13, weight: .medium)
            range.textColor = AppTheme.textPrimary
            range.textAlignment = .right
            let h = UIStackView(arrangedSubviews: [week, icon, desc, range])
            h.axis = .horizontal
            h.spacing = 8
            h.alignment = .center
            row.addSubview(h)
            h.snp.makeConstraints { $0.edges.equalToSuperview().inset(UIEdgeInsets(top: 0, left: 8, bottom: 0, right: 8)) }
            futureStack.addArrangedSubview(row)
            if index < days.count - 1 {
                let sep = UIView()
                sep.backgroundColor = AppTheme.divider
                sep.snp.makeConstraints { $0.height.equalTo(1 / UIScreen.main.scale) }
                futureStack.addArrangedSubview(sep)
            }
        }
        stack.addArrangedSubview(futureCard)
    }

    private func sectionTitle(_ text: String) -> UILabel {
        let l = UILabel()
        l.text = text
        l.font = .systemFont(ofSize: 15, weight: .semibold)
        l.textColor = AppTheme.textPrimary
        return l
    }

    private func kvRow(_ title: String, _ value: String) -> UIView {
        let row = UIStackView()
        row.axis = .horizontal
        row.distribution = .equalSpacing
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 14)
        t.textColor = AppTheme.textSecondary
        let v = UILabel()
        v.text = value
        v.font = .systemFont(ofSize: 14, weight: .medium)
        v.textColor = AppTheme.textPrimary
        v.textAlignment = .right
        row.addArrangedSubview(t)
        row.addArrangedSubview(v)
        return row
    }

    @objc private func refreshTapped() {
        Task {
            let city = await WeatherCityLocator.resolveCity()
            let info = await JuheWeatherService.shared.refresh(city: city)
            await MainActor.run {
                weather = info
                rebuild()
            }
        }
    }
}
