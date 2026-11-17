//
//  LinLanForwardTargetViewController.swift
//  TodayToTrackRun
//
//  对照 Android ForwardTargetScreen
//

import ImSDK_Plus_Swift
import SnapKit
import UIKit

struct LinLanForwardTarget {
    let key: String
    let title: String
    let receiver: String?
    let groupId: String?
    let isGroup: Bool
    let faceUrl: String?
}

final class LinLanForwardTargetViewController: UIViewController {
    enum Filter: Int { case all = 0, group, friend }

    var onPicked: (([LinLanForwardTarget]) -> Void)?
    private let excludeConversationId: String?

    private var allTargets: [LinLanForwardTarget] = []
    private var filtered: [LinLanForwardTarget] = []
    private var selected = Set<String>()
    private var filter: Filter = .all
    private var searchText = ""

    private let segment = UISegmentedControl(items: ["全部", "群聊", "好友"])
    private let search = UISearchBar()
    private let selectAllButton = UIButton(type: .system)
    private let tableView = UITableView(frame: .zero, style: .plain)
    private let sendItem = UIBarButtonItem(title: "发送", style: .done, target: nil, action: nil)

    init(excludeConversationId: String? = nil) {
        self.excludeConversationId = excludeConversationId
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "选择会话"
        view.backgroundColor = UIColor(red: 0.95, green: 0.95, blue: 0.97, alpha: 1)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "xmark"),
            style: .plain,
            target: self,
            action: #selector(close)
        )
        sendItem.target = self
        sendItem.action = #selector(send)
        sendItem.tintColor = UIColor(red: 0, green: 0.48, blue: 1, alpha: 1)
        navigationItem.rightBarButtonItem = sendItem
        refreshSendTitle()

        segment.selectedSegmentIndex = 0
        segment.addTarget(self, action: #selector(filterChanged), for: .valueChanged)
        search.placeholder = "搜索会话"
        search.searchBarStyle = .minimal
        search.delegate = self

        selectAllButton.contentHorizontalAlignment = .leading
        selectAllButton.titleLabel?.font = .systemFont(ofSize: 16)
        selectAllButton.tintColor = UIColor(red: 0, green: 0.48, blue: 1, alpha: 1)
        selectAllButton.addTarget(self, action: #selector(toggleSelectAll), for: .touchUpInside)
        selectAllButton.snp.makeConstraints { $0.height.equalTo(42) }

        tableView.dataSource = self
        tableView.delegate = self
        tableView.backgroundColor = .white
        tableView.rowHeight = 64
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "c")
        tableView.tableFooterView = UIView()

        let header = UIStackView(arrangedSubviews: [segment, search, selectAllButton])
        header.axis = .vertical
        header.spacing = 8
        header.isLayoutMarginsRelativeArrangement = true
        header.layoutMargins = UIEdgeInsets(top: 8, left: 16, bottom: 4, right: 16)

        view.addSubview(header)
        view.addSubview(tableView)
        header.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide)
            make.leading.trailing.equalToSuperview()
        }
        tableView.snp.makeConstraints { make in
            make.top.equalTo(header.snp.bottom)
            make.leading.trailing.bottom.equalToSuperview()
        }

        Task { await load() }
    }

    private func load() async {
        let friends = (try? await LinLanImManager.fetchFriendList()) ?? []
        let convs = (try? await LinLanImManager.fetchConversations()) ?? []
        var list: [LinLanForwardTarget] = []
        for f in friends {
            let key = "c2c_\(f.userId)"
            if key == excludeConversationId { continue }
            list.append(
                LinLanForwardTarget(
                    key: key,
                    title: f.displayName,
                    receiver: f.userId,
                    groupId: nil,
                    isGroup: false,
                    faceUrl: nil
                )
            )
        }

        // 好友接口在 IM 刚登录、本地资料尚未同步时可能短暂返回空。
        // 同时合并单聊会话，保证转发页仍能展示可选的好友/单聊目标。
        var knownC2C = Set(list.compactMap(\.receiver))
        for c in convs where c.type == .V2TIM_C2C {
            let fallbackID = c.conversationID.replacingOccurrences(of: "c2c_", with: "")
            let uid = (c.userID ?? fallbackID).trimmingCharacters(in: .whitespacesAndNewlines)
            guard !uid.isEmpty, uid != LinLanImManager.loginUser, knownC2C.insert(uid).inserted else { continue }
            let key = c.conversationID.isEmpty ? "c2c_\(uid)" : c.conversationID
            if key == excludeConversationId || "c2c_\(uid)" == excludeConversationId { continue }
            list.append(
                LinLanForwardTarget(
                    key: key,
                    title: c.showName?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
                        ? c.showName! : uid,
                    receiver: uid,
                    groupId: nil,
                    isGroup: false,
                    faceUrl: c.faceUrl
                )
            )
        }
        for c in convs where c.type == .V2TIM_GROUP {
            let gid = c.groupID ?? c.conversationID.replacingOccurrences(of: "group_", with: "")
            let key = c.conversationID
            if key == excludeConversationId || "group_\(gid)" == excludeConversationId { continue }
            if list.contains(where: { $0.groupId == gid }) { continue }
            list.append(
                LinLanForwardTarget(
                    key: key,
                    title: c.showName ?? gid,
                    receiver: nil,
                    groupId: gid,
                    isGroup: true,
                    faceUrl: c.faceUrl
                )
            )
        }
        allTargets = list
        applyFilter()
    }

    private func applyFilter() {
        let q = searchText.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        filtered = allTargets.filter { item in
            let tabOk: Bool = {
                switch filter {
                case .all: return true
                case .group: return item.isGroup
                case .friend: return !item.isGroup
                }
            }()
            let searchOk = q.isEmpty
                || item.title.lowercased().contains(q)
                || item.key.lowercased().contains(q)
            return tabOk && searchOk
        }
        tableView.reloadData()
        refreshSendTitle()
        refreshSelectAll()
    }

    private func refreshSendTitle() {
        sendItem.title = selected.isEmpty ? "发送" : "发送(\(selected.count))"
        sendItem.isEnabled = !selected.isEmpty
    }

    private func refreshSelectAll() {
        let visibleKeys = Set(filtered.map(\.key))
        let allSelected = !visibleKeys.isEmpty && visibleKeys.isSubset(of: selected)
        let image = UIImage(systemName: allSelected ? "checkmark.circle.fill" : "circle")
        selectAllButton.setImage(image, for: .normal)
        selectAllButton.setTitle(allSelected ? "  取消全选" : "  全选", for: .normal)
        selectAllButton.isEnabled = !visibleKeys.isEmpty
        selectAllButton.alpha = visibleKeys.isEmpty ? 0.45 : 1
    }

    @objc private func toggleSelectAll() {
        let visibleKeys = Set(filtered.map(\.key))
        guard !visibleKeys.isEmpty else { return }
        if visibleKeys.isSubset(of: selected) {
            selected.subtract(visibleKeys)
        } else {
            selected.formUnion(visibleKeys)
        }
        tableView.reloadData()
        refreshSendTitle()
        refreshSelectAll()
    }

    @objc private func filterChanged() {
        filter = Filter(rawValue: segment.selectedSegmentIndex) ?? .all
        applyFilter()
    }

    @objc private func close() {
        dismiss(animated: true)
    }

    @objc private func send() {
        let picked = allTargets.filter { selected.contains($0.key) }
        guard !picked.isEmpty else { return }
        let callback = onPicked
        dismiss(animated: true) {
            callback?(picked)
        }
    }
}

