package dev.mystore.mobile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import dev.mystore.core.AppAction
import dev.mystore.core.Screen
import dev.mystore.core.StoreViewModel
import dev.mystore.core.actionFor
import dev.mystore.core.normalizeStoreUrl
import kotlinx.coroutines.launch

// Primera vez: bienvenida y URL de la tienda
@Composable
fun SetupScreen(vm: StoreViewModel) {
    val cs = MaterialTheme.colorScheme
    Box(
        Modifier.fillMaxSize().background(Brush.verticalGradient(0f to cs.primaryContainer, 0.55f to cs.surface)),
    ) {
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).systemBarsPadding().imePadding()
                .padding(horizontal = 24.dp, vertical = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(40.dp))
            Surface(shape = androidx.compose.foundation.shape.RoundedCornerShape(32.dp), shadowElevation = 12.dp) { BrandMark(104.dp) }
            Spacer(Modifier.height(28.dp))
            Text("MyStore", style = MaterialTheme.typography.displaySmall, color = cs.onSurface)
            Spacer(Modifier.height(10.dp))
            Text(
                "Tus propias apps, siempre al día. Instala y actualiza las APK de tu servidor con un toque.",
                style = MaterialTheme.typography.bodyLarge, color = cs.onSurfaceVariant, textAlign = TextAlign.Center,
                modifier = Modifier.widthIn(max = 340.dp),
            )
            Spacer(Modifier.height(40.dp))
            Surface(shape = MaterialTheme.shapes.extraLarge, color = cs.surfaceContainerLow, modifier = Modifier.widthIn(max = 480.dp)) {
                Column(Modifier.padding(24.dp)) {
                    Text("Conecta tu tienda", style = MaterialTheme.typography.titleLarge)
                    Spacer(Modifier.height(4.dp))
                    Text(
                        "Escribe la dirección de tu servidor MyStore.",
                        style = MaterialTheme.typography.bodyMedium, color = cs.onSurfaceVariant,
                    )
                    Spacer(Modifier.height(20.dp))
                    UrlForm(vm, initial = vm.storeUrl, button = "Conectar")
                }
            }
            Spacer(Modifier.height(16.dp))
            Text("Podrás cambiarla después en Ajustes.", style = MaterialTheme.typography.bodySmall, color = cs.outline)
        }
    }
}

// Campo de URL con validación en vivo y botón que prueba la conexión
@Composable
private fun UrlForm(vm: StoreViewModel, initial: String, button: String, onlyIfChanged: Boolean = false) {
    var url by remember { mutableStateOf(initial) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val haptic = LocalHapticFeedback.current
    val invalid = url.isNotBlank() && normalizeStoreUrl(url) == null
    // En Ajustes no tiene sentido reconectar a la misma URL
    val canSubmit = url.isNotBlank() && !invalid && !busy && !(onlyIfChanged && normalizeStoreUrl(url) == initial)
    val submit = {
        if (canSubmit) scope.launch {
            busy = true
            error = vm.connect(url)
            busy = false
            haptic.performHapticFeedback(if (error == null) HapticFeedbackType.Confirm else HapticFeedbackType.Reject)
        }
        Unit
    }
    OutlinedTextField(
        url, { url = it; error = null }, Modifier.fillMaxWidth(),
        singleLine = true, enabled = !busy,
        label = { Text("URL de la tienda") },
        placeholder = { Text("tienda.ejemplo.com") },
        trailingIcon = if (url.isNotEmpty() && !busy) {{
            IconButton(onClick = { url = ""; error = null }) { Icon(Icons.Rounded.Close, "Borrar") }
        }} else null,
        isError = invalid || error != null,
        supportingText = {
            Text(
                error ?: if (invalid) "Escribe una dirección válida, por ejemplo https://tienda.ejemplo.com"
                else "Si no pones http:// se usará https://",
            )
        },
        shape = MaterialTheme.shapes.medium,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Go, autoCorrectEnabled = false),
        keyboardActions = KeyboardActions(onGo = { submit() }),
    )
    Spacer(Modifier.height(12.dp))
    Button(onClick = submit, Modifier.fillMaxWidth().height(56.dp), enabled = canSubmit) {
        if (busy) {
            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = LocalContentColor.current)
            Spacer(Modifier.width(12.dp))
            Text("Conectando…", style = MaterialTheme.typography.titleSmall)
        } else Text(button, style = MaterialTheme.typography.titleSmall)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(vm: StoreViewModel) {
    val cs = MaterialTheme.colorScheme
    val scroll = TopAppBarDefaults.exitUntilCollapsedScrollBehavior()
    val ctx = LocalContext.current
    val version = remember { runCatching { ctx.packageManager.getPackageInfo(ctx.packageName, 0).versionName }.getOrNull() ?: "—" }
    val installed = vm.apps.count { actionFor(it.versionCode, vm.installed[it.id]) != AppAction.INSTALL }
    Scaffold(
        Modifier.nestedScroll(scroll.nestedScrollConnection),
        topBar = {
            LargeTopAppBar(
                title = { Text("Ajustes") },
                navigationIcon = { BackButton { vm.screen = Screen.List } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = cs.surface, scrolledContainerColor = cs.surfaceContainer),
                scrollBehavior = scroll,
            )
        },
    ) { pad ->
        Column(
            Modifier.fillMaxSize().padding(pad).verticalScroll(rememberScrollState()).imePadding().padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Group("Tienda") {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(Modifier.size(40.dp).clip(CircleShape).background(cs.primaryContainer), Alignment.Center) {
                        Icon(Icons.Rounded.CheckCircle, null, tint = cs.onPrimaryContainer)
                    }
                    Spacer(Modifier.width(16.dp))
                    Column(Modifier.weight(1f)) {
                        Text(if (vm.error == null) "Conectada" else "Sin conexión", style = MaterialTheme.typography.titleMedium)
                        Text(vm.storeUrl, style = MaterialTheme.typography.bodyMedium, color = cs.onSurfaceVariant)
                    }
                }
                Spacer(Modifier.height(20.dp))
                UrlForm(vm, initial = vm.storeUrl, button = "Guardar y conectar", onlyIfChanged = true)
            }
            Group("Resumen") {
                Pair("Apps en la tienda", "${vm.apps.size}").let { (k, v) -> KeyValue(k, v) }
                KeyValue("Instaladas", "$installed")
                KeyValue("Versión de MyStore", version)
            }
            Group("Almacenamiento") {
                // Se relee al borrar: installed cambia con installer.ticks
                val bytes = remember(vm.installed) { vm.installer.cacheBytes() }
                KeyValue("APKs guardadas", mb(bytes))
                Text(
                    "Se guarda la última descarga de cada app para reinstalarla sin volver a bajarla.",
                    style = MaterialTheme.typography.bodyMedium, color = cs.onSurfaceVariant,
                )
                Spacer(Modifier.height(12.dp))
                OutlinedButton(onClick = { vm.installer.clearCache() }, enabled = bytes > 0) { Text("Borrar APKs guardadas") }
            }
            Spacer(Modifier.navigationBarsPadding().height(16.dp))
        }
    }
}

@Composable
private fun Group(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column {
        Text(
            title, Modifier.padding(start = 8.dp, top = 8.dp, bottom = 8.dp),
            style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.primary,
        )
        Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surfaceContainerLow) {
            Column(Modifier.fillMaxWidth().padding(20.dp), content = content)
        }
    }
}

@Composable
private fun KeyValue(k: String, v: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        Text(k, Modifier.weight(1f), style = MaterialTheme.typography.bodyLarge)
        Text(v, style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
