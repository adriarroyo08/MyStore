package dev.mystore.core

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInstaller
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withContext
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.HttpURLConnection
import java.net.URL

sealed interface InstallState {
    data object Idle : InstallState
    data class Downloading(val progress: Float) : InstallState
    data object Installing : InstallState
    data class Failed(val message: String) : InstallState
}

class Installer(private val context: Context) {
    val states: StateFlow<Map<String, InstallState>> = installStates

    // Sube al volver a primer plano para que la UI recalcule los botones
    private val _ticks = MutableStateFlow(0)
    val ticks: StateFlow<Int> = _ticks

    fun installedVersion(pkg: String): Long? = try {
        context.packageManager.getPackageInfo(pkg, 0).longVersionCode
    } catch (e: PackageManager.NameNotFoundException) {
        null
    }

    // APKs descargadas, una por app (la última), para reinstalar sin volver a bajarla
    private val apkDir = File(context.filesDir, "apks")

    private fun cacheFile(app: StoreApp) = File(apkDir, "${app.id}-${app.versionCode}.apk")

    fun isCached(app: StoreApp): Boolean = app.size > 0 && cacheFile(app).length() == app.size

    fun cacheBytes(): Long = apkDir.listFiles()?.sumOf { it.length() } ?: 0

    // Borra las APKs que ya no sirven (ver staleApks)
    fun pruneCache(apps: List<StoreApp>) {
        val files = apkDir.listFiles() ?: return
        val stale = staleApks(files.map { it.name to it.lastModified() }, apps,
            apps.associate { it.id to installedVersion(it.pkg) }, System.currentTimeMillis()).toSet()
        files.filter { it.name in stale }.forEach { it.delete() }
    }

    // Instalada por esta tienda: Android deja actualizarla sin preguntar (API 31+)
    fun isOurs(pkg: String): Boolean = try {
        context.packageManager.getInstallSourceInfo(pkg).installingPackageName == context.packageName
    } catch (e: PackageManager.NameNotFoundException) {
        false
    }

    fun clearCache() {
        apkDir.listFiles()?.forEach { it.delete() }
        _ticks.value++
    }

