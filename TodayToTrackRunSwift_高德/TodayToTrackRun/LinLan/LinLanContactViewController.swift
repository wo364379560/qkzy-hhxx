//
//  LinLanContactViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanContactViewController: UIViewController {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)

    private struct HeaderRow {
        let title: String
        let systemImage: String
        let badge: Int
        let open: () -> Void
    }

    private var headerRows: [HeaderRow] = []
    private var friends: [LinLanFriendItem] = []
    private var pendingBadge = 0

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(LinLanContactHeaderCell.self, forCellReuseIdentifier: LinLanContactHeaderCell.reuseID)
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "friend")
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalToSuperview() }
        linLanRebuildHeaders()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await linLanReload() }
    }

    private func linLanRebuildHeaders() {
        headerRows = [
            HeaderRow(title: "新朋友", systemImage: "person.badge.plus", badge: pendingBadge) { [weak self] in
                self?.linLanPush(LinLanNewFriendsViewController())
            },
            HeaderRow(title: "群组", systemImage: "person.3.fill", badge: 0) { [weak self] in
                self?.linLanPush(LinLanGroupListViewController())
            },
            HeaderRow(title: "黑名单", systemImage: "nosign", badge: 0) { [weak self] in
                self?.linLanPush(LinLanBlacklistViewController())
            },
            HeaderRow(title: "朋友圈", systemImage: "circle.grid.2x2.fill", badge: 0) { [weak self] in
                self?.linLanPush(LinLanMomentsViewController())
            },
        ]
    }

    private func linLanReload() async {
        _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
        guard let friendList = try? await LinLanImManager.fetchFriendList() else { return }
        let apps = try? await LinLanImManager.fetchIncomingFriendApplications()
        let groupApps = (try? await LinLanImManager.fetchPendingGroupApplications()) ?? []
        friends = friendList
        let friendUnread = Int(apps?.unread ?? UInt(apps?.list.count ?? 0))
        pendingBadge = friendUnread + groupApps.count
        linLanRebuildHeaders()
        tableView.reloadData()
    }
}

extension LinLanContactViewController: UITableViewDataSource, UITableViewDelegate {
    func numberOfSections(in tableView: UITableView) -> Int { 2 }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        section == 0 ? headerRows.count : max(friends.count, 1)
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        section == 1 ? "好友" : nil
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if indexPath.section == 0 {
            let cell = tableView.dequeueReusableCell(
                withIdentifier: LinLanContactHeaderCell.reuseID,
                for: indexPath
            ) as! LinLanContactHeaderCell
            let r = headerRows[indexPath.row]
            cell.configure(title: r.title, systemImage: r.systemImage, badge: r.badge)
            return cell
        }

        let cell = tableView.dequeueReusableCell(withIdentifier: "friend", for: indexPath)
        cell.backgroundColor = .white
        var content = cell.defaultContentConfiguration()
        if friends.isEmpty {
            content.text = "暂无好友"
            content.textProperties.color = AppTheme.textSecondary
            cell.contentConfiguration = content
            cell.accessoryType = .none
            cell.selectionStyle = .none
        } else {
            let f = friends[indexPath.row]
            content.text = f.displayName
            content.textProperties.color = AppTheme.textPrimary
            content.secondaryText = f.userId
            content.secondaryTextProperties.color = AppTheme.textSecondary
            content.secondaryTextProperties.font = .systemFont(ofSize: 12)
            cell.contentConfiguration = content
            cell.accessoryType = .disclosureIndicator
            cell.selectionStyle = .default
        }
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        if indexPath.section == 0 {
            headerRows[indexPath.row].open()
        } else if !friends.isEmpty {
            let f = friends[indexPath.row]
            linLanPush(
                LinLanFriendDetailViewController(
                    userId: f.userId,
                    userName: f.displayName,
                    isFriend: true
                )
            )
        }
    }
}

/// 通讯录入口行：右侧圆角数字角标（对齐消息列表未读样式）
private final class LinLanContactHeaderCell: UITableViewCell {
    static let reuseID = "LinLanContactHeaderCell"

    private let iconView = UIImageView()
    private let titleLabel = UILabel()
    private let badgeLabel = UILabel()
    private let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        backgroundColor = .white
        selectionStyle = .default

        iconView.contentMode = .scaleAspectFit
        iconView.tintColor = AppTheme.accent
        titleLabel.font = .systemFont(ofSize: 16, weight: .medium)
        titleLabel.textColor = AppTheme.textPrimary

        badgeLabel.font = .systemFont(ofSize: 11, weight: .bold)
        badgeLabel.textColor = .white
        badgeLabel.backgroundColor = AppTheme.accent
        badgeLabel.textAlignment = .center
        badgeLabel.layer.cornerRadius = 9
        badgeLabel.clipsToBounds = true
        badgeLabel.isHidden = true

        chevron.tintColor = UIColor.tertiaryLabel
        chevron.contentMode = .scaleAspectFit

        contentView.addSubview(iconView)
        contentView.addSubview(titleLabel)
        contentView.addSubview(badgeLabel)
        contentView.addSubview(chevron)

        iconView.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.size.equalTo(22)
        }
        titleLabel.snp.makeConstraints { make in
            make.leading.equalTo(iconView.snp.trailing).offset(12)
            make.centerY.equalToSuperview()
        }
        chevron.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(14)
            make.centerY.equalToSuperview()
            make.size.equalTo(12)
        }
        badgeLabel.snp.makeConstraints { make in
            make.trailing.equalTo(chevron.snp.leading).offset(-10)
            make.centerY.equalToSuperview()
            make.height.equalTo(18)
            make.width.greaterThanOrEqualTo(18)
            make.leading.greaterThanOrEqualTo(titleLabel.snp.trailing).offset(8)
        }
    }

    required init?(coder: NSCoder) { fatalError() }

    func configure(title: String, systemImage: String, badge: Int) {
        titleLabel.text = title
        iconView.image = UIImage(systemName: systemImage)
        if badge > 0 {
            badgeLabel.isHidden = false
            badgeLabel.text = badge > 99 ? " 99+ " : " \(badge) "
        } else {
            badgeLabel.isHidden = true
            badgeLabel.text = nil
        }
    }
}
