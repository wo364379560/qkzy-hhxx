//
//  LinLanChatMessageCell.swift
//  TodayToTrackRun
//
//  约束只在 init 建一次；configure 只改 constant / isActive / isHidden。
//  禁止对 cell 子视图 remakeConstraints（复用时 SnapKit 曾因此 EXC_BAD_ACCESS）。
//

import ImSDK_Plus_Swift
import UIKit

final class LinLanChatMessageCell: UITableViewCell {
    static let reuseID = "LinLanChatMessageCell"

    var onLongPress: (() -> Void)?
    var onTapMedia: (() -> Void)?
    var onTapAvatar: (() -> Void)?
    var onLongPressAvatar: (() -> Void)?
    var onTapQuote: (() -> Void)?

    private let check = UIImageView()
    private let avatar = UIImageView()
    private let nameLabel = UILabel()
    private let bubble = UIView()
    private let quoteLabel = UILabel()
    private let bodyLabel = UILabel()
    private let mediaImage = UIImageView()
    private let mediaIcon = UIImageView()
    private let mediaTitle = UILabel()
    private let mediaSub = UILabel()
    private let playBadge = UIImageView()
    private let mergerStack = UIStackView()
    private let timeLabel = UILabel()

    /// 语音气泡：微信式喇叭 + 声纹弧线（非柱状 waveform）
    private var soundMsgId: String?
    private var soundMine = false
    private var voiceAnimTimer: Timer?
    private var voiceAnimStep = 0
    private static let voiceIdleSymbol = "speaker.wave.2.fill"
    private static let voicePlaySymbols = [
        "speaker.wave.1.fill",
        "speaker.wave.2.fill",
        "speaker.wave.3.fill",
    ]

    private var avatarLeading: NSLayoutConstraint!
    private var avatarTrailing: NSLayoutConstraint!
    private var avatarWidth: NSLayoutConstraint!
    private var avatarHeight: NSLayoutConstraint!
    private var nameLeading: NSLayoutConstraint!
    private var nameTrailing: NSLayoutConstraint!
    private var bubbleLeading: NSLayoutConstraint!
    private var bubbleTrailing: NSLayoutConstraint!
    private var bubbleCenterX: NSLayoutConstraint!
    private var bubbleWidthCap: NSLayoutConstraint!
    private var bubbleFixedWidth: NSLayoutConstraint!
    private var bubbleBottomToContent: NSLayoutConstraint!
    private var timeLeading: NSLayoutConstraint!
    private var timeTrailing: NSLayoutConstraint!

    private var quoteTop: NSLayoutConstraint!
    private var quoteHeight: NSLayoutConstraint!
    private var bodyTopToBubble: NSLayoutConstraint!
    private var bodyTopToQuote: NSLayoutConstraint!
    private var bodyLeading: NSLayoutConstraint!
    private var bodyTrailing: NSLayoutConstraint!
    private var bodyBottom: NSLayoutConstraint!

    private var mediaImageTop: NSLayoutConstraint!
    private var mediaImageLeading: NSLayoutConstraint!
    private var mediaImageWidth: NSLayoutConstraint!
    private var mediaImageHeight: NSLayoutConstraint!
    private var bubbleTrailToMedia: NSLayoutConstraint!
    private var bubbleBottomToMedia: NSLayoutConstraint!

    private var mediaIconLeading: NSLayoutConstraint!
    private var mediaIconWidth: NSLayoutConstraint!
    private var mediaIconHeight: NSLayoutConstraint!
    private var mediaTitleLeading: NSLayoutConstraint!
    private var mediaTitleTop: NSLayoutConstraint!
    private var mediaTitleBottom: NSLayoutConstraint!
    private var mediaSubTop: NSLayoutConstraint!
    private var mediaSubHeight: NSLayoutConstraint!
    private var mediaSubBottom: NSLayoutConstraint!

