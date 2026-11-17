//
//  TrainingScheduleStore.swift
//  TodayToTrackRun
//
//  从 chixin 仪表盘「日程」思路移植：本地日程（标题/日期/时间/备注）
//

import Foundation

struct TrainingScheduleItem: Codable, Equatable, Identifiable {
    var id: String
    var title: String
    var dateKey: String // yyyy-MM-dd
    var timeText: String // HH:mm 或全天
    var note: String
    var done: Bool

    static func make(title: String, date: Date, timeText: String = "全天", note: String = "") -> TrainingScheduleItem {
        TrainingScheduleItem(
            id: UUID().uuidString,
            title: title,
            dateKey: TrainingScheduleStore.dayKey(date),
            timeText: timeText,
            note: note,
            done: false
        )
    }
}

enum TrainingScheduleStore {
    private static let key = "training.schedules.v1"

    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    static func dayKey(_ date: Date) -> String {
        formatter.string(from: date)
    }

    static func all() -> [TrainingScheduleItem] {
        guard
            let data = UserDefaults.standard.data(forKey: key),
            let list = try? JSONDecoder().decode([TrainingScheduleItem].self, from: data)
        else {
            return seedIfNeeded()
        }
        return list.sorted { $0.dateKey < $1.dateKey }
    }

    static func items(on date: Date) -> [TrainingScheduleItem] {
        let key = dayKey(date)
        return all().filter { $0.dateKey == key }
    }

    static func markedDateKeys(inMonthOf date: Date) -> Set<String> {
        let cal = Calendar.current
        let comps = cal.dateComponents([.year, .month], from: date)
        guard let start = cal.date(from: comps),
              let end = cal.date(byAdding: DateComponents(month: 1, day: -1), to: start)
        else { return [] }
        let startKey = dayKey(start)
        let endKey = dayKey(end)
        return Set(all().map(\.dateKey).filter { $0 >= startKey && $0 <= endKey })
    }

    static func upsert(_ item: TrainingScheduleItem) {
        var list = all()
        if let idx = list.firstIndex(where: { $0.id == item.id }) {
            list[idx] = item
        } else {
            list.append(item)
        }
        save(list)
    }

    static func toggleDone(id: String) {
        var list = all()
        guard let idx = list.firstIndex(where: { $0.id == id }) else { return }
        list[idx].done.toggle()
        save(list)
    }

    static func delete(id: String) {
        save(all().filter { $0.id != id })
    }

    private static func save(_ list: [TrainingScheduleItem]) {
        if let data = try? JSONEncoder().encode(list) {
            UserDefaults.standard.set(data, forKey: key)
        }
    }

    /// 首次空列表时给几条示例，避免日程区块空白
    private static func seedIfNeeded() -> [TrainingScheduleItem] {
        let today = Date()
        let cal = Calendar.current
        let samples = [
            TrainingScheduleItem.make(title: "轻松慢跑 30 分钟", date: today, timeText: "07:00", note: "配速随意，热身充分"),
            TrainingScheduleItem.make(
                title: "拉伸放松",
                date: cal.date(byAdding: .day, value: 1, to: today) ?? today,
                timeText: "21:00"
            ),
            TrainingScheduleItem.make(
                title: "间歇跑课",
                date: cal.date(byAdding: .day, value: 2, to: today) ?? today,
                timeText: "18:30",
                note: "400m × 6"
            )
        ]
        save(samples)
        return samples
    }
}
