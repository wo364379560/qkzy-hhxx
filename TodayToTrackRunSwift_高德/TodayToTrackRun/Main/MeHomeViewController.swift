//
//  MeHomeViewController.swift
//  TodayToTrackRun
//
//  LinLan「我的」：资料头图 + 账号设置 + 协议 + 运动设置
//

import SnapKit
import UIKit

final class MeHomeViewController: UIViewController {
    private let scrollView = UIScrollView()
    private let content = UIStackView()

    private let avatarView = UIView()
    private let avatarLetter = UILabel()
    private let nameLabel = UILabel()
    private let subtitleLabel = UILabel()
    private let editHint = UILabel()

    private let badgeStack = UIStackView()
    private let kmValue = UILabel()
    private let countValue = UILabel()
    private let kcalValue = UILabel()

    private let voiceSwitch = UISwitch()
    private let autoPauseSwitch = UISwitch()

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "我的"
        view.backgroundColor = AppTheme.pageBackground
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

        // Profile header card
        let headerCard = AppTheme.makeCard()
        let header = UIStackView()
        header.axis = .horizontal
        header.spacing = 14
        header.alignment = .center
        headerCard.addSubview(header)
        header.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }

        avatarView.backgroundColor = AppTheme.accent
        avatarView.layer.cornerRadius = 32
        avatarView.snp.makeConstraints { $0.width.height.equalTo(64) }
        avatarLetter.text = "本"
        avatarLetter.font = .systemFont(ofSize: 26, weight: .bold)
        avatarLetter.textColor = .white
        avatarLetter.textAlignment = .center
        avatarView.addSubview(avatarLetter)
        avatarLetter.snp.makeConstraints { $0.edges.equalToSuperview() }

        let nameCol = UIStackView()
        nameCol.axis = .vertical
        nameCol.spacing = 4
        nameLabel.font = .systemFont(ofSize: 22, weight: .bold)
        nameLabel.textColor = AppTheme.textPrimary
        subtitleLabel.font = .systemFont(ofSize: 13)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.numberOfLines = 2
        nameCol.addArrangedSubview(nameLabel)
        nameCol.addArrangedSubview(subtitleLabel)

        editHint.text = "编辑资料"
        editHint.font = .systemFont(ofSize: 14, weight: .semibold)
        editHint.textColor = AppTheme.accent

        header.addArrangedSubview(avatarView)
        header.addArrangedSubview(nameCol)
        header.addArrangedSubview(editHint)
        headerCard.isUserInteractionEnabled = true
        headerCard.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(linLanOpenProfile)))
        content.addArrangedSubview(headerCard)

        // 账号与设置
        content.addArrangedSubview(sectionLabel("账号与设置"))
        let accountCard = AppTheme.makeCard()
        let accountStack = UIStackView()
        accountStack.axis = .vertical
        accountCard.addSubview(accountStack)
        accountStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        accountStack.addArrangedSubview(navRow(title: "个人设置", action: #selector(linLanOpenProfile)))
        accountStack.addArrangedSubview(divider())
        accountStack.addArrangedSubview(navRow(title: "消息通知", action: #selector(linLanOpenNotify)))
        accountStack.addArrangedSubview(divider())
        accountStack.addArrangedSubview(navRow(title: "安全与隐私", action: #selector(linLanOpenSecurity)))
        accountStack.addArrangedSubview(divider())
        accountStack.addArrangedSubview(navRow(title: "隐私与合规", action: #selector(linLanOpenCompliance)))
        content.addArrangedSubview(accountCard)

        // 运动数据
        content.addArrangedSubview(sectionLabel("成就徽章"))
        let badgeCard = AppTheme.makeCard()
        badgeStack.axis = .vertical
        badgeStack.spacing = 12
        badgeCard.addSubview(badgeStack)
        badgeStack.snp.makeConstraints { $0.edges.equalToSuperview().inset(14) }
        content.addArrangedSubview(badgeCard)

        content.addArrangedSubview(sectionLabel("累计数据"))
        let statsCard = AppTheme.makeCard()
        let statsRow = UIStackView(arrangedSubviews: [
            statColumn(valueLabel: kmValue, title: "总公里", accent: true),
            statColumn(valueLabel: countValue, title: "总次数", accent: false),
            statColumn(valueLabel: kcalValue, title: "卡路里", accent: false)
        ])
        statsRow.axis = .horizontal
        statsRow.distribution = .fillEqually
        statsCard.addSubview(statsRow)
        statsRow.snp.makeConstraints { $0.edges.equalToSuperview().inset(16) }
        content.addArrangedSubview(statsCard)

        // 运动设置
        content.addArrangedSubview(sectionLabel("运动设置"))
        let settingsCard = AppTheme.makeCard()
        let settingsStack = UIStackView()
        settingsStack.axis = .vertical
        settingsCard.addSubview(settingsStack)
        settingsStack.snp.makeConstraints { $0.edges.equalToSuperview() }

        voiceSwitch.onTintColor = AppTheme.accent
        voiceSwitch.isOn = RunSettings.voiceBroadcastEnabled
        voiceSwitch.addTarget(self, action: #selector(voiceChanged), for: .valueChanged)
        autoPauseSwitch.onTintColor = AppTheme.accent
        autoPauseSwitch.isOn = RunSettings.autoPauseEnabled
        autoPauseSwitch.addTarget(self, action: #selector(autoPauseChanged), for: .valueChanged)

        settingsStack.addArrangedSubview(switchRow(title: "语音播报（每公里女声）", toggle: voiceSwitch))
        settingsStack.addArrangedSubview(divider())
        settingsStack.addArrangedSubview(switchRow(title: "自动暂停（停住约 8 秒）", toggle: autoPauseSwitch))
        settingsStack.addArrangedSubview(divider())
        settingsStack.addArrangedSubview(navRow(title: "跑步记录", action: #selector(openHistory)))
        settingsStack.addArrangedSubview(divider())
        settingsStack.addArrangedSubview(navRow(title: "饮食偏好", action: #selector(openDiet)))
        settingsStack.addArrangedSubview(divider())
        settingsStack.addArrangedSubview(navRow(title: "天气城市", action: #selector(openCity)))
        settingsStack.addArrangedSubview(divider())
      //  settingsStack.addArrangedSubview(navRow(title: "同步状态", action: #selector(openSync)))
     //   settingsStack.addArrangedSubview(divider())
      //  settingsStack.addArrangedSubview(navRow(title: "关于", action: #selector(openAbout)))
        content.addArrangedSubview(settingsCard)

        // 退出
        let logout = UIButton(type: .system)
        logout.setTitle("退出账号", for: .normal)
        logout.setTitleColor(.systemRed, for: .normal)
        logout.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        logout.backgroundColor = .white
        logout.layer.cornerRadius = 16
        logout.snp.makeConstraints { $0.height.equalTo(52) }
        logout.addTarget(self, action: #selector(linLanLogout), for: .touchUpInside)
        content.addArrangedSubview(logout)

        #if DEBUG
//        let debug = UIButton(type: .system)
//        debug.setTitle("插入演示数据（室内调试）", for: .normal)
//        debug.setTitleColor(AppTheme.accent, for: .normal)
//        debug.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
//        debug.addTarget(self, action: #selector(seedDemo), for: .touchUpInside)
//        content.addArrangedSubview(debug)
        #endif
    }

    private func reload() {
        let session = LinLanAppContainer.sessionStore().currentSession()
        let prefs = LinLanAppContainer.appPrefs()
        if session.isLoggedIn {
            let nick = prefs.cachedNickname.isEmpty ? "LinLan 用户" : prefs.cachedNickname
            nameLabel.text = nick
            subtitleLabel.text = prefs.accountPhone.isEmpty ? "手机号 未获取" : "手机号 \(prefs.accountPhone)"
            avatarLetter.text = String(nick.prefix(1))
        } else {
            nameLabel.text = "本地跑者"
            subtitleLabel.text = "未登录 · 点击完善个人资料"
            avatarLetter.text = "本"
        }

        let summaries = (try? RunRecordStore.listSummaries(limit: 500)) ?? []
        let badges = RunAchievementStore.achievements(from: summaries)
        badgeStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        for row in stride(from: 0, to: badges.count, by: 3) {
            let line = UIStackView()
            line.axis = .horizontal
            line.distribution = .fillEqually
            line.spacing = 8
            for item in badges[row..<min(row + 3, badges.count)] {
                line.addArrangedSubview(badgeCell(item))
            }
            while line.arrangedSubviews.count < 3 {
                line.addArrangedSubview(UIView())
            }
            badgeStack.addArrangedSubview(line)
        }

        let cum = RunAchievementStore.cumulative(from: summaries)
        kmValue.text = cum.km >= 100 ? String(format: "%.0f", cum.km) : String(format: "%.1f", cum.km)
        countValue.text = "\(cum.count)"
        if cum.kcal >= 1000 {
            kcalValue.text = String(format: "%.1fk", Double(cum.kcal) / 1000.0)
        } else {
            kcalValue.text = "\(cum.kcal)"
        }
    }

    private func badgeCell(_ item: RunAchievement) -> UIView {
        let box = UIView()
        let circle = UIView()
        circle.layer.cornerRadius = 28
        circle.backgroundColor = item.unlocked ? AppTheme.accentSoft : UIColor(white: 0.94, alpha: 1)
        let icon = UIImageView(image: UIImage(systemName: item.symbolName))
        icon.tintColor = item.unlocked ? AppTheme.accent : AppTheme.textSecondary
        icon.contentMode = .scaleAspectFit
        let title = UILabel()
        title.text = item.title
        title.font = .systemFont(ofSize: 11, weight: .medium)
        title.textColor = item.unlocked ? AppTheme.textPrimary : AppTheme.textSecondary
        title.textAlignment = .center
        title.numberOfLines = 2
        box.addSubview(circle)
        circle.addSubview(icon)
        box.addSubview(title)
        circle.snp.makeConstraints { make in
            make.top.centerX.equalToSuperview()
            make.width.height.equalTo(56)
        }
        icon.snp.makeConstraints { $0.center.equalToSuperview(); $0.width.height.equalTo(24) }
        title.snp.makeConstraints { make in
            make.top.equalTo(circle.snp.bottom).offset(6)
            make.leading.trailing.bottom.equalToSuperview()
        }
        return box
    }

    private func sectionLabel(_ text: String) -> UILabel {
        let l = UILabel()
        l.text = text
        l.font = .systemFont(ofSize: 13, weight: .medium)
        l.textColor = AppTheme.textSecondary
        return l
    }

    private func statColumn(valueLabel: UILabel, title: String, accent: Bool) -> UIView {
        let box = UIView()
        valueLabel.font = AppTheme.mono(28, weight: .bold)
        valueLabel.textColor = accent ? AppTheme.accent : AppTheme.textPrimary
        valueLabel.textAlignment = .center
        let t = UILabel()
        t.text = title
        t.font = .systemFont(ofSize: 12)
        t.textColor = AppTheme.textSecondary
        t.textAlignment = .center
        box.addSubview(valueLabel)
        box.addSubview(t)
        valueLabel.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
        }
        t.snp.makeConstraints { make in
            make.top.equalTo(valueLabel.snp.bottom).offset(4)
            make.leading.trailing.bottom.equalToSuperview()
        }
        return box
    }

    private func switchRow(title: String, toggle: UISwitch) -> UIView {
        let row = UIView()
        row.snp.makeConstraints { $0.height.equalTo(52) }
        let label = UILabel()
        label.text = title
        label.font = .systemFont(ofSize: 16, weight: .medium)
        label.textColor = AppTheme.textPrimary
        row.addSubview(label)
        row.addSubview(toggle)
        label.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
        }
        toggle.snp.makeConstraints { make in
            make.trailing.equalToSuperview().offset(-16)
            make.centerY.equalToSuperview()
        }
        return row
    }

    private func navRow(title: String, action: Selector) -> UIView {
        let button = UIButton(type: .system)
        button.contentHorizontalAlignment = .left
        button.contentEdgeInsets = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 16)
        button.setTitle(title, for: .normal)
        button.setTitleColor(AppTheme.textPrimary, for: .normal)
        button.titleLabel?.font = .systemFont(ofSize: 16, weight: .medium)
        button.snp.makeConstraints { $0.height.equalTo(52) }
        button.addTarget(self, action: action, for: .touchUpInside)
        let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))
        chevron.tintColor = UIColor(white: 0.75, alpha: 1)
        button.addSubview(chevron)
        chevron.snp.makeConstraints { make in
            make.trailing.equalToSuperview().offset(-16)
            make.centerY.equalToSuperview()
        }
        return button
    }

    private func divider() -> UIView {
        let wrap = UIView()
        wrap.snp.makeConstraints { $0.height.equalTo(1) }
        let line = UIView()
        line.backgroundColor = AppTheme.divider
        wrap.addSubview(line)
        line.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.trailing.equalToSuperview()
            make.centerY.equalToSuperview()
            make.height.equalTo(1 / UIScreen.main.scale)
        }
        return wrap
    }

    @objc private func voiceChanged() {
        RunSettings.voiceBroadcastEnabled = voiceSwitch.isOn
    }

    @objc private func autoPauseChanged() {
        RunSettings.autoPauseEnabled = autoPauseSwitch.isOn
    }

    @objc private func openHistory() {
        linLanPush(RunHistoryViewController())
    }

    @objc private func openDiet() {
        let sheet = UIAlertController(title: "饮食偏好", message: nil, preferredStyle: .actionSheet)
        for option in GoalPreferences.DietPreference.allCases {
            sheet.addAction(UIAlertAction(title: option.title, style: .default) { [weak self] _ in
                GoalPreferences.setDietPreference(option)
                self?.reload()
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    @objc private func openCity() {
        let alert = UIAlertController(
            title: "天气城市",
            message: "默认会定位当前城市。这里可手动覆盖；留空并保存后下次仍走定位。",
            preferredStyle: .alert
        )
        alert.addTextField { $0.text = RunSettings.weatherCity; $0.placeholder = "例如：杭州" }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "保存", style: .default) { _ in
            let city = alert.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if city.isEmpty {
                RunSettings.weatherCity = "杭州"
            } else {
                RunSettings.weatherCity = city.hasSuffix("市") ? String(city.dropLast()) : city
            }
        })
        present(alert, animated: true)
    }

    @objc private func openSync() {
        let pending = (try? RunRecordStore.unsyncedCount()) ?? 0
        let message = RunAPIConfig.isConfigured
            ? "已配置 baseURL，待同步 \(pending) 条。"
            : "一期本地优先。接口未配置 · 待同步 \(pending) 条。"
        let alert = UIAlertController(title: "同步状态", message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "好", style: .default))
        present(alert, animated: true)
    }

    @objc private func openAbout() {
        let alert = UIAlertController(
            title: "关于 LinLan",
            message: "运动 + 即时通讯\n米白 + 珊瑚橙主题",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "好", style: .default))
        present(alert, animated: true)
    }

    @objc private func linLanOpenProfile() {
        linLanPush(LinLanEditProfileViewController())
    }

    @objc private func linLanOpenNotify() {
        linLanPush(LinLanMessageNotifyViewController())
    }

    @objc private func linLanOpenSecurity() {
        linLanPush(LinLanSecurityPrivacyViewController())
    }

    @objc private func linLanOpenCompliance() {
        linLanPush(LinLanPrivacyComplianceViewController())
    }

    @objc private func linLanLogout() {
        let a = UIAlertController(title: "退出账号", message: "确认退出当前账号？", preferredStyle: .alert)
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "退出", style: .destructive) { _ in
            Task {
                await LinLanAppContainer.authRepository().logout()
                let prefs = LinLanAppContainer.appPrefs()
                prefs.cachedNickname = ""
                prefs.cachedSignalId = ""
                await MainActor.run { LinLanRootSwitcher.showLogin() }
            }
        })
        present(a, animated: true)
    }

    #if DEBUG
    @objc private func seedDemo() {
        do {
            let ids = try MockRunGenerator.seedDemoRecords(count: 3)
            let alert = UIAlertController(title: "已插入 \(ids.count) 条", message: nil, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
            reload()
        } catch {
            let alert = UIAlertController(title: "失败", message: error.localizedDescription, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
        }
    }
    #endif
}
