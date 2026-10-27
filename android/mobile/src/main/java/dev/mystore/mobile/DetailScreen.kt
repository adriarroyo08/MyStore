package dev.mystore.mobile

import android.graphics.Bitmap
import androidx.activity.compose.BackHandler
import androidx.compose.animation.*
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.Image
import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.KeyboardArrowDown
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dev.mystore.core.*
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.util.Locale

// Scroll inicial del detalle, solo lo fija MainActivity en debug
var debugScroll = 0

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DetailScreen(vm: StoreViewModel, app: StoreApp, onPermission: () -> Unit) {
    // Índice de la captura abierta en el visor (null = cerrado); atrás cierra el visor primero
    var viewer by remember { mutableStateOf<Int?>(null) }
    viewer?.let { i ->
        BackHandler { viewer = null }
        ShotViewer(vm, app.screenshots, i) { viewer = null }
        return
    }
    val scroll = rememberScrollState(debugScroll)
    val cs = MaterialTheme.colorScheme
    val tint = dominantColor(vm.icons["${app.id}:${app.versionCode}"]) ?: cs.primary
    val heroTop = lerp(cs.surface, tint, if (isSystemInDarkTheme()) 0.30f else 0.22f)
    val collapsed = scroll.value > with(LocalDensity.current) { 200.dp.toPx() }

    Box(Modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize().verticalScroll(scroll)) {
            Hero(vm, app, heroTop)
            StatsRow(app)
            ActionArea(vm, app, onPermission)
            if (app.screenshots.isNotEmpty()) Screenshots(vm, app.screenshots) { viewer = it }
            About(app)
            if (app.notes.isNotBlank()) Section("Novedades") {
                Surface(shape = MaterialTheme.shapes.large, color = cs.surfaceContainerLow) {
                    Column(Modifier.fillMaxWidth().padding(20.dp)) {
                        Text("Versión ${app.versionName}", style = MaterialTheme.typography.titleSmall, color = cs.primary)
                        Spacer(Modifier.height(6.dp))
                        Text(app.notes, style = MaterialTheme.typography.bodyLarge)
                    }
                }
            }
            Section("Información") { InfoCard(app) }
            Spacer(Modifier.navigationBarsPadding().height(32.dp))
        }
        // Barra superior transparente que se vuelve sólida al pasar la cabecera
        val barColor by androidx.compose.animation.animateColorAsState(if (collapsed) cs.surfaceContainer else Color.Transparent, label = "bar")
        TopAppBar(
            title = {
                AnimatedVisibility(collapsed, enter = fadeIn() + slideInVertically { it / 2 }, exit = fadeOut()) {
                    Text(app.label, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            },
            navigationIcon = { BackButton { vm.screen = Screen.List } },
            colors = TopAppBarDefaults.topAppBarColors(containerColor = barColor),
        )
    }
}

@Composable
private fun Hero(vm: StoreViewModel, app: StoreApp, top: Color) {
    val cs = MaterialTheme.colorScheme
    Box(Modifier.fillMaxWidth().background(Brush.verticalGradient(listOf(top, cs.surface)))) {
        Column(
            Modifier.fillMaxWidth().statusBarsPadding().padding(top = 72.dp, start = 24.dp, end = 24.dp, bottom = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(Modifier.shadow(16.dp, RoundedCornerShape(31.dp), ambientColor = top, spotColor = top)) {
                AppIcon(vm, app, 112.dp, shared = true)
            }
            Spacer(Modifier.height(20.dp))
            Text(app.label, style = MaterialTheme.typography.headlineMedium, textAlign = TextAlign.Center)
            Spacer(Modifier.height(4.dp))
            Text(app.pkg, style = MaterialTheme.typography.bodyMedium, color = cs.onSurfaceVariant)
        }
    }
}

@Composable
private fun StatsRow(app: StoreApp) {
    val cs = MaterialTheme.colorScheme
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 16.dp).height(IntrinsicSize.Min),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Stat(app.versionName.ifBlank { "—" }, "Versión", Modifier.weight(1f))
        VerticalDivider(Modifier.height(32.dp), color = cs.outlineVariant)
        Stat(mb(app.size), "Tamaño", Modifier.weight(1f))
        VerticalDivider(Modifier.height(32.dp), color = cs.outlineVariant)
        Stat(relative(app.updatedAt) ?: "—", "Actualizada", Modifier.weight(1f))
    }
}

