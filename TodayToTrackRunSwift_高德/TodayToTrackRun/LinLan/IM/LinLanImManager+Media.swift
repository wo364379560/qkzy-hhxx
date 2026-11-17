//
//  LinLanImManager+Media.swift
//  TodayToTrackRun
//

import Foundation
import ImSDK_Plus_Swift
import UIKit

struct LinLanMergerCard {
    let title: String
    let abstracts: [String]
    let layersOverLimit: Bool
}

enum LinLanImageLoader {
    private static let cache = NSCache<NSString, UIImage>()

    static func load(urlString: String?, into imageView: UIImageView, placeholderLetter: String? = nil) {
        imageView.image = nil
        if let letter = placeholderLetter, !letter.isEmpty {
            imageView.backgroundColor = AppTheme.accentSoft
            // 占位字用 tag 简单处理：先清
        }
        guard let raw = urlString?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            if let letter = placeholderLetter?.first {
                imageView.image = letterAvatar(String(letter))
            }
            return
        }
        if raw.hasPrefix("/") || raw.hasPrefix("file://") {
            let path = raw.hasPrefix("file://") ? (URL(string: raw)?.path ?? raw) : raw
            if let img = UIImage(contentsOfFile: path) {
                imageView.image = img
                return
            }
        }
        let key = raw as NSString
        if let cached = cache.object(forKey: key) {
            imageView.image = cached
            return
        }
        guard let url = URL(string: raw) else {
            if let letter = placeholderLetter?.first {
                imageView.image = letterAvatar(String(letter))
            }
            return
        }
        imageView.tag = raw.hashValue
        URLSession.shared.dataTask(with: url) { data, _, _ in
            guard let data, let img = UIImage(data: data) else { return }
            cache.setObject(img, forKey: key)
            DispatchQueue.main.async {
                if imageView.tag == raw.hashValue {
                    imageView.image = img
                }
            }
        }.resume()
    }

    static func letterAvatar(_ letter: String) -> UIImage {
        let size = CGSize(width: 72, height: 72)
        let renderer = UIGraphicsImageRenderer(size: size)
        return renderer.image { _ in
            AppTheme.accent.setFill()
            UIBezierPath(ovalIn: CGRect(origin: .zero, size: size)).fill()
            let attrs: [NSAttributedString.Key: Any] = [
                .font: UIFont.systemFont(ofSize: 28, weight: .bold),
                .foregroundColor: UIColor.white,
            ]
            let text = letter as NSString
            let tSize = text.size(withAttributes: attrs)
            text.draw(
                at: CGPoint(x: (size.width - tSize.width) / 2, y: (size.height - tSize.height) / 2),
                withAttributes: attrs
            )
        }
    }
}

