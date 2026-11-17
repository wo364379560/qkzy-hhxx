package com.balabalaim.gcloud.tim.push.appdev.push

import com.balabalaim.gcloud.tim.push.appdev.core.LinLanAppConstants
import com.tencent.imsdk.v2.V2TIMOfflinePushInfo
import org.json.JSONObject

/** Rui WEB's existing channel/template contract; verify the same IDs in its console.
 * Vendor parameter names: https://cloud.tencent.com/document/product/269/121188
 */
object LinLanOfflinePushInfo {
    fun message(sender: String, groupId: String?, summary: String): V2TIMOfflinePushInfo {
        val desc = summary.take(128).ifBlank { "发来一条消息" }
        return V2TIMOfflinePushInfo().apply {
            title = LinLanAppConstants.APP_NAME
            this.desc = desc
            disablePush(false)
            setAndroidXiaoMiChannelID("157799")
            setAndroidOPPOChannelID("push_oplus_category_service")
            setAndroidOPPOCategory("IM")
            setIOSSound(V2TIMOfflinePushInfo.IOS_OFFLINE_PUSH_DEFAULT_SOUND)
            ext = JSONObject().put("entity", JSONObject().put("action", 1).put("version", 1)
                .put("chatType", if (groupId.isNullOrBlank()) 1 else 2).put("sender", groupId ?: sender)
                .put("groupID", groupId ?: "").put("nickName", sender)).toString().toByteArray()
            vendorParams = JSONObject().put("xiaomiTemplateId", "M12762")
                .put("xiaomiTemplateParam", JSONObject().put("keywords1", sender.take(128)).put("keywords2", desc).toString())
                .put("oppoTemplateId", "6a8408344fae550145d04b8f")
                .put("oppoTitleParam", JSONObject().put("senderName", sender).toString())
                .put("oppoContentParam", JSONObject().put("text", desc).toString()).toString()
        }
    }
}
