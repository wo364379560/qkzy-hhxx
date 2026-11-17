package com.balabalaim.gcloud.tim.push.appdev.im

import androidx.activity.compose.setContent
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.test.core.app.ActivityScenario
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.MainActivity
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanDomainPoolManager
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanHTTPDNS
import com.balabalaim.gcloud.tim.push.appdev.data.network.ApiException
import com.balabalaim.gcloud.tim.push.appdev.data.network.LinLanHttpTransport
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanChatScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanTheme
import com.tencent.imsdk.v2.V2TIMMessage
import java.io.File
import java.net.URLEncoder
import java.util.concurrent.TimeUnit
import android.os.Bundle
import kotlinx.coroutines.*
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.MediaType.Companion.toMediaType
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Assume.assumeTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/** Only explicitly supplied QA identities and a newly created isolated group are used. */
@RunWith(AndroidJUnit4::class)
class SilentRecallLiveTest {
    @get:Rule val compose = createEmptyComposeRule()
    private val context get() = InstrumentationRegistry.getInstrumentation().targetContext
    private val directory get() = context.getExternalFilesDir("rui-recall-qa")!!
    private val args get() = InstrumentationRegistry.getArguments()
    private fun credentials() = JSONObject(File(directory, "credentials-private.json").readText())
    private fun session(data: JSONObject) = AuthSession(sdkAppId = data.getInt("sdk_app_id"), userId = data.getString("user_id"),
        userSig = data.getString("user_sig"), userSigExpiresAt = data.optString("user_sig_expires_at"),
        token = data.getString("token"), tokenExpiresAt = data.optString("token_expires_at"), refreshToken = data.optString("refresh_token"))
    private suspend fun login(data: JSONObject) {
        val value = session(data)
        LinLanRuntimeConfig.applySdkAppId(value.sdkAppId)
        LinLanImManager.init(context, value.sdkAppId)
        LinLanImManager.logout()
        LinLanAppContainer.sessionStore().saveSession(value)
        LinLanImManager.login(value.userId, value.userSig)
    }
    private fun phase(value: String) = InstrumentationRegistry.getInstrumentation().sendStatus(2, Bundle().apply { putString("phase", value) })
    private suspend fun request(data: JSONObject, path: String, body: JSONObject): Pair<Int, JSONObject> = withContext(Dispatchers.IO) {
        val client = LinLanHttpTransport.configure(OkHttpClient.Builder())
            .dns(LinLanHTTPDNS.dns).callTimeout(25, TimeUnit.SECONDS)
            .connectTimeout(8, TimeUnit.SECONDS).readTimeout(12, TimeUnit.SECONDS).writeTimeout(12, TimeUnit.SECONDS).build()
        try {
            val request = Request.Builder().url(credentials().getString("origin") + "/api/v1/chat" + path)
                .header("Authorization", "Bearer " + data.getString("token"))
                .post(body.toString().toRequestBody("application/json".toMediaType())).build()
            client.newCall(request).execute().use { it.code to JSONObject(it.body!!.string()) }
        } finally { client.connectionPool.evictAll(); client.dispatcher.executorService.shutdown() }
    }
    private fun receipt(report: JSONObject) = File(directory, "result.json").writeText(report.toString(2))
    private suspend fun message(group: String, id: String, seq: Long = 0): V2TIMMessage = withTimeout(25_000) {
        // Account switches must wait for SDK cloud synchronization. Group sequence is stable across identities.
        repeat(20) { attempt ->
            val history = LinLanImManager.fetchHistory("group_$group", 50)
            val found = history.firstOrNull { it.msgID == id || (seq > 0 && it.seq == seq) }
                ?: LinLanImManager.findMessages(listOf(id)).firstOrNull()
            if (found != null) return@withTimeout found
            phase("wait_for_sdk_cloud_sync_attempt_${attempt + 1}_count_${history.size}_seq_valid_${seq > 0}")
            delay(500)
        }
        error("QA message missing after bounded SDK cloud synchronization")
    }

