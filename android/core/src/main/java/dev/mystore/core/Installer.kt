package dev.mystore.core

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInstaller
import android.content.pm.PackageManager
import android.net.Uri
import android.provider.Settings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withContext
import java.io.File
import java.io.IOException
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

    fun canInstall(): Boolean = context.packageManager.canRequestPackageInstalls()

    fun permissionIntent(): Intent =
        Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:${context.packageName}"))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

    fun launch(pkg: String) {
        context.packageManager.getLaunchIntentForPackage(pkg)
            ?.let { context.startActivity(it.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) }
    }

    fun refresh() { _ticks.value++ }

    suspend fun install(app: StoreApp) {
        if (states.value[app.id].let { it is InstallState.Downloading || it is InstallState.Installing }) return
        setInstallState(app.id, InstallState.Downloading(0f))
        val file = File(context.cacheDir, "${app.id}.apk")
        try {
            withContext(Dispatchers.IO) {
                download(app, file)
                // Solo se crea la sesión con la APK completa
                if (file.length() != app.size) {
                    throw IOException("La descarga se cortó; inténtalo de nuevo")
                }
                setInstallState(app.id, InstallState.Installing)
                commit(app, file)
            }
        } catch (e: Exception) {
            val msg = if (e is IOException && e.message?.startsWith("La descarga") == true) e.message!!
            else "No se pudo descargar ${app.label}: revisa la red y reintenta"
            setInstallState(app.id, InstallState.Failed(msg))
        } finally {
            file.delete()
        }
    }

    private fun download(app: StoreApp, file: File) {
        val c = URL(app.apkUrl).openConnection() as HttpURLConnection
        c.connectTimeout = 10_000
        c.readTimeout = 30_000
        try {
            if (c.responseCode !in 200..299) throw IOException("HTTP ${c.responseCode}")
            var done = 0L
            c.inputStream.use { input ->
                file.outputStream().use { out ->
                    val buf = ByteArray(64 * 1024)
                    while (true) {
                        val n = input.read(buf)
                        if (n < 0) break
                        out.write(buf, 0, n)
                        done += n
                        if (app.size > 0) {
                            setInstallState(app.id, InstallState.Downloading((done.toFloat() / app.size).coerceAtMost(1f)))
                        }
                    }
                }
            }
        } finally {
            c.disconnect()
        }
    }

    private fun commit(app: StoreApp, file: File) {
        val pi = context.packageManager.packageInstaller
        val id = pi.createSession(PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL))
        pi.openSession(id).use { session ->
            file.inputStream().use { input ->
                session.openWrite("app.apk", 0, file.length()).use { out ->
                    input.copyTo(out)
                    session.fsync(out)
                }
            }
            val intent = Intent(context, InstallReceiver::class.java).putExtra(InstallReceiver.EXTRA_APP_ID, app.id)
            val pending = PendingIntent.getBroadcast(
                context, id, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
            )
            session.commit(pending.intentSender)
        }
    }
}
