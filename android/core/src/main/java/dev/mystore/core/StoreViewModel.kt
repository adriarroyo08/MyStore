package dev.mystore.core

import android.app.Application
import android.graphics.BitmapFactory
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch

sealed interface Screen {
    data object Setup : Screen
    data object List : Screen
    data class Detail(val id: String) : Screen
    data object Settings : Screen
}

// Lógica común de las apps tienda; `type` es "phone" o "watch"
open class StoreViewModel(app: Application, private val type: String) : AndroidViewModel(app) {
    val installer = Installer(app)
    private val settings = StoreSettings(app)
    private var client: StoreClient? = null

    var screen by mutableStateOf<Screen>(Screen.Setup)
    var apps by mutableStateOf<List<StoreApp>>(emptyList())
    var loading by mutableStateOf(false)
    var error by mutableStateOf<String?>(null)
    var query by mutableStateOf("")
    var installed by mutableStateOf<Map<String, Long?>>(emptyMap())
        private set
    var installStates by mutableStateOf<Map<String, InstallState>>(emptyMap())
        private set
    val icons = mutableStateMapOf<String, ImageBitmap?>()

    val storeUrl get() = settings.storeUrl.orEmpty()

    init {
        viewModelScope.launch {
            combine(installer.states, installer.ticks) { s, _ -> s }.collect {
                installStates = it
                refreshInstalled()
            }
        }
        settings.storeUrl?.let { client = StoreClient(it); screen = Screen.List; load() }
    }

    private fun refreshInstalled() {
        installed = apps.associate { a -> a.id to installer.installedVersion(a.pkg) }
    }

    fun load() {
        val c = client ?: return
        viewModelScope.launch {
            loading = true
            try {
                apps = c.apps(type)
                error = null
                refreshInstalled()
            } catch (e: StoreException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    // Prueba la tienda antes de guardar la URL; devuelve el error o null
    suspend fun connect(input: String): String? {
        val url = normalizeStoreUrl(input) ?: return "Escribe una URL válida (http:// o https://)"
        val c = StoreClient(url)
        return try {
            apps = c.apps(type)
            settings.storeUrl = url
            client = c
            icons.clear()
            shots.clear()
            error = null
            refreshInstalled()
            screen = Screen.List
            null
        } catch (e: StoreException) {
            e.message
        }
    }

    fun loadIcon(app: StoreApp) {
        val key = "${app.id}:${app.versionCode}"
        if (icons.containsKey(key)) return
        icons[key] = null
        val c = client ?: return
        viewModelScope.launch {
            try {
                val b = c.icon(app)
                icons[key] = BitmapFactory.decodeByteArray(b, 0, b.size)?.asImageBitmap()
            } catch (e: Exception) { /* se queda el icono genérico */ }
        }
    }

    // Capturas: caché en memoria, reducidas con inSampleSize (lado mayor <= 1600 px)
    val shots = mutableStateMapOf<String, ImageBitmap?>()

    fun loadShot(url: String) {
        if (shots.containsKey(url)) return
        shots[url] = null
        val c = client ?: return
        viewModelScope.launch {
            try {
                val b = c.bytes(url)
                val o = BitmapFactory.Options().apply { inJustDecodeBounds = true }
                BitmapFactory.decodeByteArray(b, 0, b.size, o)
                var s = 1
                while (maxOf(o.outWidth, o.outHeight) / (s * 2) >= 1600) s *= 2
                val opts = BitmapFactory.Options().apply { inSampleSize = s }
                shots[url] = BitmapFactory.decodeByteArray(b, 0, b.size, opts)?.asImageBitmap()
            } catch (e: Exception) { /* se queda el hueco gris */ }
        }
    }

    fun act(app: StoreApp, onPermission: () -> Unit) {
        when (actionFor(app.versionCode, installed[app.id])) {
            AppAction.OPEN -> installer.launch(app.pkg)
            else -> if (!installer.canInstall()) onPermission()
            else viewModelScope.launch { installer.install(app) }
        }
    }
}
