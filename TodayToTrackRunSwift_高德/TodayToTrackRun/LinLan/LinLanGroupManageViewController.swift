//
//  LinLanGroupManageViewController.swift
//  对照睿之禾 GroupManageScreen
//

import SnapKit
import UIKit

final class LinLanGroupManageViewController: UIViewController {
    private let groupId: String
    private var info: LinLanGroupInfoSnapshot?
    private var attrs: [String: String] = [:]

    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let muteAllSwitch = UISwitch()
    private let muteAdminAlsoSwitch = UISwitch()
    private let profileAccessSwitch = UISwitch()
    private let voiceSwitch = UISwitch()
    private let videoSwitch = UISwitch()

    init(groupId: String) {
        self.groupId = groupId
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "群管理"
        view.backgroundColor = AppTheme.pageBackground
        navigationItem.leftBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "chevron.left"),
            style: .plain,
            target: self,
            action: #selector(back)
        )
        tableView.dataSource = self
        tableView.delegate = self
        tableView.backgroundColor = AppTheme.pageBackground
        view.addSubview(tableView)
        tableView.snp.makeConstraints { $0.edges.equalTo(view.safeAreaLayoutGuide) }

        for sw in [muteAllSwitch, muteAdminAlsoSwitch, profileAccessSwitch, voiceSwitch, videoSwitch] {
            sw.onTintColor = AppTheme.accent
        }
        muteAllSwitch.addTarget(self, action: #selector(muteAllChanged), for: .valueChanged)
        muteAdminAlsoSwitch.addTarget(self, action: #selector(muteAdminAlsoChanged), for: .valueChanged)
        profileAccessSwitch.addTarget(self, action: #selector(profileAccessChanged), for: .valueChanged)
        voiceSwitch.addTarget(self, action: #selector(voiceChanged), for: .valueChanged)
        videoSwitch.addTarget(self, action: #selector(videoChanged), for: .valueChanged)

        Task { await reload() }
    }

    private func reload() async {
        info = try? await LinLanImManager.fetchGroupInfo(groupId: groupId)
        attrs = await LinLanGroupPolicy.fetchAll(groupId: groupId)
        muteAllSwitch.isOn = info?.allMuted == true
        muteAdminAlsoSwitch.isOn = LinLanGroupPolicy.isMuteAdminAlsoEnabled(attrs)
        profileAccessSwitch.isOn = LinLanGroupPolicy.isMemberProfileAccessEnabled(attrs)
        voiceSwitch.isOn = LinLanGroupPolicy.isVoiceCallEnabled(attrs)
        videoSwitch.isOn = LinLanGroupPolicy.isVideoCallEnabled(attrs)
        tableView.reloadData()
    }

    @objc private func back() { navigationController?.popViewController(animated: true) }

    @objc private func muteAllChanged() {
        let on = muteAllSwitch.isOn
        Task {
            do {
                try await LinLanImManager.muteAllGroupMembers(groupId: groupId, muted: on)
            } catch {
                await MainActor.run {
                    self.muteAllSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func muteAdminAlsoChanged() {
        let on = muteAdminAlsoSwitch.isOn
        Task {
            do {
                try await LinLanGroupPolicy.updateMuteAdminAlso(groupId: groupId, enabled: on)
            } catch {
                await MainActor.run {
                    self.muteAdminAlsoSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func profileAccessChanged() {
        let on = profileAccessSwitch.isOn
        Task {
            do {
                try await LinLanGroupPolicy.updateMemberProfileAccess(groupId: groupId, enabled: on)
            } catch {
                await MainActor.run {
                    self.profileAccessSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func voiceChanged() {
        let on = voiceSwitch.isOn
        Task {
            do {
                try await LinLanGroupPolicy.updateVoiceCallAccess(groupId: groupId, enabled: on)
            } catch {
                await MainActor.run {
                    self.voiceSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    @objc private func videoChanged() {
        let on = videoSwitch.isOn
        Task {
            do {
                try await LinLanGroupPolicy.updateVideoCallAccess(groupId: groupId, enabled: on)
            } catch {
                await MainActor.run {
                    self.videoSwitch.isOn = !on
                    self.toast(error.localizedDescription)
                }
            }
        }
    }

    private func toast(_ msg: String) {
        let a = UIAlertController(title: nil, message: msg, preferredStyle: .alert)
        present(a, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { [weak a] in a?.dismiss(animated: true) }
    }

    private var isOwner: Bool {
        info?.role == LinLanImManager.roleOwner || info?.role == 400
    }
}

extension LinLanGroupManageViewController: UITableViewDataSource, UITableViewDelegate {
    func numberOfSections(in tableView: UITableView) -> Int { 3 }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        switch section {
        case 0: return 2
        case 1: return 1
        default: return isOwner ? 2 : 0
        }
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        switch section {
        case 0: return "发言管理"
        case 1: return "成员资料"
        case 2: return isOwner ? "群通话（需开通）" : nil
        default: return nil
        }
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        cell.selectionStyle = .none
        cell.backgroundColor = .white
        switch (indexPath.section, indexPath.row) {
        case (0, 0):
            cell.textLabel?.text = "全员禁言"
            cell.accessoryView = muteAllSwitch
        case (0, 1):
            cell.textLabel?.text = "管理员也禁言"
            cell.accessoryView = muteAdminAlsoSwitch
        case (1, 0):
            cell.textLabel?.text = "允许查看成员资料"
            cell.accessoryView = profileAccessSwitch
        case (2, 0):
            cell.textLabel?.text = "允许群语音通话"
            cell.accessoryView = voiceSwitch
        case (2, 1):
            cell.textLabel?.text = "允许群视频通话"
            cell.accessoryView = videoSwitch
        default:
            break
        }
        return cell
    }
}
