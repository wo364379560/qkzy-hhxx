//
//  LinLanStatsViewController.swift
//  TodayToTrackRun
//
//  统计 Tab：承接原「今日 / 计划」摘要，米白 + 珊瑚橙
//

import SnapKit
import UIKit

final class LinLanStatsViewController: UIViewController {
    private let scrollView = UIScrollView()
    private let content = UIStackView()
    private let ring = RingProgressView()
    private let weekLabel = UILabel()
    private let recentBody = UILabel()
    private let kmValue = UILabel()
    private let countValue = UILabel()
    private let kcalValue = UILabel()

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "统计"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.largeTitleDisplayMode = .never
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        setupUI()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reload()
    }

    private func setupUI() {
        view.addSubview(scrollView)
        scrollView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        content.axis = .vertical
        content.spacing = 14
        content.isLayoutMarginsRelativeArrangement = true
        content.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 28, right: 16)
        scrollView.addSubview(content)
        content.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        let title = UILabel()
        title.text = "本周概览"
        title.font = .systemFont(ofSize: 13, weight: .medium)
        title.textColor = AppTheme.textSecondary
        content.addArrangedSubview(title)

        let weekCard = AppTheme.makeCard()
        let weekStack = UIStackView()
        weekStack.axis = .vertical
        weekStack.alignment = .center
        weekStack.spacing = 10
        weekCard.addSubview(weekStack)
        weekStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(20) }
        ring.snp.makeConstraints { $0.width.height.equalTo(140) }
        weekLabel.font = AppTheme.mono(18, weight: .bold)
        weekLabel.textColor = AppTheme.textPrimary
        weekLabel.textAlignment = .center
        weekLabel.numberOfLines = 2
        weekStack.addArrangedSubview(ring)
        weekStack.addArrangedSubview(weekLabel)
        content.addArrangedSubview(weekCard)

        let statsTitle = UILabel()
        statsTitle.text = "累计数据"
        statsTitle.font = .systemFont(ofSize: 13, weight: .medium)
        statsTitle.textColor = AppTheme.textSecondary
        content.addArrangedSubview(statsTitle)

        let statsCard = AppTheme.makeCard()
        let statsRow = UIStackView(arrangedSubviews: [
            linLanStatColumn(kmValue, "总公里", true),
            linLanStatColumn(countValue, "总次数", false),
            linLanStatColumn(kcalValue, "卡路里", false)
        ])
        statsRow.axis = .horizontal
        statsRow.distribution = .fillEqually
        statsCard.addSubview(statsRow)
        statsRow.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }
        content.addArrangedSubview(statsCard)

        let recentTitle = UILabel()
        recentTitle.text = "最近运动"
        recentTitle.font = .systemFont(ofSize: 13, weight: .medium)
        recentTitle.textColor = AppTheme.textSecondary
        content.addArrangedSubview(recentTitle)

        let recentCard = AppTheme.makeCard()
        recentBody.font = .systemFont(ofSize: 15)
        recentBody.textColor = AppTheme.textPrimary
        recentBody.numberOfLines = 0
        recentCard.addSubview(recentBody)
        recentBody.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }
        recentCard.isUserInteractionEnabled = true
        recentCard.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(openHistory)))
        content.addArrangedSubview(recentCard)

        content.addArrangedSubview(linLanActionButton("训练计划", action: #selector(openPlan)))
        content.addArrangedSubview(linLanActionButton("查看全部记录", action: #selector(openHistory)))
        content.addArrangedSubview(linLanActionButton("今日详情（天气/目标）", action: #selector(openToday)))
    }

    private func linLanStatColumn(_ value: UILabel, _ title: String, _ accent: Bool) -> UIView {
        let col = UIStackView()
        col.axis = .vertical
        col.alignment = .center
        col.spacing = 4
        value.font = AppTheme.mono(20, weight: .bold)
        value.textColor = accent ? AppTheme.accent : AppTheme.textPrimary
        value.text = "—"
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 12)
        t.textColor = AppTheme.textSecondary
        col.addArrangedSubview(value)
        col.addArrangedSubview(t)
        return col
    }

    private func linLanActionButton(_ title: String, action: Selector) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.setTitleColor(AppTheme.accent, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        btn.backgroundColor = .white
        btn.layer.cornerRadius = 14
        btn.contentEdgeInsets = UIEdgeInsets(top: 14, left: 16, bottom: 14, right: 16)
        btn.addTarget(self, action: action, for: .touchUpInside)
        return btn
    }

    private func reload() {
        let summaries = (try? RunRecordStore.listSummaries()) ?? []
        let totalKm = summaries.reduce(0.0) { $0 + $1.distanceMeters } / 1000.0
        let totalKcal = summaries.reduce(0.0) { $0 + Double($1.caloriesKcal) }
        kmValue.text = String(format: "%.1f", totalKm)
        countValue.text = "\(summaries.count)"
        kcalValue.text = String(format: "%.0f", totalKcal)

        // 简易周进度：本周公里 / 目标（缺省 20km）
        let cal = Calendar.current
        let weekStart = cal.date(from: cal.dateComponents([.yearForWeekOfYear, .weekOfYear], from: Date())) ?? Date()
        let weekKm = summaries
            .filter { $0.startedAt >= weekStart }
            .reduce(0.0) { $0 + $1.distanceMeters } / 1000.0
        let goal = 20.0 // 默认周目标 km，后续接 GoalPreferences
        ring.progress = CGFloat(min(weekKm / goal, 1))
        weekLabel.text = String(format: "本周 %.1f / %.0f km", weekKm, goal)

        if let latest = summaries.first {
            let df = DateFormatter()
            df.dateFormat = "M月d日 HH:mm"
            recentBody.text = "\(df.string(from: latest.startedAt))\n\(String(format: "%.2f km", latest.distanceMeters / 1000)) · 点击查看全部"
        } else {
            recentBody.text = "暂无运动记录\n去「运动」开始第一次跑步吧"
        }
    }

    @objc private func openHistory() {
        linLanPush(RunHistoryViewController())
    }

    @objc private func openPlan() {
        linLanPush(PlanHomeViewController())
    }

    @objc private func openToday() {
        linLanPush(TodayHomeViewController())
    }
}