extension LinLanForwardTargetViewController: UITableViewDataSource, UITableViewDelegate, UISearchBarDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { filtered.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        let item = filtered[indexPath.row]
        let checked = selected.contains(item.key)
        cell.selectionStyle = .none
        cell.backgroundColor = .white

        let check = UIImageView(
            image: UIImage(systemName: checked ? "checkmark.circle.fill" : "circle")
        )
        check.tintColor = checked ? UIColor(red: 0, green: 0.48, blue: 1, alpha: 1) : .lightGray
        let avatar = UIImageView()
        avatar.layer.cornerRadius = 22
        avatar.clipsToBounds = true
        avatar.contentMode = .scaleAspectFill
        if item.isGroup {
            avatar.backgroundColor = UIColor(red: 0.82, green: 0.91, blue: 1, alpha: 1)
            avatar.image = UIImage(systemName: "person.3.fill")
            avatar.tintColor = UIColor(red: 0, green: 0.48, blue: 1, alpha: 1)
            avatar.contentMode = .center
        } else {
            LinLanImageLoader.load(
                urlString: item.faceUrl,
                into: avatar,
                placeholderLetter: String(item.title.prefix(1))
            )
        }
        let name = UILabel()
        name.text = item.title
        name.font = .systemFont(ofSize: 16)
        name.textColor = .label

        cell.contentView.addSubview(check)
        cell.contentView.addSubview(avatar)
        cell.contentView.addSubview(name)
        check.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.size.equalTo(22)
        }
        avatar.snp.makeConstraints { make in
            make.leading.equalTo(check.snp.trailing).offset(12)
            make.centerY.equalToSuperview()
            make.size.equalTo(44)
        }
        name.snp.makeConstraints { make in
            make.leading.equalTo(avatar.snp.trailing).offset(12)
            make.centerY.equalToSuperview()
            make.trailing.equalToSuperview().inset(16)
        }
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        let key = filtered[indexPath.row].key
        if selected.contains(key) { selected.remove(key) } else { selected.insert(key) }
        tableView.reloadRows(at: [indexPath], with: .none)
        refreshSendTitle()
        refreshSelectAll()
    }

    func searchBar(_ searchBar: UISearchBar, textDidChange searchText: String) {
        self.searchText = searchText
        applyFilter()
    }
}
