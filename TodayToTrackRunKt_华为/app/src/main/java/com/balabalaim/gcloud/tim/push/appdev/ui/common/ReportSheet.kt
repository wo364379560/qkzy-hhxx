package com.balabalaim.gcloud.tim.push.appdev.ui.common

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Gavel
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.Report
import androidx.compose.material.icons.filled.SentimentVeryDissatisfied
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material.icons.filled.Whatshot
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import kotlinx.coroutines.launch

private data class ReportReason(
    val key: String,
    val title: String,
    val icon: ImageVector,
    val tint: Color,
)

private val ReportReasons = listOf(
    ReportReason("spam", "垃圾信息 / 广告", Icons.Default.Delete, Color(0xFFFF9500)),
    ReportReason("abuse", "骚扰 / 辱骂", Icons.Default.SentimentVeryDissatisfied, Color(0xFFFF3B30)),
    ReportReason("violence", "暴力 / 威胁", Icons.Default.Whatshot, Color(0xFFAF52DE)),
    ReportReason("fraud", "欺诈 / 诈骗", Icons.Default.Report, Color(0xFF007AFF)),
    ReportReason("porn", "色情 / 低俗", Icons.Default.Warning, Color(0xFF34C759)),
    ReportReason("illegal", "违法违规", Icons.Default.Gavel, Color(0xFFFFCC00)),
)

private val OtherReason = ReportReason("other", "其他", Icons.Default.MoreHoriz, Color(0xFF8E8E93))

