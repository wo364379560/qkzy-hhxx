//
//  LinLanGroupDetailViewController.swift
//  TodayToTrackRun
//
//  对照睿之禾 GroupDetailScreen
//

import ImSDK_Plus_Swift
import PhotosUI
import SnapKit
import UIKit

final class LinLanGroupDetailViewController: UIViewController {
    private let groupId: String
    private var groupName: String

    private var info: LinLanGroupInfoSnapshot?
    private var members: [LinLanGroupMemberItem] = []
    private var selfNameCard = ""
    private var tipLabel = UILabel()

    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let pinSwitch = UISwitch()
    private let muteSwitch = UISwitch()
    private let avatarView = UIImageView()

    private var conversationID: String { "group_\(groupId)" }
    private var selfRole: UInt { info?.role ?? 0 }
    private var canManage: Bool { LinLanImManager.isOwnerOrAdmin(role: selfRole) }
    private var isOwner: Bool { selfRole == LinLanImManager.roleOwner || selfRole == 400 }

    init(groupId: String, groupName: String) {
        self.groupId = groupId
        self.groupName = groupName
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "群聊信息"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "举报",
            style: .plain,
            target: self,
            action: #selector(reportGroup)
        )
        navigationItem.rightBarButtonItem?.tintColor = .systemRed

        pinSwitch.onTintColor = AppTheme.accent
        muteSwitch.onTintColor = AppTheme.accent
        pinSwitch.addTarget(self, action: #selector(pinChanged), for: .valueChanged)
        muteSwitch.addTarget(self, action: #selector(muteChanged), for: .valueChanged)

        tableView.dataSource = self
        tableView.delegate = self
        tableView.backgroundColor = AppTheme.pageBackground
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        tipLabel.font = .systemFont(ofSize: 13)
        tipLabel.textColor = .systemRed
        tipLabel.numberOfLines = 0
        tipLabel.textAlignment = .center

        Task { await reload() }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await reload() }
    }

    private func reload() async {
        do {
            info = try await LinLanImManager.fetchGroupInfo(groupId: groupId)
            if let n = info?.groupName, !n.isEmpty { groupName = n }
        } catch {
            showTip(error.localizedDescription)
        }
        members = (try? await LinLanImManager.fetchAllGroupMembers(groupId: groupId)) ?? []
        // 仅展示群昵称字段，勿用 displayName（会回落个人昵称造成「改了没变」）
        selfNameCard = members.first(where: { $0.userId == LinLanImManager.loginUser })?.nameCard ?? ""
        pinSwitch.isOn = await LinLanImManager.isConversationPinned(conversationID: conversationID)
        muteSwitch.isOn = await LinLanImManager.isGroupMuted(groupId: groupId)
        await MainActor.run { tableView.reloadData() }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func reportGroup() {
        LinLanReportViewController.push(
            from: self,
            targetId: groupId,
            targetType: "group",
            targetName: groupName
        )
    }

    private func showTip(_ text: String?) {
        tipLabel.text = text
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.1) { [weak a] in a?.dismiss(animated: true) }
    }

    // MARK: - Actions

