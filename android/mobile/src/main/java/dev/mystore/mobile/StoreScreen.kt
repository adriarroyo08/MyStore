package dev.mystore.mobile

import androidx.activity.compose.BackHandler
import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.rounded.*
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import dev.mystore.core.*
import kotlinx.coroutines.launch
import java.util.Locale

// Ámbitos para el elemento compartido (icono lista → detalle)
@OptIn(ExperimentalSharedTransitionApi::class)
val LocalShared = staticCompositionLocalOf<SharedTransitionScope?> { null }
val LocalAnim = staticCompositionLocalOf<AnimatedVisibilityScope?> { null }

enum class Filter { ALL, INSTALLED, UPDATES }

@OptIn(ExperimentalSharedTransitionApi::class, ExperimentalMaterial3Api::class)
@Composable
fun StoreScreen(vm: PhoneViewModel, onPermission: () -> Unit) {
    // Estado de la lista elevado para conservar scroll y filtro al volver del detalle
    val listState = rememberLazyListState()
    val barState = rememberTopAppBarState()
    val snackbar = remember { SnackbarHostState() }
    InstallFeedback(vm, snackbar)
    BackHandler(vm.screen != Screen.List && vm.screen != Screen.Setup) { vm.screen = Screen.List }

    Box(Modifier.fillMaxSize()) {
        SharedTransitionLayout {
            AnimatedContent(
                vm.screen,
                transitionSpec = {
                    val back = targetState == Screen.List
                    (fadeIn(tween(260, 60)) + slideInHorizontally(tween(320)) { if (back) -it / 10 else it / 10 }) togetherWith
                        fadeOut(tween(140))
                },
                label = "screen",
            ) { s ->
                CompositionLocalProvider(LocalShared provides this@SharedTransitionLayout, LocalAnim provides this) {
                    when (s) {
                        Screen.Setup -> SetupScreen(vm)
                        Screen.Settings -> SettingsScreen(vm)
                        Screen.List -> HomeScreen(vm, listState, barState, vm.filter, { vm.filter = it }, onPermission)
                        is Screen.Detail -> {
                            val app = vm.apps.find { it.id == s.id }
                            if (app != null) DetailScreen(vm, app, onPermission)
                            else if (!vm.loading) LaunchedEffect(Unit) { vm.screen = Screen.List }
                        }
                    }
                }
            }
        }
        SnackbarHost(snackbar, Modifier.align(Alignment.BottomCenter).navigationBarsPadding().imePadding())
    }
}

