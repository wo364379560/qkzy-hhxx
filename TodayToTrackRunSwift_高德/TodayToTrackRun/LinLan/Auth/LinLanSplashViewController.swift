//
//  LinLanSplashViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class LinLanSplashViewController: UIViewController {
    private let brandColor = UIColor(red: 0.03, green: 0.25, blue: 0.21, alpha: 1)
    private var loginDecision: Bool?
    private var animationFinished = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.pageBackground

        let characterStack = UIStackView()
        characterStack.axis = .horizontal
        characterStack.alignment = .center
        characterStack.spacing = 9

        let characterLabels = LinLanAppConstants.appName.map { character -> UILabel in
            let label = UILabel()
            label.text = String(character)
            label.font = .systemFont(ofSize: 50, weight: .semibold)
            label.textColor = brandColor
            label.alpha = 0
            label.transform = CGAffineTransform(translationX: -8, y: 5)
                .scaledBy(x: 0.92, y: 0.92)
            characterStack.addArrangedSubview(label)
            return label
        }

        let strokeView = UIView()
        strokeView.backgroundColor = brandColor
        strokeView.layer.cornerRadius = 1.5
        strokeView.transform = CGAffineTransform(scaleX: 0.001, y: 1)

        let tip = UILabel()
        tip.text = "一起运动，一路畅聊"
        tip.font = .systemFont(ofSize: 14)
        tip.textColor = AppTheme.textSecondary
        tip.textAlignment = .center
        tip.alpha = 0

        let brandContainer = UIView()
        brandContainer.addSubview(characterStack)
        brandContainer.addSubview(strokeView)
        characterStack.snp.makeConstraints { make in
            make.top.centerX.equalToSuperview()
        }
        strokeView.snp.makeConstraints { make in
            make.top.equalTo(characterStack.snp.bottom).offset(10)
            make.leading.trailing.equalTo(characterStack).inset(5)
            make.height.equalTo(3)
            make.bottom.equalToSuperview()
        }

        let stack = UIStackView(arrangedSubviews: [brandContainer, tip])
        stack.axis = .vertical
        stack.spacing = 20
        view.addSubview(stack)
        stack.snp.makeConstraints { $0.center.equalToSuperview() }

        Task { [weak self] in
            let loggedIn = await LinLanAppContainer.authRepository().decideSplashLoggedIn()
            await MainActor.run {
                self?.loginDecision = loggedIn
                self?.finishIfReady()
            }
        }

        animateBrand(characterLabels: characterLabels, strokeView: strokeView, tip: tip)
    }

    private func animateBrand(characterLabels: [UILabel], strokeView: UIView, tip: UILabel) {
        for (index, label) in characterLabels.enumerated() {
            UIView.animate(
                withDuration: 0.42,
                delay: 0.16 + Double(index) * 0.28,
                usingSpringWithDamping: 0.82,
                initialSpringVelocity: 0.15,
                options: [.curveEaseOut]
            ) {
                label.alpha = 1
                label.transform = .identity
            }
        }

        let strokeDelay = 0.16 + Double(characterLabels.count) * 0.28
        UIView.animate(
            withDuration: 0.5,
            delay: strokeDelay,
            options: [.curveEaseInOut]
        ) {
            strokeView.transform = .identity
        } completion: { [weak self] _ in
            UIView.animate(withDuration: 0.3) {
                tip.alpha = 1
            } completion: { _ in
                self?.animationFinished = true
                self?.finishIfReady()
            }
        }
    }

    private func finishIfReady() {
        guard animationFinished, let loggedIn = loginDecision else { return }
        LinLanRootSwitcher.applyRoot(loggedIn: loggedIn, animated: true)
    }
}
