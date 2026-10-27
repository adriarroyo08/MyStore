package dev.mystore.mobile

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import dev.mystore.core.*
import kotlinx.coroutines.launch
import java.util.Locale

@Composable
fun StoreScreen(vm: StoreViewModel, onPermission: () -> Unit) {
    BackHandler(vm.screen != Screen.List && vm.screen != Screen.Setup) { vm.screen = Screen.List }
    when (val s = vm.screen) {
        Screen.Setup -> SetupScreen(vm, title = "Conectar con tu tienda", onBack = null)
        Screen.Settings -> SetupScreen(vm, title = "Ajustes", onBack = { vm.screen = Screen.List })
        Screen.List -> ListScreen(vm, onPermission)
        is Screen.Detail -> {
            val app = vm.apps.find { it.id == s.id }
            if (app != null) DetailScreen(vm, app, onPermission) else vm.screen = Screen.List
        }
    }
}

@Composable
private fun SetupScreen(vm: StoreViewModel, title: String, onBack: (() -> Unit)?) {
    var url by remember { mutableStateOf(vm.storeUrl) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    Column(Modifier.fillMaxSize().systemBarsPadding().padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(title, style = MaterialTheme.typography.headlineSmall)
        OutlinedTextField(
            url, { url = it }, Modifier.fillMaxWidth(), singleLine = true,
            label = { Text("URL de la tienda") }, placeholder = { Text("http://192.168.1.10:8080") },
            isError = error != null, supportingText = error?.let { { Text(it) } },
        )
        Button(
            onClick = { scope.launch { busy = true; error = vm.connect(url); busy = false } },
            enabled = !busy,
        ) { Text(if (busy) "Conectando…" else "Conectar") }
        if (onBack != null) TextButton(onClick = onBack) { Text("Cancelar") }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ListScreen(vm: StoreViewModel, onPermission: () -> Unit) {
    val shown = if (vm.query.isBlank()) vm.apps else vm.apps.search(vm.query)
    Column(Modifier.fillMaxSize().systemBarsPadding()) {
        Row(Modifier.fillMaxWidth().padding(start = 20.dp, end = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("MyStore", Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
            TextButton(onClick = { vm.screen = Screen.Settings }) { Text("Ajustes") }
        }
        OutlinedTextField(
            vm.query, { vm.query = it }, Modifier.fillMaxWidth().padding(horizontal = 16.dp),
            singleLine = true, placeholder = { Text("Buscar apps") },
        )
        PullToRefreshBox(isRefreshing = vm.loading, onRefresh = vm::load, modifier = Modifier.weight(1f)) {
            val err = vm.error
            when {
                err != null && vm.apps.isEmpty() -> Centered {
                    Text(err, style = MaterialTheme.typography.bodyLarge)
                    Spacer(Modifier.height(12.dp))
                    Button(onClick = vm::load) { Text("Reintentar") }
                }
                vm.apps.isEmpty() && !vm.loading -> Centered { Text("Tu tienda aún no tiene apps para el móvil") }
                else -> LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(vertical = 8.dp)) {
                    if (err != null) item {
                        Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                            Text(err, Modifier.weight(1f), color = MaterialTheme.colorScheme.error)
                            TextButton(onClick = vm::load) { Text("Reintentar") }
                        }
                    }
                    items(shown, key = { it.id }) { app ->
                        Row(
                            Modifier.fillMaxWidth().clickable { vm.screen = Screen.Detail(app.id) }
                                .padding(horizontal = 16.dp, vertical = 8.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            AppIcon(vm, app, 48.dp)
                            Spacer(Modifier.width(12.dp))
                            Column(Modifier.weight(1f)) {
                                Text(app.label, maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.titleMedium)
                                Text(meta(app), style = MaterialTheme.typography.bodySmall)
                                ProgressOrError(vm, app)
                            }
                            Spacer(Modifier.width(8.dp))
                            ActionButton(vm, app, onPermission)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun DetailScreen(vm: StoreViewModel, app: StoreApp, onPermission: () -> Unit) {
    Column(Modifier.fillMaxSize().systemBarsPadding().verticalScroll(rememberScrollState()).padding(24.dp)) {
        TextButton(onClick = { vm.screen = Screen.List }) { Text("← Volver") }
        AppIcon(vm, app, 96.dp)
        Spacer(Modifier.height(12.dp))
        Text(app.label, style = MaterialTheme.typography.headlineSmall)
        Text(meta(app), style = MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(12.dp))
        ActionButton(vm, app, onPermission)
        ProgressOrError(vm, app)
        if (app.description.isNotBlank()) {
            Spacer(Modifier.height(20.dp))
            Text(app.description)
        }
        if (app.notes.isNotBlank()) {
            Spacer(Modifier.height(20.dp))
            Text("Novedades", style = MaterialTheme.typography.titleMedium)
            Text(app.notes)
        }
    }
}

// Con scroll para que tirar de la pantalla vacía siga recargando
@Composable
private fun Centered(content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(32.dp),
        verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally, content = content,
    )
}

@Composable
private fun AppIcon(vm: StoreViewModel, app: StoreApp, size: Dp) {
    LaunchedEffect(app.id, app.versionCode) { vm.loadIcon(app) }
    val bmp = vm.icons["${app.id}:${app.versionCode}"]
    val shape = RoundedCornerShape(size / 4)
    if (bmp != null) Image(bmp, null, Modifier.size(size).clip(shape))
    else Box(Modifier.size(size).clip(shape).background(MaterialTheme.colorScheme.secondaryContainer), Alignment.Center) {
        Text(app.label.take(1).uppercase(), style = MaterialTheme.typography.titleMedium)
    }
}

@Composable
private fun ProgressOrError(vm: StoreViewModel, app: StoreApp) {
    when (val s = vm.installStates[app.id]) {
        is InstallState.Downloading -> LinearProgressIndicator(progress = { s.progress }, Modifier.fillMaxWidth().padding(top = 4.dp))
        InstallState.Installing -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(top = 4.dp))
        is InstallState.Failed -> Text(s.message, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        else -> {}
    }
}

@Composable
private fun ActionButton(vm: StoreViewModel, app: StoreApp, onPermission: () -> Unit) {
    val s = vm.installStates[app.id]
    val busy = s is InstallState.Downloading || s is InstallState.Installing
    val text = when {
        s is InstallState.Downloading -> "Descargando"
        s is InstallState.Installing -> "Instalando…"
        else -> when (actionFor(app.versionCode, vm.installed[app.id])) {
            AppAction.INSTALL -> "Instalar"
            AppAction.UPDATE -> "Actualizar"
            AppAction.OPEN -> "Abrir"
        }
    }
    Button(onClick = { vm.act(app, onPermission) }, enabled = !busy) { Text(text) }
}

private fun meta(app: StoreApp) = "v${app.versionName} · " + String.format(Locale.getDefault(), "%.1f MB", app.size / 1048576.0)
