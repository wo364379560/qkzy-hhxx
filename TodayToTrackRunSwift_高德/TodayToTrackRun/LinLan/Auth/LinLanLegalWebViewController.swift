//
//  LinLanLegalWebViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit
import WebKit

final class LinLanLegalWebViewController: UIViewController {
    private let urlString: String
    private let webView = WKWebView()

    init(titleText: String, urlString: String) {
        self.urlString = urlString
        super.init(nibName: nil, bundle: nil)
        title = titleText
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground
        AppTheme.applyFrostedNav(to: navigationController?.navigationBar)
        view.addSubview(webView)
        webView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }
        if let url = URL(string: urlString) {
            webView.load(URLRequest(url: url))
        }
    }
}
