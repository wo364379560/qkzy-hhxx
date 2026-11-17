package com.balabalaim.gcloud.tim.push.appdev.im

import com.tencent.imsdk.v2.V2TIMMessage

data class MessageQuote(
    val quotedMsgId: String,
    val displayLine: String,
)

/** Reads both Web TUIKit and native quote formats. */
object LinLanQuoteHelper {
    fun resolveQuote(message: V2TIMMessage): MessageQuote? = MessageQuoteCodec.decode(message.cloudCustomData)

    fun embedQuoteCloudCustomData(quoted: V2TIMMessage, senderName: String): String {
        return MessageQuoteCodec.encode(quoted.msgID.orEmpty(), senderName,
            LinLanImManager.messagePreview(quoted), quoted.elemType, quoted.seq, quoted.timestamp)
    }

    fun senderName(message: V2TIMMessage): String {
        message.nameCard?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        message.friendRemark?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        message.nickName?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        return message.sender ?: message.userID ?: "未知"
    }

}
