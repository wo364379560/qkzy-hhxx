//
//  LinLanNewFriendsViewController.swift
//  TodayToTrackRun
//
//  新朋友：好友申请 / 群通知 分段切换
//

import ImSDK_Plus_Swift
import SnapKit
import UIKit

final class LinLanNewFriendsViewController: UIViewController {
    private enum Tab: Int {
        case friends = 0
        case groups = 1
    }

    private let segment = UISegmentedControl(items: ["好友申请", "群通知"])
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let emptyLabel = UILabel()

    private var friendApps: [V2TIMFriendApplication] = []
    private var groupApps: [V2TIMGroupApplication] = []
    private var pendingFriendIDs = Set<String>()
    private var selectedTab: Tab = .friends
    private let ownerScope = LinLanFriendOwnerScope.capture()
    private var acceptedTargets = Set<String>()
    deinit { NotificationCenter.default.removeObserver(self) }

    /// 进入时默认选中的分段（0 好友 / 1 群通知）
    var initialTab: Int = 0

    override func viewDidLoad() {
        super.viewDidLoad()
        NotificationCenter.default.addObserver(self, selector: #selector(ownerChanged(_:)), name: .linLanFriendOwnershipChanged, object: nil)
        title = "新朋友"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )

        selectedTab = Tab(rawValue: initialTab) ?? .friends
        segment.selectedSegmentIndex = selectedTab.rawValue
        segment.selectedSegmentTintColor = AppTheme.accent
        segment.setTitleTextAttributes([.foregroundColor: UIColor.white], for: .selected)
        segment.setTitleTextAttributes([.foregroundColor: AppTheme.textPrimary], for: .normal)
        segment.addTarget(self, action: #selector(tabChanged), for: .valueChanged)
        view.addSubview(segment)
        segment.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(10)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(34)
        }

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = 72
        view.addSubview(tableView)
        tableView.snp.makeConstraints { make in
            make.top.equalTo(segment.snp.bottom).offset(8)
            make.leading.trailing.bottom.equalToSuperview()
        }

        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.font = .systemFont(ofSize: 15)
        emptyLabel.textAlignment = .center
        emptyLabel.numberOfLines = 0
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalTo(tableView) }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await linLanReload() }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    @objc private func tabChanged() {
        selectedTab = Tab(rawValue: segment.selectedSegmentIndex) ?? .friends
        applyEmpty()
        tableView.reloadData()
    }

    private func linLanReload() async {
        let friends = (try? await LinLanImManager.fetchIncomingFriendApplications())?.list ?? []
        let groups = (try? await LinLanImManager.fetchPendingGroupApplications()) ?? []
        await LinLanImManager.setGroupApplicationRead()
        await MainActor.run {
            guard self.ownerScope.active else { return }
            self.friendApps = friends
            self.groupApps = groups
            self.refreshSegmentTitles()
            self.applyEmpty()
            self.tableView.reloadData()
        }
    }

    private func refreshSegmentTitles() {
        let f = friendApps.count
        let g = groupApps.count
        segment.setTitle(f > 0 ? "好友申请(\(f))" : "好友申请", forSegmentAt: 0)
        segment.setTitle(g > 0 ? "群通知(\(g))" : "群通知", forSegmentAt: 1)
    }

    private func applyEmpty() {
        switch selectedTab {
        case .friends:
            emptyLabel.text = "暂无好友申请"
            emptyLabel.isHidden = !friendApps.isEmpty
        case .groups:
            emptyLabel.text = "暂无群邀请或入群申请"
            emptyLabel.isHidden = !groupApps.isEmpty
        }
    }

    private func linLanAcceptFriend(_ app: V2TIMFriendApplication) {
        guard ownerScope.active else { return }
        guard pendingFriendIDs.insert(app.userID).inserted else { return }
        acceptedTargets.insert(app.userID)
        tableView.reloadData()
        Task {
            defer { pendingFriendIDs.remove(app.userID); tableView.reloadData() }
            do {
                try await LinLanImManager.acceptFriendApplication(app)
                await linLanReload()
            } catch {
                await MainActor.run { if self.ownerScope.active { self.linLanToast(error.localizedDescription) } }
                await linLanReload()
            }
        }
    }

    private func linLanRefuseFriend(_ app: V2TIMFriendApplication) {
        guard ownerScope.active else { return }
        guard pendingFriendIDs.insert(app.userID).inserted else { return }
        tableView.reloadData()
        Task {
            defer { pendingFriendIDs.remove(app.userID); tableView.reloadData() }
            do {
                try await LinLanImManager.refuseFriendApplication(app)
                await linLanReload()
            } catch {
                await MainActor.run { self.linLanToast(error.localizedDescription) }
                await linLanReload()
            }
        }
    }

    @objc private func ownerChanged(_ note: Notification) {
        guard ownerScope.active, note.userInfo?["scope"] as? String == ownerScope.key,
              let target = note.userInfo?["target"] as? String, acceptedTargets.contains(target),
              let result = note.object as? LinLanFriendOwnerAssignment else { return }
        navigationItem.prompt = result.state == "pending" ? "好友已建立，归属待同步" :
            ["blocked", "conflict"].contains(result.state) ? "好友已建立，归属待管理员核对" : "好友已建立，归属已同步"
    }

    private func linLanAcceptGroup(_ app: V2TIMGroupApplication) {
        Task {
            do {
                try await LinLanImManager.acceptGroupApplication(app)
                await linLanReload()
                await MainActor.run { self.linLanToast("已同意") }
            } catch {
                await MainActor.run { self.linLanToast(error.localizedDescription) }
            }
        }
    }

    private func linLanRefuseGroup(_ app: V2TIMGroupApplication) {
        Task {
            do {
                try await LinLanImManager.refuseGroupApplication(app)
                await linLanReload()
            } catch {
                await MainActor.run { self.linLanToast(error.localizedDescription) }
            }
        }
    }

    private func linLanToast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        a.addAction(UIAlertAction(title: "好", style: .default))
        present(a, animated: true)
    }

    private func makeActionAccessory(
        refuse: @escaping () -> Void,
        accept: @escaping () -> Void
    ) -> UIView {
        let refuseBtn = UIButton(type: .system)
        refuseBtn.setTitle("拒绝", for: .normal)
        refuseBtn.setTitleColor(AppTheme.textSecondary, for: .normal)
        refuseBtn.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)

        let acceptBtn = UIButton(type: .system)
        acceptBtn.setTitle("同意", for: .normal)
        acceptBtn.setTitleColor(.white, for: .normal)
        acceptBtn.backgroundColor = AppTheme.accent
        acceptBtn.layer.cornerRadius = 14
        acceptBtn.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        acceptBtn.contentEdgeInsets = UIEdgeInsets(top: 6, left: 14, bottom: 6, right: 14)

        let actions = UIStackView(arrangedSubviews: [refuseBtn, acceptBtn])
        actions.axis = .horizontal
        actions.spacing = 10
        actions.alignment = .center
        actions.bounds = CGRect(x: 0, y: 0, width: 140, height: 36)
        refuseBtn.addAction(UIAction { _ in refuse() }, for: .touchUpInside)
        acceptBtn.addAction(UIAction { _ in accept() }, for: .touchUpInside)
        return actions
    }
}

