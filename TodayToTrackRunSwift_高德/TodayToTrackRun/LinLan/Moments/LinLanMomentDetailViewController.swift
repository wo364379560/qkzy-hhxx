//
//  LinLanMomentDetailViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanMomentDetailViewController: UIViewController {
    private let postID: Int
    private let tableView = UITableView(frame: .zero, style: .grouped)
    private let likeButton = UIButton(type: .system)
    private let inputField = UITextField()
    private let sendButton = UIButton(type: .system)
    private let bottomBar = UIView()
    private var bottomBarBottom: Constraint?

    private var post: LinLanCommunityPost?
    private var comments: [LinLanCommunityComment] = []

    init(postID: Int) {
        self.postID = postID
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "动态详情"
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "ellipsis.circle"),
            style: .plain,
            target: self,
            action: #selector(more)
        )

        tableView.backgroundColor = AppTheme.pageBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.keyboardDismissMode = .interactive
        tableView.estimatedRowHeight = 56
        tableView.rowHeight = UITableView.automaticDimension
        view.addSubview(tableView)

        bottomBar.backgroundColor = .white
        let topLine = UIView()
        topLine.backgroundColor = AppTheme.divider
        bottomBar.addSubview(topLine)
        topLine.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
            make.height.equalTo(1 / UIScreen.main.scale)
        }

        likeButton.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
        likeButton.addTarget(self, action: #selector(toggleLike), for: .touchUpInside)
        inputField.placeholder = "写评论…"
        inputField.borderStyle = .roundedRect
        inputField.font = .systemFont(ofSize: 15)
        inputField.returnKeyType = .send
        inputField.delegate = self
        sendButton.setTitle("发送", for: .normal)
        sendButton.setTitleColor(.white, for: .normal)
        sendButton.backgroundColor = AppTheme.accent
        sendButton.layer.cornerRadius = 8
        sendButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .semibold)
        sendButton.addTarget(self, action: #selector(sendComment), for: .touchUpInside)

        bottomBar.addSubview(likeButton)
        bottomBar.addSubview(inputField)
        bottomBar.addSubview(sendButton)
        view.addSubview(bottomBar)

        bottomBar.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview()
            bottomBarBottom = make.bottom.equalTo(view.safeAreaLayoutGuide).constraint
            make.height.equalTo(56)
        }
        likeButton.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.centerY.equalToSuperview()
            make.width.equalTo(64)
        }
        // 发送固定 50pt 宽
        sendButton.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(12)
            make.centerY.equalToSuperview()
            make.width.equalTo(50)
            make.height.equalTo(36)
        }
        inputField.snp.makeConstraints { make in
            make.leading.equalTo(likeButton.snp.trailing).offset(8)
            make.trailing.equalTo(sendButton.snp.leading).offset(-8)
            make.centerY.equalToSuperview()
            make.height.equalTo(36)
        }
        tableView.snp.makeConstraints { make in
            make.top.leading.trailing.equalToSuperview()
            make.bottom.equalTo(bottomBar.snp.top)
        }

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(kbShow(_:)),
            name: UIResponder.keyboardWillShowNotification,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(kbHide(_:)),
            name: UIResponder.keyboardWillHideNotification,
            object: nil
        )

        Task { await load() }
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    @objc private func back() {
        view.endEditing(true)
        navigationController?.popViewController(animated: true)
    }

    @objc private func kbShow(_ n: Notification) {
        guard let frame = n.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect else { return }
        let duration = (n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? NSNumber)?.doubleValue ?? 0.25
        let curve = (n.userInfo?[UIResponder.keyboardAnimationCurveUserInfoKey] as? NSNumber)?.uintValue
            ?? UIView.AnimationOptions.curveEaseInOut.rawValue
        // 键盘顶相对当前 view；底部栏贴键盘，不再叠 safeArea
        let kbInView = view.convert(frame, from: nil)
        let overlap = max(0, view.bounds.maxY - kbInView.minY)
        let safeBottom = view.safeAreaInsets.bottom
        let offset = max(0, overlap - safeBottom)
        bottomBarBottom?.update(offset: -offset)
        UIView.animate(
            withDuration: duration,
            delay: 0,
            options: UIView.AnimationOptions(rawValue: curve << 16)
        ) {
            self.view.layoutIfNeeded()
        }
    }

    @objc private func kbHide(_ n: Notification) {
        let duration = (n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? NSNumber)?.doubleValue ?? 0.25
        let curve = (n.userInfo?[UIResponder.keyboardAnimationCurveUserInfoKey] as? NSNumber)?.uintValue
            ?? UIView.AnimationOptions.curveEaseInOut.rawValue
        bottomBarBottom?.update(offset: 0)
        UIView.animate(
            withDuration: duration,
            delay: 0,
            options: UIView.AnimationOptions(rawValue: curve << 16)
        ) {
            self.view.layoutIfNeeded()
        }
    }

    private func load() async {
        do {
            let result = try await LinLanAppContainer.apiClient().fetchCommunityPostDetail(postID: postID)
            await MainActor.run {
                self.post = result.0
                self.comments = result.1
                self.updateLike()
                self.tableView.reloadData()
            }
        } catch {
            await MainActor.run { toast(error.localizedDescription) }
        }
    }

    private func updateLike() {
        guard let post else {
            likeButton.setTitle("赞", for: .normal)
            return
        }
        let title = post.isLiked ? "已赞 \(post.likeCount)" : "赞 \(post.likeCount)"
        likeButton.setTitle(title, for: .normal)
        likeButton.setTitleColor(post.isLiked ? AppTheme.accent : AppTheme.textPrimary, for: .normal)
    }

    @objc private func toggleLike() {
        Task {
            do {
                let r = try await LinLanAppContainer.apiClient().toggleCommunityPostLike(postID: postID)
                await MainActor.run {
                    self.post?.isLiked = r.isLiked
                    self.post?.likeCount = r.likeCount
                    self.updateLike()
                    self.tableView.reloadSections(IndexSet(integer: 0), with: .none)
                }
            } catch {
                await MainActor.run { toast(error.localizedDescription) }
            }
        }
    }

    @objc private func sendComment() {
        let text = inputField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !text.isEmpty else { return }
        view.endEditing(true)
        Task {
            do {
                _ = try await LinLanAppContainer.apiClient().createCommunityComment(
                    postID: postID,
                    content: text
                )
                await MainActor.run { self.inputField.text = nil }
                await load()
            } catch {
                await MainActor.run { toast(error.localizedDescription) }
            }
        }
    }

    @objc private func more() {
        let sheet = UIAlertController(title: nil, message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "举报", style: .default) { [weak self] _ in
            self?.reportPost()
        })
        if post?.isMine == true {
            sheet.addAction(UIAlertAction(title: "删除动态", style: .destructive) { [weak self] _ in
                self?.deletePost()
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        if let pop = sheet.popoverPresentationController {
            pop.barButtonItem = navigationItem.rightBarButtonItem
        }
        present(sheet, animated: true)
    }

    private func reportPost() {
        view.endEditing(true)
        let name: String = {
            if let t = post?.title.trimmingCharacters(in: .whitespacesAndNewlines), !t.isEmpty {
                return t
            }
            return "社区动态"
        }()
        LinLanReportViewController.push(
            from: self,
            targetId: "\(postID)",
            targetType: "community_post",
            targetName: name
        )
    }

    private func deletePost() {
        Task {
            do {
                try await LinLanAppContainer.apiClient().deleteCommunityPost(postID: postID)
                await MainActor.run {
                    self.navigationController?.popViewController(animated: true)
                }
            } catch {
                await MainActor.run { toast(error.localizedDescription) }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        a.addAction(UIAlertAction(title: "好", style: .default))
        present(a, animated: true)
    }
}

extension LinLanMomentDetailViewController: UITableViewDataSource, UITableViewDelegate {
    func numberOfSections(in tableView: UITableView) -> Int { 2 }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        section == 0 ? 1 : max(comments.count, 1)
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        section == 1 ? "评论" : nil
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.backgroundColor = .white
        cell.selectionStyle = .none
        if indexPath.section == 0 {
            guard let post else {
                cell.textLabel?.text = "加载中…"
                return cell
            }
            cell.textLabel?.numberOfLines = 0
            cell.detailTextLabel?.numberOfLines = 0
            let title = post.title.isEmpty ? "动态" : post.title
            let body = LinLanCommunityImagesHelper.displayContent(for: post)
            let imgs = LinLanCommunityImagesHelper.images(for: post)
            cell.textLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
            cell.textLabel?.text = "\(post.author?.displayName ?? "用户")\n\(title)"
            var detail = body
            if !imgs.isEmpty { detail += "\n[\(imgs.count) 张图片]" }
            detail += "\n赞 \(post.likeCount) · \(post.createdAt)"
            cell.detailTextLabel?.text = detail
            cell.detailTextLabel?.textColor = AppTheme.textSecondary
        } else if comments.isEmpty {
            cell.textLabel?.text = "暂无评论"
            cell.textLabel?.textColor = AppTheme.textSecondary
        } else {
            let c = comments[indexPath.row]
            cell.textLabel?.font = .systemFont(ofSize: 15, weight: .medium)
            cell.textLabel?.text = c.author?.displayName ?? "用户"
            cell.detailTextLabel?.text = "\(c.content)\n\(c.createdAt)"
            cell.detailTextLabel?.textColor = AppTheme.textSecondary
            cell.detailTextLabel?.numberOfLines = 0
        }
        return cell
    }
}

extension LinLanMomentDetailViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        sendComment()
        return true
    }
}
