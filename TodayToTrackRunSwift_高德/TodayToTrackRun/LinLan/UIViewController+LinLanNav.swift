//
//  UIViewController+LinLanNav.swift
//  TodayToTrackRun
//

import UIKit

extension UIViewController {
    /// 统一 push，并隐藏底部 TabBar
    func linLanPush(_ viewController: UIViewController, animated: Bool = true) {
        viewController.hidesBottomBarWhenPushed = true
        navigationController?.pushViewController(viewController, animated: animated)
    }
}
