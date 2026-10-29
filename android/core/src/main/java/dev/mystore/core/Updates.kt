package dev.mystore.core

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.pm.PackageManager
import android.net.ConnectivityManager
import android.os.BatteryManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.work.*
import java.util.concurrent.TimeUnit

const val EXTRA_SHOW_UPDATES = "dev.mystore.SHOW_UPDATES"

fun pendingUpdates(apps: List<StoreApp>, installed: Map<String, Long?>): List<StoreApp> =
    apps.filter { actionFor(it.versionCode, installed[it.id]) == AppAction.UPDATE }

// Solo se avisa si aparece algo que no se había notificado
fun isNewSet(previous: Set<String>, current: Set<String>): Boolean = !previous.containsAll(current)

fun updatesText(labels: List<String>): String =
    if (labels.size >= 3) "${labels.size} apps: ${labels.joinToString(", ")}" else labels.joinToString(", ")

// APKs guardadas que sobran (nombre "<id>-<versionCode>.apk"): app fuera de la tienda, ya hay una
// versión más nueva instalada, más de 30 días sin usarse o descargas a medias de hace más de un día
fun staleApks(files: List<Pair<String, Long>>, apps: List<StoreApp>, installed: Map<String, Long?>, now: Long): List<String> {
    val ids = apps.map { it.id }.toSet()
    return files.filter { (name, modified) ->
        if (name.endsWith(".part")) return@filter now - modified > 86_400_000L
        val base = name.removeSuffix(".apk")
        val id = base.substringBeforeLast('-')
        val code = base.substringAfterLast('-').toLongOrNull() ?: return@filter true
        id !in ids || (installed[id] ?: 0) > code || now - modified > 30 * 86_400_000L
    }.map { it.first }
}

// La propia tienda se actualiza la última: al instalarse se cierra el proceso
fun autoUpdateOrder(apps: List<StoreApp>, ownPkg: String): List<StoreApp> = apps.sortedBy { it.pkg == ownPkg }

fun needsNotificationPermission(context: Context): Boolean =
    Build.VERSION.SDK_INT >= 33 &&
        context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED

class UpdateWorker(private val ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        try {
            val url = StoreSettings(ctx).storeUrl?.takeIf { it.isNotBlank() } ?: return Result.success()
            val type = inputData.getString("type") ?: "phone"
            val apps = StoreClient(url, deviceName(ctx, type)).apps(type)
            val installer = Installer(ctx)
            installer.pruneCache(apps)
            var pending = pendingUpdates(apps, apps.associate { it.id to installer.installedVersion(it.pkg) })
            // Las instaladas desde la tienda se actualizan solas; el resto se avisa como siempre
            if (StoreSettings(ctx).autoUpdate && idleConditions(ctx)) {
                val auto = pending.filter { installer.isOurs(it.pkg) }
                pending = pending - auto.toSet()
                autoUpdateOrder(auto, ctx.packageName).forEach { installer.install(it, auto = true) }
            }
            val nm = ctx.getSystemService(NotificationManager::class.java)
            val prefs = ctx.getSharedPreferences("mystore_updates", Context.MODE_PRIVATE)
            if (pending.isEmpty()) {
                nm.cancel(NOTIF_ID)
                prefs.edit().remove("notified").apply()
                return Result.success()
            }
            val current = pending.map { "${it.id}:${it.versionCode}" }.toSet()
            if (!isNewSet(prefs.getStringSet("notified", emptySet()).orEmpty(), current)) return Result.success()
            if (needsNotificationPermission(ctx)) return Result.success()
            nm.createNotificationChannel(NotificationChannel("updates", "Actualizaciones", NotificationManager.IMPORTANCE_DEFAULT))
            val launch = ctx.packageManager.getLaunchIntentForPackage(ctx.packageName)?.putExtra(EXTRA_SHOW_UPDATES, true)
            val tap = launch?.let { PendingIntent.getActivity(ctx, 0, it, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE) }
            nm.notify(NOTIF_ID, NotificationCompat.Builder(ctx, "updates")
                .setSmallIcon(R.drawable.ic_stat_update)
                .setContentTitle("Actualizaciones disponibles")
                .setContentText(updatesText(pending.map { it.label }))
                .setContentIntent(tap).setAutoCancel(true).build())
            prefs.edit().putStringSet("notified", current).apply()
        } catch (e: Exception) {
            // Red caída u otro fallo: se reintenta en el siguiente periodo
        }
        return Result.success()
    }

    private companion object { const val NOTIF_ID = 1001 }
}

// Cargando y con red sin límite de datos (WiFi): el momento de bajar actualizaciones sin preguntar
private fun idleConditions(ctx: Context): Boolean {
    val battery = ctx.getSystemService(BatteryManager::class.java)
    val net = ctx.getSystemService(ConnectivityManager::class.java)
    return battery.isCharging && !net.isActiveNetworkMetered
}

// Instalar una app pedida desde el móvil (en el reloj): input "pkg" y "type"
class RemoteInstallWorker(private val ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        val url = StoreSettings(ctx).storeUrl?.takeIf { it.isNotBlank() } ?: return Result.failure()
        val type = inputData.getString("type") ?: "watch"
        val pkg = inputData.getString("pkg") ?: return Result.failure()
        return try {
            val app = StoreClient(url, deviceName(ctx, type)).apps(type).find { it.pkg == pkg } ?: return Result.failure()
            Installer(ctx).install(app, auto = true)
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }
}

object UpdateChecks {
    fun installRemote(context: Context, pkg: String, type: String) {
        WorkManager.getInstance(context).enqueue(OneTimeWorkRequestBuilder<RemoteInstallWorker>()
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .setInputData(workDataOf("pkg" to pkg, "type" to type)).build())
    }

    private fun data(type: String) = workDataOf("type" to type)

    fun schedule(context: Context, type: String) {
        val req = PeriodicWorkRequestBuilder<UpdateWorker>(6, TimeUnit.HOURS)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .setInputData(data(type)).build()
        WorkManager.getInstance(context).enqueueUniquePeriodicWork("mystore-updates", ExistingPeriodicWorkPolicy.KEEP, req)
        // Otra pasada cuando está cargando con WiFi, que es cuando se actualiza solo
        val idle = PeriodicWorkRequestBuilder<UpdateWorker>(6, TimeUnit.HOURS)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.UNMETERED).setRequiresCharging(true).build())
            .setInputData(data(type)).build()
        WorkManager.getInstance(context).enqueueUniquePeriodicWork("mystore-autoupdate", ExistingPeriodicWorkPolicy.KEEP, idle)
    }

    fun checkNow(context: Context, type: String) {
        WorkManager.getInstance(context).enqueue(OneTimeWorkRequestBuilder<UpdateWorker>().setInputData(data(type)).build())
    }
}
