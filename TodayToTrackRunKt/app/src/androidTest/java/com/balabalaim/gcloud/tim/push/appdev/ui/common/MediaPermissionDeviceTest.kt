package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.Manifest
import android.app.Activity
import android.content.ClipData
import android.content.Intent
import android.net.Uri
import android.os.ParcelFileDescriptor
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.test.core.app.ActivityScenario
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.balabalaim.gcloud.tim.push.appdev.MainActivity
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.LinLanTheme
import java.io.File
import kotlinx.coroutines.*
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Assume.assumeTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class MediaPermissionDeviceTest {
    @get:Rule val compose = createEmptyComposeRule()
    private val instrumentation get() = InstrumentationRegistry.getInstrumentation()
    private val context get() = instrumentation.targetContext
    private val name = "rui_feature_permissions_qa_v1"
    private val features = listOf(FeaturePermission.IMAGE_PICK, FeaturePermission.VIDEO_PICK, FeaturePermission.FILE_PICK, FeaturePermission.MICROPHONE)
    private fun shell(command: String) = ParcelFileDescriptor.AutoCloseInputStream(instrumentation.uiAutomation.executeShellCommand(command))
        .bufferedReader().use { it.readText() }

    @Composable private fun Harness(store: FeaturePermissionStore) {
        Column {
            features.forEach { feature -> key(feature) {
                var count by remember { mutableIntStateOf(0) }
                val action = rememberExplainedPermissionAction(feature.systemPermissions, "测试${feature.title}用途",
                    listOf(feature), store) { count++ }
                Button(onClick = action) { Text(feature.title) }
                Text("${feature.key}:$count")
            } }
            for (video in listOf(false, true)) key(video) {
                val launcher = rememberLauncherForActivityResult(LinLanVisualMediaPicker(video = video, multiple = true)) { }
                Button(onClick = { launchVisualMediaPicker(context, launcher) }) { Text(if (video) "打开视频相册" else "打开图片相册") }
            }
        }
    }

    @Test fun choicesPersistAndMediaUsesGallery() = runBlocking {
        val args = InstrumentationRegistry.getArguments()
        assumeTrue(args.getString("ruiMediaQa") == "true")
        val cold = args.getString("ruiMediaStage") == "cold"
        if (!cold) context.getSharedPreferences(name, 0).edit().clear().commit()
        val store = FeaturePermissionStore(context, name)
        val report = JSONObject().put("stage", if (cold) "cold" else "choices")
        val scenario = ActivityScenario.launch(MainActivity::class.java)
        try {
            delay(5_000)
            scenario.onActivity { it.setContent { LinLanTheme { Harness(store) } } }
            compose.waitUntil(20_000) { compose.onAllNodesWithText("选择图片与头像").fetchSemanticsNodes(atLeastOneRootRequired = false).isNotEmpty() }
            if (!cold) {
                for (feature in features) {
                    compose.onNodeWithText(feature.title).performClick()
                    compose.onNodeWithText("拒绝").performClick()
                    assertEquals(FeaturePermissionPolicy.Choice.DENIED, FeaturePermissionStore(context, name).choice(feature))
                    repeat(3) { compose.onNodeWithText(feature.title).performClick() }
                    compose.onNodeWithText("睿之禾权限说明").assertDoesNotExist()
                    compose.onNodeWithText("${feature.key}:0").assertExists()
                }
                report.put("image_video_file_microphone_denial_blocks_action_without_repeat_dialog", true)
                for (feature in features.filter { it != FeaturePermission.MICROPHONE }) {
                    store.policy.reset(feature)
                    compose.onNodeWithText(feature.title).performClick()
                    compose.onNodeWithText("允许").performClick()
                    compose.onNodeWithText("${feature.key}:1").assertExists()
                    compose.onNodeWithText(feature.title).performClick()
                    compose.onNodeWithText("${feature.key}:2").assertExists()
                    compose.onNodeWithText("睿之禾权限说明").assertDoesNotExist()
                }
                context.getSharedPreferences(name, 0).edit().putBoolean("qa_flush", true).commit()
                report.put("reset_reprompts_once_then_allow_runs_without_repeated_dialog", true)
            } else {
                for (feature in features) {
                    compose.onNodeWithText(feature.title).performClick()
                    compose.onNodeWithText("睿之禾权限说明").assertDoesNotExist()
                    compose.onNodeWithText("${feature.key}:${if (feature == FeaturePermission.MICROPHONE) 0 else 1}").assertExists()
                }
                report.put("new_process_keeps_allow_and_deny_choices", true)
                for (video in listOf(false, true)) {
                    val picker = LinLanVisualMediaPicker(video = video, multiple = true)
                    val intent = picker.createIntent(context, Unit)
                    val native = androidx.activity.result.contract.ActivityResultContracts.PickVisualMedia.isPhotoPickerAvailable(context)
                    if (!native) {
                        assertEquals(Intent.ACTION_GET_CONTENT, intent.action)
                        assertNull(intent.data)
                        assertNotNull(intent.component)
                        assertFalse(intent.component!!.packageName.contains("documentsui", ignoreCase = true))
                        assertTrue(intent.getBooleanExtra(Intent.EXTRA_ALLOW_MULTIPLE, false))
                    }
                    assertEquals(if (video) "video/*" else "image/*", intent.type)
                    assertNotEquals(Intent.ACTION_OPEN_DOCUMENT, intent.action)
                    compose.onNodeWithText(if (video) "打开视频相册" else "打开图片相册").performClick()
                    withTimeout(10_000) {
                        while (true) {
                            val focus = shell("dumpsys window windows").lineSequence().firstOrNull { it.contains("mCurrentFocus=") }.orEmpty()
                            if (intent.component?.packageName?.let { focus.contains(it) } == true ||
                                (native && focus.contains("photopicker", ignoreCase = true))) break
                            delay(200)
                        }
                    }
                    shell("input keyevent 4")
                    compose.waitUntil(10_000) { compose.onAllNodesWithText("打开图片相册").fetchSemanticsNodes(atLeastOneRootRequired = false).isNotEmpty() }
                    report.put(if (video) "real_video_gallery_opened" else "real_image_gallery_opened", true)
                }
                val first = Uri.parse("content://qa/media/1"); val second = Uri.parse("content://qa/media/2")
                val result = Intent().setData(first).apply {
                    clipData = ClipData.newRawUri("QA", first).apply { addItem(ClipData.Item(second)) }
                }
                assertEquals(listOf(first, second), LinLanVisualMediaPicker(multiple = true).parseResult(Activity.RESULT_OK, result))
                assertEquals(listOf(first), LinLanVisualMediaPicker().parseResult(Activity.RESULT_OK, result))
                assertTrue(LinLanVisualMediaPicker().parseResult(Activity.RESULT_CANCELED, result).isEmpty())
                val invalid = Intent().setData(Uri.parse("file:///private/file"))
                assertTrue(LinLanVisualMediaPicker().parseResult(Activity.RESULT_OK, invalid).isEmpty())
                report.put("single_multi_cancel_and_non_content_result_rules_passed", true)
            }
        } finally {
            scenario.close()
            if (cold) context.getSharedPreferences(name, 0).edit().clear().commit()
            val directory = context.getExternalFilesDir("rui-media-qa")!!
            directory.mkdirs(); File(directory, "result.json").writeText(report.toString(2))
        }
    }
}
