//
//  AMapBootstrap.swift
//  TodayToTrackRun
//

import AMapFoundationKit
import AMapLocationKit
import AMapSearchKit
import Foundation
import MAMapKit

enum AMapBootstrap {
    /// 高德 iOS Key（高德版专用）。
    static let apiKey = "d8284dfb4b431e234359831e7275a1da"
    private static var didConfigure = false

    /// App 启动或进入地图前调用；`MAMapView` 必须在此之后再创建
    static func configureIfNeeded() {
        guard !didConfigure else { return }
        didConfigure = true
        MAMapView.updatePrivacyShow(.didShow, privacyInfo: .didContain)
        MAMapView.updatePrivacyAgree(.didAgree)
        AMapLocationManager.updatePrivacyShow(.didShow, privacyInfo: .didContain)
        AMapLocationManager.updatePrivacyAgree(.didAgree)
        AMapSearchAPI.updatePrivacyShow(.didShow, privacyInfo: .didContain)
        AMapSearchAPI.updatePrivacyAgree(.didAgree)

        AMapServices.shared().apiKey = apiKey
        AMapServices.shared().enableHTTPS = true
    }

    @available(*, deprecated, renamed: "configureIfNeeded")
    static func configure() {
        configureIfNeeded()
    }
}
