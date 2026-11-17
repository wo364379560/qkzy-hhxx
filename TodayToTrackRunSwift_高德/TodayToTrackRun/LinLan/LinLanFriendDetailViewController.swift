//
//  LinLanFriendDetailViewController.swift
//  TodayToTrackRun
//
//  对照 Android LinLanFriendDetailScreen
//

import SnapKit
import UIKit

final class LinLanFriendDetailViewController: UIViewController {
    private let userId: String
    private var userName: String
    private var isFriend: Bool

    private let scroll = UIScrollView()
    private let stack = UIStackView()
    private let nameLabel = UILabel()
    private let idLabel = UILabel()
    private let letterLabel = UILabel()
    private let remarkValue = UILabel()
    private let tipLabel = UILabel()
    private let pinSwitch = UISwitch()
    private let muteSwitch = UISwitch()
    private let primaryButton = UIButton(type: .system)

    private var remark = ""
    private var nickName = ""

    init(userId: String, userName: String, isFriend: Bool) {
        self.userId = userId
        self.userName = userName
        self.isFriend = isFriend
        super.init(nibName: nil, bundle: nil)
    }

    convenience init(userName: String, isFriend: Bool) {
        self.init(userId: userName, userName: userName, isFriend: isFriend)
    }

    required init?(coder: NSCoder) { fatalError() }