/** Full navigation destination used by personal/group details. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanReportScreen(
    targetId: String,
    targetType: String,
    targetName: String,
    onBack: () -> Unit,
) {
    var selected by remember { mutableStateOf<String?>(null) }
    var description by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }
    val danger = Color(0xFFFF3B30)

    fun submit() {
        val category = selected
        if (category.isNullOrBlank()) {
            Toast.makeText(context, "请选择举报原因", Toast.LENGTH_SHORT).show()
            return
        }
        if (submitting) return
        scope.launch {
            submitting = true
            runCatching {
                api.submitReport(
                    targetId = targetId,
                    targetType = targetType,
                    targetName = targetName,
                    category = category,
                    description = description.trim().ifBlank { null },
                )
            }.onSuccess {
                Toast.makeText(context, "举报已提交", Toast.LENGTH_SHORT).show()
                onBack()
            }.onFailure {
                Toast.makeText(context, it.message ?: "提交失败", Toast.LENGTH_SHORT).show()
            }
            submitting = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("举报") },
                navigationIcon = { LinLanNavBackButton(onClick = onBack) },
                actions = {
                    TextButton(enabled = !submitting, onClick = ::submit) {
                        Text(if (submitting) "提交中…" else "提交", color = danger)
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(Color(0xFFF2F2F7))
                .verticalScroll(rememberScrollState())
                .padding(padding)
                .padding(16.dp),
        ) {
            Text(
                "举报 ${targetName.ifBlank { "该账号" }}",
                fontSize = 17.sp,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                "请选择举报原因，我们会认真处理",
                color = Color(0xFF8E8E93),
                fontSize = 13.sp,
                modifier = Modifier.padding(top = 4.dp, bottom = 14.dp),
            )
            (ReportReasons + OtherReason).forEach { reason ->
                ReasonCard(
                    reason = reason,
                    selected = selected == reason.key,
                    onClick = { selected = reason.key },
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(10.dp))
            }
            Text("补充说明（选填）", fontSize = 14.sp, fontWeight = FontWeight.Medium)
            Spacer(Modifier.height(8.dp))
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 120.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color.White)
                    .padding(12.dp),
            ) {
                BasicTextField(
                    value = description,
                    onValueChange = { if (it.length <= 200) description = it },
                    modifier = Modifier.fillMaxWidth().heightIn(min = 92.dp),
                    textStyle = androidx.compose.ui.text.TextStyle(fontSize = 14.sp, color = Color(0xFF111111)),
                    cursorBrush = SolidColor(danger),
                    decorationBox = { inner ->
                        if (description.isEmpty()) {
                            Text("请描述具体情况，帮助我们更快处理…", color = Color(0xFFAEAEB2), fontSize = 14.sp)
                        }
                        inner()
                    },
                )
                Text(
                    "${description.length}/200",
                    color = Color(0xFFAEAEB2),
                    fontSize = 11.sp,
                    modifier = Modifier.align(Alignment.BottomEnd),
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LinLanReportSheet(
    targetId: String,
    targetType: String,
    targetName: String,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var selected by remember { mutableStateOf<String?>(null) }
    var description by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val api = remember { LinLanAppContainer.apiClient() }
    val danger = Color(0xFFFF3B30)

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Color.White,
        shape = RoundedCornerShape(topStart = 20.dp, topEnd = 20.dp),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp)
                .padding(bottom = 28.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.Top,
            ) {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(RoundedCornerShape(10.dp))
                        .background(danger.copy(alpha = 0.12f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Default.Shield, null, tint = danger, modifier = Modifier.size(20.dp))
                }
                Spacer(modifier = Modifier.width(10.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text("举报", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                    Text(
                        "请选择举报原因，我们会认真处理",
                        color = Color(0xFF8E8E93),
                        fontSize = 12.sp,
                        modifier = Modifier.padding(top = 2.dp),
                    )
                }
                IconButton(onClick = onDismiss) {
                    Icon(
                        Icons.Default.Close,
                        contentDescription = "关闭",
                        tint = Color(0xFF8E8E93),
                    )
                }
            }

            Spacer(modifier = Modifier.height(14.dp))
            val grid = ReportReasons.chunked(2)
            grid.forEach { row ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    row.forEach { reason ->
                        ReasonCard(
                            reason = reason,
                            selected = selected == reason.key,
                            onClick = { selected = reason.key },
                            modifier = Modifier.weight(1f),
                        )
                    }
                    if (row.size == 1) Spacer(modifier = Modifier.weight(1f))
                }
                Spacer(modifier = Modifier.height(10.dp))
            }
            ReasonCard(
                reason = OtherReason,
                selected = selected == OtherReason.key,
                onClick = { selected = OtherReason.key },
                modifier = Modifier.fillMaxWidth(),
            )

            Spacer(modifier = Modifier.height(16.dp))
            Text("补充说明（选填）", fontSize = 14.sp, fontWeight = FontWeight.Medium)
            Spacer(modifier = Modifier.height(8.dp))
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 110.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(Color(0xFFF2F2F7))
                    .padding(12.dp),
            ) {
                BasicTextField(
                    value = description,
                    onValueChange = { if (it.length <= 200) description = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 80.dp),
                    textStyle = androidx.compose.ui.text.TextStyle(
                        fontSize = 14.sp,
                        color = Color(0xFF111111),
                    ),
                    cursorBrush = SolidColor(danger),
                    decorationBox = { inner ->
                        if (description.isEmpty()) {
                            Text(
                                "请描述具体情况，帮助我们更快处理…",
                                color = Color(0xFFAEAEB2),
                                fontSize = 14.sp,
                            )
                        }
                        inner()
                    },
                )
                Text(
                    "${description.length}/200",
                    color = Color(0xFFAEAEB2),
                    fontSize = 11.sp,
                    modifier = Modifier.align(Alignment.BottomEnd),
                )
            }

            Spacer(modifier = Modifier.height(18.dp))
            Button(
                onClick = {
                    val category = selected
                    if (category.isNullOrBlank()) {
                        Toast.makeText(context, "请选择举报原因", Toast.LENGTH_SHORT).show()
                        return@Button
                    }
                    if (submitting) return@Button
                    scope.launch {
                        submitting = true
                        runCatching {
                            api.submitReport(
                                targetId = targetId,
                                targetType = targetType,
                                targetName = targetName,
                                category = category,
                                description = description.trim().ifBlank { null },
                            )
                        }.onSuccess {
                            Toast.makeText(context, "举报已提交", Toast.LENGTH_SHORT).show()
                            onDismiss()
                        }.onFailure {
                            Toast.makeText(context, it.message ?: "提交失败", Toast.LENGTH_SHORT).show()
                        }
                        submitting = false
                    }
                },
                enabled = !submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
                shape = RoundedCornerShape(24.dp),
                colors = ButtonDefaults.buttonColors(containerColor = danger),
            ) {
                Icon(Icons.Default.Shield, null, tint = Color.White, modifier = Modifier.size(18.dp))
                Spacer(modifier = Modifier.width(6.dp))
                Text(if (submitting) "提交中…" else "提交举报", fontSize = 16.sp)
            }
        }
    }
}

@Composable
private fun ReasonCard(
    reason: ReportReason,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val border = if (selected) Color(0xFFFF3B30) else Color(0xFFE5E5EA)
    val bg = if (selected) Color(0xFFFFF5F5) else Color(0xFFFAFAFA)
    Row(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .border(1.dp, border, RoundedCornerShape(12.dp))
            .background(bg)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(28.dp)
                .clip(RoundedCornerShape(8.dp))
                .background(reason.tint.copy(alpha = 0.15f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(reason.icon, null, tint = reason.tint, modifier = Modifier.size(16.dp))
        }
        Spacer(modifier = Modifier.width(8.dp))
        Text(
            reason.title,
            modifier = Modifier.weight(1f),
            fontSize = 13.sp,
            maxLines = 1,
        )
        Box(
            modifier = Modifier
                .size(18.dp)
                .clip(CircleShape)
                .border(1.5.dp, if (selected) Color(0xFFFF3B30) else Color(0xFFC7C7CC), CircleShape)
                .background(if (selected) Color(0xFFFF3B30) else Color.Transparent),
            contentAlignment = Alignment.Center,
        ) {
            if (selected) {
                Box(
                    modifier = Modifier
                        .size(7.dp)
                        .clip(CircleShape)
                        .background(Color.White),
                )
            }
        }
    }
}
