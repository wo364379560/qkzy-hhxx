package com.balabalaim.gcloud.tim.push.appdev.im

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.core.app.ActivityScenario
import androidx.activity.compose.setContent
import com.balabalaim.gcloud.tim.push.appdev.ui.main.LinLanChatScreen
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanTheme
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import com.balabalaim.gcloud.tim.push.appdev.MainActivity
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.data.model.AuthSession
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanNotifyRouter
import androidx.test.platform.app.InstrumentationRegistry
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.tencent.imsdk.v2.V2TIMManager
import java.io.File
import kotlinx.coroutines.*
import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.Rule
import org.junit.runner.RunWith

/** Opt-in real SDK reads. Credentials are supplied privately at run time, never in source/arguments. */
@RunWith(AndroidJUnit4::class)
class GroupPermissionLiveTest {
    @get:Rule val compose = createEmptyComposeRule()
    @Test(timeout = 120_000) fun realMembershipAndInvalidationRecovery() = runBlocking {
        assumeTrue(InstrumentationRegistry.getArguments().getString("ruiGroupLive") == "true")
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val directory = context.getExternalFilesDir("rui-group-qa")!!
        val data = JSONObject(File(directory, "session-private.json").readText()).getJSONObject("data")
        val user = data.getString("user_id")
        val app = data.getInt("sdk_app_id")
        LinLanRuntimeConfig.applySdkAppId(app)
        LinLanImManager.init(context, app)
        LinLanImManager.login(user, data.getString("user_sig"))
        val rows = JSONArray()
        val memo = LinLanImManager::class.java.getDeclaredField("permissionMemo").apply { isAccessible = true }
            .get(LinLanImManager) as PermissionMemo<LinLanImManager.GroupSendPermission>
        try {
            val direct = runCatching { memo.get("$app:$user:isolated-direct") { LinLanImManager.GroupSendPermission.Allowed } }
            val directError = direct.exceptionOrNull()
            InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply {
                putString("rui_group_direct", "succeeded=${direct.isSuccess} exception=${directError?.javaClass?.simpleName} message=${directError?.message}")
            })
            val groups = LinLanImManager.fetchJoinedGroups()
            var candidate: String? = null
            for (group in groups.take(20)) {
                val id = group.groupID ?: continue
                val permission = LinLanImManager.resolveGroupSendPermission(id)
                rows.put(JSONObject().put("group_hash", id.hashCode().toUInt().toString(16))
                    .put("role", group.role).put("all_muted", group.isAllMuted)
                    .put("allows_send", permission.allowsSend).put("hint", permission.blockedHint))
                if (permission.allowsSend && candidate == null) candidate = id
            }
            val report = JSONObject().put("joined_groups", groups.size).put("read_results", rows)
            File(directory, "result.json").writeText(report.toString(2))
            assertNotNull("No confirmed sendable group; inspect the private membership receipt", candidate)
            val id = candidate!!
            val key = "$app:$user:$id"
            memo.invalidateGroup(id)
            val entered = CompletableDeferred<Unit>()
            val finish = CompletableDeferred<Unit>()
            val oldRead = async {
                runCatching { memo.get(key) { entered.complete(Unit); finish.await(); LinLanImManager.GroupSendPermission.Allowed } }
            }
            entered.await()
            val resolved = async(start = CoroutineStart.UNDISPATCHED) { LinLanImManager.resolveGroupSendPermission(id) }
            yield()
            memo.invalidateGroup(id)
            finish.complete(Unit)
            val value = resolved.await()
            assertTrue(oldRead.await().exceptionOrNull() is PermissionChanged)
            val expectRace = InstrumentationRegistry.getArguments().getString("ruiExpectPermissionRace") == "true"
            if (expectRace) {
                assertFalse(value.allowsSend)
                assertEquals("权限已变化或未确认，请重试", value.blockedHint)
            } else {
                assertTrue("A fresh confirmed read must recover after invalidation: ${value.blockedHint}", value.allowsSend)
            }
            report.put("invalidation_recovered", value.allowsSend).put("invalidation_hint", value.blockedHint)
            File(directory, "result.json").writeText(report.toString(2))
            if (InstrumentationRegistry.getArguments().getString("ruiCreateGroup") == "true") {
                val created = LinLanImManager.createCommunityGroup("权限回归-20261105-${System.currentTimeMillis().toString().takeLast(5)}", emptyList())
                report.put("new_group_created", true)
                try {
                    assertTrue(LinLanImManager.resolveGroupSendPermission(created).allowsSend)
                    assertEquals("This isolated test group must contain only this account", 1, LinLanImManager.fetchAllGroupMembers(created).size)
                    val store = LinLanAppContainer.sessionStore()
                    val saved = store.currentSession()
                    val session = AuthSession(sdkAppId = app, userId = user, userSig = data.getString("user_sig"),
                        userSigExpiresAt = data.optString("user_sig_expires_at"), token = data.getString("token"),
                        tokenExpiresAt = data.optString("token_expires_at"), refreshToken = data.optString("refresh_token"))
                    store.saveSession(session)
                    var activity: ActivityScenario<MainActivity>? = null
                    try {
                        activity = ActivityScenario.launch(MainActivity::class.java)
                        delay(6_000)
                        activity.onActivity { target ->
                            // Render the production chat component; do not depend on splash/navigation timing.
                            target.setContent { LinLanTheme {
                                LinLanChatScreen(onOpenRealname = {}, conversationId = "group_$created", title = "权限回归", isGroup = true, onBack = {})
                            } }
                        }
                        try {
                            compose.waitUntil(20_000) {
                                compose.onAllNodesWithText("权限回归", useUnmergedTree = true).fetchSemanticsNodes().isNotEmpty() &&
                                    compose.onAllNodes(hasSetTextAction()).fetchSemanticsNodes().isNotEmpty()
                            }
                        } catch (error: Throwable) {
                            File(directory, "ui-private.txt").writeText(compose.onRoot(useUnmergedTree = true).printToString())
                            throw error
                        }
                        compose.onNodeWithText("权限已变化或未确认，请重试").assertDoesNotExist()
                        report.put("new_group_composer_visible", true)
                        val text = "安卓建群权限回归，仅本账号测试"
                        compose.onAllNodes(hasSetTextAction())[0].performTextInput(text)
                        File(directory, "ui-private.txt").writeText(compose.onRoot(useUnmergedTree = true).printToString())
                        compose.onNodeWithText("发送").performClick()
                        compose.waitUntil(15_000) { compose.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty() }
                        report.put("new_group_UI_send_shown", true)
                        // Independently require a real SDK send acknowledgement, only in this one-member group.
                        val sent = LinLanImManager.sendTextMessage("权限回归 SDK 确认", null, created)
                        assertEquals(com.tencent.imsdk.v2.V2TIMMessage.V2TIM_MSG_STATUS_SEND_SUCC, sent.status)
                        report.put("new_group_SDK_send_acknowledged", true)
                    } finally {
                        activity?.close()
                        store.replaceSessionIfCurrent(session, saved)
                    }
                } finally {
                    LinLanImManager.dismissGroup(created)
                    report.put("new_test_group_dismissed", true)
                    File(directory, "result.json").writeText(report.toString(2))
                }
            }
            InstrumentationRegistry.getInstrumentation().sendStatus(2, android.os.Bundle().apply {
                putString("rui_group", "real_sdk_login=true joined_groups=${groups.size} invalidation_recovered=${value.allowsSend}")
            })
        } finally {
            LinLanImManager.logout()
        }
    }
}
