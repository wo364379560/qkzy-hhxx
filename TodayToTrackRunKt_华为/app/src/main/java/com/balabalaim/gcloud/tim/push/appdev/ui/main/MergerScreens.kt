package com.balabalaim.gcloud.tim.push.appdev.ui.main

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import android.widget.VideoView
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.InsertDriveFile
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.FileProvider
import coil.compose.AsyncImage
import com.balabalaim.gcloud.tim.push.appdev.im.LinLanImManager
import com.balabalaim.gcloud.tim.push.appdev.ui.common.LinLanNavBackButton
import com.tencent.imsdk.v2.V2TIMMessage
import kotlinx.coroutines.launch
import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanMergerDetailScreen(
    onBack: () -> Unit,
) {
    var source by remember { mutableStateOf(MergerDetailBus.pending) }
    val card = remember(source?.msgID) { source?.let { LinLanImManager.parseMergerCard(it) } }
    val children = remember { mutableStateListOf<V2TIMMessage>() }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var mediaPreview by remember { mutableStateOf<ChatMediaPreview?>(null) }
    val title = card?.title?.ifBlank { "聊天记录" } ?: "聊天记录"
    val context = LocalContext.current

    LaunchedEffect(source?.msgID) {
        val msg = source
        if (msg == null) {
            loading = false
            error = "聊天记录不存在"
            return@LaunchedEffect
        }
        if (card?.layersOverLimit == true) {
            loading = false
            error = "聊天记录层数过多，无法查看"
            return@LaunchedEffect
        }
        loading = true
        runCatching { LinLanImManager.downloadMergerMessage(msg) }
            .onSuccess {
                children.clear()
                children.addAll(it.sortedBy { m -> m.timestamp })
                error = null
            }
            .onFailure { error = it.message }
        loading = false
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title, maxLines = 1) },
                navigationIcon = { LinLanNavBackButton(onClick = onBack) },
            )
        },
        containerColor = Color(0xFFF2F2F7),
    ) { padding ->
        when {
            loading -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
            }
            error != null -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(error.orEmpty(), color = Color.Gray)
                }
            }
            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding)
                        .padding(horizontal = 12.dp, vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    items(children, key = { it.msgID ?: it.hashCode().toString() }) { msg ->
                        MergerChildRow(
                            msg = msg,
                            onOpenImage = { mediaPreview = ChatMediaPreview.Image(it) },
                            onOpenVideo = { mediaPreview = ChatMediaPreview.Video(it) },
                            onOpenFile = { path, name -> openMergerFile(context, path, name) },
                            onOpenMerger = {
                                MergerDetailBus.pending = it
                                source = it
                            },
                        )
                    }
                }
            }
        }
    }
    mediaPreview?.let { preview ->
        MergerMediaPreviewDialog(preview = preview, onDismiss = { mediaPreview = null })
    }
}

@Composable
private fun MergerChildRow(
    msg: V2TIMMessage,
    onOpenImage: (String) -> Unit,
    onOpenVideo: (String) -> Unit,
    onOpenFile: (path: String, name: String) -> Unit,
    onOpenMerger: (V2TIMMessage) -> Unit,
) {
    val name = LinLanImManager.displayNameOfMessage(msg).ifBlank { "用户" }
    val preview = LinLanImManager.messagePreview(msg)
    val time = remember(msg.timestamp) {
        SimpleDateFormat("MM/dd HH:mm", Locale.getDefault()).format(Date(msg.timestamp * 1000))
    }
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var loading by remember(msg.msgID) { mutableStateOf(false) }
    val tappable = msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE ||
        msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO ||
        msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_FILE ||
        msg.elemType == V2TIMMessage.V2TIM_ELEM_TYPE_MERGER

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(Color.White)
            .then(
                if (tappable) {
                    Modifier.clickable {
                        when (msg.elemType) {
                            V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> {
                                val url = LinLanImManager.imageUrlOf(msg, preferLarge = true)
                                    ?: LinLanImManager.imageUrlOf(msg)
                                if (!url.isNullOrBlank()) onOpenImage(url)
                            }
                            V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> {
                                if (loading) return@clickable
                                scope.launch {
                                    loading = true
                                    val path = LinLanImManager.resolveVideoPlayable(context, msg)
                                    loading = false
                                    if (path.isNullOrBlank()) {
                                        Toast.makeText(context, "视频暂不可播放", Toast.LENGTH_SHORT).show()
                                    } else {
                                        onOpenVideo(path)
                                    }
                                }
                            }
                            V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                                if (loading) return@clickable
                                scope.launch {
                                    loading = true
                                    val path = LinLanImManager.resolveFileLocal(context, msg)
                                    loading = false
                                    val fileName = msg.fileElem?.fileName ?: "文件"
                                    if (path.isNullOrBlank()) {
                                        Toast.makeText(context, "文件暂不可打开", Toast.LENGTH_SHORT).show()
                                    } else if (LinLanImManager.isImageFileName(fileName)) {
                                        onOpenImage(path)
                                    } else {
                                        onOpenFile(path, fileName)
                                    }
                                }
                            }
                            V2TIMMessage.V2TIM_ELEM_TYPE_MERGER -> onOpenMerger(msg)
                            else -> Unit
                        }
                    }
                } else {
                    Modifier
                },
            )
            .padding(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            modifier = Modifier
                .size(36.dp)
                .clip(CircleShape)
                .background(Color(0xFFD0E8FF)),
            contentAlignment = Alignment.Center,
        ) {
            if (!msg.faceUrl.isNullOrBlank()) {
                AsyncImage(
                    model = msg.faceUrl,
                    contentDescription = null,
                    modifier = Modifier.fillMaxSize(),
                    contentScale = ContentScale.Crop,
                )
            } else {
                Text(name.take(1), color = Color(0xFF007AFF), fontWeight = FontWeight.SemiBold)
            }
        }
        Spacer(modifier = Modifier.width(10.dp))
        Column(modifier = Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(name, fontWeight = FontWeight.Medium, fontSize = 14.sp, modifier = Modifier.weight(1f))
                Text(time, color = Color(0xFF8E8E93), fontSize = 11.sp)
            }
            Spacer(modifier = Modifier.height(4.dp))
            when (msg.elemType) {
                V2TIMMessage.V2TIM_ELEM_TYPE_IMAGE -> {
                    val url = LinLanImManager.imageUrlOf(msg) ?: LinLanImManager.imageUrlOf(msg, preferLarge = true)
                    if (!url.isNullOrBlank()) {
                        AsyncImage(
                            model = url,
                            contentDescription = "图片",
                            contentScale = ContentScale.Crop,
                            modifier = Modifier
                                .widthIn(max = 180.dp)
                                .heightIn(max = 180.dp)
                                .clip(RoundedCornerShape(8.dp)),
                        )
                    } else {
                        Text(preview, color = Color(0xFF333333), fontSize = 15.sp)
                    }
                }
                V2TIMMessage.V2TIM_ELEM_TYPE_VIDEO -> {
                    Box(
                        modifier = Modifier
                            .width(140.dp)
                            .height(180.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .background(Color(0xFF333333)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(Icons.Default.PlayArrow, contentDescription = null, tint = Color.White, modifier = Modifier.size(36.dp))
                        if (loading) {
                            CircularProgressIndicator(modifier = Modifier.size(24.dp), color = Color.White)
                        }
                    }
                }
                V2TIMMessage.V2TIM_ELEM_TYPE_FILE -> {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            Icons.AutoMirrored.Filled.InsertDriveFile,
                            contentDescription = null,
                            tint = Color(0xFF007AFF),
                            modifier = Modifier.size(28.dp),
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Column {
                            Text(
                                msg.fileElem?.fileName ?: "文件",
                                color = Color(0xFF333333),
                                fontSize = 15.sp,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                            Text(
                                if (loading) "加载中…" else "点击查看",
                                color = Color(0xFF8E8E93),
                                fontSize = 12.sp,
                            )
                        }
                    }
                }
                else -> Text(preview, color = Color(0xFF333333), fontSize = 15.sp)
            }
        }
    }
}

