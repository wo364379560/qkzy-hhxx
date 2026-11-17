//
//  LinLanAppConstants.swift
//  TodayToTrackRun
//
//  对照 lingdianLantingKotlin / chi_chuanluxinTx 协议清单；隐私页域名：ys.ruizhihexx.xyz
//

import Foundation

enum LinLanAppConstants {
    static let sdkAppId: Int = 1_600_163_137
    static let appName = "睿之禾"
    static let apiPrefix = "/api/v1/chat"
    static let probePath = "/ping"
    static let bootstrapPath = "/api/v1/public/bootstrap/domain-pool"
    static let fallbackOrigin = "https://api.prfgsu.com"

    /// 腾讯云 IM 控制台 APNs 证书 ID：Debug 使用开发证书，Release 使用生产证书。
    /// 走 SDK 自带 `setAPNS`，不需要额外接入 TIMPush。
    #if DEBUG
    static let apnsBusinessID: Int = 49_795
    #else
    static let apnsBusinessID: Int = 49_794
    #endif

    private static let legalHost = "https://ys.ruizhihexx.xyz"

    static let userAgreementURL = "\(legalHost)/user_agreement.html"
    static let privacyPolicyURL = "\(legalHost)/privacy_protocol.html"
    static let personalInfoProtocolURL = "\(legalHost)/personal_info_protocol.html"
    static let statementProtocolURL = "\(legalHost)/statement_protocol.html"
    static let minorProtectionURL = "\(legalHost)/weichengnian.html"
    static let thirdInfoShareURL = "\(legalHost)/third_info_share_list.html"
    /// 第三方 SDK 说明（若站点尚未上传该页，需补 third_party_sdk.html）
    static let thirdPartySDKURL = "\(legalHost)/third_party_sdk.html"
    static let supportURL = "https://shchuanlu.com/support.html"

    static let icpFilingNumber = "沪ICP备2026029882号-2A"
    static let icpFilingURL = "https://beian.miit.gov.cn/"
}
