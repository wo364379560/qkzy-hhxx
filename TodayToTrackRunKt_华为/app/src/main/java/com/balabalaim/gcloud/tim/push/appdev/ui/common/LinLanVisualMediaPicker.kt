package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.net.Uri
import android.provider.MediaStore
import android.widget.Toast
import androidx.activity.result.ActivityResultLauncher
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContract
import androidx.activity.result.contract.ActivityResultContracts

/** Never falls back to DocumentsUI for images or videos on pre-photo-picker devices. */
class LinLanVisualMediaPicker(private val video: Boolean = false, private val multiple: Boolean = false) :
    ActivityResultContract<Unit, List<Uri>>() {
    override fun createIntent(context: Context, input: Unit): Intent {
        if (ActivityResultContracts.PickVisualMedia.isPhotoPickerAvailable(context)) {
            val request = PickVisualMediaRequest(if (video) ActivityResultContracts.PickVisualMedia.VideoOnly
                else ActivityResultContracts.PickVisualMedia.ImageOnly)
            return if (multiple) {
                val contract = if (video) ActivityResultContracts.PickMultipleVisualMedia(9)
                    else ActivityResultContracts.PickMultipleVisualMedia()
                contract.createIntent(context, request)
            } else ActivityResultContracts.PickVisualMedia().createIntent(context, request)
        }
        val intent = Intent(Intent.ACTION_PICK).apply {
            setDataAndType(if (video) MediaStore.Video.Media.EXTERNAL_CONTENT_URI
                else MediaStore.Images.Media.EXTERNAL_CONTENT_URI, if (video) "video/*" else "image/*")
            putExtra(Intent.EXTRA_ALLOW_MULTIPLE, multiple)
        }
        // A browser can also register ACTION_PICK. Prefer the installed system media picker.
        val gallery = context.packageManager.queryIntentActivities(intent, PackageManager.MATCH_DEFAULT_ONLY)
            .firstOrNull { info ->
                val app = info.activityInfo.applicationInfo
                (app.flags and (ApplicationInfo.FLAG_SYSTEM or ApplicationInfo.FLAG_UPDATED_SYSTEM_APP)) != 0 &&
                    !info.activityInfo.packageName.contains("documentsui", ignoreCase = true)
            } ?: throw ActivityNotFoundException("No system gallery picker")
        if (!multiple) return intent.setClassName(gallery.activityInfo.packageName, gallery.activityInfo.name)

        // ACTION_PICK is commonly single-select. GET_CONTENT supports ALLOW_MULTIPLE;
        // bind it to the gallery package so DocumentsUI/file managers cannot intercept it.
        val multiIntent = Intent(Intent.ACTION_GET_CONTENT).apply {
            type = if (video) "video/*" else "image/*"
            addCategory(Intent.CATEGORY_OPENABLE)
            setPackage(gallery.activityInfo.packageName)
            putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true)
        }
        val multiGallery = context.packageManager.queryIntentActivities(multiIntent, PackageManager.MATCH_DEFAULT_ONLY)
            .firstOrNull() ?: throw ActivityNotFoundException("System gallery has no multiple-selection entry")
        return multiIntent.setClassName(multiGallery.activityInfo.packageName, multiGallery.activityInfo.name)
    }

    override fun parseResult(resultCode: Int, intent: Intent?): List<Uri> {
        if (resultCode != Activity.RESULT_OK || intent == null) return emptyList()
        val selected = buildList {
            intent.data?.let { add(it) }
            intent.clipData?.let { clips -> for (index in 0 until clips.itemCount) clips.getItemAt(index).uri?.let { add(it) } }
        }.filter { it.scheme == "content" }.distinct()
        // Legacy galleries may ignore selection limits; the existing video queue checks all results.
        return if (multiple) selected else selected.take(1)
    }
}

fun launchVisualMediaPicker(context: Context, launcher: ActivityResultLauncher<Unit>) {
    try { launcher.launch(Unit) }
    catch (_: ActivityNotFoundException) { Toast.makeText(context, "未找到可用相册，请先安装或启用系统图库", Toast.LENGTH_LONG).show() }
    catch (_: SecurityException) { Toast.makeText(context, "无法打开相册，请检查系统图库是否可用", Toast.LENGTH_LONG).show() }
}