extension LinLanImManager {
    static func imageUrl(of msg: V2TIMMessage, preferLarge: Bool = false) -> String? {
        let images = msg.imageElem?.imageList ?? []
        if preferLarge {
            if let u = images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_LARGE })?.url, !u.isEmpty { return u }
            if let u = images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_ORIGIN })?.url, !u.isEmpty { return u }
        }
        if let u = images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_THUMB })?.url, !u.isEmpty { return u }
        return images.first?.url
    }

    static func parseMergerCard(_ msg: V2TIMMessage) -> LinLanMergerCard? {
        if let elem = msg.mergerElem {
            return LinLanMergerCard(
                title: (elem.title?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? "聊天记录",
                abstracts: elem.abstractList,
                layersOverLimit: elem.layersOverLimit
            )
        }
        // 兼容：部分端合并转发以自定义消息下发
        guard msg.elemType == .V2TIM_ELEM_TYPE_CUSTOM,
              let data = msg.customElem?.data,
              let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return nil }
        let title = (root["title"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let abstracts = root["abstractList"] as? [String]
            ?? (root["abstractList"] as? [[String: Any]])?.compactMap { $0["abstract"] as? String }
            ?? []
        if title.isEmpty && abstracts.isEmpty { return nil }
        return LinLanMergerCard(
            title: title.isEmpty ? "聊天记录" : title,
            abstracts: abstracts,
            layersOverLimit: false
        )
    }

    static func downloadMergerMessage(_ msg: V2TIMMessage) async throws -> [V2TIMMessage] {
        guard let elem = msg.mergerElem else { return [] }
        return try await withCheckedThrowingContinuation { cont in
            elem.downloadMergerMessage(succ: { list in
                cont.resume(returning: list)
            }, fail: { code, desc in
                cont.resume(throwing: LinLanImError.apiFailed(code, desc))
            })
        }
    }

    private static var mediaCacheDir: URL {
        let dir = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("linlan_chat_media", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    private static func fileExists(_ path: String?) -> Bool {
        guard let path, !path.isEmpty else { return false }
        return FileManager.default.fileExists(atPath: path)
    }

    /// 图片：合并转发下载出的子消息有时没有即时 URL，需通过 SDK 下载。
    static func resolveImagePlayable(_ msg: V2TIMMessage) async -> String? {
        guard let elem = msg.imageElem else { return nil }
        if fileExists(elem.path) { return elem.path }
        let images = elem.imageList
        guard let image = images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_ORIGIN })
                ?? images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_LARGE })
                ?? images.first(where: { $0.type == .V2TIM_IMAGE_TYPE_THUMB })
                ?? images.first
        else { return nil }
        let identity = (image.uuid?.isEmpty == false ? image.uuid! : msg.msgID)
            .replacingOccurrences(of: "/", with: "_")
        let out = mediaCacheDir.appendingPathComponent("image_\(identity.isEmpty ? "unknown" : identity).jpg")
        if fileExists(out.path) { return out.path }
        let downloaded: Bool = await withCheckedContinuation { cont in
            image.downloadImage(path: out.path, progress: nil, succ: {
                cont.resume(returning: true)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
        if downloaded, fileExists(out.path) { return out.path }
        if let url = image.url?.trimmingCharacters(in: .whitespacesAndNewlines), !url.isEmpty {
            return url
        }
        return nil
    }

    /// 返回本地路径，或可播放的远程 URL 字符串
    static func resolveVideoPlayable(_ msg: V2TIMMessage) async -> String? {
        guard let elem = msg.videoElem else { return nil }
        if fileExists(elem.videoPath) { return elem.videoPath }
        let uuid = (elem.videoUUID?.isEmpty == false ? elem.videoUUID! : msg.msgID)
        let out = mediaCacheDir.appendingPathComponent("video_\(uuid).mp4")
        if fileExists(out.path) { return out.path }
        let downloaded: Bool = await withCheckedContinuation { cont in
            elem.downloadVideo(path: out.path, progress: nil, succ: {
                cont.resume(returning: true)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
        if downloaded, fileExists(out.path) { return out.path }
        return await withCheckedContinuation { cont in
            elem.getVideoUrl { cont.resume(returning: $0) }
        }
    }

    /// 语音：优先本地 path，否则下载到缓存，再退回远程 URL
    static func resolveSoundPlayable(_ msg: V2TIMMessage) async -> String? {
        guard let elem = msg.soundElem else { return nil }
        if fileExists(elem.path) { return elem.path }
        let uuid = (elem.uuid?.isEmpty == false ? elem.uuid! : msg.msgID)
        let out = mediaCacheDir.appendingPathComponent("sound_\(uuid).m4a")
        if fileExists(out.path) { return out.path }
        let downloaded: Bool = await withCheckedContinuation { cont in
            elem.downloadSound(path: out.path, progress: nil, succ: {
                cont.resume(returning: true)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
        if downloaded, fileExists(out.path) { return out.path }
        return await withCheckedContinuation { cont in
            elem.getUrl { cont.resume(returning: $0) }
        }
    }

    /// 返回本地路径，或远程 URL（下载失败时）
    static func resolveFileLocal(_ msg: V2TIMMessage) async -> (pathOrUrl: String, fileName: String)? {
        guard let elem = msg.fileElem else { return nil }
        let fileName = (elem.filename?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap {
            $0.isEmpty ? nil : $0
        } ?? "文件"
        if fileExists(elem.path) { return (elem.path!, fileName) }
        let safe = fileName.replacingOccurrences(of: "/", with: "_")
        let uuid = (elem.uuid?.isEmpty == false ? elem.uuid! : msg.msgID)
        let out = mediaCacheDir.appendingPathComponent("file_\(uuid)_\(safe)")
        if fileExists(out.path) { return (out.path, fileName) }
        let downloaded: Bool = await withCheckedContinuation { cont in
            elem.downloadFile(path: out.path, progress: nil, succ: {
                cont.resume(returning: true)
            }, fail: { _, _ in
                cont.resume(returning: false)
            })
        }
        if downloaded, fileExists(out.path) { return (out.path, fileName) }
        let url: String? = await withCheckedContinuation { cont in
            elem.getUrl { cont.resume(returning: $0) }
        }
        guard let url, !url.isEmpty else { return nil }
        return (url, fileName)
    }

    static func loadImageData(urlString: String) async -> Data? {
        let raw = urlString.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !raw.isEmpty else { return nil }
        if raw.hasPrefix("/") || raw.hasPrefix("file://") {
            let path = raw.hasPrefix("file://") ? (URL(string: raw)?.path ?? raw) : raw
            return try? Data(contentsOf: URL(fileURLWithPath: path))
        }
        guard let url = URL(string: raw) else { return nil }
        return try? await URLSession.shared.data(from: url).0
    }

    static func mergeForwardTitle(
        messages: [V2TIMMessage],
        isGroup: Bool,
        conversationTitle: String
    ) -> String {
        if isGroup {
            let name = conversationTitle.trimmingCharacters(in: .whitespacesAndNewlines)
            return "\((name.isEmpty ? "群聊" : name))的聊天记录"
        }
        let names = Array(
            Set(messages.map { LinLanQuoteHelper.senderName($0) }.filter { !$0.isEmpty })
        ).prefix(2)
        switch names.count {
        case 0: return "聊天记录"
        case 1: return "\(names[0])的聊天记录"
        default: return "\(names[0])和\(names[1])的聊天记录"
        }
    }

    static func clockTime(_ date: Date?) -> String {
        guard let date else { return "" }
        let fmt = DateFormatter()
        fmt.dateFormat = "HH:mm"
        return fmt.string(from: date)
    }

}
