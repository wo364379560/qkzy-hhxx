//
//  LinLanConversationViewController.swift
//  TodayToTrackRun
//

import ImSDK_Plus_Swift
import SnapKit
import UIKit

final class LinLanConversationViewController: UIViewController {
    private let tableView = UITableView(frame: .zero, style: .plain)
    private let emptyLabel = UILabel()
    private var conversations: [V2TIMConversation] = []
    private var reloadWorkItem: DispatchWorkItem?

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        tableView.backgroundColor = AppTheme.pageBackground
        tableView.separatorInset = UIEdgeInsets(top: 0, left: 76, bottom: 0, right: 0)
        tableView.rowHeight = 72
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(LinLanConversationCell.self, forCellReuseIdentifier: LinLanConversationCell.reuseID)
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalToSuperview() }

        emptyLabel.text = "暂无会话\n点击右上角 + 添加好友开始聊天"
        emptyLabel.textAlignment = .center
        emptyLabel.numberOfLines = 0
        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.font = .systemFont(ofSize: 15)
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalToSuperview() }
        linLanApplyEmpty()

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRealtimeNeedRefresh),
            name: .linLanDidRecvMessage,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRealtimeNeedRefresh),
            name: .linLanDidRevokeMessage,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRealtimeNeedRefresh),
            name: .linLanGroupMembershipChanged,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRealtimeNeedRefresh),
            name: .linLanCurrentUserRemovedFromGroup,
            object: nil
        )
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await linLanReload() }
    }

    /// 节流：避免每条群消息都全量拉会话打满主线程
    @objc private func onRealtimeNeedRefresh() {
        reloadWorkItem?.cancel()
        let item = DispatchWorkItem { [weak self] in
            Task { await self?.linLanReload() }
        }
        reloadWorkItem = item
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.45, execute: item)
    }

    private func linLanReload() async {
        _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
        do {
            let list = try await LinLanImManager.fetchConversations()
            let unread = list.reduce(0) { $0 + Int($1.unreadCount) }
            await MainActor.run {
                self.conversations = list
                self.linLanApplyEmpty()
                self.updateTabBadge(unread)
            }
        } catch {
            await MainActor.run { self.linLanApplyEmpty() }
        }
    }

    private func updateTabBadge(_ unread: Int) {
        guard let items = tabBarController?.tabBar.items, items.count > 2 else { return }
        let item = items[2] // 消息 Tab
        item.badgeValue = unread > 0 ? (unread > 99 ? "99+" : "\(unread)") : nil
    }

    private func linLanApplyEmpty() {
        emptyLabel.isHidden = !conversations.isEmpty
        tableView.isHidden = conversations.isEmpty
        tableView.reloadData()
    }

    private static func isMuted(_ conv: V2TIMConversation) -> Bool {
        conv.recvOpt == .V2TIM_RECEIVE_NOT_NOTIFY_MESSAGE
    }

    private func conversationId(of conv: V2TIMConversation) -> String {
        let raw = conv.conversationID.trimmingCharacters(in: .whitespacesAndNewlines)
        if !raw.isEmpty { return raw }
        return LinLanImManager.conversationId(userID: conv.userID, groupID: conv.groupID)
    }

    private func displayName(of conv: V2TIMConversation) -> String {
        if let name = conv.showName, !name.isEmpty { return name }
        return conv.userID ?? conv.groupID ?? "会话"
    }

    private func togglePin(_ conv: V2TIMConversation) {
        let cid = conversationId(of: conv)
        guard !cid.isEmpty else { return }
        let next = !conv.isPinned
        Task {
            do {
                try await LinLanImManager.pinConversation(conversationID: cid, pinned: next)
                await linLanReload()
                await MainActor.run { self.toast(next ? "已置顶" : "已取消置顶") }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func confirmDelete(_ conv: V2TIMConversation) {
        let cid = conversationId(of: conv)
        guard !cid.isEmpty else { return }
        let name = displayName(of: conv)
        let alert = UIAlertController(
            title: "删除会话",
            message: "确定删除「\(name)」？删除后该会话会从消息列表消失。",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "删除", style: .destructive) { [weak self] _ in
            Task { await self?.deleteConversation(cid) }
        })
        present(alert, animated: true)
    }

    private func deleteConversation(_ conversationID: String) async {
        do {
            try await LinLanImManager.deleteConversation(conversationID: conversationID)
            await linLanReload()
            await MainActor.run { self.toast("已删除") }
        } catch {
            await MainActor.run { self.toast(error.localizedDescription) }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak a] in
            a?.dismiss(animated: true)
        }
    }
}

