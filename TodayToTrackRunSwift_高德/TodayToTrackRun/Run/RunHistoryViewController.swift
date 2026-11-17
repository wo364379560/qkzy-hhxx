//
//  RunHistoryViewController.swift
//  TodayToTrackRun
//

import UIKit

final class RunHistoryViewController: UIViewController {
    private let tableView = UITableView(frame: .zero, style: .plain)
    private var records: [RunRecordSummary] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "跑步记录"
        view.backgroundColor = .systemBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.rowHeight = 88
        tableView.register(RunHistoryCell.self, forCellReuseIdentifier: RunHistoryCell.reuseID)
        view.addSubview(tableView)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reload()
    }

    private func reload() {
        do {
            records = try RunRecordStore.listSummaries()
            tableView.reloadData()
            navigationItem.prompt = records.isEmpty ? "暂无本地记录" : nil
        } catch {
            let alert = UIAlertController(title: "读取失败", message: error.localizedDescription, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
        }
    }
}

extension RunHistoryViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        records.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: RunHistoryCell.reuseID, for: indexPath) as! RunHistoryCell
        cell.configure(records[indexPath.row])
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let record = records[indexPath.row]
        do {
            let detail = try RunRecordStore.detail(runID: record.id)
            let vc = RunDetailViewController(detail: detail)
            linLanPush(vc)
        } catch {
            let alert = UIAlertController(title: "打开失败", message: error.localizedDescription, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "好", style: .default))
            present(alert, animated: true)
        }
    }

    func tableView(
        _ tableView: UITableView,
        trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath
    ) -> UISwipeActionsConfiguration? {
        let delete = UIContextualAction(style: .destructive, title: "删除") { [weak self] _, _, done in
            guard let self else { done(false); return }
            let id = self.records[indexPath.row].id
            do {
                try RunRecordStore.delete(runID: id)
                self.records.remove(at: indexPath.row)
                tableView.deleteRows(at: [indexPath], with: .automatic)
                done(true)
            } catch {
                done(false)
            }
        }
        return UISwipeActionsConfiguration(actions: [delete])
    }
}

private final class RunHistoryCell: UITableViewCell {
    static let reuseID = "RunHistoryCell"
    private let thumbnail = TrackThumbnailView()
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        accessoryType = .disclosureIndicator
        titleLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        subtitleLabel.font = .systemFont(ofSize: 13)
        subtitleLabel.textColor = .secondaryLabel
        subtitleLabel.numberOfLines = 2

        contentView.addSubview(thumbnail)
        contentView.addSubview(titleLabel)
        contentView.addSubview(subtitleLabel)
        thumbnail.translatesAutoresizingMaskIntoConstraints = false
        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false

        NSLayoutConstraint.activate([
            thumbnail.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
            thumbnail.centerYAnchor.constraint(equalTo: contentView.centerYAnchor),
            thumbnail.widthAnchor.constraint(equalToConstant: 64),
            thumbnail.heightAnchor.constraint(equalToConstant: 64),

            titleLabel.leadingAnchor.constraint(equalTo: thumbnail.trailingAnchor, constant: 12),
            titleLabel.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -8),
            titleLabel.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 18),

            subtitleLabel.leadingAnchor.constraint(equalTo: titleLabel.leadingAnchor),
            subtitleLabel.trailingAnchor.constraint(equalTo: titleLabel.trailingAnchor),
            subtitleLabel.topAnchor.constraint(equalTo: titleLabel.bottomAnchor, constant: 6)
        ])
    }

    required init?(coder: NSCoder) { fatalError() }

    func configure(_ record: RunRecordSummary) {
        let formatter = DateFormatter()
        formatter.dateFormat = "M月d日 HH:mm"
        titleLabel.text = "\(record.formattedDistance) · \(formatter.string(from: record.startedAt))"
        let place = record.cityName?.isEmpty == false ? record.cityName! : "本地跑步"
        let syncTag = record.isSynced ? "已同步" : "未同步"
        let kcal = record.caloriesKcal > 0
            ? record.caloriesKcal
            : RunMetricsEstimator.estimatedCalories(
                distanceMeters: record.distanceMeters,
                durationSeconds: record.durationSeconds
            )
        subtitleLabel.text = "\(place) · \(syncTag)\n\(record.formattedDuration) · \(record.formattedPace) · \(kcal) kcal"
        thumbnail.coordinates = record.previewCoordinates
    }
}
