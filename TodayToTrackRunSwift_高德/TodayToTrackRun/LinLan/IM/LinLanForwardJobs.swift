import CryptoKit
import Foundation
import ImSDK_Plus_Swift
import UIKit

enum LinLanForwardFailure: LocalizedError {
    case notDispatched(String)
    var errorDescription: String? { if case .notDispatched(let reason) = self { return reason }; return nil }
}

struct LinLanForwardItemResult: Codable {
    let target: String
    let name: String
    let index: Int
    let sourceID: String
    var status = "pending"
    var error = ""
    var sdkID = ""
}

struct LinLanNativeForwardJob: Codable {
    let id: String
    let owner: String
    let sdkAppID: Int
    var items: [LinLanForwardItemResult]
    var running = false
    var resumable = true
    var serverRequestID: String?
    var serverState: String?
    var serverNotice: String?
}

private struct LinLanServerForwardPlan {
    let requestID: String
    let targets: [[String: Any]]
    let messages: [[String: Any]]

    var contentHash: String {
        let data = (try? JSONSerialization.data(
            withJSONObject: ["targets": targets, "messages": messages],
            options: [.sortedKeys]
        )) ?? Data()
        return SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
    }

    func encoded() -> Data? {
        try? JSONSerialization.data(withJSONObject: [
            "request_id": requestID,
            "targets": targets,
            "messages": messages,
        ], options: [.sortedKeys])
    }

    init?(data: Data) {
        guard let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let requestID = root["request_id"] as? String,
              let targets = root["targets"] as? [[String: Any]],
              let messages = root["messages"] as? [[String: Any]],
              !requestID.isEmpty, !targets.isEmpty, !messages.isEmpty else { return nil }
        self.requestID = requestID
        self.targets = targets
        self.messages = messages
    }

    init(requestID: String, targets: [[String: Any]], messages: [[String: Any]]) {
        self.requestID = requestID
        self.targets = targets
        self.messages = messages
    }
}

private enum LinLanServerForward {
    static let targetChunkSize = 20
    private static let messageChunkSize = 100

    static func makePlan(messages: [V2TIMMessage], targets: [LinLanForwardTarget]) async -> LinLanServerForwardPlan? {
        let targetRows: [[String: Any]] = targets.compactMap { target in
            let conversationID: String
            if target.isGroup, let groupID = target.groupId, !groupID.isEmpty {
                conversationID = "GROUP\(groupID)"
            } else if let receiver = target.receiver, !receiver.isEmpty {
                conversationID = "C2C\(receiver)"
            } else {
                return nil
            }
            return ["conversation_id": conversationID, "name": String(target.title.prefix(128))]
        }
        guard targetRows.count == targets.count else { return nil }

        var messageRows: [[String: Any]] = []
        for message in messages {
            guard let row = await map(message) else { return nil }
            messageRows.append(row)
        }
        guard !messageRows.isEmpty else { return nil }
        return LinLanServerForwardPlan(
            requestID: UUID().uuidString.replacingOccurrences(of: "-", with: ""),
            targets: targetRows,
            messages: messageRows
        )
    }

    static func submit(
        _ plan: LinLanServerForwardPlan,
        progress: @escaping (String) -> Void
    ) async throws -> [String: Any] {
        let api = LinLanAppContainer.apiClient()
        var status = try await retrying {
            try await api.initForwardJob([
                "request_id": plan.requestID,
                "content_hash": plan.contentHash,
                "target_count": plan.targets.count,
                "message_count": plan.messages.count,
            ])
        }
        var targetOffset = integer(status["uploaded_targets"])
        var messageOffset = integer(status["uploaded_messages"])
        guard targetOffset <= plan.targets.count, messageOffset <= plan.messages.count else {
            throw LinLanForwardFailure.notDispatched("服务器返回的批量进度无效")
        }

        while targetOffset < plan.targets.count {
            let end = min(targetOffset + targetChunkSize, plan.targets.count)
            let offset = targetOffset
            status = try await retrying {
                try await api.uploadForwardTargets(
                    requestID: plan.requestID,
                    offset: offset,
                    targets: Array(plan.targets[offset..<end])
                )
            }
            let confirmed = integer(status["uploaded_targets"])
            targetOffset = confirmed > offset ? confirmed : end
            progress("正在分批提交会话 \(targetOffset)/\(plan.targets.count)（每批 \(targetChunkSize) 个）")
        }

        while messageOffset < plan.messages.count {
            let end = min(messageOffset + messageChunkSize, plan.messages.count)
            let offset = messageOffset
            status = try await retrying {
                try await api.uploadForwardMessages(
                    requestID: plan.requestID,
                    offset: offset,
                    messages: Array(plan.messages[offset..<end])
                )
            }
            let confirmed = integer(status["uploaded_messages"])
            messageOffset = confirmed > offset ? confirmed : end
        }
        return try await retrying { try await api.commitForwardJob(requestID: plan.requestID) }
    }

