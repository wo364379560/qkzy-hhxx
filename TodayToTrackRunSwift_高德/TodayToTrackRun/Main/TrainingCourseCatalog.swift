//
//  TrainingCourseCatalog.swift
//  TodayToTrackRun
//
//  本地训练课程包：5K / 10K / 半马（无需服务端）
//

import Foundation

struct TrainingCourseSession: Equatable {
    let dayOffset: Int
    let title: String
    let timeText: String
    let note: String
}

struct TrainingCourse: Equatable, Identifiable {
    let id: String
    let title: String
    let subtitle: String
    let weeks: Int
    let targetLabel: String
    let sessions: [TrainingCourseSession]
}

enum TrainingCourseCatalog {
    static let all: [TrainingCourse] = [course5K, course10K, courseHalf]

    static let course5K = TrainingCourse(
        id: "course_5k",
        title: "5 公里入门",
        subtitle: "4 周 · 每周 3 练 · 适合零基础",
        weeks: 4,
        targetLabel: "目标 5km",
        sessions: [
            .init(dayOffset: 0, title: "走跑结合 25 分钟", timeText: "07:00", note: "跑 2 分钟 + 走 2 分钟 × 6"),
            .init(dayOffset: 2, title: "轻松慢跑 20 分钟", timeText: "18:30", note: "能说话的配速"),
            .init(dayOffset: 4, title: "间歇：快跑 1′ / 走 2′ × 6", timeText: "18:30", note: "热身 5′ + 拉伸"),
            .init(dayOffset: 7, title: "轻松跑 25 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 9, title: "爬坡或天桥练习 20 分钟", timeText: "18:30", note: "注意膝盖缓冲"),
            .init(dayOffset: 11, title: "轻松跑 30 分钟", timeText: "18:30", note: ""),
            .init(dayOffset: 14, title: "配速跑 3km", timeText: "07:00", note: "目标配速 6'30\"～7'"),
            .init(dayOffset: 16, title: "恢复跑 20 分钟", timeText: "18:30", note: ""),
            .init(dayOffset: 18, title: "长距离尝试 4km", timeText: "08:00", note: "中途可走 1～2 次"),
            .init(dayOffset: 21, title: "轻松跑 25 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 23, title: "赛前减量 20 分钟", timeText: "18:30", note: "别追求速度"),
            .init(dayOffset: 25, title: "5 公里挑战日", timeText: "08:00", note: "完成即可，时间次要")
        ]
    )

    static let course10K = TrainingCourse(
        id: "course_10k",
        title: "10 公里进阶",
        subtitle: "6 周 · 已有 5km 基础",
        weeks: 6,
        targetLabel: "目标 10km",
        sessions: [
            .init(dayOffset: 0, title: "轻松跑 35 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 2, title: "间歇 400m × 6", timeText: "18:30", note: "组间走 90 秒"),
            .init(dayOffset: 4, title: "长距离 7km", timeText: "08:00", note: "匀速"),
            .init(dayOffset: 7, title: "轻松跑 40 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 9, title: "节奏跑 5km", timeText: "18:30", note: "比轻松跑快 20～30 秒/km"),
            .init(dayOffset: 11, title: "长距离 8km", timeText: "08:00", note: ""),
            .init(dayOffset: 14, title: "轻松跑 40 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 16, title: "间歇 800m × 4", timeText: "18:30", note: ""),
            .init(dayOffset: 18, title: "长距离 9km", timeText: "08:00", note: ""),
            .init(dayOffset: 21, title: "轻松跑 35 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 23, title: "节奏跑 6km", timeText: "18:30", note: ""),
            .init(dayOffset: 25, title: "长距离 10km 试跑", timeText: "08:00", note: "可略走休"),
            .init(dayOffset: 28, title: "恢复跑 30 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 30, title: "赛前减量 25 分钟", timeText: "18:30", note: ""),
            .init(dayOffset: 32, title: "10 公里挑战日", timeText: "08:00", note: "")
        ]
    )

    static let courseHalf = TrainingCourse(
        id: "course_half",
        title: "半马备赛",
        subtitle: "8 周 · 需有 10km 能力",
        weeks: 8,
        targetLabel: "目标 21.1km",
        sessions: [
            .init(dayOffset: 0, title: "轻松跑 50 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 2, title: "间歇 1km × 4", timeText: "18:30", note: "组间慢跑 3′"),
            .init(dayOffset: 4, title: "长距离 14km", timeText: "07:30", note: "补水计划"),
            .init(dayOffset: 7, title: "轻松跑 45 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 9, title: "节奏跑 8km", timeText: "18:30", note: ""),
            .init(dayOffset: 11, title: "长距离 16km", timeText: "07:30", note: ""),
            .init(dayOffset: 14, title: "恢复跑 40 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 16, title: "间歇 1km × 5", timeText: "18:30", note: ""),
            .init(dayOffset: 18, title: "长距离 18km", timeText: "07:30", note: ""),
            .init(dayOffset: 21, title: "轻松跑 45 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 23, title: "节奏跑 10km", timeText: "18:30", note: ""),
            .init(dayOffset: 25, title: "长距离 20km", timeText: "07:30", note: "最长一练"),
            .init(dayOffset: 28, title: "减量轻松跑 35 分钟", timeText: "07:00", note: ""),
            .init(dayOffset: 30, title: "赛前激活 25 分钟", timeText: "18:30", note: "含几次加速跑"),
            .init(dayOffset: 32, title: "半马比赛日", timeText: "07:00", note: "按既定配速执行")
        ]
    )

    private static let enrolledKey = "training.enrolledCourseId"
    private static let enrolledAtKey = "training.enrolledAt"

    static var enrolledCourseID: String? {
        UserDefaults.standard.string(forKey: enrolledKey)
    }

    static var enrolledCourse: TrainingCourse? {
        guard let id = enrolledCourseID else { return nil }
        return all.first { $0.id == id }
    }

    /// 报名：写入未来日程，并记为当前课程
    @discardableResult
    static func enroll(_ course: TrainingCourse, startingFrom start: Date = Date()) -> Int {
        UserDefaults.standard.set(course.id, forKey: enrolledKey)
        UserDefaults.standard.set(start.timeIntervalSince1970, forKey: enrolledAtKey)

        let cal = Calendar.current
        var added = 0
        for session in course.sessions {
            guard let date = cal.date(byAdding: .day, value: session.dayOffset, to: cal.startOfDay(for: start)) else {
                continue
            }
            let item = TrainingScheduleItem.make(
                title: "【\(course.title)】\(session.title)",
                date: date,
                timeText: session.timeText,
                note: session.note
            )
            TrainingScheduleStore.upsert(item)
            added += 1
        }
        return added
    }

    static func clearEnrollment() {
        UserDefaults.standard.removeObject(forKey: enrolledKey)
        UserDefaults.standard.removeObject(forKey: enrolledAtKey)
    }
}
