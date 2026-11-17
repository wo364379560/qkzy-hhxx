package com.balabalaim.gcloud.tim.push.appdev.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val LightColorScheme = lightColorScheme(
    primary = LinLanAccent,
    onPrimary = LinLanOnAccent,
    secondary = LinLanInk,
    onSecondary = LinLanOnAccent,
    tertiary = LinLanAccent,
    background = LinLanCanvas,
    onBackground = LinLanInk,
    surface = LinLanSurface,
    onSurface = LinLanInk,
    error = LinLanDanger,
    outline = LinLanHairline,
    primaryContainer = LinLanAccentSoft,
    surfaceVariant = LinLanCanvas,
    onSurfaceVariant = LinLanInkSecondary,
)

@Composable
fun LinLanTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = LightColorScheme,
        typography = Typography,
        content = content,
    )
}
