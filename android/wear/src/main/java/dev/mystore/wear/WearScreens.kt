package dev.mystore.wear

import android.text.InputType
import android.view.inputmethod.EditorInfo
import android.widget.EditText
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.widget.doAfterTextChanged

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.items
import androidx.wear.compose.foundation.lazy.rememberScalingLazyListState
import androidx.wear.compose.material.*
import dev.mystore.core.*
import kotlinx.coroutines.launch
import java.util.Locale

private val Green = Color(0xFF4CAF50)

@Composable
fun WearApp(vm: StoreViewModel, onPermission: () -> Unit) {
    BackHandler(vm.screen != Screen.List && vm.screen != Screen.Setup) { vm.screen = Screen.List }
    when (val s = vm.screen) {
        Screen.Setup -> SetupScreen(vm, onBack = null)
        Screen.Settings -> SetupScreen(vm, onBack = { vm.screen = Screen.List })
        Screen.List -> ListScreen(vm)
        is Screen.Detail -> {
            val app = vm.apps.find { it.id == s.id }
            if (app != null) DetailScreen(vm, app, onPermission) else vm.screen = Screen.List
        }
    }
}

@Composable
private fun SetupScreen(vm: StoreViewModel, onBack: (() -> Unit)?) {
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    // EditText clásico: el RemoteInput de Mobvoi cierra la tarea sin devolver el texto y el teclado
    // de Wear no confirma el texto en un BasicTextField de Compose.
    var url by remember { mutableStateOf(vm.storeUrl.ifEmpty { "https://" }) }
    val connect = { scope.launch { busy = true; error = vm.connect(url); busy = false } }
    ScalingLazyColumn(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally) {
        item { ListHeader { Text(if (onBack == null) "Conectar" else "Ajustes") } }
        item {
            AndroidView(
                factory = { ctx ->
                    EditText(ctx).apply {
                        setText(url)
                        isSingleLine = true
                        inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI
                        imeOptions = EditorInfo.IME_ACTION_GO
                        setTextColor(android.graphics.Color.WHITE)
                        textSize = 14f
                        doAfterTextChanged { url = it?.toString().orEmpty() }
                        setOnEditorActionListener { _, _, _ -> connect(); true }
                    }
                },
                modifier = Modifier.fillMaxWidth().clip(CircleShape).background(Color(0xFF303030)).padding(horizontal = 10.dp),
            )
        }
        item {
            Chip(
                onClick = { connect() },
                label = { Text(if (busy) "Conectando…" else "Conectar") },
                enabled = !busy && url.isNotBlank(), modifier = Modifier.fillMaxWidth(),
            )
        }
        error?.let { item { Text(it, color = MaterialTheme.colors.error, textAlign = TextAlign.Center, style = MaterialTheme.typography.caption2) } }
        if (onBack != null) item {
            Chip(onClick = onBack, label = { Text("Cancelar") }, colors = ChipDefaults.secondaryChipColors(), modifier = Modifier.fillMaxWidth())
        }
    }
}

@Composable
private fun ListScreen(vm: StoreViewModel) {
    val state = rememberScalingLazyListState()
    val err = vm.error
    Scaffold(timeText = { TimeText() }, positionIndicator = { PositionIndicator(state) }) {
        ScalingLazyColumn(Modifier.fillMaxSize(), state = state, horizontalAlignment = Alignment.CenterHorizontally) {
            item { ListHeader { Text("MyStore") } }
            when {
                err != null && vm.apps.isEmpty() -> {
                    item { Text(err, textAlign = TextAlign.Center, style = MaterialTheme.typography.body2) }
                    item { Chip(onClick = vm::load, label = { Text("Reintentar") }, modifier = Modifier.fillMaxWidth()) }
                }
                vm.loading && vm.apps.isEmpty() -> item { Text("Cargando…") }
                vm.apps.isEmpty() -> item { Text("No hay apps para el reloj", textAlign = TextAlign.Center) }
                else -> items(vm.apps, key = { it.id }) { app ->
                    val update = actionFor(app.versionCode, vm.installed[app.id]) == AppAction.UPDATE
                    Chip(
                        onClick = { vm.screen = Screen.Detail(app.id) },
                        label = { Text(app.label, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                        icon = { AppIcon(vm, app, 32.dp) },
                        secondaryLabel = if (update) ({ Text("● Actualizar", color = Green) }) else null,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            }
            if (err != null && vm.apps.isNotEmpty()) item {
                Chip(onClick = vm::load, label = { Text("Reintentar") }, secondaryLabel = { Text(err, maxLines = 2) }, modifier = Modifier.fillMaxWidth())
            }
            item {
                Chip(
                    onClick = { vm.screen = Screen.Settings }, label = { Text("Ajustes") },
                    colors = ChipDefaults.secondaryChipColors(), modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}

@Composable
private fun DetailScreen(vm: StoreViewModel, app: StoreApp, onPermission: () -> Unit) {
    val s = vm.installStates[app.id]
    val busy = s is InstallState.Downloading || s is InstallState.Installing
    val label = when (actionFor(app.versionCode, vm.installed[app.id])) {
        AppAction.INSTALL -> "Instalar"
        AppAction.UPDATE -> "Actualizar"
        AppAction.OPEN -> "Abrir"
    }
    ScalingLazyColumn(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally) {
        item { AppIcon(vm, app, 48.dp) }
        item { Text(app.label, style = MaterialTheme.typography.title3, textAlign = TextAlign.Center) }
        item {
            Text(
                "v${app.versionName} · " + String.format(Locale.getDefault(), "%.1f MB", app.size / 1048576.0),
                style = MaterialTheme.typography.caption2,
            )
        }
        item {
            // Progreso circular alrededor del botón durante la descarga
            Box(Modifier.size(72.dp), Alignment.Center) {
                when (s) {
                    is InstallState.Downloading -> CircularProgressIndicator(progress = s.progress, Modifier.fillMaxSize())
                    InstallState.Installing -> CircularProgressIndicator(Modifier.fillMaxSize())
                    else -> {}
                }
                Button(onClick = { vm.act(app, onPermission) }, enabled = !busy, modifier = Modifier.size(60.dp)) {
                    Text(if (busy) "…" else label, style = MaterialTheme.typography.caption2, maxLines = 1)
                }
            }
        }
        if (s is InstallState.Failed) item {
            Text(s.message, color = MaterialTheme.colors.error, textAlign = TextAlign.Center, style = MaterialTheme.typography.caption2)
        }
        if (app.description.isNotBlank()) item { Text(app.description, style = MaterialTheme.typography.body2, textAlign = TextAlign.Center) }
    }
}

@Composable
private fun AppIcon(vm: StoreViewModel, app: StoreApp, size: Dp) {
    LaunchedEffect(app.id, app.versionCode) { vm.loadIcon(app) }
    val bmp = vm.icons["${app.id}:${app.versionCode}"]
    val m = Modifier.size(size).clip(CircleShape)
    if (bmp != null) Image(bmp, null, m)
    else Box(m.background(MaterialTheme.colors.surface), Alignment.Center) { Text(app.label.take(1).uppercase()) }
}
