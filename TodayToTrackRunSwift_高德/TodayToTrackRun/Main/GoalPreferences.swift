//
//  GoalPreferences.swift
//  TodayToTrackRun
//

import Foundation

struct GoalPreset: Codable, Equatable {
    var title: String
    var subtitle: String
}

struct TodoItem: Equatable {
    let id: Int
    let title: String
    var done: Bool
}

enum GoalPreferences {
    private static let activeKey = "goal.activeIndex"
    private static let todoKey = "goal.todoStates"
    private static let streakKey = "goal.streakDays"
    private static let weekKey = "goal.weekCompleted"
    private static let customGoalsKey = "goal.customList"
    private static let dietKey = "goal.dietPreference"

    enum DietPreference: String, CaseIterable {
        case fatLoss
        case maintain
        case muscle

        var title: String {
            switch self {
            case .fatLoss: return "减脂 / 低糖低脂"
            case .maintain: return "均衡维持"
            case .muscle: return "增肌高蛋白"
            }
        }
    }

    static var dietPreference: DietPreference {
        let raw = UserDefaults.standard.string(forKey: dietKey) ?? DietPreference.fatLoss.rawValue
        return DietPreference(rawValue: raw) ?? .fatLoss
    }

    static var dietPreferenceTitle: String { dietPreference.title }

    static func setDietPreference(_ value: DietPreference) {
        UserDefaults.standard.set(value.rawValue, forKey: dietKey)
    }

    private static let builtIn: [GoalPreset] = [
        GoalPreset(title: "目标 1 · 减脂入门", subtitle: "每周跑 3 次，每次 30 分钟"),
        GoalPreset(title: "目标 2 · 5 公里", subtitle: "4 周内完成连续 5km"),
        GoalPreset(title: "目标 3 · 配速挑战", subtitle: "配速稳定在 6'/km 内")
    ]

    static var presets: [GoalPreset] {
        if let data = UserDefaults.standard.data(forKey: customGoalsKey),
           let decoded = try? JSONDecoder().decode([GoalPreset].self, from: data),
           !decoded.isEmpty {
            return decoded
        }
        return builtIn
    }

    static var activeIndex: Int {
        let maxIndex = max(presets.count - 1, 0)
        return min(max(UserDefaults.standard.integer(forKey: activeKey), 0), maxIndex)
    }

    static var activeGoalTitle: String {
        presets[safe: activeIndex]?.title ?? builtIn[0].title
    }

    static var activeGoalSubtitle: String {
        presets[safe: activeIndex]?.subtitle ?? builtIn[0].subtitle
    }

    static func setActiveIndex(_ index: Int) {
        let maxIndex = max(presets.count - 1, 0)
        UserDefaults.standard.set(min(max(index, 0), maxIndex), forKey: activeKey)
    }

    static func replaceAllGoals(_ goals: [GoalPreset]) {
        let list = goals.isEmpty ? builtIn : goals
        if let data = try? JSONEncoder().encode(list) {
            UserDefaults.standard.set(data, forKey: customGoalsKey)
        }
        setActiveIndex(min(activeIndex, list.count - 1))
    }

    static func upsertGoal(_ goal: GoalPreset, at index: Int?) {
        var list = presets
        if let index, list.indices.contains(index) {
            list[index] = goal
        } else {
            list.append(goal)
        }
        replaceAllGoals(list)
        if index == nil {
            setActiveIndex(list.count - 1)
        }
    }

    static func deleteGoal(at index: Int) {
        var list = presets
        guard list.indices.contains(index), list.count > 1 else { return }
        list.remove(at: index)
        replaceAllGoals(list)
    }

    static func resetToBuiltIn() {
        UserDefaults.standard.removeObject(forKey: customGoalsKey)
        setActiveIndex(0)
    }

    static func todos() -> [TodoItem] {
        let defaults = defaultTodos()
        let saved = UserDefaults.standard.dictionary(forKey: todoKey) as? [String: Bool] ?? [:]
        return defaults.map { item in
            var copy = item
            copy.done = saved["\(item.id)"] ?? false
            return copy
        }
    }