    @objc private func pinChanged() {
        let on = pinSwitch.isOn
        Task {
            do {
                try await LinLanImManager.pinConversation(conversationID: conversationID, pinned: on)
            } catch {
                await MainActor.run {
                    self.pinSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func muteChanged() {
        let on = muteSwitch.isOn
        Task {
            do {
                try await LinLanImManager.setGroupMute(groupId: groupId, muted: on)
            } catch {
                await MainActor.run {
                    self.muteSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    private func inviteFriends() {
        Task {
            let friends = (try? await LinLanImManager.fetchFriendList()) ?? []
            let existing = Set(members.map(\.userId))
            let candidates = friends.filter { !existing.contains($0.userId) }
            await MainActor.run {
                guard !candidates.isEmpty else {
                    self.toast("没有可邀请的好友")
                    return
                }
                let sheet = UIAlertController(title: "邀请好友", message: "可多选", preferredStyle: .alert)
                var selected = Set<String>()
                // 用简化列表：前 20 个各自 toggle
                for f in candidates.prefix(20) {
                    sheet.addAction(UIAlertAction(title: "○ \(f.displayName)", style: .default) { [weak self] action in
                        guard let self else { return }
                        if selected.contains(f.userId) {
                            selected.remove(f.userId)
                        } else {
                            selected.insert(f.userId)
                        }
                        // 递归重新弹出不太友好；直接邀请当前点的人
                        Task {
                            do {
                                let opt = self.info?.groupApproveOpt ?? .V2TIM_GROUP_ADD_AUTH
                                try await LinLanImManager.inviteUsersToGroup(
                                    groupId: self.groupId,
                                    userIds: [f.userId]
                                )
                                await self.reload()
                                await MainActor.run {
                                    // FORBID/需要验证：对方需在「新朋友」同意；允许任何人则直接进群
                                    if opt == .V2TIM_GROUP_ADD_ANY {
                                        self.toast("已邀请 \(f.displayName)")
                                    } else {
                                        self.toast("已邀请 \(f.displayName)，对方需在「新朋友」中同意入群")
                                    }
                                }
                            } catch {
                                await MainActor.run { self.toast(error.localizedDescription) }
                            }
                        }
                        _ = action
                    })
                }
                sheet.addAction(UIAlertAction(title: "完成", style: .cancel))
                self.present(sheet, animated: true)
            }
        }
    }

    private func editGroupName() {
        guard canManage else { return }
        let a = UIAlertController(title: "群聊名称", message: nil, preferredStyle: .alert)
        a.addTextField { $0.text = self.groupName }
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "保存", style: .default) { [weak self] _ in
            guard let self else { return }
            let name = a.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !name.isEmpty else { return }
            Task {
                do {
                    try await LinLanImManager.setGroupName(groupId: self.groupId, name: name)
                    await self.reload()
                } catch {
                    await MainActor.run { self.toast(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }

    private func editNameCard() {
        let a = UIAlertController(title: "我的群昵称", message: nil, preferredStyle: .alert)
        a.addTextField { $0.text = self.selfNameCard }
        a.addAction(UIAlertAction(title: "取消", style: .cancel))
        a.addAction(UIAlertAction(title: "保存", style: .default) { [weak self] _ in
            guard let self else { return }
            let card = a.textFields?.first?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            Task {
                do {
                    try await LinLanImManager.setSelfGroupNameCard(groupId: self.groupId, nameCard: card)
                    // 清成员缓存，避免仍读到旧群昵称
                    LinLanGroupMemberCache.invalidate(groupId: self.groupId)
                    await MainActor.run {
                        self.selfNameCard = card
                        if let idx = self.members.firstIndex(where: { $0.userId == LinLanImManager.loginUser }) {
                            self.members[idx].nameCard = card
                            if !card.isEmpty {
                                self.members[idx].displayName = card
                            }
                        }
                        self.tableView.reloadData()
                        self.toast(card.isEmpty ? "已清除群昵称" : "群昵称已更新")
                    }
                    await self.reload()
                } catch {
                    await MainActor.run { self.toast(error.localizedDescription) }
                }
            }
        })
        present(a, animated: true)
    }

    private func cycleAddOpt() {
        guard canManage, let info else { return }
        let next = LinLanImManager.nextGroupAddOpt(info.groupAddOpt)
        Task {
            do {
                try await LinLanImManager.setGroupAddOpt(groupId: groupId, opt: next)
                await reload()
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func cycleApproveOpt() {
        guard canManage, let info else { return }
        let next = LinLanImManager.nextGroupAddOpt(info.groupApproveOpt)
        Task {
            do {
                try await LinLanImManager.setGroupApproveOpt(groupId: groupId, opt: next)
                await reload()
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func quitOrDismiss() {
        if isOwner {
            let a = UIAlertController(title: "解散群聊", message: "解散后群聊将不可恢复", preferredStyle: .alert)
            a.addAction(UIAlertAction(title: "取消", style: .cancel))
            a.addAction(UIAlertAction(title: "解散", style: .destructive) { [weak self] _ in
                guard let self else { return }
                Task {
                    do {
                        try await LinLanImManager.dismissGroup(groupId: self.groupId)
                        await MainActor.run {
                            self.navigationController?.popToRootViewController(animated: true)
                        }
                    } catch {
                        await MainActor.run { self.toast(error.localizedDescription) }
                    }
                }
            })
            present(a, animated: true)
        } else {
            let a = UIAlertController(title: "退出群聊", message: "确定退出该群？", preferredStyle: .alert)
            a.addAction(UIAlertAction(title: "取消", style: .cancel))
            a.addAction(UIAlertAction(title: "退出", style: .destructive) { [weak self] _ in
                guard let self else { return }
                Task {
                    do {
                        try await LinLanImManager.quitGroup(groupId: self.groupId)
                        await MainActor.run {
                            self.navigationController?.popToRootViewController(animated: true)
                        }
                    } catch {
                        await MainActor.run { self.toast(error.localizedDescription) }
                    }
                }
            })
            present(a, animated: true)
        }
    }

    private func pickGroupAvatar() {
        guard canManage else { return }
        var config = PHPickerConfiguration()
        config.filter = .images
        config.selectionLimit = 1
        let picker = PHPickerViewController(configuration: config)
        picker.delegate = self
        present(picker, animated: true)
    }
}

extension LinLanGroupDetailViewController: UITableViewDataSource, UITableViewDelegate {
    func numberOfSections(in tableView: UITableView) -> Int { 6 }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        switch section {
        case 0: return 1
        case 1: return 2 + min(members.count, 8) // 邀请 + 预览 + 查看全部
        case 2: return canManage ? 2 : 1 // 公告 + 管理
        case 3: return canManage ? 5 : 3 // 名称/昵称/ID/加群/邀请
        case 4: return 2
        case 5: return 1
        default: return 0
        }
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        switch section {
        case 1: return "群成员 (\(info?.memberCount ?? UInt(members.count)))"
        case 2: return "功能"
        case 3: return "设置"
        default: return nil
        }
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .value1, reuseIdentifier: nil)
        cell.backgroundColor = .white
        cell.selectionStyle = .default
        cell.accessoryType = .disclosureIndicator
        cell.accessoryView = nil
        cell.textLabel?.textColor = AppTheme.textPrimary
        cell.detailTextLabel?.textColor = AppTheme.textSecondary

        switch indexPath.section {
        case 0:
            cell.selectionStyle = canManage ? .default : .none
            cell.accessoryType = .none
            let wrap = UIView()
            avatarView.layer.cornerRadius = 28
            avatarView.clipsToBounds = true
            avatarView.contentMode = .scaleAspectFill
            avatarView.backgroundColor = AppTheme.accentSoft
            LinLanImageLoader.load(
                urlString: info?.faceUrl,
                into: avatarView,
                placeholderLetter: String(groupName.prefix(1))
            )
            let name = UILabel()
            name.text = groupName
            name.font = .systemFont(ofSize: 18, weight: .bold)
            let col = UIStackView(arrangedSubviews: [name])
            col.axis = .vertical
            wrap.addSubview(avatarView)
            wrap.addSubview(col)
            avatarView.snp.makeConstraints { make in
                make.leading.equalToSuperview().offset(16)
                make.centerY.equalToSuperview()
                make.size.equalTo(56)
            }
            col.snp.makeConstraints { make in
                make.leading.equalTo(avatarView.snp.trailing).offset(12)
                make.centerY.equalToSuperview()
                make.trailing.equalToSuperview().inset(16)
            }
            cell.contentView.subviews.forEach { $0.removeFromSuperview() }
            cell.contentView.addSubview(wrap)
            wrap.snp.makeConstraints { make in
                make.edges.equalToSuperview()
                make.height.equalTo(80)
            }
            cell.textLabel?.text = nil
            cell.detailTextLabel?.text = nil

        case 1:
            let previewCount = min(members.count, 8)
            if indexPath.row == 0 {
                cell.textLabel?.text = "邀请好友"
                cell.imageView?.image = UIImage(systemName: "plus.circle.fill")
                cell.imageView?.tintColor = AppTheme.accent
                cell.detailTextLabel?.text = nil
            } else if indexPath.row == previewCount + 1 {
                cell.textLabel?.text = "查看全部成员"
                cell.detailTextLabel?.text = "\(info?.memberCount ?? UInt(members.count))"
            } else {
                let m = members[indexPath.row - 1]
                cell.textLabel?.text = m.displayName
                var role = ""
                if m.isOwner { role = "群主" }
                else if m.isAdmin { role = "管理员" }
                cell.detailTextLabel?.text = role
            }

        case 2:
            if indexPath.row == 0 {
                cell.textLabel?.text = "群公告"
                let notice = info?.notification.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                cell.detailTextLabel?.text = notice.isEmpty ? "未设置" : notice
            } else {
                cell.textLabel?.text = "群管理"
                cell.detailTextLabel?.text = nil
            }

        case 3:
            switch indexPath.row {
            case 0:
                cell.textLabel?.text = "群聊名称"
                cell.detailTextLabel?.text = groupName
                if !canManage { cell.accessoryType = .none; cell.selectionStyle = .none }
            case 1:
                cell.textLabel?.text = "我的群昵称"
                cell.detailTextLabel?.text = selfNameCard.isEmpty ? "未设置" : selfNameCard
            case 2:
                cell.textLabel?.text = "群 ID"
                cell.detailTextLabel?.text = "复制"
            case 3:
                cell.textLabel?.text = "主动加群方式"
                cell.detailTextLabel?.text = LinLanImManager.groupAddOptLabel(info?.groupAddOpt ?? .V2TIM_GROUP_ADD_ANY)
            case 4:
                cell.textLabel?.text = "邀请进群方式"
                cell.detailTextLabel?.text = LinLanImManager.groupAddOptLabel(info?.groupApproveOpt ?? .V2TIM_GROUP_ADD_ANY)
            default: break
            }

        case 4:
            cell.selectionStyle = .none
            cell.accessoryType = .none
            if indexPath.row == 0 {
                cell.textLabel?.text = "置顶聊天"
                cell.accessoryView = pinSwitch
            } else {
                cell.textLabel?.text = "消息免打扰"
                cell.accessoryView = muteSwitch
            }

        case 5:
            cell.textLabel?.text = isOwner ? "解散群聊" : "退出群聊"
            cell.textLabel?.textColor = .systemRed
            cell.textLabel?.textAlignment = .center
            cell.accessoryType = .none
            cell.detailTextLabel?.text = nil

        default:
            break
        }
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch indexPath.section {
        case 0:
            pickGroupAvatar()
        case 1:
            if indexPath.row == 0 {
                inviteFriends()
            } else {
                linLanPush(LinLanGroupMembersViewController(groupId: groupId, selfRole: selfRole))
            }
        case 2:
            if indexPath.row == 0 {
                linLanPush(LinLanGroupNoticeViewController(groupId: groupId, canEdit: canManage))
            } else {
                linLanPush(LinLanGroupManageViewController(groupId: groupId))
            }
        case 3:
            switch indexPath.row {
            case 0: editGroupName()
            case 1: editNameCard()
            case 2:
                UIPasteboard.general.string = groupId
                toast("已复制群 ID")
            case 3: cycleAddOpt()
            case 4: cycleApproveOpt()
            default: break
            }
        case 5:
            quitOrDismiss()
        default:
            break
        }
    }

    func tableView(_ tableView: UITableView, titleForFooterInSection section: Int) -> String? {
        if section == 1 && members.count > 8 {
            return "点击成员区域查看全部"
        }
        return nil
    }
}

extension LinLanGroupDetailViewController: PHPickerViewControllerDelegate {
    func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
        picker.dismiss(animated: true)
        guard let provider = results.first?.itemProvider, provider.canLoadObject(ofClass: UIImage.self) else { return }
        provider.loadObject(ofClass: UIImage.self) { [weak self] obj, _ in
            guard let self, let image = obj as? UIImage,
                  let data = image.jpegData(compressionQuality: 0.85) else { return }
            let url = FileManager.default.temporaryDirectory.appendingPathComponent("gavatar_\(UUID().uuidString).jpg")
            try? data.write(to: url)
            Task {
                do {
                    let avatar = try await LinLanAppContainer.apiClient().uploadGroupAvatar(
                        groupId: self.groupId,
                        filePath: url.path
                    )
                    try await LinLanImManager.setGroupFaceUrl(groupId: self.groupId, faceUrl: avatar)
                    await self.reload()
                    await MainActor.run { self.toast("群头像已更新") }
                } catch {
                    await MainActor.run { self.toast(error.localizedDescription) }
                }
            }
        }
    }
}