    private var conversationID: String { "c2c_\(userId)" }
    private var isSelf: Bool {
        let me = LinLanImManager.loginUser
        return !me.isEmpty && me == userId
    }
    private var displayName: String {
        if !remark.isEmpty { return remark }
        if !nickName.isEmpty { return nickName }
        return userName
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "个人详情"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        view.addSubview(scroll)
        scroll.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        stack.axis = .vertical
        stack.spacing = 12
        stack.isLayoutMarginsRelativeArrangement = true
        stack.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 28, right: 16)
        scroll.addSubview(stack)
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }

        let card = AppTheme.makeCard()
        let avatar = UIView()
        avatar.backgroundColor = AppTheme.accent
        avatar.layer.cornerRadius = 36
        letterLabel.textColor = .white
        letterLabel.font = .systemFont(ofSize: 28, weight: .bold)
        letterLabel.textAlignment = .center
        nameLabel.font = .systemFont(ofSize: 22, weight: .bold)
        nameLabel.textColor = AppTheme.textPrimary
        idLabel.font = .systemFont(ofSize: 13)
        idLabel.textColor = AppTheme.textSecondary
        idLabel.text = "ID \(userId)"

        card.addSubview(avatar)
        avatar.addSubview(letterLabel)
        card.addSubview(nameLabel)
        card.addSubview(idLabel)
        avatar.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().inset(16)
            make.width.height.equalTo(72)
            make.bottom.equalToSuperview().inset(16)
        }
        letterLabel.snp.makeConstraints { $0.edges.equalToSuperview() }
        nameLabel.snp.makeConstraints { make in
            make.leading.equalTo(avatar.snp.trailing).offset(14)
            make.top.equalTo(avatar).offset(8)
            make.trailing.equalToSuperview().inset(16)
        }
        idLabel.snp.makeConstraints { make in
            make.leading.equalTo(nameLabel)
            make.top.equalTo(nameLabel.snp.bottom).offset(8)
        }
        stack.addArrangedSubview(card)

        let remarkCard = AppTheme.makeCard()
        let remarkBtn = UIButton(type: .system)
        remarkBtn.contentHorizontalAlignment = .left
        remarkBtn.contentEdgeInsets = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 16)
        remarkBtn.snp.makeConstraints { $0.height.equalTo(52) }
        let remarkTitle = UILabel()
        remarkTitle.text = "备注"
        remarkTitle.font = .systemFont(ofSize: 16, weight: .medium)
        remarkTitle.textColor = AppTheme.textPrimary
        remarkValue.font = .systemFont(ofSize: 15)
        remarkValue.textColor = AppTheme.accent
        remarkValue.textAlignment = .right
        remarkBtn.addSubview(remarkTitle)
        remarkBtn.addSubview(remarkValue)
        remarkTitle.snp.makeConstraints { make in
            make.leading.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        remarkValue.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
            make.leading.greaterThanOrEqualTo(remarkTitle.snp.trailing).offset(8)
        }
        remarkBtn.addTarget(self, action: #selector(linLanEditRemark), for: .touchUpInside)
        remarkCard.addSubview(remarkBtn)
        remarkBtn.snp.makeConstraints { $0.edges.equalToSuperview() }
        remarkCard.tag = 1001
        stack.addArrangedSubview(remarkCard)

        tipLabel.font = .systemFont(ofSize: 13)
        tipLabel.textColor = .systemRed
        tipLabel.numberOfLines = 0
        tipLabel.isHidden = true
        stack.addArrangedSubview(tipLabel)

        primaryButton.layer.cornerRadius = 10
        primaryButton.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        primaryButton.setTitleColor(.white, for: .normal)
        primaryButton.backgroundColor = AppTheme.accent
        primaryButton.snp.makeConstraints { $0.height.equalTo(48) }
        stack.addArrangedSubview(primaryButton)

        let switchCard = AppTheme.makeCard()
        switchCard.tag = 1002
        let switchStack = UIStackView()
        switchStack.axis = .vertical
        switchCard.addSubview(switchStack)
        switchStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        pinSwitch.onTintColor = AppTheme.accent
        muteSwitch.onTintColor = AppTheme.accent
        pinSwitch.addTarget(self, action: #selector(linLanPinChanged), for: .valueChanged)
        muteSwitch.addTarget(self, action: #selector(linLanMuteChanged), for: .valueChanged)
        switchStack.addArrangedSubview(switchRow(title: "置顶聊天", toggle: pinSwitch))
        switchStack.addArrangedSubview(divider())
        switchStack.addArrangedSubview(switchRow(title: "消息免打扰", toggle: muteSwitch))
        stack.addArrangedSubview(switchCard)

        let dangerCard = AppTheme.makeCard()
        dangerCard.tag = 1003
        let dangerStack = UIStackView()
        dangerStack.axis = .vertical
        dangerCard.addSubview(dangerStack)
        dangerStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        dangerStack.addArrangedSubview(actionRow(title: "拉黑", color: AppTheme.textPrimary, action: #selector(linLanBlack)))
        dangerStack.addArrangedSubview(divider())
        dangerStack.addArrangedSubview(actionRow(title: "删除好友", color: .systemRed, action: #selector(linLanDelete)))
        stack.addArrangedSubview(dangerCard)

        let reportCard = AppTheme.makeCard()
        reportCard.tag = 1004
        let reportStack = UIStackView()
        reportStack.axis = .vertical
        reportCard.addSubview(reportStack)
        reportStack.snp.makeConstraints { $0.edges.equalToSuperview() }
        reportStack.addArrangedSubview(actionRow(title: "举报", color: .systemRed, action: #selector(linLanReport)))
        stack.addArrangedSubview(reportCard)

        linLanApplyUI()
        Task { await linLanReload() }
    }

    private func linLanApplyUI() {
        nameLabel.text = nickName.isEmpty ? userName : nickName
        letterLabel.text = String(displayName.prefix(1))
        idLabel.text = isSelf ? "ID \(userId)（我）" : "ID \(userId)"
        remarkValue.text = remark.isEmpty ? "点击设置备注" : remark
        remarkValue.textColor = remark.isEmpty ? AppTheme.accent : AppTheme.textSecondary

        let remarkCard = stack.arrangedSubviews.first(where: { $0.tag == 1001 })
        let switchCard = stack.arrangedSubviews.first(where: { $0.tag == 1002 })
        let dangerCard = stack.arrangedSubviews.first(where: { $0.tag == 1003 })
        // 自己：不展示好友操作区
        remarkCard?.isHidden = !isFriend || isSelf
        switchCard?.isHidden = !isFriend || isSelf
        dangerCard?.isHidden = !isFriend || isSelf

        primaryButton.removeTarget(nil, action: nil, for: .allEvents)
        primaryButton.isEnabled = true
        primaryButton.alpha = 1
        if isSelf {
            primaryButton.setTitle("这是我自己", for: .normal)
            primaryButton.isEnabled = false
            primaryButton.alpha = 0.55
        } else if isFriend {
            primaryButton.setTitle("发消息", for: .normal)
            primaryButton.addTarget(self, action: #selector(linLanChat), for: .touchUpInside)
        } else {
            primaryButton.setTitle("添加好友", for: .normal)
            primaryButton.addTarget(self, action: #selector(linLanAddFriend), for: .touchUpInside)
        }
    }

    private func linLanReload() async {
        // 勿用 async let：工程默认 MainActor 隔离时会触发 swift_task_dealloc 崩溃
        let list = (try? await LinLanImManager.fetchFriendList()) ?? []
        let friend = list.contains { $0.userId == userId }
        let friendItem = list.first { $0.userId == userId }
        let profile = await LinLanImManager.fetchUserBrief(userId: userId)
        var pinned = false
        var muted = false
        if friend {
            pinned = await LinLanImManager.isConversationPinned(conversationID: conversationID)
            muted = await LinLanImManager.isC2CMuted(userId: userId)
        }

        isFriend = friend
        if let profile {
            nickName = profile.nickName
            if !profile.nickName.isEmpty { userName = profile.nickName }
        }
        if let friendItem {
            remark = friendItem.remark
            if !friendItem.nickName.isEmpty { nickName = friendItem.nickName }
        } else {
            remark = ""
        }
        pinSwitch.isOn = pinned
        muteSwitch.isOn = muted
        linLanApplyUI()
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
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalToSuperview()
        }
        return row
    }

    private func actionRow(title: String, color: UIColor, action: Selector) -> UIButton {
        let b = UIButton(type: .system)
        b.setTitle(title, for: .normal)
        b.setTitleColor(color, for: .normal)
        b.titleLabel?.font = .systemFont(ofSize: 16, weight: .medium)
        b.snp.makeConstraints { $0.height.equalTo(52) }
        b.addTarget(self, action: action, for: .touchUpInside)
        return b
    }

    private func divider() -> UIView {
        let w = UIView()
        w.snp.makeConstraints { $0.height.equalTo(1) }
        let line = UIView()
        line.backgroundColor = AppTheme.divider
        w.addSubview(line)
        line.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.trailing.equalToSuperview()
            make.centerY.equalToSuperview()
            make.height.equalTo(1 / UIScreen.main.scale)
        }
        return w
    }

    private func linLanTip(_ text: String?) {
        tipLabel.text = text
        tipLabel.isHidden = (text == nil || text?.isEmpty == true)
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func linLanChat() {
        linLanPush(
            LinLanChatViewController(
                titleText: displayName,
                userID: userId,
                groupID: nil
            )
        )
    }

    @objc private func linLanAddFriend() {
        if isSelf {
            linLanTip("不能添加自己为好友")
            return
        }
        Task {
            do {
                let outcome = try await LinLanImManager.addFriend(userId: userId)
                await MainActor.run {
                    self.linLanTip(outcome.toast)
                    if outcome == .addedDirectly || outcome == .alreadyFriends {
                        self.isFriend = true
                        self.linLanApplyUI()
                    }
                }
            } catch {
                await MainActor.run { self.linLanTip(error.localizedDescription) }
            }
        }
    }

    @objc private func linLanEditRemark() {
        let a = UIAlertController(title: "设置备注", message: nil, preferredStyle: .alert)
        a.addTextField {
            $0.text = self.remark
            $0.placeholder = "填写备注，方便识别"
        }
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "保存", style: .default) { [weak self] _ in
            guard let self else { return }
            let text = a.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            Task {
                do {
                    try await LinLanImManager.setFriendRemark(userId: self.userId, remark: text)
                    await MainActor.run {
                        self.remark = text
                        self.linLanApplyUI()
                        self.linLanTip("备注已更新")
                    }
                } catch {
                    await MainActor.run { self.linLanTip(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }

    @objc private func linLanPinChanged() {
        let on = pinSwitch.isOn
        Task {
            do {
                try await LinLanImManager.pinConversation(conversationID: conversationID, pinned: on)
                await MainActor.run { self.linLanTip(on ? "已置顶" : "已取消置顶") }
            } catch {
                await MainActor.run {
                    self.pinSwitch.isOn = !on
                    self.linLanTip(error.localizedDescription)
                }
            }
        }
    }

    @objc private func linLanMuteChanged() {
        let on = muteSwitch.isOn
        Task {
            do {
                try await LinLanImManager.setC2CMute(userId: userId, muted: on)
                await MainActor.run { self.linLanTip(on ? "已开启免打扰" : "已关闭免打扰") }
            } catch {
                await MainActor.run {
                    self.muteSwitch.isOn = !on
                    self.linLanTip(error.localizedDescription)
                }
            }
        }
    }

    @objc private func linLanReport() {
        LinLanReportViewController.push(
            from: self,
            targetId: userId,
            targetType: "user",
            targetName: displayName
        )
    }

    @objc private func linLanBlack() {
        let a = UIAlertController(
            title: "拉黑",
            message: "拉黑后将解除好友关系，且无法再收到对方消息。移出黑名单后需重新添加好友才能聊天。",
            preferredStyle: .alert
        )
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "拉黑", style: .destructive) { [weak self] _ in
            guard let self else { return }
            Task {
                do {
                    try await LinLanImManager.addToBlackList(userId: self.userId)
                    await MainActor.run {
                        self.linLanTip("已拉黑（好友关系已解除）")
                        self.navigationController?.popViewController(animated: true)
                    }
                } catch {
                    await MainActor.run { self.linLanTip(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }

    @objc private func linLanDelete() {
        let a = UIAlertController(
            title: "删除好友",
            message: "确定删除 \(displayName)？将同时删除聊天入口。",
            preferredStyle: .alert
        )
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "删除", style: .destructive) { [weak self] _ in
            guard let self else { return }
            Task {
                do {
                    try await LinLanImManager.deleteFriend(userId: self.userId)
                    try? await LinLanImManager.deleteConversation(conversationID: self.conversationID)
                    await MainActor.run {
                        self.navigationController?.popViewController(animated: true)
                    }
                } catch {
                    await MainActor.run { self.linLanTip(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }
}