// Avisa con snackbar y vibración cuando una instalación termina o falla
@Composable
private fun InstallFeedback(vm: StoreViewModel, host: SnackbarHostState) {
    val haptic = LocalHapticFeedback.current
    val scope = rememberCoroutineScope()
    var prev by remember { mutableStateOf(vm.installStates) }
    LaunchedEffect(vm.installStates) {
        val cur = vm.installStates
        for ((id, s) in cur) {
            val before = prev[id]
            if (before == s) continue
            val app = vm.apps.find { it.id == id } ?: continue
            if (s is InstallState.Failed) {
                haptic.performHapticFeedback(HapticFeedbackType.Reject)
                scope.launch { host.showSnackbar(s.message, withDismissAction = true) }
            } else if (before == InstallState.Installing && s == InstallState.Idle &&
                actionFor(app.versionCode, vm.installed[id]) == AppAction.OPEN
            ) {
                haptic.performHapticFeedback(HapticFeedbackType.Confirm)
                scope.launch { host.showSnackbar("${app.label} se instaló correctamente") }
            }
        }
        prev = cur
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun HomeScreen(
    vm: StoreViewModel, listState: LazyListState, barState: TopAppBarState,
    filter: Filter, onFilter: (Filter) -> Unit, onPermission: () -> Unit,
) {
    val scroll = TopAppBarDefaults.exitUntilCollapsedScrollBehavior(barState)
    val actions = vm.apps.associate { it.id to actionFor(it.versionCode, vm.installed[it.id]) }
    val updates = vm.apps.filter { actions[it.id] == AppAction.UPDATE }
    val base = if (vm.query.isBlank()) vm.apps else vm.apps.search(vm.query)
    val shown = when (filter) {
        Filter.ALL -> base
        Filter.INSTALLED -> base.filter { actions[it.id] != AppAction.INSTALL }
        Filter.UPDATES -> base.filter { actions[it.id] == AppAction.UPDATE }
    }
    Scaffold(
        Modifier.nestedScroll(scroll.nestedScrollConnection),
        topBar = {
            LargeTopAppBar(
                title = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        BrandMark(32.dp)
                        Spacer(Modifier.width(12.dp))
                        Text("MyStore")
                    }
                },
                actions = {
                    IconButton(onClick = { vm.screen = Screen.Settings }) { Icon(Icons.Outlined.Settings, "Ajustes") }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    scrolledContainerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
                scrollBehavior = scroll,
            )
        },
    ) { pad ->
        PullToRefreshBox(
            isRefreshing = vm.loading && vm.apps.isNotEmpty(), onRefresh = vm::load,
            modifier = Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()),
        ) {
            LazyColumn(
                Modifier.fillMaxSize(), listState,
                contentPadding = PaddingValues(bottom = pad.calculateBottomPadding() + 24.dp),
            ) {
                item(key = "search") { SearchField(vm.query) { vm.query = it } }
                item(key = "chips") { FilterChips(filter, onFilter, updates.size) }
                val err = vm.error
                when {
                    vm.apps.isEmpty() && vm.loading -> items(7) { SkeletonRow() }
                    err != null && vm.apps.isEmpty() -> item {
                        Message(Icons.Rounded.Warning, "No se pudo conectar con la tienda", err, "Reintentar", vm::load)
                    }
                    vm.apps.isEmpty() -> item {
                        Message(Icons.Rounded.ShoppingCart, "Aún no hay apps", "Sube una APK de móvil a tu tienda y aparecerá aquí.", "Recargar", vm::load)
                    }
                    else -> {
                        if (err != null) item(key = "err") { ErrorStrip(err, vm::load) }
                        if (updates.isNotEmpty() && vm.query.isBlank() && filter != Filter.INSTALLED) item(key = "updates") {
                            UpdatesCard(vm, updates) { updates.forEach { a -> if (vm.installStates[a.id] !is InstallState.Downloading) vm.act(a, onPermission) } }
                        }
                        if (shown.isEmpty()) item(key = "none") {
                            when {
                                vm.query.isNotBlank() -> Message(Icons.Rounded.Search, "Sin resultados", "Ninguna app coincide con «${vm.query.trim()}».", "Borrar búsqueda") { vm.query = "" }
                                filter == Filter.UPDATES -> Message(Icons.Rounded.CheckCircle, "Todo al día", "Tus apps tienen la última versión.", null, null)
                                else -> Message(Icons.Rounded.Home, "Nada instalado todavía", "Instala una app desde la pestaña Todas.", "Ver todas") { onFilter(Filter.ALL) }
                            }
                        }
                        items(shown, key = { it.id }) { app ->
                            AppRow(vm, app, actions[app.id] ?: AppAction.INSTALL, onPermission, Modifier.animateItem())
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SearchField(query: String, onChange: (String) -> Unit) {
    val focus = LocalFocusManager.current
    TextField(
        query, onChange,
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp).heightIn(min = 56.dp),
        placeholder = { Text("Buscar en tu tienda") },
        leadingIcon = { Icon(Icons.Rounded.Search, null) },
        trailingIcon = {
            AnimatedVisibility(query.isNotEmpty(), enter = fadeIn() + scaleIn(), exit = fadeOut() + scaleOut()) {
                IconButton(onClick = { onChange("") }) { Icon(Icons.Rounded.Close, "Borrar búsqueda") }
            }
        },
        singleLine = true,
        shape = CircleShape,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = androidx.compose.foundation.text.KeyboardActions(onSearch = { focus.clearFocus() }),
        colors = TextFieldDefaults.colors(
            focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
            unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
            focusedIndicatorColor = Color.Transparent, unfocusedIndicatorColor = Color.Transparent,
        ),
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FilterChips(filter: Filter, onFilter: (Filter) -> Unit, updates: Int) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Filter.entries.forEach { f ->
            val selected = f == filter
            FilterChip(
                selected, { onFilter(f) },
                label = {
                    Text(when (f) { Filter.ALL -> "Todas"; Filter.INSTALLED -> "Instaladas"; Filter.UPDATES -> "Actualizaciones" })
                    if (f == Filter.UPDATES && updates > 0) {
                        Spacer(Modifier.width(6.dp))
                        Badge(containerColor = MaterialTheme.colorScheme.primary) { Text("$updates") }
                    }
                },
                leadingIcon = if (selected) {{ Icon(Icons.Rounded.Check, null, Modifier.size(FilterChipDefaults.IconSize)) }} else null,
                shape = CircleShape,
            )
        }
    }
}

@Composable
private fun UpdatesCard(vm: StoreViewModel, updates: List<StoreApp>, onUpdateAll: () -> Unit) {
    val haptic = LocalHapticFeedback.current
    Surface(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.primaryContainer,
    ) {
        Column(Modifier.padding(20.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Iconos solapados de las apps con actualización
                Box(Modifier.width(36.dp + 22.dp * (minOf(updates.size, 3) - 1)).height(36.dp)) {
                    updates.take(3).forEachIndexed { i, a ->
                        Box(
                            Modifier.padding(start = 22.dp * i).size(36.dp).clip(RoundedCornerShape(11.dp))
                                .border(2.dp, MaterialTheme.colorScheme.primaryContainer, RoundedCornerShape(11.dp)),
                        ) { AppIcon(vm, a, 36.dp) }
                    }
                }
                Spacer(Modifier.width(14.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        "Actualizaciones disponibles", style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer,
                    )
                    val n = updates.size
                    Text(
                        (if (n == 1) "1 app" else "$n apps") + " · " + mb(updates.sumOf { it.size }),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.75f),
                    )
                }
            }
            Spacer(Modifier.height(16.dp))
            Button(
                onClick = { haptic.performHapticFeedback(HapticFeedbackType.Confirm); onUpdateAll() },
                Modifier.fillMaxWidth().height(48.dp),
            ) {
                Icon(Icons.Rounded.Refresh, null, Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                Text("Actualizar todo")
            }
        }
    }
}

@Composable
private fun AppRow(vm: StoreViewModel, app: StoreApp, action: AppAction, onPermission: () -> Unit, modifier: Modifier) {
    val s = vm.installStates[app.id]
    Row(
        modifier.fillMaxWidth().clickable { vm.screen = Screen.Detail(app.id) }
            .padding(start = 20.dp, end = 16.dp, top = 10.dp, bottom = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AppIcon(vm, app, 56.dp, shared = true)
        Spacer(Modifier.width(16.dp))
        Column(Modifier.weight(1f)) {
            Text(app.label, maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.titleMedium)
            if (app.summary.isNotEmpty()) Text(
                app.summary, maxLines = 1, overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(2.dp))
            if (s is InstallState.Failed) Text(
                "Error al instalar", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.error,
            ) else Text(
                when (action) {
                    AppAction.INSTALL -> "${mb(app.size)}  ·  v${app.versionName}"
                    AppAction.UPDATE -> "Nueva versión ${app.versionName}"
                    AppAction.OPEN -> "Instalada  ·  v${app.versionName}"
                },
                maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.labelMedium,
                color = if (action == AppAction.UPDATE) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline,
            )
        }
        Spacer(Modifier.width(12.dp))
        ActionPill(vm, app, action, onPermission)
    }
}

// Botón compacto que se transforma en anillo de progreso al descargar
@Composable
private fun ActionPill(vm: StoreViewModel, app: StoreApp, action: AppAction, onPermission: () -> Unit) {
    val s = vm.installStates[app.id]
    val haptic = LocalHapticFeedback.current
    val busy = s is InstallState.Downloading || s is InstallState.Installing
    AnimatedContent(
        busy, contentAlignment = Alignment.Center,
        transitionSpec = { (fadeIn(tween(200)) + scaleIn(initialScale = 0.8f)) togetherWith fadeOut(tween(120)) using SizeTransform(clip = false) },
        label = "pill",
    ) { b ->
        if (b) {
            Box(
                Modifier.size(44.dp).clip(CircleShape).clickable(enabled = s is InstallState.Installing) { vm.act(app, onPermission) },
                contentAlignment = Alignment.Center,
            ) {
                val st = vm.installStates[app.id]
                if (st is InstallState.Downloading) {
                    val p by animateFloatAsState(st.progress, label = "p")
                    CircularProgressIndicator(
                        progress = { p }, Modifier.fillMaxSize(), strokeWidth = 3.dp,
                        trackColor = MaterialTheme.colorScheme.secondaryContainer, gapSize = 0.dp,
                    )
                    Text("${(st.progress * 100).toInt()}%", style = MaterialTheme.typography.labelSmall)
                } else CircularProgressIndicator(Modifier.fillMaxSize(), strokeWidth = 3.dp)
            }
        } else {
            val onClick = { haptic.performHapticFeedback(HapticFeedbackType.ContextClick); vm.act(app, onPermission) }
            val pad = PaddingValues(horizontal = 18.dp)
            when (action) {
                AppAction.UPDATE -> Button(onClick, Modifier.height(40.dp), contentPadding = pad) { Text("Actualizar") }
                AppAction.INSTALL -> FilledTonalButton(onClick, Modifier.height(40.dp), contentPadding = pad) { Text("Instalar") }
                AppAction.OPEN -> OutlinedButton(onClick, Modifier.height(40.dp), contentPadding = pad) { Text("Abrir") }
            }
        }
    }
}

@Composable
private fun ErrorStrip(err: String, onRetry: () -> Unit) {
    Surface(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        shape = MaterialTheme.shapes.medium, color = MaterialTheme.colorScheme.errorContainer,
    ) {
        Row(Modifier.padding(start = 16.dp, end = 8.dp, top = 6.dp, bottom = 6.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Rounded.Warning, null, tint = MaterialTheme.colorScheme.onErrorContainer)
            Spacer(Modifier.width(12.dp))
            Text(err, Modifier.weight(1f), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onErrorContainer)
            TextButton(onClick = onRetry) { Text("Reintentar") }
        }
    }
}

// Estado vacío o de error: icono, título, explicación y acción
@Composable
fun Message(icon: ImageVector, title: String, body: String, action: String?, onAction: (() -> Unit)?) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 40.dp, vertical = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(88.dp).clip(CircleShape).background(MaterialTheme.colorScheme.secondaryContainer),
            contentAlignment = Alignment.Center,
        ) { Icon(icon, null, Modifier.size(40.dp), tint = MaterialTheme.colorScheme.onSecondaryContainer) }
        Spacer(Modifier.height(20.dp))
        Text(title, style = MaterialTheme.typography.titleLarge, textAlign = TextAlign.Center)
        Spacer(Modifier.height(8.dp))
        Text(body, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center)
        if (action != null && onAction != null) {
            Spacer(Modifier.height(24.dp))
            FilledTonalButton(onClick = onAction) { Text(action) }
        }
    }
}

@Composable
fun shimmer(): Brush {
    val t = rememberInfiniteTransition(label = "shimmer")
    val x by t.animateFloat(-400f, 1400f, infiniteRepeatable(tween(1300, easing = LinearEasing)), label = "x")
    val c = MaterialTheme.colorScheme.surfaceContainerHighest
    val hi = MaterialTheme.colorScheme.surfaceContainerLow
    return Brush.linearGradient(listOf(c, hi, c), start = androidx.compose.ui.geometry.Offset(x, 0f), end = androidx.compose.ui.geometry.Offset(x + 400f, 200f))
}

@Composable
private fun SkeletonRow() {
    val b = shimmer()
    Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(56.dp).clip(RoundedCornerShape(16.dp)).background(b))
        Spacer(Modifier.width(16.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Box(Modifier.fillMaxWidth(0.45f).height(14.dp).clip(CircleShape).background(b))
            Box(Modifier.fillMaxWidth(0.8f).height(12.dp).clip(CircleShape).background(b))
        }
        Spacer(Modifier.width(16.dp))
        Box(Modifier.width(84.dp).height(40.dp).clip(CircleShape).background(b))
    }
}

// Logo: bolsa con estrella del icono del lanzador
@Composable
fun BrandMark(size: Dp) {
    Box(Modifier.size(size).clip(RoundedCornerShape(size * 0.3f)).background(BrandBlue)) {
        // El primer plano adaptativo trae margen de seguridad: se amplía para llenar el hueco
        Image(painterResource(R.drawable.ic_launcher_foreground), null, Modifier.fillMaxSize().graphicsLayer { scaleX = 1.45f; scaleY = 1.45f })
    }
}

@OptIn(ExperimentalSharedTransitionApi::class)
@Composable
fun AppIcon(vm: StoreViewModel, app: StoreApp, size: Dp, shared: Boolean = false) {
    LaunchedEffect(app.id, app.versionCode) { vm.loadIcon(app) }
    val bmp = vm.icons["${app.id}:${app.versionCode}"]
    val shape = RoundedCornerShape(size * 0.28f)
    var m: Modifier = Modifier
    val st = LocalShared.current
    val av = LocalAnim.current
    if (shared && st != null && av != null) with(st) {
        m = Modifier.sharedElement(rememberSharedContentState("icon-${app.id}"), av)
    }
    Box(
        m.size(size).clip(shape).background(MaterialTheme.colorScheme.surfaceContainerHighest),
        contentAlignment = Alignment.Center,
    ) {
        if (bmp != null) Image(bmp, app.label, Modifier.fillMaxSize())
        else Text(
            app.label.take(1).uppercase(), style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
fun BackButton(onBack: () -> Unit) {
    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Atrás") }
}

val Es: Locale = Locale.forLanguageTag("es-ES")

fun mb(bytes: Long): String = String.format(Es, "%.1f MB", bytes / 1048576.0)
