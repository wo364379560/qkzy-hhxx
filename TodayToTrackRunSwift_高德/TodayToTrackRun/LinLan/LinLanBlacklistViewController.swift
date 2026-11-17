//
//  LinLanBlacklistViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanBlacklistViewController: UIViewController {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let emptyLabel = UILabel()
    private var rows: [LinLanFriendItem] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "黑名单"
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

        emptyLabel.text = "黑名单为空"
        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.font = .systemFont(ofSize: 15)
        emptyLabel.textAlignment = .center
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalToSuperview() }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await linLanReload() }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }

    private func linLanReload() async {
        do {
            let list = try await LinLanImManager.fetchBlackList()
            await MainActor.run {
                self.rows = list
                self.emptyLabel.isHidden = !list.isEmpty
                self.tableView.reloadData()
            }
        } catch {
            await MainActor.run {
                self.rows = []
                self.emptyLabel.isHidden = false
                self.emptyLabel.text = error.localizedDescription
                self.tableView.reloadData()
            }
        }
    }
}

extension LinLanBlacklistViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        rows.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = .white
        let item = rows[indexPath.row]
        cell.textLabel?.text = item.displayName
        cell.detailTextLabel?.text = item.userId
        cell.detailTextLabel?.textColor = AppTheme.textSecondary
        return cell
    }

    func tableView(
        _ tableView: UITableView,
        trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath
    ) -> UISwipeActionsConfiguration? {
        let item = rows[indexPath.row]
        let action = UIContextualAction(style: .destructive, title: "移除") { [weak self] _, _, done in
            guard let self else { done(false); return }
            Task {
                do {
                    try await LinLanImManager.removeFromBlackList(userId: item.userId)
                    await self.linLanReload()
                    await MainActor.run {
                        done(true)
                        self.promptReAddFriend(item)
                    }
                } catch {
                    await MainActor.run {
                        let a = UIAlertController(
                            title: nil,
                            message: error.localizedDescription,
                            preferredStyle: .alert
                        )
                        a.addAction(UIAlertAction(title: "好", style: .default))
                        self.present(a, animated: true)
                        done(false)
                    }
                }
            }
        }
        return UISwipeActionsConfiguration(actions: [action])
    }

    /// 拉黑会解除好友；移出后引导重新加回
    private func promptReAddFriend(_ item: LinLanFriendItem) {
        let a = UIAlertController(
            title: "已移出黑名单",
            message: "拉黑会解除好友关系。是否重新添加「\(item.displayName)」为好友？",
            preferredStyle: .alert
        )
        a.addAction(UIAlertAction(title: "稍后", style: .cancel))
        a.addAction(UIAlertAction(title: "重新添加", style: .default) { [weak self] _ in
            guard let self else { return }
            Task {
                do {
                    let outcome = try await LinLanImManager.addFriend(userId: item.userId)
                    await MainActor.run {
                        let tip = UIAlertController(
                            title: nil,
                            message: outcome.toast,
                            preferredStyle: .alert
                        )
                        tip.addAction(UIAlertAction(title: "好", style: .default))
                        self.present(tip, animated: true)
                    }
                } catch {
                    await MainActor.run {
                        let tip = UIAlertController(
                            title: nil,
                            message: error.localizedDescription,
                            preferredStyle: .alert
                        )
                        tip.addAction(UIAlertAction(title: "好", style: .default))
                        self.present(tip, animated: true)
                    }
                }
            }
        })
        present(a, animated: true)
    }
}
