import Foundation
import CryptoKit

struct LinLanAttachmentCard {
    let fileID: String
    let name: String
    let size: Int64
    let isVideo: Bool
    let contentType: String
    static func parse(_ text: String?) -> LinLanAttachmentCard? {
        guard let line = text?.components(separatedBy: .newlines).first(where: { $0.hasPrefix("CN_ATTACHMENT_V1:") }),
              let data = String(line.dropFirst("CN_ATTACHMENT_V1:".count)).data(using: .utf8),
              let body = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let id = body["file_id"] as? String, !id.isEmpty, let name = body["name"] as? String else { return nil }
        return LinLanAttachmentCard(fileID: id, name: name, size: (body["size"] as? NSNumber)?.int64Value ?? 0,
            isVideo: body["is_video"] as? Bool ?? false, contentType: body["content_type"] as? String ?? "application/octet-stream")
    }
}

enum LinLanMultipartUploader {
    static func upload(path: String, fileName: String, conversation: String, contentType: String, isVideo: Bool,
        active: @escaping () -> Bool,
        request: @escaping (String, String, [String: Any]) async throws -> [String: Any]) async throws -> [String: Any] {
        func requireActive() throws {
            try Task.checkCancellation()
            if !active() { throw LinLanAPIError.message("账号已切换，上传已暂停") }
        }
        let handle = try FileHandle(forReadingFrom: URL(fileURLWithPath: path))
        defer { try? handle.close() }
        let size = try handle.seekToEnd()
        try handle.seek(toOffset: 0)
        var hash = SHA256()
        while let bytes = try handle.read(upToCount: 1024 * 1024), !bytes.isEmpty {
            try requireActive()
            hash.update(data: bytes)
            await Task.yield()
        }
        let fingerprint = hash.finalize().map { String(format: "%02x", $0) }.joined()
        let identity = Data("\(conversation):\(fileName):\(size):\(contentType):\(isVideo):\(fingerprint)".utf8)
        let id = SHA256.hash(data: identity).map { String(format: "%02x", $0) }.joined()
        try requireActive()
        var state = try await request("POST", "/upload/attachment/multipart/init", ["request_id": id, "content_hash": fingerprint,
            "conversation_id": conversation, "file_name": fileName, "file_size": size, "content_type": contentType, "is_video": isVideo])
        guard state["content_hash"] as? String == fingerprint, (state["file_size"] as? NSNumber)?.uint64Value == size,
              let fileID = state["file_id"] as? String, let partSize = state["part_size"] as? Int, partSize == 8 * 1024 * 1024 else {
            throw LinLanAPIError.message("上传任务与原文件不一致")
        }
        let base = "/upload/attachment/multipart/\(fileID)"
        if state["state"] as? String == "uploading" {
            let confirmed = Dictionary(uniqueKeysWithValues: ((state["parts"] as? [[String: Any]]) ?? []).compactMap { row -> (Int, Int)? in
                guard let part = row["part_number"] as? Int, let length = row["size"] as? Int else { return nil }; return (part, length)
            })
            var offset: UInt64 = 0
            var part = 1
            while offset < size {
                try requireActive()
                let count = Int(min(UInt64(partSize), size - offset))
                if confirmed[part] != count {
                    try handle.seek(toOffset: offset)
                    guard let bytes = try handle.read(upToCount: count), bytes.count == count else { throw LinLanAPIError.message("本地文件已变化") }
                    for attempt in 0...2 {
                        do {
                            let signature = try await request("POST", "\(base)/parts/\(part)", [:])
                            try requireActive()
                            guard let signed = signature["put_url"] as? String, let url = URL(string: signed) else { throw LinLanAPIError.message("分片授权无效") }
                            var put = URLRequest(url: url, timeoutInterval: 180)
                            put.httpMethod = "PUT"
                            let (_, response) = try await URLSession.shared.upload(for: put, from: bytes)
                            guard let status = (response as? HTTPURLResponse)?.statusCode, (200..<300).contains(status) else { throw LinLanAPIError.message("分片上传被存储服务拒绝") }
                            break
                        } catch {
                            try requireActive()
                            let progress = try await request("GET", base, [:])
                            if (progress["parts"] as? [[String: Any]])?.contains(where: { $0["part_number"] as? Int == part && $0["size"] as? Int == count }) == true { break }
                            if attempt == 2 { throw error }
                            try await Task.sleep(nanoseconds: UInt64(500_000_000 * (1 << attempt)))
                        }
                    }
                }
                offset += UInt64(count)
                part += 1
            }
            try requireActive()
            state = try await request("POST", "\(base)/complete", [:])
        }
        let deadline = Date().addingTimeInterval(600)
        while state["state"] as? String != "uploaded" {
            try requireActive()
            if ["aborted", "failed", "cleaned"].contains(state["state"] as? String ?? "") { throw LinLanAPIError.message("文件合并或内容校验失败") }
            if Date() > deadline { throw LinLanAPIError.message("文件仍在服务端确认，重新选择同一文件可继续查询，请勿重复发送") }
            try await Task.sleep(nanoseconds: 1_500_000_000)
            state = try await request("GET", base, [:])
        }
        try requireActive()
        return state
    }
}
