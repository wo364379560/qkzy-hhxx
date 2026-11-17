//
//  LinLanPlaceholderViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanPlaceholderViewController: UIViewController {
    private let titleText: String
    private let hint: String

    init(titleText: String, hint: String) {
        self.titleText = titleText
        self.hint = hint
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
        let label = UILabel()
        label.text = hint
        label.textAlignment = .center
        label.numberOfLines = 0
        label.textColor = AppTheme.textSecondary
        label.font = .systemFont(ofSize: 15)
        view.addSubview(label)
        label.snp.makeConstraints { $0.center.equalToSuperview().inset(32); $0.leading.trailing.equalToSuperview().inset(32) }
    }

    @objc private func linLanBack() { navigationController?.popViewController(animated: true) }
}