    private static func map(_ message: V2TIMMessage) async -> [String: Any]? {
        let msgType: String
        let content: [String: Any]
        switch message.elemType {
        case .V2TIM_ELEM_TYPE_TEXT:
            guard let text = message.textElem?.text,
                  !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return nil }
            msgType = "TIMTextElem"
            content = ["Text": text]
        case .V2TIM_ELEM_TYPE_CUSTOM:
            guard let elem = message.customElem, let data = elem.data else { return nil }
            msgType = "TIMCustomElem"
            content = [
                "Data": string(data),
                "Desc": elem.desc ?? "",
                "Ext": elem.ext ?? "",
            ]
        case .V2TIM_ELEM_TYPE_LOCATION:
            guard let elem = message.locationElem,
                  elem.latitude.isFinite, abs(elem.latitude) <= 90,
                  elem.longitude.isFinite, abs(elem.longitude) <= 180 else { return nil }
            msgType = "TIMLocationElem"
            content = ["Desc": elem.desc ?? "", "Latitude": elem.latitude, "Longitude": elem.longitude]
        case .V2TIM_ELEM_TYPE_FACE:
            guard let elem = message.faceElem, elem.index >= 0 else { return nil }
            msgType = "TIMFaceElem"
            var value: [String: Any] = ["Index": elem.index]
            if let data = elem.data { value["Data"] = string(data) }
            content = value
        case .V2TIM_ELEM_TYPE_IMAGE:
            guard let elem = message.imageElem, !elem.imageList.isEmpty, elem.imageList.count <= 3,
                  let uuid = elem.imageList.compactMap(\.uuid).first, !uuid.isEmpty else { return nil }
            var seen = Set<Int>()
            var images: [[String: Any]] = []
            for image in elem.imageList {
                let type = [1, 3, 2][min(max(image.type.rawValue, 0), 2)]
                guard seen.insert(type).inserted, let url = image.url, validURL(url) else { return nil }
                images.append([
                    "Type": type, "Size": max(0, image.size), "Width": max(0, image.width),
                    "Height": max(0, image.height), "URL": url,
                ])
            }
            images.sort { ($0["Type"] as? Int ?? 0) < ($1["Type"] as? Int ?? 0) }
            msgType = "TIMImageElem"
            let sampleURL = elem.imageList.compactMap(\.url).first ?? ""
            content = [
                "UUID": uuid,
                "ImageFormat": imageFormat(path: elem.path ?? sampleURL),
                "ImageInfoArray": images,
            ]
        case .V2TIM_ELEM_TYPE_FILE:
            guard let elem = message.fileElem, let uuid = elem.uuid, !uuid.isEmpty,
                  let name = elem.filename, !name.isEmpty,
                  let url = await fileURL(elem), validURL(url) else { return nil }
            msgType = "TIMFileElem"
            content = [
                "UUID": uuid, "Url": url, "FileSize": max(0, elem.fileSize),
                "FileName": name, "Download_Flag": 2,
            ]
        case .V2TIM_ELEM_TYPE_SOUND:
            guard let elem = message.soundElem, let uuid = elem.uuid, !uuid.isEmpty,
                  let url = await soundURL(elem), validURL(url) else { return nil }
            msgType = "TIMSoundElem"
            content = [
                "UUID": uuid, "Url": url, "Size": max(0, elem.dataSize),
                "Second": max(0, elem.duration), "Download_Flag": 2,
            ]
        case .V2TIM_ELEM_TYPE_VIDEO:
            guard let elem = message.videoElem,
                  let videoUUID = elem.videoUUID, !videoUUID.isEmpty,
                  let thumbUUID = elem.snapshotUUID, !thumbUUID.isEmpty,
                  let videoURL = await videoURL(elem), validURL(videoURL),
                  let thumbURL = await snapshotURL(elem), validURL(thumbURL) else { return nil }
            msgType = "TIMVideoFileElem"
            content = [
                "VideoUUID": videoUUID, "VideoUrl": videoURL, "VideoSize": max(0, elem.videoSize),
                "VideoSecond": max(0, elem.duration), "VideoFormat": elem.videoType ?? "",
                "VideoDownloadFlag": 2, "ThumbUUID": thumbUUID, "ThumbUrl": thumbURL,
                "ThumbSize": max(0, elem.snapshotSize), "ThumbWidth": max(0, elem.snapshotWidth),
                "ThumbHeight": max(0, elem.snapshotHeight), "ThumbFormat": "", "ThumbDownloadFlag": 2,
            ]
        default:
            return nil
        }
        let body: [[String: Any]] = [["MsgType": msgType, "MsgContent": content]]
        guard let data = try? JSONSerialization.data(withJSONObject: body), data.count <= 10_000 else { return nil }
        return ["body": body, "preview": String(LinLanImManager.messagePreview(message).prefix(120))]
    }

    private static func retrying(_ operation: () async throws -> [String: Any]) async throws -> [String: Any] {
        var attempt = 0
        while true {
            do { return try await operation() }
            catch {
                if case LinLanAPIError.http(_, let status, _) = error,
                   [400, 401, 403, 404, 409, 413].contains(status) { throw error }
                if case LinLanAPIError.message = error { throw error }
                guard attempt < 5 else { throw error }
                let delay = min(8.0, 0.5 * pow(2.0, Double(attempt)))
                attempt += 1
                try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            }
        }
    }

    static func integer(_ value: Any?) -> Int {
        if let value = value as? Int { return value }
        if let value = value as? NSNumber { return value.intValue }
        if let value = value as? String { return Int(value) ?? 0 }
        return 0
    }

    static func boolean(_ value: Any?) -> Bool {
        if let value = value as? Bool { return value }
        if let value = value as? NSNumber { return value.boolValue }
        return false
    }

    private static func string(_ data: Data) -> String {
        String(data: data, encoding: .utf8) ?? data.base64EncodedString()
    }

    private static func validURL(_ value: String) -> Bool {
        guard let url = URL(string: value), let scheme = url.scheme?.lowercased() else { return false }
        return (scheme == "https" || scheme == "http") && url.host?.isEmpty == false
    }

    private static func imageFormat(path: String) -> Int {
        switch URL(string: path)?.pathExtension.lowercased() ?? (path as NSString).pathExtension.lowercased() {
        case "jpg", "jpeg": return 1
        case "gif": return 2
        case "png": return 3
        case "bmp": return 4
        default: return 255
        }
    }

    private static func fileURL(_ elem: V2TIMFileElem) async -> String? {
        await withCheckedContinuation { continuation in elem.getUrl { continuation.resume(returning: $0) } }
    }

    private static func soundURL(_ elem: V2TIMSoundElem) async -> String? {
        await withCheckedContinuation { continuation in elem.getUrl { continuation.resume(returning: $0) } }
    }

    private static func videoURL(_ elem: V2TIMVideoElem) async -> String? {
        await withCheckedContinuation { continuation in elem.getVideoUrl { continuation.resume(returning: $0) } }
    }

    private static func snapshotURL(_ elem: V2TIMVideoElem) async -> String? {
        await withCheckedContinuation { continuation in elem.getSnapshotUrl { continuation.resume(returning: $0) } }
    }
}