@Composable
private fun Stat(value: String, label: String, modifier: Modifier) {
    Column(modifier.padding(horizontal = 4.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Text(value, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun ActionArea(vm: StoreViewModel, app: StoreApp, onPermission: () -> Unit) {
    val cs = MaterialTheme.colorScheme
    val haptic = LocalHapticFeedback.current
    val s = vm.installStates[app.id]
    val action = actionFor(app.versionCode, vm.installed[app.id])
    Column(Modifier.padding(horizontal = 24.dp, vertical = 8.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            // Con actualización pendiente se puede seguir abriendo la versión instalada
            if (action == AppAction.UPDATE && s !is InstallState.Downloading && s !is InstallState.Installing) {
                OutlinedButton(
                    onClick = { vm.installer.launch(app.pkg) }, Modifier.weight(1f).height(56.dp),
                ) { Text("Abrir", style = MaterialTheme.typography.titleSmall) }
            }
            val busy = s is InstallState.Downloading || s is InstallState.Installing
            val progress by animateFloatAsState((s as? InstallState.Downloading)?.progress ?: 0f, label = "p")
            val bg by androidx.compose.animation.animateColorAsState(
                if (busy) cs.secondaryContainer else if (action == AppAction.OPEN) cs.primaryContainer else cs.primary, label = "bg",
            )
            val fg = if (busy) cs.onSecondaryContainer else if (action == AppAction.OPEN) cs.onPrimaryContainer else cs.onPrimary
            Box(
                Modifier.weight(if (action == AppAction.UPDATE && !busy) 1.4f else 1f).height(56.dp).clip(CircleShape).background(bg)
                    .clickable(enabled = s !is InstallState.Downloading) {
                        haptic.performHapticFeedback(HapticFeedbackType.Confirm)
                        vm.act(app, onPermission)
                    },
                contentAlignment = Alignment.Center,
            ) {
                if (s is InstallState.Downloading) Box(
                    Modifier.align(Alignment.CenterStart).fillMaxHeight().fillMaxWidth(progress).background(cs.primary.copy(alpha = 0.28f)),
                )
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (s is InstallState.Installing) {
                        CircularProgressIndicator(Modifier.size(18.dp), color = fg, strokeWidth = 2.dp)
                        Spacer(Modifier.width(10.dp))
                    }
                    Text(
                        when {
                            s is InstallState.Downloading -> "Descargando  ${(s.progress * 100).toInt()} %"
                            s is InstallState.Installing -> "Instalando…"
                            action == AppAction.INSTALL -> "Instalar"
                            action == AppAction.UPDATE -> "Actualizar"
                            else -> "Abrir"
                        },
                        style = MaterialTheme.typography.titleSmall, color = fg,
                    )
                }
            }
        }
        AnimatedVisibility(s is InstallState.Failed) {
            val msg = (s as? InstallState.Failed)?.message.orEmpty()
            Row(Modifier.padding(top = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Rounded.Warning, null, Modifier.size(18.dp), tint = cs.error)
                Spacer(Modifier.width(8.dp))
                Text(msg, style = MaterialTheme.typography.bodyMedium, color = cs.error)
            }
        }
    }
}

@Composable
private fun Screenshots(vm: StoreViewModel, shots: List<String>, onOpen: (Int) -> Unit) {
    Section("Capturas", padContent = false) {
        LazyRow(contentPadding = PaddingValues(horizontal = 24.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            itemsIndexed(shots) { i, url ->
                val bmp = vm.shot(url)
                LaunchedEffect(url, bmp == null) { vm.loadShot(url) }
                val m = Modifier.height(260.dp).clip(RoundedCornerShape(20.dp)).clickable { onOpen(i) }
                if (bmp != null) Image(bmp, "Captura ${i + 1}", m, contentScale = ContentScale.FillHeight)
                else Box(m.width(146.dp).background(shimmer()))
            }
        }
    }
}

@Composable
private fun About(app: StoreApp) {
    var expanded by rememberSaveable { mutableStateOf(false) }
    var overflows by remember { mutableStateOf(false) }
    val desc = app.description.ifBlank { "El autor no ha añadido una descripción." }
    Section("Acerca de esta app") {
        Column(Modifier.animateContentSize()) {
            Text(
                desc, style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = if (expanded) Int.MAX_VALUE else 4, overflow = TextOverflow.Ellipsis,
                onTextLayout = { if (!expanded) overflows = it.hasVisualOverflow },
            )
            if (overflows || expanded) {
                val rot by animateFloatAsState(if (expanded) 180f else 0f, label = "rot")
                TextButton(onClick = { expanded = !expanded }, contentPadding = PaddingValues(horizontal = 0.dp)) {
                    Text(if (expanded) "Mostrar menos" else "Leer más")
                    Icon(Icons.Rounded.KeyboardArrowDown, null, Modifier.rotate(rot))
                }
            }
        }
    }
}

@Composable
private fun InfoCard(app: StoreApp) {
    Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surfaceContainerLow) {
        Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
            InfoRow("Paquete", app.pkg)
            InfoRow("Versión", "${app.versionName} (${app.versionCode})")
            InfoRow("Tamaño de descarga", mb(app.size))
            fullDate(app.updatedAt)?.let { InfoRow("Publicada", it) }
        }
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 10.dp)) {
        Text(label, Modifier.weight(1f), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Spacer(Modifier.width(16.dp))
        Text(value, style = MaterialTheme.typography.bodyMedium, textAlign = TextAlign.End, maxLines = 2, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
private fun Section(title: String, padContent: Boolean = true, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxWidth().padding(top = 28.dp)) {
        Text(title, Modifier.padding(horizontal = 24.dp), style = MaterialTheme.typography.titleLarge)
        Spacer(Modifier.height(12.dp))
        Box(if (padContent) Modifier.padding(horizontal = 24.dp) else Modifier) { content() }
    }
}

// Visor a pantalla completa, deslizable entre capturas
@Composable
private fun ShotViewer(vm: StoreViewModel, shots: List<String>, start: Int, onClose: () -> Unit) {
    val pager = rememberPagerState(start) { shots.size }
    Box(Modifier.fillMaxSize().background(Color.Black)) {
        HorizontalPager(pager, Modifier.fillMaxSize().systemBarsPadding()) { i ->
            val bmp = vm.shot(shots[i])
            LaunchedEffect(shots[i], bmp == null) { vm.loadShot(shots[i]) }
            Box(Modifier.fillMaxSize(), Alignment.Center) {
                if (bmp != null) Image(bmp, "Captura ${i + 1}", Modifier.fillMaxSize(), contentScale = ContentScale.Fit)
                else CircularProgressIndicator(color = Color.White)
            }
        }
        IconButton(
            onClose, Modifier.systemBarsPadding().padding(8.dp),
            colors = IconButtonDefaults.iconButtonColors(containerColor = Color.White.copy(alpha = 0.15f), contentColor = Color.White),
        ) { Icon(Icons.Rounded.Close, "Cerrar") }
        Surface(
            Modifier.align(Alignment.BottomCenter).systemBarsPadding().padding(16.dp),
            shape = CircleShape, color = Color.White.copy(alpha = 0.15f), contentColor = Color.White,
        ) { Text("${pager.currentPage + 1} / ${shots.size}", Modifier.padding(horizontal = 14.dp, vertical = 6.dp), style = MaterialTheme.typography.labelLarge) }
    }
}

// Color dominante del icono: el tono con más peso (saturación × brillo)
@Composable
private fun dominantColor(img: ImageBitmap?): Color? = remember(img) { img?.let { dominant(it.asAndroidBitmap()) } }

private fun dominant(src: Bitmap): Color? {
    val b = Bitmap.createScaledBitmap(src, 24, 24, true)
    val hsv = FloatArray(3)
    val weight = DoubleArray(12)
    val sums = Array(12) { DoubleArray(3) }
    for (y in 0 until 24) for (x in 0 until 24) {
        val p = b.getPixel(x, y)
        if (android.graphics.Color.alpha(p) < 128) continue
        android.graphics.Color.colorToHSV(p, hsv)
        val w = (hsv[1] * hsv[2]).toDouble()
        if (w < 0.08) continue
        val k = (hsv[0] / 30f).toInt().coerceIn(0, 11)
        weight[k] += w
        sums[k][0] += android.graphics.Color.red(p) * w
        sums[k][1] += android.graphics.Color.green(p) * w
        sums[k][2] += android.graphics.Color.blue(p) * w
    }
    val k = weight.indices.maxBy { weight[it] }
    if (weight[k] < 1.0) return null // icono casi gris: se usa el color del tema
    return Color((sums[k][0] / weight[k] / 255).toFloat(), (sums[k][1] / weight[k] / 255).toFloat(), (sums[k][2] / weight[k] / 255).toFloat())
}

private fun millis(iso: String): Long? = try { Instant.parse(iso).toEpochMilli() } catch (e: Exception) { null }

// "hace 3 h", siempre en español como el resto de la interfaz
private fun relative(iso: String): String? = millis(iso)?.let {
    val m = (System.currentTimeMillis() - it) / 60_000
    when {
        m < 1 -> "ahora"
        m < 60 -> "hace $m min"
        m < 1440 -> "hace ${m / 60} h"
        m < 2880 -> "ayer"
        m < 43200 -> "hace ${m / 1440} días"
        m < 525600 -> "hace ${m / 43200} meses"
        else -> "hace ${m / 525600} años"
    }
}

private fun fullDate(iso: String): String? = millis(iso)?.let {
    DateTimeFormatter.ofLocalizedDate(FormatStyle.LONG).withLocale(Es).format(Instant.ofEpochMilli(it).atZone(ZoneId.systemDefault()))
}
