//
//  CustomGoalEditorViewController.swift
//  TodayToTrackRun
//

import SnapKit
import UIKit

final class CustomGoalEditorViewController: UIViewController {
    private let titleField = UITextField()
    private let subtitleField = UITextField()
    private let editingIndex: Int?

    init(editingIndex: Int? = nil) {
        self.editingIndex = editingIndex
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = editingIndex == nil ? "新建目标" : "编辑目标"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            title: "保存",
            style: .done,
            target: self,
            action: #selector(save)
        )

        let card = UIView()
        card.backgroundColor = .white
        card.layer.cornerRadius = 12
        view.addSubview(card)
        card.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(20)
            make.leading.trailing.equalToSuperview().inset(16)
        }

        configure(field: titleField, placeholder: "目标名称，如：目标 4 · 晨跑习惯")
        configure(field: subtitleField, placeholder: "说明，如：每周 4 次，每次 40 分钟")
        if let editingIndex, GoalPreferences.presets.indices.contains(editingIndex) {
            let g = GoalPreferences.presets[editingIndex]
            titleField.text = g.title
            subtitleField.text = g.subtitle
        }

        let stack = UIStackView(arrangedSubviews: [titleField, subtitleField])
        stack.axis = .vertical
        stack.spacing = 12
        card.addSubview(stack)
        stack.snp.makeConstraints { $0.edges.equalToSuperview().inset(14) }
    }

    private func configure(field: UITextField, placeholder: String) {
        field.placeholder = placeholder
        field.borderStyle = .roundedRect
        field.backgroundColor = UIColor(white: 0.97, alpha: 1)
        field.font = .systemFont(ofSize: 15)
    }

    @objc private func save() {
        let titleText = titleField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let subtitleText = subtitleField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !titleText.isEmpty else {
            let alert = UIAlertController(title: "请填写目标名称", message: nil, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
            return
        }
        GoalPreferences.upsertGoal(
            GoalPreset(title: titleText, subtitle: subtitleText.isEmpty ? "自定义目标" : subtitleText),
            at: editingIndex
        )
        navigationController?.popViewController(animated: true)
    }
}