@MainActor
final class LinLanForwardJobs {
    static let shared = LinLanForwardJobs()
    static let changed = Notification.Name("rui.forward.results.changed")
    private(set) var current: LinLanNativeForwardJob?
    private var sources: [V2TIMMessage] = []
    private var targets: [LinLanForwardTarget] = []
    private var merged = false
    private var title = ""
    private var stopped = false
    private var loadedOwner = ""

    @discardableResult
    func restore() -> Bool {
        let session = LinLanAppContainer.sessionStore().currentSession()
        let owner = "\(session.resolvedSdkAppId):\(session.userId)"
        guard session.isLoggedIn, loadedOwner != owner else { return false }
        loadedOwner = owner
        sources = []
        targets = []
        current = nil
        guard let data = UserDefaults.standard.data(forKey: "rui.forward.results.\(owner)"),
              var job = try? JSONDecoder().decode(LinLanNativeForwardJob.self, from: data) else { return false }
        if let requestID = job.serverRequestID, !requestID.isEmpty {
            job.running = true
            job.resumable = false
            job.serverNotice = "正在恢复服务器批量任务"
            current = job
            publish()
            if let plan = loadPlan(for: job) {
                launchServer(plan, localJobID: job.id)
            } else {
                Task { await monitorServerJob(localJobID: job.id, requestID: requestID) }
            }
            return true
        }
        job.running = false
        job.resumable = false
        for index in job.items.indices {
            if job.items[index].status == "sending" {
                job.items[index].status = "unknown"
                job.items[index].error = "应用已重启，请核对接收端；不会自动重发"
            } else if job.items[index].status == "pending" {
                job.items[index].status = "interrupted"
                job.items[index].error = "应用已重启，未自动恢复发送"
            }
        }
        current = job
        publish()
        return job.items.contains { ["unknown", "interrupted", "failed"].contains($0.status) }
    }