    private var mergerLeading: NSLayoutConstraint!
    private var mergerTrailing: NSLayoutConstraint!
    private var mergerTop: NSLayoutConstraint!
    private var mergerBottom: NSLayoutConstraint!
    private var mergerWidth: NSLayoutConstraint!

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        selectionStyle = .none
        backgroundColor = .clear
        contentView.backgroundColor = .clear
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(onSoundPlaybackDidChange(_:)),
            name: .linLanSoundPlaybackDidChange,
            object: nil
        )

        [
            check, avatar, nameLabel, bubble, quoteLabel, bodyLabel,
            mediaImage, mediaIcon, mediaTitle, mediaSub, playBadge, mergerStack, timeLabel,
        ].forEach { $0.translatesAutoresizingMaskIntoConstraints = false }

        check.tintColor = AppTheme.accent
        check.contentMode = .scaleAspectFit
        check.isHidden = true

        avatar.layer.cornerRadius = 18
        avatar.clipsToBounds = true
        avatar.contentMode = .scaleAspectFill
        avatar.backgroundColor = AppTheme.accentSoft
        avatar.isUserInteractionEnabled = true
        let avatarPress = UILongPressGestureRecognizer(target: self, action: #selector(longPressAvatar(_:)))
        let avatarTap = UITapGestureRecognizer(target: self, action: #selector(tapAvatar))
        avatarTap.require(toFail: avatarPress)
        avatar.addGestureRecognizer(avatarPress)
        avatar.addGestureRecognizer(avatarTap)

        nameLabel.font = .systemFont(ofSize: 12, weight: .medium)
        nameLabel.textColor = AppTheme.textSecondary

        bubble.layer.cornerRadius = 12
        bubble.clipsToBounds = true

        quoteLabel.isUserInteractionEnabled = true
        let quoteTap = UITapGestureRecognizer(target: self, action: #selector(tapQuote))
        quoteLabel.addGestureRecognizer(quoteTap)
        quoteLabel.font = .systemFont(ofSize: 12)
        quoteLabel.textColor = AppTheme.textSecondary
        quoteLabel.numberOfLines = 2
        quoteLabel.backgroundColor = UIColor.black.withAlphaComponent(0.06)
        quoteLabel.layer.cornerRadius = 6
        quoteLabel.clipsToBounds = true

        bodyLabel.font = .systemFont(ofSize: 16)
        bodyLabel.numberOfLines = 0
        bodyLabel.adjustsFontSizeToFitWidth = false
        // 聊天内容可能是连续中文、数字或长链接。按单词换行会让无空格长文本
        // 无法折行并被气泡裁掉；按字符换行才能保证正文完整显示。
        bodyLabel.lineBreakMode = .byCharWrapping
        bodyLabel.setContentCompressionResistancePriority(.required, for: .vertical)
        bodyLabel.setContentHuggingPriority(.required, for: .vertical)

        mediaImage.contentMode = .scaleAspectFill
        mediaImage.clipsToBounds = true
        mediaImage.layer.cornerRadius = 8

        mediaIcon.contentMode = .scaleAspectFit
        mediaTitle.font = .systemFont(ofSize: 15, weight: .medium)
        mediaTitle.numberOfLines = 2
        mediaSub.font = .systemFont(ofSize: 12)
        mediaSub.textColor = AppTheme.textSecondary

        playBadge.image = UIImage(systemName: "play.circle.fill")
        playBadge.tintColor = .white
        playBadge.isHidden = true

        mergerStack.axis = .vertical
        mergerStack.spacing = 4
        mergerStack.isHidden = true

        timeLabel.font = .systemFont(ofSize: 11)
        timeLabel.textColor = AppTheme.textSecondary

        contentView.addSubview(check)
        contentView.addSubview(avatar)
        contentView.addSubview(nameLabel)
        contentView.addSubview(bubble)
        contentView.addSubview(timeLabel)
        bubble.addSubview(quoteLabel)
        bubble.addSubview(bodyLabel)
        bubble.addSubview(mediaImage)
        mediaImage.addSubview(playBadge)
        bubble.addSubview(mediaIcon)
        bubble.addSubview(mediaTitle)
        bubble.addSubview(mediaSub)
        bubble.addSubview(mergerStack)

        // MARK: avatar / name / bubble / time
        avatarWidth = avatar.widthAnchor.constraint(equalToConstant: 36)
        avatarHeight = avatar.heightAnchor.constraint(equalToConstant: 36)
        avatarLeading = avatar.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16)
        avatarTrailing = avatar.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16)

        nameLeading = nameLabel.leadingAnchor.constraint(equalTo: avatar.trailingAnchor, constant: 8)
        nameTrailing = nameLabel.trailingAnchor.constraint(equalTo: avatar.leadingAnchor, constant: -8)

        bubbleLeading = bubble.leadingAnchor.constraint(equalTo: avatar.trailingAnchor, constant: 8)
        bubbleTrailing = bubble.trailingAnchor.constraint(equalTo: avatar.leadingAnchor, constant: -8)
        bubbleCenterX = bubble.centerXAnchor.constraint(equalTo: contentView.centerXAnchor)
        bubbleWidthCap = bubble.widthAnchor.constraint(lessThanOrEqualTo: contentView.widthAnchor, multiplier: 0.72)
        bubbleFixedWidth = bubble.widthAnchor.constraint(equalToConstant: 120)
        bubbleFixedWidth.priority = .defaultHigh

        timeLeading = timeLabel.leadingAnchor.constraint(equalTo: bubble.leadingAnchor)
        timeTrailing = timeLabel.trailingAnchor.constraint(equalTo: bubble.trailingAnchor)

        bubbleBottomToContent = bubble.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -22)
        NSLayoutConstraint.activate([
            check.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 8),
            check.centerYAnchor.constraint(equalTo: avatar.centerYAnchor),
            check.widthAnchor.constraint(equalToConstant: 22),
            check.heightAnchor.constraint(equalToConstant: 22),

            avatar.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 8),
            avatarWidth, avatarHeight, avatarLeading,

            nameLabel.topAnchor.constraint(equalTo: avatar.topAnchor),
            nameLabel.widthAnchor.constraint(lessThanOrEqualTo: contentView.widthAnchor, multiplier: 0.65),
            nameLeading,

            bubble.topAnchor.constraint(equalTo: nameLabel.bottomAnchor, constant: 4),
            bubbleBottomToContent,
            bubbleWidthCap, bubbleLeading,

            timeLabel.topAnchor.constraint(equalTo: bubble.bottomAnchor, constant: 2),
            timeLabel.bottomAnchor.constraint(lessThanOrEqualTo: contentView.bottomAnchor, constant: -4),
            timeLeading,
        ])
        avatarTrailing.isActive = false
        nameTrailing.isActive = false
        bubbleTrailing.isActive = false
        bubbleCenterX.isActive = false
        bubbleFixedWidth.isActive = false
        timeTrailing.isActive = false

        // MARK: quote / body
        quoteTop = quoteLabel.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 10)
        quoteHeight = quoteLabel.heightAnchor.constraint(equalToConstant: 0)
        bodyTopToBubble = bodyLabel.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 8)
        bodyTopToQuote = bodyLabel.topAnchor.constraint(equalTo: quoteLabel.bottomAnchor, constant: 6)
        bodyLeading = bodyLabel.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 10)
        bodyTrailing = bodyLabel.trailingAnchor.constraint(equalTo: bubble.trailingAnchor, constant: -10)
        bodyBottom = bodyLabel.bottomAnchor.constraint(equalTo: bubble.bottomAnchor, constant: -8)
        NSLayoutConstraint.activate([
            quoteTop,
            quoteLabel.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 10),
            quoteLabel.trailingAnchor.constraint(equalTo: bubble.trailingAnchor, constant: -10),
            quoteHeight,
            bodyTopToBubble, bodyLeading, bodyTrailing, bodyBottom,
        ])
        bodyTopToQuote.isActive = false

        // MARK: media image：top+leading+宽高，bubble 右/底跟随图片
        mediaImageTop = mediaImage.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 2)
        mediaImageLeading = mediaImage.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 2)
        mediaImageWidth = mediaImage.widthAnchor.constraint(equalToConstant: 0)
        mediaImageHeight = mediaImage.heightAnchor.constraint(equalToConstant: 0)
        bubbleTrailToMedia = bubble.trailingAnchor.constraint(equalTo: mediaImage.trailingAnchor, constant: 2)
        bubbleBottomToMedia = bubble.bottomAnchor.constraint(equalTo: mediaImage.bottomAnchor, constant: 2)
        NSLayoutConstraint.activate([
            mediaImageTop, mediaImageLeading, mediaImageWidth, mediaImageHeight,
            playBadge.centerXAnchor.constraint(equalTo: mediaImage.centerXAnchor),
            playBadge.centerYAnchor.constraint(equalTo: mediaImage.centerYAnchor),
            playBadge.widthAnchor.constraint(equalToConstant: 44),
            playBadge.heightAnchor.constraint(equalToConstant: 44),
        ])
        bubbleTrailToMedia.isActive = false
        bubbleBottomToMedia.isActive = false

        // MARK: file / sound row
        mediaIconLeading = mediaIcon.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 12)
        mediaIconWidth = mediaIcon.widthAnchor.constraint(equalToConstant: 0)
        mediaIconHeight = mediaIcon.heightAnchor.constraint(equalToConstant: 0)
        mediaTitleLeading = mediaTitle.leadingAnchor.constraint(equalTo: mediaIcon.trailingAnchor, constant: 8)
        mediaTitleTop = mediaTitle.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 12)
        mediaTitleBottom = mediaTitle.bottomAnchor.constraint(equalTo: bubble.bottomAnchor, constant: -12)
        mediaSubTop = mediaSub.topAnchor.constraint(equalTo: mediaTitle.bottomAnchor, constant: 4)
        mediaSubHeight = mediaSub.heightAnchor.constraint(equalToConstant: 0)
        mediaSubBottom = mediaSub.bottomAnchor.constraint(equalTo: bubble.bottomAnchor, constant: -10)
        NSLayoutConstraint.activate([
            mediaIconLeading,
            mediaIcon.centerYAnchor.constraint(equalTo: bubble.centerYAnchor),
            mediaIconWidth, mediaIconHeight,
            mediaTitleLeading,
            mediaTitle.trailingAnchor.constraint(equalTo: bubble.trailingAnchor, constant: -12),
            mediaTitleTop, mediaTitleBottom,
            mediaSub.leadingAnchor.constraint(equalTo: mediaTitle.leadingAnchor),
            mediaSub.trailingAnchor.constraint(equalTo: mediaTitle.trailingAnchor),
            mediaSubTop, mediaSubHeight,
        ])
        mediaSubBottom.isActive = false

        // MARK: merger
        mergerLeading = mergerStack.leadingAnchor.constraint(equalTo: bubble.leadingAnchor, constant: 12)
        mergerTrailing = mergerStack.trailingAnchor.constraint(equalTo: bubble.trailingAnchor, constant: -12)
        mergerTop = mergerStack.topAnchor.constraint(equalTo: bubble.topAnchor, constant: 12)
        mergerBottom = mergerStack.bottomAnchor.constraint(equalTo: bubble.bottomAnchor, constant: -12)
        mergerWidth = mergerStack.widthAnchor.constraint(equalToConstant: 0)
        NSLayoutConstraint.activate([mergerLeading, mergerTop, mergerWidth])
        mergerTrailing.isActive = false
        mergerBottom.isActive = false

        let lp = UILongPressGestureRecognizer(target: self, action: #selector(handleLongPress(_:)))
        bubble.addGestureRecognizer(lp)
        let bubbleTap = UITapGestureRecognizer(target: self, action: #selector(tapMedia))
        bubbleTap.require(toFail: lp)
        bubbleTap.require(toFail: quoteTap)
        bubble.addGestureRecognizer(bubbleTap)
        bubble.isUserInteractionEnabled = true
    }

    required init?(coder: NSCoder) { fatalError() }

    deinit {
        voiceAnimTimer?.invalidate()
        NotificationCenter.default.removeObserver(self)
    }

    @objc private func handleLongPress(_ g: UILongPressGestureRecognizer) {
        if g.state == .began { onLongPress?() }
    }

    @objc private func tapMedia() { onTapMedia?() }
    @objc private func tapAvatar() { onTapAvatar?() }
    @objc private func tapQuote() { onTapQuote?() }
    @objc private func longPressAvatar(_ gesture: UILongPressGestureRecognizer) {
        if gesture.state == .began { onLongPressAvatar?() }
    }

    func showReadStatus(message: V2TIMMessage, peerRead: Bool) {
        guard message.isSelf, message.groupID?.isEmpty ?? true, message.status == .V2TIM_MSG_STATUS_SEND_SUCC else { return }
        timeLabel.text = "\(LinLanImManager.clockTime(message.timestamp)) · \(peerRead || message.isPeerRead ? "已读" : "未读")"
    }

    @objc private func onSoundPlaybackDidChange(_ note: Notification) {
        guard let soundMsgId, !mediaIcon.isHidden else { return }
        let playingId = note.userInfo?["msgId"] as? String
        setVoicePlaying(playingId == soundMsgId)
    }

    func configure(
        message: V2TIMMessage,
        multiSelect: Bool,
        selected: Bool,
        isGroup: Bool,
        highlighted: Bool = false
    ) {
        assert(Thread.isMainThread)
        let mine = message.isSelf
        let revoked = LinLanImManager.isRevokedMessage(message)
        let inset: CGFloat = multiSelect ? 36 : 16

        check.isHidden = !multiSelect
        check.image = UIImage(systemName: selected ? "checkmark.circle.fill" : "circle")
        contentView.backgroundColor = highlighted
            ? AppTheme.accent.withAlphaComponent(0.18)
            : .clear

        clearContent()

        if message.elemType == .V2TIM_ELEM_TYPE_GROUP_TIPS {
            applySystemTipLayout()
            let tip = LinLanImManager.groupTipsTipText(message)
            showSystemTip(tip.isEmpty ? "群提示" : tip)
            return
        }

        applySideLayout(mine: mine, inset: inset, isGroup: isGroup, message: message)

        if revoked {
            showText(LinLanImManager.revokedTipText(message), mine: mine, secondary: true)
            return
        }

        if let quote = LinLanQuoteHelper.resolveQuote(message) {
            quoteLabel.isHidden = false
            quoteLabel.text = "  \(quote.displayLine)  "
            quoteHeight.isActive = false
            bodyTopToBubble.isActive = false
            bodyTopToQuote.isActive = true
        }

        if let card = LinLanImManager.parseMergerCard(message) {
            showMerger(card)
            return
        }

        switch message.elemType {
        case .V2TIM_ELEM_TYPE_IMAGE:
            showImage(url: LinLanImManager.imageUrl(of: message) ?? message.imageElem?.path)
        case .V2TIM_ELEM_TYPE_VIDEO:
            playBadge.isHidden = false
            showImage(url: message.videoElem?.snapshotPath, height: 220)
            if mediaImage.image == nil {
                message.videoElem?.getSnapshotUrl { [weak self] url in
                    DispatchQueue.main.async {
                        guard let self else { return }
                        LinLanImageLoader.load(urlString: url, into: self.mediaImage)
                    }
                }
            }
        case .V2TIM_ELEM_TYPE_SOUND:
            showSound(
                seconds: max(1, message.soundElem?.duration ?? 0),
                mine: mine,
                msgId: message.msgID
            )
        case .V2TIM_ELEM_TYPE_FILE:
            showFile(
                icon: "doc.fill",
                title: message.fileElem?.filename ?? "文件",
                sub: LinLanImManager.clockTime(message.timestamp),
                mine: mine
            )
        case .V2TIM_ELEM_TYPE_TEXT:
            let raw = (message.textElem?.text ?? "")
                .replacingOccurrences(of: "\r\n", with: "\n")
                .replacingOccurrences(of: "\r", with: "\n")
            if let attachment = LinLanAttachmentCard.parse(raw) {
                showFile(icon: attachment.isVideo ? "video.fill" : "doc.fill", title: attachment.name, sub: "点击预览或下载", mine: mine)
            } else { showText(raw, mine: mine, secondary: false) }
        case .V2TIM_ELEM_TYPE_MERGER:
            if let card = LinLanImManager.parseMergerCard(message) {
                showMerger(card)
            } else {
                showText("[聊天记录]", mine: mine, secondary: false)
            }
        case .V2TIM_ELEM_TYPE_CUSTOM:
            if let rtc = LinLanRtcSignalingPayload.parseFromMessage(message) {
                let icon = rtc.mediaType == .video ? "video.fill" : "phone.fill"
                showFile(
                    icon: icon,
                    title: rtc.displayText(isSelf: mine),
                    sub: LinLanImManager.clockTime(message.timestamp),
                    mine: mine
                )
            } else {
                showText(LinLanImManager.messagePreview(message), mine: mine, secondary: false)
            }
        default:
            showText(LinLanImManager.messagePreview(message), mine: mine, secondary: false)
        }
    }

    private func clearContent() {
        quoteLabel.isHidden = true
        quoteLabel.text = nil
        quoteHeight.isActive = true
        quoteHeight.constant = 0
        bodyTopToQuote.isActive = false
        bodyTopToBubble.isActive = true
        bodyTopToBubble.constant = 8
        bodyLeading.constant = 10
        bodyTrailing.constant = -10
        bodyBottom.constant = -8
        bodyLabel.isHidden = true
        bodyLabel.attributedText = nil
        bodyLabel.text = nil

        mediaImage.isHidden = true
        mediaImage.image = nil
        playBadge.isHidden = true
        mediaImageWidth.constant = 0
        mediaImageHeight.constant = 0
        bubbleTrailToMedia.isActive = false
        bubbleBottomToMedia.isActive = false

        stopVoiceAnim()
        soundMsgId = nil
        soundMine = false
        mediaIcon.isHidden = true
        mediaIcon.transform = .identity
        mediaTitle.isHidden = true
        mediaSub.isHidden = true
        mediaIconWidth.constant = 0
        mediaIconHeight.constant = 0
        mediaTitleBottom.isActive = true
        mediaSubHeight.isActive = true
        mediaSubHeight.constant = 0
        mediaSubBottom.isActive = false

        mergerStack.isHidden = true
        mergerStack.arrangedSubviews.forEach {
            mergerStack.removeArrangedSubview($0)
            $0.removeFromSuperview()
        }
        mergerWidth.constant = 0
        mergerTrailing.isActive = false
        mergerBottom.isActive = false

        bubble.backgroundColor = .white
        bubble.layer.borderWidth = 0
        bubbleFixedWidth.isActive = false
        bubbleWidthCap.isActive = true
        bubbleBottomToContent.isActive = true
    }

    private func applySideLayout(
        mine: Bool,
        inset: CGFloat,
        isGroup: Bool,
        message: V2TIMMessage
    ) {
        avatar.isHidden = false
        timeLabel.isHidden = false
        avatarWidth.constant = 36
        avatarHeight.constant = 36
        avatarLeading.constant = inset
        avatarTrailing.constant = -inset

        bubbleCenterX.isActive = false
        avatarLeading.isActive = !mine
        avatarTrailing.isActive = mine
        bubbleLeading.isActive = !mine
        bubbleTrailing.isActive = mine
        nameLeading.isActive = !mine
        nameTrailing.isActive = mine
        timeLeading.isActive = !mine
        timeTrailing.isActive = mine
        nameLabel.textAlignment = mine ? .right : .left

        let display = LinLanQuoteHelper.senderName(message)
        nameLabel.isHidden = mine && !isGroup
        nameLabel.text = display
        LinLanImageLoader.load(
            urlString: message.faceURL,
            into: avatar,
            placeholderLetter: String(display.prefix(1))
        )
        timeLabel.text = LinLanImManager.clockTime(message.timestamp)
    }

    private func applySystemTipLayout() {
        check.isHidden = true
        avatar.isHidden = true
        nameLabel.isHidden = true
        nameLabel.text = nil
        timeLabel.isHidden = true
        avatarWidth.constant = 0
        avatarHeight.constant = 0
        avatarLeading.isActive = true
        avatarTrailing.isActive = false
        bubbleLeading.isActive = false
        bubbleTrailing.isActive = false
        bubbleCenterX.isActive = true
        nameLeading.isActive = true
        nameTrailing.isActive = false
        timeLeading.isActive = true
        timeTrailing.isActive = false
    }

    private func showSystemTip(_ text: String) {
        bodyLabel.isHidden = false
        bodyLabel.preferredMaxLayoutWidth = UIScreen.main.bounds.width * 0.78 - 24
        let para = NSMutableParagraphStyle()
        para.alignment = .center
        bodyLabel.attributedText = NSAttributedString(
            string: text,
            attributes: [
                .font: UIFont.systemFont(ofSize: 12),
                .foregroundColor: AppTheme.textSecondary,
                .paragraphStyle: para,
            ]
        )
        bubble.backgroundColor = UIColor(white: 0.93, alpha: 1)
        bodyTopToBubble.constant = 6
        bodyLeading.constant = 10
        bodyTrailing.constant = -10
        bodyBottom.constant = -6
    }

    private func showText(_ text: String, mine: Bool, secondary: Bool) {
        bodyLabel.isHidden = false
        // Cell 会在文本、图片、文件等多种消息间复用。每次显示正文时都恢复
        // “无限行 + 按字符换行”，防止长中文、连续数字或链接被截断。
        bodyLabel.numberOfLines = 0
        bodyLabel.lineBreakMode = .byCharWrapping
        let maxBubble = UIScreen.main.bounds.width * 0.72
        let font = UIFont.systemFont(ofSize: secondary ? 13 : 16)
        // 始终用气泡的最大可用宽度计算 UILabel 的多行 intrinsic height。
        // 不先用文本 boundingRect 反推宽度，避免某些复合 Unicode/长文本
        // 在自适应 Cell 第一次布局时得到过短的高度并被 bubble 裁切。
        bodyLabel.preferredMaxLayoutWidth = max(1, maxBubble - 20)
        let para = NSMutableParagraphStyle()
        para.lineBreakMode = .byCharWrapping
        para.lineSpacing = 2
        let color: UIColor = secondary ? AppTheme.textSecondary : (mine ? .white : AppTheme.textPrimary)
        bodyLabel.attributedText = NSAttributedString(
            string: text,
            attributes: [
                .font: font,
                .foregroundColor: color,
                .paragraphStyle: para,
            ]
        )
        bubble.backgroundColor = secondary ? UIColor(white: 0.93, alpha: 1) : (mine ? AppTheme.accent : .white)
    }

#if DEBUG
    /// 只供单元测试验证长文本没有在显示层被改写或截断。
    func configureTextForLayoutTest(_ text: String) {
        clearContent()
        showText(text, mine: false, secondary: false)
    }

    var displayedBodyTextForTesting: String {
        bodyLabel.attributedText?.string ?? bodyLabel.text ?? ""
    }

    var bodyUsesUnlimitedCharacterWrappingForTesting: Bool {
        bodyLabel.numberOfLines == 0 && bodyLabel.lineBreakMode == .byCharWrapping
    }
#endif

    private func showImage(url: String?, height: CGFloat = 180) {
        bodyLabel.isHidden = true
        mediaImage.isHidden = false
        bubble.backgroundColor = .clear
        LinLanImageLoader.load(urlString: url, into: mediaImage)
        mediaImageWidth.constant = 180
        mediaImageHeight.constant = height
        // 图片撑开气泡：断开 bubble→content 底边，改跟图片底边
        bubbleBottomToContent.isActive = false
        bubbleTrailToMedia.isActive = true
        bubbleBottomToMedia.isActive = true
        bubbleFixedWidth.isActive = false
        bubbleWidthCap.isActive = true
    }

    private func showSound(seconds: Int, mine: Bool, msgId: String) {
        bodyLabel.isHidden = true
        mediaIcon.isHidden = false
        mediaTitle.isHidden = false
        mediaSub.isHidden = true
        soundMsgId = msgId
        soundMine = mine
        mediaIcon.image = UIImage(systemName: Self.voiceIdleSymbol)
        mediaIcon.tintColor = mine ? .white : AppTheme.accent
        // 自己发出的语音：喇叭朝左（微信习惯）
        mediaIcon.transform = mine ? CGAffineTransform(scaleX: -1, y: 1) : .identity
        mediaTitle.text = "\(seconds)\""
        mediaTitle.textColor = mine ? .white : AppTheme.textPrimary
        bubble.backgroundColor = mine ? AppTheme.accent : .white

        mediaIconWidth.constant = 22
        mediaIconHeight.constant = 22
        mediaTitleBottom.isActive = true
        mediaSubHeight.isActive = true
        mediaSubHeight.constant = 0
        mediaSubBottom.isActive = false

        bubbleWidthCap.isActive = false
        bubbleFixedWidth.constant = min(200, max(88, CGFloat(seconds) * 5 + 72))
        bubbleFixedWidth.isActive = true

        setVoicePlaying(LinLanChatMediaPresenter.playingSoundMsgId == msgId)
    }

    private func setVoicePlaying(_ playing: Bool) {
        if playing {
            startVoiceAnim()
        } else {
            stopVoiceAnim()
            mediaIcon.image = UIImage(systemName: Self.voiceIdleSymbol)
            mediaIcon.transform = soundMine ? CGAffineTransform(scaleX: -1, y: 1) : .identity
        }
    }

    private func startVoiceAnim() {
        stopVoiceAnim()
        voiceAnimStep = 0
        applyVoiceAnimFrame()
        let timer = Timer(timeInterval: 0.32, repeats: true) { [weak self] _ in
            guard let self else { return }
            self.voiceAnimStep = (self.voiceAnimStep + 1) % Self.voicePlaySymbols.count
            self.applyVoiceAnimFrame()
        }
        RunLoop.main.add(timer, forMode: .common)
        voiceAnimTimer = timer
    }

    private func stopVoiceAnim() {
        voiceAnimTimer?.invalidate()
        voiceAnimTimer = nil
        voiceAnimStep = 0
    }

    private func applyVoiceAnimFrame() {
        let name = Self.voicePlaySymbols[voiceAnimStep]
        mediaIcon.image = UIImage(systemName: name)
        mediaIcon.transform = soundMine ? CGAffineTransform(scaleX: -1, y: 1) : .identity
    }

    private func showFile(icon: String, title: String, sub: String, mine: Bool) {
        bodyLabel.isHidden = true
        mediaIcon.isHidden = false
        mediaTitle.isHidden = false
        mediaSub.isHidden = false
        mediaIcon.image = UIImage(systemName: icon)
        mediaIcon.tintColor = mine ? .white : AppTheme.accent
        mediaTitle.text = title
        mediaTitle.textColor = mine ? .white : AppTheme.textPrimary
        mediaSub.text = sub
        mediaSub.textColor = mine ? UIColor.white.withAlphaComponent(0.85) : AppTheme.textSecondary
        bubble.backgroundColor = mine ? AppTheme.accent : .white

        mediaIconWidth.constant = 28
        mediaIconHeight.constant = 28
        mediaTitleBottom.isActive = false
        mediaSubHeight.isActive = false
        mediaSubBottom.isActive = true
        bubbleFixedWidth.isActive = false
        bubbleWidthCap.isActive = true
    }

    private func showMerger(_ card: LinLanMergerCard) {
        bodyLabel.isHidden = true
        mergerStack.isHidden = false
        bubble.backgroundColor = .white
        bubble.layer.borderWidth = 1
        bubble.layer.borderColor = UIColor(white: 0.87, alpha: 1).cgColor

        mergerWidth.constant = 200
        mergerTrailing.isActive = true
        mergerBottom.isActive = true
        bubbleFixedWidth.isActive = false
        bubbleWidthCap.isActive = true

        let title = UILabel()
        title.translatesAutoresizingMaskIntoConstraints = false
        title.text = card.title
        title.font = .systemFont(ofSize: 15, weight: .semibold)
        title.textColor = AppTheme.textPrimary
        title.numberOfLines = 2
        mergerStack.addArrangedSubview(title)

        for line in card.abstracts.prefix(4) {
            let l = UILabel()
            l.translatesAutoresizingMaskIntoConstraints = false
            l.text = line
            l.font = .systemFont(ofSize: 12)
            l.textColor = AppTheme.textSecondary
            l.numberOfLines = 1
            mergerStack.addArrangedSubview(l)
        }

        let divider = UIView()
        divider.translatesAutoresizingMaskIntoConstraints = false
        divider.backgroundColor = AppTheme.divider
        divider.heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
        mergerStack.addArrangedSubview(divider)

        let foot = UILabel()
        foot.translatesAutoresizingMaskIntoConstraints = false
        foot.text = "聊天记录"
        foot.font = .systemFont(ofSize: 12)
        foot.textColor = AppTheme.textSecondary
        mergerStack.addArrangedSubview(foot)
    }

    override func prepareForReuse() {
        super.prepareForReuse()
        onLongPress = nil
        onTapMedia = nil
        onTapAvatar = nil
        onLongPressAvatar = nil
        onTapQuote = nil
        stopVoiceAnim()
        soundMsgId = nil
        clearContent()
    }
}
