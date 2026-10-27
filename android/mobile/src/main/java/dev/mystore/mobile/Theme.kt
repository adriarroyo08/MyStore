package dev.mystore.mobile

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

// Azul de marca (el del icono del lanzador)
val BrandBlue = Color(0xFF1B5E9E)

private val Light = lightColorScheme(
    primary = BrandBlue, onPrimary = Color.White,
    primaryContainer = Color(0xFFD3E4FF), onPrimaryContainer = Color(0xFF001C38),
    secondary = Color(0xFF545F71), onSecondary = Color.White,
    secondaryContainer = Color(0xFFD8E3F8), onSecondaryContainer = Color(0xFF111C2B),
    tertiary = Color(0xFF8B5000), onTertiary = Color.White,
    tertiaryContainer = Color(0xFFFFDCBE), onTertiaryContainer = Color(0xFF2D1600),
    background = Color(0xFFF8F9FF), onBackground = Color(0xFF191C20),
    surface = Color(0xFFF8F9FF), onSurface = Color(0xFF191C20),
    surfaceVariant = Color(0xFFDFE2EB), onSurfaceVariant = Color(0xFF43474E),
    outline = Color(0xFF73777F), outlineVariant = Color(0xFFC3C6CF),
    surfaceContainerLowest = Color.White, surfaceContainerLow = Color(0xFFF2F3FA),
    surfaceContainer = Color(0xFFECEEF4), surfaceContainerHigh = Color(0xFFE6E8EE),
    surfaceContainerHighest = Color(0xFFE1E2E8), surfaceDim = Color(0xFFD8DAE0), surfaceBright = Color(0xFFF8F9FF),
)

private val Dark = darkColorScheme(
    primary = Color(0xFFA2C9FF), onPrimary = Color(0xFF00315C),
    primaryContainer = Color(0xFF004881), onPrimaryContainer = Color(0xFFD3E4FF),
    secondary = Color(0xFFBCC7DC), onSecondary = Color(0xFF263141),
    secondaryContainer = Color(0xFF3D4758), onSecondaryContainer = Color(0xFFD8E3F8),
    tertiary = Color(0xFFFFB870), onTertiary = Color(0xFF4A2800),
    tertiaryContainer = Color(0xFF693C00), onTertiaryContainer = Color(0xFFFFDCBE),
    background = Color(0xFF111418), onBackground = Color(0xFFE1E2E8),
    surface = Color(0xFF111418), onSurface = Color(0xFFE1E2E8),
    surfaceVariant = Color(0xFF43474E), onSurfaceVariant = Color(0xFFC3C6CF),
    outline = Color(0xFF8D9199), outlineVariant = Color(0xFF43474E),
    surfaceContainerLowest = Color(0xFF0C0E13), surfaceContainerLow = Color(0xFF191C20),
    surfaceContainer = Color(0xFF1D2024), surfaceContainerHigh = Color(0xFF282A2F),
    surfaceContainerHighest = Color(0xFF33353A), surfaceDim = Color(0xFF111418), surfaceBright = Color(0xFF37393E),
)

private val base = Typography()
private val AppTypography = Typography(
    displaySmall = base.displaySmall.copy(fontWeight = FontWeight.Bold, letterSpacing = (-0.5).sp),
    headlineMedium = base.headlineMedium.copy(fontWeight = FontWeight.Bold, letterSpacing = (-0.25).sp),
    headlineSmall = base.headlineSmall.copy(fontWeight = FontWeight.SemiBold),
    titleLarge = base.titleLarge.copy(fontWeight = FontWeight.SemiBold),
    titleMedium = base.titleMedium.copy(fontWeight = FontWeight.SemiBold),
    labelLarge = base.labelLarge.copy(fontWeight = FontWeight.SemiBold, letterSpacing = 0.2.sp),
    bodyLarge = base.bodyLarge.copy(lineHeight = 26.sp),
)

private val AppShapes = Shapes(
    extraSmall = RoundedCornerShape(8.dp), small = RoundedCornerShape(12.dp),
    medium = RoundedCornerShape(16.dp), large = RoundedCornerShape(24.dp), extraLarge = RoundedCornerShape(32.dp),
)

// Material You en Android 12+; si no, el esquema de marca
@Composable
fun MyStoreTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    val ctx = LocalContext.current
    val scheme = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> if (dark) dynamicDarkColorScheme(ctx) else dynamicLightColorScheme(ctx)
        dark -> Dark
        else -> Light
    }
    MaterialTheme(colorScheme = scheme, typography = AppTypography, shapes = AppShapes, content = content)
}
