//
//  RunCompletionViewController.swift
//  TodayToTrackRun
//
//  跑后小结：距离/配速/消耗 + 本周目标进度。
//

import SnapKit
import UIKit

final class RunCompletionViewController: UIViewController {
    private let snapshot: RunSessionSnapshot
    private let cityName: String?
    private let calories: Int
    private let avgHR: Int
    var onViewHistory: (() -> Void)?

    private let card = UIView()
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()
    private let metricsStack = UIStackView()
    private let progressLabel = UILabel()
    private let progressView = UIProgressView(progressViewStyle: .default)
    private let goalLabel = UILabel()
    private let doneButton = UIButton(type: .system)
    private let historyButton = UIButton(type: .system)

    init(snapshot: RunSessionSnapshot, cityName: String?, calories: Int, avgHR: Int) {
        self.snapshot = snapshot
        self.cityName = cityName
        self.calories = calories
        self.avgHR = avgHR
        super.init(nibName: nil, bundle: nil)
        modalPresentationStyle = .pageSheet
        if let sheet = sheetPresentationController {
            sheet.detents = [.medium(), .large()]
            sheet.prefersGrabberVisible = true
        }
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        setupUI()
        fill()
    }

    private func setupUI() {
        card.backgroundColor = .white
        card.layer.cornerRadius = 16
        view.addSubview(card)
        card.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(20)
            make.leading.trailing.equalToSuperview().inset(20)
        }

        titleLabel.font = .systemFont(ofSize: 22, weight: .bold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.textAlignment = .center
        subtitleLabel.font = .systemFont(ofSize: 14)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.textAlignment = .center
        subtitleLabel.numberOfLines = 0

        metricsStack.axis = .horizontal
        metricsStack.distribution = .fillEqually
        metricsStack.spacing = 8

        progressLabel.font = .systemFont(ofSize: 13, weight: .medium)
        progressLabel.textColor = AppTheme.textSecondary
        progressView.progressTintColor = AppTheme.accent
        progressView.trackTintColor = UIColor(white: 0.9, alpha: 1)

        goalLabel.font = .systemFont(ofSize: 14)
        goalLabel.textColor = AppTheme.textPrimary
        goalLabel.numberOfLines = 0
        goalLabel.textAlignment = .center

        doneButton.setTitle("完成", for: .normal)
        doneButton.setTitleColor(.white, for: .normal)
        doneButton.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        doneButton.backgroundColor = AppTheme.accent
        doneButton.layer.cornerRadius = 12
        doneButton.addTarget(self, action: #selector(doneTapped), for: .touchUpInside)

        historyButton.setTitle("查看记录", for: .normal)
        historyButton.setTitleColor(AppTheme.accent, for: .normal)
        historyButton.titleLabel?.font = .systemFont(ofSize: 15, weight: .medium)
        historyButton.addTarget(self, action: #selector(historyTapped), for: .touchUpInside)

        let stack = UIStackView(arrangedSubviews: [
            titleLabel, subtitleLabel, metricsStack, progressLabel, progressView, goalLabel
        ])
        stack.axis = .vertical
        stack.spacing = 12
        card.addSubview(stack)
        stack.snp.makeConstraints { $0.edges.equalToSuperview().inset(18) }
        progressView.snp.makeConstraints { $0.height.equalTo(8) }

        view.addSubview(doneButton)
        view.addSubview(historyButton)
        doneButton.snp.makeConstraints { make in
            make.top.equalTo(card.snp.bottom).offset(20)
            make.leading.trailing.equalToSuperview().inset(20)
            make.height.equalTo(48)
        }
        historyButton.snp.makeConstraints { make in
            make.top.equalTo(doneButton.snp.bottom).offset(8)
            make.centerX.equalToSuperview()
        }
    }

    private func fill() {
        titleLabel.text = "本趟已完成"
        let place = cityName ?? "本地跑步"
        subtitleLabel.text = "\(place) · 已计入今日打卡"
        metricsStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        [
            ("距离", snapshot.formattedDistance),
            ("用时", snapshot.formattedDuration),
            ("配速", snapshot.formattedPace.replacingOccurrences(of: "/km", with: "")),
            ("消耗", "\(calories) kcal"),
            ("心率", avgHR > 0 ? "\(avgHR)" : "--")
        ].forEach { title, value in
            metricsStack.addArrangedSubview(metricTile(title: title, value: value))
        }

        let week = GoalPreferences.completedDaysThisWeek()
        progressLabel.text = "本周目标进度  \(week)/7 天 · 连续 \(GoalPreferences.streakDays()) 天"
        progressView.progress = Float(week) / 7
        goalLabel.text = "当前目标：\(GoalPreferences.activeGoalTitle)\n\(GoalPreferences.activeGoalSubtitle)"
    }

    private func metricTile(title: String, value: String) -> UIView {
        let box = UIView()
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 11)
        t.textColor = AppTheme.textSecondary
        t.textAlignment = .center
        let v = UILabel()
        v.text = value
        v.font = .systemFont(ofSize: 13, weight: .semibold)
        v.textColor = AppTheme.textPrimary
        v.textAlignment = .center
        v.adjustsFontSizeToFitWidth = true
        v.minimumScaleFactor = 0.7
        box.addSubview(t)
        box.addSubview(v)
        t.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
        }
        v.snp.makeConstraints { make in
            make.top.equalTo(t.snp.bottom).offset(4)
            make.leading.trailing.bottom.equalToSuperview()
        }
        return box
    }

    @objc private func doneTapped() {
        dismiss(animated: true)
    }

    @objc private func historyTapped() {
        dismiss(animated: true) { [weak self] in
            self?.onViewHistory?()
        }
    }
}
