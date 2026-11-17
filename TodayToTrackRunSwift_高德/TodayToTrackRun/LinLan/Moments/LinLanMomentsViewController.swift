//
//  LinLanMomentsViewController.swift
//  TodayToTrackRun
//
//  朋友圈列表（对照 ShangLan TongHao / Square）
//

import SnapKit
import UIKit

final class LinLanMomentsViewController: UIViewController {
    private let segmented = UISegmentedControl(items: ["推荐", "最新"])
    private let tableView = UITableView(frame: .zero, style: .plain)
    private let emptyLabel = UILabel()
    private let refresh = UIRefreshControl()

    private var posts: [LinLanCommunityPost] = []
    private var feedType = "recommended"
    private var loading = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "朋友圈"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .compose,
            target: self,
            action: #selector(createPost)
        )
        navigationItem.rightBarButtonItem?.tintColor = AppTheme.accent

        segmented.selectedSegmentIndex = 0
        segmented.addTarget(self, action: #selector(feedChanged), for: .valueChanged)
        view.addSubview(segmented)
        segmented.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(8)
            make.leading.trailing.equalToSuperview().inset(16)
        }

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.separatorInset = UIEdgeInsets(top: 0, left: 16, bottom: 0, right: 16)
        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 120
        tableView.register(LinLanMomentCell.self, forCellReuseIdentifier: LinLanMomentCell.reuseID)
        refresh.addTarget(self, action: #selector(pullRefresh), for: .valueChanged)
        tableView.refreshControl = refresh
        view.addSubview(tableView)
        tableView.snp.makeConstraints { make in
            make.top.equalTo(segmented.snp.bottom).offset(8)
            make.leading.trailing.bottom.equalToSuperview()
        }

        emptyLabel.text = "暂无动态"
        emptyLabel.textColor = AppTheme.textSecondary
        emptyLabel.font = .systemFont(ofSize: 15)
        emptyLabel.textAlignment = .center
        emptyLabel.isHidden = true
        view.addSubview(emptyLabel)
        emptyLabel.snp.makeConstraints { $0.center.equalToSuperview() }

        Task { await loadFeed() }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        Task { await loadFeed() }
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }

    @objc private func feedChanged() {
        feedType = segmented.selectedSegmentIndex == 0 ? "recommended" : "latest"
        Task { await loadFeed() }
    }

    @objc private func pullRefresh() {
        Task { await loadFeed() }
    }

    @objc private func createPost() {
        let vc = LinLanCreateMomentViewController()
        vc.onPosted = { [weak self] in
            Task { await self?.loadFeed() }
        }
        present(UINavigationController(rootViewController: vc), animated: true)
    }

    private func loadFeed() async {
        guard !loading else { return }
        loading = true
        do {
            _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
            let list = try await LinLanAppContainer.apiClient().fetchCommunityFeed(feed: feedType)
            await MainActor.run {
                self.posts = list
                self.emptyLabel.isHidden = !list.isEmpty
                self.tableView.reloadData()
                self.refresh.endRefreshing()
                self.loading = false
            }
        } catch {
            await MainActor.run {
                self.refresh.endRefreshing()
                self.loading = false
                self.toast(error.localizedDescription)
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        a.addAction(UIAlertAction(title: "好", style: .default))
        present(a, animated: true)
    }
}

extension LinLanMomentsViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { posts.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(
            withIdentifier: LinLanMomentCell.reuseID,
            for: indexPath
        ) as! LinLanMomentCell
        cell.configure(posts[indexPath.row])
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let post = posts[indexPath.row]
        guard post.id > 0 else { return }
        linLanPush(LinLanMomentDetailViewController(postID: post.id))
    }
}

final class LinLanMomentCell: UITableViewCell {
    static let reuseID = "LinLanMomentCell"

    private let authorLabel = UILabel()
    private let titleLabel = UILabel()
    private let bodyLabel = UILabel()
    private let metaLabel = UILabel()
    private let thumb = UIImageView()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        selectionStyle = .default
        backgroundColor = .white
        authorLabel.font = .systemFont(ofSize: 13, weight: .medium)
        authorLabel.textColor = AppTheme.accent
        titleLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.numberOfLines = 2
        bodyLabel.font = .systemFont(ofSize: 14)
        bodyLabel.textColor = AppTheme.textSecondary
        bodyLabel.numberOfLines = 3
        metaLabel.font = .systemFont(ofSize: 12)
        metaLabel.textColor = AppTheme.textSecondary
        thumb.contentMode = .scaleAspectFill
        thumb.clipsToBounds = true
        thumb.layer.cornerRadius = 8
        thumb.backgroundColor = AppTheme.pageBackground
        thumb.isHidden = true

        let col = UIStackView(arrangedSubviews: [authorLabel, titleLabel, bodyLabel, metaLabel])
        col.axis = .vertical
        col.spacing = 4
        contentView.addSubview(col)
        contentView.addSubview(thumb)
        col.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().inset(14)
            make.bottom.equalToSuperview().inset(14)
            make.trailing.lessThanOrEqualTo(thumb.snp.leading).offset(-10)
        }
        thumb.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(14)
            make.centerY.equalToSuperview()
            make.size.equalTo(64)
        }
    }

    required init?(coder: NSCoder) { fatalError() }

    func configure(_ post: LinLanCommunityPost) {
        authorLabel.text = post.author?.displayName ?? "用户"
        let title = post.title.trimmingCharacters(in: .whitespacesAndNewlines)
        titleLabel.text = title.isEmpty ? "动态" : title
        titleLabel.isHidden = title.isEmpty
        bodyLabel.text = LinLanCommunityImagesHelper.displayContent(for: post)
        metaLabel.text = "赞 \(post.likeCount) · 评论 \(post.commentCount) · \(post.createdAt)"
        let images = LinLanCommunityImagesHelper.images(for: post)
        if let first = images.first {
            thumb.isHidden = false
            LinLanImageLoader.load(urlString: first, into: thumb)
        } else {
            thumb.isHidden = true
            thumb.image = nil
        }
    }
}