    static func toggleTodo(id: Int) {
        var saved = UserDefaults.standard.dictionary(forKey: todoKey) as? [String: Bool] ?? [:]
        let key = "\(id)"
        saved[key] = !(saved[key] ?? false)
        UserDefaults.standard.set(saved, forKey: todoKey)
        if id == 1, saved[key] == true {
            markTodayCompleted()
        }
    }

    static func markRunCompletedToday() {
        var saved = UserDefaults.standard.dictionary(forKey: todoKey) as? [String: Bool] ?? [:]
        saved["1"] = true
        UserDefaults.standard.set(saved, forKey: todoKey)
        markTodayCompleted()
    }

    static func streakDays() -> Int {
        UserDefaults.standard.integer(forKey: streakKey)
    }

    static func completedDaysThisWeek() -> Int {
        pruneWeekIfNeeded()
        let map = UserDefaults.standard.dictionary(forKey: weekKey) as? [String: Bool] ?? [:]
        return map.values.filter { $0 }.count
    }

    /// 本周一到周日的打卡状态（true = 已完成跑步日）
    static func weekSchedule() -> [(weekdayLabel: String, date: Date, completed: Bool, isToday: Bool)] {
        pruneWeekIfNeeded()
        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let weekday = calendar.component(.weekday, from: today) // 1=Sun
        let mondayOffset = (weekday + 5) % 7 // days since Monday
        guard let monday = calendar.date(byAdding: .day, value: -mondayOffset, to: today) else {
            return []
        }
        let labels = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"]
        let map = UserDefaults.standard.dictionary(forKey: weekKey) as? [String: Bool] ?? [:]
        return (0..<7).compactMap { offset in
            guard let date = calendar.date(byAdding: .day, value: offset, to: monday) else { return nil }
            let key = dayFormatter.string(from: date)
            return (
                weekdayLabel: labels[offset],
                date: date,
                completed: map[key] == true,
                isToday: calendar.isDate(date, inSameDayAs: today)
            )
        }
    }

    static func toggleWeekDayCompleted(_ date: Date) {
        pruneWeekIfNeeded()
        let key = dayFormatter.string(from: date)
        var week = UserDefaults.standard.dictionary(forKey: weekKey) as? [String: Bool] ?? [:]
        let was = week[key] == true
        week[key] = !was
        UserDefaults.standard.set(week, forKey: weekKey)
        if !was {
            UserDefaults.standard.set(streakDays() + 1, forKey: streakKey)
        } else {
            UserDefaults.standard.set(max(0, streakDays() - 1), forKey: streakKey)
        }
    }

    private static func markTodayCompleted() {
        pruneWeekIfNeeded()
        let dayKey = dayFormatter.string(from: Date())
        var week = UserDefaults.standard.dictionary(forKey: weekKey) as? [String: Bool] ?? [:]
        if week[dayKey] != true {
            week[dayKey] = true
            UserDefaults.standard.set(week, forKey: weekKey)
            UserDefaults.standard.set(streakDays() + 1, forKey: streakKey)
        }
    }

    /// 跨周时清掉不属于本周的日期键
    private static func pruneWeekIfNeeded() {
        let keys = Set(weekScheduleKeys())
        var week = UserDefaults.standard.dictionary(forKey: weekKey) as? [String: Bool] ?? [:]
        let filtered = week.filter { keys.contains($0.key) }
        if filtered.count != week.count {
            UserDefaults.standard.set(filtered, forKey: weekKey)
        }
    }

    private static func weekScheduleKeys() -> [String] {
        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let weekday = calendar.component(.weekday, from: today)
        let mondayOffset = (weekday + 5) % 7
        guard let monday = calendar.date(byAdding: .day, value: -mondayOffset, to: today) else { return [] }
        return (0..<7).compactMap { offset in
            guard let date = calendar.date(byAdding: .day, value: offset, to: monday) else { return nil }
            return dayFormatter.string(from: date)
        }
    }

    private static func defaultTodos() -> [TodoItem] {
        [
            TodoItem(id: 1, title: "晨跑或快走 30 分钟", done: false),
            TodoItem(id: 2, title: "记录三餐（低糖低脂）", done: false),
            TodoItem(id: 3, title: "跑后拉伸 10 分钟", done: false),
            TodoItem(id: 4, title: "喝水 8 杯", done: false)
        ]
    }

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f
    }()
}

private extension Array {
    subscript(safe index: Int) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}
