//
//  LinLanGroupListViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanGroupListViewController: UIViewController {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let emptyLabel = UILabel()
    private var groups: [LinLanGroupItem] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "群组"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        emptyLabel.text = "暂无群组"
        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.font = .systemFont(ofSize: 15)
        emptyLabel.textAlignment = .center
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalToSuperview() }

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onMembershipChanged),
            name: .linLanGroupMembershipChanged,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onMembershipChanged),
            name: .linLanCurrentUserRemovedFromGroup,
            object: nil
        )
    }

    deinit { NotificationCenter.default.removeObserver(self) }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await linLanReload() }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func onMembershipChanged() {
        Task { await linLanReload() }
    }

    private func linLanReload() async {
        let scope = LinLanImManager.currentAccountStorageScope
        guard !scope.isEmpty else { return }
        do {
            let snapshot = try await LinLanImManager.fetchJoinedGroupSnapshot()
            await MainActor.run {
                guard LinLanImManager.currentAccountStorageScope == scope else { return }
                let removed = LinLanGroupMembershipState.removedGroupIDs(scope: scope)
                let fetched = snapshot.groups.filter {
                    !LinLanGroupMembershipPolicy.isKnownRemoved($0.groupId, removedGroupIDs: removed)
                }
                if snapshot.isComplete,
                   LinLanGroupMembershipState.isCurrent(snapshot.membershipToken) {
                    self.groups = fetched
                } else {
                    // Community 查询短暂失败或期间成员关系变化时，只合并新结果，
                    // 不用不完整快照清空已显示的群。
                    var byID = Dictionary(uniqueKeysWithValues: self.groups.map { ($0.groupId, $0) })
                    fetched.forEach { byID[$0.groupId] = $0 }
                    self.groups = byID.values
                        .filter { !LinLanGroupMembershipPolicy.isKnownRemoved($0.groupId, removedGroupIDs: removed) }
                        .sorted { $0.groupName.localizedCompare($1.groupName) == .orderedAscending }
                }
                self.emptyLabel.text = "暂无群组"
                self.emptyLabel.isHidden = !self.groups.isEmpty
                self.tableView.reloadData()
            }
        } catch {
            await MainActor.run {
                guard LinLanImManager.currentAccountStorageScope == scope else { return }
                // 网络/Community 查询失败不得清空已有群列表。
                if self.groups.isEmpty {
                    self.emptyLabel.isHidden = false
                    self.emptyLabel.text = error.localizedDescription
                }
            }
        }
    }
}

extension LinLanGroupListViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        groups.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = .white
        cell.accessoryType = .disclosureIndicator
        let g = groups[indexPath.row]
        cell.textLabel?.text = g.groupName
        cell.textLabel?.font = .systemFont(ofSize: 16, weight: .medium)
        cell.detailTextLabel?.text = g.groupId
        cell.detailTextLabel?.textColor = AppTheme.textSecondary
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let g = groups[indexPath.row]
        linLanPush(
            LinLanChatViewController(
                titleText: g.groupName,
                userID: nil,
                groupID: g.groupId
            )
        )
    }
}
