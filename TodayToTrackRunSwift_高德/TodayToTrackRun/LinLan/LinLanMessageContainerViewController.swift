//
//  LinLanMessageContainerViewController.swift
//  TodayToTrackRun
//
//  消息 Tab：顶部 Segment「消息 | 通讯录」
//

import SnapKit
import UIKit

final class LinLanMessageContainerViewController: UIViewController {
    private let segment = UISegmentedControl(items: ["消息", "通讯录"])
    private let container = UIView()
    private lazy var conversationVC = LinLanConversationViewController()
    private lazy var contactVC = LinLanContactViewController()
    private weak var currentChild: UIViewController?

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "消息"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.largeTitleDisplayMode = .never
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)

        navigationItem.rightBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "plus"),
            style: .plain,
            target: self,
            action: #selector(linLanPlusTapped)
        )

        segment.selectedSegmentIndex = 0
        segment.selectedSegmentTintColor = AppTheme.accent
        segment.setTitleTextAttributes([.foregroundColor: UIColor.white], for: .selected)
        segment.setTitleTextAttributes([.foregroundColor: AppTheme.textPrimary], for: .normal)
        segment.addTarget(self, action: #selector(linLanSegmentChanged), for: .valueChanged)

        view.addSubview(segment)
        view.addSubview(container)
        segment.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(8)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(34)
        }
        container.snp.makeConstraints { make in
            make.top.equalTo(segment.snp.bottom).offset(10)
            make.leading.trailing.bottom.equalToSuperview()
        }

        linLanShowChild(conversationVC)
    }

    @objc private func linLanSegmentChanged() {
        linLanShowChild(segment.selectedSegmentIndex == 0 ? conversationVC : contactVC)
        title = segment.selectedSegmentIndex == 0 ? "消息" : "通讯录"
    }

    private func linLanShowChild(_ child: UIViewController) {
        if currentChild === child { return }
        currentChild?.willMove(toParent: nil)
        currentChild?.view.removeFromSuperview()
        currentChild?.removeFromParent()

        addChild(child)
        container.addSubview(child.view)
        child.view.snp.makeConstraints { $0.edges.equalToSuperview() }
        child.didMove(toParent: self)
        currentChild = child
    }

    @objc private func linLanPlusTapped() {
        let sheet = UIAlertController(title: nil, message: nil, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "添加好友", style: .default) { [weak self] _ in
            self?.linLanPush(LinLanAddFriendViewController())
        })
        sheet.addAction(UIAlertAction(title: "新建群", style: .default) { [weak self] _ in
            self?.linLanPush(LinLanCreateGroupViewController())
        })
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }
}
