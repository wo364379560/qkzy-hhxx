//
//  LinLanAddFriendViewController.swift
//  TodayToTrackRun
//
//  对照 Android LinLanAddFriendScreen：HTTP 搜索 + TIM addFriend
//

import SnapKit
import UIKit

final class LinLanAddFriendViewController: UIViewController {
    private let field = UITextField()
    private let searchButton = UIButton(type: .system)
    private let messageLabel = UILabel()
    private let tableView = UITableView(frame: .zero, style: .plain)

    private var results: [LinLanSearchUser] = []
    private var friendIds = Set<String>()
    private var searching = false
    private var addingIds = Set<String>()
    private var waitingIds = Set<String>()
    private let ownerScope = LinLanFriendOwnerScope.capture()

    deinit { NotificationCenter.default.removeObserver(self) }

    override func viewDidLoad() {
        super.viewDidLoad()
        NotificationCenter.default.addObserver(self, selector: #selector(ownerChanged(_:)), name: .linLanFriendOwnershipChanged, object: nil)
        title = "添加好友"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        field.placeholder = "搜索 LinLan号 / 手机号"
        field.borderStyle = .roundedRect
        field.backgroundColor = .white
        field.clearButtonMode = .whileEditing
        field.returnKeyType = .search
        field.delegate = self

        searchButton.setTitle("搜索", for: .normal)
        searchButton.setTitleColor(.white, for: .normal)
        searchButton.backgroundColor = AppTheme.accent
        searchButton.layer.cornerRadius = 10
        searchButton.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        searchButton.addTarget(self, action: #selector(linLanSearch), for: .touchUpInside)

        messageLabel.font = .systemFont(ofSize: 13)
        messageLabel.textColor = AppTheme.accent
        messageLabel.numberOfLines = 0
        messageLabel.isHidden = true

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = 64
        tableView.keyboardDismissMode = .onDrag
        tableView.separatorInset = UIEdgeInsets(top: 0, left: 72, bottom: 0, right: 0)

        view.addSubview(field)
        view.addSubview(searchButton)
        view.addSubview(messageLabel)
        view.addSubview(tableView)
        field.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(16)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(48)
        }
        searchButton.snp.makeConstraints { make in
            make.top.equalTo(field.snp.bottom).offset(12)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(48)
        }
        messageLabel.snp.makeConstraints { make in
            make.top.equalTo(searchButton.snp.bottom).offset(10)
            make.leading.trailing.equalToSuperview().inset(16)
        }
        tableView.snp.makeConstraints { make in
            make.top.equalTo(messageLabel.snp.bottom).offset(8)
            make.leading.trailing.bottom.equalToSuperview()
        }

        Task {
            let friends = (try? await LinLanImManager.fetchFriendList()) ?? []
            await MainActor.run {
                guard self.ownerScope.active else { return }
                self.friendIds = Set(friends.map(\.userId))
            }
        }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func linLanSearch() {
        view.endEditing(true)
        let q = field.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !q.isEmpty else {
            linLanShowMessage("请输入搜索关键词")
            return
        }
        guard !searching else { return }
        searching = true
        searchButton.setTitle("搜索中…", for: .normal)
        searchButton.isEnabled = false
        linLanShowMessage(nil)

        Task {
            do {
                _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
                let list = try await LinLanAppContainer.apiClient().searchUsersPreferExact(keyword: q)
                await MainActor.run {
                    guard self.ownerScope.active else { return }
                    self.results = list
                    if list.isEmpty {
                        self.linLanShowMessage("未找到用户")
                    } else {
                        self.linLanShowMessage(nil)
                    }
                    self.tableView.reloadData()
                    self.searching = false
                    self.searchButton.setTitle("搜索", for: .normal)
                    self.searchButton.isEnabled = true
                }
            } catch {
                await MainActor.run {
                    guard self.ownerScope.active else { return }
                    self.results = []
                    self.tableView.reloadData()
                    self.linLanShowMessage(error.localizedDescription)
                    self.searching = false
                    self.searchButton.setTitle("搜索", for: .normal)
                    self.searchButton.isEnabled = true
                }
            }
        }
    }

    private func linLanAdd(_ user: LinLanSearchUser) {
        guard ownerScope.active, !waitingIds.contains(user.imUserId), addingIds.insert(user.imUserId).inserted else { return }
        tableView.reloadData()
        Task {
            defer { addingIds.remove(user.imUserId); if ownerScope.active { tableView.reloadData() } }
            do {
                let outcome = try await LinLanImManager.addFriend(userId: user.imUserId)
                await MainActor.run {
                    guard self.ownerScope.active else { return }
                    self.linLanShowMessage(outcome.toast)
                    if outcome == .pendingConfirm { self.waitingIds.insert(user.imUserId); self.tableView.reloadData() }
                    if outcome == .addedDirectly || outcome == .alreadyFriends {
                        self.friendIds.insert(user.imUserId)
                        self.tableView.reloadData()
                    }
                }
            } catch {
                await MainActor.run { if self.ownerScope.active { self.linLanShowMessage(error.localizedDescription) } }
            }
        }
    }

    @objc private func ownerChanged(_ note: Notification) {
        guard ownerScope.active, note.userInfo?["scope"] as? String == ownerScope.key,
              let target = note.userInfo?["target"] as? String,
              results.contains(where: { $0.imUserId == target }),
              let result = note.object as? LinLanFriendOwnerAssignment else { return }
        if result.state != "pending" { waitingIds.remove(target) }
        switch result.state {
        case "pending": linLanShowMessage("好友已建立，归属待同步")
        case "blocked", "conflict": linLanShowMessage("好友已建立，归属待管理员核对")
        default: linLanShowMessage("好友已建立，归属已同步")
        }
        friendIds.insert(target); tableView.reloadData()
    }

    private func linLanShowMessage(_ text: String?) {
        messageLabel.text = text
        messageLabel.isHidden = (text == nil || text?.isEmpty == true)
        messageLabel.textColor = (text?.contains("失败") == true || text?.contains("错误") == true)
            ? .systemRed
            : AppTheme.accent
    }
}

extension LinLanAddFriendViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        linLanSearch()
        return true
    }
}

