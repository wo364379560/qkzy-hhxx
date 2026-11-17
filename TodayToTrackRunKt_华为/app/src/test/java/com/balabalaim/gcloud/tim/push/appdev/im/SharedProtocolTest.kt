package com.balabalaim.gcloud.tim.push.appdev.im

import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingPayload
import com.balabalaim.gcloud.tim.push.appdev.rtc.LinLanRtcSignalingAction
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class SharedProtocolTest {
    @Test fun androidAndWebUseTheSameStringRoomFixture() {
        val raw = javaClass.classLoader!!.getResourceAsStream("rui_call_v2.json")!!.bufferedReader().readText()
        val payload = LinLanRtcSignalingPayload.parse(raw)!!
        assertEquals("string", payload.roomType); assertEquals("rui_fixed_interop_room", payload.roomId)
        val encoded = JSONObject(payload.toJson())
        assertEquals("rtc_call_v1", encoded.getString("businessID")); assertEquals(2, encoded.getInt("version")); assertEquals("string", encoded.getString("roomType"))
        assertEquals(LinLanRtcSignalingAction.BUSY, LinLanRtcSignalingPayload.parse(payload.copy(action = LinLanRtcSignalingAction.BUSY).toJson())!!.action)
    }
    @Test fun legacyRoomsStayTypedAndUnknownInvitesAreRejected() {
        val oldWeb = """{"longdo_type":"longdo_rtc_call_invite_v1","longdo_version":1,"longdo_room_id":12345,"longdo_media_type":"longdo_video"}"""
        assertEquals("number", LinLanRtcSignalingPayload.parse(oldWeb)!!.roomType)
        val oldApp = """{"businessID":"rtc_call_v1","action":"invite","version":1,"roomId":"call_legacy","mediaType":1}"""
        assertEquals("string", LinLanRtcSignalingPayload.parse(oldApp)!!.roomType)
        assertNull(LinLanRtcSignalingPayload.parse(oldApp.replace("rtc_call_v1", "unrelated_business")))
        assertNull(LinLanRtcSignalingPayload.parse(oldApp.replace("\"version\":1", "\"version\":99")))
        assertEquals("call_legacy", LinLanRtcSignalingPayload.parse(JSONObject().put("businessID", 1).put("data", oldApp).toString())!!.roomId)
    }
    @Test fun ruiLegacyAndNewCardsKeepExactGroupIDs() {
        val groupID = "@TGS#_@TGS#real_id"
        val group = RuiSharedCard.parse("""{"businessID":"group_card_v1","groupId":"$groupID","groupName":"Test"}""")!!
        assertEquals(groupID, group.id); assertTrue(group.isGroup)
        assertEquals(group, RuiSharedCard.parse(group.encode()))
        assertNotNull(RuiSharedCard.parse("""{"businessID":"contact_card_v1","userId":"alice","nickname":"Alice"}"""))
        assertNotNull(RuiSharedCard.parse("""{"businessID":"zhihuan_contact_card","version":1,"userId":"alice","nickname":"Alice"}"""))
        assertNull(RuiSharedCard.parse("""{"businessID":"group_card_v1","groupId":"","groupName":"bad"}"""))
        val image = RuiSharedCard.parse("""{"businessID":"contact_card_v1","userId":"alice","avatar":"javascript:alert(1)"}""")!!
        assertEquals("", image.avatar)
    }
    @Test fun genericPermissionErrorDoesNotSuggestChangingInviteSetting() {
        val error = LinLanSdkOperationError("邀请入群", 10007)
        assertTrue(error.message!!.contains("10007")); assertTrue(error.message!!.contains("权限")); assertFalse(error.message!!.contains("开关"))
    }
}