@Composable
fun MergerMessageCard(
    title: String,
    abstracts: List<String>,
    onClick: (() -> Unit)? = null,
) {
    Column(
        modifier = Modifier
            .widthIn(min = 180.dp, max = 240.dp)
            .clip(RoundedCornerShape(10.dp))
            .border(1.dp, Color(0xFFDDDDDD), RoundedCornerShape(10.dp))
            .background(Color.White)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 12.dp, vertical = 10.dp),
    ) {
        Text(
            title,
            fontWeight = FontWeight.SemiBold,
            fontSize = 15.sp,
            color = Color(0xFF111111),
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
        Spacer(modifier = Modifier.height(6.dp))
        abstracts.take(4).forEach { line ->
            Text(
                line,
                color = Color(0xFF8E8E93),
                fontSize = 12.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(bottom = 2.dp),
            )
        }
        HorizontalDivider(
            modifier = Modifier.padding(vertical = 8.dp),
            color = Color(0xFFE5E5EA),
        )
        Text("聊天记录", color = Color(0xFF8E8E93), fontSize = 12.sp)
    }
}

@Composable
private fun MergerMediaPreviewDialog(preview: ChatMediaPreview, onDismiss: () -> Unit) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black)
                .clickable(onClick = onDismiss),
        ) {
            when (preview) {
                is ChatMediaPreview.Image -> {
                    AsyncImage(
                        model = preview.model,
                        contentDescription = "图片预览",
                        contentScale = ContentScale.Fit,
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(8.dp),
                    )
                }
                is ChatMediaPreview.Video -> {
                    AndroidView(
                        factory = { ctx ->
                            VideoView(ctx).apply {
                                if (preview.pathOrUrl.startsWith("http")) {
                                    setVideoURI(Uri.parse(preview.pathOrUrl))
                                } else {
                                    setVideoPath(preview.pathOrUrl)
                                }
                                setOnPreparedListener { it.isLooping = false; start() }
                            }
                        },
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(8.dp)
                            .align(Alignment.Center),
                    )
                }
            }
        }
    }
}

private fun openMergerFile(context: Context, pathOrUrl: String, fileName: String) {
    runCatching {
        if (pathOrUrl.startsWith("http://") || pathOrUrl.startsWith("https://")) {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(pathOrUrl)))
            return
        }
        val file = File(pathOrUrl)
        val uri = FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            file,
        )
        val mime = when {
            LinLanImManager.isImageFileName(fileName) -> "image/*"
            fileName.lowercase().endsWith(".mp4") || fileName.lowercase().endsWith(".mov") -> "video/*"
            fileName.lowercase().endsWith(".pdf") -> "application/pdf"
            else -> "*/*"
        }
        val intent = Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, mime)
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        context.startActivity(Intent.createChooser(intent, "打开文件"))
    }.onFailure {
        Toast.makeText(context, it.message ?: "无法打开文件", Toast.LENGTH_SHORT).show()
    }
}
