//
//  LinLanCreateGroupViewController.swift
//  TodayToTrackRun
//
//  对照 Android LinLanCreateGroupScreen：选好友 + 建 Community
//

import SnapKit
import UIKit

final class LinLanCreateGroupViewController: UIViewController {
    private let nameField = UITextField()
    private let tipLabel = UILabel()
    private let messageLabel = UILabel()
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var friends: [LinLanFriendItem] = []
    private var selected = Set<String>()
    private var creating = false
    private var createdGroupId: String?
    private let selectAllButton = UIButton(type: .system)

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "新建群聊"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "创建",
            style: .done,
            target: self,
            action: #selector(linLanCreate)
        )
        navigationItem.rightBarButtonItem?.tintColor = AppTheme.accent

        nameField.placeholder = "群名称"
        nameField.borderStyle = .roundedRect
        nameField.backgroundColor = .white
        nameField.clearButtonMode = .whileEditing

        tipLabel.text = "群类型：Community（社群）· 选择成员（可选）"
        tipLabel.font = .systemFont(ofSize: 13)
        tipLabel.textColor = AppTheme.textSecondary
        tipLabel.numberOfLines = 0

        messageLabel.font = .systemFont(ofSize: 13)
        messageLabel.textColor = .systemRed
        messageLabel.numberOfLines = 0
        messageLabel.isHidden = true

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.allowsMultipleSelection = true

        let header = UIView()
        header.addSubview(nameField)
        header.addSubview(tipLabel)
        header.addSubview(messageLabel)
        header.addSubview(selectAllButton)
        selectAllButton.setTitle("全选（最多100人）", for: .normal)
        selectAllButton.addTarget(self, action: #selector(linLanSelectAll), for: .touchUpInside)
        selectAllButton.snp.makeConstraints { make in
            make.top.equalTo(tipLabel.snp.bottom).offset(4)
            make.leading.equalToSuperview().offset(16)
            make.height.equalTo(36)
        }
        nameField.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(44)
        }
        tipLabel.snp.makeConstraints { make in
            make.top.equalTo(nameField.snp.bottom).offset(10)
            make.leading.trailing.equalToSuperview().inset(16)
        }
        messageLabel.snp.makeConstraints { make in
            make.top.equalTo(selectAllButton.snp.bottom).offset(6)
            make.leading.trailing.equalToSuperview().inset(16)
            make.bottom.equalToSuperview().inset(8)
        }
        header.frame = CGRect(x: 0, y: 0, width: view.bounds.width, height: 120)
        tableView.tableHeaderView = header

        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        Task { await linLanReloadFriends() }
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        guard let header = tableView.tableHeaderView else { return }
        let target = header.systemLayoutSizeFitting(
            CGSize(width: tableView.bounds.width, height: UIView.layoutFittingCompressedSize.height)
        )
        if abs(header.frame.height - target.height) > 0.5 {
            header.frame.size.height = max(target.height, 110)
            tableView.tableHeaderView = header
        }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    private func linLanReloadFriends() async {
        let list = (try? await LinLanImManager.fetchFriendList()) ?? []
        await MainActor.run {
            self.friends = list
            self.tableView.reloadData()
        }
    }

    @objc private func linLanCreate() {
        view.endEditing(true)
        let name = nameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !name.isEmpty else {
            linLanShow("请输入群名称")
            return
        }
        if let gid = createdGroupId {
            linLanOpenCreatedGroup(gid, name: name)
            return
        }
        guard !creating else { return }
        creating = true
        navigationItem.rightBarButtonItem?.isEnabled = false
        navigationItem.rightBarButtonItem?.title = "创建中"
        linLanShow(nil)

        Task {
            do {
                _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
                let gid = try await LinLanImManager.createCommunityGroup(
                    name: name,
                    memberUserIds: Array(selected)
                )
                await MainActor.run {
                    self.linLanOpenCreatedGroup(gid, name: name)
                }
            } catch {
                await MainActor.run {
                    self.creating = false
                    self.navigationItem.rightBarButtonItem?.isEnabled = true
                    if let partial = error as? LinLanImManager.PartialGroupCreation {
                        self.createdGroupId = partial.groupId
                    }
                    self.navigationItem.rightBarButtonItem?.title = self.createdGroupId == nil ? "创建" : "进入群聊"
                    self.linLanShow(error.localizedDescription)
                }
            }
        }
    }

    private func linLanOpenCreatedGroup(_ gid: String, name: String) {
        linLanPush(LinLanChatViewController(titleText: name, userID: nil, groupID: gid))
        if var stack = navigationController?.viewControllers {
            stack.removeAll { $0 === self }
            navigationController?.viewControllers = stack
        }
    }

    @objc private func linLanSelectAll() {
        guard !creating, createdGroupId == nil else { return }
        if selected.count >= min(100, friends.count) { selected.removeAll() }
        else {
            for friend in friends where selected.count < 100 { selected.insert(friend.userId) }
        }
        linLanUpdateSelection()
    }

    private func linLanUpdateSelection() {
        tipLabel.text = "已选 \(selected.count)/100 位好友"
        selectAllButton.setTitle(!selected.isEmpty && selected.count >= min(100, friends.count) ? "取消全选" : "全选（最多100人）", for: .normal)
        tableView.reloadData()
    }

    private func linLanShow(_ text: String?) {
        messageLabel.text = text
        messageLabel.isHidden = (text == nil || text?.isEmpty == true)
    }
}

extension LinLanCreateGroupViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        max(friends.count, 1)
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        "选择成员"
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        cell.backgroundColor = .white
        if friends.isEmpty {
            cell.textLabel?.text = "暂无好友可选"
            cell.textLabel?.textColor = AppTheme.textSecondary
            cell.selectionStyle = .none
            return cell
        }
        let f = friends[indexPath.row]
        cell.textLabel?.text = f.displayName
        cell.textLabel?.textColor = AppTheme.textPrimary
        let on = selected.contains(f.userId)
        cell.accessoryType = on ? .checkmark : .none
        cell.tintColor = AppTheme.accent
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        guard !friends.isEmpty, !creating, createdGroupId == nil else { return }
        let id = friends[indexPath.row].userId
        if selected.contains(id) { selected.remove(id) }
        else if selected.count < 100 { selected.insert(id) }
        else { linLanShow("最多选择 100 位好友") }
        linLanUpdateSelection()
    }
}
