//
//  LinLanZLPhotoPickerHelper.swift
//  TodayToTrackRun
//
//  聊天相册：ZLPhotoBrowser（项目已 Pod）。比系统 PHPicker 首次弹出更稳，避免键盘导致上移。
//

import AVFoundation
import Photos
import UIKit
import ZLPhotoBrowser

enum LinLanZLPhotoPickerHelper {
    private static func applyChineseUI() {
        ZLPhotoUIConfiguration.default().languageType = .chineseSimplified
    }

    /// 选图片（最多 9 张）
    @discardableResult
    static func presentImages(
        from vc: UIViewController,
        maxCount: Int = 9,
        onPick: @escaping ([ZLResultModel]) -> Void
    ) -> ZLPhotoPicker {
        applyChineseUI()
        let config = ZLPhotoConfiguration.default()
        config.maxSelectCount = max(1, maxCount)
        config.showSelectBtnWhenSingleSelect = true
        config.showSelectedIndex = true
        config.showSelectCountOnDoneBtn = true
        config.allowSelectImage = true
        config.allowSelectVideo = false
        config.allowMixSelect = false
        config.allowEditImage = false
        config.allowSelectGif = true
        config.allowSelectLivePhoto = false

        let picker = ZLPhotoPicker()
        picker.selectImageBlock = { results, _ in onPick(results) }
        picker.showPhotoLibrary(sender: vc)
        return picker
    }

    /// 选单个视频
    @discardableResult
    static func presentVideo(
        from vc: UIViewController,
        onPick: @escaping (ZLResultModel) -> Void
    ) -> ZLPhotoPicker {
        applyChineseUI()
        let config = ZLPhotoConfiguration.default()
        config.maxSelectCount = 1
        config.allowSelectImage = false
        config.allowSelectVideo = true
        config.allowMixSelect = false
        config.allowEditImage = false
        config.maxSelectVideoDuration = 3600
        config.maxSelectVideoDataSize = 300 * 1024 // KB

        let picker = ZLPhotoPicker()
        picker.selectImageBlock = { results, _ in
            if let first = results.first { onPick(first) }
        }
        picker.showPhotoLibrary(sender: vc)
        return picker
    }

    static func saveImage(_ image: UIImage, quality: CGFloat = 0.85) -> String? {
        guard let data = image.jpegData(compressionQuality: quality) else { return nil }
        let path = FileManager.default.temporaryDirectory
            .appendingPathComponent("chat_img_\(UUID().uuidString).jpg")
        do {
            try data.write(to: path)
            return path.path
        } catch {
            return nil
        }
    }

    static func exportVideoPath(from asset: PHAsset) async -> (path: String, duration: Int)? {
        await withCheckedContinuation { cont in
            let opts = PHVideoRequestOptions()
            opts.isNetworkAccessAllowed = true
            opts.deliveryMode = .highQualityFormat
            PHImageManager.default().requestAVAsset(forVideo: asset, options: opts) { avAsset, _, _ in
                guard let urlAsset = avAsset as? AVURLAsset else {
                    cont.resume(returning: nil)
                    return
                }
                let src = urlAsset.url
                let ext = src.pathExtension.isEmpty ? "mp4" : src.pathExtension
                let dest = FileManager.default.temporaryDirectory
                    .appendingPathComponent("chat_vid_\(UUID().uuidString).\(ext)")
                do {
                    try? FileManager.default.removeItem(at: dest)
                    try FileManager.default.copyItem(at: src, to: dest)
                    let sec = Int(round(CMTimeGetSeconds(urlAsset.duration)))
                    cont.resume(returning: (dest.path, max(1, sec)))
                } catch {
                    cont.resume(returning: (src.path, max(1, Int(round(CMTimeGetSeconds(urlAsset.duration))))))
                }
            }
        }
    }
}