    @Test(timeout = 180_000) fun moderatorRecallAndColdHistory() = runBlocking {
        assumeTrue(args.getString("ruiRecallLive") == "true")
        val all = credentials(); val owner = all.getJSONObject("owner"); val member = all.getJSONObject("member"); val admin = all.getJSONObject("admin")
        val fixtureFile = File(directory, "fixture-private.json")
        val saved = LinLanAppContainer.sessionStore().currentSession()
        val generation = LinLanAppContainer.sessionStore().sessionGeneration()
        val report = JSONObject().put("stage", args.getString("ruiRecallStage", "live"))
        var group = if (fixtureFile.exists()) JSONObject(fixtureFile.readText()).optString("group") else ""
        var keepForColdStart = false
        // Exercise the production REST client against the same bundled, approved ESA origin as QA setup.
        val originField = LinLanDomainPoolManager::class.java.getDeclaredField("activeOrigin").apply { isAccessible = true }
        val savedOrigin = originField.get(LinLanDomainPoolManager)
        originField.set(LinLanDomainPoolManager, all.getString("origin"))
        try {
            if (args.getString("ruiRecallStage") == "cleanup") return@runBlocking
            if (args.getString("ruiRecallStage") == "cold") {
                assertTrue(fixtureFile.exists())
                val fixture = JSONObject(fixtureFile.readText())
                phase("cold_member_login_history")
                login(member)
                val history = LinLanImManager.fetchHistory("group_$group", 50)
                val state = LinLanImManager::class.java.getDeclaredField("recalled").apply { isAccessible = true }.get(LinLanImManager) as RecalledMessageState
                val scope = state.current()!!
                for (key in listOf("first", "live")) assertTrue("Recall evidence must survive a new process", state.isRevoked(scope, fixture.getString(key), false))
                for (key in listOf("first", "live", "admin")) {
                    val id = fixture.getString(key)
                    val found = history.firstOrNull { it.msgID == id } ?: LinLanImManager.findMessages(listOf(id)).firstOrNull()
                    if (found != null) {
                        assertTrue(LinLanImManager.shouldSilentlyHideRevokedMessage(found))
                        assertEquals("", LinLanImManager.messagePreview(found))
                    }
                }
                report.put("cold_process_and_history_silent", true)
                phase("verify_member_permission_denial")
                val control = message(group, fixture.getString("control"), fixture.getLong("control_seq"))
                val denied = runCatching { LinLanImManager.revokeMessage(control) }
                assertTrue("An ordinary member cannot revoke another member", denied.isFailure)
                val rejection = denied.exceptionOrNull()
                assertTrue("The real API must reject management permission", rejection is ApiException && rejection.status == 400 && rejection.message.orEmpty().contains("群管理权限"))
                assertFalse(LinLanImManager.isRevokedMessage(control))
                report.put("rejected_recall_keeps_original", true)
                phase("verify_own_recall_hint")
                val own = LinLanImManager.sendTextMessage("本人撤回保留提示 QA", null, group)
                LinLanImManager.revokeMessage(own)
                assertTrue(LinLanImManager.isRevokedMessage(own))
                assertFalse(LinLanImManager.shouldSilentlyHideRevokedMessage(own))
                assertEquals("你撤回了一条消息", LinLanImManager.messagePreview(own))
                report.put("confirmed_own_recall_keeps_hint", true)
                return@runBlocking
            }

            phase("owner_login")
            login(owner)
            phase("create_isolated_group")
            group = LinLanImManager.createCommunityGroup("无痕撤回QA-${System.currentTimeMillis().toString().takeLast(6)}",
                listOf(member.getString("user_id"), admin.getString("user_id")))
            val fixture = JSONObject().put("group", group)
            fixtureFile.writeText(fixture.toString())
            phase("verify_three_members")
            assertEquals(3, LinLanImManager.fetchAllGroupMembers(group).size)
            phase("authorize_qa_admin_http")
            val addAdmin = request(owner, "/groups/${URLEncoder.encode(group, "UTF-8")}/admins", JSONObject().put("user_id", admin.getString("user_id")))
            assertEquals("QA admin setup must be authorized", 0, addAdmin.second.optInt("code", -1))
            phase("send_control")
            val control = LinLanImManager.sendTextMessage("权限拒绝对照 QA", null, group)
            fixture.put("control", control.msgID).put("control_seq", control.seq)

            phase("member_message_then_owner_recall")
            login(member)
            val first = LinLanImManager.sendTextMessage("群主无痕撤回 QA-A", null, group)
            fixture.put("first", first.msgID); fixtureFile.writeText(fixture.toString())
            login(owner)
            val target = message(group, first.msgID, first.seq)
            assertFalse(target.isSelf)
            phase("owner_production_recall_begin")
            LinLanImManager.revokeMessage(target)
            phase("owner_production_recall_confirmed")
            assertTrue(LinLanImManager.shouldSilentlyHideRevokedMessage(target))
            assertFalse(LinLanImManager.canForwardMessage(target))
            report.put("owner_rest_route_and_stale_cache_silent", true)

            phase("member_live_ui")
            login(member)
            val live = LinLanImManager.sendTextMessage("成员端实时消失 QA-B", null, group)
            fixture.put("live", live.msgID); fixtureFile.writeText(fixture.toString())
            var activity: ActivityScenario<MainActivity>? = null
            try {
                activity = ActivityScenario.launch(MainActivity::class.java)
                delay(6_000)
                activity.onActivity { targetActivity -> targetActivity.setContent { LinLanTheme {
                    LinLanChatScreen(onOpenRealname = {}, conversationId = "group_$group", title = "无痕撤回QA", isGroup = true, onBack = {})
                } } }
                compose.waitUntil(20_000) { compose.onAllNodesWithText("成员端实时消失 QA-B").fetchSemanticsNodes(atLeastOneRootRequired = false).isNotEmpty() }
                phase("member_message_visible")
                phase("owner_recall_http_member_online")
                val response = request(owner, "/messages/group/revoke", JSONObject().put("group_id", group).put("msg_seq", live.seq))
                assertEquals(0, response.second.optInt("code", -1))
                compose.waitUntil(15_000) { compose.onAllNodesWithText("成员端实时消失 QA-B").fetchSemanticsNodes().isEmpty() }
                compose.onNodeWithText("你撤回了一条消息").assertDoesNotExist()
                compose.onNodeWithText("对方撤回了一条消息").assertDoesNotExist()
                report.put("member_live_UI_silent_including_own_incoming_moderation", true)
            } finally { activity?.close() }

            phase("admin_recall_then_member_history")
            // The third QA identity is a real group admin, independently authorized by the existing API.
            login(member)
            val adminTarget = LinLanImManager.sendTextMessage("管理员无痕撤回 QA-C", null, group)
            fixture.put("admin", adminTarget.msgID); fixtureFile.writeText(fixture.toString())
            login(admin)
            LinLanImManager.revokeMessage(message(group, adminTarget.msgID, adminTarget.seq))
            report.put("admin_rest_route_silent", true)
            login(member)
            val historical = message(group, adminTarget.msgID)
            assertTrue(LinLanImManager.shouldSilentlyHideRevokedMessage(historical))
            report.put("member_relogin_history_silent", true)
            context.getSharedPreferences("rui_recalled_messages_v1", 0).edit().putBoolean("qa_flush", true).commit()
            keepForColdStart = true
        } finally {
            if (!keepForColdStart && group.isNotBlank()) {
                phase("dismiss_isolated_group")
                login(owner); LinLanImManager.dismissGroup(group)
                report.put("isolated_test_group_dismissed", true)
                fixtureFile.delete()
            }
            val store = LinLanAppContainer.sessionStore()
            val current = store.currentSession()
            if (store.sessionGeneration() == generation && current.userId in listOf(owner.getString("user_id"), member.getString("user_id"), admin.getString("user_id"))) {
                store.replaceSessionIfCurrent(current, saved)
            }
            LinLanImManager.logout()
            originField.set(LinLanDomainPoolManager, savedOrigin)
            receipt(report)
        }
    }
}
