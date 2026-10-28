package dev.mystore.core

import android.app.Application
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.util.LruCache
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

// Mayor potencia de 2 que deja el lado mayor decodificado <= target
fun sampleSize(width: Int, height: Int, target: Int): Int {
    val max = maxOf(width, height)
    var s = 1
    while (max / s > target) s *= 2
    return s
}

sealed interface Screen {
    data object Setup : Screen
    data object List : Screen
    data class Detail(val id: String) : Screen
    data object Settings : Screen
}

// Lógica común de las apps tienda; `type` es "phone" o "watch"
open class StoreViewModel(app: Application, private val type: String, private val shotTarget: Int = 1080) : AndroidViewModel(app) {
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
                clearShots()
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
            clearShots()
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

    // Capturas: caché LRU en bytes; shotTick fuerza la recomposición al cambiarla
    private val shotCache = object : LruCache<String, Bitmap>((Runtime.getRuntime().maxMemory() / 8).toInt()) {
        override fun sizeOf(key: String, value: Bitmap) = value.byteCount
    }
    private val shotLoading = HashSet<String>()
    private var shotTick by mutableIntStateOf(0)

    fun shot(url: String): ImageBitmap? {
        shotTick // lectura para suscribir la composición
        return shotCache.get(url)?.asImageBitmap()
    }

    fun loadShot(url: String) {
        if (shotCache.get(url) != null || !shotLoading.add(url)) return
        val c = client ?: run { shotLoading.remove(url); return }
        viewModelScope.launch {
            try {
                val b = c.bytes(url)
                val bmp = withContext(Dispatchers.Default) {
                    val o = BitmapFactory.Options().apply { inJustDecodeBounds = true }
                    BitmapFactory.decodeByteArray(b, 0, b.size, o)
                    val opts = BitmapFactory.Options().apply { inSampleSize = sampleSize(o.outWidth, o.outHeight, shotTarget) }
                    BitmapFactory.decodeByteArray(b, 0, b.size, opts)
                }
                if (bmp != null) { shotCache.put(url, bmp); shotTick++ }
            } catch (e: Exception) { /* se queda el hueco gris */ }
            finally { shotLoading.remove(url) }
        }
    }

    private fun clearShots() { shotCache.evictAll(); shotTick++ }

    fun act(app: StoreApp, onPermission: () -> Unit) {
        when (actionFor(app.versionCode, installed[app.id])) {
            AppAction.OPEN -> installer.launch(app.pkg)
            else -> if (!installer.canInstall()) onPermission()
            else viewModelScope.launch { installer.install(app) }
        }
    }

    // Versión anterior: Android no deja bajar de versión, así que solo si no hay una más nueva instalada
    fun installVersion(app: StoreApp, v: StoreVersion, onPermission: () -> Unit) {
        if ((installed[app.id] ?: 0) > v.versionCode) return
        if (!installer.canInstall()) onPermission()
        else viewModelScope.launch { installer.install(app.asVersion(v)) }
    }
}
