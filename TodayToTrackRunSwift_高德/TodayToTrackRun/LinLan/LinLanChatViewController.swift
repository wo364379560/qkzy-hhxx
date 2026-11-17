//
//  LinLanChatViewController.swift
//  TodayToTrackRun
//

import AVFoundation
import ImSDK_Plus_Swift
import PhotosUI
import SnapKit
import UniformTypeIdentifiers
import UIKit
import ZLPhotoBrowser

final class LinLanChatViewController: UIViewController {
    private var peerReadThrough: TimeInterval = 0
    private var peerReadIDs = Set<String>()
    private var quoteLookupTask: Task<Void, Never>?
    private let titleText: String
    private let userID: String?
    private let groupID: String?

    private let tableView = UITableView(frame: .zero, style: .plain)
    private let historyRefresh = UIRefreshControl()
    private let inputBar = UIView()
    private let quoteBar = UIView()
    private let quoteLabel = UILabel()
    private let textView = UITextView()
    private let inputPlaceholder = UILabel()
    private let sendButton = UIButton(type: .system)
    private let muteBar = UILabel()
    private let toolStack = UIStackView()
    private let composerStack = UIStackView()
    private let multiBar = UIStackView()
    private let mentionTable = UITableView(frame: .zero, style: .plain)
    private let emojiPanel = UIView()
    private let pinnedBar = UILabel()
    private let bottomStack = UIStackView()
    private let homeInsetView = UIView()
    private let newMessageButton = UIButton(type: .system)

    private var bottomStackBottom: Constraint?
    private var homeInsetHeight: Constraint?
    private var emojiHeight: Constraint?
    private var multiHeight: Constraint?
    private var muteHeight: Constraint?
    private var textHeight: Constraint?
    private var emojiVisible = false
    private var messages: [V2TIMMessage] = []
    private var oldestMsg: V2TIMMessage?
    private var hasMore = true
    private var loadingMore = false
    private var loadingInitial = false
    private var replyTarget: V2TIMMessage?
    private var multiSelect = false
    private var selectedIds = Set<String>()
    private var mentionCandidates: [LinLanGroupMemberItem] = []
    private var mentionVisible: [LinLanGroupMemberItem] = []
    private var mentionNameMap: [String: String] = [:]
    /// 微信式：麦克风切换「按住说话」模式
    private var voiceMode = false
    private var recording = false
    private var recordWillCancel = false
    private var holdTalkPressing = false
    private var recorder: AVAudioRecorder?
    private var recordURL: URL?
    private var recordStartedAt: Date?
    private var recordTimer: Timer?
    private weak var micButton: UIButton?
    /// ZL 相册必须强持有，否则回调/内部 SnapKit 布局会野指针崩溃
    private var photoPicker: ZLPhotoPicker?
    private let holdTalkButton = UIButton(type: .system)
    private let recordHUD = UIView()
    private let recordHUDIcon = UIImageView()
    private let recordHUDTip = UILabel()
    private let recordHUDTime = UILabel()
    private var selfGroupRole: UInt = 0
    private var memberRoleMap: [String: UInt] = [:]
    /// 群「允许查看成员资料」；拉取失败按禁止
    private var groupProfileAccessEnabled = true
    private var groupSendBlocked = false
    private var groupSendBlockedHint = "已禁言"
    private var pinCycleIndex = 0
    private var highlightedMsgId: String?
    private var pendingNewMessageCount = 0
    /// 群消息置顶以腾讯 IM 为唯一真源；WEB、iOS 和其他端共享。
    private var groupPinnedMessages: [V2TIMMessage] = []
    private static let maxMessagesInMemory = 280

    private var conversationId: String {
        LinLanImManager.conversationId(userID: userID, groupID: groupID)
    }

    private var isGroup: Bool { groupID?.isEmpty == false }

    /// 安全助手是系统通知会话，只允许查看消息，不提供发送能力。
    private var isSecurityAssistant: Bool {
        !isGroup && titleText.trimmingCharacters(in: .whitespacesAndNewlines) == "安全助手"
    }

    init(titleText: String, userID: String? = nil, groupID: String? = nil) {
        self.titleText = titleText
        self.userID = userID
        self.groupID = groupID
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = titleText
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(linLanBack)
        )
        let more = UIBarButtonItem(
            image: UIImage(systemName: "ellipsis"),
            style: .plain,
            target: self,
            action: #selector(openPeerDetail)
        )
        more.tintColor = AppTheme.accent
        if isGroup {
            navigationItem.rightBarButtonItem = more
        } else {
            let audio = UIBarButtonItem(
                image: UIImage(systemName: "phone"),
                style: .plain,
                target: self,
                action: #selector(startAudioCall)
            )
            let video = UIBarButtonItem(
                image: UIImage(systemName: "video"),
                style: .plain,
                target: self,
                action: #selector(startVideoCall)
            )
            audio.tintColor = AppTheme.accent
            video.tintColor = AppTheme.accent
            navigationItem.rightBarButtonItems = [more, video, audio]
        }
        navigationController?.navigationBar.tintColor = AppTheme.accent