extension LinLanConversationViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        conversations.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(
            withIdentifier: LinLanConversationCell.reuseID,
            for: indexPath
        ) as! LinLanConversationCell
        let c = conversations[indexPath.row]
        let title = (c.showName?.isEmpty == false ? c.showName! : (c.userID ?? c.groupID ?? "会话"))
        let preview = LinLanImManager.messagePreview(c.lastMessage)
        let time = LinLanImManager.formatTime(c.lastMessage?.timestamp)
        cell.linLanConfigure(
            title: title,
            preview: preview,
            time: time,
            unread: Int(c.unreadCount),
            pinned: c.isPinned,
            muted: Self.isMuted(c)
        )
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let c = conversations[indexPath.row]
        let title = (c.showName?.isEmpty == false ? c.showName! : (c.userID ?? c.groupID ?? "会话"))
        linLanPush(
            LinLanChatViewController(
                titleText: title,
                userID: c.userID,
                groupID: c.groupID
            )
        )
    }

    func tableView(
        _ tableView: UITableView,
        trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath
    ) -> UISwipeActionsConfiguration? {
        guard conversations.indices.contains(indexPath.row) else { return nil }
        let conv = conversations[indexPath.row]
        let pin = UIContextualAction(
            style: .normal,
            title: conv.isPinned ? "取消置顶" : "置顶"
        ) { [weak self] _, _, done in
            self?.togglePin(conv)
            done(true)
        }
        pin.backgroundColor = AppTheme.accent
        let delete = UIContextualAction(style: .destructive, title: "删除") { [weak self] _, _, done in
            self?.confirmDelete(conv)
            done(true)
        }
        let config = UISwipeActionsConfiguration(actions: [delete, pin])
        config.performsFirstActionWithFullSwipe = false
        return config
    }

    func tableView(
        _ tableView: UITableView,
        contextMenuConfigurationForRowAt indexPath: IndexPath,
        point: CGPoint
    ) -> UIContextMenuConfiguration? {
        guard conversations.indices.contains(indexPath.row) else { return nil }
        let conv = conversations[indexPath.row]
        return UIContextMenuConfiguration(identifier: nil, previewProvider: nil) { [weak self] _ in
            let pin = UIAction(
                title: conv.isPinned ? "取消置顶" : "置顶",
                image: UIImage(systemName: conv.isPinned ? "pin.slash" : "pin")
            ) { _ in
                self?.togglePin(conv)
            }
            let delete = UIAction(
                title: "删除",
                image: UIImage(systemName: "trash"),
                attributes: .destructive
            ) { _ in
                self?.confirmDelete(conv)
            }
            return UIMenu(title: "", children: [pin, delete])
        }
    }
}

final class LinLanConversationCell: UITableViewCell {
    static let reuseID = "LinLanConversationCell"
    private let avatar = UIView()
    private let letter = UILabel()
    private let titleLabel = UILabel()
    private let previewLabel = UILabel()
    private let timeLabel = UILabel()
    private let badge = UILabel()
    private let pinIcon = UIImageView(image: UIImage(systemName: "pin.fill"))
    private let muteIcon = UIImageView(image: UIImage(systemName: "bell.slash.fill"))

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        backgroundColor = .clear
        contentView.backgroundColor = .clear
        avatar.backgroundColor = AppTheme.accent
        avatar.layer.cornerRadius = 24
        letter.textColor = .white
        letter.font = .systemFont(ofSize: 18, weight: .bold)
        letter.textAlignment = .center
        titleLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        previewLabel.font = .systemFont(ofSize: 13)
        previewLabel.textColor = AppTheme.textSecondary
        timeLabel.font = .systemFont(ofSize: 12)
        timeLabel.textColor = AppTheme.textSecondary
        badge.font = .systemFont(ofSize: 11, weight: .bold)
        badge.textColor = .white
        badge.backgroundColor = AppTheme.accent
        badge.textAlignment = .center
        badge.layer.cornerRadius = 9
        badge.clipsToBounds = true
        badge.isHidden = true
        pinIcon.tintColor = AppTheme.accent
        pinIcon.contentMode = .scaleAspectFit
        pinIcon.isHidden = true
        muteIcon.tintColor = AppTheme.textSecondary
        muteIcon.contentMode = .scaleAspectFit
        muteIcon.isHidden = true

        contentView.addSubview(avatar)
        avatar.addSubview(letter)
        contentView.addSubview(titleLabel)
        contentView.addSubview(previewLabel)
        contentView.addSubview(timeLabel)
        contentView.addSubview(badge)
        contentView.addSubview(pinIcon)
        contentView.addSubview(muteIcon)
        avatar.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.width.height.equalTo(48)
        }
        letter.snp.makeConstraints { $0.edges.equalToSuperview() }
        timeLabel.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.top.equalToSuperview().offset(14)
        }
        muteIcon.snp.makeConstraints { make in
            make.trailing.equalTo(timeLabel.snp.leading).offset(-6)
            make.centerY.equalTo(timeLabel)
            make.size.equalTo(14)
        }
        pinIcon.snp.makeConstraints { make in
            make.trailing.equalTo(muteIcon.snp.leading).offset(-4)
            make.centerY.equalTo(timeLabel)
            make.size.equalTo(12)
        }
        titleLabel.snp.makeConstraints { make in
            make.leading.equalTo(avatar.snp.trailing).offset(12)
            make.top.equalToSuperview().offset(14)
            make.trailing.lessThanOrEqualTo(pinIcon.snp.leading).offset(-8)
        }
        previewLabel.snp.makeConstraints { make in
            make.leading.equalTo(titleLabel)
            make.top.equalTo(titleLabel.snp.bottom).offset(4)
            make.trailing.equalToSuperview().inset(40)
        }
        badge.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.centerY.equalTo(previewLabel)
            make.height.equalTo(18)
            make.width.greaterThanOrEqualTo(18)
        }
    }

    required init?(coder: NSCoder) { fatalError() }

    func linLanConfigure(
        title: String,
        preview: String,
        time: String,
        unread: Int,
        pinned: Bool,
        muted: Bool
    ) {
        titleLabel.text = title
        previewLabel.text = preview
        timeLabel.text = time
        letter.text = String(title.prefix(1))
        pinIcon.isHidden = !pinned
        muteIcon.isHidden = !muted
        contentView.backgroundColor = pinned
            ? UIColor(red: 1, green: 0.97, blue: 0.91, alpha: 1)
            : .white
        if unread > 0 {
            badge.isHidden = false
            badge.text = unread > 99 ? " 99+ " : " \(unread) "
            // 免打扰用灰色角标，普通未读用主题色
            badge.backgroundColor = muted ? AppTheme.textSecondary : AppTheme.accent
        } else {
            badge.isHidden = true
        }
    }
}