    // El sistema pide confirmación; al volver, onResume → refresh() recalcula los botones
    fun uninstall(pkg: String) {
        context.startActivity(Intent(Intent.ACTION_DELETE, Uri.parse("package:$pkg")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
    }

    fun canInstall(): Boolean = context.packageManager.canRequestPackageInstalls()

    fun permissionIntent(): Intent =
        Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:${context.packageName}"))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

    fun launch(pkg: String) {
        context.packageManager.getLaunchIntentForPackage(pkg)
            ?.let { context.startActivity(it.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) }
    }

    // Si el usuario salió del diálogo del sistema no llega ningún STATUS: se comprueba la sesión
    fun refresh() {
        installStates.value.forEach { (id, st) -> if (st is InstallState.Installing) checkStale(id) }
        _ticks.value++
    }

    private fun checkStale(appId: String) {
        val sid = sessionIds[appId]
        if (sid == null || context.packageManager.packageInstaller.getSessionInfo(sid) == null) {
            sessionIds.remove(appId)
            confirmIntents.remove(appId)
            setInstallState(appId, InstallState.Idle)
        }
    }

    // auto: sin la app delante (segundo plano o pedido desde el móvil); si el sistema pide confirmación
    // sale una notificación en vez del diálogo
    suspend fun install(app: StoreApp, auto: Boolean = false) {
        val cur = states.value[app.id]
        if (cur is InstallState.Downloading) return
        if (cur is InstallState.Installing) {
            // Pulsar de nuevo: reabre el diálogo pendiente o libera el botón
            checkStale(app.id)
            if (states.value[app.id] is InstallState.Installing) {
                val confirm = confirmIntents[app.id]
                if (confirm != null) {
                    try {
                        context.startActivity(Intent(confirm).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
                    } catch (e: Exception) {
                        abandonStored(app.id)
                    }
                } else abandonStored(app.id)
            }
            return
        }
        setInstallState(app.id, InstallState.Downloading(0f))
        val pi = context.packageManager.packageInstaller
        var downloaded = false
        try {
            withContext(Dispatchers.IO) {
                val params = PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL)
                if (Build.VERSION.SDK_INT >= 31) params.setRequireUserAction(PackageInstaller.SessionParams.USER_ACTION_NOT_REQUIRED)
                val sid = pi.createSession(params)
                pi.openSession(sid).use { session ->
                    try {
                        download(app, session)
                        downloaded = true
                        sessionIds[app.id] = sid
                        setInstallState(app.id, InstallState.Installing)
                        commit(app, session, sid, auto)
                    } catch (e: Exception) {
                        session.abandon()
                        throw e
                    }
                }
            }
        } catch (e: Exception) {
            sessionIds.remove(app.id)
            val msg = when {
                e is IOException && e.message?.startsWith("La descarga") == true -> e.message!!
                downloaded -> "No se pudo instalar ${app.label}: el sistema rechazó la instalación"
                else -> "No se pudo descargar ${app.label}: revisa la red y reintenta"
            }
            setInstallState(app.id, InstallState.Failed(msg))
        }
    }

    private fun abandonStored(appId: String) {
        sessionIds.remove(appId)?.let {
            try { context.packageManager.packageInstaller.abandonSession(it) } catch (e: Exception) { }
        }
        confirmIntents.remove(appId)
        setInstallState(appId, InstallState.Idle)
    }

    // Si ya está guardada se instala desde el disco; si no, el cuerpo HTTP va a la sesión y a la vez
    // a apks/, sustituyendo la copia anterior de esa app
    private fun download(app: StoreApp, session: PackageInstaller.Session) {
        val cached = cacheFile(app)
        if (isCached(app)) {
            cached.inputStream().use { copyToSession(app, it, session, null) }
            cached.setLastModified(System.currentTimeMillis()) // usada: se libra de la limpieza de 30 días
            return
        }
        val part = File(apkDir, cached.name + ".part")
        val c = URL(app.apkUrl).openConnection() as HttpURLConnection
        c.connectTimeout = 10_000
        c.readTimeout = 30_000
        try {
            if (c.responseCode !in 200..299) throw IOException("HTTP ${c.responseCode}")
            val cache = try { apkDir.mkdirs(); part.outputStream() } catch (e: IOException) { null }
            val kept = try {
                c.inputStream.use { copyToSession(app, it, session, cache) }
            } finally {
                cache?.close() // ya cerrado si fue bien; aquí por si la red falló a medias
            }
            if (kept) {
                apkDir.listFiles()?.filter { it.name.startsWith("${app.id}-") && it != part }?.forEach { it.delete() }
                part.renameTo(cached)
            }
        } finally {
            part.delete() // si no se renombró, era una descarga a medias
            c.disconnect()
        }
    }

    // Devuelve si la copia en caché quedó completa; un fallo al guardarla (disco lleno) no para la instalación
    private fun copyToSession(app: StoreApp, input: InputStream, session: PackageInstaller.Session, cache: OutputStream?): Boolean {
        var copy = cache
        var done = 0L
        session.openWrite("base.apk", 0, app.size).use { out ->
            val buf = ByteArray(64 * 1024)
            while (true) {
                val n = input.read(buf)
                if (n < 0) break
                out.write(buf, 0, n)
                copy?.let { try { it.write(buf, 0, n) } catch (e: IOException) { runCatching { it.close() }; copy = null } }
                done += n
                if (app.size > 0) {
                    setInstallState(app.id, InstallState.Downloading((done.toFloat() / app.size).coerceAtMost(1f)))
                }
            }
            if (done != app.size) throw IOException("La descarga se cortó; inténtalo de nuevo")
            session.fsync(out)
        }
        return copy?.let { try { it.close(); true } catch (e: IOException) { false } } ?: false
    }

    private fun commit(app: StoreApp, session: PackageInstaller.Session, id: Int, auto: Boolean) {
        val intent = Intent(context, InstallReceiver::class.java).putExtra(InstallReceiver.EXTRA_APP_ID, app.id)
            .putExtra(InstallReceiver.EXTRA_LABEL, app.label).putExtra(InstallReceiver.EXTRA_AUTO, auto)
        val pending = PendingIntent.getBroadcast(
            context, id, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
        )
        session.commit(pending.intentSender)
    }
}
