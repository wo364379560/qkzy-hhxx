//
//  TodayHomeViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class TodayHomeViewController: UIViewController {
    private let scrollView = UIScrollView()
    private let content = UIStackView()

    private let greetingLabel = UILabel()
    private let dateLabel = UILabel()
    private let weatherCard = WeatherCardView()
    private let ring = RingProgressView()
    private let goalTitle = UILabel()
    private let goalHint = UILabel()
    private let startButton = UIButton(type: .system)
    private let paceValue = UILabel()
    private let hrValue = UILabel()
    private let kcalValue = UILabel()
    private let scheduleStack = UIStackView()
    private let recentTitle = UILabel()
    private let recentBody = UILabel()

    private var weather = WeatherInfo.placeholder
    private var weatherLoading = true

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "今日"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.largeTitleDisplayMode = .never
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        #if DEBUG
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "调试",
            style: .plain,
            target: self,
            action: #selector(debugMenu)
        )
        #endif
        setupUI()
        Task { await reloadWeather() }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        refreshSummary()
        reloadSchedulePreview()
    }

    private func setupUI() {
        scrollView.alwaysBounceVertical = true
        view.addSubview(scrollView)
        scrollView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        content.axis = .vertical
        content.spacing = 14
        content.isLayoutMarginsRelativeArrangement = true
        content.layoutMargins = UIEdgeInsets(top: 12, left: 16, bottom: 28, right: 16)
        scrollView.addSubview(content)
        content.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        greetingLabel.font = .systemFont(ofSize: 26, weight: .heavy)
        greetingLabel.textColor = AppTheme.textPrimary
        dateLabel.font = .systemFont(ofSize: 14)
        dateLabel.textColor = AppTheme.textSecondary
        content.addArrangedSubview(greetingLabel)
        content.addArrangedSubview(dateLabel)
        content.setCustomSpacing(6, after: greetingLabel)

        weatherCard.addTarget(self, action: #selector(openWeatherDetail), for: .touchUpInside)
        content.addArrangedSubview(weatherCard)
        weatherCard.snp.makeConstraints { $0.height.equalTo(132) }

        let dashCard = AppTheme.makeCard()
        let dashStack = UIStackView()
        dashStack.axis = .horizontal
        dashStack.spacing = 12
        dashStack.alignment = .center
        dashCard.addSubview(dashStack)
        dashStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }

        ring.snp.makeConstraints { $0.width.height.equalTo(118) }
        let right = UIStackView()
        right.axis = .vertical
        right.spacing = 8
        goalTitle.font = AppTheme.title(16)
        goalTitle.textColor = AppTheme.textPrimary
        goalTitle.numberOfLines = 0
        goalHint.font = .systemFont(ofSize: 13)
        goalHint.textColor = AppTheme.textSecondary
        goalHint.numberOfLines = 0
        stylePrimary(startButton, title: "开始跑步")
        startButton.addTarget(self, action: #selector(startRun), for: .touchUpInside)
        startButton.snp.makeConstraints { $0.height.equalTo(44) }
        [goalTitle, goalHint, startButton].forEach { right.addArrangedSubview($0) }
        dashStack.addArrangedSubview(ring)
        dashStack.addArrangedSubview(right)
        content.addArrangedSubview(dashCard)

        content.addArrangedSubview(sectionTitle("最近一次"))
        let metricsRow = UIStackView(arrangedSubviews: [
            metricTile(title: "配速", valueLabel: paceValue),
            metricTile(title: "心率", valueLabel: hrValue),
            metricTile(title: "消耗", valueLabel: kcalValue)
        ])
        metricsRow.axis = .horizontal
        metricsRow.spacing = 10
        metricsRow.distribution = .fillEqually
        content.addArrangedSubview(metricsRow)

        content.addArrangedSubview(sectionTitle("今日日程"))
        let scheduleCard = AppTheme.makeCard()
        scheduleStack.axis = .vertical
        scheduleStack.spacing = 0
        scheduleCard.addSubview(scheduleStack)
        scheduleStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(4) }
        content.addArrangedSubview(scheduleCard)

        let moreSchedule = UIButton(type: .system)
        moreSchedule.setTitle("管理全部日程 →", for: .normal)
        moreSchedule.setTitleColor(AppTheme.accent, for: .normal)
        moreSchedule.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        moreSchedule.contentHorizontalAlignment = .left
        moreSchedule.addTarget(self, action: #selector(goPlan), for: .touchUpInside)
        content.addArrangedSubview(moreSchedule)

        content.addArrangedSubview(sectionTitle("建议"))
        let adviceCard = AppTheme.makeCard()
        recentTitle.font = .systemFont(ofSize: 15, weight: .semibold)
        recentTitle.textColor = AppTheme.textPrimary
        recentTitle.text = "饮食 · 运动"
        recentBody.font = .systemFont(ofSize: 13)
        recentBody.textColor = AppTheme.textSecondary
        recentBody.numberOfLines = 0
        recentBody.text = "减脂期优先低糖低脂；跑步先保证时长，再追配速。"
        adviceCard.addSubview(recentTitle)
        adviceCard.addSubview(recentBody)
        recentTitle.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview().inset(14)
        }
        recentBody.snp.makeConstraints { make in
            make.top.equalTo(recentTitle.snp.bottom).offset(6)
            make.leading.trailing.bottom.equalToSuperview().inset(14)
        }
        content.addArrangedSubview(adviceCard)
    }

    private func refreshSummary() {
        greetingLabel.text = greeting()
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "yyyy年M月d日 EEEE"
        dateLabel.text = formatter.string(from: Date())

        goalTitle.text = GoalPreferences.activeGoalTitle
        goalHint.text = GoalPreferences.activeGoalSubtitle
        let week = GoalPreferences.completedDaysThisWeek()
        ring.progress = CGFloat(week) / 7
        ring.centerTitle = "\(week)/7"
        ring.centerSubtitle = "本周"

        if let latest = try? RunRecordStore.listSummaries(limit: 1).first {
            paceValue.text = latest.formattedPace.replacingOccurrences(of: "/km", with: "")
            let hr = latest.averageHeartRate > 0
                ? latest.averageHeartRate
                : RunMetricsEstimator.estimatedAverageHeartRate(
                    paceSecondsPerKm: latest.averagePaceSecondsPerKm,
                    durationSeconds: latest.durationSeconds
                )
            hrValue.text = hr > 0 ? "\(hr)" : "--"
            let kcal = latest.caloriesKcal > 0
                ? latest.caloriesKcal
                : RunMetricsEstimator.estimatedCalories(
                    distanceMeters: latest.distanceMeters,
                    durationSeconds: latest.durationSeconds
                )
            kcalValue.text = "\(kcal)"
        } else {
            paceValue.text = "--"
            hrValue.text = "--"
            kcalValue.text = "--"
        }

        weatherCard.configure(weather, loading: weatherLoading)
    }

    private func reloadSchedulePreview() {
        scheduleStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let items = TrainingScheduleStore.items(on: Date())
        if items.isEmpty {
            let empty = UILabel()
            empty.text = "今天暂无训练日程，去计划页添加"
            empty.font = .systemFont(ofSize: 14)
            empty.textColor = AppTheme.textSecondary
            empty.textAlignment = .center
            empty.heightAnchor.constraint(equalToConstant: 52).isActive = true
            scheduleStack.addArrangedSubview(empty)
            return
        }
        for item in items.prefix(4) {
            let row = UIButton(type: .system)
            row.contentHorizontalAlignment = .left
            row.contentEdgeInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
            let mark = item.done ? "✓" : "○"
            row.setTitle("\(mark)  \(item.timeText)  \(item.title)", for: .normal)
            row.setTitleColor(item.done ? AppTheme.textSecondary : AppTheme.textPrimary, for: .normal)
            row.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
            row.tag = item.id.hashValue
            row.accessibilityIdentifier = item.id
            row.addAction(UIAction { [weak self] _ in
                TrainingScheduleStore.toggleDone(id: item.id)
                self?.reloadSchedulePreview()
            }, for: .touchUpInside)
            scheduleStack.addArrangedSubview(row)
        }
    }

    private func reloadWeather() async {
        weatherLoading = true
        await MainActor.run { weatherCard.configure(weather, loading: true) }
        // 对齐小确：先定位当前城市，失败再回退已保存城市
        let city = await WeatherCityLocator.resolveCity()
        let info = await JuheWeatherService.shared.refresh(city: city)
        await MainActor.run {
            weather = info
            weatherLoading = false
            weatherCard.configure(info, loading: false)
        }
    }

    private func greeting() -> String {
        let hour = Calendar.current.component(.hour, from: Date())
        switch hour {
        case 6...11: return "早上好"
        case 12...13: return "中午好"
        case 14...17: return "下午好"
        default: return "晚上好"
        }
    }

    @objc private func startRun() {
        guard let tabs = tabBarController else { return }
        // 底部顺序：运动、统计、消息、我的。今日页是从统计页 push 进来的，
        // 切换前清掉统计导航栈，避免下次点统计仍停留在今日详情。
        navigationController?.popToRootViewController(animated: false)
        tabs.selectedIndex = 0
    }

    @objc private func goPlan() {
        linLanPush(PlanHomeViewController(initialSection: .schedules))
    }

    @objc private func openWeatherDetail() {
        let vc = WeatherDetailViewController(weather: weather)
        linLanPush(vc)
    }

    #if DEBUG
    @objc private func debugMenu() {
        let sheet = UIAlertController(title: "调试", message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "室内模拟跑步", style: .default) { [weak self] _ in
            self?.navigationController?.popToRootViewController(animated: false)
            self?.tabBarController?.selectedIndex = 0
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) {
                NotificationCenter.default.post(name: .todayTrackStartSimulatedRun, object: nil)
            }
        })
        sheet.addAction(UIAlertAction(title: "插入演示记录", style: .default) { _ in
            _ = try? MockRunGenerator.seedDemoRecords(count: 3)
        })
        sheet.addAction(UIAlertAction(title: "刷新天气", style: .default) { [weak self] _ in
            Task { await self?.reloadWeather() }
        })
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }
    #endif

    private func stylePrimary(_ button: UIButton, title: String) {
        button.setTitle(title, for: .normal)
        button.setTitleColor(.white, for: .normal)
        button.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        button.backgroundColor = AppTheme.accent
        button.layer.cornerRadius = 12
    }

    private func sectionTitle(_ text: String) -> UILabel {
        let l = UILabel()
        l.text = text
        l.font = .systemFont(ofSize: 15, weight: .semibold)
        l.textColor = AppTheme.textPrimary
        return l
    }

    private func metricTile(title: String, valueLabel: UILabel) -> UIView {
        let card = AppTheme.makeCard()
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 12)
        t.textColor = AppTheme.textSecondary
        valueLabel.font = AppTheme.mono(18, weight: .semibold)
        valueLabel.textColor = AppTheme.textPrimary
        valueLabel.text = "--"
        card.addSubview(t)
        card.addSubview(valueLabel)
        t.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview().inset(12)
        }
        valueLabel.snp.makeConstraints { make in
            make.top.equalTo(t.snp.bottom).offset(6)
            make.leading.trailing.bottom.equalToSuperview().inset(12)
        }
        return card
    }
}

extension Notification.Name {
    static let todayTrackStartSimulatedRun = Notification.Name("todayTrackStartSimulatedRun")
}
