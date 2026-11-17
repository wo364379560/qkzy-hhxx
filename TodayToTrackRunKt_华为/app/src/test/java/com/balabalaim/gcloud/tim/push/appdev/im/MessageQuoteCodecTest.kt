package com.balabalaim.gcloud.tim.push.appdev.im

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class MessageQuoteCodecTest {
    @Test fun readsWebReply() {
        val raw = """{"messageReply":{"messageID":"app-original","messageSender":"测试用户","messageAbstract":"App 原消息","messageType":1,"version":1}}"""
        assertEquals(MessageQuote("app-original", "测试用户: App 原消息"), MessageQuoteCodec.decode(raw))
    }
    @Test fun supportsLegacyNativeAndMalformedWebFallback() {
        val raw = """{"messageReply":{"messageAbstract":"缺少 ID"},"atomicx_quote_v1":{"msgID":"ios-original","msgSender":"iOS","msgAbstract":"原消息"}}"""
        assertEquals(MessageQuote("ios-original", "iOS: 原消息"), MessageQuoteCodec.decode(raw))
    }
    @Test fun rejectsBrokenMetadata() {
        listOf(null, "", "not-json", "[]", "{}", """{"messageReply":{"messageID":null}}""").forEach {
            assertNull(MessageQuoteCodec.decode(it))
        }
    }
    @Test fun suppliesMediaSummaryWithoutNullSender() {
        val raw = """{"messageReply":{"messageID":"picture","messageSender":null,"messageType":3}}"""
        assertEquals(MessageQuote("picture", "[图片]"), MessageQuoteCodec.decode(raw))
    }
    @Test fun writesBothProtocolsAndPreservesText() {
        val raw = MessageQuoteCodec.encode("original", "群名片", "引号\"和换行\n内容 😀", 1, 123, 456)
        val root = JSONObject(raw)
        val web = root.getJSONObject("messageReply")
        assertEquals("original", web.getString("messageID"))
        assertEquals(123L, web.getLong("messageSequence"))
        assertEquals(456L, web.getLong("messageTime"))
        val native = JSONObject().put("atomicx_quote_v1", root.getJSONObject("atomicx_quote_v1"))
        assertEquals(MessageQuoteCodec.decode(raw), MessageQuoteCodec.decode(native.toString()))
        assertEquals("群名片: 引号\"和换行\n内容 😀", MessageQuoteCodec.decode(raw)?.displayLine)
    }
}
