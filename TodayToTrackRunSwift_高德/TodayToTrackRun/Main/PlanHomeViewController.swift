//
//  PlanHomeViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class PlanHomeViewController: UIViewController {
    enum InitialSection {
        case top
        case schedules
    }

    private let scrollView = UIScrollView()
    private let content = UIStackView()
    private let goalsStack = UIStackView()
    private let courseStack = UIStackView()
    private let todoStack = UIStackView()
    private let weekStack = UIStackView()
    private let scheduleStack = UIStackView()
    private let streakLabel = UILabel()
    private let enrolledLabel = UILabel()
    private let initialSection: InitialSection
    private var didApplyInitialScroll = false

    init(initialSection: InitialSection = .top) {
        self.initialSection = initialSection
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        initialSection = .top
        super.init(coder: coder)
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "计划"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.rightBarButtonItems = [
            UIBarButtonItem(title: "加日程", style: .plain, target: self, action: #selector(addSchedule)),
            UIBarButtonItem(title: "新建目标", style: .plain, target: self, action: #selector(addGoal))
        ]
        setupUI()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadGoals()
        reloadTodos()
        reloadWeekSchedule()
        reloadTrainingSchedules()
        reloadCourses()
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        scrollToInitialSectionIfNeeded()
    }

    private func setupUI() {
        view.addSubview(scrollView)
        scrollView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        content.axis = .vertical
        content.spacing = 12
        content.isLayoutMarginsRelativeArrangement = true
        content.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 24, right: 16)
        scrollView.addSubview(content)
        content.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        content.addArrangedSubview(sectionLabel("训练课程"))
        enrolledLabel.font = .systemFont(ofSize: 13)
        enrolledLabel.textColor = AppTheme.textSecondary
        enrolledLabel.numberOfLines = 0
        content.addArrangedSubview(enrolledLabel)
        courseStack.axis = .vertical
        courseStack.spacing = 8
        content.addArrangedSubview(courseStack)

        content.addArrangedSubview(sectionLabel("自设运动目标"))
        goalsStack.axis = .vertical
        goalsStack.spacing = 8
        content.addArrangedSubview(goalsStack)

        let actions = UIStackView()
        actions.axis = .horizontal
        actions.spacing = 8
        actions.distribution = .fillEqually
        let addBtn = makeActionButton("自定义目标")
        addBtn.addTarget(self, action: #selector(addGoal), for: .touchUpInside)
        let resetBtn = makeActionButton("恢复默认 3 套")
        resetBtn.addTarget(self, action: #selector(resetGoals), for: .touchUpInside)
        actions.addArrangedSubview(addBtn)
        actions.addArrangedSubview(resetBtn)
        content.addArrangedSubview(actions)

        streakLabel.font = .systemFont(ofSize: 14, weight: .medium)
        streakLabel.textColor = AppTheme.textSecondary
        content.addArrangedSubview(streakLabel)

        content.addArrangedSubview(sectionLabel("今日待办 · 打卡"))
        todoStack.axis = .vertical
        todoStack.spacing = 8
        content.addArrangedSubview(todoStack)

        content.addArrangedSubview(sectionLabel("本周打卡"))
        let weekCard = AppTheme.makeCard()
        weekStack.axis = .vertical
        weekStack.spacing = 0
        weekCard.addSubview(weekStack)
        weekStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(4) }
        content.addArrangedSubview(weekCard)
        reloadWeekSchedule()

        content.addArrangedSubview(sectionLabel("训练日程"))
        let scheduleCard = AppTheme.makeCard()
        scheduleStack.axis = .vertical
        scheduleStack.spacing = 0
        scheduleCard.addSubview(scheduleStack)
        scheduleStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(4) }
        content.addArrangedSubview(scheduleCard)
        reloadTrainingSchedules()
    }

    private func scrollToInitialSectionIfNeeded() {
        guard !didApplyInitialScroll, initialSection == .schedules else { return }
        didApplyInitialScroll = true
        view.layoutIfNeeded()
        let target = scheduleStack.convert(scheduleStack.bounds, to: scrollView)
        let topPadding: CGFloat = 52
        let y = max(-scrollView.adjustedContentInset.top, target.minY - topPadding)
        scrollView.setContentOffset(CGPoint(x: 0, y: y), animated: false)
    }

    private func reloadTrainingSchedules() {
        scheduleStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let upcoming = TrainingScheduleStore.all().prefix(12)
        if upcoming.isEmpty {
            let empty = UILabel()
            empty.text = "暂无日程，点右上角「加日程」"
            empty.font = .systemFont(ofSize: 14)
            empty.textColor = AppTheme.textSecondary
            empty.textAlignment = .center
            empty.heightAnchor.constraint(equalToConstant: 48).isActive = true
            scheduleStack.addArrangedSubview(empty)
            return
        }
        for item in upcoming {
            let row = UIButton(type: .system)
            row.contentHorizontalAlignment = .left
            row.contentEdgeInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
            let mark = item.done ? "✓" : "○"
            row.setTitle("\(mark)  \(item.dateKey) \(item.timeText)\n    \(item.title)", for: .normal)
            row.titleLabel?.numberOfLines = 2
            row.setTitleColor(item.done ? AppTheme.textSecondary : AppTheme.textPrimary, for: .normal)
            row.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
            let id = item.id
            row.addAction(UIAction { [weak self] _ in
                TrainingScheduleStore.toggleDone(id: id)
                self?.reloadTrainingSchedules()
            }, for: .touchUpInside)
            let long = UILongPressGestureRecognizer(target: self, action: #selector(scheduleLongPress(_:)))
            row.accessibilityIdentifier = id
            row.addGestureRecognizer(long)
            scheduleStack.addArrangedSubview(row)
        }
    }

    @objc private func scheduleLongPress(_ gesture: UILongPressGestureRecognizer) {
        guard gesture.state == .began,
              let button = gesture.view as? UIButton,
              let id = button.accessibilityIdentifier
        else { return }
        let sheet = UIAlertController(title: "日程", message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "删除", style: .destructive) { [weak self] _ in
            TrainingScheduleStore.delete(id: id)
            self?.reloadTrainingSchedules()
        })
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    @objc private func addSchedule() {
        let alert = UIAlertController(title: "添加训练日程", message: "默认加到今天", preferredStyle: .alert)
        alert.addTextField { $0.placeholder = "例如：轻松慢跑 30 分钟" }
        alert.addTextField { $0.placeholder = "时间（如 07:00）"; $0.text = "07:00" }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "添加", style: .default) { [weak self] _ in
            let title = alert.textFields?[0].text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            let time = alert.textFields?[1].text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "全天"
            guard !title.isEmpty else { return }
            let item = TrainingScheduleItem.make(title: title, date: Date(), timeText: time.isEmpty ? "全天" : time)
            TrainingScheduleStore.upsert(item)
            self?.reloadTrainingSchedules()
        })
        present(alert, animated: true)
    }

    private func reloadCourses() {
        courseStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        if let enrolled = TrainingCourseCatalog.enrolledCourse {
            enrolledLabel.text = "当前课程：\(enrolled.title)（报名后任务已写入训练日程）"
        } else {
            enrolledLabel.text = "选择课程后，会自动生成后续训练日程（纯本地）"
        }
        for course in TrainingCourseCatalog.all {
            let card = AppTheme.makeCard()
            let title = UILabel()
            title.text = course.title
            title.font = .systemFont(ofSize: 16, weight: .semibold)
            title.textColor = AppTheme.textPrimary
            let sub = UILabel()
            sub.text = "\(course.subtitle)\n\(course.targetLabel) · \(course.sessions.count) 次课"
            sub.font = .systemFont(ofSize: 13)
            sub.textColor = AppTheme.textSecondary
            sub.numberOfLines = 0
            let btn = UIButton(type: .system)
            let isCurrent = TrainingCourseCatalog.enrolledCourseID == course.id
            btn.setTitle(isCurrent ? "已报名 · 再生成一次" : "报名并生成日程", for: .normal)
            btn.setTitleColor(.white, for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
            btn.backgroundColor = AppTheme.accent
            btn.layer.cornerRadius = 10
            btn.contentEdgeInsets = UIEdgeInsets(top: 10, left: 12, bottom: 10, right: 12)
            let courseID = course.id
            btn.addAction(UIAction { [weak self] _ in
                self?.enrollCourse(id: courseID)
            }, for: .touchUpInside)
            let stack = UIStackView(arrangedSubviews: [title, sub, btn])
            stack.axis = .vertical
            stack.spacing = 8
            card.addSubview(stack)
            stack.snp.makeConstraints { $0.edges.equalToSuperview().inset(14) }
            courseStack.addArrangedSubview(card)
        }
    }

    private func enrollCourse(id: String) {
        guard let course = TrainingCourseCatalog.all.first(where: { $0.id == id }) else { return }
        let count = TrainingCourseCatalog.enroll(course)
        reloadCourses()
        reloadTrainingSchedules()
        let alert = UIAlertController(
            title: "已报名 \(course.title)",
            message: "已写入 \(count) 条训练日程，可在下方「训练日程」和今日页查看。",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "好", style: .default))
        present(alert, animated: true)
    }

    private func reloadWeekSchedule() {
        weekStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        for (index, day) in GoalPreferences.weekSchedule().enumerated() {
            let row = UIButton(type: .system)
            row.tag = index
            row.contentHorizontalAlignment = .left
            row.contentEdgeInsets = UIEdgeInsets(top: 10, left: 12, bottom: 10, right: 12)
            let mark = day.completed ? "✓" : "○"
            let todayMark = day.isToday ? " · 今天" : ""
            row.setTitle("\(mark)  \(day.weekdayLabel)\(todayMark)", for: .normal)
            row.setTitleColor(
                day.completed ? AppTheme.accent : AppTheme.textPrimary,
                for: .normal
            )
            row.titleLabel?.font = .systemFont(ofSize: 14, weight: day.isToday ? .semibold : .regular)
            row.addTarget(self, action: #selector(toggleWeekDay(_:)), for: .touchUpInside)
            weekStack.addArrangedSubview(row)
        }
    }

    @objc private func toggleWeekDay(_ sender: UIButton) {
        let days = GoalPreferences.weekSchedule()
        guard days.indices.contains(sender.tag) else { return }
        GoalPreferences.toggleWeekDayCompleted(days[sender.tag].date)
        reloadWeekSchedule()
        reloadGoals()
    }

    private func reloadGoals() {
        goalsStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let active = GoalPreferences.activeIndex
        for (index, preset) in GoalPreferences.presets.enumerated() {
            let button = UIButton(type: .system)
            button.tag = index
            button.contentHorizontalAlignment = .left
            button.titleLabel?.numberOfLines = 0
            button.layer.cornerRadius = 12
            button.layer.borderWidth = 1.5
            button.contentEdgeInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
            button.setTitle("\(preset.title)\n\(preset.subtitle)", for: .normal)
            button.setTitleColor(AppTheme.textPrimary, for: .normal)
            button.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
            let selected = index == active
            button.layer.borderColor = (selected ? AppTheme.accent : UIColor(white: 0.85, alpha: 1)).cgColor
            button.backgroundColor = selected ? AppTheme.accent.withAlphaComponent(0.08) : .white
            button.addTarget(self, action: #selector(selectGoal(_:)), for: .touchUpInside)

            let long = UILongPressGestureRecognizer(target: self, action: #selector(goalLongPress(_:)))
            button.addGestureRecognizer(long)
            goalsStack.addArrangedSubview(button)
        }
        streakLabel.text = "当前连续打卡 \(GoalPreferences.streakDays()) 天 · 本周已完成 \(GoalPreferences.completedDaysThisWeek())/7 · 长按目标可编辑/删除"
    }

    private func reloadTodos() {
        todoStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        for item in GoalPreferences.todos() {
            let row = UIButton(type: .system)
            row.contentHorizontalAlignment = .left
            row.backgroundColor = .white
            row.layer.cornerRadius = 10
            row.contentEdgeInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
            row.tag = item.id
            let mark = item.done ? "✓" : "○"
            row.setTitle("\(mark)  \(item.title)", for: .normal)
            row.setTitleColor(item.done ? AppTheme.textSecondary : AppTheme.textPrimary, for: .normal)
            row.addTarget(self, action: #selector(toggleTodo(_:)), for: .touchUpInside)
            todoStack.addArrangedSubview(row)
        }
    }

    @objc private func selectGoal(_ sender: UIButton) {
        GoalPreferences.setActiveIndex(sender.tag)
        reloadGoals()
    }

    @objc private func goalLongPress(_ gesture: UILongPressGestureRecognizer) {
        guard gesture.state == .began, let button = gesture.view as? UIButton else { return }
        let index = button.tag
        let sheet = UIAlertController(title: GoalPreferences.presets[safe: index]?.title, message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "编辑", style: .default) { [weak self] _ in
            self?.linLanPush(CustomGoalEditorViewController(editingIndex: index))
        })
        if GoalPreferences.presets.count > 1 {
            sheet.addAction(UIAlertAction(title: "删除", style: .destructive) { [weak self] _ in
                GoalPreferences.deleteGoal(at: index)
                self?.reloadGoals()
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    @objc private func addGoal() {
        linLanPush(CustomGoalEditorViewController())
    }

    @objc private func resetGoals() {
        GoalPreferences.resetToBuiltIn()
        reloadGoals()
    }

    @objc private func toggleTodo(_ sender: UIButton) {
        GoalPreferences.toggleTodo(id: sender.tag)
        reloadTodos()
        reloadGoals()
    }

    private func sectionLabel(_ text: String) -> UILabel {
        let l = UILabel()
        l.text = text
        l.font = .systemFont(ofSize: 15, weight: .semibold)
        l.textColor = AppTheme.textPrimary
        return l
    }

    private func makeActionButton(_ title: String) -> UIButton {
        let b = UIButton(type: .system)
        b.setTitle(title, for: .normal)
        b.setTitleColor(AppTheme.accent, for: .normal)
        b.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        b.backgroundColor = .white
        b.layer.cornerRadius = 10
        b.contentEdgeInsets = UIEdgeInsets(top: 10, left: 8, bottom: 10, right: 8)
        return b
    }
}

private extension Array {
    subscript(safe index: Int) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}
