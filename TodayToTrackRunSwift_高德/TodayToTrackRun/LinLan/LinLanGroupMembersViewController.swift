//
//  LinLanGroupMembersViewController.swift
//  对照睿之禾 GroupMembersScreen
//

import ImSDK_Plus_Swift
import SnapKit
import UIKit

final class LinLanGroupMembersViewController: UIViewController {
    private let groupId: String
    private var selfRole: UInt = 0
    private var members: [LinLanGroupMemberItem] = []
    private var filtered: [LinLanGroupMemberItem] = []
    /// true=允许查看成员资料；false=禁止私聊
    private var profileAccessEnabled = true

    private let search = UISearchBar()
    private let tableView = UITableView(frame: .zero, style: .plain)

    init(groupId: String, selfRole: UInt) {
        self.groupId = groupId
        self.selfRole = selfRole
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    private var canManage: Bool { LinLanImManager.isOwnerOrAdmin(role: selfRole) }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "群成员"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        search.placeholder = "搜索成员"
        search.delegate = self
        search.searchBarStyle = .minimal
        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = 60
        tableView.backgroundColor = .white
        view.addSubview(search)
        view.addSubview(tableView)
        search.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide)
            make.leading.trailing.equalToSuperview()
        }
        tableView.snp.makeConstraints { make in
            make.top.equalTo(search.snp.bottom)
            make.leading.trailing.bottom.equalToSuperview()
        }
        Task { await reload() }
    }

    private func reload() async {
        async let access = LinLanGroupPolicy.fetchMemberProfileAccessEnabled(groupId: groupId)
        async let list = LinLanImManager.fetchAllGroupMembers(groupId: groupId)
        profileAccessEnabled = await access
        members = (try? await list) ?? []
        applyFilter()
    }

    private func applyFilter() {
        let q = (search.text ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        filtered = q.isEmpty
            ? members
            : members.filter {
                $0.displayName.lowercased().contains(q) || $0.userId.lowercased().contains(q)
            }
        tableView.reloadData()
        title = "群成员 (\(members.count))"
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }

    private func showActions(for member: LinLanGroupMemberItem) {
        guard canManage, member.userId != LinLanImManager.loginUser else {
            openProfile(member)
            return
        }
        // 管理员不能管理群主/其他管理员（群主除外）
        if selfRole != LinLanImManager.roleOwner && selfRole != 400 {
            if member.isOwner || member.isAdmin {
                openProfile(member)
                return
            }
        }

        let sheet = UIAlertController(title: member.displayName, message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "查看资料", style: .default) { [weak self] _ in
            self?.openProfile(member)
        })
        if member.isMuted {
            sheet.addAction(UIAlertAction(title: "解除禁言", style: .default) { [weak self] _ in
                self?.mute(member, seconds: 0)
            })
        } else {
            sheet.addAction(UIAlertAction(title: "禁言 30 天", style: .default) { [weak self] _ in
                self?.mute(member, seconds: 30 * 24 * 3600)
            })
        }
        if selfRole == LinLanImManager.roleOwner || selfRole == 400 {
            if member.isAdmin {
                sheet.addAction(UIAlertAction(title: "取消管理员", style: .default) { [weak self] _ in
                    self?.setRole(member, role: .V2TIM_GROUP_MEMBER_ROLE_MEMBER)
                })
            } else if !member.isOwner {
                sheet.addAction(UIAlertAction(title: "设为管理员", style: .default) { [weak self] _ in
                    self?.setRole(member, role: .V2TIM_GROUP_MEMBER_ROLE_ADMIN)
                })
            }
        }
        if !member.isOwner {
            sheet.addAction(UIAlertAction(title: "移出群聊", style: .destructive) { [weak self] _ in
                self?.kick(member)
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func openProfile(_ member: LinLanGroupMemberItem) {
        guard LinLanImManager.canViewMemberProfile(
            profileAccessEnabled: profileAccessEnabled,
            selfRole: selfRole,
            targetRole: member.role,
            targetUserId: member.userId
        ) else {
            toast("群已禁止查看成员资料")
            return
        }
        linLanPush(
            LinLanFriendDetailViewController(
                userId: member.userId,
                userName: member.displayName,
                isFriend: false
            )
        )
    }

    private func mute(_ member: LinLanGroupMemberItem, seconds: UInt) {
        Task {
            do {
                try await LinLanImManager.muteGroupMember(
                    groupId: groupId,
                    userId: member.userId,
                    seconds: seconds
                )
                LinLanGroupMemberCache.invalidate(groupId: groupId)
                let until: UInt = seconds == 0
                    ? 0
                    : UInt(Date().timeIntervalSince1970) + seconds
                await MainActor.run {
                    self.updateMemberRow(userId: member.userId) { $0.muteUntil = until }
                    self.toast(seconds == 0 ? "已解除禁言" : "已禁言")
                }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func kick(_ member: LinLanGroupMemberItem) {
        Task {
            do {
                try await LinLanImManager.kickGroupMember(groupId: groupId, userIds: [member.userId])
                LinLanGroupMemberCache.invalidate(groupId: groupId)
                await MainActor.run { self.removeMemberRow(member) }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    /// 移出成功：只删对应行，不整表 reload
    private func removeMemberRow(_ member: LinLanGroupMemberItem) {
        members.removeAll { $0.userId == member.userId }
        guard let row = filtered.firstIndex(where: { $0.userId == member.userId }) else {
            title = "群成员 (\(members.count))"
            return
        }
        filtered.remove(at: row)
        title = "群成员 (\(members.count))"
        tableView.deleteRows(at: [IndexPath(row: row, section: 0)], with: .automatic)
    }

    /// 禁言 / 设管理员：只刷新对应行
    private func updateMemberRow(userId: String, mutate: (inout LinLanGroupMemberItem) -> Void) {
        if let i = members.firstIndex(where: { $0.userId == userId }) {
            mutate(&members[i])
        }
        guard let row = filtered.firstIndex(where: { $0.userId == userId }) else { return }
        mutate(&filtered[row])
        tableView.reloadRows(at: [IndexPath(row: row, section: 0)], with: .none)
    }

    private func setRole(_ member: LinLanGroupMemberItem, role: V2TIMGroupMemberRole) {
        Task {
            do {
                try await LinLanImManager.setGroupMemberRole(
                    groupId: groupId,
                    userId: member.userId,
                    role: role
                )
                LinLanGroupMemberCache.invalidate(groupId: groupId)
                let roleValue = UInt(role.rawValue)
                await MainActor.run {
                    self.updateMemberRow(userId: member.userId) { $0.role = roleValue }
                    self.toast(role == .V2TIM_GROUP_MEMBER_ROLE_ADMIN ? "已设为管理员" : "已取消管理员")
                }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }
}

extension LinLanGroupMembersViewController: UITableViewDataSource, UITableViewDelegate, UISearchBarDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { filtered.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        let m = filtered[indexPath.row]
        cell.textLabel?.text = m.displayName
        var detail = m.userId
        if m.isOwner { detail += " · 群主" }
        else if m.isAdmin { detail += " · 管理员" }
        if m.isMuted { detail += " · 禁言中" }
        cell.detailTextLabel?.text = detail
        cell.detailTextLabel?.textColor = AppTheme.textSecondary
        cell.accessoryType = .disclosureIndicator
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        showActions(for: filtered[indexPath.row])
    }

    func searchBar(_ searchBar: UISearchBar, textDidChange searchText: String) {
        applyFilter()
    }
}