    func start(_ list: [V2TIMMessage], to selected: [LinLanForwardTarget], merge: Bool, title: String) -> Bool {
        guard current?.running != true, !list.isEmpty, !selected.isEmpty else { return false }
        let session = LinLanAppContainer.sessionStore().currentSession()
        guard session.isLoggedIn, LinLanImManager.loginUser == session.userId else { return false }
        sources = list.sorted { $0.timestamp == $1.timestamp ? $0.seq < $1.seq : $0.timestamp < $1.timestamp }
        var seen = Set<String>()
        targets = selected.filter { seen.insert($0.key).inserted }
        merged = merge
        self.title = title
        let count = merge ? 1 : sources.count
        let rows = targets.flatMap { target in (0..<count).map { index in
            LinLanForwardItemResult(target: target.key, name: target.title, index: index,
                sourceID: merge ? sources.map { $0.msgID }.joined(separator: ",") : sources[index].msgID)
        } }
        current = LinLanNativeForwardJob(id: UUID().uuidString, owner: session.userId, sdkAppID: session.resolvedSdkAppId, items: rows)
        loadedOwner = "\(session.resolvedSdkAppId):\(session.userId)"
        if targets.count > LinLanServerForward.targetChunkSize, !merge {
            let localJobID = current?.id ?? ""
            current?.running = true
            current?.resumable = false
            current?.serverState = "preparing"
            current?.serverNotice = "正在准备后台批量转发"
            publish()
            Task {
                let plan = await LinLanServerForward.makePlan(messages: sources, targets: targets)
                guard current?.id == localJobID else { return }
                if let plan {
                    current?.serverRequestID = plan.requestID
                    current?.serverState = "uploading"
                    savePlan(plan, for: current!)
                    publish()
                    launchServer(plan, localJobID: localJobID)
                } else {
                    // 服务器不支持的复合消息仍用 SDK 原样转发。
                    current?.running = false
                    current?.resumable = true
                    current?.serverState = nil
                    current?.serverNotice = "此消息类型已切换为原样转发"
                    publish()
                    run()
                }
            }
        } else {
            run()
        }
        return true
    }

    func cancel() {
        stopped = true
        if let requestID = current?.serverRequestID, let localJobID = current?.id {
            current?.serverNotice = "正在停止服务器未发部分"
            publish()
            Task {
                do {
                    _ = try await LinLanAppContainer.apiClient().cancelForwardJob(requestID: requestID)
                    guard current?.id == localJobID else { return }
                    clearPlan(for: current!)
                    for index in current!.items.indices where current!.items[index].status != "success" {
                        current?.items[index].status = "cancelled"
                        current?.items[index].error = "已停止，未发送"
                    }
                    current?.running = false
                    current?.serverState = "cancelled"
                    current?.serverNotice = "已停止服务器未发部分"
                    publish()
                } catch {
                    guard current?.id == localJobID else { return }
                    current?.serverNotice = "停止结果暂时无法确认：\(error.localizedDescription)"
                    publish()
                }
            }
            return
        }
        guard var job = current else { return }
        for index in job.items.indices where job.items[index].status == "pending" {
            job.items[index].status = "cancelled"
            job.items[index].error = "已停止，未发送"
        }
        current = job
        publish()
    }
    func retry() { if current?.resumable == true, current?.running == false { run() } }
    private func publish() {
        guard let current else { return }
        if let data = try? JSONEncoder().encode(current) {
            UserDefaults.standard.set(data, forKey: "rui.forward.results.\(current.sdkAppID):\(current.owner)")
        }
        NotificationCenter.default.post(name: Self.changed, object: nil)
    }
    private func update(_ position: Int, status: String, error: String = "", sdkID: String = "") {
        guard current?.items.indices.contains(position) == true else { return }
        current?.items[position].status = status
        current?.items[position].error = error
        if !sdkID.isEmpty { current?.items[position].sdkID = sdkID }
        publish()
    }