        setupUI()
        bindKeyboard()
        NotificationCenter.default.addObserver(self, selector: #selector(onReadReceipts(_:)), name: .linLanDidReadMessages, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(markVisibleRead), name: UIApplication.didBecomeActiveNotification, object: nil)
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRecv(_:)),
            name: .linLanDidRecvMessage,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onRevoke(_:)),
            name: .linLanDidRevokeMessage,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onGroupSendPolicyChanged(_:)),
            name: .linLanGroupSendPolicyChanged,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onCurrentUserRemovedFromGroup(_:)),
            name: .linLanCurrentUserRemovedFromGroup,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onGroupPinnedMessagesChanged(_:)),
            name: .linLanGroupPinnedMessagesChanged,
            object: nil
        )

        Task {
            LinLanImManager.registerMessageListeners()
            await loadInitial()
            if isGroup {
                await preloadMembers()
                await refreshSendPermission()
                await reloadGroupPinnedMessages()
            } else {
                await MainActor.run { refreshPinnedBar() }
            }
        }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        // 进入或返回聊天页时不自动恢复键盘，只在用户直接点击输入框时聚焦。
        textView.resignFirstResponder()
        LinLanMessageAlertCenter.setActiveConversation(conversationId)
        if isGroup {
            Task {
                await verifyCurrentGroupMembership()
                await refreshSendPermission()
                await reloadGroupPinnedMessages()
            }
        }
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        if LinLanForwardJobs.shared.restore() { showForwardResults() }

        markVisibleRead()
        LinLanImManager.refreshReadReceipts(messages)
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        composerGeneration += 1
        quoteLookupTask?.cancel()
        if LinLanMessageAlertCenter.activeConversationId == conversationId {
            LinLanMessageAlertCenter.setActiveConversation(nil)
        }
    }

    deinit {
        recordTimer?.invalidate()
        quoteLookupTask?.cancel()
        NotificationCenter.default.removeObserver(self)
    }

    // MARK: - UI

    private func setupUI() {
        tableView.backgroundColor = AppTheme.pageBackground
        tableView.separatorStyle = .none
        tableView.keyboardDismissMode = .interactive
        tableView.alwaysBounceVertical = true
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(LinLanChatMessageCell.self, forCellReuseIdentifier: LinLanChatMessageCell.reuseID)
        tableView.estimatedRowHeight = 64
        tableView.rowHeight = UITableView.automaticDimension
        historyRefresh.tintColor = AppTheme.accent
        historyRefresh.attributedTitle = NSAttributedString(
            string: "下拉加载更早消息",
            attributes: [.font: UIFont.systemFont(ofSize: 13), .foregroundColor: AppTheme.textSecondary]
        )
        historyRefresh.addTarget(self, action: #selector(onPullLoadMore), for: .valueChanged)
        tableView.refreshControl = historyRefresh

        pinnedBar.font = .systemFont(ofSize: 13, weight: .medium)
        pinnedBar.textColor = AppTheme.accent
        pinnedBar.backgroundColor = AppTheme.accentSoft
        pinnedBar.numberOfLines = 1
        pinnedBar.isHidden = true
        pinnedBar.isUserInteractionEnabled = true
        pinnedBar.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(cyclePinnedLocate)))
        pinnedBar.addGestureRecognizer(UILongPressGestureRecognizer(target: self, action: #selector(longPressPinnedBar(_:))))

        newMessageButton.backgroundColor = .white
        newMessageButton.tintColor = AppTheme.accent
        newMessageButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .medium)
        newMessageButton.layer.cornerRadius = 18
        newMessageButton.layer.shadowColor = UIColor.black.cgColor
        newMessageButton.layer.shadowOpacity = 0.12
        newMessageButton.layer.shadowRadius = 6
        newMessageButton.layer.shadowOffset = CGSize(width: 0, height: 2)
        newMessageButton.contentEdgeInsets = UIEdgeInsets(top: 0, left: 14, bottom: 0, right: 14)
        newMessageButton.isHidden = true
        newMessageButton.addTarget(self, action: #selector(showNewestMessages), for: .touchUpInside)

        quoteBar.backgroundColor = UIColor(red: 0.93, green: 0.95, blue: 1, alpha: 1)
        quoteBar.isHidden = true
        quoteLabel.font = .systemFont(ofSize: 13)
        quoteLabel.textColor = AppTheme.textSecondary
        quoteLabel.numberOfLines = 1
        let cancelQuote = UIButton(type: .system)
        cancelQuote.setTitle("取消", for: .normal)
        cancelQuote.addTarget(self, action: #selector(clearReply), for: .touchUpInside)
        quoteBar.addSubview(quoteLabel)
        quoteBar.addSubview(cancelQuote)
        quoteLabel.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.centerY.equalToSuperview()
            make.trailing.lessThanOrEqualTo(cancelQuote.snp.leading).offset(-8)
        }
        cancelQuote.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(12)
            make.centerY.equalToSuperview()
        }
        inputBar.backgroundColor = .white
        textView.font = .systemFont(ofSize: 16)
        textView.backgroundColor = AppTheme.pageBackground
        textView.layer.cornerRadius = 8
        textView.delegate = self
        textView.isScrollEnabled = false
        textView.returnKeyType = .default
        textView.textContainer.maximumNumberOfLines = 0
        textView.textContainer.widthTracksTextView = true
        textView.textContainer.lineBreakMode = .byCharWrapping
        textView.textContainer.lineFragmentPadding = 4
        textView.textContainerInset = UIEdgeInsets(top: 10, left: 8, bottom: 10, right: 8)
        // 保留粘贴的换行
        textView.smartInsertDeleteType = .no
        inputPlaceholder.text = "请输入"
        inputPlaceholder.font = textView.font
        inputPlaceholder.textColor = .placeholderText
        inputPlaceholder.isUserInteractionEnabled = false
        textView.addSubview(inputPlaceholder)
        inputPlaceholder.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.top.equalToSuperview().offset(10)
        }
        sendButton.setTitle("发送", for: .normal)
        sendButton.setTitleColor(AppTheme.accent, for: .normal)
        sendButton.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        sendButton.addTarget(self, action: #selector(sendText), for: .touchUpInside)

        holdTalkButton.setTitle("按住 说话", for: .normal)
        holdTalkButton.setTitleColor(AppTheme.textPrimary, for: .normal)
        holdTalkButton.titleLabel?.font = .systemFont(ofSize: 16, weight: .medium)
        holdTalkButton.backgroundColor = UIColor(white: 0.94, alpha: 1)
        holdTalkButton.layer.cornerRadius = 8
        holdTalkButton.isHidden = true
        holdTalkButton.isExclusiveTouch = true
        let holdPress = UILongPressGestureRecognizer(target: self, action: #selector(holdTalkPress(_:)))
        holdPress.minimumPressDuration = 0.12
        holdPress.allowableMovement = .greatestFiniteMagnitude
        holdTalkButton.addGestureRecognizer(holdPress)

        toolStack.axis = .horizontal
        toolStack.distribution = .fillEqually
        toolStack.spacing = 4
        var toolItems: [(String, Selector)] = [
            ("mic.fill", #selector(toggleVoiceMode)),
            ("face.smiling", #selector(toggleEmoji)),
            ("photo", #selector(pickImage)),
            ("video", #selector(pickVideo)),
            ("paperclip", #selector(pickFile)),
        ]
        // 单聊没有群 @，不展示入口
        if isGroup {
            toolItems.append(("at", #selector(manualMention)))
        }
        for item in toolItems {
            let b = UIButton(type: .system)
            b.setImage(UIImage(systemName: item.0), for: .normal)
            b.tintColor = AppTheme.textSecondary
            b.addTarget(self, action: item.1, for: .touchUpInside)
            if item.0 == "mic.fill" { micButton = b }
            toolStack.addArrangedSubview(b)
        }

        multiBar.axis = .horizontal
        multiBar.distribution = .fillEqually
        multiBar.isHidden = true
        multiBar.backgroundColor = .white
        let cancelMulti = UIButton(type: .system)
        cancelMulti.setTitle("取消", for: .normal)
        cancelMulti.addTarget(self, action: #selector(exitMulti), for: .touchUpInside)
        let fwdOne = UIButton(type: .system)
        fwdOne.setTitle("逐条转发", for: .normal)
        fwdOne.addTarget(self, action: #selector(forwardSelectedOneByOne), for: .touchUpInside)
        let fwdMerge = UIButton(type: .system)
        fwdMerge.setTitle("合并转发", for: .normal)
        fwdMerge.addTarget(self, action: #selector(forwardSelectedMerged), for: .touchUpInside)
        [cancelMulti, fwdOne, fwdMerge].forEach {
            $0.setTitleColor(AppTheme.accent, for: .normal)
            multiBar.addArrangedSubview($0)
        }

        mentionTable.isHidden = true
        mentionTable.dataSource = self
        mentionTable.delegate = self
        mentionTable.layer.cornerRadius = 10
        mentionTable.layer.borderWidth = 1
        mentionTable.layer.borderColor = AppTheme.divider.cgColor
        mentionTable.rowHeight = 44
        mentionTable.register(UITableViewCell.self, forCellReuseIdentifier: "m")

        emojiPanel.backgroundColor = .white
        emojiPanel.clipsToBounds = true
        buildEmojiPanel()

        muteBar.textAlignment = .center
        muteBar.font = .systemFont(ofSize: 15, weight: .medium)
        muteBar.textColor = AppTheme.textSecondary
        muteBar.backgroundColor = .white
        muteBar.isHidden = true

        inputBar.addSubview(textView)
        inputBar.addSubview(sendButton)
        inputBar.addSubview(holdTalkButton)
        toolStack.isLayoutMarginsRelativeArrangement = true
        toolStack.layoutMargins = UIEdgeInsets(top: 0, left: 8, bottom: 0, right: 8)

        composerStack.axis = .vertical
        composerStack.spacing = 0
        composerStack.backgroundColor = .white
        composerStack.addArrangedSubview(toolStack)
        composerStack.addArrangedSubview(inputBar)

        homeInsetView.backgroundColor = .white

        bottomStack.axis = .vertical
        bottomStack.spacing = 0
        bottomStack.addArrangedSubview(quoteBar)
        bottomStack.addArrangedSubview(muteBar)
        bottomStack.addArrangedSubview(composerStack)
        bottomStack.addArrangedSubview(emojiPanel)
        bottomStack.addArrangedSubview(multiBar)
        bottomStack.addArrangedSubview(homeInsetView)
        bottomStack.backgroundColor = .white

        view.addSubview(pinnedBar)
        view.addSubview(tableView)
        view.addSubview(mentionTable)
        view.addSubview(bottomStack)
        view.addSubview(newMessageButton)

        pinnedBar.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide)
            make.leading.trailing.equalToSuperview()
            make.height.equalTo(0)
        }
        // 贴屏幕底，白色延伸到 Home Indicator；安全区用 homeInsetView 垫高
        bottomStack.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview()
            bottomStackBottom = make.bottom.equalToSuperview().constraint
        }
        homeInsetView.snp.makeConstraints { make in
            homeInsetHeight = make.height.equalTo(0).constraint
        }
        
        toolStack.snp.makeConstraints { $0.height.equalTo(36) }
        inputBar.setContentHuggingPriority(.required, for: .vertical)
        inputBar.setContentCompressionResistancePriority(.required, for: .vertical)
        emojiPanel.snp.makeConstraints { make in
            emojiHeight = make.height.equalTo(0).constraint
        }
        multiBar.snp.makeConstraints { make in
            multiHeight = make.height.equalTo(0).constraint
        }
        muteBar.snp.makeConstraints { make in
            muteHeight = make.height.equalTo(0).constraint
        }
        quoteBar.snp.makeConstraints { $0.height.equalTo(0) }
        tableView.snp.makeConstraints { make in
            make.top.equalTo(pinnedBar.snp.bottom)
            make.leading.trailing.equalToSuperview()
            make.bottom.equalTo(bottomStack.snp.top)
        }
        textView.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.top.equalToSuperview().offset(8)
            make.bottom.equalToSuperview().inset(8)
            make.trailing.equalTo(sendButton.snp.leading).offset(-8)
            textHeight = make.height.equalTo(44).constraint
        }
        sendButton.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(12)
            make.centerY.equalTo(textView)
            make.width.equalTo(48)
        }
        holdTalkButton.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.trailing.equalToSuperview().inset(12)
            make.top.equalToSuperview().offset(8)
            make.bottom.equalToSuperview().inset(8)
            make.height.equalTo(44)
        }
        mentionTable.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview().inset(12)
            make.bottom.equalTo(bottomStack.snp.top).offset(-4)
            make.height.equalTo(0)
        }
        newMessageButton.snp.makeConstraints { make in
            make.trailing.equalToSuperview().inset(16)
            make.bottom.equalTo(bottomStack.snp.top).offset(-12)
            make.height.equalTo(36)
        }

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissComposer))
        tap.cancelsTouchesInView = false
        tableView.addGestureRecognizer(tap)
        setupRecordHUD()
        view.bringSubviewToFront(recordHUD)

        if isSecurityAssistant {
            applySecurityAssistantReadOnlyUI()
        }
    }

    private func applySecurityAssistantReadOnlyUI() {
        textView.resignFirstResponder()
        replyTarget = nil
        quoteBar.isHidden = true
        muteBar.isHidden = true
        composerStack.isHidden = true
        inputBar.isHidden = true
        toolStack.isHidden = true
        emojiPanel.isHidden = true
        multiBar.isHidden = true
        homeInsetView.isHidden = true
        emojiHeight?.update(offset: 0)
        multiHeight?.update(offset: 0)
        muteHeight?.update(offset: 0)
        homeInsetHeight?.update(offset: 0)
        quoteBar.snp.updateConstraints { $0.height.equalTo(0) }
        bottomStack.isHidden = true
        bottomStack.isUserInteractionEnabled = false
    }

    private func buildEmojiPanel() {
        let emojis = [
            "😀", "😁", "😂", "🤣", "😊", "😍", "😘", "😎",
            "😢", "😭", "😡", "👍", "🙏", "🎉", "❤️", "🔥",
            "✨", "💪", "🤝", "👏",
            // 新增 14 个
            "🤔", "😴", "😷", "🤩", "😅", "🙃", "😇", "🥰",
            "👌", "✌️", "👀", "💯", "⭐", "🎈"
        ]
        let scroll = UIScrollView()
        scroll.showsVerticalScrollIndicator = false
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 10
        var row: UIStackView?
        for (i, e) in emojis.enumerated() {
            if i % 8 == 0 {
                row = UIStackView()
                row?.axis = .horizontal
                row?.distribution = .fillEqually
                stack.addArrangedSubview(row!)
            }
            let b = UIButton(type: .system)
            b.setTitle(e, for: .normal)
            b.titleLabel?.font = .systemFont(ofSize: 28)
            b.addAction(UIAction { [weak self] _ in
                self?.textView.insertText(e)
                self?.adjustInputHeight()
            }, for: .touchUpInside)
            row?.addArrangedSubview(b)
        }
        emojiPanel.addSubview(scroll)
        scroll.addSubview(stack)
        scroll.snp.makeConstraints { $0.edges.equalToSuperview() }
        stack.snp.makeConstraints { make in
            make.edges.equalToSuperview().inset(UIEdgeInsets(top: 12, left: 12, bottom: 16, right: 12))
            make.width.equalTo(scroll.snp.width).offset(-24)
        }
    }

    // MARK: - History

    private func loadInitial() async {
        guard !loadingInitial else { return }
        loadingInitial = true
        do {
            _ = try? await LinLanAppContainer.authRepository().ensureSessionReady()
            if isGroup, let gid = groupID {
                if let info = try? await LinLanImManager.fetchGroupInfo(groupId: gid) {
                    selfGroupRole = info.role
                }
            }
            let raw = try await LinLanImManager.fetchHistory(
                conversationId: conversationId,
                count: LinLanImManager.historyPageSize,
                lastMsg: nil
            )
            oldestMsg = raw.last
            hasMore = raw.count >= LinLanImManager.historyPageSize
            let list = raw.reversed().filter {
                !LinLanImManager.shouldSilentlyHideRevokedMessage($0) &&
                    !LinLanImManager.shouldHideRtcSignalingInChat($0) &&
                    !LinLanImManager.shouldHideGroupRemovalTip($0, viewerRole: self.selfGroupRole)
            }
            await MainActor.run {
                self.messages = Array(list).filter { !LinLanImManager.shouldSilentlyHideRevokedMessage($0) }
                self.trimMessagesIfNeeded(keepTail: true)
                self.tableView.reloadData()
                self.scrollToBottom(animated: false)
                self.updateHistoryRefreshState()
                self.loadingInitial = false
            }
            // 内容不足一屏时继续往前补，避免「只有 30 条且拉不动」
            markVisibleRead()
            LinLanImManager.refreshReadReceipts(messages)
            await fillHistoryIfContentShort()
        } catch {
            await MainActor.run {
                self.loadingInitial = false
                self.updateHistoryRefreshState()
                self.toast(error.localizedDescription)
            }
        }
    }

    @objc private func onPullLoadMore() {
        Task { await loadMore() }
    }

    private func loadMore() async {
        guard hasMore, !loadingMore, let oldest = oldestMsg else {
            await MainActor.run {
                if self.historyRefresh.isRefreshing { self.historyRefresh.endRefreshing() }
                self.updateHistoryRefreshState()
            }
            return
        }
        loadingMore = true
        do {
            let raw = try await LinLanImManager.fetchHistory(
                conversationId: conversationId,
                count: LinLanImManager.historyPageSize,
                lastMsg: oldest
            )
            if raw.isEmpty || raw.count < LinLanImManager.historyPageSize {
                hasMore = false
            }
            if let last = raw.last { oldestMsg = last }
            let older = Array(raw.reversed()).filter {
                !LinLanImManager.shouldSilentlyHideRevokedMessage($0) &&
                    !LinLanImManager.shouldHideRtcSignalingInChat($0) &&
                    !LinLanImManager.shouldHideGroupRemovalTip($0, viewerRole: self.selfGroupRole)
            }
            await MainActor.run {
                let oldHeight = self.tableView.contentSize.height
                let oldOffset = self.tableView.contentOffset.y
                if !older.isEmpty {
                    self.messages.insert(contentsOf: older.filter { !LinLanImManager.shouldSilentlyHideRevokedMessage($0) }, at: 0)
                    self.trimMessagesIfNeeded(keepTail: false)
                    self.tableView.reloadData()
                    self.tableView.layoutIfNeeded()
                    let newHeight = self.tableView.contentSize.height
                    self.tableView.contentOffset = CGPoint(x: 0, y: oldOffset + (newHeight - oldHeight))
                }
                self.loadingMore = false
                if self.historyRefresh.isRefreshing { self.historyRefresh.endRefreshing() }
                self.updateHistoryRefreshState()
            }
        } catch {
            await MainActor.run {
                self.loadingMore = false
                if self.historyRefresh.isRefreshing { self.historyRefresh.endRefreshing() }
                self.updateHistoryRefreshState()
                self.toast(error.localizedDescription)
            }
        }
    }

    /// 首屏消息偏少、列表滚不动时，自动再拉几页更早记录
    private func fillHistoryIfContentShort() async {
        for _ in 0..<4 {
            let needMore = await MainActor.run { () -> Bool in
                self.tableView.layoutIfNeeded()
                guard self.hasMore else { return false }
                let visible = self.tableView.bounds.height
                guard visible > 1 else { return false }
                return self.tableView.contentSize.height < visible + 80
            }
            guard needMore else { break }
            await loadMore()
        }
    }

    private func updateHistoryRefreshState() {
        if hasMore {
            historyRefresh.attributedTitle = NSAttributedString(
                string: "下拉加载更早消息",
                attributes: [.font: UIFont.systemFont(ofSize: 13), .foregroundColor: AppTheme.textSecondary]
            )
        } else {
            historyRefresh.attributedTitle = NSAttributedString(
                string: "没有更多消息了",
                attributes: [.font: UIFont.systemFont(ofSize: 13), .foregroundColor: AppTheme.textSecondary]
            )
        }
    }

    /// 内存封顶：只保留最近 N 条，避免社群刷屏 OOM
    private func trimMessagesIfNeeded(keepTail: Bool) {
        let maxN = Self.maxMessagesInMemory
        guard messages.count > maxN else { return }
        if keepTail {
            messages = Array(messages.suffix(maxN))
            oldestMsg = messages.first
            hasMore = true
        } else {
            // 正在看历史：丢掉较新的尾部，保留刚加载的更早消息
            let drop = messages.count - maxN
            if drop > 0 {
                messages.removeLast(drop)
            }
        }
    }

    private func scrollToBottom(animated: Bool) {
        guard !messages.isEmpty else { return }
        let ip = IndexPath(row: messages.count - 1, section: 0)
        tableView.scrollToRow(at: ip, at: .bottom, animated: animated)
        clearNewMessageIndicator()
    }

    private func isNearBottom(threshold: CGFloat = 88) -> Bool {
        tableView.layoutIfNeeded()
        let visibleBottom = tableView.contentOffset.y + tableView.bounds.height - tableView.adjustedContentInset.bottom
        return tableView.contentSize.height - visibleBottom <= threshold
    }

    private func viewportAnchor() -> (messageID: String, offset: CGFloat)? {
        guard let row = tableView.indexPathsForVisibleRows?.min(), messages.indices.contains(row.row) else { return nil }
        return (messages[row.row].msgID, tableView.rectForRow(at: row).minY - tableView.contentOffset.y)
    }

    private func restoreViewportAnchor(_ anchor: (messageID: String, offset: CGFloat)?) {
        guard let anchor, let row = messages.firstIndex(where: { $0.msgID == anchor.messageID }) else { return }
        tableView.layoutIfNeeded()
        let y = tableView.rectForRow(at: IndexPath(row: row, section: 0)).minY - anchor.offset
        tableView.setContentOffset(CGPoint(x: 0, y: y), animated: false)
    }

    private func refreshNewMessageIndicator() {
        newMessageButton.setTitle("\(pendingNewMessageCount) 条新消息  ↓", for: .normal)
        newMessageButton.isHidden = pendingNewMessageCount == 0
    }

    private func clearNewMessageIndicator() {
        guard pendingNewMessageCount != 0 || !newMessageButton.isHidden else { return }
        pendingNewMessageCount = 0
        newMessageButton.isHidden = true
    }

    @objc private func showNewestMessages() {
        scrollToBottom(animated: true)
    }

    // MARK: - Keyboard

    private func bindKeyboard() {
        NotificationCenter.default.addObserver(self, selector: #selector(kbShow(_:)), name: UIResponder.keyboardWillShowNotification, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(kbHide(_:)), name: UIResponder.keyboardWillHideNotification, object: nil)
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        if isSecurityAssistant {
            homeInsetHeight?.update(offset: 0)
            return
        }
        // 键盘弹起时不垫 Home Indicator，避免双重留白
        let kbUp = (bottomStackBottom?.layoutConstraints.first?.constant ?? 0) < -1
        homeInsetHeight?.update(offset: kbUp ? 0 : view.safeAreaInsets.bottom)
    }

    @objc private func kbShow(_ n: Notification) {
        guard
            let frame = n.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect,
            let duration = n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? Double
        else { return }
        let shouldFollowNewest = isNearBottom()
        if emojiVisible { setEmojiVisible(false, animated: false) }
        // bottomStack 贴屏幕底，直接用键盘高度上推
        let height = max(0, frame.height)
        homeInsetHeight?.update(offset: 0)
        bottomStackBottom?.update(offset: -height)
        UIView.animate(withDuration: duration) {
            self.view.layoutIfNeeded()
        }
        if shouldFollowNewest { scrollToBottom(animated: true) }
    }

    @objc private func kbHide(_ n: Notification) {
        let duration = (n.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? Double) ?? 0.25
        if emojiVisible { return }
        bottomStackBottom?.update(offset: 0)
        homeInsetHeight?.update(offset: view.safeAreaInsets.bottom)
        UIView.animate(withDuration: duration) {
            self.view.layoutIfNeeded()
        }
    }

    @objc private func dismissComposer() {
        textView.resignFirstResponder()
        setEmojiVisible(false, animated: true)
    }

    private func setEmojiVisible(_ show: Bool, animated: Bool) {
        emojiVisible = show
        emojiHeight?.update(offset: show ? 240 : 0)
        if show {
            bottomStackBottom?.update(offset: 0)
        }
        let updates = { self.view.layoutIfNeeded() }
        if animated {
            UIView.animate(withDuration: 0.25, animations: updates)
        } else {
            updates()
        }
    }

    // MARK: - Realtime

    @objc private func onRecv(_ n: Notification) {
        guard let msg = n.object as? V2TIMMessage else { return }
        if isGroup, LinLanImManager.isSameGroup(msg.groupID, groupID) {
            Task { await self.refreshSendPermission() }
        }
        guard LinLanImManager.belongsToConversation(msg, userID: userID, groupID: groupID) else { return }
        if LinLanImManager.shouldSilentlyHideRevokedMessage(msg) { return }
        if LinLanImManager.shouldHideRtcSignalingInChat(msg) { return }
        if LinLanImManager.shouldHideGroupRemovalTip(msg, viewerRole: selfGroupRole) { return }
        if messages.contains(where: { $0.msgID == msg.msgID }) { return }
        appendLocal(msg)
        markVisibleRead()
    }

    @objc private func markVisibleRead() {
        guard isViewLoaded, view.window != nil, navigationController?.topViewController === self,
              UIApplication.shared.applicationState == .active else { return }
        LinLanImManager.markChatRead(userID: userID, groupID: groupID, messages: messages)
    }

    @objc private func onReadReceipts(_ note: Notification) {
        guard !isGroup, let receipts = note.object as? [V2TIMMessageReceipt] else { return }
        let legacy = note.userInfo?["legacy"] as? Bool ?? false
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            for receipt in receipts where receipt.userID == self.userID {
                if legacy { self.peerReadThrough = max(self.peerReadThrough, TimeInterval(receipt.timestamp)) }
                else if receipt.isPeerRead, let id = receipt.msgID { self.peerReadIDs.insert(id) }
            }
            self.tableView.reloadData()
        }
    }

    @objc private func onGroupSendPolicyChanged(_ n: Notification) {
        guard isGroup else { return }
        let gid = (n.object as? String) ?? ""
        guard LinLanImManager.isSameGroup(gid, groupID) else { return }
        Task { await refreshSendPermission() }
    }

    @objc private func onCurrentUserRemovedFromGroup(_ n: Notification) {
        guard let gid = n.object as? String,
              LinLanImManager.isSameGroup(gid, groupID) else { return }
        exitRemovedGroupChat()
    }

    @objc private func onGroupPinnedMessagesChanged(_ n: Notification) {
        guard let gid = n.object as? String,
              LinLanImManager.isSameGroup(gid, groupID) else { return }
        Task { await reloadGroupPinnedMessages() }
    }

    @objc private func onRevoke(_ n: Notification) {
        guard let msgId = n.object as? String, !msgId.isEmpty else { return }
        let owner = LinLanImManager.recallScope()
        Task { @MainActor [weak self] in
            guard let self else { return }
            let found = try? await LinLanImManager.findMessages([msgId])
            guard LinLanImManager.recalled.isCurrent(owner) else { return }
            let idx = self.messages.firstIndex(where: { $0.msgID == msgId })
            let updated = found?.first ?? idx.map { self.messages[$0] }
            if let updated, !LinLanImManager.shouldSilentlyHideRevokedMessage(updated) {
                if let idx { self.messages[idx] = updated }
            } else if let idx { self.messages.remove(at: idx) }
            if self.replyTarget?.msgID == msgId { self.clearReply() }
            self.selectedIds.remove(msgId)
            self.tableView.reloadData()
            self.refreshPinnedBar()
        }
    }

    // MARK: - Send

    @objc private func sendText() {
        guard !isSecurityAssistant else { return }
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        // 只去掉首尾空白，完整保留中间换行（含空行）
        let text = Self.normalizedOutgoingText(textView.text)
        guard !text.isEmpty else { return }
        let atIds = parseAtUserIds(from: text)
        let cloud = replyTarget.flatMap {
            LinLanQuoteHelper.embedQuoteCloudCustomData(
                quoted: $0,
                senderName: LinLanQuoteHelper.senderName($0)
            )
        }
        textView.text = ""
        refreshInputPlaceholder()
        adjustInputHeight()
        clearReply()
        Task {
            do {
                let msg = try await LinLanImManager.sendTextMessage(
                    text: text,
                    receiver: userID,
                    groupId: groupID,
                    cloudCustomData: cloud,
                    atUserIds: atIds
                )
                await MainActor.run { self.appendLocal(msg) }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    /// 统一换行符为 `\n`，仅裁首尾空白，不压扁多行格式
    private static func normalizedOutgoingText(_ raw: String?) -> String {
        guard let raw else { return "" }
        return raw
            .replacingOccurrences(of: "\r\n", with: "\n")
            .replacingOccurrences(of: "\r", with: "\n")
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func adjustInputHeight() {
        let minH: CGFloat = 44
        let maxH: CGFloat = 110
        let width = textView.bounds.width > 1
            ? textView.bounds.width
            : view.bounds.width - 12 - 48 - 12 - 8
        let fitting = textView.sizeThatFits(CGSize(width: width, height: .greatestFiniteMagnitude))
        let h = min(maxH, max(minH, ceil(fitting.height)))
        textView.isScrollEnabled = fitting.height > maxH + 0.5
        textHeight?.update(offset: h)
        view.layoutIfNeeded()
    }

    private func refreshInputPlaceholder() {
        inputPlaceholder.isHidden = !textView.text.isEmpty
    }

    private func appendLocal(_ msg: V2TIMMessage) {
        if LinLanImManager.shouldSilentlyHideRevokedMessage(msg) { return }
        if LinLanImManager.shouldHideRtcSignalingInChat(msg) { return }
        if LinLanImManager.shouldHideGroupRemovalTip(msg, viewerRole: selfGroupRole) { return }
        if messages.contains(where: { $0.msgID == msg.msgID }) { return }
        let shouldFollowNewest = msg.isSelf || isNearBottom()
        let anchor = shouldFollowNewest ? nil : viewportAnchor()
        messages.append(msg)
        // 阅读历史时不丢当前可视锚点；回到底部后的下一条消息会恢复内存上限。
        if shouldFollowNewest { trimMessagesIfNeeded(keepTail: true) }
        tableView.reloadData()
        if shouldFollowNewest {
            scrollToBottom(animated: true)
        } else {
            restoreViewportAnchor(anchor)
            pendingNewMessageCount += 1
            refreshNewMessageIndicator()
        }
    }

    private func parseAtUserIds(from text: String) -> [String] {
        guard isGroup else { return [] }
        var ids: [String] = []
        if text.contains("@\(LinLanImManager.atAllDisplay)") {
            ids.append(LinLanImManager.atAllTag)
        }
        for (name, uid) in mentionNameMap where name != LinLanImManager.atAllDisplay {
            if text.contains("@\(name)") { ids.append(uid) }
        }
        return Array(Set(ids))
    }

    // MARK: - Media / voice / emoji / file

    @objc private func toggleEmoji() {
        if voiceMode {
            voiceMode = false
            applyVoiceModeUI()
        }
        if emojiVisible {
            setEmojiVisible(false, animated: true)
            return
        }
        textView.resignFirstResponder()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.05) { [weak self] in
            self?.setEmojiVisible(true, animated: true)
        }
    }

    // MARK: - 语音（微信经典：按住说话 / 上滑取消）

    private func setupRecordHUD() {
        recordHUD.isHidden = true
        recordHUD.backgroundColor = UIColor.black.withAlphaComponent(0.72)
        recordHUD.layer.cornerRadius = 12
        recordHUD.clipsToBounds = true

        recordHUDIcon.image = UIImage(systemName: "mic.fill")
        recordHUDIcon.tintColor = .white
        recordHUDIcon.contentMode = .scaleAspectFit

        recordHUDTip.text = "手指上滑，取消发送"
        recordHUDTip.font = .systemFont(ofSize: 13, weight: .medium)
        recordHUDTip.textColor = .white
        recordHUDTip.textAlignment = .center
        recordHUDTip.numberOfLines = 2

        recordHUDTime.text = "0\""
        recordHUDTime.font = .monospacedDigitSystemFont(ofSize: 15, weight: .medium)
        recordHUDTime.textColor = UIColor.white.withAlphaComponent(0.9)
        recordHUDTime.textAlignment = .center

        view.addSubview(recordHUD)
        [recordHUDIcon, recordHUDTime, recordHUDTip].forEach { recordHUD.addSubview($0) }
        recordHUD.snp.makeConstraints { make in
            make.center.equalToSuperview()
            make.width.equalTo(150)
            make.height.equalTo(150)
        }
        recordHUDIcon.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(28)
            make.centerX.equalToSuperview()
            make.size.equalTo(44)
        }
        recordHUDTime.snp.makeConstraints { make in
            make.top.equalTo(recordHUDIcon.snp.bottom).offset(8)
            make.leading.trailing.equalToSuperview().inset(8)
        }
        recordHUDTip.snp.makeConstraints { make in
            make.leading.trailing.equalToSuperview().inset(10)
            make.bottom.equalToSuperview().inset(14)
        }
    }

    @objc private func toggleVoiceMode() {
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        if recording { return }
        voiceMode.toggle()
        applyVoiceModeUI()
    }

    private var composerGeneration = 0

    private func applyVoiceModeUI() {
        composerGeneration += 1
        if voiceMode {
            textView.resignFirstResponder()
            setEmojiVisible(false, animated: true)
            textView.isHidden = true
            sendButton.isHidden = true
            holdTalkButton.isHidden = false
            holdTalkButton.setTitle("按住 说话", for: .normal)
            holdTalkButton.backgroundColor = UIColor(white: 0.94, alpha: 1)
            micButton?.setImage(UIImage(systemName: "keyboard"), for: .normal)
            micButton?.tintColor = AppTheme.accent
            textHeight?.update(offset: 44)
        } else {
            holdTalkButton.isHidden = true
            textView.isHidden = false
            sendButton.isHidden = false
            micButton?.setImage(UIImage(systemName: "mic.fill"), for: .normal)
            micButton?.tintColor = AppTheme.textSecondary
            adjustInputHeight()
        }
    }

    @objc private func holdTalkPress(_ g: UILongPressGestureRecognizer) {
        switch g.state {
        case .began:
            holdTalkPressing = true
            startHoldRecord()
        case .changed:
            guard recording else { return }
            let loc = g.location(in: holdTalkButton)
            // 手指上滑离开按钮区域 → 取消
            let cancel = loc.y < -50
            updateRecordCancelState(cancel)
        case .ended:
            holdTalkPressing = false
            guard recording else { return }
            stopRecord(send: !recordWillCancel)
        case .cancelled, .failed:
            holdTalkPressing = false
            if recording { stopRecord(send: false) }
        default:
            break
        }
    }

    private func startHoldRecord() {
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        let generation = composerGeneration
        AVAudioSession.sharedInstance().requestRecordPermission { [weak self] ok in
            guard let self else { return }
            guard ok else {
                DispatchQueue.main.async { self.toast("需要麦克风权限") }
                return
            }
            DispatchQueue.main.async {
                guard self.composerGeneration == generation, self.view.window != nil, self.voiceMode, self.holdTalkPressing, !self.recording else { return }
                do {
                    let session = AVAudioSession.sharedInstance()
                    try session.setCategory(
                        .playAndRecord,
                        mode: .default,
                        options: [.defaultToSpeaker, .allowBluetooth]
                    )
                    try session.setActive(true)
                    // 录到 Caches，避免临时目录被清后无法回放本地文件
                    let dir = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)[0]
                        .appendingPathComponent("linlan_voice", isDirectory: true)
                    try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
                    let url = dir.appendingPathComponent("voice_\(Int(Date().timeIntervalSince1970)).m4a")
                    let settings: [String: Any] = [
                        AVFormatIDKey: Int(kAudioFormatMPEG4AAC),
                        AVSampleRateKey: 44100,
                        AVNumberOfChannelsKey: 1,
                        AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue,
                    ]
                    let recorder = try AVAudioRecorder(url: url, settings: settings)
                    recorder.prepareToRecord()
                    recorder.record()
                    self.recorder = recorder
                    self.recordURL = url
                    self.recordStartedAt = Date()
                    self.recording = true
                    self.recordWillCancel = false
                    self.holdTalkButton.setTitle("松开 结束", for: .normal)
                    self.holdTalkButton.backgroundColor = UIColor(white: 0.88, alpha: 1)
                    self.showRecordHUD()
                    self.startRecordTimer()
                    UIImpactFeedbackGenerator(style: .light).impactOccurred()
                } catch {
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    private func stopRecord(send: Bool) {
        guard recording else { return }
        recording = false
        recordTimer?.invalidate()
        recordTimer = nil
        recorder?.stop()
        // 录完释放录音会话，避免后续播放被 category 选项卡死
        try? AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)
        let duration = Int(Date().timeIntervalSince(recordStartedAt ?? Date()))
        let url = recordURL
        recorder = nil
        recordURL = nil
        recordStartedAt = nil
        recordWillCancel = false
        holdTalkButton.setTitle("按住 说话", for: .normal)
        holdTalkButton.backgroundColor = UIColor(white: 0.94, alpha: 1)
        hideRecordHUD()

        guard send, let url, duration >= 1 else {
            if let url { try? FileManager.default.removeItem(at: url) }
            if send, duration < 1 { toast("说话时间太短") }
            return
        }
        Task {
            do {
                let msg = try await LinLanImManager.sendSoundMessage(
                    path: url.path,
                    durationSec: max(duration, 1),
                    receiver: userID,
                    groupId: groupID
                )
                await MainActor.run { self.appendLocal(msg) }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private func updateRecordCancelState(_ cancel: Bool) {
        guard recordWillCancel != cancel else { return }
        recordWillCancel = cancel
        if cancel {
            recordHUD.backgroundColor = UIColor.systemRed.withAlphaComponent(0.85)
            recordHUDIcon.image = UIImage(systemName: "arrow.uturn.backward")
            recordHUDTip.text = "松开手指，取消发送"
            holdTalkButton.setTitle("松开 取消", for: .normal)
        } else {
            recordHUD.backgroundColor = UIColor.black.withAlphaComponent(0.72)
            recordHUDIcon.image = UIImage(systemName: "mic.fill")
            recordHUDTip.text = "手指上滑，取消发送"
            holdTalkButton.setTitle("松开 结束", for: .normal)
        }
    }

    private func showRecordHUD() {
        recordHUDIcon.image = UIImage(systemName: "mic.fill")
        recordHUDTip.text = "手指上滑，取消发送"
        recordHUDTime.text = "0\""
        recordHUD.backgroundColor = UIColor.black.withAlphaComponent(0.72)
        recordHUD.isHidden = false
        recordHUD.alpha = 0
        recordHUD.transform = CGAffineTransform(scaleX: 0.9, y: 0.9)
        view.bringSubviewToFront(recordHUD)
        UIView.animate(withDuration: 0.15) {
            self.recordHUD.alpha = 1
            self.recordHUD.transform = .identity
        }
    }

    private func hideRecordHUD() {
        UIView.animate(withDuration: 0.12, animations: {
            self.recordHUD.alpha = 0
        }, completion: { _ in
            self.recordHUD.isHidden = true
            self.recordHUD.transform = .identity
        })
    }

    private func startRecordTimer() {
        recordTimer?.invalidate()
        recordTimer = Timer.scheduledTimer(withTimeInterval: 0.2, repeats: true) { [weak self] _ in
            guard let self, let start = self.recordStartedAt else { return }
            let sec = Int(Date().timeIntervalSince(start))
            self.recordHUDTime.text = "\(sec)\""
            if sec >= 60 { self.stopRecord(send: !self.recordWillCancel) }
        }
    }

    @objc private func pickImage() {
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        // 收起键盘/表情，避免系统相册弹层时页面上顶
        textView.resignFirstResponder()
        setEmojiVisible(false, animated: false)
        if voiceMode {
            voiceMode = false
            applyVoiceModeUI()
        }
        photoPicker = LinLanZLPhotoPickerHelper.presentImages(from: self, maxCount: 9) { [weak self] results in
            guard let self else { return }
            self.photoPicker = nil
            Task { await self.sendPickedImages(results) }
        }
        photoPicker?.cancelBlock = { [weak self] in self?.photoPicker = nil }
    }

    @objc private func pickVideo() {
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        textView.resignFirstResponder()
        setEmojiVisible(false, animated: false)
        if voiceMode {
            voiceMode = false
            applyVoiceModeUI()
        }
        photoPicker = LinLanZLPhotoPickerHelper.presentVideo(from: self) { [weak self] result in
            guard let self else { return }
            self.photoPicker = nil
            Task { await self.sendPickedVideo(result) }
        }
        photoPicker?.cancelBlock = { [weak self] in self?.photoPicker = nil }
    }

    private func sendPickedImages(_ results: [ZLResultModel]) async {
        var failCount = 0
        for item in results {
            guard let path = LinLanZLPhotoPickerHelper.saveImage(item.image) else {
                failCount += 1
                continue
            }
            do {
                let msg = try await LinLanImManager.sendImageMessage(
                    path: path,
                    receiver: userID,
                    groupId: groupID
                )
                await MainActor.run { self.appendLocal(msg) }
            } catch {
                failCount += 1
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
        if failCount > 0, failCount == results.count {
            await MainActor.run { self.toast("图片发送失败") }
        }
    }

    private func sendPickedVideo(_ result: ZLResultModel) async {
        await MainActor.run { self.toast("视频处理中…") }
        guard let exported = await LinLanZLPhotoPickerHelper.exportVideoPath(from: result.asset) else {
            await MainActor.run { self.toast("视频导出失败") }
            return
        }
        let snap = Self.videoSnapshot(url: URL(fileURLWithPath: exported.path)) ?? ""
        do {
            let msg = try await LinLanImManager.sendVideoMessage(
                videoPath: exported.path,
                type: (exported.path as NSString).pathExtension.isEmpty
                    ? "mp4"
                    : (exported.path as NSString).pathExtension,
                durationSec: exported.duration,
                snapshotPath: snap,
                receiver: userID,
                groupId: groupID
            )
            await MainActor.run { self.appendLocal(msg) }
        } catch {
            await MainActor.run { self.toast(error.localizedDescription) }
        }
    }

    @objc private func pickFile() {
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.data, .item, .content])
        picker.delegate = self
        picker.allowsMultipleSelection = false
        present(picker, animated: true)
    }

    // MARK: - @ mention

    private func preloadMembers() async {
        guard let gid = groupID else { return }
        // 自己的角色走群资料，避免大群成员截断后拿不到身份、无法撤回他人
        if let info = try? await LinLanImManager.fetchGroupInfo(groupId: gid) {
            selfGroupRole = info.role
        }
        // fail-closed：拉不到属性时按禁止私聊处理
        groupProfileAccessEnabled = await LinLanGroupPolicy.fetchMemberProfileAccessEnabled(groupId: gid)
        let members = (try? await LinLanGroupMemberCache.members(
            groupId: gid,
            maxCount: LinLanGroupMemberCache.mentionLimit
        )) ?? []
        memberRoleMap = Dictionary(uniqueKeysWithValues: members.map { ($0.userId, $0.role) })
        var list = members
        let canAtAll = LinLanImManager.isOwnerOrAdmin(role: selfGroupRole)
            || LinLanImManager.canAtAll(members: members)
        if canAtAll {
            list.insert(
                LinLanGroupMemberItem(userId: LinLanImManager.atAllTag, displayName: LinLanImManager.atAllDisplay, role: 400),
                at: 0
            )
        }
        var map: [String: String] = [:]
        for m in list { map[m.displayName] = m.userId }
        await MainActor.run {
            self.mentionCandidates = list
            self.mentionNameMap = map
        }
    }

    @objc private func manualMention() {
        guard isGroup else { return }
        textView.insertText("@")
        showMention(filter: "")
    }

    private func showMention(filter: String) {
        let q = filter.trimmingCharacters(in: .whitespacesAndNewlines)
        mentionVisible = mentionCandidates.filter {
            q.isEmpty || $0.displayName.localizedCaseInsensitiveContains(q)
        }
        let show = isGroup && !mentionVisible.isEmpty
        mentionTable.isHidden = !show
        mentionTable.snp.updateConstraints { $0.height.equalTo(show ? min(220, mentionVisible.count * 44) : 0) }
        mentionTable.reloadData()
    }

    private func hideMention() {
        mentionTable.isHidden = true
        mentionTable.snp.updateConstraints { $0.height.equalTo(0) }
    }

    private func insertMention(_ item: LinLanGroupMemberItem) {
        let text = textView.text ?? ""
        if let range = text.range(of: "@", options: .backwards) {
            let prefix = String(text[..<range.lowerBound])
            textView.text = prefix + "@\(item.displayName) "
        } else {
            textView.insertText("@\(item.displayName) ")
        }
        mentionNameMap[item.displayName] = item.userId
        hideMention()
        refreshInputPlaceholder()
        adjustInputHeight()
    }

    // MARK: - Long press / multi / forward / pin / revoke / reply

    private func longPress(at index: Int) {
        guard messages.indices.contains(index) else { return }
        let msg = messages[index]
        if multiSelect {
            toggleSelect(msg)
            return
        }
        if LinLanImManager.isRevokedMessage(msg) { return }

        let sheet = UIAlertController(title: nil, message: nil, preferredStyle: .actionSheet)
        if !isSecurityAssistant {
            sheet.addAction(UIAlertAction(title: "回复", style: .default) { [weak self] _ in
                self?.setReply(msg)
            })
        }
        if msg.elemType == .V2TIM_ELEM_TYPE_TEXT,
           let raw = msg.textElem?.text, !raw.isEmpty,
           LinLanAttachmentCard.parse(raw) == nil {
            sheet.addAction(UIAlertAction(title: "复制", style: .default) { [weak self] _ in
                let copied = raw
                    .replacingOccurrences(of: "\r\n", with: "\n")
                    .replacingOccurrences(of: "\r", with: "\n")
                UIPasteboard.general.string = copied
                self?.toast("已复制")
            })
        }
        if isGroup {
            // 阅读群置顶对所有群成员开放；管理只开放给群主/管理员。
            if LinLanImManager.isOwnerOrAdmin(role: selfGroupRole),
               msg.elemType != .V2TIM_ELEM_TYPE_GROUP_TIPS {
                let pinned = groupPinnedMessages.contains { $0.msgID == msg.msgID }
                sheet.addAction(UIAlertAction(title: pinned ? "取消置顶" : "置顶", style: .default) { [weak self] _ in
                    Task { await self?.setGroupMessagePinned(msg, pinned: !pinned) }
                })
            }
        } else {
            let pinned = LinLanPinnedMessageStore.isPinned(conversationId: conversationId, msgId: msg.msgID)
            sheet.addAction(UIAlertAction(title: pinned ? "取消置顶" : "置顶", style: .default) { [weak self] _ in
                guard let self else { return }
                LinLanPinnedMessageStore.setPinned(
                    conversationId: self.conversationId,
                    msgId: msg.msgID,
                    pinned: !pinned,
                    summary: LinLanImManager.messagePreview(msg),
                    sender: LinLanQuoteHelper.senderName(msg)
                )
                self.pinCycleIndex = 0
                self.refreshPinnedBar()
                self.toast(!pinned ? "已置顶" : "已取消置顶")
            })
        }
        if !groupSendBlocked, LinLanImManager.canForwardMessage(msg) {
            sheet.addAction(UIAlertAction(title: "转发", style: .default) { [weak self] _ in
                self?.pickForwardTarget(messages: [msg], merge: false)
            })
        }
        if !groupSendBlocked, !isSecurityAssistant {
            sheet.addAction(UIAlertAction(title: "多选", style: .default) { [weak self] _ in
                self?.enterMulti(selecting: msg)
            })
        }
        let senderRole = memberRoleMap[msg.sender ?? ""] ?? 0
        if LinLanImManager.canRevokeMessage(msg, selfGroupRole: selfGroupRole, senderMemberRole: senderRole) {
            sheet.addAction(UIAlertAction(title: "撤回", style: .destructive) { [weak self] _ in
                Task { await self?.performRevoke(msg) }
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func performRevoke(_ msg: V2TIMMessage) async {
        let owner = LinLanImManager.recallScope()
        do {
            try await LinLanImManager.revokeMessage(msg)
            await MainActor.run {
                guard LinLanImManager.recalled.isCurrent(owner),
                      let idx = self.messages.firstIndex(where: { $0.msgID == msg.msgID }) else { return }
                if LinLanImManager.shouldSilentlyHideRevokedMessage(msg) { self.messages.remove(at: idx) }
                else { self.messages[idx] = msg }
                if self.replyTarget?.msgID == msg.msgID { self.clearReply() }
                self.selectedIds.remove(msg.msgID)
                self.tableView.reloadData()
                self.refreshPinnedBar()
            }
        } catch is CancellationError { }
          catch { await MainActor.run { self.toast(error.localizedDescription) } }
    }

    private func setReply(_ msg: V2TIMMessage) {
        replyTarget = msg
        quoteBar.isHidden = false
        quoteBar.snp.updateConstraints { $0.height.equalTo(36) }
        quoteLabel.text = LinLanQuoteHelper.resolveQuote(msg)?.displayLine
            ?? "\(LinLanQuoteHelper.senderName(msg)): \(LinLanImManager.messagePreview(msg))"
    }

    @objc private func clearReply() {
        replyTarget = nil
        quoteBar.isHidden = true
        quoteBar.snp.updateConstraints { $0.height.equalTo(0) }
        quoteLabel.text = nil
    }

    private func enterMulti(selecting: V2TIMMessage?) {
        guard !isSecurityAssistant else { return }
        if groupSendBlocked {
            toast(groupSendBlockedHint)
            return
        }
        multiSelect = true
        selectedIds.removeAll()
        if let selecting { selectedIds.insert(selecting.msgID) }
        dismissComposer()
        multiHeight?.update(offset: 48)
        multiBar.isHidden = false
        composerStack.isHidden = true
        inputBar.isHidden = true
        toolStack.isHidden = true
        tableView.reloadData()
    }

    @objc private func exitMulti() {
        multiSelect = false
        selectedIds.removeAll()
        multiHeight?.update(offset: 0)
        multiBar.isHidden = true
        if isSecurityAssistant {
            applySecurityAssistantReadOnlyUI()
        } else if groupSendBlocked {
            composerStack.isHidden = true
            inputBar.isHidden = true
            toolStack.isHidden = true
        } else {
            composerStack.isHidden = false
            inputBar.isHidden = false
            toolStack.isHidden = false
        }
        tableView.reloadData()
    }

    private func toggleSelect(_ msg: V2TIMMessage) {
        if selectedIds.contains(msg.msgID) { selectedIds.remove(msg.msgID) }
        else { selectedIds.insert(msg.msgID) }
        tableView.reloadData()
    }

    private func selectedMessages() -> [V2TIMMessage] {
        messages.filter { selectedIds.contains($0.msgID) }
    }

    @objc private func forwardSelectedOneByOne() {
        let list = selectedMessages()
        guard !list.isEmpty else { return }
        pickForwardTarget(messages: list, merge: false)
    }

    @objc private func forwardSelectedMerged() {
        let list = selectedMessages()
        guard !list.isEmpty else { return }
        pickForwardTarget(messages: list, merge: true)
    }

    private func pickForwardTarget(messages: [V2TIMMessage], merge: Bool) {
        let vc = LinLanForwardTargetViewController(excludeConversationId: conversationId)
        vc.onPicked = { [weak self] targets in
            self?.doForward(messages, to: targets, merge: merge)
        }
        let nav = UINavigationController(rootViewController: vc)
        nav.modalPresentationStyle = .pageSheet
        present(nav, animated: true)
    }

    private func doForward(_ list: [V2TIMMessage], to targets: [LinLanForwardTarget], merge: Bool) {
        let title = LinLanImManager.mergeForwardTitle(messages: list, isGroup: isGroup, conversationTitle: titleText)
        guard LinLanForwardJobs.shared.start(list, to: targets, merge: merge, title: title) else {
            toast("已有转发任务正在执行，请先查看结果")
            showForwardResults()
            return
        }
        exitMulti()
        showForwardResults()
    }

    @objc private func showForwardResults() {
        present(UINavigationController(rootViewController: LinLanForwardResultsViewController()), animated: true)
    }

    private func refreshPinnedBar() {
        if isGroup {
            let pins = visibleGroupPinnedMessages()
            pinnedBar.isHidden = pins.isEmpty
            if pins.isEmpty {
                pinnedBar.text = nil
            } else {
                if pinCycleIndex >= pins.count { pinCycleIndex = 0 }
                let message = pins[pinCycleIndex]
                let label = pins.count > 1 ? "置顶消息 \(pinCycleIndex + 1)/\(pins.count)" : "置顶消息"
                let sender = LinLanQuoteHelper.senderName(message)
                let body = LinLanImManager.messagePreview(message)
                pinnedBar.text = "  \(label)  \(sender): \(body)"
            }
            pinnedBar.snp.updateConstraints { $0.height.equalTo(pins.isEmpty ? 0 : 34) }
            return
        }
        let pins = LinLanPinnedMessageStore.list(conversationId: conversationId)
        pinnedBar.isHidden = pins.isEmpty
        if pins.isEmpty {
            pinnedBar.text = nil
        } else {
            if pinCycleIndex >= pins.count { pinCycleIndex = 0 }
            let cur = pins[pinCycleIndex]
            let label = pins.count > 1 ? "\(pinCycleIndex + 1)/\(pins.count)" : "置顶"
            pinnedBar.text = "  \(label)  \(cur.displayLine)"
        }
        pinnedBar.snp.updateConstraints { $0.height.equalTo(pins.isEmpty ? 0 : 34) }
    }

    /// 点击置顶条：定位当前条并高亮，多条时轮询下一条
    @objc private func cyclePinnedLocate() {
        if isGroup {
            let pins = visibleGroupPinnedMessages()
            guard !pins.isEmpty else { return }
            if pinCycleIndex >= pins.count { pinCycleIndex = 0 }
            locatePinnedMessage(msgId: pins[pinCycleIndex].msgID)
            if pins.count > 1 {
                pinCycleIndex = (pinCycleIndex + 1) % pins.count
                refreshPinnedBar()
            }
            return
        }
        let pins = LinLanPinnedMessageStore.list(conversationId: conversationId)
        guard !pins.isEmpty else { return }
        if pinCycleIndex >= pins.count { pinCycleIndex = 0 }
        let cur = pins[pinCycleIndex]
        locatePinnedMessage(msgId: cur.msgId)
        if pins.count > 1 {
            pinCycleIndex = (pinCycleIndex + 1) % pins.count
            refreshPinnedBar()
        }
    }

    @objc private func longPressPinnedBar(_ gesture: UILongPressGestureRecognizer) {
        guard gesture.state == .began, isGroup,
              LinLanImManager.isOwnerOrAdmin(role: selfGroupRole) else { return }
        let pins = visibleGroupPinnedMessages()
        guard !pins.isEmpty else { return }
        if pinCycleIndex >= pins.count { pinCycleIndex = 0 }
        let message = pins[pinCycleIndex]
        let sheet = UIAlertController(title: nil, message: LinLanImManager.messagePreview(message), preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "取消置顶", style: .destructive) { [weak self] _ in
            Task { await self?.setGroupMessagePinned(message, pinned: false) }
        })
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func visibleGroupPinnedMessages() -> [V2TIMMessage] {
        groupPinnedMessages.filter {
            !LinLanImManager.shouldSilentlyHideRevokedMessage($0) &&
                !LinLanImManager.shouldHideGroupRemovalTip($0, viewerRole: selfGroupRole)
        }
    }

    private func reloadGroupPinnedMessages() async {
        guard let gid = groupID?.trimmingCharacters(in: .whitespacesAndNewlines), !gid.isEmpty else { return }
        do {
            let pins = try await LinLanImManager.fetchPinnedGroupMessages(groupId: gid)
            await MainActor.run {
                guard LinLanImManager.isSameGroup(gid, self.groupID) else { return }
                self.groupPinnedMessages = pins
                self.pinCycleIndex = min(self.pinCycleIndex, max(pins.count - 1, 0))
                self.refreshPinnedBar()
            }
        } catch {
            // 被动刷新失败保留上次可见数据，不把网络失败误当成“全部取消置顶”。
        }
    }

    private func setGroupMessagePinned(_ message: V2TIMMessage, pinned: Bool) async {
        guard let gid = groupID, LinLanImManager.isOwnerOrAdmin(role: selfGroupRole) else { return }
        do {
            try await LinLanImManager.setGroupMessagePinned(groupId: gid, message: message, pinned: pinned)
            await reloadGroupPinnedMessages()
            await MainActor.run {
                self.pinCycleIndex = 0
                self.refreshPinnedBar()
                self.toast(pinned ? "已置顶" : "已取消置顶")
            }
        } catch {
            await MainActor.run { self.toast(error.localizedDescription) }
        }
    }

    private func verifyCurrentGroupMembership() async {
        guard let gid = groupID?.trimmingCharacters(in: .whitespacesAndNewlines), !gid.isEmpty else { return }
        if LinLanGroupMembershipState.isRemoved(groupID: gid) {
            await MainActor.run { self.exitRemovedGroupChat() }
            return
        }
        guard let snapshot = try? await LinLanImManager.fetchJoinedGroupSnapshot() else { return }
        let joined = snapshot.groups.contains { LinLanImManager.isSameGroup($0.groupId, gid) }
        // fetchJoinedGroupSnapshot 已在 token 仍有效时原子校准重新入群；
        // 这里不再无条件清除 removed，避免旧快照覆盖同时到达的 kick。
        if !joined,
           snapshot.isComplete,
           LinLanGroupMembershipState.isCurrent(snapshot.membershipToken) {
            LinLanImManager.handleCurrentUserRemovedFromGroup(gid)
        }
    }

    private func exitRemovedGroupChat() {
        guard isViewLoaded else { return }
        groupPinnedMessages.removeAll()
        guard let nav = navigationController,
              let index = nav.viewControllers.firstIndex(where: { $0 === self }) else { return }
        if index > 0 {
            nav.popToViewController(nav.viewControllers[index - 1], animated: true)
        } else {
            nav.popViewController(animated: true)
        }
    }

    private func locatePinnedMessage(msgId: String) {
        if let idx = messages.firstIndex(where: { $0.msgID == msgId }) {
            highlightMessage(at: idx, msgId: msgId)
            return
        }
        quoteLookupTask?.cancel()
        quoteLookupTask = Task { [weak self] in
            guard let self else { return }
            do {
                var cursor: V2TIMMessage?
                var seen = Set<String>()
                while !Task.isCancelled {
                    let page = try await LinLanImManager.fetchHistory(conversationId: self.conversationId, lastMsg: cursor)
                    guard !Task.isCancelled else { return }
                    if let found = page.first(where: { $0.msgID == msgId }) {
                        if !self.messages.contains(where: { $0.msgID == msgId }) {
                            let index = self.messages.firstIndex(where: { $0.timestamp > found.timestamp }) ?? self.messages.count
                            self.messages.insert(found, at: index)
                        }
                        self.tableView.reloadData()
                        self.tableView.layoutIfNeeded()
                        if let index = self.messages.firstIndex(where: { $0.msgID == msgId }) { self.highlightMessage(at: index, msgId: msgId) }
                        return
                    }
                    guard page.count >= LinLanImManager.historyPageSize, let next = page.last, seen.insert(next.msgID).inserted else { break }
                    cursor = next
                }
                if !Task.isCancelled { self.toast("原消息已删除或超出历史消息保留期限") }
            } catch { if !Task.isCancelled { self.toast("加载原消息失败，请重试") } }
        }
    }

    private func highlightMessage(at index: Int, msgId: String) {
        highlightedMsgId = msgId
        let ip = IndexPath(row: index, section: 0)
        tableView.scrollToRow(at: ip, at: .middle, animated: true)
        tableView.reloadRows(at: [ip], with: .none)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.6) { [weak self] in
            guard let self else { return }
            if self.highlightedMsgId == msgId {
                self.highlightedMsgId = nil
                if let i = self.messages.firstIndex(where: { $0.msgID == msgId }) {
                    self.tableView.reloadRows(at: [IndexPath(row: i, section: 0)], with: .fade)
                }
            }
        }
    }

    // MARK: - Helpers

    @objc private func linLanBack() {
        if multiSelect { exitMulti(); return }
        navigationController?.popViewController(animated: true)
    }

    @objc private func openPeerDetail() {
        if multiSelect { return }
        if let gid = groupID, !gid.isEmpty {
            linLanPush(LinLanGroupDetailViewController(groupId: gid, groupName: titleText))
            return
        }
        guard let uid = userID, !uid.isEmpty else { return }
        linLanPush(
            LinLanFriendDetailViewController(
                userId: uid,
                userName: titleText,
                isFriend: true
            )
        )
    }

    private func refreshSendPermission() async {
        guard let gid = groupID, !gid.isEmpty else {
            await MainActor.run {
                self.groupSendBlocked = false
                self.applyMuteUi()
            }
            return
        }
        if let info = try? await LinLanImManager.fetchGroupInfo(groupId: gid) {
            await MainActor.run { self.selfGroupRole = info.role }
        }
        let permission = await LinLanImManager.resolveGroupSendPermission(groupId: gid)
        await MainActor.run {
            self.groupSendBlocked = !permission.allowsSend
            self.groupSendBlockedHint = permission.blockedHint.isEmpty ? "已禁言" : permission.blockedHint
            self.applyMuteUi()
        }
    }

    private func applyMuteUi() {
        muteBar.text = groupSendBlockedHint
        muteBar.isHidden = !groupSendBlocked
        muteHeight?.update(offset: groupSendBlocked ? 48 : 0)
        if groupSendBlocked {
            if multiSelect {
                multiSelect = false
                selectedIds.removeAll()
                multiHeight?.update(offset: 0)
                multiBar.isHidden = true
                tableView.reloadData()
            }
            textView.resignFirstResponder()
            setEmojiVisible(false, animated: false)
            voiceMode = false
            composerStack.isHidden = true
            toolStack.isHidden = true
            inputBar.isHidden = true
            multiBar.isHidden = true
        } else if !multiSelect {
            composerStack.isHidden = false
            toolStack.isHidden = false
            inputBar.isHidden = false
            applyVoiceModeUI()
        }
        view.layoutIfNeeded()
    }

    @objc private func startAudioCall() {
        startRtcCall(video: false)
    }

    @objc private func startVideoCall() {
        startRtcCall(video: true)
    }

    private func startRtcCall(video: Bool) {
        guard let uid = userID, !uid.isEmpty else { return }
        Task {
            let granted = await LinLanRtcCallManager.requestCallPermissions(needCamera: video)
            await MainActor.run {
                guard granted else {
                    self.toast(video ? "需要麦克风和摄像头权限" : "需要麦克风权限")
                    return
                }
                LinLanRtcCallManager.shared.startCall(
                    peerUserId: uid,
                    peerName: self.titleText,
                    mediaType: video ? .video : .audio
                )
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak a] in
            a?.dismiss(animated: true)
        }
    }

    private func saveImageToTemp(_ image: UIImage) -> String? {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("img_\(UUID().uuidString).jpg")
        guard let data = image.jpegData(compressionQuality: 0.85) else { return nil }
        do {
            try data.write(to: url)
            return url.path
        } catch {
            return nil
        }
    }
}

// MARK: - Table / Text

extension LinLanChatViewController: UITableViewDataSource, UITableViewDelegate, UITextViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        if tableView === mentionTable { return mentionVisible.count }
        return messages.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if tableView === mentionTable {
            let cell = tableView.dequeueReusableCell(withIdentifier: "m", for: indexPath)
            cell.textLabel?.text = mentionVisible[indexPath.row].displayName
            return cell
        }
        let cell = tableView.dequeueReusableCell(
            withIdentifier: LinLanChatMessageCell.reuseID,
            for: indexPath
        ) as! LinLanChatMessageCell
        let msg = messages[indexPath.row]
        cell.configure(
            message: msg,
            multiSelect: multiSelect,
            selected: selectedIds.contains(msg.msgID),
            isGroup: isGroup,
            highlighted: highlightedMsgId == msg.msgID
        )
        cell.showReadStatus(message: msg, peerRead: peerReadIDs.contains(msg.msgID) || (peerReadThrough > 0 && msg.timestamp.timeIntervalSince1970 <= peerReadThrough))
        cell.onTapQuote = { [weak self] in
            guard let self, !self.multiSelect, let quote = LinLanQuoteHelper.resolveQuote(msg) else { return }
            self.locatePinnedMessage(msgId: quote.quotedMsgId)
        }
        cell.onLongPressAvatar = { [weak self] in self?.mentionSender(msg) }
        cell.onLongPress = { [weak self] in self?.longPress(at: indexPath.row) }
        cell.onTapAvatar = { [weak self] in
            self?.openMemberProfile(from: msg)
        }
        if !multiSelect {
            switch msg.elemType {
            case .V2TIM_ELEM_TYPE_IMAGE,
                 .V2TIM_ELEM_TYPE_VIDEO,
                 .V2TIM_ELEM_TYPE_FILE,
                 .V2TIM_ELEM_TYPE_SOUND,
                 .V2TIM_ELEM_TYPE_MERGER:
                cell.onTapMedia = { [weak self] in
                    guard let self else { return }
                    LinLanChatMediaPresenter.open(message: msg, from: self)
                }
            default:
                if let attachment = LinLanAttachmentCard.parse(msg.textElem?.text) {
                    cell.onTapMedia = { [weak self] in
                        Task { @MainActor in
                            do {
                                let url = try await LinLanAppContainer.apiClient().attachmentAccess(fileID: attachment.fileID)
                                await UIApplication.shared.open(url)
                            } catch { self?.toast("文件暂时无法访问，请稍后重试") }
                        }
                    }
                } else if LinLanImManager.parseMergerCard(msg) != nil {
                    cell.onTapMedia = { [weak self] in
                        guard let self else { return }
                        LinLanChatMediaPresenter.open(message: msg, from: self)
                    }
                } else {
                    cell.onTapMedia = nil
                }
            }
        }
        return cell
    }

    private func mentionSender(_ msg: V2TIMMessage) {
        guard isGroup, !multiSelect, !groupSendBlocked, !msg.isSelf,
              let uid = msg.sender, !uid.isEmpty else { return }
        let name = LinLanQuoteHelper.senderName(msg)
        if voiceMode { voiceMode = false; applyVoiceModeUI() }
        setEmojiVisible(false, animated: false)
        textView.insertText("@\(name) ")
        mentionNameMap[name] = uid
        hideMention()
        adjustInputHeight()
    }

    private func openMemberProfile(from msg: V2TIMMessage) {
        if multiSelect { return }
        if msg.isSelf { return }
        if !isGroup {
            let uid = (msg.sender ?? msg.userID ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !uid.isEmpty else { return }
            linLanPush(
                LinLanFriendDetailViewController(
                    userId: uid,
                    userName: LinLanQuoteHelper.senderName(msg),
                    isFriend: true
                )
            )
            return
        }
        let uid = (msg.sender ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !uid.isEmpty else { return }
        let role = memberRoleMap[uid] ?? 0
        guard LinLanImManager.canViewMemberProfile(
            profileAccessEnabled: groupProfileAccessEnabled,
            selfRole: selfGroupRole,
            targetRole: role,
            targetUserId: uid
        ) else {
            toast("群已禁止查看成员资料")
            return
        }
        linLanPush(
            LinLanFriendDetailViewController(
                userId: uid,
                userName: LinLanQuoteHelper.senderName(msg),
                isFriend: false
            )
        )
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        if tableView === mentionTable {
            insertMention(mentionVisible[indexPath.row])
            return
        }
        tableView.deselectRow(at: indexPath, animated: true)
        if multiSelect { toggleSelect(messages[indexPath.row]) }
    }

    func scrollViewDidScroll(_ scrollView: UIScrollView) {
        guard scrollView === tableView else { return }
        if isNearBottom() { clearNewMessageIndicator() }
        guard hasMore, !loadingMore, !loadingInitial, oldestMsg != nil else { return }
        // 滚到顶部附近自动续拉（与下拉刷新互补）
        let topY = -scrollView.adjustedContentInset.top
        if scrollView.contentOffset.y <= topY + 48 {
            Task { await loadMore() }
        }
    }

    func textViewDidChange(_ textView: UITextView) {
        refreshInputPlaceholder()
        adjustInputHeight()
        let text = textView.text ?? ""
        guard isGroup, let at = text.range(of: "@", options: .backwards) else {
            hideMention()
            return
        }
        let after = text[at.upperBound...]
        if after.contains(" ") || after.contains("\n") {
            hideMention()
            return
        }
        showMention(filter: String(after))
    }

    func textView(
        _ textView: UITextView,
        shouldChangeTextIn range: NSRange,
        replacementText text: String
    ) -> Bool {
        // Return 键的 `\n` 直接保留；粘贴其他系统的文本时只统一换行符，不压缩多行内容。
        guard text.contains("\r") else { return true }
        let normalized = text
            .replacingOccurrences(of: "\r\n", with: "\n")
            .replacingOccurrences(of: "\r", with: "\n")
        let current = textView.text as NSString
        guard range.location <= current.length, NSMaxRange(range) <= current.length else { return true }
        textView.text = current.replacingCharacters(in: range, with: normalized)
        textView.selectedRange = NSRange(
            location: range.location + (normalized as NSString).length,
            length: 0
        )
        textViewDidChange(textView)
        return false
    }
}

// MARK: - Pickers

extension LinLanChatViewController: PHPickerViewControllerDelegate, UIImagePickerControllerDelegate, UINavigationControllerDelegate, UIDocumentPickerDelegate {
    func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
        picker.dismiss(animated: true)
        let providers = results.map(\.itemProvider).filter { $0.canLoadObject(ofClass: UIImage.self) }
        guard !providers.isEmpty else { return }
        Task { [weak self] in
            guard let self else { return }
            var failCount = 0
            for provider in providers {
                let image: UIImage? = await withCheckedContinuation { cont in
                    provider.loadObject(ofClass: UIImage.self) { obj, _ in
                        cont.resume(returning: obj as? UIImage)
                    }
                }
                guard let image, let path = self.saveImageToTemp(image) else {
                    failCount += 1
                    continue
                }
                do {
                    let msg = try await LinLanImManager.sendImageMessage(
                        path: path,
                        receiver: self.userID,
                        groupId: self.groupID
                    )
                    await MainActor.run { self.appendLocal(msg) }
                } catch {
                    failCount += 1
                    await MainActor.run { self.toast(error.localizedDescription) }
                }
            }
            if failCount > 0, failCount == providers.count {
                await MainActor.run { self.toast("图片发送失败") }
            }
        }
    }

    func imagePickerControllerDidCancel(_ picker: UIImagePickerController) {
        picker.dismiss(animated: true)
    }

    func imagePickerController(
        _ picker: UIImagePickerController,
        didFinishPickingMediaWithInfo info: [UIImagePickerController.InfoKey: Any]
    ) {
        picker.dismiss(animated: true)
        guard let url = info[.mediaURL] as? URL else { return }
        Task {
            do {
                let snap = Self.videoSnapshot(url: url) ?? ""
                let duration = await Self.videoDuration(url: url)
                let msg = try await LinLanImManager.sendVideoMessage(
                    videoPath: url.path,
                    type: url.pathExtension.isEmpty ? "mp4" : url.pathExtension,
                    durationSec: duration,
                    snapshotPath: snap,
                    receiver: userID,
                    groupId: groupID
                )
                await MainActor.run { self.appendLocal(msg) }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
        guard let url = urls.first else { return }
        let accessed = url.startAccessingSecurityScopedResource()
        defer { if accessed { url.stopAccessingSecurityScopedResource() } }
        let dest = FileManager.default.temporaryDirectory.appendingPathComponent(url.lastPathComponent)
        try? FileManager.default.removeItem(at: dest)
        do {
            try FileManager.default.copyItem(at: url, to: dest)
        } catch {
            toast(error.localizedDescription)
            return
        }
        Task {
            do {
                let msg = try await LinLanImManager.sendFileMessage(
                    path: dest.path,
                    fileName: dest.lastPathComponent,
                    receiver: userID,
                    groupId: groupID
                )
                await MainActor.run { self.appendLocal(msg) }
            } catch {
                await MainActor.run { self.toast(error.localizedDescription) }
            }
        }
    }

    private static func videoSnapshot(url: URL) -> String? {
        let asset = AVURLAsset(url: url)
        let gen = AVAssetImageGenerator(asset: asset)
        gen.appliesPreferredTrackTransform = true
        guard let cg = try? gen.copyCGImage(at: .zero, actualTime: nil) else { return nil }
        let image = UIImage(cgImage: cg)
        let path = FileManager.default.temporaryDirectory.appendingPathComponent("snap_\(UUID().uuidString).jpg")
        guard let data = image.jpegData(compressionQuality: 0.7) else { return nil }
        try? data.write(to: path)
        return path.path
    }

    private static func videoDuration(url: URL) async -> Int {
        let asset = AVURLAsset(url: url)
        do {
            let duration = try await asset.load(.duration)
            return max(1, Int(CMTimeGetSeconds(duration).rounded()))
        } catch {
            return 1
        }
    }
}
