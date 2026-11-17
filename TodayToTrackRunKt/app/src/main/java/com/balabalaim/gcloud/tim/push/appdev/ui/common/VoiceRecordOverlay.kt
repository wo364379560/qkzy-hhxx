package com.balabalaim.gcloud.tim.push.appdev.ui.common

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.balabalaim.gcloud.tim.push.appdev.ui.theme.MintAccent
import kotlin.math.max

@Composable
fun LinLanVoiceRecordOverlay(
    visible: Boolean,
    cancelMode: Boolean,
    level: Float,
) {
    if (!visible) return
    val pulse = rememberInfiniteTransition(label = "voicePulse")
    val scale by pulse.animateFloat(
        initialValue = 1f,
        targetValue = 1.28f,
        animationSpec = infiniteRepeatable(
            animation = tween(700, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "voicePulseScale",
    )
    val alpha by pulse.animateFloat(
        initialValue = 0.55f,
        targetValue = 0.15f,
        animationSpec = infiniteRepeatable(
            animation = tween(700, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse,
        ),
        label = "voicePulseAlpha",
    )
    val panelColor = if (cancelMode) Color(0xF2C72E2E) else Color(0xF01F1F1F)
    val iconBg = if (cancelMode) Color.White.copy(alpha = 0.2f) else MintAccent
    val clamped = level.coerceIn(0.08f, 1f)

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black.copy(alpha = 0.28f)),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier
                .size(176.dp)
                .background(panelColor, RoundedCornerShape(18.dp)),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Box(contentAlignment = Alignment.Center) {
                if (!cancelMode) {
                    Box(
                        modifier = Modifier
                            .size(72.dp)
                            .scale(scale)
                            .background(MintAccent.copy(alpha = alpha), CircleShape),
                    )
                }
                Box(
                    modifier = Modifier
                        .size(56.dp)
                        .background(iconBg, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = if (cancelMode) Icons.Default.Close else Icons.Default.Mic,
                        contentDescription = null,
                        tint = Color.White,
                        modifier = Modifier.size(26.dp),
                    )
                }
            }
            if (!cancelMode) {
                Spacer(Modifier.height(14.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    val heights = listOf(10f, 14f, 22f, 14f, 10f)
                    val factors = listOf(0.55f, 0.75f, 1f, 0.75f, 0.55f)
                    heights.forEachIndexed { index, base ->
                        val h = max(6f, base * (0.35f + clamped * factors[index]))
                        Box(
                            modifier = Modifier
                                .width(4.dp)
                                .height(h.dp)
                                .background(Color.White.copy(alpha = 0.85f), RoundedCornerShape(2.dp)),
                        )
                    }
                }
            }
            Spacer(Modifier.height(10.dp))
            Text(
                text = if (cancelMode) "松开手指，取消发送" else "松开 发送\n上滑 取消",
                color = Color.White,
                fontSize = 14.sp,
                textAlign = TextAlign.Center,
                lineHeight = 18.sp,
            )
        }
    }
}