    private func launchServer(_ plan: LinLanServerForwardPlan, localJobID: String) {
        stopped = false
        current?.running = true
        publish()
        Task {
            do {
                _ = try await LinLanServerForward.submit(plan) { [weak self] notice in
                    Task { @MainActor in
                        guard self?.current?.id == localJobID else { return }
                        self?.current?.serverNotice = notice
                        self?.publish()
                    }
                }
                guard current?.id == localJobID else { return }
                clearPlan(for: current!)
                current?.serverState = "queued"
                current?.serverNotice = "任务已交给服务器，退出页面后仍会继续发送"
                for index in current!.items.indices { current?.items[index].status = "queued" }
                publish()
                await monitorServerJob(localJobID: localJobID, requestID: plan.requestID)
            } catch {
                guard current?.id == localJobID else { return }
                current?.running = false
                current?.serverState = "uploading"
                current?.serverNotice = "批量提交暂停：\(error.localizedDescription)；会使用同一任务继续，不会重复转发"
                for index in current!.items.indices where current!.items[index].status != "success" {
                    current?.items[index].status = "interrupted"
                    current?.items[index].error = "等待继续提交"
                }
                publish()
            }
        }
    }

    private func monitorServerJob(localJobID: String, requestID: String) async {
        defer {
            if current?.id == localJobID { current?.running = false; publish() }
        }
        while current?.id == localJobID, !stopped {
            do {
                let status = try await LinLanAppContainer.apiClient().forwardJobStatus(requestID: requestID)
                guard current?.id == localJobID else { return }
                let state = status["state"] as? String ?? "queued"
                let running = LinLanServerForward.boolean(status["running"])
                current?.serverState = state
                current?.serverNotice = state == "uploading"
                    ? "任务尚未提交完整，正在等待继续提交"
                    : "服务器进度：成功 \(LinLanServerForward.integer(status["success"])) / \(LinLanServerForward.integer(status["total"]))"
                await loadServerProgress(localJobID: localJobID, requestID: requestID, allPages: !running)
                publish()
                if state == "uploading" || !running { return }
                try await Task.sleep(nanoseconds: 1_500_000_000)
            } catch {
                guard current?.id == localJobID else { return }
                current?.serverNotice = "服务器任务仍在执行，进度查询暂时失败：\(error.localizedDescription)"
                publish()
                try? await Task.sleep(nanoseconds: 3_000_000_000)
            }
        }
    }

    private func loadServerProgress(localJobID: String, requestID: String, allPages: Bool) async {
        var cursor = "0"
        repeat {
            guard current?.id == localJobID else { return }
            guard let page = try? await LinLanAppContainer.apiClient().forwardJobPage(requestID: requestID, cursor: cursor) else { return }
            applyServerPage(page)
            let next = page["next_cursor"] as? String ?? ""
            if !allPages || next.isEmpty || next == cursor { return }
            cursor = next
        } while true
    }

    private func applyServerPage(_ page: [String: Any]) {
        guard let rows = page["items"] as? [[String: Any]] else { return }
        for row in rows {
            guard let conversationID = row["conversation_id"] as? String else { continue }
            let ordinal = LinLanServerForward.integer(row["index"])
            guard let position = current?.items.firstIndex(where: {
                serverConversationID(localKey: $0.target) == conversationID && $0.index == ordinal
            }) else { continue }
            current?.items[position].status = row["status"] as? String ?? "unknown"
            current?.items[position].error = row["error"] as? String ?? ""
            current?.items[position].sdkID = row["message_id"] as? String ?? ""
        }
    }

    private func serverConversationID(localKey: String) -> String {
        if localKey.hasPrefix("c2c_") { return "C2C" + localKey.dropFirst(4) }
        if localKey.hasPrefix("group_") { return "GROUP" + localKey.dropFirst(6) }
        return localKey
    }

    private func planKey(for job: LinLanNativeForwardJob) -> String {
        "rui.forward.plan.\(job.sdkAppID):\(job.owner):\(job.serverRequestID ?? job.id)"
    }

    private func savePlan(_ plan: LinLanServerForwardPlan, for job: LinLanNativeForwardJob) {
        if let data = plan.encoded() { UserDefaults.standard.set(data, forKey: planKey(for: job)) }
    }

    private func loadPlan(for job: LinLanNativeForwardJob) -> LinLanServerForwardPlan? {
        guard let data = UserDefaults.standard.data(forKey: planKey(for: job)) else { return nil }
        return LinLanServerForwardPlan(data: data)
    }