extension LinLanAddFriendViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        results.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = .white
        let user = results[indexPath.row]
        cell.textLabel?.text = user.displayName
        cell.textLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        cell.detailTextLabel?.text = user.wananId.isEmpty ? user.imUserId : user.wananId
        cell.detailTextLabel?.textColor = AppTheme.textSecondary
        cell.selectionStyle = .none

        let action = UIButton(type: .system)
        action.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        action.contentEdgeInsets = UIEdgeInsets(top: 6, left: 12, bottom: 6, right: 12)
        action.layer.cornerRadius = 14
        if friendIds.contains(user.imUserId) {
            action.setTitle("发消息", for: .normal)
            action.setTitleColor(AppTheme.accent, for: .normal)
            action.addAction(UIAction { [weak self] _ in
                self?.linLanPush(
                    LinLanChatViewController(
                        titleText: user.displayName,
                        userID: user.imUserId,
                        groupID: nil
                    )
                )
            }, for: .touchUpInside)
        } else {
            let busy = addingIds.contains(user.imUserId), waiting = waitingIds.contains(user.imUserId)
            action.setTitle(busy ? "处理中" : waiting ? "待验证" : "添加", for: .normal)
            action.isEnabled = !busy && !waiting && user.canAddFriend
            action.setTitleColor(.white, for: .normal)
            action.backgroundColor = AppTheme.accent
            action.addAction(UIAction { [weak self] _ in
                self?.linLanAdd(user)
            }, for: .touchUpInside)
        }
        action.sizeToFit()
        cell.accessoryView = action
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        let user = results[indexPath.row]
        guard friendIds.contains(user.imUserId) else { return }
        linLanPush(
            LinLanFriendDetailViewController(
                userId: user.imUserId,
                userName: user.displayName,
                isFriend: true
            )
        )
    }
}