extension LinLanNewFriendsViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        switch selectedTab {
        case .friends: return friendApps.count
        case .groups: return groupApps.count
        }
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = .white
        cell.selectionStyle = .none
        cell.textLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        cell.detailTextLabel?.textColor = AppTheme.textSecondary
        cell.detailTextLabel?.numberOfLines = 2

        switch selectedTab {
        case .friends:
            let app = friendApps[indexPath.row]
            let name = (app.nickName?.isEmpty == false ? app.nickName! : app.userID)
            cell.textLabel?.text = name
            cell.detailTextLabel?.text = app.addWording?.isEmpty == false
                ? app.addWording
                : "请求添加你为好友"
            cell.accessoryView = makeActionAccessory(
                refuse: { [weak self] in self?.linLanRefuseFriend(app) },
                accept: { [weak self] in self?.linLanAcceptFriend(app) }
            )
            cell.accessoryView?.isUserInteractionEnabled = !pendingFriendIDs.contains(app.userID)
            cell.accessoryView?.alpha = pendingFriendIDs.contains(app.userID) ? 0.5 : 1
        case .groups:
            let app = groupApps[indexPath.row]
            cell.textLabel?.text = LinLanImManager.groupApplicationTitle(app)
            cell.detailTextLabel?.text = LinLanImManager.groupApplicationSubtitle(app)
            cell.accessoryView = makeActionAccessory(
                refuse: { [weak self] in self?.linLanRefuseGroup(app) },
                accept: { [weak self] in self?.linLanAcceptGroup(app) }
            )
        }
        return cell
    }
}