    private func clearPlan(for job: LinLanNativeForwardJob) {
        UserDefaults.standard.removeObject(forKey: planKey(for: job))
    }

    private func run() {
        guard let job = current, !job.running, !sources.isEmpty else { return }
        let selected = targets, original = sources, merge = merged, mergeTitle = title
        stopped = false
        current?.running = true
        publish()
        // Owned by the job, so dismissing the result view does not cancel sends.
        Task {
            defer {
                if current?.id == job.id { current?.running = false; publish() }
            }
            for target in selected {
                for position in job.items.indices.filter({ job.items[$0].target == target.key }) {
                    guard current?.id == job.id, let item = current?.items[position] else { return }
                    if ["success", "cancelled"].contains(item.status) { continue }
                    if item.status == "unknown" { break }
                    let session = LinLanAppContainer.sessionStore().currentSession()
                    if stopped || session.userId != job.owner || session.resolvedSdkAppId != job.sdkAppID || LinLanImManager.loginUser != job.owner {
                        update(position, status: "cancelled", error: "已停止，未发送")
                        continue
                    }
                    update(position, status: "sending")
                    do {
                        let sent: V2TIMMessage
                        if merge { sent = try await LinLanImManager.forwardMessagesMerged(original, title: mergeTitle, receiver: target.receiver, groupId: target.groupId) }
                        else { sent = try await LinLanImManager.forwardMessage(original[item.index], receiver: target.receiver, groupId: target.groupId) }
                        guard current?.id == job.id else { return }
                        update(position, status: "success", sdkID: sent.msgID)
                    } catch {
                        guard current?.id == job.id else { return }
                        var unknown = true
                        if error is LinLanForwardFailure { unknown = false }
                        if case LinLanImError.apiFailed(let code, _) = error { unknown = [2800, 2801, 2805, 60008].contains(code) }
                        update(position, status: unknown ? "unknown" : "failed", error: error.localizedDescription)
                        break
                    }
                    try? await Task.sleep(nanoseconds: 1_000_000_000)
                }
            }
        }
    }
}

@MainActor
final class LinLanForwardResultsViewController: UITableViewController {
    private var observer: NSObjectProtocol?
    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.leftBarButtonItem = UIBarButtonItem(title: "收起", style: .plain, target: self, action: #selector(close))
        observer = NotificationCenter.default.addObserver(forName: LinLanForwardJobs.changed, object: nil, queue: .main) { [weak self] _ in
            Task { @MainActor in self?.reload() }
        }
        reload()
    }
    deinit { if let observer { NotificationCenter.default.removeObserver(observer) } }
    private func reload() {
        let job = LinLanForwardJobs.shared.current
        title = "已发送 \(job?.items.filter { $0.status == "success" }.count ?? 0)/\(job?.items.count ?? 0)"
        navigationItem.prompt = job?.serverNotice
        if job?.running == true {
            navigationItem.rightBarButtonItem = UIBarButtonItem(
                title: "停止未发", style: .plain, target: self, action: #selector(action)
            )
        } else if job?.resumable == true,
                  job?.items.contains(where: { ["failed", "pending"].contains($0.status) }) == true {
            navigationItem.rightBarButtonItem = UIBarButtonItem(
                title: "补发失败项", style: .plain, target: self, action: #selector(action)
            )
        } else {
            // 全部发送成功（或没有可补发项）时不显示误导性操作。
            navigationItem.rightBarButtonItem = nil
        }
        tableView.reloadData()
    }
    @objc private func close() { dismiss(animated: true) }
    @objc private func action() { if LinLanForwardJobs.shared.current?.running == true { LinLanForwardJobs.shared.cancel() } else { LinLanForwardJobs.shared.retry() } }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { LinLanForwardJobs.shared.current?.items.count ?? 0 }
    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        if let item = LinLanForwardJobs.shared.current?.items[indexPath.row] {
            let label = [
                "pending":"排队中", "uploading":"提交中", "queued":"已交给服务器",
                "preparing":"准备中", "sending":"发送中", "retrying":"重试中",
                "success":"已发送", "failed":"失败", "blocked":"已阻止",
                "unknown":"待确认", "interrupted":"已中断", "cancelled":"已停止",
            ][item.status] ?? "待确认"
            cell.textLabel?.text = "\(item.name) · 第\(item.index + 1)条：\(label)"
            cell.detailTextLabel?.text = item.error
            cell.detailTextLabel?.numberOfLines = 0
        }
        return cell
    }
}
